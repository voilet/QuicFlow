# Docker/K8s 部署配置分层设计方案

## 1. 当前实现问题分析

### 1.1 现状

当前配置分布在三个层级：

| 层级 | 配置内容 | 问题 |
|------|----------|------|
| **项目级别** | 完整的 ContainerDeployConfig（镜像、端口、卷、网络、资源限制、健康检查等） | 配置过于全面，变更需要修改项目 |
| **版本级别** | 镜像地址、环境变量、副本数、K8s YAML | 覆盖能力不足，无法覆盖端口等 |
| **任务级别** | 无 | 缺少临时覆盖能力 |

### 1.2 存在的问题

1. **配置职责混乱**
   - 资源限制 (`memory_limit`, `cpu_limit`) 在项目级别，但不同版本可能需要不同配置
   - 环境变量在两个层级都有，合并逻辑不清晰

2. **灵活性不足**
   - 版本无法覆盖端口、卷挂载等配置
   - 任务执行时无法临时调整配置
   - 同一版本部署到不同环境时配置完全相同

3. **运维困难**
   - 紧急扩容需要修改版本或项目
   - A/B 测试无法在同一版本用不同配置
   - 金丝雀发布无法使用不同资源配置

4. **镜像版本管理混乱**
   - 项目配置了 `image`，版本也配置了 `container_image`
   - 优先级不明确，容易配置错误

---

## 2. 设计原则

### 2.1 分层原则

