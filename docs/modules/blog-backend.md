# 模块文档：kkg-blog-backend

## 职责

博客后端负责：
1. 用户认证与资料管理
2. 文章草稿/版本/发布
3. 评论、点赞、收藏、通知
4. 管理端用户与内容治理、审计日志
5. 博客搜索索引同步

## 目录结构

1. `cmd/api`：程序入口
2. `internal/router`：路由注册
3. `internal/handler`：HTTP 入参与响应
4. `internal/service`：业务规则与编排
5. `internal/repository`：数据访问层（MySQL）
6. `internal/model`：实体模型
7. `internal/search`：ES 客户端封装
8. `internal/storage`：MinIO 存储封装

## 核心业务逻辑

1. 文章系统  
   - 先草稿，后发布  
   - 版本可回滚  
   - 发布内容进入公开列表与检索索引

2. 用户权限  
   - `super_admin`：可管理所有用户与内容  
   - `admin`：可管理题目/推文/审计相关能力（按具体接口控制）  
   - `user`：仅处理本人内容

3. 软删除语义  
   - 管理侧“删除”是状态变更  
   - 非管理查询路径默认过滤隐藏/删除状态

## 与其他模块关系

1. 前端：通过 `/api/v1/*` 访问。  
2. OJ：通过题解绑定能力与博客文章关联。  
3. ES：文章/用户搜索依赖 ES 索引。  

## 运行与配置

1. 主要配置来源：环境变量（`.env`）。  
2. 最低依赖：MySQL、Redis、MinIO。  
3. 可选依赖：Elasticsearch（搜索可用时启用）。  
