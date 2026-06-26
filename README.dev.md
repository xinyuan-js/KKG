# KKG 项目快速启动

面向本地开发，最短路径：配置 -> 启动 -> 验证。

## 0. 环境要求

- Docker Desktop（已启动）
- 可用端口：`80`、`3001`、`3307`、`6379`、`8080`、`8082`、`9000`、`9001`

## 1. 配置

```bash
cp .env.example .env
```

必填位置：
- `.env`：`MYSQL_PASSWORD`、`REDIS_PASSWORD`、`RABBITMQ_PASSWORD`、`MINIO_*`、`SUPER_ADMIN_PASSWORD`
- OJ、博客、鉴权都已合并到 `kkg-backend`，OJ 私有配置通过 `.env` 的 `OJ_*`、`AGENT_*`、`JUDGE_*` 变量控制。

## 2. 启动：Dev Container

用 VS Code / JetBrains Dev Containers 打开仓库，选择 `Reopen in Container`。开发容器会挂载当前仓库，并启动一组完整开发依赖：

- `mysql`
- `redis`
- `minio`
- `rabbitmq`
- `oj-sandbox`
- `api`（运行 `kkg-backend` 单体后端）
- `web`
- `nginx`

Dev Container 会自动让 `.devcontainer/.env` 指向根目录 `.env`，所有开发服务共享同一份环境配置。

进入容器后，工作目录是：

```bash
/workspaces/awesomeProject
```

手动启动完整开发环境：

```bash
docker compose --env-file .env -f .devcontainer/docker-compose.yml up -d --build
```

## 3. 验证

- 前端：`http://127.0.0.1:3001`
- 网关：`http://127.0.0.1`
- 单体后端健康检查：`http://127.0.0.1:8080/health`
- OJ API：`http://127.0.0.1:8080/api/v1/oj/*`

## 4. 故障排查

- `dial tcp mysql:3306`：MySQL 未就绪，检查 Dev Container 的 compose 服务状态。

## 5. 详细文档

- 使用指南：[docs/setup-guide.md](./docs/setup-guide.md)
- 架构说明：[docs/architecture.md](./docs/architecture.md)
- Git 规范：[docs/git-guide.md](./docs/git-guide.md)
