# Git 规范与安全基线

## 1. 必做配置

在仓库根目录执行：

```bash
git config commit.template .gitmessage.txt
```

作用：统一提交信息结构，减少“无上下文提交”。

## 2. 已落地规则

1. `.gitignore`
   - 忽略本地密钥与私有配置：`.env`、`config.local.yaml` 等
   - 忽略构建缓存：`.cache`、`.next`、`node_modules`、`tmp`

2. `.gitattributes`
   - 统一文本换行符为 LF
   - 图片/字体按 binary 处理，避免误差异

3. `kkg-backend` 配置
   - `.env.example`：可提交模板
   - `.env`：本地私有（已忽略）
   - 环境变量：最高优先级

## 3. 敏感信息管理

1. 禁止提交：
   - API Key
   - 数据库真实密码
   - RabbitMQ/Redis 生产凭据

2. 推荐做法：
   - 模板放 `*.example.*`
   - 实值放本地 `*.local.*` 或 CI Secret
   - 已泄露密钥必须轮换

## 4. 提交与分支建议

1. 提交粒度：单一主题，避免“功能 + 重构 + 样式”混在一个提交。  
2. 分支命名建议：`feat/*`、`fix/*`、`refactor/*`、`chore/*`。  
3. 合并前至少保证：
   - `kkg-backend` 可编译并通过 `go test ./...`
   - 前端页面可启动并完成基本路由跳转
