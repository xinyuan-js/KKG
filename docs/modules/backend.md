# 模块文档：kkg-backend

## 职责

`kkg-backend` 是项目唯一后端服务，承载：

1. 统一用户、登录、Redis Session 双 token
2. 博客文章、草稿、版本、互动、搜索
3. OJ 题目、提交、排行、题解绑定
4. 管理中心、用户管理、审计日志
5. AI 题解任务与自动发布
6. 判题队列消费与沙盒调用

## 目录

1. `cmd/api`：后端进程入口
2. `internal/bootstrap`：应用启动、迁移、种子数据、搜索同步
3. `internal/config`：唯一配置入口，读取根目录 `.env`
4. `internal/router`：统一路由注册
5. `internal/middleware`：统一 Session、内部接口鉴权
6. `internal/session`：Redis Session 双 token
7. `internal/handler`、`internal/service`、`internal/repository`、`internal/model`：博客与通用业务
8. `internal/oj`：OJ 业务域模块
9. `migrations/merge`：历史数据合并与校验脚本
10. `backups`：合并前数据库备份，保留用于数据恢复

## 接口

后端原始接口统一在：

```text
/api/v1/*
```

网关下的前端访问入口：

```text
/blog-api/*
/blog-api/api/v1/oj/*
```

## 鉴权

认证不再使用 JWT。当前机制是：

1. 登录成功后生成随机 `access_token` 和 `refresh_token`
2. Cookie 只保存 opaque token
3. Redis 保存 `auth:access:*` 和 `auth:refresh:*`
4. refresh 会轮换双 token
5. logout 会删除当前 session

## 启动

后端不再维护独立 `docker-compose.yml`。统一从仓库根目录启动：

```bash
docker compose --env-file .env up -d --build api
```

或直接使用：

```bash
make dev-api
```
