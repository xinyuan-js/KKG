# 新环境拉取即用指南

适用目标：新同事拉取仓库后，按本文完成配置并在本机跑通博客 + OJ。

## 1. 前置条件

1. 已安装 Docker Desktop（并已启动）
2. 本机可用端口：`80`、`3001`、`3307`、`6379`、`8080`、`8121`、`8082`、`9000`、`9001`
3. 已安装 `make`

## 2. 第一步：填写根配置 `.env`

1. 复制模板：
```bash
cp .env.example .env
```

2. 打开根目录 `.env`，至少修改以下字段（必须改）：

- `JWT_SECRET`：博客登录签名密钥，建议 32+ 位随机字符串
- `MYSQL_ROOT_PASSWORD`
- `MYSQL_PASSWORD`
- `REDIS_PASSWORD`
- `MINIO_ROOT_PASSWORD`
- `MINIO_SECRET_KEY`
- `RABBITMQ_PASSWORD`
- `SUPER_ADMIN_PASSWORD`

3. 这些字段都在文件 [`.env.example`](/Users/zhuojianshuo/GolandProjects/awesomeProject/.env.example) 里有对应占位值，直接按同名键填写即可。

## 3. 第二步：填写 OJ 配置

OJ 有两层配置：

1. 基础模板（可提交）  
`kkg-oj-backend/config.yaml`

2. 本地私有覆盖（不要提交）  
`kkg-oj-backend/config.local.yaml`

创建本地覆盖文件：
```bash
cp kkg-oj-backend/config.local.example.yaml kkg-oj-backend/config.local.yaml
```

然后编辑 [kkg-oj-backend/config.local.yaml](/Users/zhuojianshuo/GolandProjects/awesomeProject/kkg-oj-backend/config.local.yaml) 填写：
- `rabbitmq.url`
- `blog.agent_password`
- `agent.api_key`（如启用 AI 题解）
- `jwt_secret`

参考模板见 [config.local.example.yaml](/Users/zhuojianshuo/GolandProjects/awesomeProject/kkg-oj-backend/config.local.example.yaml)。

## 4. 第三步：启动顺序（严格按顺序）

1. 启动核心中间件：
```bash
make dev-core
```

2. 启动 RabbitMQ + OJ 组件：
```bash
make dev-pro
make oj-dev
```

3. 启动博客后端：
```bash
make api-dev
```

4. 启动前端：
```bash
make web-dev
```

5. 启动网关（可选但推荐）：
```bash
make gateway-run
```

## 5. 第四步：验收地址

1. 博客后端健康检查：`http://127.0.0.1:8080/health`
2. OJ 后端：`http://127.0.0.1:8121`
3. 前端：`http://127.0.0.1:3001`
4. 网关统一入口：`http://127.0.0.1`

## 6. 常见错误与处理

1. `invalid config: JWT_SECRET is required`  
原因：`.env` 里还在用占位值。  
处理：把 `.env` 的 `JWT_SECRET` 改为随机强密钥后重启 `make api-dev`。

2. `dial tcp 127.0.0.1:3307 connect: ...`  
原因：MySQL 未启动或端口冲突。  
处理：`make dev-core`，再用 `make ps` 检查。

3. OJ Agent 发布题解失败  
原因：`kkg-oj-backend/config.local.yaml` 缺少 `agent.api_key` 或 `blog.agent_password`。  
处理：补齐本地配置并重启 `make oj-api-restart`。

## 7. 提交前检查（避免泄露）

1. 不要提交 `.env` 与 `config.local.yaml`
2. 仅提交 `.env.example`、`config.example.yaml` 这类模板
3. 提交前执行：
```bash
git status
```
确保没有本地密钥文件进入暂存区
