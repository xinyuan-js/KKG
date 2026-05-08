# KKG 项目快速启动

面向本地开发，最短路径：配置 -> 启动 -> 验证。

## 0. 环境要求

- Docker Desktop（已启动）
- `make`
- 可用端口：`80`、`3001`、`3307`、`6379`、`8080`、`8121`、`8082`、`9000`、`9001`

## 1. 配置

```bash
cp .env.example .env
cp kkg-oj-backend/config.local.example.yaml kkg-oj-backend/config.local.yaml
```

必填位置：
- `.env`：`JWT_SECRET`、`MYSQL_PASSWORD`、`REDIS_PASSWORD`、`RABBITMQ_PASSWORD`、`MINIO_*`、`SUPER_ADMIN_PASSWORD`
- `kkg-oj-backend/config.local.yaml`：`rabbitmq.url`、`blog.agent_password`、`jwt_secret`（可选：`agent.api_key`）

## 2. 启动

```bash
make dev-core
make dev-pro
make oj-dev
make api-dev
make web-dev
make gateway-run
```

## 3. 验证

- 前端：`http://127.0.0.1:3001`
- 网关：`http://127.0.0.1`
- 博客后端健康检查：`http://127.0.0.1:8080/health`

## 4. 故障排查

- `invalid config: JWT_SECRET is required`：`.env` 仍是占位值，修改后重启 `make api-dev`
- `dial tcp 127.0.0.1:3307`：MySQL 未就绪，执行 `make dev-core` 并检查 `make ps`

## 5. 详细文档

- 使用指南：[docs/setup-guide.md](./docs/setup-guide.md)
- 架构说明：[docs/architecture.md](./docs/architecture.md)
- Git 规范：[docs/git-guide.md](./docs/git-guide.md)