```
┌─────────────────────────────────────────────────────────────────┐
│                        任务级别 (Task)                           │
│    临时覆盖：环境变量追加、副本数、镜像tag、资源限制覆盖              │
└─────────────────────────────────────────────────────────────────┘
                              ▲ 覆盖
┌─────────────────────────────────────────────────────────────────┐
│                        版本级别 (Version)                        │
│    发布配置：镜像tag、环境变量、资源限制、启动命令、部署脚本          │
└─────────────────────────────────────────────────────────────────┘
                              ▲ 覆盖
┌─────────────────────────────────────────────────────────────────┐
│                        项目级别 (Project)                        │
│    基础设施：仓库地址、容器名、端口、卷、网络、安全、健康检查         │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 核心原则

| 原则 | 说明 |
|------|------|
| **基础设施稳定** | 端口、卷、网络等基础设施配置在项目级别，不随版本变化 |
| **版本是发布单元** | 版本包含完整的发布内容，可独立回滚 |
| **任务可临时调整** | 执行时可覆盖非基础设施配置 |
| **配置合并透明** | 合并规则明确，用户能预知最终配置 |

---

## 3. 详细设计方案

### 3.1 项目级别配置（基础设施层）

> **定位**：定义"在哪里运行"，通常由运维/架构师配置，变更频率低

#### Docker 容器项目配置

```go
type ProjectContainerConfig struct {
    // ========== 镜像仓库（固定） ==========
    Registry        string `json:"registry,omitempty"`         // 镜像仓库地址
    RegistryUser    string `json:"registry_user,omitempty"`    // 仓库认证
    RegistryPass    string `json:"registry_pass,omitempty"`
    ImagePullPolicy string `json:"image_pull_policy,omitempty"` // always/ifnotpresent/never

    // ========== 容器标识（固定） ==========
    ContainerName   string `json:"container_name"`              // 容器名称模板
    Hostname        string `json:"hostname,omitempty"`

    // ========== 网络配置（固定） ==========
    Ports           []PortMapping `json:"ports,omitempty"`      // 端口映射
    NetworkMode     string        `json:"network_mode,omitempty"`
    Networks        []string      `json:"networks,omitempty"`
    DNS             []string      `json:"dns,omitempty"`
    ExtraHosts      []string      `json:"extra_hosts,omitempty"`

    // ========== 存储配置（固定） ==========
    Volumes         []VolumeMount `json:"volumes,omitempty"`    // 卷挂载
    TmpfsMounts     []TmpfsMount  `json:"tmpfs_mounts,omitempty"`

    // ========== 安全配置（固定） ==========
    Privileged      bool     `json:"privileged,omitempty"`
    CapAdd          []string `json:"cap_add,omitempty"`
    CapDrop         []string `json:"cap_drop,omitempty"`
    SecurityOpt     []string `json:"security_opt,omitempty"`
    ReadOnlyRootfs  bool     `json:"read_only_rootfs,omitempty"`

    // ========== 设备配置（固定） ==========
    Devices         []DeviceMapping `json:"devices,omitempty"`
    GPUs            string          `json:"gpus,omitempty"`

    // ========== 运行时配置（固定） ==========
    Runtime         string `json:"runtime,omitempty"`
    Init            bool   `json:"init,omitempty"`
    PidMode         string `json:"pid_mode,omitempty"`

    // ========== 日志配置（固定） ==========
    LogDriver       string            `json:"log_driver,omitempty"`
    LogOpts         map[string]string `json:"log_opts,omitempty"`

    // ========== 健康检查模板（固定） ==========
    HealthCheck     *ContainerHealthCheck `json:"health_check,omitempty"`

    // ========== 重启策略（固定） ==========
    RestartPolicy   string `json:"restart_policy,omitempty"`

    // ========== 部署行为（固定） ==========
    StopTimeout     int  `json:"stop_timeout,omitempty"`
    RemoveOld       bool `json:"remove_old,omitempty"`
    PullBeforeStop  bool `json:"pull_before_stop,omitempty"`

    // ========== 默认资源限制（可被版本/任务覆盖） ==========
    DefaultResources *ResourceLimits `json:"default_resources,omitempty"`
}
```

#### K8s 项目配置

```go
type ProjectK8sConfig struct {
    // ========== 集群配置（固定） ==========
    KubeConfig      string `json:"kubeconfig,omitempty"`
    KubeContext     string `json:"kube_context,omitempty"`
    Namespace       string `json:"namespace,omitempty"`

    // ========== 资源配置（固定） ==========
    ResourceType    string `json:"resource_type,omitempty"`    // deployment/statefulset/daemonset
    ResourceName    string `json:"resource_name,omitempty"`
    ContainerName   string `json:"container_name,omitempty"`

    // ========== 镜像仓库（固定） ==========
    Registry        string `json:"registry,omitempty"`
    RegistryUser    string `json:"registry_user,omitempty"`
    RegistryPass    string `json:"registry_pass,omitempty"`
    ImagePullSecret string `json:"image_pull_secret,omitempty"`
    ImagePullPolicy string `json:"image_pull_policy,omitempty"`

    // ========== 服务暴露（固定） ==========
    ServiceType     string    `json:"service_type,omitempty"`
    ServicePorts    []K8sPort `json:"service_ports,omitempty"`

    // ========== 更新策略（固定） ==========
    UpdateStrategy  string `json:"update_strategy,omitempty"`
    MaxUnavailable  string `json:"max_unavailable,omitempty"`
    MaxSurge        string `json:"max_surge,omitempty"`

    // ========== 超时配置（固定） ==========
    RolloutTimeout  int `json:"rollout_timeout,omitempty"`

    // ========== 默认副本数和资源（可被版本/任务覆盖） ==========
    DefaultReplicas  int             `json:"default_replicas,omitempty"`
    DefaultResources *ResourceLimits `json:"default_resources,omitempty"`
}

