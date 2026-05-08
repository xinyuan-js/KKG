# 模块文档：kkg-oj-backend

## 职责

OJ 后端负责：
1. 题目管理（增删改查、权限控制）
2. 提交记录管理（提交、自测、查询）
3. 判题任务调度（本地/队列模式）
4. 判题结果回写与状态流转
5. 题目与博客题解绑定
6. OJ 排行逻辑（含 Redis 排行能力）

## 目录结构

1. `cmd/server`：服务入口
2. `internal/app`：路由装配
3. `internal/handler`：HTTP 处理
4. `internal/service`：业务规则
5. `internal/infra/db`：MySQL 初始化
6. `internal/infra/mq`：RabbitMQ 封装
7. `internal/model/entity`：题目/提交等实体
8. `internal/config`：配置加载与校验

## 判题链路（推荐模式）

1. 用户提交代码 -> 写入提交记录（状态 `waiting`）  
2. 提交任务入 RabbitMQ (`oj.judge`)  
3. 消费者取任务，调用 `kkg-sandbox`  
4. 根据测试用例结果更新状态（AC/WA/CE/TLE/MLE/RE）  
5. 触发前端状态更新（轮询或 SSE）

## 配置与安全

1. 配置分层：
   - `config.yaml`：基础可提交配置
   - `config.local.yaml`：本地私有覆盖（忽略）
   - 环境变量：最高优先级
2. 启动校验：
   - `jwt_secret` 必填
   - `agent.enabled=true` 时必须给 `agent.api_key/base_url/model`

## 与其他模块关系

1. 前端通过 `/oj-api/*` 路径访问。  
2. 沙盒通过 `judge.sandbox_url` 调用。  
3. 博客通过题解绑定接口联动。  
