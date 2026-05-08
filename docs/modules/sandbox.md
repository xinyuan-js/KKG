# 模块文档：kkg-sandbox

## 职责

`kkg-sandbox` 是独立判题执行器，负责：
1. 接收代码执行请求
2. 在受限环境中编译并运行
3. 返回结构化判题结果

## 对外接口

1. `GET /health`：健康检查  
2. `POST /executeCode`：执行入口（带 `auth` 头）

请求核心字段：
- `language`（当前仅支持 `go`）
- `code`
- `inputList`

## 执行模式

1. `container`（默认）  
   每个请求在临时容器中运行，隔离更强。
2. `direct`  
   本地进程执行，开发调试用，不建议线上。

## 安全边界

1. 使用 `--network none` 隔离网络。  
2. `--cap-drop ALL` + `no-new-privileges` 降权。  
3. 限制 CPU、内存、进程数、超时时间与输出大小。  
4. 仅开放必要目录作为工作区。  

## 风险提醒

当前容器模式依赖挂载 `/var/run/docker.sock`，属于高权限通道。生产环境建议：
1. 将沙盒部署到独立节点/独立 Docker daemon。  
2. 或改为“沙盒调度器 + 远端 runner 节点”模型。  
