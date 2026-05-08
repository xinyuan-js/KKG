# 博客系统项目需求总览（V1）

## 1. 项目定位

- 目标: 构建基于 Gin 的前后端分离博客系统，支持内容创作、审核发布、互动与后台管理。
- 架构: `Gin API + Next.js 主站 + 管理后台`。
- 原则: 高可维护、可扩展、可观测、可运维。

## 2. 技术与中间件基线

- 后端: Go + Gin（RESTful API）
- 前端: Next.js + React + TypeScript
- 数据层: MySQL
- 缓存与会话: Redis
- 对象存储: MinIO
- 消息队列: RabbitMQ（基础）/ Kafka（事件流增强）
- 搜索: Elasticsearch
- 可观测: Prometheus + Grafana + Jaeger

## 3. 角色与权限（RBAC）

- 游客: 浏览内容
- 用户: 登录、评论、点赞、收藏
- 作者: 文章创作、草稿管理、提审
- 编辑: 内容审核、推荐管理
- 管理员: 用户管理、权限管理、系统配置、审计

## 4. 核心业务模块

- 用户与认证: 注册、登录、JWT + Refresh Token、会话管理
- 内容管理: 文章 CRUD、草稿、发布、下线、软删除
- 分类与标签: 分类树、标签体系、SEO 友好链接
- 评论互动: 评论审核、点赞、收藏、阅读统计
- 管理后台: 仪表盘、内容审核、审计日志

## 5. 博客编写子系统设计

- 存储: `raw_content`（Markdown 源文）+ `html_content`（渲染产物）
- 状态机: `draft -> reviewing -> published -> offline/rejected/deleted`
- 能力点:
  - 自动保存（防丢稿）
  - 版本快照与回滚
  - 提审与审核流
  - 定时发布
- 一致性: 发布后进行缓存失效，避免脏读

## 6. API 约定（V1）

- 路由前缀: `/api/v1`
- 基础接口:
  - `POST /posts` 新建草稿
  - `PUT /posts/:id/draft` 保存草稿
  - `POST /posts/:id/submit-review` 提审
  - `POST /posts/:id/publish` 发布
  - `POST /posts/:id/offline` 下线
  - `GET /posts/:id/versions` 版本列表
  - `POST /posts/:id/rollback/:version` 回滚
- 规范: 统一响应体、统一错误码、统一鉴权中间件

## 7. 数据库最小模型（V1）

- `users`: 用户与角色
- `posts`: 文章主体（含状态、发布时间）
- `post_versions`: 版本快照
- `post_categories` / `post_tags`: 分类标签关系
- `post_audit_logs`: 审核与操作审计

## 8. 非功能需求

- 安全: 密码哈希、XSS 防护、SQL 注入防护、限流
- 性能: 热点缓存、分页优化、异步解耦
- 稳定性: 健康检查、优雅停机、幂等设计
- 可观测性: 指标、日志、链路追踪

## 9. 里程碑

- M1: 用户认证 + 文章发布 + 分类标签 + 基础后台
- M2: 评论/点赞/收藏 + Redis 缓存 + 审计日志
- M3: 搜索 + 消息队列 + 对象存储完善 + 监控告警
- M4: 容器化发布、CI/CD、读写分离与高可用增强
