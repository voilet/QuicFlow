# 部署中心功能实现 TODO

## 一、数据结构定义（P0 - 核心基础）

### 1.1 命令类型和基础结构
- [ ] **deploy-01**: 定义统一的部署命令类型和数据结构（`pkg/command/types.go`）
  - `deploy.shell` - Shell脚本部署
  - `deploy.tar` - tar.gz安装包部署
  - `deploy.docker` - Docker镜像部署
  - `deploy.rollback` - 回滚操作
  - `docker.monitor` - Docker监控上报

- [ ] **deploy-02**: 定义统一的部署状态枚举（`DeployStatus`）
  - 基础状态：`pending`, `executing`, `success`, `failed`, `timeout`
  - Shell特有：`script_running`, `health_check`
  - tar.gz特有：`uploading`, `extracting`, `installing`
  - Docker特有：`pulling`, `stopping`, `starting`
  - 回滚状态：`rolling_back`, `rolled_back`, `rollback_failed`

- [ ] **deploy-03**: 定义统一的部署回调结构（`DeployCallbackPayload`）
  - 包含：`callback_id`, `command_type`, `deploy_type`, `status`, `success`, `result`, `error`, `duration`, `timestamp`

- [ ] **deploy-04**: 定义统一的状态上报结构（`DeployStatusReport`）
  - 包含：`deploy_id`, `command_type`, `status`, `progress`, `stage`, `message`, `logs`, `timestamp`

- [ ] **deploy-05**: 定义部署结果结构
  - `DeployShellResult` - Shell部署结果
  - `DeployTarResult` - tar.gz部署结果
  - `DeployDockerResult` - Docker部署结果

- [ ] **deploy-06**: 定义健康检查结果结构（`HealthCheckResult`）
  - 支持：HTTP健康检查、TCP端口检查、自定义脚本检查
  - 包含：`success`, `check_type`, `response_time`, `retry_count`

- [ ] **deploy-07**: 定义回滚结果结构（`RollbackResult`）
  - 统一的回滚结果格式，包含成功/失败、耗时、错误信息

## 二、Shell部署实现（P0 - 核心功能）

### 2.1 客户端实现
- [ ] **deploy-08**: 实现Shell部署命令处理器（`cmd/client/router.go`）
  - 解析部署参数（脚本内容、工作目录、环境变量、超时时间）
  - 执行Shell脚本（带超时控制）
  - 捕获脚本输出（stdout/stderr）
  - 执行健康检查（如果配置）
  - 失败时执行回滚

- [ ] **deploy-09**: 实现Shell部署状态上报逻辑（客户端）
  - 每个阶段上报状态：`pending` → `executing` → `script_running` → `health_check` → `success/failed`
  - 上报进度百分比（0-100）
  - 上报阶段消息

- [ ] **deploy-10**: 实现Shell部署回调逻辑（客户端）
  - 部署完成后发送回调（`command.callback`事件）
  - 包含执行结果、健康检查结果、回滚结果（如果执行了）

### 2.2 回滚机制
- [ ] **deploy-17**: 实现自动回滚机制（客户端）
  - 检测部署失败条件：脚本执行失败、脚本超时、健康检查失败
  - 自动触发回滚逻辑
  - 回滚脚本执行后再次健康检查

- [ ] **deploy-18**: 实现回滚命令处理器（客户端）
  - 支持Shell回滚、tar.gz回滚、Docker回滚
  - 回滚脚本执行和结果上报

## 三、tar.gz部署实现（P1 - 重要功能）

### 3.1 大文件传输
- [ ] **deploy-11**: 实现大文件分块传输协议（`pkg/transport`）
  - 支持tar.gz文件传输
  - 单块大小：64KB-1MB（根据QUIC流特性）
  - 支持断点续传
  - 传输完成后校验MD5/SHA256

### 3.2 客户端实现
- [ ] **deploy-12**: 实现tar.gz部署命令处理器（客户端）
  - 接收文件分块并重组
  - 校验文件完整性（哈希校验）
  - 解压到指定目录
  - 执行安装脚本（如`install.sh`）
  - 启动服务并健康检查

