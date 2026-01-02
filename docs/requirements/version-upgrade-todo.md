# 版本升级增强功能 - 开发任务清单

## 概述
本文档基于 [需求规格说明书](./version-upgrade-enhancement.md) 生成的可执行任务清单。

---

## Phase 1: 核心功能 (P0)

### 1.1 版本升级自动继承 ClientID

#### 1.1.1 数据模型扩展
- [x] **TASK-001**: 在 `DeployTask` 模型中添加 `source_version` 和 `auto_select_clients` 字段
  - 文件: `pkg/release/models/models.go`
  - 修改 `DeployTask` 结构体
  - 添加迁移语句

- [x] **TASK-002**: 创建 `InstallationInfo` 响应结构体
  - 文件: `pkg/release/models/models.go`
  - 用于返回已安装目标信息

#### 1.1.2 数据库迁移
- [x] **TASK-003**: 编写数据库迁移脚本
  - 文件: `pkg/release/models/database.go`
  - 添加 `deploy_tasks` 表的新字段 (GORM AutoMigrate)

#### 1.1.3 API 实现
- [x] **TASK-004**: 实现 `GET /release/projects/:id/installations` 端点
  - 文件: `pkg/release/api/handlers.go`
  - 查询项目下所有 `TargetInstallation` 记录
  - 关联 `Target` 获取 `client_id` 和环境信息

- [x] **TASK-005**: 修改 `POST /release/tasks` 端点支持自动选择客户端
  - 文件: `pkg/release/api/handlers.go`
  - 当 `auto_select_clients=true` 时，自动查询已安装的 ClientID
  - 支持 `source_version` 过滤

#### 1.1.4 引擎逻辑
- [x] **TASK-006**: 修改发布引擎支持自动客户端选择
  - 文件: `pkg/release/engine/engine.go`
  - 在创建任务时自动填充 `ClientIDs`

---

### 1.2 脚本启动进程脱离机制

#### 1.2.1 执行器改进
- [x] **TASK-007**: 修改 `ReleaseExecuteParams` 添加进程脱离选项
  - 文件: `pkg/command/types.go`
  - 添加 `DetachProcess` 和 `DetachMethod` 字段

- [x] **TASK-008**: 修改客户端脚本执行逻辑支持进程脱离
  - 文件: `pkg/router/handlers/release.go`
  - 使用 `syscall.SysProcAttr{Setsid: true}` 创建新会话
  - 支持 systemd/nohup/setsid 三种方式

- [ ] **TASK-009**: 创建进程脱离辅助函数
  - 文件: `pkg/router/handlers/release.go`
  - 提供 `startDetached()` 工具函数

#### 1.2.2 脚本模板
- [ ] **TASK-010**: 创建标准化部署脚本模板
  - 文件: `docs/templates/deploy-script.sh`
  - 包含进程脱离最佳实践

---

### 1.3 容器名称前缀配置

#### 1.3.1 数据模型
- [x] **TASK-011**: 创建 `ContainerNamingConfig` 结构体
  - 文件: `pkg/release/models/models.go`
  - 包含 Prefix, Separator, Template 等字段

- [x] **TASK-012**: 在 `Project` 模型中添加 `ContainerNaming` 字段
  - 文件: `pkg/release/models/models.go`
  - 添加 JSONB 类型字段

#### 1.3.2 数据库迁移
- [x] **TASK-013**: 添加 `projects.container_naming` 迁移
  - 文件: `pkg/release/models/database.go` (GORM AutoMigrate)

#### 1.3.3 容器名称生成
- [x] **TASK-014**: 实现容器名称生成器
  - 文件: `pkg/release/executor/container.go`（新建）
  - 支持变量替换: ${PREFIX}, ${ENV}, ${VERSION}, ${TIMESTAMP}
  - 实现名称唯一性校验

- [ ] **TASK-015**: 修改容器部署流程使用新的命名规则
  - 文件: `pkg/release/executor/remote.go`
  - 在 `ExecuteContainerDeploy` 中应用命名配置

#### 1.3.4 API 扩展
- [ ] **TASK-016**: 修改项目 API 支持容器命名配置
  - 文件: `pkg/release/api/handlers.go`
  - 在创建/更新项目时处理 `container_naming`

---

## Phase 2: 上报功能 (P1)

### 2.1 进程采集和上报

#### 2.1.1 数据模型
- [x] **TASK-017**: 创建 `ProcessMonitorConfig` 和 `ProcessMatchRule` 结构体
  - 文件: `pkg/release/models/models.go`

- [x] **TASK-018**: 在 `Version` 模型中添加 `ProcessConfig` 字段
  - 文件: `pkg/release/models/models.go`