// ResourceLimits 资源限制（可被版本/任务覆盖）
type ResourceLimits struct {
    CPURequest    string `json:"cpu_request,omitempty"`
    CPULimit      string `json:"cpu_limit,omitempty"`
    MemoryRequest string `json:"memory_request,omitempty"`
    MemoryLimit   string `json:"memory_limit,omitempty"`
}
```

---

### 3.2 版本级别配置（发布单元）

> **定位**：定义"运行什么"，由开发/发布人员配置，每次发布创建新版本

```go
type VersionDeployConfig struct {
    // ========== 镜像（必填） ==========
    Image           string `json:"image"`                      // 完整镜像地址或仅 tag

    // ========== 环境变量（增量） ==========
    // 合并规则：项目基础 + 版本覆盖
    Environment     map[string]string `json:"environment,omitempty"`

    // ========== 资源覆盖（可选） ==========
    // 如果设置，覆盖项目默认值
    Resources       *ResourceLimits `json:"resources,omitempty"`

    // ========== K8s 副本数覆盖（可选） ==========
    Replicas        *int `json:"replicas,omitempty"`

    // ========== 启动命令覆盖（可选） ==========
    Command         []string `json:"command,omitempty"`
    Entrypoint      []string `json:"entrypoint,omitempty"`
    WorkingDir      string   `json:"working_dir,omitempty"`

    // ========== 健康检查覆盖（可选） ==========
    // 完全覆盖项目配置
    HealthCheck     *ContainerHealthCheck `json:"health_check,omitempty"`

    // ========== 部署脚本（版本特定） ==========
    PreScript       string `json:"pre_script,omitempty"`       // 部署前脚本
    PostScript      string `json:"post_script,omitempty"`      // 部署后脚本

    // ========== K8s YAML 覆盖（高级） ==========
    // 如果设置，可以覆盖自动生成的 YAML
    K8sYAMLPatch    string `json:"k8s_yaml_patch,omitempty"`   // YAML patch
    K8sYAMLFull     string `json:"k8s_yaml_full,omitempty"`    // 完整 YAML（忽略其他配置）
}
```

---

### 3.3 任务级别配置（执行覆盖）

> **定位**：临时调整，用于金丝雀、A/B 测试、紧急扩容等场景

```go
type TaskOverrideConfig struct {
    // ========== 镜像覆盖（临时） ==========
    // 用于测试未发布的镜像
    Image           string `json:"image,omitempty"`

    // ========== 环境变量追加（临时） ==========
    // 追加到版本环境变量之上
    EnvironmentAdd  map[string]string `json:"environment_add,omitempty"`

    // ========== 资源覆盖（临时） ==========
    // 紧急扩容或金丝雀使用不同资源
    Resources       *ResourceLimits `json:"resources,omitempty"`

    // ========== 副本数覆盖（临时） ==========
    // 紧急扩缩容
    Replicas        *int `json:"replicas,omitempty"`

    // ========== 启动命令覆盖（临时） ==========
    // 调试用
    Command         []string `json:"command,omitempty"`
}
```

---

### 3.4 配置合并规则

```
最终配置 = 项目配置 + 版本覆盖 + 任务覆盖

合并优先级（从低到高）：
1. 项目配置（基础）
2. 版本配置（覆盖）
3. 任务配置（覆盖）

合并策略：
- 端口/卷/网络：仅使用项目配置（不可覆盖）
- 环境变量：递进合并（项目 → 版本追加 → 任务追加）
- 资源限制：完全覆盖（后者完全替代前者）
- 副本数：完全覆盖
- 命令/入口点：完全覆盖
- 健康检查：完全覆盖
```

**合并示例：**

```yaml
# 项目配置
project:
  ports: [{host: 8080, container: 80}]
  environment: {APP_ENV: production}
  default_resources: {memory_limit: 512m}

# 版本配置
version:
  image: myapp:v1.2.0
  environment: {LOG_LEVEL: info, FEATURE_X: enabled}
  resources: {memory_limit: 1g}

# 任务配置（金丝雀）
task:
  environment_add: {CANARY: true}
  resources: {memory_limit: 2g}

# 最终配置
final:
  ports: [{host: 8080, container: 80}]  # 仅项目
  image: myapp:v1.2.0                    # 版本
  environment:                           # 合并
    APP_ENV: production
    LOG_LEVEL: info
    FEATURE_X: enabled
    CANARY: true
  resources: {memory_limit: 2g}          # 任务覆盖
