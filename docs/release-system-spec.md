# 发布系统功能需求文档 v1.0

## 1. 概述

发布系统是一个支持多种部署方式的自动化发布平台，通过 QUIC 协议与远程客户端通信，实现应用的自动化部署、更新和回滚。

### 1.1 技术栈

| 组件 | 技术选型 |
|------|----------|
| 数据库 | PostgreSQL |
| ORM | GORM |
| 通信协议 | QUIC |
| 前端 | Vue 3 + Element Plus |

### 1.2 核心特性

- **多部署方式**：容器、K8s、脚本、Git 拉取
- **灰度发布**：支持按比例或指定设备灰度
- **多集群金丝雀**：按顺序逐集群部署
- **制品管理**：内置仓库 + 外部仓库对接
- **定时发布**：发布窗口限制和定时任务
- **服务依赖**：支持服务间依赖顺序
- **灵活回滚**：整体回滚或单目标回滚
- **状态上报**：Client 状态实时上报，Server 统一分析

## 2. 系统架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           发布系统 Server                                 │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   任务编排器   │  │   变量管理器   │  │   制品管理器   │  │   调度器     │ │
│  │  Orchestrator │  │ VarManager   │  │ ArtifactMgr  │  │  Scheduler   │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘ │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   发布引擎    │  │   审批管理    │  │   通知中心    │  │   状态分析器  │ │
│  │ ReleaseEngine│  │ ApprovalMgr  │  │ NotifyCenter │  │ StatusAnalyzer│ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────────────────┤
│                         QUIC 传输层 + 状态收集                            │
├─────────────────────────────────────────────────────────────────────────┤
│                         PostgreSQL (GORM)                                │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
         ┌──────────────────────────┼──────────────────────────┐
         ▼                          ▼                          ▼
  ┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
  │   Client A      │       │   Client B      │       │   Client C      │
  │   (K8s集群1)    │       │   (K8s集群2)    │       │   (Docker主机)   │
  │                 │       │                 │       │                 │
  │  ┌───────────┐  │       │  ┌───────────┐  │       │  ┌───────────┐  │
  │  │ 执行引擎   │  │       │  │ 执行引擎   │  │       │  │ 执行引擎   │  │
  │  │ 状态上报   │  │       │  │ 状态上报   │  │       │  │ 状态上报   │  │
  │  └───────────┘  │       │  └───────────┘  │       │  └───────────┘  │
  └─────────────────┘       └─────────────────┘       └─────────────────┘