- [x] **TASK-019**: 创建 `ProcessReport` 和 `ProcessInfo` 模型
  - 文件: `pkg/release/models/models.go`

#### 2.1.2 数据库迁移
- [x] **TASK-020**: 创建 `process_reports` 表迁移
  - 文件: `pkg/release/models/database.go` (GORM AutoMigrate)
  - 包含索引创建

- [x] **TASK-021**: 添加 `versions.process_config` 迁移
  - 文件: `pkg/release/models/database.go` (GORM AutoMigrate)

#### 2.1.3 进程采集器
- [x] **TASK-022**: 创建进程采集器模块
  - 文件: `pkg/process/collector.go`（新建）
  - 依赖 `gopsutil` 库（使用原生 /proc 实现）
  - 实现按规则匹配进程

- [x] **TASK-023**: 实现进程匹配规则
  - 文件: `pkg/process/collector.go`
  - 支持 name, cmdline, pidfile, port 四种匹配方式

- [x] **TASK-024**: 实现进程信息采集
  - 文件: `pkg/process/collector.go`
  - 采集 PID, Name, Cmdline, CPU, Memory 等

#### 2.1.4 命令处理器
- [x] **TASK-025**: 添加 `process.collect` 命令类型
  - 文件: `pkg/command/types.go`
  - 添加参数和结果结构体

- [x] **TASK-026**: 实现 `ProcessCollect` 命令处理器
  - 文件: `pkg/router/handlers/process.go`（新建）
  - 调用进程采集器

- [x] **TASK-027**: 添加 `process.report` 命令类型
  - 文件: `pkg/command/types.go`

- [x] **TASK-028**: 实现 `ProcessReport` 命令处理器
  - 文件: `pkg/router/handlers/process.go`

#### 2.1.5 服务端 API
- [x] **TASK-029**: 实现 `POST /release/process-report` 端点
  - 文件: `pkg/release/api/handlers.go`
  - 接收并存储进程上报数据

- [x] **TASK-030**: 实现 `GET /release/projects/:id/processes` 端点
  - 文件: `pkg/release/api/handlers.go`
  - 查询项目下所有客户端的进程状态

#### 2.1.6 自动上报集成
- [ ] **TASK-031**: 在部署完成后自动触发进程采集
  - 文件: `pkg/release/engine/engine.go`
  - 部署成功后发送 `process.collect` 命令

- [x] **TASK-032**: 注册进程命令处理器
  - 文件: `pkg/router/handlers/register.go`
  - 添加新命令的注册

---

### 2.2 容器采集和上报

#### 2.2.1 数据模型
- [x] **TASK-033**: 创建 `ContainerReport` 和 `ContainerInfo` 模型
  - 文件: `pkg/release/models/models.go`

#### 2.2.2 数据库迁移
- [x] **TASK-034**: 创建 `container_reports` 表迁移
  - 文件: `pkg/release/models/database.go` (GORM AutoMigrate)

#### 2.2.3 容器采集器
- [x] **TASK-035**: 创建容器采集器模块
  - 文件: `pkg/container/collector.go`（新建）
  - 使用 Docker SDK（原生 HTTP API）采集容器信息

- [x] **TASK-036**: 实现容器资源统计采集
  - 文件: `pkg/container/collector.go`
  - 采集 CPU、内存、网络统计

- [x] **TASK-037**: 实现按前缀匹配项目
  - 文件: `pkg/container/collector.go`
  - 根据容器名称前缀关联项目

#### 2.2.4 命令处理器
- [x] **TASK-038**: 添加 `container.collect` 和 `container.report` 命令类型
  - 文件: `pkg/command/types.go`

- [x] **TASK-039**: 实现容器命令处理器
  - 文件: `pkg/router/handlers/container.go`（新建）

#### 2.2.5 服务端 API
- [x] **TASK-040**: 实现 `POST /release/container-report` 端点
  - 文件: `pkg/release/api/handlers.go`

- [x] **TASK-041**: 实现 `GET /release/projects/:id/containers` 端点
  - 文件: `pkg/release/api/handlers.go`

- [x] **TASK-042**: 实现 `GET /release/containers/overview` 端点
  - 文件: `pkg/release/api/handlers.go`
  - 全局容器概览统计

#### 2.2.6 容器命令注册
- [x] **TASK-043**: 注册容器命令处理器
  - 文件: `pkg/router/handlers/register.go`

---

## Phase 3: 增强功能 (P2)

### 3.1 进程心跳和告警