```

---

## 4. UI/UX 设计建议

### 4.1 项目创建/编辑界面

```
┌─────────────────────────────────────────────────────────────────┐
│  新建项目                                                        │
├─────────────────────────────────────────────────────────────────┤
│  基本信息                                                        │
│  ├─ 项目名称: [____________]                                     │
│  ├─ 部署类型: ○ 脚本 ○ 容器 ○ Git ○ K8s                          │
│  └─ 描述:     [____________]                                     │
├─────────────────────────────────────────────────────────────────┤
│  基础设施配置 (通常不需要修改)                       [展开/收起]   │
│  ├─ 镜像仓库                                                     │
│  │   ├─ Registry: [____________]                                │
│  │   └─ 认证:     [用户名] [密码]                                │
│  ├─ 网络配置                                                     │
│  │   ├─ 端口映射: [+ 添加端口]                                   │
│  │   └─ 网络模式: [bridge ▼]                                    │
│  ├─ 存储配置                                                     │
│  │   └─ 卷挂载: [+ 添加卷]                                       │
│  ├─ 安全配置                                                     │
│  │   └─ ☐ 特权模式                                              │
│  └─ 默认资源限制                                                 │
│      ├─ 内存: [512m]                                            │
│      └─ CPU:  [0.5]                                             │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 版本创建界面

```
┌─────────────────────────────────────────────────────────────────┐
│  新建版本                                                        │
├─────────────────────────────────────────────────────────────────┤
│  版本信息                                                        │
│  ├─ 版本号: [v1.2.0_______]                                     │
│  └─ 说明:   [新增功能X，修复BugY]                                 │
├─────────────────────────────────────────────────────────────────┤
│  发布配置                                                        │
│  ├─ 镜像*:  [myapp:v1.2.0__]                                    │
│  ├─ 环境变量: [+ 添加]                                           │
│  │   LOG_LEVEL=info                                             │
│  │   FEATURE_X=enabled                                          │
│  └─ 资源限制 (留空使用项目默认)                                   │
│      ├─ 内存: [1g____] (项目默认: 512m)                          │
│      └─ CPU:  [______] (项目默认: 0.5)                           │
├─────────────────────────────────────────────────────────────────┤
│  部署脚本 (可选)                                                  │
│  ├─ 部署前: [编辑脚本]                                           │
│  └─ 部署后: [编辑脚本]                                           │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 部署任务创建界面

```
┌─────────────────────────────────────────────────────────────────┐
│  创建部署任务                                                    │
├─────────────────────────────────────────────────────────────────┤
│  部署版本: v1.2.0                    操作: ○ 部署 ○ 回滚          │
├─────────────────────────────────────────────────────────────────┤
│  目标选择                                                        │
│  ☑ client-01                                                    │
│  ☑ client-02                                                    │
│  ☐ client-03                                                    │
├─────────────────────────────────────────────────────────────────┤
│  临时覆盖 (可选)                                     [展开/收起]   │
│  ├─ 镜像覆盖:    [____________] (默认: myapp:v1.2.0)             │
│  ├─ 追加环境变量: [+ 添加]                                        │
│  ├─ 副本数覆盖:   [__] (默认: 3)                                  │
│  └─ 资源覆盖:                                                    │
│      ├─ 内存: [____] (默认: 1g)                                  │
│      └─ CPU:  [____] (默认: 0.5)                                 │
├─────────────────────────────────────────────────────────────────┤
│  金丝雀配置                                                      │
│  ☑ 启用金丝雀发布                                                │
│  ├─ 比例: [10]%                                                  │
│  └─ 观察时间: [30] 分钟                                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. 数据模型修改

### 5.1 新增/修改的模型

