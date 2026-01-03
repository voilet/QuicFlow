# Docker 容器发布完整需求设计

## 概述

本文档定义了 QUIC-Flow 发布系统中 Docker 容器部署的完整需求，涵盖 `docker run` 命令的所有常用参数。

## 参考示例

```bash
docker run -d \
  --privileged \
  --name k3s-server \
  -p 6443:6443 \
  -v /opt/data:/data \
  --restart=unless-stopped \
  rancher/k3s:v1.24.4-k3s1 server
```

## 当前实现状态

当前 `ContainerDeployConfig` 已支持：
- [x] 镜像配置 (image, registry, credentials, pull_policy)
- [x] 容器名称
- [x] 端口映射 (host_port, container_port, protocol, host_ip)
- [x] 卷挂载 (host_path, container_path, read_only, type)
- [x] 环境变量
- [x] 网络列表
- [x] 重启策略
- [x] 启动命令/入口点
- [x] 资源限制 (memory_limit, cpu_limit, memory_reserve)
- [x] 健康检查
- [x] 部署策略

## 缺失功能需求

### 1. 安全配置 (Security)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--privileged` | privileged | bool | 特权模式，给予容器所有权限 |
| `--cap-add` | cap_add | []string | 添加 Linux capabilities |
| `--cap-drop` | cap_drop | []string | 移除 Linux capabilities |
| `--security-opt` | security_opt | []string | 安全选项 (如 apparmor, seccomp) |
| `--read-only` | read_only_rootfs | bool | 只读根文件系统 |
| `--no-new-privileges` | no_new_privileges | bool | 禁止获取新权限 |

常用 Capabilities:
- `SYS_ADMIN`: 系统管理 (如 mount)
- `NET_ADMIN`: 网络管理
- `SYS_PTRACE`: 进程追踪
- `SYS_TIME`: 修改系统时间
- `IPC_LOCK`: 锁定内存

### 2. 用户配置 (User)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--user` | user | string | 运行用户 (UID:GID 或 username) |
| `--group-add` | group_add | []string | 附加用户组 |
| `--userns` | userns_mode | string | 用户命名空间模式 |

### 3. 网络配置 (Network)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--network` | network_mode | string | 网络模式 (bridge/host/none/container:name) |
| `--hostname` | hostname | string | 容器主机名 |
| `--domainname` | domainname | string | 域名 |
| `--dns` | dns | []string | DNS 服务器 |
| `--dns-search` | dns_search | []string | DNS 搜索域 |
| `--dns-opt` | dns_opt | []string | DNS 选项 |
| `--add-host` | extra_hosts | []string | 添加 /etc/hosts 条目 |
| `--mac-address` | mac_address | string | MAC 地址 |
| `--ip` | ip_address | string | IPv4 地址 |
| `--ip6` | ip6_address | string | IPv6 地址 |
| `--link` | links | []string | 连接其他容器 (已废弃但仍常用) |
| `--expose` | expose_ports | []int | 暴露端口但不映射 |

### 4. 设备映射 (Device)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--device` | devices | []DeviceMapping | 设备映射 |
| `--gpus` | gpus | string | GPU 配置 (all 或具体 GPU ID) |
| `--device-cgroup-rule` | device_cgroup_rules | []string | 设备 cgroup 规则 |

```go
type DeviceMapping struct {
    HostPath      string `json:"host_path"`       // 主机设备路径
    ContainerPath string `json:"container_path"`  // 容器设备路径
    Permissions   string `json:"permissions"`     // rwm 权限
}
```

### 5. 日志配置 (Logging)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--log-driver` | log_driver | string | 日志驱动 (json-file/syslog/journald/none) |
| `--log-opt` | log_opts | map[string]string | 日志驱动选项 |

常用日志选项:
- `max-size`: 单个日志文件最大大小 (如 10m)
- `max-file`: 保留的日志文件数量 (如 3)
- `compress`: 是否压缩轮转的日志文件

### 6. 资源限制增强 (Resource Limits)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--cpuset-cpus` | cpuset_cpus | string | CPU 绑定 (如 "0,1" 或 "0-3") |
| `--cpuset-mems` | cpuset_mems | string | 内存节点绑定 |
| `--cpu-shares` | cpu_shares | int | CPU 权重 (默认 1024) |
| `--memory-swap` | memory_swap | string | 内存+交换限制 |
| `--memory-swappiness` | memory_swappiness | int | 内存交换倾向 (0-100) |
| `--pids-limit` | pids_limit | int | 进程数限制 |
| `--ulimit` | ulimits | []Ulimit | 资源限制 |
| `--oom-kill-disable` | oom_kill_disable | bool | 禁用 OOM Killer |
| `--oom-score-adj` | oom_score_adj | int | OOM 分数调整 (-1000 到 1000) |
| `--shm-size` | shm_size | string | /dev/shm 大小 |

