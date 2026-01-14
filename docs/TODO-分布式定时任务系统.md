# 分布式定时任务系统 - 开发任务清单

> 基于 PRD V1.2.0 和技术方案 V1.1.0 生成
> 生成日期: 2025-01-09
> 项目代号: Quartz-Flow

---

## 任务统计

| 阶段 | 任务数 | 状态 |
|-----|-------|------|
| Phase 1: 基础设施 | 4 | ⏳ 待开始 |
| Phase 2: 服务端调度 | 5 | ⏳ 待开始 |
| Phase 3: 客户端引擎 | 4 | ⏳ 待开始 |
| Phase 4: 本地存储同步 | 3 | ⏳ 待开始 |
| Phase 5: API 接口 | 5 | ⏳ 待开始 |
| Phase 6: 前端基础架构 | 7 | ⏳ 待开始 |
| Phase 7: 前端公共组件 | 6 | ⏳ 待开始 |
| Phase 8: 前端-任务管理 | 4 | ⏳ 待开始 |
| Phase 9: 前端-其他模块 | 3 | ⏳ 待开始 |
| Phase 10: 监控可观测性 | 2 | ⏳ 待开始 |
| Phase 11: 部署测试 | 5 | ⏳ 待开始 |
| **合计** | **52** | - |

---

## Phase 1: 基础设施 - 数据库和协议

### 1.1 编写数据库迁移脚本
- [ ] 创建 `tb_task` 表（定时任务表）
- [ ] 创建 `tb_task_group` 表（主机分组表）
- [ ] 创建 `tb_task_group_relation` 表（任务分组关联表）
- [ ] 创建 `tb_execution` 表（执行记录表）
- [ ] 扩展 `tb_client` 表（添加 task_version 字段）
- [ ] 编写索引和约束

### 1.2 定义 Protobuf 协议
- [ ] 创建 `pkg/protocol/task.proto`
- [ ] 定义任务配置消息（TaskConfigPush, TaskConfigPull）
- [ ] 定义任务执行消息（TaskExecution, TaskProgress, TaskResult）
- [ ] 定义状态同步消息（DailyStatsReport, OfflineSyncRequest）