```go
// VersionDeployConfig 版本部署配置
type VersionDeployConfig struct {
    // 镜像 (可以是完整地址或仅 tag)
    Image string `json:"image"`

    // 环境变量 (增量合并到项目配置)
    Environment map[string]string `json:"environment,omitempty"`

    // 资源限制覆盖
    Resources *ResourceLimits `json:"resources,omitempty"`

    // K8s 副本数覆盖
    Replicas *int `json:"replicas,omitempty"`

    // 启动命令覆盖
    Command    []string `json:"command,omitempty"`
    Entrypoint []string `json:"entrypoint,omitempty"`
    WorkingDir string   `json:"working_dir,omitempty"`

    // 健康检查覆盖 (完全替换)
    HealthCheck *ContainerHealthCheck `json:"health_check,omitempty"`

    // 部署脚本
    PreScript  string `json:"pre_script,omitempty"`
    PostScript string `json:"post_script,omitempty"`

    // K8s YAML (高级)
    K8sYAMLPatch string `json:"k8s_yaml_patch,omitempty"`
    K8sYAMLFull  string `json:"k8s_yaml_full,omitempty"`
}

// TaskOverrideConfig 任务覆盖配置
type TaskOverrideConfig struct {
    Image          string            `json:"image,omitempty"`
    EnvironmentAdd map[string]string `json:"environment_add,omitempty"`
    Resources      *ResourceLimits   `json:"resources,omitempty"`
    Replicas       *int              `json:"replicas,omitempty"`
    Command        []string          `json:"command,omitempty"`
}

// 修改 Version 模型
type Version struct {
    // ... 原有字段 ...

    // 新增：统一的部署配置
    DeployConfig *VersionDeployConfig `gorm:"type:jsonb" json:"deploy_config,omitempty"`
}

// 修改 DeployTask 模型
type DeployTask struct {
    // ... 原有字段 ...

    // 新增：任务覆盖配置
    OverrideConfig *TaskOverrideConfig `gorm:"type:jsonb" json:"override_config,omitempty"`
}
```

### 5.2 配置合并函数

```go
// MergeDeployConfig 合并配置
func MergeDeployConfig(
    project *ProjectContainerConfig,
    version *VersionDeployConfig,
    task *TaskOverrideConfig,
) *FinalDeployConfig {
    final := &FinalDeployConfig{}

    // 1. 复制项目基础配置（不可覆盖部分）
    final.Ports = project.Ports
    final.Volumes = project.Volumes
    final.Networks = project.Networks
    final.NetworkMode = project.NetworkMode
    final.Privileged = project.Privileged
    final.SecurityOpt = project.SecurityOpt
    final.LogDriver = project.LogDriver
    final.LogOpts = project.LogOpts
    final.RestartPolicy = project.RestartPolicy
    // ...

    // 2. 应用版本配置
    final.Image = version.Image
    final.Environment = mergeEnv(project.DefaultEnv, version.Environment)
    final.Resources = coalesce(version.Resources, project.DefaultResources)
    final.HealthCheck = coalesce(version.HealthCheck, project.HealthCheck)
    // ...

    // 3. 应用任务覆盖（如果有）
    if task != nil {
        if task.Image != "" {
            final.Image = task.Image
        }
        final.Environment = mergeEnv(final.Environment, task.EnvironmentAdd)
        if task.Resources != nil {
            final.Resources = task.Resources
        }
        if task.Replicas != nil {
            final.Replicas = *task.Replicas
        }
    }

    return final
}
```

---

## 6. 迁移计划

### 6.1 Phase 1: 数据模型升级

1. 新增 `VersionDeployConfig` 和 `TaskOverrideConfig` 类型
2. 修改 `Version` 模型，添加 `DeployConfig` 字段
3. 修改 `DeployTask` 模型，添加 `OverrideConfig` 字段
4. 编写数据迁移脚本，将现有配置迁移到新结构

### 6.2 Phase 2: 后端 API 升级

1. 修改版本创建/更新 API，支持新的配置结构
2. 修改任务创建 API，支持覆盖配置
3. 实现配置合并逻辑
4. 添加配置预览 API（显示合并后的最终配置）

### 6.3 Phase 3: 前端 UI 升级

1. 简化项目创建界面，只展示基础设施配置
2. 重新设计版本创建界面，突出发布配置
3. 添加任务创建时的覆盖配置区域
4. 添加配置预览功能

### 6.4 Phase 4: 兼容性处理

1. 保留旧字段的读取能力（向后兼容）
2. 新旧配置自动转换
3. 废弃旧字段，添加迁移提示

---

## 7. 总结

