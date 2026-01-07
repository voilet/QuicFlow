# QUIC-Flow 极简发布系统 PRD

> **核心理念**: 纯部署工具，不做构建，保持轻量
>
> 设计原则: 只做必须的，其他交给专业工具
>
> 文档版本: v2.0 (Minimal)
> 创建日期: 2026-01-07

---

## 目录

1. [执行摘要](#执行摘要)
2. [系统定位](#系统定位)
3. [核心功能](#核心功能)
4. [模块设计](#模块设计)
5. [技术架构](#技术架构)
6. [实施计划](#实施计划)

---

## 执行摘要

### 背景

QUIC-Flow 当前已具备完整的部署能力（多策略、多环境、多目标），但缺少三个基础能力：
1. **凭证管理散乱** - 镜像仓库密码、SSH Key 硬编码
2. **手动触发效率低** - 每次 push 后需手动操作
3. **权限控制缺失** - 任何人都能部署

### 设计原则

> **专注部署，不做构建**
> **简单够用，拒绝过度设计**

| 原则 | 说明 |
|------|------|
| 不做构建 | GitLab CI / GitHub Actions 更专业 |
| 不做复杂权限 | 两档够用：可部署 / 只读 |
| 不做大日志系统 | 文件日志 + grep |
| 复用现有能力 | 回调通知、部署引擎已有 |

### 核心优化（3 周完成）

| 模块 | 功能 | 优先级 | 工作量 |
|------|------|--------|--------|
| 凭证中心 | 统一加密存储凭证 | P0 | 1周 |
| Webhook 触发 | Git push 自动部署 | P0 | 1周 |
| 简化权限 | 项目成员 + 两档权限 | P0 | 1周 |

---

## 系统定位

### 不是什么

```
❌ 不是 CI 平台         ← 用 GitLab CI / GitHub Actions
❌ 不是构建系统         ← 用 Docker Hub / Harbor
❌ 不是日志平台         ← 用 grep / VictoriaLogs（可选）
❌ 不是权限中台         ← 只做项目级权限
```

### 是什么

```
✅ 纯 CD 工具          ← 专注部署执行
✅ 部署网关            ← 接收 Webhook 触发部署
✅ 凭证保险箱          ← 安全存储敏感信息
✅ 轻量好部署          ← 单二进制，依赖少
```

### 与业界工具对比

| 工具 | 定位 | QUIC-Flow 的选择 |
|------|------|------------------|
| Jenkins | CI/CD 全能 | 不做构建，只做部署 |
| GitLab CI | CI 为主 | GitLab 构建 + QUIC-Flow 部署 |
| ArgoCD | K8s 专属 | 支持更多部署类型 |
| Flux | GitOps | QUIC-Flow 更灵活 |

---

## 核心功能

### 已有功能（保持）

```
部署类型:
├── Docker (容器)
├── Kubernetes (Pod)
├── Script (脚本)
└── Git Pull (代码同步)

部署策略:
├── Rolling Update (滚动)
├── Canary (金丝雀)
└── Blue-Green (蓝绿)

其他:
├── 多环境管理
├── 审批流程
├── 变量系统
├── 回调通知 (已有)
└── 实时日志 (已有)
```

### 新增功能（本 PRD）

```
凭证中心:
├── 镜像仓库凭证 (Docker Hub / Harbor)
├── Git 凭证 (SSH / Token)
└── 统一加密存储

Webhook 触发:
├── GitHub Webhook
├── GitLab Webhook
└── 自动创建部署任务

权限控制:
├── 项目成员管理
└── 两档权限 (可部署 / 只读)
```

---

## 模块设计

### 模块一: 凭证中心

#### 1.1 设计原则

> **简单安全**: 支持常用凭证类型，AES-256 加密

#### 1.2 支持的凭证类型

| 类型 | 用途 | 示例 |
|------|------|------|
| `docker_registry` | 镜像仓库认证 | Docker Hub, Harbor, GitLab Registry |
| `git_ssh` | Git SSH 克隆 | GitHub SSH, GitLab SSH |
| `git_token` | Git API 访问 | GitHub PAT, GitLab Token |
| `username_password` | 通用用户名密码 | 自定义服务认证 |

#### 1.3 数据模型

```go
// 凭证
type Credential struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    Name            string          `json:"name"`             // 凭证名称
    Type            string          `json:"type"`             // docker_registry/git_ssh/git_token/username_password
    Description     string          `json:"description"`

    // 加密数据 (JSON 加密后存储)
    EncryptedData   string          `json:"-" gorm:"type:text"`

    // 范围
    Scope           string          `json:"scope"`            // global/project
    ProjectID       *string         `json:"project_id,omitempty" gorm:"index"`

    // 元数据
    CreatedBy       string          `json:"created_by"`
    CreatedAt       time.Time       `json:"created_at"`
    LastUsedAt      *time.Time      `json:"last_used_at"`
}

// 解密后的凭证数据
type CredentialData struct {
    Username        string          `json:"username,omitempty"`
    Password        string          `json:"password,omitempty"`     // 或 Token
    SSHKey          string          `json:"ssh_key,omitempty"`      // PEM 格式
    SSHPassphrase   string          `json:"ssh_passphrase,omitempty"`
    ServerURL       string          `json:"server_url,omitempty"`   // docker.io, github.com
}

// 项目关联凭证
type ProjectCredential struct {
    ProjectID       string          `gorm:"primaryKey" json:"project_id"`
    CredentialID    string          `gorm:"primaryKey" json:"credential_id"`
    Alias           string          `json:"alias"`            // 在项目中使用的别名
}
```

#### 1.4 加密方案

```go
// AES-256-GCM 加密
type Cipher struct {
    key            []byte          // 从环境变量 QUIC_FLOW_SECRET_KEY 派生
}

func (c *Cipher) Encrypt(plaintext string) (string, error)
func (c *Cipher) Decrypt(ciphertext string) (string, error)
```

**密钥管理**:
- 密钥来源: 环境变量 `QUIC_FLOW_SECRET_KEY` (32 字节 hex)
- 启动时验证: 密钥无效则拒绝启动
- 密钥轮换: 导出 → 用新密钥加密 → 重新导入 (手动操作)

#### 1.5 API 设计

```
POST   /api/release/credentials                    # 创建凭证
GET    /api/release/credentials                    # 列出凭证
GET    /api/release/credentials/:id                # 获取凭证详情 (不返回敏感数据)
PUT    /api/release/credentials/:id                # 更新凭证
DELETE /api/release/credentials/:id                # 删除凭证

# 项目关联
GET    /api/release/projects/:id/credentials       # 列出项目可用凭证
POST   /api/release/projects/:id/credentials       # 关联凭证到项目
DELETE /api/release/projects/:id/credentials/:cid  # 取消关联
```

#### 1.6 使用场景

```yaml
# 部署配置中使用凭证
deploy:
  type: kubernetes
  image: my-registry.com/app:${VERSION}
  image_pull_secret: "${credential.harbor}"  # 引用项目凭证
```

---

### 模块二: Webhook 触发

#### 2.1 设计原则

> **最小实现**: 只做 GitHub/GitLab Push 触发

#### 2.2 数据模型

```go
// Webhook 配置
type WebhookConfig struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    ProjectID       string          `json:"project_id" gorm:"index"`
    Name            string          `json:"name"`
    Enabled         bool            `json:"enabled"`

    // 触发条件
    Source          string          `json:"source"`           // github/gitlab
    BranchFilter    []string        `json:"branch_filter"`    // ["main", "release/*"]

    // 触发动作
    TargetEnv       string          `json:"target_env"`       // 触发后部署的环境
    AutoDeploy      bool            `json:"auto_deploy"`      // 是否自动部署

    // Webhook 验证
    Secret          string          `json:"-" gorm:"type:text"` // HMAC 密钥

    // 元数据
    URL             string          `json:"url"`              // 公开 URL，用于配置到 Git
    CreatedAt       time.Time       `json:"created_at"`
}

// 触发记录
type TriggerRecord struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    WebhookID       string          `json:"webhook_id" gorm:"index"`
    Source          string          `json:"source"`
    Branch          string          `json:"branch"`
    Commit          string          `json:"commit"`
    Committer       string          `json:"committer"`
    Message         string          `json:"message"`

    // 触发结果
    TaskID          *string         `json:"task_id,omitempty"`
    Status          string          `json:"status"`           // success/failed/skipped
    Error           string          `json:"error,omitempty"`

    TriggeredAt     time.Time       `json:"triggered_at"`
}
```

#### 2.3 Webhook 处理流程

```
┌─────────────┐     Push      ┌─────────────┐
│  GitHub /   │──────────────▶│ QUIC-Flow   │
│  GitLab     │   Webhook     │  Receiver   │
└─────────────┘               └─────────────┘
                                    │
                                    ▼
                             1. 验证签名 (HMAC)
                                    │
                                    ▼
                             2. 匹配 WebhookConfig
                                    │
                                    ▼
                             3. 检查分支过滤
                                    │
                                    ▼
                             4. 创建部署任务
                                    │
                                    ▼
                             5. 记录 TriggerRecord
```

#### 2.4 API 设计

```
# Webhook 接收
POST   /api/release/webhooks/:id/trigger        # 接收 Webhook (公开端点)
POST   /api/release/webhooks/:id/test           # 测试 Webhook

# Webhook 管理
GET    /api/release/projects/:id/webhooks       # 列出项目 Webhook
POST   /api/release/projects/:id/webhooks       # 创建 Webhook
PUT    /api/release/webhooks/:id                # 更新 Webhook
DELETE /api/release/webhooks/:id                # 删除 Webhook

# 触发历史
GET    /api/release/webhooks/:id/triggers       # 触发历史
GET    /api/release/triggers/:id                # 触发详情
```

#### 2.5 签名验证

```go
// GitHub
func VerifyGitHub(signature string, payload []byte, secret string) bool {
    expected := "sha256=" + hmacSHA256(payload, secret)
    return hmac.Equal(signature, expected)
}

// GitLab
func VerifyGitLab(token string, secret string) bool {
    return token == secret
}
```

#### 2.6 Webhook 配置示例

**GitHub**:
```
Settings → Webhooks → Add webhook
→ Payload URL: https://quic-flow.example.com/api/release/webhooks/{id}/trigger
→ Content type: application/json
→ Secret: {从界面获取}
→ Events: Push events
```

**GitLab**:
```
Settings → Webhooks
→ URL: https://quic-flow.example.com/api/release/webhooks/{id}/trigger
→ Secret token: {从界面获取}
→ Trigger: Push events
```

---

### 模块三: 简化权限

#### 3.1 设计原则

> **两档权限**: 可部署 / 只读，够用就好

#### 3.2 数据模型

```go
// 项目成员
type ProjectMember struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    ProjectID       string          `json:"project_id" gorm:"index"`
    UserID          string          `json:"user_id" gorm:"index"`
    Role            string          `json:"role"`            // maintainer/viewer

    AddedBy         string          `json:"added_by"`
    AddedAt         time.Time       `json:"added_at"`

    // 关联
    User            *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// 用户 (简化)
type User struct {
    ID              string          `gorm:"primaryKey" json:"id"`
    Username        string          `gorm:"uniqueIndex" json:"username"`
    DisplayName     string          `json:"display_name"`
    Email           string          `json:"email"`

    // 全局角色
    IsAdmin         bool            `json:"is_admin"`        // 超级管理员

    CreatedAt       time.Time       `json:"created_at"`
}
```

#### 3.3 权限定义

| 角色 | 名称 | 权限 |
|------|------|------|
| `admin` | 超级管理员 | 全部权限，跨项目 |
| `maintainer` | 项目维护者 | 修改配置、部署所有环境、管理成员 |
| `viewer` | 访客 | 只读查看 |

#### 3.4 权限矩阵

| 操作 | admin | maintainer | viewer |
|------|-------|------------|--------|
| 修改项目配置 | ✓ | ✓ | ✗ |
| 部署生产环境 | ✓ | ✓ | ✗ |
| 部署其他环境 | ✓ | ✓ | ✗ |
| 查看任务日志 | ✓ | ✓ | ✓ |
| 管理成员 | ✓ | ✓ | ✗ |
| 删除项目 | ✓ | ✗ | ✗ |

#### 3.5 API 设计

```
# 项目成员
GET    /api/release/projects/:id/members          # 列出成员
POST   /api/release/projects/:id/members          # 添加成员
PUT    /api/release/projects/:id/members/:uid     # 修改角色
DELETE /api/release/projects/:id/members/:uid     # 移除成员

# 用户 (如果已有用户系统可跳过)
GET    /api/release/users                         # 列出用户
POST   /api/release/users                         # 创建用户
```

#### 3.6 认证方式

**方案 A: Header 认证** (推荐用于内网)
```
X-User-ID: user123
```

**方案 B: API Key**
```
Authorization: Bearer {api_key}
```

**方案 C: OAuth/OIDC** (如需企业集成)

> 初始版本支持方案 A + B，C 可后续扩展

---

## 技术架构

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
│  Project    │  Deploy     │  Webhook    │   Credential      │
│  APIs       │  APIs       │  APIs       │   APIs            │
└─────────────┴─────────────┴─────────────┴───────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       Core Services                         │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│  Engine     │  CredStore  │  Webhook    │   Auth            │
│  (已有)     │  (新增)     │  (新增)     │   (新增)          │
└─────────────┴─────────────┴─────────────┴───────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure                           │
├─────────────┬─────────────┬─────────────┬───────────────────┤
│ PostgreSQL  │  Log Files  │  Callback   │                   │
│  (元数据)   │  (日志)     │  (通知)     │                   │
└─────────────┴─────────────┴─────────────┴───────────────────┘
```

### 目录结构

```
pkg/release/
├── api/
│   ├── handlers.go           # 主 API 处理器
│   ├── credential_api.go     # 凭证 API (新增)
│   ├── webhook_api.go        # Webhook API (新增)
│   └── auth_middleware.go    # 认证中间件 (新增)
├── engine/
│   └── engine.go             # 部署引擎 (已有)
├── credential/               # 凭证模块 (新增)
│   ├── cipher.go             # 加密/解密
│   ├── manager.go            # 凭证管理器
│   └── types.go              # 凭证类型
├── webhook/                  # Webhook 模块 (新增)
│   ├── receiver.go           # Webhook 接收器
│   ├── github.go             # GitHub 处理
│   ├── gitlab.go             # GitLab 处理
│   └── verifier.go           # 签名验证
├── auth/                     # 认证模块 (新增)
│   ├── middleware.go         # 认证中间件
│   ├── authorizer.go         # 权限检查
│   └── types.go              # 用户类型
├── models/
│   ├── models.go             # 现有模型
│   ├── credential.go         # 凭证模型 (新增)
│   ├── webhook.go            # Webhook 模型 (新增)
│   └── auth.go               # 认证模型 (新增)
└── log/
    └── local.go              # 本地日志 (已有)
```

### 核心接口

```go
// 凭证管理器
type CredentialManager interface {
    Create(cred *Credential) error
    Get(id string) (*Credential, error)
    List(projectID string) ([]*Credential, error)
    Decrypt(cred *Credential) (*CredentialData, error)
    Delete(id string) error
}

// Webhook 处理器
type WebhookHandler interface {
    HandleGitHub(ctx context.Context, payload []byte, signature string, webhookID string) error
    HandleGitLab(ctx context.Context, payload []byte, token string, webhookID string) error
}

// 权限检查器
type Authorizer interface {
    CanDeploy(userID, projectID, env string) bool
    CanConfigure(userID, projectID string) bool
    CanView(userID, projectID string) bool
}
```

### 配置

```go
type Config struct {
    // 数据库
    Database   DatabaseConfig   `json:"database"`

    // 服务器
    Server     ServerConfig     `json:"server"`

    // 凭证加密
    SecretKey  string           `json:"secret_key"`        // 环境变量

    // Webhook
    Webhook    WebhookConfig    `json:"webhook"`
}

type WebhookConfig struct {
    PublicURL  string           `json:"public_url"`        // https://quic-flow.example.com
}
```

---

## 实施计划

### 总体时间线: **3 周**

```
Week 1              Week 2              Week 3
┌────────┬────────┐  ┌────────┬────────┐  ┌────────┬────────┐
│ 凭证   │        │  │ Webhook│        │  │ 权限   │ 前端   │
│ 中心   │        │  │ 触发   │        │  │ 控制   │ 集成   │
└────────┴────────┘  └────────┴────────┘  └────────┴────────┘
```

### Week 1: 凭证中心

| 任务 | 工作量 |
|------|--------|
| 数据模型 + 迁移 | 0.5天 |
| 加密/解密实现 | 1天 |
| CredentialManager | 1天 |
| API 实现 | 1天 |
| 前端凭证管理页面 | 1.5天 |

### Week 2: Webhook 触发

| 任务 | 工作量 |
|------|--------|
| 数据模型 + 迁移 | 0.5天 |
| GitHub Webhook 处理 | 1天 |
| GitLab Webhook 处理 | 1天 |
| 签名验证 | 0.5天 |
| API 实现 | 1天 |
| 前端 Webhook 配置页面 | 1天 |
| 触发历史页面 | 0.5天 |

### Week 3: 权限 + 集成

| 任务 | 工作量 |
|------|--------|
| 用户模型 (如需要) | 0.5天 |
| 认证中间件 | 0.5天 |
| 权限检查 | 0.5天 |
| 成员管理 API | 0.5天 |
| 前端成员管理页面 | 1天 |
| 集成测试 | 1天 |
| Bug 修复 | 1天 |

---

## 附录

### A. 不做的功能

| 功能 | 理由 | 替代方案 |
|------|------|----------|
| Docker 构建 | CI 系统更专业 | GitLab CI / GitHub Actions |
| 日志搜索 | 过度设计 | grep / tail -f |
| 复杂告警规则 | 不必要 | 已有回调通知 |
| 定时触发 | 需求不明确 | 后续按需添加 |
| 完整 RBAC | 过度设计 | 两档权限够用 |
| 审计日志 | 过度设计 | 数据库时间戳 |

### B. 配置示例

#### Webhook 配置到 GitHub

```bash
# 1. 创建 Webhook
curl -X POST http://quic-flow/api/release/projects/123/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "main-push",
    "source": "github",
    "branch_filter": ["main", "release/*"],
    "target_env": "production",
    "auto_deploy": true
  }'

# 返回: { "id": "wh-xxx", "url": "https://quic-flow.com/api/release/webhooks/wh-xxx/trigger", "secret": "xxx" }

# 2. 配置到 GitHub
# Settings → Webhooks → Add webhook
# Payload URL: {url}
# Secret: {secret}
```

#### 凭证使用

```yaml
# 项目配置
project:
  name: my-app

environments:
  - name: production
    targets:
      - name: k8s-prod
        type: kubernetes
        image: harbor.company.com/my-app:${VERSION}
        image_pull_secret: "${credential.harbor-prod}"  # 引用凭证别名
```

### C. API 认证

```bash
# Header 认证
curl -X POST http://quic-flow/api/release/projects/123/deploy \
  -H "X-User-ID: admin" \
  -H "X-User-Role: admin"

# API Key 认证
curl -X POST http://quic-flow/api/release/projects/123/deploy \
  -H "Authorization: Bearer quic_xxx"
```

### D. 参考资源

- [GitHub Webhooks](https://docs.github.com/en/developers/webhooks-and-events)
- [GitLab Webhooks](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)
- [Docker Registry Authentication](https://docs.docker.com/registry/spec/auth/)
