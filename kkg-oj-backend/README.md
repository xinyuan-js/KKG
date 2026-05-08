# kkg-oj-backend 配置说明

## 配置文件层级
1. `config.yaml`：可提交的基础配置（不含密钥）
2. `config.local.yaml`：本地私有覆盖配置（已被 `.gitignore` 忽略）
3. 环境变量：最高优先级（例如 `AGENT_API_KEY`、`JWT_SECRET`）

## 初始化
1. 复制 `config.example.yaml` 为 `config.yaml`（如果你要重建配置）
2. 复制 `config.local.example.yaml` 为 `config.local.yaml`
3. 在 `config.local.yaml` 或环境变量中填写敏感配置：
   - `agent.api_key`
   - `blog.agent_password`
   - `jwt_secret`
   - `rabbitmq.url`（如有账号密码）

## 启动安全校验
- `jwt_secret` 为空时，服务拒绝启动。
- `agent.enabled=true` 时，`agent.base_url / agent.api_key / agent.model` 必填。
- `rabbitmq.enabled=true` 时，`rabbitmq.url` 必填。