```go
type Ulimit struct {
    Name string `json:"name"`  // nofile, nproc, etc.
    Soft int64  `json:"soft"`  // 软限制
    Hard int64  `json:"hard"`  // 硬限制
}
```

### 7. 存储配置 (Storage)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--tmpfs` | tmpfs_mounts | []TmpfsMount | tmpfs 挂载 |
| `--storage-opt` | storage_opts | map[string]string | 存储驱动选项 |
| `--volume-driver` | volume_driver | string | 卷驱动 |

```go
type TmpfsMount struct {
    ContainerPath string `json:"container_path"`
    Size          string `json:"size,omitempty"`     // 大小限制
    Mode          string `json:"mode,omitempty"`     // 权限模式
}
```

### 8. 运行时配置 (Runtime)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--runtime` | runtime | string | 运行时 (runc/nvidia/etc) |
| `--workdir` | working_dir | string | 工作目录 |
| `--init` | init | bool | 使用 tini 作为 init 进程 |
| `--pid` | pid_mode | string | PID 命名空间 (host/container:name) |
| `--ipc` | ipc_mode | string | IPC 命名空间 |
| `--uts` | uts_mode | string | UTS 命名空间 |
| `--cgroup-parent` | cgroup_parent | string | 父 cgroup |
| `--sysctl` | sysctls | map[string]string | 内核参数 |
| `--stop-signal` | stop_signal | string | 停止信号 (默认 SIGTERM) |
| `--stop-timeout` | stop_timeout | int | 停止超时 (秒) |
| `--tty` | tty | bool | 分配 TTY |
| `--stdin-open` | stdin_open | bool | 保持 stdin 开启 |
| `--attach-stdin` | attach_stdin | bool | 附加到 stdin |
| `--attach-stdout` | attach_stdout | bool | 附加到 stdout |
| `--attach-stderr` | attach_stderr | bool | 附加到 stderr |

### 9. 标签和元数据 (Labels)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--label` | labels | map[string]string | 容器标签 |
| `--label-file` | label_file | string | 从文件读取标签 |
| `--annotation` | annotations | map[string]string | 容器注解 |

### 10. 平台配置 (Platform)

| 参数 | 字段名 | 类型 | 说明 |
|------|--------|------|------|
| `--platform` | platform | string | 目标平台 (linux/amd64, linux/arm64) |

## 完整数据模型设计

