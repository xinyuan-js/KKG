# KKG Backend

`kkg-backend` 是项目唯一后端服务，包含博客、OJ、管理、AI 题解、判题队列消费和统一认证。

## 目录

```text
cmd/api                 进程入口
internal/bootstrap      应用启动、迁移、种子数据
internal/config         统一配置
internal/router         统一路由
internal/session        Redis Session 双 token
internal/handler        博客与通用 HTTP 处理
internal/service        博客与通用业务逻辑
internal/repository     博客与通用数据访问
internal/model          统一用户与博客实体
internal/oj             OJ 业务域
migrations/merge        历史数据合并脚本
backups                 合并前数据库备份
```

## 启动

后端不再维护独立 compose。请从仓库根目录启动：

```bash
make dev-api
```

或：

```bash
docker compose --env-file .env up -d --build api
```

## 接口

```text
/api/v1/auth/*
/api/v1/posts/*
/api/v1/me/*
/api/v1/admin/*
/api/v1/oj/*
```

网关入口：

```text
/blog-api/*
/blog-api/api/v1/oj/*
```

## 认证

当前认证是 Redis Session 双 token：

1. Cookie 保存 `access_token`、`refresh_token`
2. token 是随机 opaque 字符串，不是 JWT
3. Redis 保存服务端 session
4. refresh 会轮换双 token
5. logout 会立即删除 session

## 数据保护

`backups/` 和 `migrations/merge/` 用于保留、迁移、校验旧数据。不要删除生产或本地迁移备份。
