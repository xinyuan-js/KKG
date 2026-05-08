package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	maxCodeBytes   = 128 * 1024
	maxOutputBytes = 1 * 1024 * 1024
)

type ExecuteCodeRequest struct {
	InputList []string `json:"inputList"`
	Code      string   `json:"code"`
	Language  string   `json:"language"`
}

type JudgeInfo struct {
	Message string `json:"message"`
	Memory  int64  `json:"memory"`
	Time    int64  `json:"time"`
}

type ExecuteCodeResponse struct {
	OutputList []string  `json:"outputList"`
	Message    string    `json:"message"`
	Status     int       `json:"status"`
	JudgeInfo  JudgeInfo `json:"judgeInfo"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/executeCode", executeCode)

	addr := getEnv("SANDBOX_ADDR", ":8082")
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("go sandbox listening at %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func executeCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authSecret := getEnv("SANDBOX_AUTH_SECRET", "secretKey")
	if r.Header.Get("auth") != authSecret {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req ExecuteCodeRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req); err != nil {
		writeResp(w, http.StatusBadRequest, ExecuteCodeResponse{
			Message: "invalid request body",
			Status:  3,
			JudgeInfo: JudgeInfo{
				Message: "请求格式错误",
			},
		})
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		writeResp(w, http.StatusOK, ExecuteCodeResponse{
			Message: "code is empty",
			Status:  3,
			JudgeInfo: JudgeInfo{
				Message: "代码为空",
			},
		})
		return
	}
	if len(req.Code) > maxCodeBytes {
		writeResp(w, http.StatusOK, ExecuteCodeResponse{
			Message: "code too large",
			Status:  3,
			JudgeInfo: JudgeInfo{
				Message: "代码过大，超过 128KB 限制",
			},
		})
		return
	}
	if err := validateGoCode(req.Code); err != nil {
		writeResp(w, http.StatusOK, ExecuteCodeResponse{
			Message: err.Error(),
			Status:  3,
			JudgeInfo: JudgeInfo{
				Message: err.Error(),
			},
		})
		return
	}

	resp, err := run(req)
	if err != nil {
		writeResp(w, http.StatusOK, ExecuteCodeResponse{
			Message: err.Error(),
			Status:  3,
			JudgeInfo: JudgeInfo{
				Message: err.Error(),
			},
		})
		return
	}
	writeResp(w, http.StatusOK, *resp)
}

func run(req ExecuteCodeRequest) (*ExecuteCodeResponse, error) {
	lang := strings.ToLower(strings.TrimSpace(req.Language))
	switch lang {
	case "go":
		return runGo(req)
	default:
		return nil, fmt.Errorf("unsupported language: %s", req.Language)
	}
}

func runGo(req ExecuteCodeRequest) (*ExecuteCodeResponse, error) {
	if strings.EqualFold(getEnv("SANDBOX_GO_MODE", "container"), "container") {
		return runGoInContainer(req)
	}
	return runGoDirect(req)
}

func runGoDirect(req ExecuteCodeRequest) (*ExecuteCodeResponse, error) {
	root, err := os.MkdirTemp("", "yuoj-go-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)

	mainFile := filepath.Join(root, "main.go")
	if err = os.WriteFile(mainFile, []byte(req.Code), 0o644); err != nil {
		return nil, err
	}

	compileCtx, cancelCompile := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancelCompile()
	compileCmd := exec.CommandContext(compileCtx, goBinary(), "build", "-o", "main-bin", "main.go")
	compileCmd.Dir = root
	out, err := runCommandLimited(compileCmd, maxOutputBytes)
	if err != nil {
		if errors.Is(compileCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("compile timeout")
		}
		if strings.Contains(runErrText(err), "output limit exceeded") {
			return nil, fmt.Errorf("compile output limit exceeded")
		}
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = runErrText(err)
		}
		return nil, fmt.Errorf("compile failed: %s", msg)
	}

	outputs := make([]string, 0, max(1, len(req.InputList)))
	maxTimeMs := int64(0)
	for _, input := range normalizeInputs(req.InputList) {
		start := time.Now()
		runCtx, cancelRun := context.WithTimeout(context.Background(), 6*time.Second)
		runCmd := exec.CommandContext(runCtx, "./main-bin")
		runCmd.Dir = root
		runCmd.Stdin = strings.NewReader(input)
		runOut, runErr := runCommandLimited(runCmd, maxOutputBytes)
		cancelRun()
		elapsed := time.Since(start).Milliseconds()
		if elapsed > maxTimeMs {
			maxTimeMs = elapsed
		}
		if runErr != nil {
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("time limit exceeded")
			}
			if strings.Contains(runErrText(runErr), "output limit exceeded") {
				return nil, fmt.Errorf("output limit exceeded")
			}
			msg := strings.TrimSpace(string(runOut))
			if msg == "" {
				msg = runErrText(runErr)
			}
			return nil, fmt.Errorf("runtime error: %s", msg)
		}
		outputs = append(outputs, strings.TrimSpace(string(runOut)))
	}

	return &ExecuteCodeResponse{
		OutputList: outputs,
		Message:    "ok",
		Status:     1,
		JudgeInfo: JudgeInfo{
			Message: "OK",
			Memory:  0,
			Time:    maxTimeMs,
		},
	}, nil
}

func runGoInContainer(req ExecuteCodeRequest) (*ExecuteCodeResponse, error) {
	baseDir := strings.TrimSpace(getEnv("SANDBOX_WORKDIR", "/private/tmp/kkg-sandbox"))
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	root, err := os.MkdirTemp(baseDir, "yuoj-go-runner-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)
	_ = os.Chmod(root, 0o777)

	if err = os.WriteFile(filepath.Join(root, "main.go"), []byte(req.Code), 0o644); err != nil {
		return nil, err
	}
	inputs := normalizeInputs(req.InputList)
	for i, in := range inputs {
		if err = os.WriteFile(filepath.Join(root, fmt.Sprintf("input_%d.txt", i)), []byte(in), 0o644); err != nil {
			return nil, err
		}
	}

	image := getEnv("SANDBOX_RUNNER_IMAGE", "golang:1.23-bookworm")
	cpuLimit := getEnv("SANDBOX_RUNNER_CPUS", "1")
	memLimit := getEnv("SANDBOX_RUNNER_MEMORY", "256m")
	pidsLimit := getEnv("SANDBOX_RUNNER_PIDS", "64")
	outputLimit := getEnv("SANDBOX_MAX_OUTPUT_BYTES", "1048576")
	timeoutSec := atoiDefault(getEnv("SANDBOX_RUNNER_TIMEOUT_SEC", "12"), 12)
	if timeoutSec < 2 {
		timeoutSec = 2
	}
	runScript := "set -eu;" +
		"cd /workspace;" +
		"export GO111MODULE=off GOCACHE=/workspace/.gocache GOTMPDIR=/workspace/.tmp TMPDIR=/workspace/.tmp;" +
		"mkdir -p /workspace/.gocache /workspace/.tmp;" +
		"/usr/local/go/bin/go build -o /workspace/main-bin main.go >/workspace/compile.out 2>/workspace/compile.err;" +
		"for f in /workspace/input_*.txt; do " +
		"n=${f#/workspace/input_}; n=${n%.txt}; " +
		"/workspace/main-bin < \"$f\" > \"/workspace/output_${n}.txt\" 2> \"/workspace/runtime_${n}.err\"; " +
		"done"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"--network", "none",
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges:true",
		"--pids-limit", pidsLimit,
		"--memory", memLimit,
		"--cpus", cpuLimit,
		"--user", "65534:65534",
		"--tmpfs", "/tmp:exec,size=128m,mode=1777",
		"-v", root+":/workspace:rw",
		image,
		"sh", "-lc", runScript,
	)
	out, runErr := runCommandLimited(cmd, maxOutputBytes)
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("time limit exceeded")
	}
	if runErr != nil {
		compileErr := readTrim(filepath.Join(root, "compile.err"))
		if compileErr != "" {
			return nil, fmt.Errorf("compile failed: %s", compileErr)
		}
		if len(out) > 0 {
			return nil, fmt.Errorf("runtime error: %s", strings.TrimSpace(string(out)))
		}
		return nil, fmt.Errorf("runtime error: %s", runErr.Error())
	}

	limit, _ := strconv.ParseInt(outputLimit, 10, 64)
	if limit <= 0 {
		limit = maxOutputBytes
	}
	outputs := make([]string, 0, len(inputs))
	var totalBytes int64
	for i := range inputs {
		outFile := filepath.Join(root, fmt.Sprintf("output_%d.txt", i))
		b, rErr := os.ReadFile(outFile)
		if rErr != nil {
			runtimeErr := readTrim(filepath.Join(root, fmt.Sprintf("runtime_%d.err", i)))
			if runtimeErr == "" {
				runtimeErr = rErr.Error()
			}
			return nil, fmt.Errorf("runtime error: %s", runtimeErr)
		}
		totalBytes += int64(len(b))
		if totalBytes > limit {
			return nil, fmt.Errorf("output limit exceeded")
		}
		outputs = append(outputs, strings.TrimSpace(string(b)))
	}

	return &ExecuteCodeResponse{
		OutputList: outputs,
		Message:    "ok",
		Status:     1,
		JudgeInfo: JudgeInfo{
			Message: "OK",
			Memory:  0,
			// 容器模式下这里不计入编译与容器启动开销，避免误判 TLE。
			Time: 0,
		},
	}, nil
}

func normalizeInputs(input []string) []string {
	if len(input) == 0 {
		return []string{""}
	}
	return input
}

func writeResp(w http.ResponseWriter, status int, v ExecuteCodeResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func getEnv(k, fallback string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return fallback
	}
	return v
}

func goBinary() string {
	if _, err := os.Stat("/usr/local/go/bin/go"); err == nil {
		return "/usr/local/go/bin/go"
	}
	return "go"
}

func runErrText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func atoiDefault(s string, d int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return d
	}
	return v
}

func readTrim(file string) string {
	b, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func validateGoCode(code string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", code, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("代码解析失败: %v", err)
	}
	blocked := map[string]struct{}{
		"os/exec": {},
		"syscall": {},
		"unsafe":  {},
		"net":     {},
		"net/http": {},
	}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if _, ok := blocked[path]; ok {
			return fmt.Errorf("禁止导入危险包: %s", path)
		}
	}
	return nil
}

func runCommandLimited(cmd *exec.Cmd, limit int64) ([]byte, error) {
	var lb limitedBuffer
	lb.limit = limit
	cmd.Stdout = &lb
	cmd.Stderr = &lb
	err := cmd.Run()
	if lb.truncated {
		return lb.buf, fmt.Errorf("output limit exceeded")
	}
	return lb.buf, err
}

type limitedBuffer struct {
	buf       []byte
	limit     int64
	truncated bool
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.truncated {
		return len(p), nil
	}
	remain := l.limit - int64(len(l.buf))
	if remain <= 0 {
		l.truncated = true
		return len(p), nil
	}
	if int64(len(p)) > remain {
		l.buf = append(l.buf, p[:remain]...)
		l.truncated = true
		return len(p), nil
	}
	l.buf = append(l.buf, p...)
	return len(p), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