```go
// ContainerDeployConfig 容器部署配置（完整版）
type ContainerDeployConfig struct {
    // ==================== 镜像配置 ====================
    Image           string `json:"image"`                        // 镜像地址
    Registry        string `json:"registry,omitempty"`           // 镜像仓库
    RegistryUser    string `json:"registry_user,omitempty"`      // 仓库用户名
    RegistryPass    string `json:"registry_pass,omitempty"`      // 仓库密码
    ImagePullPolicy string `json:"image_pull_policy,omitempty"`  // always/ifnotpresent/never
    Platform        string `json:"platform,omitempty"`           // linux/amd64 等

    // ==================== 容器基础配置 ====================
    ContainerName string            `json:"container_name"`          // 容器名称
    Hostname      string            `json:"hostname,omitempty"`      // 主机名
    Domainname    string            `json:"domainname,omitempty"`    // 域名
    User          string            `json:"user,omitempty"`          // 运行用户
    GroupAdd      []string          `json:"group_add,omitempty"`     // 附加组
    WorkingDir    string            `json:"working_dir,omitempty"`   // 工作目录
    Environment   map[string]string `json:"environment,omitempty"`   // 环境变量
    Labels        map[string]string `json:"labels,omitempty"`        // 容器标签
    Command       []string          `json:"command,omitempty"`       // 启动命令
    Entrypoint    []string          `json:"entrypoint,omitempty"`    // 入口点

    // ==================== 端口配置 ====================
    Ports       []PortMapping `json:"ports,omitempty"`        // 端口映射
    ExposePorts []int         `json:"expose_ports,omitempty"` // 暴露端口

    // ==================== 网络配置 ====================
    NetworkMode string              `json:"network_mode,omitempty"`  // bridge/host/none/container:name
    Networks    []string            `json:"networks,omitempty"`      // 加入的网络
    DNS         []string            `json:"dns,omitempty"`           // DNS 服务器
    DNSSearch   []string            `json:"dns_search,omitempty"`    // DNS 搜索域
    DNSOpt      []string            `json:"dns_opt,omitempty"`       // DNS 选项
    ExtraHosts  []string            `json:"extra_hosts,omitempty"`   // 额外主机 (host:ip)
    MacAddress  string              `json:"mac_address,omitempty"`   // MAC 地址
    IPv4Address string              `json:"ipv4_address,omitempty"`  // IPv4 地址
    IPv6Address string              `json:"ipv6_address,omitempty"`  // IPv6 地址
    Links       []string            `json:"links,omitempty"`         // 连接容器

    // ==================== 存储配置 ====================
    Volumes      []VolumeMount  `json:"volumes,omitempty"`       // 卷挂载
    TmpfsMounts  []TmpfsMount   `json:"tmpfs_mounts,omitempty"`  // tmpfs 挂载
    VolumeDriver string         `json:"volume_driver,omitempty"` // 卷驱动
    StorageOpts  map[string]string `json:"storage_opts,omitempty"` // 存储选项

    // ==================== 安全配置 ====================
    Privileged       bool     `json:"privileged,omitempty"`          // 特权模式
    CapAdd           []string `json:"cap_add,omitempty"`             // 添加的 capabilities
    CapDrop          []string `json:"cap_drop,omitempty"`            // 移除的 capabilities
    SecurityOpt      []string `json:"security_opt,omitempty"`        // 安全选项
    ReadOnlyRootfs   bool     `json:"read_only_rootfs,omitempty"`    // 只读根文件系统
    NoNewPrivileges  bool     `json:"no_new_privileges,omitempty"`   // 禁止新权限
    UsernsMode       string   `json:"userns_mode,omitempty"`         // 用户命名空间

    // ==================== 设备配置 ====================
    Devices           []DeviceMapping `json:"devices,omitempty"`             // 设备映射
    GPUs              string          `json:"gpus,omitempty"`                // GPU 配置
    DeviceCgroupRules []string        `json:"device_cgroup_rules,omitempty"` // 设备 cgroup 规则

    // ==================== 资源限制 ====================
    MemoryLimit      string   `json:"memory_limit,omitempty"`       // 内存限制
    MemoryReserve    string   `json:"memory_reserve,omitempty"`     // 内存预留
    MemorySwap       string   `json:"memory_swap,omitempty"`        // 交换限制
    MemorySwappiness *int     `json:"memory_swappiness,omitempty"`  // 交换倾向
    CPULimit         string   `json:"cpu_limit,omitempty"`          // CPU 限制
    CPUShares        int      `json:"cpu_shares,omitempty"`         // CPU 权重
    CpusetCpus       string   `json:"cpuset_cpus,omitempty"`        // CPU 绑定
    CpusetMems       string   `json:"cpuset_mems,omitempty"`        // 内存节点绑定
    PidsLimit        int64    `json:"pids_limit,omitempty"`         // 进程数限制
    Ulimits          []Ulimit `json:"ulimits,omitempty"`            // ulimit 设置
    OomKillDisable   bool     `json:"oom_kill_disable,omitempty"`   // 禁用 OOM
    OomScoreAdj      int      `json:"oom_score_adj,omitempty"`      // OOM 分数
    ShmSize          string   `json:"shm_size,omitempty"`           // /dev/shm 大小

    // ==================== 运行时配置 ====================
    Runtime      string            `json:"runtime,omitempty"`        // 运行时
    Init         bool              `json:"init,omitempty"`           // 使用 init
    PidMode      string            `json:"pid_mode,omitempty"`       // PID 模式
    IpcMode      string            `json:"ipc_mode,omitempty"`       // IPC 模式
    UtsMode      string            `json:"uts_mode,omitempty"`       // UTS 模式
    CgroupParent string            `json:"cgroup_parent,omitempty"`  // 父 cgroup
    Sysctls      map[string]string `json:"sysctls,omitempty"`        // 内核参数
    StopSignal   string            `json:"stop_signal,omitempty"`    // 停止信号
    StopTimeout  int               `json:"stop_timeout,omitempty"`   // 停止超时
    Tty          bool              `json:"tty,omitempty"`            // 分配 TTY
    StdinOpen    bool              `json:"stdin_open,omitempty"`     // 保持 stdin

    // ==================== 日志配置 ====================
    LogDriver string            `json:"log_driver,omitempty"` // 日志驱动
    LogOpts   map[string]string `json:"log_opts,omitempty"`   // 日志选项

    // ==================== 健康检查 ====================
    HealthCheck *ContainerHealthCheck `json:"health_check,omitempty"`

    // ==================== 重启策略 ====================
    RestartPolicy      string `json:"restart_policy,omitempty"`       // no/always/on-failure/unless-stopped
    RestartMaxRetries  int    `json:"restart_max_retries,omitempty"`  // on-failure 最大重试次数

    // ==================== 部署策略 ====================
    RemoveOld      bool `json:"remove_old,omitempty"`       // 移除旧容器
    KeepOldCount   int  `json:"keep_old_count,omitempty"`   // 保留旧容器数
    PullBeforeStop bool `json:"pull_before_stop,omitempty"` // 先拉取再停止
    AutoRemove     bool `json:"auto_remove,omitempty"`      // 退出时自动删除
}
```

