package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	SandboxType string
	SandboxURL  string
	AuthSecret  string
}

type Case struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type Info struct {
	Message string `json:"message"`
	Memory  int64  `json:"memory"`
	Time    int64  `json:"time"`
	Score   int64  `json:"score"`
}

type ExecuteCodeResp struct {
	OutputList []string `json:"outputList"`
	JudgeInfo  Info     `json:"judgeInfo"`
}

func ExecuteCode(cfg Config, language, code string, inputList []string) (*ExecuteCodeResp, error) {
	if cfg.SandboxType == "remote" && cfg.SandboxURL != "" {
		reqBody, _ := json.Marshal(map[string]interface{}{
			"language":  language,
			"code":      code,
			"inputList": inputList,
		})
		req, _ := http.NewRequest(http.MethodPost, cfg.SandboxURL, bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(cfg.AuthSecret) != "" {
			req.Header.Set("auth", cfg.AuthSecret)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var out ExecuteCodeResp
		if err = json.Unmarshal(body, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	return runLocalCode(language, code, inputList)
}

func runLocalCode(language, code string, inputList []string) (*ExecuteCodeResp, error) {
	lang := strings.ToLower(strings.TrimSpace(language))
	switch lang {
	case "go":
		return runLocalGo(code, inputList)
	default:
		return nil, fmt.Errorf("unsupported language in local sandbox: %s", language)
	}
}

func runLocalGo(code string, inputList []string) (*ExecuteCodeResp, error) {
	root, err := os.MkdirTemp("", "oj-go-run-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)
	mainFile := filepath.Join(root, "main.go")
	if err = os.WriteFile(mainFile, []byte(code), 0o644); err != nil {
		return nil, err
	}
	inputs := inputList
	if len(inputs) == 0 {
		inputs = []string{""}
	}
	outputs := make([]string, 0, len(inputs))
	maxTime := int64(0)
	for _, input := range inputs {
		start := time.Now()
		runCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		goBin := "go"
		if _, statErr := os.Stat("/usr/local/go/bin/go"); statErr == nil {
			goBin = "/usr/local/go/bin/go"
		}
		cmd := exec.CommandContext(runCtx, goBin, "run", "main.go")
		cmd.Dir = root
		cmd.Stdin = strings.NewReader(input)
		out, runErr := cmd.CombinedOutput()
		cancel()
		cost := time.Since(start).Milliseconds()
		if cost > maxTime {
			maxTime = cost
		}
		if runErr != nil {
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("time limit exceeded")
			}
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = runErr.Error()
			}
			if strings.Contains(msg, "syntax error") || strings.Contains(msg, "undefined") {
				return nil, fmt.Errorf("compile error: %s", msg)
			}
			return nil, fmt.Errorf("runtime error: %s", msg)
		}
		outputs = append(outputs, strings.TrimSpace(string(out)))
	}
	return &ExecuteCodeResp{
		OutputList: outputs,
		JudgeInfo: Info{
			Message: "OK",
			Memory:  0,
			Time:    maxTime,
		},
	}, nil
}

func Judge(cases []Case, outputs []string, sandbox Info, judgeConfigJSON string) Info {
	msg := strings.TrimSpace(strings.ToLower(sandbox.Message))
	if msg != "" && msg != "ok" && msg != "accepted" {
		if strings.Contains(msg, "compile") {
			return Info{Message: "Compile Error", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		if strings.Contains(msg, "runtime") || strings.Contains(msg, "panic") || strings.Contains(msg, "exception") {
			return Info{Message: "Runtime Error", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		if strings.Contains(msg, "time") {
			return Info{Message: "Time Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		if strings.Contains(msg, "memory") {
			return Info{Message: "Memory Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
		}
		return Info{Message: sandbox.Message, Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
	}

	passed := int64(0)
	total := int64(len(cases))
	for i, c := range cases {
		got := ""
		if i < len(outputs) {
			got = strings.TrimSpace(outputs[i])
		}
		if strings.TrimSpace(c.Output) == got {
			passed++
		}
	}
	cfg := map[string]int64{}
	_ = json.Unmarshal([]byte(judgeConfigJSON), &cfg)
	if m, ok := cfg["memoryLimit"]; ok && m > 0 && sandbox.Memory > m {
		return Info{Message: "Memory Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
	}
	if t, ok := cfg["timeLimit"]; ok && t > 0 && sandbox.Time > t {
		return Info{Message: "Time Limit Exceeded", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
	}
	if total == 0 {
		return Info{Message: "Accepted", Memory: sandbox.Memory, Time: sandbox.Time, Score: 100}
	}
	score := passed * 100 / total
	if passed == total {
		return Info{Message: "Accepted", Memory: sandbox.Memory, Time: sandbox.Time, Score: 100}
	}
	if passed > 0 {
		return Info{Message: "Partially Correct", Memory: sandbox.Memory, Time: sandbox.Time, Score: score}
	}
	return Info{Message: "Wrong Answer", Memory: sandbox.Memory, Time: sandbox.Time, Score: 0}
}
