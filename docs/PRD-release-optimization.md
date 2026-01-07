# QUIC-Flow 发布系统优化需求文档

> 基于当前功能分析，参考 Jenkins 等业界标准，从产品角度提出的系统优化方案
>
> **设计原则**: 轻量级、聚焦核心、避免过度设计
>
> 文档版本: v1.1
> 创建日期: 2026-01-07

---

## 目录

1. [执行摘要](#执行摘要)
2. [当前系统功能分析](#当前系统功能分析)
3. [与 Jenkins 功能对比](#与-jenkins-功能对比)
4. [核心模块优化需求](#核心模块优化需求)
5. [技术架构设计](#技术架构设计)
6. [实施路线图](#实施路线图)

---

## 执行摘要

### 背景

QUIC-Flow 当前已实现一套基于 QUIC 协议的分布式发布系统，支持多种部署类型（Docker、Kubernetes、脚本）、部署策略（滚动、金丝雀、蓝绿）和回调通知机制。

### 设计原则

> **轻量级优先**: 避免复杂架构，专注核心价值，快速迭代

| 原则 | 说明 |
|------|------|
| 聚焦核心 | 只做发布系统，不做完整 CI 平台 |
| 避免插件 | 内置最常用功能，不搞插件生态 |
| 简单集成 | 通过标准协议集成，不深度耦合 |
| 渐进增强 | 保持轻量内核，可选功能模块化 |

### 核心优化方向

| 模块 | 当前状态 | 目标状态 | 优先级 | 复杂度 |
|------|----------|----------|--------|--------|
| 构建系统 | 无 | 内置 Docker 构建 | P0 | 低 |
| 凭证管理 | 散落配置 | 统一加密存储 | P0 | 低 |
| 制品管理 | 无 | Docker 制品关联 | P0 | 低 |
| 权限系统 | 基础审批 | 项目级权限 | P0 | 低 |
| 日志系统 | 文件轮转 | VictoriaLogs 集成 | P1 | 低 |
| 监控告警 | 基础回调 | 简化告警规则 | P1 | 中 |
| 触发器 | 手动触发 | Webhook + 定时 | P1 | 低 |

### 不做的功能（保持轻量）

- ❌ 插件系统 - 增加复杂度
- ❌ 完整 CI 流水线 - Jenkins 更专业
- ❌ 复杂 RBAC - 项目级权限足够
- ❌ 多集群联邦 - 过度设计
- ❌ 移动端 APP - Web 响应式即可
- ❌ 可视化流水线编辑器 - YAML 配置更简单

---

## 当前系统功能分析

### 已实现功能

#### 1. 核心发布能力

```
部署类型:
├── Container (Docker)
├── Kubernetes
├── Script (自定义脚本)
└── Git Pull (代码同步)

部署策略:
├── Rolling Update (滚动更新)
├── Canary Release (金丝雀发布)
└── Blue-Green (蓝绿部署)
```

#### 2. 项目与环境管理

- **项目模型**: 支持多项目、多环境、多目标配置
- **环境隔离**: Dev/Test/Staging/Prod 多环境管理
- **目标管理**: 支持客户端节点、容器、Pod 等多种目标类型
- **配置继承**: 项目 → 环境 → 任务三级配置合并机制

#### 3. 流水线编排

- **阶段化执行**: 支持多阶段 Pipeline 定义
- **任务编排**: 每阶段支持多任务并行执行
- **条件执行**: 支持任务执行条件配置
- **审批流程**: 支持阶段级审批控制

#### 4. 版本管理

- **版本创建**: 支持基于 Git 的版本管理
- **版本追踪**: 完整的发布历史记录
- **版本对比**: 支持配置差异对比

#### 5. 变量系统

- **三级变量**: 项目级、环境级、任务级变量
- **变量覆盖**: 下层变量覆盖上层变量
- **敏感变量**: 支持密钥类型变量

#### 6. 回调通知

- **多渠道支持**: Webhook、飞书
- **事件触发**: 任务开始/成功/失败等事件
- **模板系统**: 可自定义消息模板
- **重试机制**: 失败自动重试

#### 7. 远程执行

- **QUIC 协议**: 基于 QUIC 的高性能远程执行
- **实时日志**: SSE 实时日志推送
- **进程上报**: 客户端进程状态上报
- **容器上报**: 容器状态上报

### 痛点与限制

| 痛点 | 描述 | 影响 | 解决方案 |
|------|------|------|----------|
| 无构建能力 | 需要手动准备镜像 | 流程断裂 | 内置 Docker 构建 |
| 凭证散落 | 敏感信息硬编码 | 安全风险 | 凭证中心 |
| 日志管理难 | 文件存储难查询 | 问题定位难 | VictoriaLogs 集成 |
| 手动触发 | 无自动化触发 | 效率低 | Webhook + 定时 |

---

## 与 Jenkins 功能对比

### 定位差异

| 维度 | Jenkins | QUIC-Flow |
|------|---------|-----------|
| 定位 | 完整 CI/CD 平台 | 轻量级 CD (持续部署) |
| 构建 | 强大 | 内置 Docker 构建 |
| 部署 | 需插件 | 原生多策略 |
| 架构 | 重量级、插件化 | 轻量级、一体化 |
| 适用场景 | 复杂 CI/CD | 快速部署发布 |

### 核心功能对比

| 功能 | Jenkins | QUIC-Flow (当前) | QUIC-Flow (目标) |
|------|---------|------------------|------------------|
| 部署策略 | 插件实现 | ✓ 原生支持 | ✓ 保持领先 |
| 多环境管理 | 需配置 | ✓ 原生支持 | ✓ 保持领先 |
| 蓝绿/金丝雀 | 插件 | ✓ 原生支持 | ✓ 保持领先 |
| Docker 构建 | ✓ | ✗ | ✓ 新增 |
| 凭证管理 | ✓ | ✗ | ✓ 新增 |
| Webhook 触发 | ✓ | ✗ | ✓ 新增 |
| 定时触发 | ✓ | ✗ | ✓ 新增 |
| 日志查询 | ✓ 文件 | ✗ 文件轮转 | ✓ VictoriaLogs |
| 回滚 | 需配置 | ✓ 原生支持 | ✓ 保持领先 |

> **策略**: 不追求 Jenkins 功能完整性，专注于部署领域，做轻量好用的 CD 平台

---

## 核心模块优化需求

### 模块一: Docker 构建 (P0)

#### 1.1 内置构建能力

**设计原则**: 只做 Docker 镜像构建，不追求通用构建系统

**功能范围**:

| 功能 | 说明 | 复杂度 |
|------|------|--------|
| Docker 构建 | 调用 docker build 构建镜像 | 低 |
| 构建日志 | 实时推送构建日志 | 低 |
| 镜像推送 | 推送到指定镜像仓库 | 低 |
| 构建缓存 | 支持 BuildKit 缓存 | 低 |

**数据模型**:

```go
// 扩展 Project 模型
type Project struct {
    // ... 现有字段
    BuildConfig     *BuildConfig    `gorm:"type:jsonb" json:"build_config,omitempty"`
}

type BuildConfig struct {
    Enabled         bool            `json:"enabled"`           // 是否启用构建
    Dockerfile      string          `json:"dockerfile"`        // Dockerfile 路径
    BuildContext    string          `json:"build_context"`     // 构建上下文
    BuildArgs       map[string]string `json:"build_args"`     // 构建参数
    TargetImage     string          `json:"target_image"`      // 目标镜像模板
    RegistryID      *string         `json:"registry_id"`       // 关联凭证
}

// 构建记录
type BuildRecord struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    ProjectID       string          `json:"project_id"`
    VersionID       string          `json:"version_id"`
    Status          string          `json:"status"`           // running/success/failed
    Image           string          `json:"image"`            // 构建的镜像
    Digest          string          `json:"digest"`           // 镜像摘要
    Duration        int64           `json:"duration"`         // 构建时长
    StartedAt       *time.Time      `json:"started_at"`
    CompletedAt     *time.Time      `json:"completed_at"`
    Error           string          `json:"error,omitempty"`
}
```

**API 设计**:

```
POST   /api/release/projects/:id/build           # 触发构建
GET    /api/release/builds/:id                   # 获取构建状态
GET    /api/release/builds/:id/logs/stream       # 构建日志流
```

#### 1.2 镜像制品关联

**功能**: 自动关联构建产物与版本

```go
// 镜像制品（简化版）
type ImageArtifact struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    VersionID       string          `json:"version_id"`
    Image           string          `json:"image"`            // 完整镜像地址
    Digest          string          `json:"digest"`           // sha256:xxx
    Size            int64           `json:"size"`             // 镜像大小
    CreatedAt       time.Time       `json:"created_at"`
}
```

---

### 模块二: 凭证中心 (P0)

#### 2.1 统一凭证管理

**设计原则**: 简单安全，支持常用凭证类型

**支持的凭证类型**:

| 类型 | 场景 | 优先级 |
|------|------|--------|
| 用户名密码 | Docker Hub/Git 认证 | P0 |
| 访问令牌 | GitLab/GitHub API | P0 |
| SSH 私钥 | Git SSH 克隆 | P1 |

**数据模型**:

```go
type Credential struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    Name            string          `json:"name"`             // 凭证名称
    Type            string          `json:"type"`             // username_token/ssh/access_token
    Description     string          `json:"description"`
    EncryptedData   string          `json:"-" gorm:"type:text"` // AES-256 加密
    Scope           string          `json:"scope"`            // global/project
    ProjectID       *string         `json:"project_id,omitempty"`
    CreatedBy       string          `json:"created_by"`
    CreatedAt       time.Time       `json:"created_at"`
    LastUsedAt      *time.Time      `json:"last_used_at"`
}

// 使用审计（轻量级）
type CredentialLog struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    CredentialID    string          `json:"credential_id"`
    Action          string          `json:"action"`           // create/use/update/delete
    UserID          string          `json:"user_id"`
    IPAddress       string          `json:"ip_address"`
    CreatedAt       time.Time       `json:"created_at"`
}
```

**加密方案**:

- 算法: AES-256-GCM
- 密钥来源: 环境变量 `QUIC_FLOW_SECRET_KEY`
- 密钥轮换: 重启时自动检测

**API 设计**:

```
POST   /api/release/credentials                    # 创建凭证
GET    /api/release/credentials                    # 列出凭证
PUT    /api/release/credentials/:id                # 更新凭证
DELETE /api/release/credentials/:id                # 删除凭证
GET    /api/release/credentials/:id/logs           # 使用日志
```

---

### 模块三: 项目权限 (P0)

#### 3.1 简化权限模型

**设计原则**: 项目级权限足够，不搞复杂 RBAC

**角色定义**:

| 角色 | 权限范围 |
|------|----------|
| `owner` | 项目完全控制 |
| `maintainer` | 配置和部署 |
| `developer` | 开发环境部署 |
| `viewer` | 只读访问 |

**数据模型**:

```go
type ProjectMember struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    ProjectID       string          `json:"project_id" gorm:"index"`
    UserID          string          `json:"user_id" gorm:"index"`
    Role            string          `json:"role"` // owner/maintainer/developer/viewer
    AddedBy         string          `json:"added_by"`
    AddedAt         time.Time       `json:"added_at"`

    // 关联
    User            *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type User struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    Username        string          `gorm:"uniqueIndex" json:"username"`
    DisplayName     string          `json:"display_name"`
    Email           string          `json:"email"`
    IsAdmin         bool            `json:"is_admin"`          // 超级管理员
    CreatedAt       time.Time       `json:"created_at"`
}
```

**权限矩阵**:

| 操作 | owner | maintainer | developer | viewer |
|------|-------|------------|-----------|--------|
| 修改项目配置 | ✓ | ✓ | ✗ | ✗ |
| 管理成员 | ✓ | ✗ | ✗ | ✗ |
| 部署生产 | ✓ | ✓ | ✗ | ✗ |
| 部署开发 | ✓ | ✓ | ✓ | ✗ |
| 查看日志 | ✓ | ✓ | ✓ | ✓ |
| 触发构建 | ✓ | ✓ | ✓ | ✗ |

---

### 模块四: VictoriaLogs 日志 (P1)

#### 4.1 日志集成方案

**选择 VictoriaLogs 的原因**:

| 优势 | 说明 |
|------|------|
| 轻量级 | 单二进制，资源占用低 |
| 高性能 | 相比 ES 资源占用减少 10x |
| 兼容性 | 支持 Loki API，易于集成 |
| 成本低 | 开源免费，存储压缩率高 |

**架构设计**:

```
┌─────────────┐     ┌─────────────┐     ┌──────────────────┐
│ QUIC-Flow   │────▶│ VictoriaLogs│◀────│   Grafana        │
│   Server    │     │   (可选)    │     │   (可选可视化)   │
└─────────────┘     └─────────────┘     └──────────────────┘
       │
       │ 1. 实时日志 (SSE)
       ▼
┌─────────────┐
│   前端展示   │
└─────────────┘
```

**功能范围**:

| 功能 | 实现方式 | 优先级 |
|------|----------|--------|
| 实时日志流 | SSE 直接推送 | P0 (已有) |
| 日志查询 | 集成 VictoriaLogs | P1 |
| 日志导出 | VL 导出功能 | P2 |

**集成配置**:

```go
type LogsConfig struct {
    // 后端存储 (可选)
    VictoriaLogs  *VictoriaLogsConfig `json:"victoria_logs,omitempty"`

    // 本地存储 (默认)
    LocalStorage  *LocalLogsConfig    `json:"local_storage,omitempty"`
}

type VictoriaLogsConfig struct {
    Enabled        bool   `json:"enabled"`
    URL            string `json:"url"`              // http://victoria-logs:9428
    AccountID      string `json:"account_id"`       // 0
    retention      int    `json:"retention"`        // 保留天数
}

// 日志写入接口
type LogWriter interface {
    Write(taskID string, line string) error
    Close(taskID string) error
}
```

**API 集成**:

```
# 查询日志 (通过 VictoriaLogs)
GET    /api/release/logs/query?task_id=xxx&filter=error

# 导出日志
GET    /api/release/logs/:id/export
```

---

### 模块五: 自动化触发器 (P1)

#### 5.1 Webhook 触发

**功能**: Git 推送自动触发部署

**支持平台**:

| 平台 | 优先级 |
|------|--------|
| GitHub | P0 |
| GitLab | P0 |
| Gitee | P1 |

**数据模型**:

```go
type WebhookTrigger struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    ProjectID       string          `json:"project_id"`
    Name            string          `json:"name"`
    Enabled         bool            `json:"enabled"`

    // 触发条件
    Source          string          `json:"source"`           // github/gitlab/gitee
    BranchFilter    []string        `json:"branch_filter"`    // 分支过滤，如 ["main", "release/*"]
    EventTypes      []string        `json:"event_types"`      // push/tag_create

    // 触发动作
    Action          string          `json:"action"`           // deploy/build
    TargetEnv       string          `json:"target_env"`       // 目标环境

    // Webhook 信息
    Secret          string          `json:"-"`                // 验证密钥
    URL             string          `json:"url"`              // 回调 URL
}

// 触发记录
type TriggerRecord struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    TriggerID       string          `json:"trigger_id"`
    Source          string          `json:"source"`
    Branch          string          `json:"branch"`
    Commit          string          `json:"commit"`
    Committer       string          `json:"committer"`
    TaskID          *string         `json:"task_id,omitempty"`
    Status          string          `json:"status"`           // success/failed/skipped
    TriggeredAt     time.Time       `json:"triggered_at"`
}
```

**Webhook 处理流程**:

```
GitHub Push ──▶ Webhook Receiver ──▶ 验证签名
                                      │
                                      ▼
                                  匹配触发器
                                      │
                                      ▼
                              创建部署任务
                                      │
                                      ▼
                              关联触发记录
```

#### 5.2 定时触发

**功能**: Cron 表达式定时触发部署

```go
type ScheduleTrigger struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    ProjectID       string          `json:"project_id"`
    Name            string          `json:"name"`
    Enabled         bool            `json:"enabled"`

    // Cron 配置
    Cron            string          `json:"cron"`             // "0 2 * * *" 每天凌晨2点
    Timezone        string          `json:"timezone"`         // "Asia/Shanghai"

    // 触发动作
    Action          string          `json:"action"`           // deploy
    TargetEnv       string          `json:"target_env"`

    // 状态
    NextRun         *time.Time      `json:"next_run"`
    LastRun         *time.Time      `json:"last_run"`
}
```

**API 设计**:

```
# Webhook 触发
POST   /api/release/webhooks/:id/trigger        # 接收 Webhook
GET    /api/release/projects/:id/webhooks       # 列出 Webhook
POST   /api/release/projects/:id/webhooks       # 创建 Webhook

# 定时触发
GET    /api/release/projects/:id/schedules      # 列出定时任务
POST   /api/release/projects/:id/schedules      # 创建定时任务
```

---

### 模块六: 简化告警 (P1)

#### 6.1 告警规则

**设计原则**: 简单够用，不搞复杂告警引擎

**告警场景**:

| 场景 | 级别 | 默认通知 |
|------|------|----------|
| 生产部署失败 | P0 | 立即通知 |
| 金丝雀失败 | P0 | 立即通知 |
| 构建失败 | P1 | 汇总通知 |
| 部署超时 | P1 | 立即通知 |

**数据模型**:

```go
type AlertRule struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    ProjectID       *string         `json:"project_id,omitempty"` // 空=全局规则
    Name            string          `json:"name"`

    // 触发条件
    EventType       string          `json:"event_type"`       // deploy_failed/build_failed/timeout
    Environment     string          `json:"environment"`      // prod/staging/dev/all

    // 告警配置
    Severity        string          `json:"severity"`         // critical/warning/info
    Enabled         bool            `json:"enabled"`

    // 通知渠道 (使用已有的回调系统)
    CallbackIDs     []string        `json:"callback_ids"`     // 关联的回调配置
}

// 告警记录
type AlertRecord struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    RuleID          string          `json:"rule_id"`
    Severity        string          `json:"severity"`
    Title           string          `json:"title"`
    Message         string          `json:"message"`
    TaskID          string          `json:"task_id"`
    Notified        bool            `json:"notified"`
    CreatedAt       time.Time       `json:"created_at"`
}
```

**集成现有回调系统**:

告警直接使用已有的 `CallbackConfig` 和 `CallbackManager`，无需新增通知渠道。

**API 设计**:

```
GET    /api/release/alerts/rules                     # 列出告警规则
POST   /api/release/alerts/rules                     # 创建规则
PUT    /api/release/alerts/rules/:id                 # 更新规则
GET    /api/release/alerts/history                   # 告警历史
```

---

## 技术架构设计

### 架构原则

> **单体优先**: 保持单体架构，避免微服务复杂性
> **模块化**: 代码模块化设计，便于维护
> **可插拔**: 可选功能通过开关控制

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend                             │
│                    (Vue 3 + Element Plus)                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       API Layer                             │
│                      (Gin Framework)                        │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│  Project    │  Build      │  Deploy     │   Trigger         │
│  APIs       │  APIs       │  APIs       │   APIs            │
└─────────────┴─────────────┴─────────────┴───────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Core Services                         │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│  Engine     │  Builder    │  Credential │   Webhook         │
│  (部署引擎)  │  (构建)     │  (凭证)     │   (触发器)        │
└─────────────┴─────────────┴─────────────┴───────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure                           │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│ PostgreSQL  │   Docker    │VictoriaLogs │  Callback         │
│  (元数据)   │  (构建)     │  (日志)     │  (通知)           │
└─────────────┴─────────────┴─────────────┴───────────────────┘
```

### 目录结构

```
pkg/release/
├── api/                    # API 层
│   ├── handlers.go         # 主 API 处理器
│   ├── build_api.go        # 构建 API
│   ├── credential_api.go   # 凭证 API
│   └── webhook_api.go      # Webhook API
├── engine/                 # 核心引擎
│   └── engine.go           # 部署引擎
├── builder/                # 构建模块 (新增)
│   ├── docker.go           # Docker 构建
│   └── builder.go          # 构建接口
├── credential/             # 凭证模块 (新增)
│   ├── crypto.go           # 加密/解密
│   └── manager.go          # 凭证管理
├── trigger/                # 触发器模块 (新增)
│   ├── webhook.go          # Webhook 处理
│   └── scheduler.go        # 定时任务
├── alert/                  # 告警模块
│   └── rule.go             # 告警规则 (新增)
├── models/                 # 数据模型
│   ├── models.go           # 现有模型
│   ├── build.go            # 构建模型 (新增)
│   └── credential.go       # 凭证模型 (新增)
└── log/                    # 日志模块
    ├── local.go            # 本地日志
    └── victorialogs.go     # VictoriaLogs (新增)
```

### 核心接口设计

```go
// 构建器接口
type Builder interface {
    Build(ctx context.Context, req *BuildRequest) (*BuildResult, error)
    GetLogs(ctx context.Context, buildID string) (<-chan LogLine, error)
    Cancel(ctx context.Context, buildID string) error
}

// Docker 构建实现
type DockerBuilder struct {
    dockerClient  *client.Client
    credManager   *credential.Manager
}

// 凭证管理器
type CredentialManager struct {
    db            *gorm.DB
    cipher        Cipher
}

// 触发器接口
type Trigger interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

### 配置管理

```go
type Config struct {
    // 数据库
    Database   DatabaseConfig   `json:"database"`

    // 服务器
    Server     ServerConfig     `json:"server"`

    // 构建配置
    Build      BuildConfig      `json:"build"`

    // 日志配置
    Logs       LogsConfig       `json:"logs"`

    // 功能开关
    Features   FeatureFlags     `json:"features"`
}

type FeatureFlags struct {
    BuildEnabled      bool  `json:"build_enabled"`       // 构建功能
    VictoriaLogs      bool  `json:"victoria_logs"`       // VictoriaLogs 集成
    WebhookEnabled    bool  `json:"webhook_enabled"`     // Webhook 触发
    ScheduleEnabled   bool  `json:"schedule_enabled"`    // 定时触发
}
```

---

## 实施路线图

### Phase 1: 基础能力 (2个月)

**目标**: 补齐核心缺失功能

| 模块 | 功能 | 工作量 |
|------|------|--------|
| 凭证中心 | 加密存储、基础类型 | 1周 |
| Docker 构建 | 镜像构建和推送 | 2周 |
| 镜像制品 | 与版本关联 | 1周 |
| 项目权限 | 简化权限模型 | 1周 |
| 前端集成 | 新功能 UI | 2周 |

### Phase 2: 自动化 (1个月)

**目标**: 提升自动化水平

| 模块 | 功能 | 工作量 |
|------|------|--------|
| Webhook 触发 | GitHub/GitLab 集成 | 1周 |
| 定时触发 | Cron 任务 | 1周 |
| 告警规则 | 简化告警 | 1周 |

### Phase 3: 可观测性 (1个月)

**目标**: 完善日志和监控

| 模块 | 功能 | 工作量 |
|------|------|--------|
| VictoriaLogs | 日志集成 | 1周 |
| 日志查询 | 前端界面 | 1周 |
| 指标采集 | 基础指标 | 1周 |

### 总体时间线

```
Month 1              Month 2              Month 3              Month 4
┌────────┬────────┐  ┌────────┬────────┐  ┌────────┬────────┐  ┌────────┬────────┐
│ 凭证   │ 构建   │  │ 权限   │ Webhook│  │ VictoriaLogs    │  │  优化  │        │
│ 中心   │        │  │        │ 触发   │  │                 │  │  收尾  │        │
└────────┴────────┘  └────────┴────────┘  └────────┴────────┘  └────────┴────────┘
     Phase 1                                Phase 2             Phase 3
```

---

## 附录

### A. 术语表

| 术语 | 定义 |
|------|------|
| CD | 持续部署 Continuous Deployment |
| Canary | 金丝雀发布，渐进式灰度 |
| Blue-Green | 蓝绿部署，零停机切换 |
| Rolling | 滚动更新，逐步替换 |
| Artifact | 制品，构建产物 |

### B. 参考资源

- [VictoriaLogs 文档](https://docs.victoriametrics.com/VictoriaLogs/)
- [Docker Build API](https://docs.docker.com/engine/api/)
- [GitHub Webhooks](https://docs.github.com/en/developers/webhooks-and-events)
- [ArgoCD](https://argoproj.github.io/argo-cd/) - K8s 部署参考
