# kkg-sandbox

Go 版本判题沙盒服务。

## 接口

- `GET /health` -> `ok`
- `POST /executeCode`
  - Header: `auth: secretKey`（可通过环境变量覆盖）
  - Body:
    ```json
    {
      "language": "go",
      "code": "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"hello\")}",
      "inputList": ["", "1 2"]
    }
    ```
  - Response:
    ```json
    {
      "outputList": ["hello", "hello"],
      "message": "ok",
      "status": 1,
      "judgeInfo": { "message": "OK", "memory": 0, "time": 12 }
    }
    ```

## 环境变量

- `SANDBOX_ADDR`：监听地址，默认 `:8082`
- `SANDBOX_AUTH_SECRET`：鉴权密钥，默认 `secretKey`
- `SANDBOX_GO_MODE`：`container`（默认，任务级临时容器）或 `direct`（进程内执行）
- `SANDBOX_RUNNER_IMAGE`：runner 镜像，默认 `golang:1.23-bookworm`
- `SANDBOX_RUNNER_TIMEOUT_SEC`：runner 总超时秒数，默认 `12`
- `SANDBOX_MAX_OUTPUT_BYTES`：输出上限字节，默认 `1048576`
- `SANDBOX_WORKDIR`：宿主共享工作目录，默认 `/private/tmp/kkg-sandbox`

## 启动

```bash
cd kkg-sandbox
go run .
```

## 说明

- 当前实现支持 `language=go`。
- 默认使用“每次请求一次性容器”执行模型（`docker run --rm`）。
- runner 容器使用 `--network none`、`--cap-drop ALL`、`no-new-privileges`、`pids/memory/cpu` 限制。
- 代码大小、危险包导入、输出体积均有限制。
- 使用容器模式时，沙盒进程必须能访问 Docker Daemon（当前通过挂载 `/var/run/docker.sock`）。