| 配置类型 | 项目级别 | 版本级别 | 任务级别 |
|----------|----------|----------|----------|
| 镜像仓库 | ✅ | | |
| 容器名称 | ✅ | | |
| 端口映射 | ✅ | | |
| 卷挂载 | ✅ | | |
| 网络配置 | ✅ | | |
| 安全配置 | ✅ | | |
| 设备/GPU | ✅ | | |
| 日志驱动 | ✅ | | |
| 重启策略 | ✅ | | |
| 镜像 tag | 默认值 | ✅ 必填 | ⚙️ 覆盖 |
| 环境变量 | 默认值 | ✅ 增量 | ⚙️ 追加 |
| 资源限制 | 默认值 | ⚙️ 覆盖 | ⚙️ 覆盖 |
| 副本数 | 默认值 | ⚙️ 覆盖 | ⚙️ 覆盖 |
| 启动命令 | | ⚙️ 覆盖 | ⚙️ 覆盖 |
| 健康检查 | 模板 | ⚙️ 覆盖 | |
| 部署脚本 | 默认值 | ⚙️ 条件覆盖 | |

---

## 8. 部署脚本优先级说明

### 8.1 部署脚本配置层级

部署脚本（`pre_script` 和 `post_script`）在三个层级都可以配置：

| 层级 | 作用 | 说明 |
|------|------|------|
| **项目级别** | 默认脚本 | 所有版本共享的默认部署脚本 |
| **版本级别** | 版本特定脚本 | 仅当非空时覆盖项目脚本 |
| **任务级别** | 不支持 | 部署脚本不能在任务级别临时覆盖 |

### 8.2 条件覆盖策略

部署脚本采用**条件覆盖**策略：

```
如果 版本脚本 != "" {
    使用版本脚本
} 否则 {
    使用项目脚本（默认值）
}
```

**策略说明：**

1. **项目脚本作为默认值**
   - 项目级别的 `pre_script` 和 `post_script` 作为所有版本的默认脚本
   - 适用于所有版本都需要执行的通用操作（如服务重启、缓存清理等）

2. **版本脚本条件覆盖**
   - 如果版本配置了非空的脚本，则完全替换项目脚本
   - 如果版本脚本为空，则继续使用项目默认脚本
   - 适用于版本特定的操作（如数据迁移、配置更新等）

### 8.3 使用场景

| 场景 | 项目脚本 | 版本脚本 | 最终执行 |
|------|----------|----------|----------|
| 仅使用通用脚本 | `restart.sh` | 空 | `restart.sh` |
| 版本特定脚本 | `restart.sh` | `migrate.sh` | `migrate.sh` |
| 无需脚本 | 空 | 空 | 无 |
| 仅版本脚本 | 空 | `deploy.sh` | `deploy.sh` |

### 8.4 配置示例

```yaml
# 项目配置
project:
  container_config:
    pre_script: |
      #!/bin/bash
      # 通用：停止旧服务
      docker stop myapp || true
    post_script: |
      #!/bin/bash
      # 通用：清理缓存
      docker exec myapp rm -rf /tmp/cache

# 版本配置 v1.2.0（使用版本特定脚本）
version:
  deploy_config:
    pre_script: |
      #!/bin/bash
      # 版本特定：数据迁移
      docker exec myapp ./migrate.sh
    post_script: ""  # 空值，使用项目的 post_script

# 版本配置 v1.3.0（使用项目默认脚本）
version:
  deploy_config:
    pre_script: ""   # 空值，使用项目的 pre_script
    post_script: ""  # 空值，使用项目的 post_script
```

### 8.5 支持的项目类型

| 项目类型 | 项目脚本 | 版本脚本 | 说明 |
|----------|----------|----------|------|
| Git Pull | ✅ | ✅ | 支持 `pre_script` 和 `post_script` |
| 容器部署 | ✅ | ✅ | 支持 `pre_script` 和 `post_script` |
| Kubernetes | ✅ | ✅ | 支持 `pre_script` 和 `post_script` |
| 脚本部署 | - | ✅ | 使用 `install_script`、`update_script` 等 |

---

**设计收益：**
1. 配置职责清晰，减少混乱
2. 版本是完整的发布单元，可独立回滚
3. 支持金丝雀、A/B 测试、紧急扩容等场景
4. 减少配置重复，降低出错概率