- [ ] **deploy-13**: 实现tar.gz部署状态上报和回调（客户端）
  - 上传进度上报（0-30%）
  - 解压进度上报（30-60%）
  - 安装进度上报（60-90%）
  - 健康检查进度（90-100%）
  - 最终回调包含完整结果

## 四、Docker部署实现（P1 - 重要功能）

### 4.1 客户端实现
- [ ] **deploy-14**: 实现Docker部署命令处理器（客户端）
  - 拉取镜像（`docker pull`）
  - 停止旧容器（如果存在）
  - 启动新容器（`docker run`）
  - 健康检查（容器状态、HTTP/TCP检查）

- [ ] **deploy-15**: 实现Docker部署状态上报和回调（客户端）
  - 拉取镜像进度上报
  - 容器停止/启动状态上报
  - 健康检查状态上报
  - 最终回调包含容器ID、健康检查结果

### 4.2 Docker监控
- [ ] **deploy-26**: 实现Docker监控上报（客户端）
  - 定期上报容器状态（Running/Exited/Paused）
  - 上报健康检查结果
  - 上报资源使用（CPU/内存）

- [ ] **deploy-27**: 实现Docker监控处理器（服务端）
  - 接收并存储监控数据
  - 更新容器状态记录

- [ ] **deploy-28**: 实现Docker监控查询API（`pkg/api/http_server.go`）
  - `GET /api/docker/monitor/:client_id` - 查询客户端Docker监控数据

## 五、健康检查实现（P0 - 核心功能）

- [ ] **deploy-16**: 实现健康检查功能（客户端）
  - HTTP健康检查：端口 + 路径，支持超时和重试
  - TCP端口检查：连接测试
  - 自定义脚本检查：执行脚本并判断退出码
  - 健康检查超时：默认60秒，可配置
  - 健康检查重试：失败后重试3次，每次间隔10秒

## 六、服务端实现（P0 - 核心功能）

### 6.1 部署记录管理
- [ ] **deploy-19**: 实现部署记录管理（服务端）
  - `DeployRecord`结构：包含状态历史、结果、错误信息
  - 部署记录存储（内存或持久化）
  - 部署记录查询和更新

### 6.2 事件处理
- [ ] **deploy-20**: 实现部署回调处理器（`cmd/server/router.go`）
  - `handleDeployCallback`处理`command.callback`事件
  - 解析回调载荷
  - 更新部署记录状态
  - 如果失败且需要回滚，触发回滚
  - 通知等待的监听器

- [ ] **deploy-21**: 实现状态上报处理器（`cmd/server/router.go`）
  - `handleDeployStatusReport`处理`deploy.status`事件
  - 解析状态上报
  - 更新部署记录状态（实时）
  - 存储状态历史（可选）
  - 触发WebSocket推送（如果前端需要实时显示）

### 6.3 并发控制
- [ ] **deploy-22**: 实现部署并发控制（服务端）
  - 同一客户端同一时间只能有一个部署任务
  - 部署前检查锁状态
  - 部署完成后释放锁

## 七、API接口实现（P0 - 核心功能）

### 7.1 部署API（异步）
- [ ] **deploy-23**: 实现部署API接口（`pkg/api/http_server.go`）
  - `POST /api/deploy/shell` - Shell部署（立即返回202，通过回调获取结果）
  - `POST /api/deploy/tar` - tar.gz部署（立即返回202，通过回调获取结果）
  - `POST /api/deploy/docker` - Docker部署（立即返回202，通过回调获取结果）

### 7.2 查询API（同步）
- [ ] **deploy-24**: 实现部署查询API（`pkg/api/http_server.go`）
  - `GET /api/deploy/:deploy_id` - 查询部署状态和结果
  - `GET /api/deploy/:deploy_id/status` - 查询部署状态历史
  - `GET /api/deploy/:client_id/list` - 查询客户端的所有部署记录

### 7.3 回滚API（异步）
- [ ] **deploy-25**: 实现回滚API接口（`pkg/api/http_server.go`）
  - `POST /api/deploy/:deploy_id/rollback` - 手动触发回滚（立即返回202，通过回调获取结果）

