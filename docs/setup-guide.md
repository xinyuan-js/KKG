# 项目使用指南

目标：代码拉取后完成配置并运行博客 + OJ。

## 1. 前置条件

1. 已安装 Docker Desktop（并已启动）
2. 本机可用端口：`80`、`3001`、`3307`、`6379`、`8080`、`8082`、`9000`、`9001`
3. 已安装 `make`

## 2. 配置根环境变量 `.env`

1. 复制模板：
```bash
cp .env.example .env
```

2. 打开根目录 `.env`，修改以下必填字段：

- `MYSQL_ROOT_PASSWORD`
- `MYSQL_PASSWORD`
- `REDIS_PASSWORD`
- `MINIO_ROOT_PASSWORD`
- `MINIO_SECRET_KEY`
- `RABBITMQ_PASSWORD`
- `SUPER_ADMIN_PASSWORD`

3. 字段定义见 [`.env.example`](/Users/zhuojianshuo/GolandProjects/awesomeProject/.env.example)。

## 3. 配置 OJ 与 Agent

OJ 已合并到 `kkg-backend`，不再需要单独的 OJ 配置文件。需要开启 AI 题解时，编辑根目录 `.env`：

- `OJ_AGENT_ENABLED=true`
- `OJ_AGENT_API_KEY`
- `OJ_AGENT_BASE_URL`
- `OJ_AGENT_MODEL`
- `OJ_BLOG_AGENT_PASSWORD`

## 4. 启动顺序（严格按顺序）

1. 启动核心中间件：
```bash
make dev-core
```

2. 启动单体后端与判题沙盒：
```bash
make dev-api
```

3. 启动前端：
```bash
make web-dev
```

4. 启动网关（可选但推荐）：
```bash
make gateway-run
```

## 5. 验证地址

1. 单体后端健康检查：`http://127.0.0.1:8080/health`
2. OJ API：`http://127.0.0.1:8080/api/v1/oj`
3. 前端：`http://127.0.0.1:3001`
4. 网关统一入口：`http://127.0.0.1`

## 6. 常见错误

1. `dial tcp 127.0.0.1:3307 connect: ...`  
原因：MySQL 未启动或端口冲突。  
处理：`make dev-core`，再用 `make ps` 检查。

2. OJ Agent 发布题解失败  
原因：`.env` 缺少 `OJ_AGENT_API_KEY` 或 `OJ_BLOG_AGENT_PASSWORD`。  
处理：补齐本地配置并重启 `make api-restart`。

## 7. 提交前检查

1. 不要提交 `.env`
2. 仅提交 `.env.example` 这类模板
3. 提交前执行：
```bash
git status
```
确保没有本地密钥文件进入暂存区
