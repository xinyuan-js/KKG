# KKG 开发快速引导

本文件只保留“正确配置 + 正确启动”的最短路径。  
完整说明请看 [docs/setup-guide.md](./docs/setup-guide.md)。

## 1) 先配置

1. 复制配置模板：
```bash
cp .env.example .env
cp kkg-oj-backend/config.local.example.yaml kkg-oj-backend/config.local.yaml
```

2. 必填项位置：
- 根目录 `.env`：`JWT_SECRET`、`MYSQL_PASSWORD`、`REDIS_PASSWORD`、`RABBITMQ_PASSWORD`、`MINIO_*`、`SUPER_ADMIN_PASSWORD`
- `kkg-oj-backend/config.local.yaml`：`rabbitmq.url`、`blog.agent_password`、`jwt_secret`、（可选）`agent.api_key`

## 2) 再启动（按顺序）

```bash
make dev-core
make dev-pro
make oj-dev
make api-dev
make web-dev
make gateway-run
```

## 3) 验证

- 博客后端: `http://127.0.0.1:8080/health`
- 前端: `http://127.0.0.1:3001`
- 网关: `http://127.0.0.1`

## 4) 常见错误

1. `invalid config: JWT_SECRET is required`  
`.env` 还在用占位值，改完重启 `make api-dev`。

2. `dial tcp 127.0.0.1:3307 ...`  
MySQL 未就绪，先执行 `make dev-core` 并检查 `make ps`。

## 5) 文档入口

- 一键上手: [docs/setup-guide.md](./docs/setup-guide.md)
- 架构: [docs/architecture.md](./docs/architecture.md)
- Git 规范: [docs/git-guide.md](./docs/git-guide.md)
- 模块文档: [docs/modules](./docs/modules)