## 前端表单设计

### 1. 基础配置 Tab
- 镜像地址 (必填)
- 容器名称 (必填)
- 重启策略 (下拉选择)
- 启动命令 (可选)
- 工作目录 (可选)

### 2. 端口配置 Tab
- 端口映射列表 (动态添加)
  - 主机端口
  - 容器端口
  - 协议 (tcp/udp)
  - 绑定 IP (可选)

### 3. 存储配置 Tab
- 卷挂载列表 (动态添加)
  - 主机路径
  - 容器路径
  - 只读模式
  - 挂载类型
- Tmpfs 挂载

### 4. 网络配置 Tab
- 网络模式 (bridge/host/none/custom)
- 主机名
- DNS 服务器
- DNS 搜索域
- 额外 hosts

### 5. 环境变量 Tab
- 环境变量列表 (KEY=VALUE)
- 支持从文件导入

### 6. 资源限制 Tab
- 内存限制
- CPU 限制
- CPU 绑定
- 进程数限制
- Ulimit 配置

### 7. 安全配置 Tab (高级)
- 特权模式 (开关)
- Capabilities 配置
- 安全选项
- 只读根文件系统

### 8. 日志配置 Tab (高级)
- 日志驱动
- 日志选项 (max-size, max-file)

### 9. 健康检查 Tab
- 检查命令
- 间隔/超时/重试

## API 接口设计

### 创建项目时的容器配置
```json
POST /api/release/projects
{
  "name": "k3s-server",
  "type": "container",
  "container_config": {
    "image": "rancher/k3s:v1.24.4-k3s1",
    "container_name": "k3s-server",
    "privileged": true,
    "ports": [
      {"host_port": 6443, "container_port": 6443}
    ],
    "command": ["server"],
    "restart_policy": "unless-stopped"
  }
}
```

### 创建版本时可覆盖配置
```json
POST /api/release/projects/{id}/versions
{
  "version": "v1.24.4",
  "container_image": "rancher/k3s:v1.24.4-k3s1",
  "container_config_override": {
    "environment": {
      "K3S_TOKEN": "xxx"
    }
  }
}
```

## 实现任务清单

### Phase 1: 后端模型更新
1. [ ] 更新 `ContainerDeployConfig` 结构体添加所有新字段
2. [ ] 添加 `DeviceMapping`, `TmpfsMount`, `Ulimit` 等子类型
3. [ ] 更新数据库模型和迁移

### Phase 2: Docker 命令生成器
4. [ ] 实现 `GenerateDockerRunCommand()` 函数
5. [ ] 支持所有配置参数到命令行参数的转换
6. [ ] 添加命令验证逻辑

### Phase 3: API 更新
7. [ ] 更新项目创建/编辑 API 接收新字段
8. [ ] 更新版本创建 API 支持配置覆盖
9. [ ] 添加配置验证中间件

### Phase 4: 前端表单
10. [ ] 更新项目创建对话框 - 基础配置
11. [ ] 添加端口配置组件
12. [ ] 添加存储配置组件
13. [ ] 添加网络配置组件
14. [ ] 添加资源限制组件
15. [ ] 添加安全配置组件
16. [ ] 添加日志配置组件

### Phase 5: 部署执行器
17. [ ] 更新远程执行器支持新参数
18. [ ] 实现容器部署前验证
19. [ ] 实现容器健康检查等待

### Phase 6: 测试和文档
20. [ ] 单元测试
21. [ ] 集成测试
22. [ ] 更新 API 文档