## 八、增强功能（P2 - 可选功能）

### 8.1 版本管理
- [ ] **deploy-29**: 实现版本快照管理（客户端）
  - 部署前创建版本快照
  - 记录当前部署目录状态
  - 记录当前运行的进程/服务
  - 支持快速回滚到快照状态

### 8.2 安全约束
- [ ] **deploy-30**: 实现部署约束检查（客户端）
  - 脚本大小限制（最大100KB）
  - 危险命令检测（可选，白名单机制）
  - 权限检查（工作目录、执行用户）
  - 资源限制（CPU、内存）

### 8.3 可靠性增强
- [ ] **deploy-31**: 实现回调丢失恢复机制（客户端）
  - 重连后检查未完成的部署任务
  - 主动上报部署状态
  - 服务端超时后标记为失败并触发回滚

- [ ] **deploy-32**: 实现部署日志收集（客户端）
  - 收集部署过程中的日志
  - 通过状态上报或单独接口上报日志
  - 服务端存储日志

### 8.4 持久化
- [ ] **deploy-33**: 实现部署记录持久化（服务端）
  - 可选：存储到数据库（SQLite/PostgreSQL）
  - 可选：存储到文件系统（JSON文件）
  - 支持部署历史查询和清理

## 九、实现优先级

### P0（核心功能 - 必须实现）
1. 数据结构定义（deploy-01 至 deploy-07）
2. Shell部署实现（deploy-08 至 deploy-10）
3. 健康检查实现（deploy-16）
4. 自动回滚机制（deploy-17, deploy-18）
5. 服务端事件处理（deploy-19 至 deploy-22）
6. API接口实现（deploy-23 至 deploy-25）

### P1（重要功能 - 建议实现）
7. tar.gz部署实现（deploy-11 至 deploy-13）
8. Docker部署实现（deploy-14, deploy-15）
9. Docker监控实现（deploy-26 至 deploy-28）

### P2（增强功能 - 可选实现）
10. 版本快照管理（deploy-29）
11. 部署约束检查（deploy-30）
12. 回调丢失恢复（deploy-31）
13. 部署日志收集（deploy-32）
14. 部署记录持久化（deploy-33）

## 十、关键技术要点

### 10.1 回调机制
- 所有部署命令必须设置 `NeedCallback=true`
- 所有部署命令必须设置 `CallbackID=deploy_id`
- 回调使用 `command.callback` 事件类型
- 回调通过 `MESSAGE_TYPE_EVENT` 发送，`WaitAck=false`

### 10.2 状态上报
- 状态上报使用 `deploy.status` 事件类型
- 状态上报通过 `MESSAGE_TYPE_EVENT` 发送，`WaitAck=false`
- 每个部署阶段都要上报状态
- 服务端实时更新部署记录状态

### 10.3 部署流程
1. Web调用部署API → 服务端创建部署记录（状态：pending）
2. 服务端下发命令（NeedCallback=true, CallbackID=deploy_id）
3. 立即返回202 Accepted（不等待结果）
4. 客户端执行部署 → 过程中上报状态
5. 客户端完成部署 → 发送回调
6. 服务端接收回调 → 更新部署记录状态
7. 如果失败 → 自动触发回滚

### 10.4 回滚机制
- 回滚触发条件：脚本执行失败、脚本超时、健康检查失败
- 回滚策略：脚本内嵌回滚、独立回滚脚本、版本快照回滚
- 回滚执行后再次健康检查
- 回滚结果通过回调上报

## 十一、注意事项

1. **所有部署操作都必须有回调**：确保服务端能获取最终结果
2. **所有部署操作都必须上报状态**：确保服务端能实时了解执行进度
3. **统一的数据结构**：便于维护和扩展
4. **并发控制**：同一客户端同一时间只能有一个部署任务
5. **错误处理**：任何错误都要通过回调上报
6. **超时控制**：所有操作都要有超时限制
7. **健康检查**：部署后必须进行健康检查
8. **自动回滚**：部署失败或健康检查失败必须自动回滚