```

## 3. 核心功能模块

### 3.1 部署目标类型

| 类型 | 说明 | 执行方式 | 回滚支持 |
|------|------|----------|----------|
| **Container** | Docker 容器部署 | docker pull/run/stop/rm | 镜像版本回滚 |
| **Kubernetes** | K8s 集群部署 | kubectl apply/rollout | rollout undo |
| **Script** | 脚本执行部署 | 自定义脚本 (安装/更新/回滚/卸载) | 回滚脚本 |
| **GitPull** | Git 仓库拉取部署 | git clone/pull + 构建脚本 | git checkout |

### 3.2 脚本部署操作类型

脚本部署支持四种操作类型，每种操作对应独立的脚本：

| 操作类型 | 说明 | 脚本字段 | 使用场景 |
|----------|------|----------|----------|
| **install** | 首次安装部署 | `install_script` | 新服务首次部署 |
| **update** | 更新已有服务 | `update_script` | 版本升级、配置更新 |
| **rollback** | 回滚到指定版本 | `rollback_script` | 发布失败、紧急回退 |
| **uninstall** | 完全卸载服务 | `uninstall_script` | 下线服务、清理环境 |

#### 3.2.1 脚本部署配置示例

```yaml
deploy:
  type: script
  config:
    # 工作目录
    work_dir: "/opt/apps/${APP_NAME}"

    # 解释器 (可选，默认 /bin/bash)
    interpreter: "/bin/bash"

    # 公共环境变量
    environment:
      APP_NAME: "${APP_NAME}"
      APP_VERSION: "${RELEASE_VERSION}"
      APP_ENV: "${RELEASE_ENV}"
      APP_PORT: "${APP_PORT}"
      BACKUP_DIR: "${BACKUP_DIR}"

    # 安装脚本 - 首次部署时执行
    install_script: |
      #!/bin/bash
      set -e

      echo "Installing ${APP_NAME} version ${APP_VERSION}..."

      # 创建目录结构
      mkdir -p ${WORK_DIR}/{bin,conf,logs,data}

      # 下载制品
      curl -fsSL "${ARTIFACT_URL}" -o /tmp/${APP_NAME}-${APP_VERSION}.tar.gz

      # 解压部署
      tar -xzf /tmp/${APP_NAME}-${APP_VERSION}.tar.gz -C ${WORK_DIR}

      # 配置服务
      cp ${WORK_DIR}/conf/${APP_ENV}.yaml ${WORK_DIR}/conf/app.yaml

      # 注册 systemd 服务
      cat > /etc/systemd/system/${APP_NAME}.service <<EOF
      [Unit]
      Description=${APP_NAME} Service
      After=network.target

      [Service]
      Type=simple
      User=${APP_USER}
      WorkingDirectory=${WORK_DIR}
      ExecStart=${WORK_DIR}/bin/${APP_NAME}
      Restart=always
      RestartSec=5

      [Install]
      WantedBy=multi-user.target
      EOF

      systemctl daemon-reload
      systemctl enable ${APP_NAME}
      systemctl start ${APP_NAME}

      echo "Install completed successfully"

    # 更新脚本 - 版本升级时执行
    update_script: |
      #!/bin/bash
      set -e

      echo "Updating ${APP_NAME} to version ${APP_VERSION}..."

      # 备份当前版本
      BACKUP_PATH="${BACKUP_DIR}/${APP_NAME}/$(date +%Y%m%d_%H%M%S)"
      mkdir -p ${BACKUP_PATH}
      cp -r ${WORK_DIR}/bin ${BACKUP_PATH}/
      cp -r ${WORK_DIR}/conf ${BACKUP_PATH}/
      echo "${CURRENT_VERSION}" > ${BACKUP_PATH}/version.txt

      # 停止服务
      systemctl stop ${APP_NAME} || true

      # 下载新版本
      curl -fsSL "${ARTIFACT_URL}" -o /tmp/${APP_NAME}-${APP_VERSION}.tar.gz

      # 备份旧二进制
      mv ${WORK_DIR}/bin/${APP_NAME} ${WORK_DIR}/bin/${APP_NAME}.bak

      # 部署新版本
      tar -xzf /tmp/${APP_NAME}-${APP_VERSION}.tar.gz -C ${WORK_DIR}

      # 更新配置 (保留自定义配置)
      if [ -f ${WORK_DIR}/conf/app.yaml.local ]; then
        cp ${WORK_DIR}/conf/app.yaml.local ${WORK_DIR}/conf/app.yaml.local.bak
      fi

      # 启动服务
      systemctl start ${APP_NAME}

      # 健康检查
      sleep 5
      if ! systemctl is-active --quiet ${APP_NAME}; then
        echo "Service failed to start, rolling back..."
        mv ${WORK_DIR}/bin/${APP_NAME}.bak ${WORK_DIR}/bin/${APP_NAME}
        systemctl start ${APP_NAME}
        exit 1
      fi

      # 清理
      rm -f ${WORK_DIR}/bin/${APP_NAME}.bak
      rm -f /tmp/${APP_NAME}-${APP_VERSION}.tar.gz

      echo "Update completed successfully"

    # 回滚脚本 - 回滚到指定版本
    rollback_script: |
      #!/bin/bash
      set -e

      echo "Rolling back ${APP_NAME} to version ${ROLLBACK_VERSION}..."

      # 查找备份
      if [ -n "${ROLLBACK_VERSION}" ]; then
        BACKUP_PATH=$(find ${BACKUP_DIR}/${APP_NAME} -name "version.txt" -exec grep -l "^${ROLLBACK_VERSION}$" {} \; | head -1 | xargs dirname)
      else
        # 使用最近的备份
        BACKUP_PATH=$(ls -td ${BACKUP_DIR}/${APP_NAME}/*/ | head -1)
      fi

      if [ -z "${BACKUP_PATH}" ] || [ ! -d "${BACKUP_PATH}" ]; then
        echo "ERROR: Backup not found for version ${ROLLBACK_VERSION}"
        exit 1
      fi

      echo "Using backup from: ${BACKUP_PATH}"

      # 停止服务
      systemctl stop ${APP_NAME} || true

      # 恢复备份
      cp -r ${BACKUP_PATH}/bin/* ${WORK_DIR}/bin/
      cp -r ${BACKUP_PATH}/conf/* ${WORK_DIR}/conf/

      # 启动服务
      systemctl start ${APP_NAME}

      # 验证
      sleep 5
      if systemctl is-active --quiet ${APP_NAME}; then
        echo "Rollback completed successfully"
      else
        echo "ERROR: Service failed to start after rollback"
        exit 1
      fi

    # 卸载脚本 - 完全移除服务
    uninstall_script: |
      #!/bin/bash
      set -e

      echo "Uninstalling ${APP_NAME}..."

      # 停止并禁用服务
      systemctl stop ${APP_NAME} || true
      systemctl disable ${APP_NAME} || true

      # 删除 systemd 服务文件
      rm -f /etc/systemd/system/${APP_NAME}.service
      systemctl daemon-reload

      # 可选：备份数据
      if [ "${KEEP_DATA}" = "true" ]; then
        ARCHIVE_PATH="${BACKUP_DIR}/${APP_NAME}/uninstall_$(date +%Y%m%d_%H%M%S)"
        mkdir -p ${ARCHIVE_PATH}
        cp -r ${WORK_DIR}/data ${ARCHIVE_PATH}/ || true
        cp -r ${WORK_DIR}/conf ${ARCHIVE_PATH}/ || true
        echo "Data archived to: ${ARCHIVE_PATH}"
      fi

      # 删除应用目录
      rm -rf ${WORK_DIR}

      # 清理日志
      rm -rf /var/log/${APP_NAME}

      echo "Uninstall completed successfully"

    # 脚本超时配置
    timeouts:
      install: 600    # 安装超时 10 分钟
      update: 300     # 更新超时 5 分钟
      rollback: 180   # 回滚超时 3 分钟
      uninstall: 120  # 卸载超时 2 分钟
```

#### 3.2.2 脚本操作变量

脚本执行时可使用以下内置变量：

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `${WORK_DIR}` | 工作目录 | /opt/apps/myapp |
| `${APP_NAME}` | 应用名称 | myapp |
| `${APP_VERSION}` | 目标版本 | 1.2.3 |
| `${CURRENT_VERSION}` | 当前版本 | 1.2.2 |
| `${ROLLBACK_VERSION}` | 回滚目标版本 | 1.2.1 |
| `${ARTIFACT_URL}` | 制品下载地址 | https://... |
| `${BACKUP_DIR}` | 备份目录 | /data/backup |
| `${APP_ENV}` | 运行环境 | prod |
| `${KEEP_DATA}` | 卸载时保留数据 | true/false |

#### 3.2.3 脚本部署操作流程

```
┌─────────────────────────────────────────────────────────────────┐
│                        脚本部署操作流程                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────┐    检测目标状态     ┌──────────────────────┐       │
│  │ 发起部署 │ ──────────────────▶ │ 目标是否已安装该服务？ │       │
│  └─────────┘                     └──────────────────────┘       │
│                                          │                       │
│                          ┌───────────────┴───────────────┐       │
│                          ▼                               ▼       │
│                    ┌──────────┐                   ┌──────────┐   │
│                    │ 未安装    │                   │ 已安装    │   │
│                    └──────────┘                   └──────────┘   │
│                          │                               │       │
│                          ▼                               ▼       │
│                  ┌──────────────┐              ┌──────────────┐  │
│                  │ install_script│              │ update_script │  │
│                  │   (首次安装)   │              │   (版本更新)   │  │
│                  └──────────────┘              └──────────────┘  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────┐                              ┌──────────────┐      │
│  │ 发起回滚 │ ────────────────────────────▶ │ rollback_script│      │
│  └─────────┘                              │  (版本回滚)    │      │
│                                           └──────────────┘      │
│                                                                 │
│  ┌─────────┐                              ┌──────────────┐      │
│  │ 发起卸载 │ ────────────────────────────▶ │uninstall_script│     │
│  └─────────┘                              │  (完全移除)    │      │
│                                           └──────────────┘      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.3 发布流程 (Pipeline)

```yaml
pipeline:
  name: "标准发布流程"

  # 阶段1: 发布前
  pre_release:
    - name: "健康检查"
      type: health_check
      config:
        type: http
        url: "http://localhost:${APP_PORT}/health"
        timeout: 10

    - name: "备份当前版本"
      type: backup
      config:
        backup_type: full  # full/incremental/config_only
        retention: 5

    - name: "运行测试"
      type: script
      config:
        script: "./run_tests.sh"
        timeout: 300

  # 阶段2: 发布
  release:
    - name: "停止旧服务"
      type: stop_service
      config:
        graceful_timeout: 30

    - name: "部署新版本"
      type: deploy

    - name: "启动服务"
      type: start_service
      config:
        wait_ready: true
        ready_timeout: 60

  # 阶段3: 发布后
  post_release:
    - name: "健康验证"
      type: health_check
      config:
        type: http
        url: "http://localhost:${APP_PORT}/health"
        retry: 3
        interval: 10

    - name: "清理旧版本"
      type: cleanup
      config:
        keep_versions: 3

    - name: "发送通知"
      type: notify
      config:
        channels: ["dingtalk", "email"]
```

### 3.4 任务类型详解

| 任务类型 | 说明 | 配置项 |
|----------|------|--------|
| `health_check` | 健康检查 | type(http/tcp/script), url, timeout, retry |
| `backup` | 备份当前版本 | backup_type, path, retention |
| `script` | 自定义脚本 | script, interpreter, timeout, work_dir |
| `stop_service` | 停止服务 | graceful_timeout, force_kill |
| `start_service` | 启动服务 | wait_ready, ready_timeout |
| `deploy` | 执行部署 | (根据部署类型自动配置) |
| `rollback` | 回滚操作 | target_version, skip_checks |
| `cleanup` | 清理旧版本 | keep_versions, clean_backups |
| `notify` | 发送通知 | channels, template |
| `wait` | 等待 | duration / wait_for_approval |
| `approval` | 人工审批 | approvers, timeout |
| `condition` | 条件判断 | expression, on_true, on_false |
| `db_migrate` | 数据库迁移 | migration_path, rollback_on_fail |

### 3.5 变量系统

#### 3.5.1 变量优先级

| 优先级 | 来源 | 说明 | 示例 |
|--------|------|------|------|
| 1 (最低) | 系统变量 | 系统内置 | ${RELEASE_TIME} |
| 2 | 项目变量 | 项目级别 | ${APP_NAME} |
| 3 | 环境变量 | 环境级别 | ${DB_HOST} |
| 4 | 发布变量 | 单次发布 | ${DEPLOY_TAG} |
| 5 (最高) | 运行时变量 | 动态生成 | ${BACKUP_PATH} |

#### 3.5.2 内置系统变量

```yaml
# 发布相关
${RELEASE_ID}           # 发布ID (UUID)
${RELEASE_VERSION}      # 发布版本号
${RELEASE_ENV}          # 发布环境 (dev/test/staging/prod)
${RELEASE_USER}         # 发布操作人
${RELEASE_TIME}         # 发布时间 (RFC3339)
${RELEASE_TIMESTAMP}    # 发布时间戳 (Unix)

# 目标相关
${TARGET_ID}            # 目标ID
${TARGET_NAME}          # 目标名称
${TARGET_HOST}          # 目标主机
${TARGET_IP}            # 目标IP
${TARGET_CLIENT_ID}     # QUIC客户端ID
${TARGET_LABELS}        # 目标标签 (JSON)

# Git相关
${GIT_REPO}             # Git仓库地址
${GIT_BRANCH}           # Git分支
${GIT_COMMIT}           # Git提交SHA (完整)
${GIT_COMMIT_SHORT}     # Git提交SHA (短)
${GIT_TAG}              # Git标签
${GIT_MESSAGE}          # 提交信息

# 容器相关
${IMAGE_REGISTRY}       # 镜像仓库地址
${IMAGE_NAME}           # 镜像名称
${IMAGE_TAG}            # 镜像标签
${IMAGE_FULL}           # 完整镜像地址
${CONTAINER_NAME}       # 容器名称

# K8s相关
${K8S_CLUSTER}          # 集群名称
${K8S_NAMESPACE}        # 命名空间
${K8S_DEPLOYMENT}       # Deployment名称
${K8S_REPLICAS}         # 副本数
${K8S_CONTEXT}          # kubectl context

# 路径相关
${APP_DIR}              # 应用目录
${BACKUP_DIR}           # 备份目录
${LOG_DIR}              # 日志目录
${TEMP_DIR}             # 临时目录
```

#### 3.5.3 自定义变量

```yaml
variables:
  # 普通变量
  APP_NAME: "my-service"
  APP_PORT: "8080"

  # 密钥变量（加密存储，日志脱敏）
  DB_PASSWORD:
    type: secret
    value: "encrypted:AES256:xxxx"

  # 环境差异化变量
  API_URL:
    type: env_specific
    values:
      dev: "http://dev-api.internal"
      test: "http://test-api.internal"
      staging: "http://staging-api.internal"
      prod: "https://api.example.com"

  # 模板变量（运行时计算）
  BACKUP_PATH:
    type: template
    value: "${BACKUP_DIR}/${APP_NAME}/${RELEASE_TIMESTAMP}"

  # 引用外部变量
  EXTERNAL_CONFIG:
    type: external
    source: "vault"  # vault/env/file
    key: "secret/myapp/config"
```

## 4. 数据模型 (GORM)

### 4.1 项目 (Project)

```go
type Project struct {
    ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    Name        string         `gorm:"uniqueIndex;size:100;not null"`
    Description string         `gorm:"size:500"`
    Type        DeployType     `gorm:"size:20;not null"` // container/kubernetes/script/gitpull
    RepoURL     string         `gorm:"size:500"`
    RepoType    string         `gorm:"size:20"` // git/svn

    // 关联
    Environments []Environment `gorm:"foreignKey:ProjectID"`
    Variables    []Variable    `gorm:"foreignKey:ProjectID"`
    Pipelines    []Pipeline    `gorm:"foreignKey:ProjectID"`

    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type DeployType string

const (
    DeployTypeContainer  DeployType = "container"
    DeployTypeKubernetes DeployType = "kubernetes"
    DeployTypeScript     DeployType = "script"
    DeployTypeGitPull    DeployType = "gitpull"
)
```

### 4.2 环境 (Environment)

```go
type Environment struct {
    ID             string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ProjectID      string    `gorm:"type:uuid;index;not null"`
    Name           string    `gorm:"size:50;not null"` // dev/test/staging/prod
    Description    string    `gorm:"size:200"`

    // 发布窗口
    ReleaseWindow  *ReleaseWindow `gorm:"type:jsonb"`

    // 审批配置
    RequireApproval bool   `gorm:"default:false"`
    Approvers       []string `gorm:"type:jsonb"`

    // 关联
    Project    Project   `gorm:"foreignKey:ProjectID"`
    Targets    []Target  `gorm:"foreignKey:EnvironmentID"`
    Variables  []Variable `gorm:"foreignKey:EnvironmentID"`

    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  gorm.DeletedAt `gorm:"index"`
}

type ReleaseWindow struct {
    Enabled     bool     `json:"enabled"`
    Timezone    string   `json:"timezone"`     // Asia/Shanghai
    AllowedDays []int    `json:"allowed_days"` // 1-7 (Monday-Sunday)
    StartTime   string   `json:"start_time"`   // HH:MM
    EndTime     string   `json:"end_time"`     // HH:MM
}
```

### 4.3 部署目标 (Target)

```go
type Target struct {
    ID            string            `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    EnvironmentID string            `gorm:"type:uuid;index;not null"`
    ClientID      string            `gorm:"size:100;index;not null"` // QUIC客户端ID
    Name          string            `gorm:"size:100;not null"`
    Type          TargetType        `gorm:"size:20;not null"` // host/k8s-cluster
    Status        TargetStatus      `gorm:"size:20;default:'unknown'"`
    Labels        map[string]string `gorm:"type:jsonb"`
    Config        TargetConfig      `gorm:"type:jsonb"`
    Priority      int               `gorm:"default:0"` // 部署优先级，用于金丝雀

    // 关联
    Environment Environment `gorm:"foreignKey:EnvironmentID"`

    LastSeenAt *time.Time
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  gorm.DeletedAt `gorm:"index"`
}

type TargetType string

const (
    TargetTypeHost       TargetType = "host"
    TargetTypeK8sCluster TargetType = "k8s-cluster"
)

type TargetStatus string

const (
    TargetStatusOnline  TargetStatus = "online"
    TargetStatusOffline TargetStatus = "offline"
    TargetStatusUnknown TargetStatus = "unknown"
)

type TargetConfig struct {
    // Docker配置
    DockerHost    string `json:"docker_host,omitempty"`
    DockerTLSPath string `json:"docker_tls_path,omitempty"`

    // K8s配置
    KubeConfig  string `json:"kubeconfig,omitempty"`
    KubeContext string `json:"kube_context,omitempty"`
    Namespace   string `json:"namespace,omitempty"`

    // 通用配置
    WorkDir     string `json:"work_dir,omitempty"`
    User        string `json:"user,omitempty"`
    SSHKey      string `json:"ssh_key,omitempty"`
}
```

### 4.4 变量 (Variable)

```go
type Variable struct {
    ID            string       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ProjectID     *string      `gorm:"type:uuid;index"`
    EnvironmentID *string      `gorm:"type:uuid;index"`

    Name          string       `gorm:"size:100;not null"`
    Value         string       `gorm:"type:text"`
    Type          VariableType `gorm:"size:20;default:'plain'"`
    Description   string       `gorm:"size:200"`

    // 环境差异化值
    EnvValues     map[string]string `gorm:"type:jsonb"`

    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type VariableType string

const (
    VariableTypePlain       VariableType = "plain"
    VariableTypeSecret      VariableType = "secret"
    VariableTypeEnvSpecific VariableType = "env_specific"
    VariableTypeTemplate    VariableType = "template"
)
```

### 4.5 流水线 (Pipeline)

```go
type Pipeline struct {
    ID          string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ProjectID   string  `gorm:"type:uuid;index;not null"`
    Name        string  `gorm:"size:100;not null"`
    Description string  `gorm:"size:500"`
    IsDefault   bool    `gorm:"default:false"`

    // 阶段配置
    Stages      []Stage `gorm:"type:jsonb"`

    // 关联
    Project     Project `gorm:"foreignKey:ProjectID"`

    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type Stage struct {
    Name     string      `json:"name"`
    Phase    StagePhase  `json:"phase"` // pre_release/release/post_release
    Tasks    []Task      `json:"tasks"`
    OnError  ErrorAction `json:"on_error"` // continue/stop/rollback
    Parallel bool        `json:"parallel"` // 任务是否并行执行
}

type StagePhase string

const (
    StagePhasePreRelease  StagePhase = "pre_release"
    StagePhaseRelease     StagePhase = "release"
    StagePhasePostRelease StagePhase = "post_release"
)

type ErrorAction string

const (
    ErrorActionContinue ErrorAction = "continue"
    ErrorActionStop     ErrorAction = "stop"
    ErrorActionRollback ErrorAction = "rollback"
)

type Task struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Type        TaskType       `json:"type"`
    Config      map[string]any `json:"config"`
    Timeout     int            `json:"timeout"`     // 超时时间(秒)
    Retry       int            `json:"retry"`       // 重试次数
    RetryDelay  int            `json:"retry_delay"` // 重试间隔(秒)
    Condition   string         `json:"condition"`   // 执行条件表达式
    DependsOn   []string       `json:"depends_on"`  // 依赖的任务ID
}

type TaskType string

const (
    TaskTypeHealthCheck TaskType = "health_check"
    TaskTypeBackup      TaskType = "backup"
    TaskTypeScript      TaskType = "script"
    TaskTypeStopService TaskType = "stop_service"
    TaskTypeStartService TaskType = "start_service"
    TaskTypeDeploy      TaskType = "deploy"
    TaskTypeRollback    TaskType = "rollback"
    TaskTypeCleanup     TaskType = "cleanup"
    TaskTypeNotify      TaskType = "notify"
    TaskTypeWait        TaskType = "wait"
    TaskTypeApproval    TaskType = "approval"
    TaskTypeCondition   TaskType = "condition"
    TaskTypeDBMigrate   TaskType = "db_migrate"
)
```

### 4.6 发布记录 (Release)

```go
type Release struct {
    ID             string              `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ProjectID      string              `gorm:"type:uuid;index;not null"`
    EnvironmentID  string              `gorm:"type:uuid;index;not null"`
    PipelineID     string              `gorm:"type:uuid;index;not null"`
    Version        string              `gorm:"size:50;not null"`
    Operation      OperationType       `gorm:"size:20;not null;default:'deploy'"` // 操作类型
    Status         ReleaseStatus       `gorm:"size:20;not null;index"`

    // 发布配置
    Strategy       ReleaseStrategy     `gorm:"type:jsonb"`
    Variables      map[string]string   `gorm:"type:jsonb"`
    TargetIDs      []string            `gorm:"type:jsonb"` // 指定目标
    RollbackConfig *RollbackConfig     `gorm:"type:jsonb"`

    // 定时发布
    ScheduledAt    *time.Time          `gorm:"index"`

    // 执行结果
    Results        []TargetResult      `gorm:"type:jsonb"`

    // 元信息
    CreatedBy      string              `gorm:"size:100;not null"`
    ApprovedBy     *string             `gorm:"size:100"`

    CreatedAt      time.Time
    StartedAt      *time.Time
    FinishedAt     *time.Time
    UpdatedAt      time.Time
}

// 操作类型
type OperationType string

const (
    OperationTypeDeploy    OperationType = "deploy"    // 部署 (自动判断 install/update)
    OperationTypeInstall   OperationType = "install"   // 强制安装
    OperationTypeUpdate    OperationType = "update"    // 强制更新
    OperationTypeRollback  OperationType = "rollback"  // 回滚
    OperationTypeUninstall OperationType = "uninstall" // 卸载
)

type ReleaseStatus string

const (
    ReleaseStatusPending    ReleaseStatus = "pending"
    ReleaseStatusScheduled  ReleaseStatus = "scheduled"
    ReleaseStatusApproving  ReleaseStatus = "approving"
    ReleaseStatusRunning    ReleaseStatus = "running"
    ReleaseStatusPaused     ReleaseStatus = "paused"
    ReleaseStatusSuccess    ReleaseStatus = "success"
    ReleaseStatusFailed     ReleaseStatus = "failed"
    ReleaseStatusCancelled  ReleaseStatus = "cancelled"
    ReleaseStatusRollback   ReleaseStatus = "rollback"
)

type ReleaseStrategy struct {
    Type            StrategyType `json:"type"` // rolling/blue_green/canary

    // 滚动更新配置
    BatchSize       int          `json:"batch_size"`
    BatchInterval   int          `json:"batch_interval"` // 秒

    // 金丝雀配置
    CanaryPercent   int          `json:"canary_percent"`   // 灰度比例 (1-100)
    CanaryTargets   []string     `json:"canary_targets"`   // 指定灰度目标ID
    VerifyDuration  int          `json:"verify_duration"`  // 验证时长(秒)
    AutoPromote     bool         `json:"auto_promote"`     // 自动全量

    // 蓝绿配置
    SwitchTimeout   int          `json:"switch_timeout"`
    KeepOldVersion  bool         `json:"keep_old_version"`
}

type StrategyType string

const (
    StrategyTypeRolling   StrategyType = "rolling"
    StrategyTypeBlueGreen StrategyType = "blue_green"
    StrategyTypeCanary    StrategyType = "canary"
)

type RollbackConfig struct {
    Granularity     RollbackGranularity `json:"granularity"` // all/single
    AutoRollback    bool                `json:"auto_rollback"`
    Conditions      []RollbackCondition `json:"conditions"`
    TargetVersion   string              `json:"target_version,omitempty"`
}

type RollbackGranularity string

const (
    RollbackGranularityAll    RollbackGranularity = "all"
    RollbackGranularitySingle RollbackGranularity = "single"
)

type RollbackCondition struct {
    Type      string `json:"type"`      // health_check_failed/error_rate/response_time
    Threshold any    `json:"threshold"` // 阈值
}

type TargetResult struct {
    TargetID   string              `json:"target_id"`
    TargetName string              `json:"target_name"`
    Status     TargetReleaseStatus `json:"status"`
    StartedAt  *time.Time          `json:"started_at"`
    FinishedAt *time.Time          `json:"finished_at"`
    Stages     []StageResult       `json:"stages"`
    Error      string              `json:"error,omitempty"`
}

type TargetReleaseStatus string

const (
    TargetReleaseStatusPending   TargetReleaseStatus = "pending"
    TargetReleaseStatusRunning   TargetReleaseStatus = "running"
    TargetReleaseStatusSuccess   TargetReleaseStatus = "success"
    TargetReleaseStatusFailed    TargetReleaseStatus = "failed"
    TargetReleaseStatusSkipped   TargetReleaseStatus = "skipped"
    TargetReleaseStatusRollback  TargetReleaseStatus = "rollback"
)

type StageResult struct {
    Name      string       `json:"name"`
    Phase     StagePhase   `json:"phase"`
    Status    string       `json:"status"`
    Tasks     []TaskResult `json:"tasks"`
    StartedAt *time.Time   `json:"started_at"`
    FinishedAt *time.Time  `json:"finished_at"`
}

type TaskResult struct {
    ID         string     `json:"id"`
    Name       string     `json:"name"`
    Type       TaskType   `json:"type"`
    Status     string     `json:"status"`
    Output     string     `json:"output,omitempty"`
    Error      string     `json:"error,omitempty"`
    StartedAt  *time.Time `json:"started_at"`
    FinishedAt *time.Time `json:"finished_at"`
    RetryCount int        `json:"retry_count"`
}
```

### 4.7 脚本部署配置 (ScriptDeployConfig)

```go
// 脚本部署配置 - 存储在 Project 或 Pipeline 中
type ScriptDeployConfig struct {
    // 工作目录
    WorkDir     string `json:"work_dir"`

    // 解释器
    Interpreter string `json:"interpreter"` // 默认 /bin/bash

    // 环境变量
    Environment map[string]string `json:"environment"`

    // 四种操作脚本
    InstallScript   string `json:"install_script"`   // 安装脚本
    UpdateScript    string `json:"update_script"`    // 更新脚本
    RollbackScript  string `json:"rollback_script"`  // 回滚脚本
    UninstallScript string `json:"uninstall_script"` // 卸载脚本

    // 超时配置
    Timeouts ScriptTimeouts `json:"timeouts"`
}

type ScriptTimeouts struct {
    Install   int `json:"install"`   // 安装超时(秒)，默认 600
    Update    int `json:"update"`    // 更新超时(秒)，默认 300
    Rollback  int `json:"rollback"`  // 回滚超时(秒)，默认 180
    Uninstall int `json:"uninstall"` // 卸载超时(秒)，默认 120
}

// 目标安装状态 - 记录每个目标的安装信息
type TargetInstallation struct {
    ID            string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    TargetID      string    `gorm:"type:uuid;index;not null"`
    ProjectID     string    `gorm:"type:uuid;index;not null"`
    Version       string    `gorm:"size:50;not null"` // 当前安装版本
    Status        string    `gorm:"size:20;not null"` // installed/uninstalled/failed
    InstalledAt   time.Time `gorm:"not null"`
    LastUpdatedAt *time.Time

    // 备份信息
    BackupPath    string `gorm:"size:500"` // 最近备份路径
    BackupCount   int    `gorm:"default:0"` // 备份数量

    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// 安装状态常量
const (
    InstallStatusInstalled   = "installed"
    InstallStatusUninstalled = "uninstalled"
    InstallStatusFailed      = "failed"
    InstallStatusUnknown     = "unknown"
)
```

### 4.8 制品 (Artifact)

```go
type Artifact struct {
    ID          string       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ProjectID   string       `gorm:"type:uuid;index;not null"`
    Name        string       `gorm:"size:200;not null"`
    Version     string       `gorm:"size:50;not null"`
    Type        ArtifactType `gorm:"size:20;not null"`

    // 存储信息
    StorageType StorageType  `gorm:"size:20;not null"` // local/s3/harbor/nexus
    StoragePath string       `gorm:"size:500;not null"` // 存储路径或URL
    Size        int64        `gorm:"not null"`
    Checksum    string       `gorm:"size:64"` // SHA256

    // 元数据
    Metadata    map[string]string `gorm:"type:jsonb"`

    // 关联
    Project     Project `gorm:"foreignKey:ProjectID"`

    CreatedBy   string    `gorm:"size:100"`
    CreatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type ArtifactType string

const (
    ArtifactTypeImage   ArtifactType = "image"   // Docker镜像
    ArtifactTypeBinary  ArtifactType = "binary"  // 二进制文件
    ArtifactTypeArchive ArtifactType = "archive" // 压缩包
    ArtifactTypeConfig  ArtifactType = "config"  // 配置文件
)

type StorageType string

const (
    StorageTypeLocal  StorageType = "local"
    StorageTypeS3     StorageType = "s3"
    StorageTypeHarbor StorageType = "harbor"
    StorageTypeNexus  StorageType = "nexus"
)
```

### 4.9 服务依赖 (ServiceDependency)

```go
type ServiceDependency struct {
    ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ProjectID   string `gorm:"type:uuid;index;not null"`

    // 依赖关系: ServiceID 依赖于 DependsOnID
    ServiceID   string `gorm:"type:uuid;index;not null"` // 当前服务(项目)
    DependsOnID string `gorm:"type:uuid;index;not null"` // 依赖的服务(项目)

    // 依赖配置
    Required    bool   `gorm:"default:true"`  // 是否必须
    WaitReady   bool   `gorm:"default:true"`  // 是否等待依赖就绪
    Timeout     int    `gorm:"default:300"`   // 等待超时(秒)

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 4.10 状态上报记录 (StatusReport)

```go
type StatusReport struct {
    ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ReleaseID    string    `gorm:"type:uuid;index;not null"`
    TargetID     string    `gorm:"type:uuid;index;not null"`
    ClientID     string    `gorm:"size:100;index;not null"`

    // 状态信息
    Phase        StagePhase `gorm:"size:20;not null"`
    TaskID       string     `gorm:"size:50"`
    TaskName     string     `gorm:"size:100"`
    Status       string     `gorm:"size:20;not null"`
    Progress     int        `gorm:"default:0"` // 0-100
    Message      string     `gorm:"type:text"`

    // 健康指标
    Metrics      map[string]any `gorm:"type:jsonb"`

    ReportedAt   time.Time `gorm:"index;not null"`
    CreatedAt    time.Time
}

// 索引优化
func (StatusReport) TableName() string {
    return "status_reports"
}

// 创建复合索引
// CREATE INDEX idx_status_reports_release_target ON status_reports(release_id, target_id, reported_at DESC);
```

### 4.11 审批记录 (Approval)

```go
type Approval struct {
    ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
    ReleaseID  string         `gorm:"type:uuid;index;not null"`
    Status     ApprovalStatus `gorm:"size:20;not null;index"`

    Approvers  []string       `gorm:"type:jsonb"` // 需要审批的人员
    ApprovedBy *string        `gorm:"size:100"`   // 实际审批人
    Comment    string         `gorm:"type:text"`

    ExpireAt   time.Time      `gorm:"index"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type ApprovalStatus string

const (
    ApprovalStatusPending  ApprovalStatus = "pending"
    ApprovalStatusApproved ApprovalStatus = "approved"
    ApprovalStatusRejected ApprovalStatus = "rejected"
    ApprovalStatusExpired  ApprovalStatus = "expired"
)
```

## 5. 制品管理

### 5.1 内置制品仓库

```yaml
artifact_storage:
  type: local
  config:
    base_path: "/data/artifacts"
    max_size: "100GB"
    cleanup_policy:
      max_age_days: 90
      keep_latest: 10
```

### 5.2 外部仓库对接

```yaml
# Docker Harbor
artifact_registry:
  type: harbor
  config:
    url: "https://harbor.example.com"
    project: "myproject"
    username: "${HARBOR_USER}"
    password: "${HARBOR_PASSWORD}"

# Nexus
artifact_registry:
  type: nexus
  config:
    url: "https://nexus.example.com"
    repository: "releases"
    username: "${NEXUS_USER}"
    password: "${NEXUS_PASSWORD}"

# S3/MinIO
artifact_storage:
  type: s3
  config:
    endpoint: "https://s3.example.com"
    bucket: "artifacts"
    access_key: "${S3_ACCESS_KEY}"
    secret_key: "${S3_SECRET_KEY}"
    region: "us-east-1"
```

## 6. 灰度发布策略

### 6.1 按比例灰度

```yaml
strategy:
  type: canary
  config:
    # 灰度比例 (10% 的目标先更新)
    canary_percent: 10

    # 验证配置
    verify_duration: 300  # 观察5分钟
    health_check:
      type: http
      url: "http://localhost:8080/health"
      success_threshold: 3

    # 自动/手动全量
    auto_promote: false

    # 失败自动回滚
    auto_rollback: true
```

### 6.2 指定设备灰度

```yaml
strategy:
  type: canary
  config:
    # 指定特定目标进行灰度
    canary_targets:
      - "target-uuid-1"  # 指定的目标ID
      - "target-uuid-2"

    # 或按标签选择
    canary_selector:
      labels:
        canary: "true"
        region: "cn-north"

    verify_duration: 600
    auto_promote: false
```

### 6.3 多集群金丝雀

```yaml
strategy:
  type: canary
  config:
    # 按集群优先级顺序部署
    cluster_order:
      - priority: 1
        targets: ["cluster-dev"]
        verify_duration: 60

      - priority: 2
        targets: ["cluster-staging"]
        verify_duration: 300
        approval_required: true

      - priority: 3
        targets: ["cluster-prod-1", "cluster-prod-2"]
        canary_percent: 20
        verify_duration: 600

      - priority: 4
        targets: ["cluster-prod-1", "cluster-prod-2"]
        full_rollout: true
```

## 7. 定时发布与发布窗口

### 7.1 发布窗口配置

```yaml
release_window:
  enabled: true
  timezone: "Asia/Shanghai"

  # 允许发布的时间段
  allowed_windows:
    - days: [1, 2, 3, 4, 5]  # 周一到周五
      start_time: "10:00"
      end_time: "12:00"
    - days: [1, 2, 3, 4, 5]
      start_time: "14:00"
      end_time: "17:00"

  # 禁止发布的日期
  blackout_dates:
    - "2024-02-09"  # 除夕
    - "2024-02-10"  # 春节
    - "2024-10-01"  # 国庆

  # 紧急发布白名单用户
  emergency_users:
    - "admin"
    - "oncall"
```

### 7.2 定时发布

```yaml
scheduled_release:
  # 指定时间发布
  schedule_at: "2024-12-15T10:00:00+08:00"

  # 或 cron 表达式 (周期性)
  cron: "0 10 * * 1-5"  # 工作日上午10点

  # 发布前提醒
  reminder:
    enabled: true
    before_minutes: [60, 30, 10]
    channels: ["dingtalk", "email"]
```

## 8. 服务依赖管理

### 8.1 依赖定义

```yaml
dependencies:
  # 当前服务依赖的服务
  depends_on:
    - service: "database-migration"
      required: true
      wait_ready: true
      timeout: 300

    - service: "config-service"
      required: true
      wait_ready: true
      health_check:
        url: "http://config-service:8080/health"

    - service: "cache-service"
      required: false  # 可选依赖
      wait_ready: false
```

### 8.2 依赖检查流程

```
1. 解析依赖图，检测循环依赖
2. 拓扑排序确定部署顺序
3. 按顺序执行：
   a. 部署依赖服务
   b. 等待健康检查通过
   c. 部署当前服务
```

## 9. 状态上报与分析

### 9.1 Client 上报协议

```go
// Client -> Server 状态上报
type StatusReportRequest struct {
    ReleaseID  string         `json:"release_id"`
    TargetID   string         `json:"target_id"`
    ClientID   string         `json:"client_id"`
    Phase      string         `json:"phase"`
    TaskID     string         `json:"task_id"`
    TaskName   string         `json:"task_name"`
    Status     string         `json:"status"`  // running/success/failed
    Progress   int            `json:"progress"` // 0-100
    Message    string         `json:"message"`
    Metrics    map[string]any `json:"metrics"`  // CPU/Memory/网络等
    Timestamp  time.Time      `json:"timestamp"`
}

// Server -> Client 控制指令
type ControlCommand struct {
    Command    string         `json:"command"` // pause/resume/cancel/rollback
    ReleaseID  string         `json:"release_id"`
    Params     map[string]any `json:"params"`
}
```

### 9.2 Server 状态分析

```go
type ReleaseAnalytics struct {
    ReleaseID        string    `json:"release_id"`
    TotalTargets     int       `json:"total_targets"`
    SuccessTargets   int       `json:"success_targets"`
    FailedTargets    int       `json:"failed_targets"`
    PendingTargets   int       `json:"pending_targets"`

    // 时间统计
    TotalDuration    int64     `json:"total_duration_ms"`
    AvgTargetDuration int64    `json:"avg_target_duration_ms"`

    // 阶段耗时
    PhaseDurations   map[string]int64 `json:"phase_durations"`

    // 健康指标
    HealthScore      float64   `json:"health_score"` // 0-100
    Alerts           []Alert   `json:"alerts"`
}

type Alert struct {
    Level    string    `json:"level"` // info/warning/error
    TargetID string    `json:"target_id"`
    Message  string    `json:"message"`
    Time     time.Time `json:"time"`
}
```

## 10. API 接口

### 10.1 项目管理

```
POST   /api/release/projects              # 创建项目
GET    /api/release/projects              # 项目列表
GET    /api/release/projects/:id          # 项目详情
PUT    /api/release/projects/:id          # 更新项目
DELETE /api/release/projects/:id          # 删除项目
```

### 10.2 环境管理

```
POST   /api/release/projects/:id/environments     # 创建环境
GET    /api/release/projects/:id/environments     # 环境列表
GET    /api/release/environments/:id              # 环境详情
PUT    /api/release/environments/:id              # 更新环境
DELETE /api/release/environments/:id              # 删除环境
```

### 10.3 目标管理

```
POST   /api/release/environments/:id/targets      # 添加目标
GET    /api/release/environments/:id/targets      # 目标列表
PUT    /api/release/targets/:id                   # 更新目标
DELETE /api/release/targets/:id                   # 删除目标
GET    /api/release/targets/:id/status            # 目标状态
```

### 10.4 流水线管理

```
POST   /api/release/projects/:id/pipelines        # 创建流水线
GET    /api/release/projects/:id/pipelines        # 流水线列表
GET    /api/release/pipelines/:id                 # 流水线详情
PUT    /api/release/pipelines/:id                 # 更新流水线
DELETE /api/release/pipelines/:id                 # 删除流水线
POST   /api/release/pipelines/:id/validate        # 验证流水线
```

### 10.5 变量管理

```
POST   /api/release/variables                     # 创建变量
GET    /api/release/projects/:id/variables        # 项目变量列表
GET    /api/release/environments/:id/variables    # 环境变量列表
PUT    /api/release/variables/:id                 # 更新变量
DELETE /api/release/variables/:id                 # 删除变量
```

### 10.6 制品管理

```
POST   /api/release/artifacts/upload              # 上传制品
GET    /api/release/projects/:id/artifacts        # 制品列表
GET    /api/release/artifacts/:id                 # 制品详情
GET    /api/release/artifacts/:id/download        # 下载制品
DELETE /api/release/artifacts/:id                 # 删除制品
POST   /api/release/artifacts/sync                # 同步外部仓库
```

### 10.7 发布管理

```
POST   /api/release/deploys                       # 创建发布
GET    /api/release/deploys                       # 发布列表
GET    /api/release/deploys/:id                   # 发布详情
GET    /api/release/deploys/:id/logs              # 发布日志
GET    /api/release/deploys/:id/analytics         # 发布分析

# 发布控制
POST   /api/release/deploys/:id/start             # 开始发布
POST   /api/release/deploys/:id/pause             # 暂停发布
POST   /api/release/deploys/:id/resume            # 恢复发布
POST   /api/release/deploys/:id/cancel            # 取消发布
POST   /api/release/deploys/:id/rollback          # 回滚发布
POST   /api/release/deploys/:id/promote           # 金丝雀全量发布
POST   /api/release/deploys/:id/retry             # 重试失败任务

# 操作类型 (脚本部署)
POST   /api/release/install                       # 安装服务 (首次部署)
POST   /api/release/update                        # 更新服务
POST   /api/release/uninstall                     # 卸载服务

# 单目标操作
POST   /api/release/deploys/:id/targets/:tid/rollback  # 单目标回滚
POST   /api/release/deploys/:id/targets/:tid/retry     # 单目标重试
POST   /api/release/deploys/:id/targets/:tid/uninstall # 单目标卸载
```

### 10.8 审批管理

```
GET    /api/release/approvals                     # 待审批列表
GET    /api/release/approvals/:id                 # 审批详情
POST   /api/release/approvals/:id/approve         # 同意
POST   /api/release/approvals/:id/reject          # 拒绝
```

### 10.9 状态上报

```
POST   /api/release/status/report                 # Client 状态上报
GET    /api/release/deploys/:id/status            # 发布状态汇总
WS     /api/release/deploys/:id/watch             # WebSocket 实时状态
```

### 10.10 依赖管理

```
POST   /api/release/dependencies                  # 创建依赖
GET    /api/release/projects/:id/dependencies     # 依赖列表
DELETE /api/release/dependencies/:id              # 删除依赖
GET    /api/release/projects/:id/dependency-graph # 依赖图
POST   /api/release/dependencies/validate         # 验证依赖(检测循环)
```

## 11. 通知配置

```yaml
notifications:
  channels:
    dingtalk:
      webhook: "${DINGTALK_WEBHOOK}"
      secret: "${DINGTALK_SECRET}"

    wechat_work:
      webhook: "${WECHAT_WEBHOOK}"

    email:
      smtp_host: "smtp.example.com"
      smtp_port: 587
      username: "${SMTP_USER}"
      password: "${SMTP_PASSWORD}"
      from: "deploy@example.com"

    webhook:
      url: "https://example.com/webhook"
      headers:
        Authorization: "Bearer ${WEBHOOK_TOKEN}"

  templates:
    release_start: |
      🚀 发布开始
      项目: ${PROJECT_NAME}
      环境: ${RELEASE_ENV}
      版本: ${RELEASE_VERSION}
      操作人: ${RELEASE_USER}

    release_success: |
      ✅ 发布成功
      项目: ${PROJECT_NAME}
      版本: ${RELEASE_VERSION}
      耗时: ${DURATION}

    release_failed: |
      ❌ 发布失败
      项目: ${PROJECT_NAME}
      版本: ${RELEASE_VERSION}
      错误: ${ERROR_MESSAGE}

    approval_required: |
      ⏳ 需要审批
      项目: ${PROJECT_NAME}
      环境: ${RELEASE_ENV}
      申请人: ${RELEASE_USER}
      链接: ${APPROVAL_URL}

  rules:
    - events: [release_start, release_success]
      channels: [dingtalk]

    - events: [release_failed]
      channels: [dingtalk, email]

    - events: [approval_required]
      channels: [dingtalk, wechat_work]
      recipients: ["${APPROVERS}"]
```

## 12. 后续扩展 (TODO)

### 12.1 配置中心集成
- [ ] Nacos 集成
- [ ] Apollo 集成
- [ ] Consul 集成

### 12.2 更多部署目标
- [ ] Helm Chart 部署
- [ ] Terraform 部署
- [ ] Ansible Playbook

### 12.3 高级功能
- [ ] A/B 测试发布
- [ ] 流量镜像
- [ ] 自动扩缩容集成

---

*文档版本: v1.0*
*最后更新: 2026-01-01*