- [ ] **TASK-044**: 实现进程定期上报机制
  - 文件: `pkg/process/monitor.go`（新建）
  - 后台定时采集和上报

- [ ] **TASK-045**: 实现进程退出检测
  - 文件: `pkg/process/monitor.go`
  - 检测监控的进程是否退出

- [ ] **TASK-046**: 实现进程告警通知
  - 文件: `pkg/release/alert/process.go`（新建）
  - 进程退出、CPU/内存超限告警

### 3.2 容器告警

- [ ] **TASK-047**: 实现容器状态告警
  - 文件: `pkg/release/alert/container.go`（新建）
  - 容器停止、重启告警

### 3.3 K8s Pod 支持

- [ ] **TASK-048**: 创建 K8s Pod 采集器
  - 文件: `pkg/k8s/collector.go`（新建）
  - 使用 client-go 采集 Pod 信息

- [ ] **TASK-049**: 实现 Pod 状态上报
  - 文件: `pkg/router/handlers/k8s.go`（新建）

---

## 依赖项安装

- [x] **TASK-050**: ~~添加 `gopsutil` 依赖~~ (使用原生 /proc 实现，无需外部依赖)
  ```bash
  # 不需要：使用原生 /proc 实现
  ```

- [x] **TASK-051**: ~~添加 Docker SDK 依赖~~ (使用原生 HTTP API 实现，无需外部依赖)
  ```bash
  # 不需要：使用原生 Docker HTTP API
  ```

- [ ] **TASK-052**: 添加 K8s client-go 依赖（Phase 3）
  ```bash
  go get k8s.io/client-go
  ```

---

## 测试任务

### 单元测试
- [ ] **TEST-001**: 进程匹配规则测试
- [ ] **TEST-002**: 容器名称生成器测试
- [ ] **TEST-003**: 进程采集器测试
- [ ] **TEST-004**: 容器采集器测试

### 集成测试
- [ ] **TEST-005**: 版本升级自动选择客户端测试
- [ ] **TEST-006**: 进程脱离机制测试
- [ ] **TEST-007**: 进程上报 API 测试
- [ ] **TEST-008**: 容器上报 API 测试

### 端到端测试
- [ ] **TEST-009**: 完整部署流程测试（含进程上报）
- [ ] **TEST-010**: 容器部署流程测试（含容器上报）

---

## 文档任务

- [ ] **DOC-001**: 更新 API 文档
- [ ] **DOC-002**: 编写部署脚本模板指南
- [ ] **DOC-003**: 编写进程监控配置指南
- [ ] **DOC-004**: 编写容器命名配置指南

---

## 任务优先级矩阵

| 优先级 | 任务范围 | 预计任务数 |
|-------|---------|----------|
| P0 | TASK-001 ~ TASK-016 | 16 |
| P1 | TASK-017 ~ TASK-043 | 27 |
| P2 | TASK-044 ~ TASK-049 | 6 |
| 依赖 | TASK-050 ~ TASK-052 | 3 |
| 测试 | TEST-001 ~ TEST-010 | 10 |
| 文档 | DOC-001 ~ DOC-004 | 4 |

**总计: 66 个任务**

---

## 关键文件变更清单

### 需修改的现有文件
| 文件路径 | 变更类型 |
|---------|---------|
| `pkg/release/models/models.go` | 添加多个模型和字段 |
| `pkg/release/models/database.go` | 添加迁移逻辑 |
| `pkg/release/api/handlers.go` | 添加 6 个新端点 |
| `pkg/release/engine/engine.go` | 修改发布流程 |
| `pkg/release/executor/remote.go` | 修改容器部署 |
| `pkg/command/types.go` | 添加新命令类型 |
| `pkg/router/handlers/release.go` | 添加进程脱离逻辑 |
| `pkg/router/register.go` | 注册新命令 |

### 需创建的新文件
| 文件路径 | 描述 |
|---------|------|
| `pkg/process/collector.go` | 进程采集器 |
| `pkg/process/matcher.go` | 进程匹配规则 |
| `pkg/process/monitor.go` | 进程监控服务 |
| `pkg/container/collector.go` | 容器采集器 |
| `pkg/router/handlers/process.go` | 进程命令处理器 |
| `pkg/router/handlers/container.go` | 容器命令处理器 |
| `pkg/release/executor/container.go` | 容器名称生成器 |
| `pkg/release/alert/process.go` | 进程告警 |
| `pkg/release/alert/container.go` | 容器告警 |
| `docs/templates/deploy-script.sh` | 部署脚本模板 |

---

*任务清单版本: v1.0*
*生成日期: 2026-01-02*
*基于需求文档: version-upgrade-enhancement.md*