### 1.3 生成 Protobuf Go 代码
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       pkg/protocol/*.proto
```

### 1.4 版本管理
- [ ] 更新 go.mod 依赖版本
- [ ] 添加 `github.com/robfig/cron/v3`

---

## Phase 2: 服务端调度模块 (robfig/cron)

### 2.1 实现 Cron 调度器封装
**文件**: `pkg/scheduler/cron.go`

```go
type CronScheduler struct {
    cron           *cron.Cron
    taskDispatcher *TaskDispatcher
    jobRegistry    map[cron.EntryID]string
}

func NewCronScheduler(*zap.Logger, *TaskDispatcher) *CronScheduler
func (s *CronScheduler) Start()
func (s *CronScheduler) Stop()
func (s *CronScheduler) AddTask(task Task) (cron.EntryID, error)
func (s *CronScheduler) RemoveTask(taskID string) error
func (s *CronScheduler) GetNextRunTime(taskID string) (time.Time, error)
```

### 2.2 实现任务分发器
**文件**: `pkg/scheduler/dispatcher.go`

```go
type TaskDispatcher struct {
    dispatcher  *dispatcher.Dispatcher
    sessionMgr  *session.Manager
    taskStore   TaskStore
}

func (d *TaskDispatcher) Dispatch(ctx context.Context, task Task) error
func (d *TaskDispatcher) dispatchToClient(ctx, client, task) error
```

### 2.3 实现任务管理器
**文件**: `pkg/scheduler/manager.go`

```go
type TaskManager struct {
    cron          *CronScheduler
    dispatcher    *TaskDispatcher
    taskStore     TaskStore
    configVersion int64
}

func (m *TaskManager) Initialize(ctx) error
func (m *TaskManager) CreateTask(req *CreateTaskRequest) (*Task, error)
func (m *TaskManager) UpdateTask(req *UpdateTaskRequest) error
func (m *TaskManager) EnableTask(taskID string) error
func (m *TaskManager) DisableTask(taskID string) error
func (m *TaskManager) DeleteTask(taskID string) error
func (m *TaskManager) TriggerTask(taskID string) error
```

### 2.4 定义数据模型
**文件**: `pkg/model/task.go`, `pkg/model/execution.go`, `pkg/model/group.go`

```go
// Task 定时任务
type Task struct {
    ID              string
    Name            string
    Description     string
    ExecutorType    int
    ExecutorConfig  string
    CronExpr        string
    Timeout         int
    RetryCount      int
    RetryInterval   int
    Concurrency     int
    Status          int
    CreatedBy       string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// Execution 执行记录
type Execution struct {
    ID              string
    TaskID          string
    TaskName        string
    ClientID        string
    GroupID         int64
    ExecutionType   int
    Status          int
    StartTime       *time.Time
    EndTime         *time.Time
    Duration        int
    ExitCode        int
    Output          string
    ErrorMsg        string
    RetryCount      int
    CreatedAt       time.Time
}
```

### 2.5 实现数据访问层
**文件**: `pkg/store/task_store.go`, `pkg/store/execution_store.go`, `pkg/store/group_store.go`

```go
type TaskStore interface {
    Create(ctx, *Task) error
    Update(ctx, *Task) error
    Delete(ctx, string) error
    GetByID(ctx, string) (*Task, error)
    List(ctx, *ListParams) ([]*Task, int64, error)
    ListEnabled(ctx) ([]*Task, error)
    BindGroup(ctx, string, int64) error
    UnbindGroup(ctx, string, int64) error
    GetGroupIDs(ctx, string) ([]int64, error)
}
```

---

## Phase 3: 客户端任务引擎

### 3.1 实现任务引擎
**文件**: `pkg/client/task/engine.go`

```go
type Engine struct {
    clientID     string
    groupID      string
    dispatcher   *dispatcher.Dispatcher
    storage      *LocalStorage
    syncMgr      *SyncManager
    executorPool *executor.Pool
    localTasks   map[string]*LocalTask
    runningTasks map[string]context.CancelFunc
}

func (e *Engine) Start() error
func (e *Engine) Stop()
func (e *Engine) handleConfigPush(msg) error
func (e *Engine) handleExecution(msg) error
func (e *Engine) execute(ctx, execMsg, config) error
func (e *Engine) pullConfig() error
```

### 3.2 实现 Shell 执行器
**文件**: `pkg/client/executor/shell.go`

```go
type ShellExecutor struct{}

func (e *ShellExecutor) Execute(ctx, *ExecutionContext) *Result
```

### 3.3 实现 HTTP 执行器
**文件**: `pkg/client/executor/http.go`

```go
type HTTPExecutor struct {
    client *http.Client
}

func (e *HTTPExecutor) Execute(ctx, *ExecutionContext) *Result
```

### 3.4 实现执行器池
**文件**: `pkg/client/executor/pool.go`

```go
type Pool struct {
    executors map[ExecutorType]Executor
}

func (p *Pool) Register(ExecutorType, Executor)
func (p *Pool) GetExecutor(ExecutorType) (Executor, error)
```

---

## Phase 4: 客户端本地存储与同步

### 4.1 实现本地存储
**文件**: `pkg/client/task/storage.go`

```go
type LocalStorage struct {
    clientID  string
    basePath  string
    retention int
}

func (s *LocalStorage) SaveExecution(record *ExecutionRecord) error
func (s *LocalStorage) UpdateDailyStats(record *ExecutionRecord) error
func (s *LocalStorage) GetPendingSync() ([]ExecutionRecord, error)
func (s *LocalStorage) MarkSynced(executionID string) error
func (s *LocalStorage) CleanupExpired() error
func (s *LocalStorage) GetDailyStats(date string) (*DailyStats, error)
```

### 4.2 实现按天统计模块
**文件**: `pkg/client/task/stats.go`

```go
type DailyStats struct {
    Date     string
    ClientID string
    Tasks    map[string]*TaskStats
}

type TaskStats struct {
    TaskName       string
    SuccessCount   int32
    FailureCount   int32
    TimeoutCount   int32
    CancelledCount int32
    AvgDurationMs  int64
    MaxDurationMs  int32
    MinDurationMs  int32
    Hourly         map[string]int32
}
```

### 4.3 实现离线同步管理器
**文件**: `pkg/client/task/sync.go`

```go
type SyncManager struct {
    storage      *LocalStorage
    dispatcher   *dispatcher.Dispatcher
    clientID     string
    syncInterval time.Duration
}

func (m *SyncManager) Start(ctx)
func (m *SyncManager) OnReconnect()
func (m *SyncManager) syncPending()
func (m *SyncManager) syncDailyStats()
```

---

## Phase 5: API 接口

### 5.1 实现任务管理 API
**文件**: `pkg/api/task.go`

```go
func (api *TaskAPI) ListTasks(c *gin.Context)
func (api *TaskAPI) CreateTask(c *gin.Context)
func (api *TaskAPI) UpdateTask(c *gin.Context)
func (api *TaskAPI) DeleteTask(c *gin.Context)
func (api *TaskAPI) EnableTask(c *gin.Context)
func (api *TaskAPI) DisableTask(c *gin.Context)
func (api *TaskAPI) TriggerTask(c *gin.Context)
func (api *TaskAPI) GetNextRunTime(c *gin.Context)
```

### 5.2 实现执行监控 API
**文件**: `pkg/api/execution.go`

```go
func (api *ExecutionAPI) GetExecutionList(c *gin.Context)
func (api *ExecutionAPI) GetExecutionDetail(c *gin.Context)
func (api *ExecutionAPI) GetExecutionLogs(c *gin.Context)
func (api *ExecutionAPI) GetExecutionStats(c *gin.Context)
```

### 5.3 实现分组管理 API
**文件**: `pkg/api/group.go`

```go
func (api *GroupAPI) GetGroups(c *gin.Context)
func (api *GroupAPI) CreateGroup(c *gin.Context)
func (api *GroupAPI) UpdateGroup(c *gin.Context)
func (api *GroupAPI) DeleteGroup(c *gin.Context)
func (api *GroupAPI) GetGroupClients(c *gin.Context)
```

### 5.4 实现健康检查 API
**文件**: `pkg/api/health.go`

```go
func (api *HealthAPI) HealthCheck(c *gin.Context)
func (api *HealthAPI) ReadinessCheck(c *gin.Context)
func (api *HealthAPI) LivenessCheck(c *gin.Context)
```

### 5.5 实现 WebSocket 推送服务
**文件**: `pkg/api/ws.go`

```go
func (api *WSAPI) HandleWebSocket(c *gin.Context)
func (api *WSAPI) BroadcastTaskStatus(taskID string, status int)
func (api *WSAPI) BroadcastExecutionUpdate(execution *Execution)
```

---

## Phase 6: 前端基础架构

### 6.1 配置 Vite + Vue 3 + TypeScript 项目
```bash
npm create vite@latest quicflow-web -- --template vue-ts
cd quicflow-web
npm install
```

### 6.2 配置 Element Plus UI 组件库
```bash
npm install element-plus @element-plus/icons-vue
```

### 6.3 配置 Vue Router 路由
**文件**: `web/src/router/index.ts`

### 6.4 配置 Pinia 状态管理
```bash
npm install pinia
```

### 6.5 配置 Vue I18n 国际化
```bash
npm install vue-i18n
```

### 6.6 实现 Axios 请求封装
**文件**: `web/src/utils/request.ts`

### 6.7 实现 WebSocket 钩子
**文件**: `web/src/composables/useWebSocket.ts`

### 6.8 定义 TypeScript 类型
**文件**: `web/src/types/task.ts`, `web/src/types/execution.ts`

---

## Phase 7: 前端公共组件

### 7.1 实现通用表格组件
**文件**: `web/src/components/common/Table.vue`

### 7.2 实现通用弹窗组件
**文件**: `web/src/components/common/Dialog.vue`

### 7.3 实现状态标签组件
**文件**: `web/src/components/common/StatusTag.vue`

### 7.4 实现 Cron 编辑器组件
**文件**: `web/src/components/task/CronEditor.vue`
- 预设模板选择
- 实时校验
- 下次执行预览

### 7.5 实现执行日志查看器
**文件**: `web/src/components/task/ExecutionLog.vue`
- 实时日志追加
- 日志搜索
- 语法高亮

### 7.6 实现 ECharts 图表组件
**文件**: `web/src/components/chart/`
- `LineChart.vue` - 折线图
- `PieChart.vue` - 饼图
- `BarChart.vue` - 柱状图

```bash
npm install echarts
```

---

## Phase 8: 前端页面 - 任务管理

### 8.1 实现任务列表页面
**文件**: `web/src/views/task/List.vue`
- 搜索、筛选、分页
- 批量操作
- 实时状态更新（WebSocket）

### 8.2 实现任务表单页面
**文件**: `web/src/views/task/Form.vue`
- 基本信息配置
- Cron 表达式编辑
- 执行器配置
- 高级配置（重试、超时、分组）

### 8.3 实现任务详情页面
**文件**: `web/src/views/task/Detail.vue`
- 任务信息展示
- 执行历史
- 快捷操作

### 8.4 实现执行监控页面
**文件**: `web/src/views/task/Execution.vue`
- 执行记录列表
- 日志查看
- 统计图表

---

## Phase 9: 前端页面 - 其他模块

### 9.1 实现概览仪表盘
**文件**: `web/src/views/dashboard/Index.vue`
- 关键指标卡片
- 任务趋势图
- 成功率统计
- 最近执行记录

### 9.2 实现分组管理页面
**文件**: `web/src/views/group/List.vue`
- 分组列表
- 主机管理
- 拖拽操作

### 9.3 实现系统设置页面
**文件**: `web/src/views/settings/Index.vue`
- 通知设置
- 用户管理
- 系统配置

---

## Phase 10: 监控与可观测性

### 10.1 定义 Prometheus 指标
**文件**: `pkg/metrics/scheduler.go`

```go
var (
    ScheduledTasksTotal     // 调度总数
    TaskExecutionDuration   // 执行耗时分布
    TaskExecutionResult      // 执行结果统计
    ClientConnections        // 客户端连接数
    ConfigSyncDuration       // 配置同步耗时
)
```

### 10.2 实现链路追踪日志
- 执行请求 TraceID
- 日志上下文传递
- 采集器集成

---

## Phase 11: 部署与测试

### 11.1 编写部署配置
- `Dockerfile` (服务端)
- `docker-compose.yml`
- `nginx.conf` (前端反向代理)

### 11.2 编写 systemd 配置
**文件**: `scripts/quicflow-client.service`

### 11.3 集成测试
- 端到端任务调度测试
- 配置推送/拉取测试
- 离线同步测试

### 11.4 性能测试
- 并发调度压测
- 大量任务场景测试

### 11.5 编写文档
- 部署文档
- 用户手册
- API 文档

---

## 开发规范

### Go 代码规范
```bash
# 格式化
go fmt ./...

# Lint
golangci-lint run

# 测试
go test -v ./...
```

### 前端代码规范
```bash
# 格式化
npm run format

# Lint
npm run lint

# 类型检查
npm run type-check
```

### Git 提交规范
```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 重构
test: 测试相关
chore: 构建/工具链相关
```

---

## 关键依赖

```go
// go.mod
require (
    github.com/robfig/cron/v3 v3.0.1
    github.com/gin-gonic/gin v1.9.1
    github.com/quic-go/quic-go v0.40.0
    gorm.io/gorm v1.25.5
    gorm.io/driver/mysql v1.5.2
    go.uber.org/zap v1.26.0
    github.com/prometheus/client_golang v1.17.0
    google.golang.org/protobuf v1.31.0
)
```

```json
// package.json
{
  "dependencies": {
    "vue": "^3.4.0",
    "element-plus": "^2.5.0",
    "pinia": "^2.1.0",
    "vue-router": "^4.2.0",
    "axios": "^1.6.0",
    "echarts": "^5.5.0",
    "vue-i18n": "^9.8.0"
  }
}
```

---

**文档版本**: V1.0.0
**最后更新**: 2025-01-09
