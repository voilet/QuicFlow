package command

import (
	"encoding/json"
	"time"
)

// ============================================================================
// 命令类型常量（Server/Client 共享）
// ============================================================================

const (
	// 系统命令
	CmdExecShell  = "exec_shell"  // 执行 Shell 命令
	CmdGetStatus  = "get_status"  // 获取客户端状态
	CmdSystemInfo = "system.info" // 获取系统信息

	// 文件操作
	CmdFileRead  = "file.read"  // 读取文件
	CmdFileWrite = "file.write" // 写入文件
	CmdFileList  = "file.list"  // 列出目录

	// 进程管理
	CmdProcessList = "process.list" // 进程列表
	CmdProcessKill = "process.kill" // 终止进程

	// 服务管理
	CmdServiceStatus  = "service.status"  // 服务状态
	CmdServiceRestart = "service.restart" // 重启服务
	CmdServiceStop    = "service.stop"    // 停止服务
	CmdServiceStart   = "service.start"   // 启动服务

	// 配置管理
	CmdConfigGet    = "config.get"    // 获取配置
	CmdConfigUpdate = "config.update" // 更新配置

	// 网络诊断
	CmdNetworkPing       = "network.ping"       // Ping 测试
	CmdNetworkTrace      = "network.trace"      // 路由追踪
	CmdNetworkInterfaces = "network.interfaces" // 获取物理网卡列表
	CmdNetworkSpeed      = "network.speed"      // 获取网卡协商速率

	// 通用
	CmdPing = "ping" // 简单存活检测
	CmdEcho = "echo" // 回显测试

	// 硬件信息
	CmdHardwareInfo = "hardware.info" // 获取完整硬件信息

	// 磁盘测试
	CmdDiskBenchmark = "disk.benchmark" // 磁盘 IO 读写测试

	// 简化的磁盘 IOPS 检测
	CmdDiskIOPS = "disk.iops" // 简化的磁盘 IOPS 检测（只返回 Read/Write IOPS）

	// 发布系统命令
	CmdReleaseExecute   = "release.execute"   // 执行发布任务（脚本部署）
	CmdReleaseStatus    = "release.status"    // 上报发布状态
	CmdReleaseCheck     = "release.check"     // 检查安装状态
	CmdContainerDeploy  = "container.deploy"  // 容器部署
	CmdGitPullDeploy    = "gitpull.deploy"    // Git 拉取部署
	CmdGitVersions      = "git.versions"      // 获取 Git 仓库版本信息

	// 进程采集和上报命令
	CmdProcessCollect = "process.collect" // 采集进程信息
	CmdProcessReport  = "process.report"  // 上报进程信息

	// 容器采集和上报命令
	CmdContainerCollect = "container.collect" // 采集容器信息
	CmdContainerReport  = "container.report"  // 上报容器信息
	CmdContainerList    = "container.list"    // 列出容器

	// K8s Pod 采集和上报命令
	CmdK8sCollect = "k8s.collect" // 采集 K8s Pod 信息
	CmdK8sReport  = "k8s.report"  // 上报 K8s Pod 信息
)

// ============================================================================
// 共享 Payload/Result 结构（Server 构造，Client 解析/返回）
// ============================================================================

// --- Shell 命令 ---

// ShellParams exec_shell 命令的参数
type ShellParams struct {
	Command string `json:"command"`           // 要执行的命令
	Timeout int    `json:"timeout,omitempty"` // 超时时间（秒），默认30秒
	WorkDir string `json:"work_dir,omitempty"` // 工作目录（可选）
}

// ShellResult exec_shell 命令的结果
type ShellResult struct {
	Success  bool   `json:"success"`   // 是否成功
	ExitCode int    `json:"exit_code"` // 退出码
	Stdout   string `json:"stdout"`    // 标准输出
	Stderr   string `json:"stderr"`    // 标准错误
	Message  string `json:"message"`   // 消息
}

// --- 状态查询 ---

// StatusResult get_status 命令的结果
type StatusResult struct {
	Status      string `json:"status"`        // 状态（running/stopped）
	Uptime      int64  `json:"uptime"`        // 运行时间（秒）
	Version     string `json:"version"`       // 客户端版本
	Hostname    string `json:"hostname"`      // 主机名
	OS          string `json:"os"`            // 操作系统
	Arch        string `json:"arch"`          // CPU架构
	GoVersion   string `json:"go_version"`    // Go版本
	NumCPU      int    `json:"num_cpu"`       // CPU核心数
	NumGoroutine int   `json:"num_goroutine"` // Goroutine数量
}

// --- 文件操作 ---

// FileReadParams file.read 命令的参数
type FileReadParams struct {
	Path      string `json:"path"`                 // 文件路径
	MaxSize   int    `json:"max_size,omitempty"`   // 最大读取大小（字节）
	Encoding  string `json:"encoding,omitempty"`   // 编码（默认utf-8）
}

// FileReadResult file.read 命令的结果
type FileReadResult struct {
	Path     string `json:"path"`      // 文件路径
	Content  string `json:"content"`   // 文件内容
	Size     int64  `json:"size"`      // 文件大小
	Truncated bool  `json:"truncated"` // 是否被截断
}

// FileWriteParams file.write 命令的参数
type FileWriteParams struct {
	Path    string `json:"path"`              // 文件路径
	Content string `json:"content"`           // 文件内容
	Mode    string `json:"mode,omitempty"`    // 写入模式（overwrite/append）
	Perm    string `json:"perm,omitempty"`    // 文件权限（如 "0644"）
}

// FileWriteResult file.write 命令的结果
type FileWriteResult struct {
	Path    string `json:"path"`    // 文件路径
	Written int64  `json:"written"` // 写入字节数
	Success bool   `json:"success"` // 是否成功
}

// --- 服务管理 ---

// ServiceParams 服务操作命令的参数
type ServiceParams struct {
	Name string `json:"name"` // 服务名称
}

// ServiceResult 服务操作命令的结果
type ServiceResult struct {
	Name    string `json:"name"`    // 服务名称
	Status  string `json:"status"`  // 状态（running/stopped/unknown）
	Success bool   `json:"success"` // 操作是否成功
	Message string `json:"message"` // 消息
}

// --- 配置管理 ---

// ConfigGetParams config.get 命令的参数
type ConfigGetParams struct {
	Key string `json:"key"` // 配置键（空表示获取全部）
}

// ConfigUpdateParams config.update 命令的参数
type ConfigUpdateParams struct {
	Key   string      `json:"key"`   // 配置键
	Value interface{} `json:"value"` // 配置值
}

// ConfigResult 配置操作的结果
type ConfigResult struct {
	Success bool        `json:"success"`         // 是否成功
	Key     string      `json:"key,omitempty"`   // 配置键
	Value   interface{} `json:"value,omitempty"` // 配置值
	Message string      `json:"message"`         // 消息
}

// ============================================================================
// 以下是原有的命令状态和管理结构
// ============================================================================

// CommandStatus 命令执行状态
type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"   // 已下发，等待客户端执行
	CommandStatusExecuting CommandStatus = "executing" // 客户端正在执行
	CommandStatusCompleted CommandStatus = "completed" // 执行完成（成功）
	CommandStatusFailed    CommandStatus = "failed"    // 执行失败
	CommandStatusTimeout   CommandStatus = "timeout"   // 执行超时
	CommandStatusCancelled CommandStatus = "cancelled" // 已取消
)

// Command 命令信息
type Command struct {
	// 基本信息
	CommandID  string        `json:"command_id"`  // 命令唯一ID（等同于msg_id）
	ClientID   string        `json:"client_id"`   // 目标客户端ID
	CommandType string       `json:"command_type"` // 命令类型（业务自定义，如 "restart", "update_config" 等）
	Payload    json.RawMessage `json:"payload"`      // 命令参数（JSON格式）

	// 状态信息
	Status     CommandStatus `json:"status"`      // 当前状态
	Result     json.RawMessage `json:"result,omitempty"`     // 执行结果（JSON格式）
	Error      string        `json:"error,omitempty"`      // 错误信息

	// 时间信息
	CreatedAt  time.Time     `json:"created_at"`  // 创建时间
	SentAt     *time.Time    `json:"sent_at,omitempty"`     // 发送时间
	CompletedAt *time.Time   `json:"completed_at,omitempty"` // 完成时间
	Timeout    time.Duration `json:"timeout"`     // 超时时长
}

// CommandRequest HTTP请求结构 - 下发命令
type CommandRequest struct {
	ClientID    string          `json:"client_id" binding:"required"`    // 目标客户端
	CommandType string          `json:"command_type" binding:"required"` // 命令类型
	Payload     json.RawMessage `json:"payload"`                         // 命令参数
	Timeout     int             `json:"timeout,omitempty"`               // 超时时间（秒），默认30s
}

// CommandResponse HTTP响应结构 - 下发命令结果
type CommandResponse struct {
	Success   bool   `json:"success"`
	CommandID string `json:"command_id"`
	Message   string `json:"message"`
}

// CommandStatusResponse HTTP响应结构 - 查询命令状态
type CommandStatusResponse struct {
	Success bool     `json:"success"`
	Command *Command `json:"command,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// CommandExecutor 客户端命令执行器接口
// 业务层需要实现此接口来处理具体的命令
type CommandExecutor interface {
	// Execute 执行命令
	// commandType: 命令类型
	// payload: 命令参数（JSON格式）
	// 返回: 执行结果（JSON格式）和错误
	Execute(commandType string, payload []byte) (result []byte, err error)
}

// CommandPayload 命令载荷（放在DataMessage.Payload中）
type CommandPayload struct {
	CommandType  string          `json:"command_type"`            // 命令类型
	Payload      json.RawMessage `json:"payload"`                 // 命令参数
	NeedCallback bool            `json:"need_callback,omitempty"` // 是否需要异步回调（执行完毕后主动上报结果）
	CallbackID   string          `json:"callback_id,omitempty"`   // 回调ID（用于关联请求和回调）
}

// CallbackPayload 回调载荷（客户端执行完命令后发送给服务器）
type CallbackPayload struct {
	CallbackID  string          `json:"callback_id"`           // 回调ID（对应CommandPayload.CallbackID）
	CommandType string          `json:"command_type"`          // 原命令类型
	Success     bool            `json:"success"`               // 执行是否成功
	Result      json.RawMessage `json:"result,omitempty"`      // 执行结果
	Error       string          `json:"error,omitempty"`       // 错误信息
	Duration    int64           `json:"duration_ms,omitempty"` // 执行耗时（毫秒）
}

// MultiCommandRequest HTTP请求结构 - 多播命令（同时下发到多个客户端）
type MultiCommandRequest struct {
	ClientIDs   []string        `json:"client_ids" binding:"required,min=1"` // 目标客户端列表
	CommandType string          `json:"command_type" binding:"required"`     // 命令类型
	Payload     json.RawMessage `json:"payload"`                             // 命令参数
	Timeout     int             `json:"timeout,omitempty"`                   // 超时时间（秒），默认30s
}

// ClientCommandResult 单个客户端的命令执行结果
type ClientCommandResult struct {
	ClientID  string          `json:"client_id"`            // 客户端ID
	CommandID string          `json:"command_id"`           // 命令ID
	Status    CommandStatus   `json:"status"`               // 执行状态
	Result    json.RawMessage `json:"result,omitempty"`     // 执行结果
	Error     string          `json:"error,omitempty"`      // 错误信息
}

// MultiCommandResponse HTTP响应结构 - 多播命令结果
type MultiCommandResponse struct {
	TaskID       string                 `json:"task_id,omitempty"` // 任务ID（用于取消）
	Success      bool                   `json:"success"`            // 整体是否成功（所有命令都发送成功）
	Total        int                    `json:"total"`              // 总客户端数
	SuccessCount int                    `json:"success_count"`      // 成功发送的数量
	FailedCount  int                    `json:"failed_count"`      // 发送失败的数量
	CancelledCount int                  `json:"cancelled_count"`   // 已取消的数量
	Results      []*ClientCommandResult `json:"results"`            // 各客户端的结果
	Message      string                 `json:"message"`            // 摘要信息
	Status       string                 `json:"status,omitempty"`   // 任务状态（running/completed/cancelled）
}

// ============================================================================
// 网络接口相关结构
// ============================================================================

// NetworkInterfacesParams network.interfaces 命令的参数
type NetworkInterfacesParams struct {
	PhysicalOnly bool `json:"physical_only,omitempty"` // 是否只返回物理网卡（默认true）
}

// NetworkInterface 单个网卡信息
type NetworkInterface struct {
	Name         string   `json:"name"`                    // 接口名称（如 eth0, ens33）
	Index        int      `json:"index"`                   // 接口索引
	HardwareAddr string   `json:"hardware_addr,omitempty"` // MAC 地址
	MTU          int      `json:"mtu"`                     // MTU 大小
	Flags        []string `json:"flags"`                   // 接口标志（up, broadcast, multicast等）
	Addresses    []string `json:"addresses,omitempty"`     // IP 地址列表
	IsPhysical   bool     `json:"is_physical"`             // 是否为物理网卡
	IsUp         bool     `json:"is_up"`                   // 接口是否启用
	Driver       string   `json:"driver,omitempty"`        // 驱动名称
	Speed        int      `json:"speed,omitempty"`         // 协商速率（Mbps），-1表示未知
	Duplex       string   `json:"duplex,omitempty"`        // 双工模式（full/half/unknown）
	LinkDetected bool     `json:"link_detected"`           // 是否检测到链路
}

// NetworkInterfacesResult network.interfaces 命令的结果
type NetworkInterfacesResult struct {
	Interfaces []NetworkInterface `json:"interfaces"` // 网卡列表
	Count      int                `json:"count"`      // 网卡数量
}

// NetworkSpeedParams network.speed 命令的参数
type NetworkSpeedParams struct {
	InterfaceName string `json:"interface_name,omitempty"` // 指定接口名称，为空则返回所有
}

// NetworkSpeedInfo 单个网卡速率信息
type NetworkSpeedInfo struct {
	Name         string `json:"name"`          // 接口名称
	Speed        int    `json:"speed"`         // 协商速率（Mbps），-1表示未知
	Duplex       string `json:"duplex"`        // 双工模式（full/half/unknown）
	LinkDetected bool   `json:"link_detected"` // 是否检测到链路
	AutoNeg      bool   `json:"auto_neg"`      // 是否自动协商
	Driver       string `json:"driver"`        // 驱动名称
	BusInfo      string `json:"bus_info"`      // 总线信息（如 PCI 地址）
}

// NetworkSpeedResult network.speed 命令的结果
type NetworkSpeedResult struct {
	Interfaces []NetworkSpeedInfo `json:"interfaces"` // 网卡速率列表
	Count      int                `json:"count"`      // 网卡数量
}

// ============================================================================
// 硬件信息相关结构
// ============================================================================

// DMIInfo DMI/SMBIOS 信息
type DMIInfo struct {
	Uevent          string `json:"uevent,omitempty"`
	BiosDate        string `json:"bios_date,omitempty"`
	Modalias        string `json:"modalias,omitempty"`
	BoardName       string `json:"board_name,omitempty"`
	SysVendor       string `json:"sys_vendor,omitempty"`
	BiosVendor      string `json:"bios_vendor,omitempty"`
	BiosVersion     string `json:"bios_version,omitempty"`
	BoardSerial     string `json:"board_serial,omitempty"`
	BoardVendor     string `json:"board_vendor,omitempty"`
	ChassisType     string `json:"chassis_type,omitempty"`
	ProductName     string `json:"product_name,omitempty"`
	ProductUUID     string `json:"product_uuid,omitempty"`
	BoardVersion    string `json:"board_version,omitempty"`
	BoardAssetTag   string `json:"board_asset_tag,omitempty"`
	ChassisSerial   string `json:"chassis_serial,omitempty"`
	ChassisVendor   string `json:"chassis_vendor,omitempty"`
	ProductSerial   string `json:"product_serial,omitempty"`
	ChassisVersion  string `json:"chassis_version,omitempty"`
	ProductVersion  string `json:"product_version,omitempty"`
	ChassisAssetTag string `json:"chassis_asset_tag,omitempty"`
}

// HostInfo 主机信息
type HostInfo struct {
	OS                   string `json:"os"`
	Procs                uint64 `json:"procs"`
	HostID               string `json:"host_id"`
	Uptime               uint64 `json:"uptime"`
	BootTime             uint64 `json:"boot_time"`
	Hostname             string `json:"hostname"`
	Platform             string `json:"platform"`
	KernelArch           string `json:"kernel_arch"`
	KernelVersion        string `json:"kernel_version"`
	PlatformFamily       string `json:"platform_family"`
	PlatformVersion      string `json:"platform_version"`
	VirtualizationRole   string `json:"virtualization_role"`
	VirtualizationSystem string `json:"virtualization_system"`
}

// DiskMountUsage 磁盘挂载点使用情况
type DiskMountUsage struct {
	MountPoint  string  `json:"mount_point"`
	UsedPercent float64 `json:"used_percent"`
}

// DiskInfo 单个磁盘信息
type DiskInfo struct {
	Kind             string           `json:"kind"`                        // HDD/SSD/NVMe
	Type             string           `json:"type"`                        // disk/partition
	Model            string           `json:"model"`                       // 磁盘型号
	Device           string           `json:"device"`                      // 设备名（如 sda, sdb）
	IsSystemDisk     bool             `json:"is_system_disk"`              // 是否为系统盘
	SizeRoundedTB    float64          `json:"size_rounded_tb"`             // 容量（TiB，二进制计算 1TiB=1024^4）
	SizeTBDecimal    float64          `json:"size_tb_decimal"`             // 容量（TB，十进制计算 1TB=1000^4，厂商标注）
	SizeRoundedBytes uint64           `json:"size_rounded_bytes"`          // 容量（字节）
	MountUsages      []DiskMountUsage `json:"mount_usages,omitempty"`      // 挂载点使用情况
}

// MemoryModule 单个内存条信息
type MemoryModule struct {
	Size         string `json:"size"`          // 容量（如 "16384 MB"）
	Type         string `json:"type"`          // 类型（如 RAM, DDR4）
	Locator      string `json:"locator"`       // 插槽位置（如 DIMM 0）
	AssetTag     string `json:"asset_tag"`     // 资产标签
	PartNumber   string `json:"part_number"`   // 部件号
	Manufacturer string `json:"manufacturer"`  // 制造商
	SerialNumber string `json:"serial_number"` // 序列号
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	Count          int            `json:"count"`            // 内存条数量
	Speed          string         `json:"speed"`            // 内存速度
	Modules        []MemoryModule `json:"modules"`          // 内存条列表
	TotalGB        float64        `json:"total_gb"`         // 总容量（GB）
	TotalBytes     uint64         `json:"total_bytes"`      // 总容量（字节）
	TotalGBRounded int            `json:"total_gb_rounded"` // 总容量（GB，四舍五入）
}

// NICInfo 网卡信息
type NICInfo struct {
	IPv6       string `json:"ipv6,omitempty"`    // IPv6 地址
	Name       string `json:"name"`              // 网卡名称
	Speed      string `json:"speed"`             // 协商速率
	Status     string `json:"status"`            // 状态（up/down）
	IPAddress  string `json:"ip_address"`        // IPv4 地址
	IsPhysical bool   `json:"is_physical"`       // 是否为物理网卡
	MACAddress string `json:"mac_address"`       // MAC 地址
}

// HardwareInfoResult hardware.info 命令的完整结果
type HardwareInfoResult struct {
	DMI                        DMIInfo    `json:"dmi"`                           // DMI/SMBIOS 信息
	MAC                        string     `json:"mac"`                           // 主 MAC 地址（无分隔符）
	Host                       HostInfo   `json:"host"`                          // 主机信息
	ModelName                  string     `json:"model_name"`                    // CPU 型号
	Disks                      []DiskInfo `json:"disks"`                         // 磁盘列表
	Memory                     MemoryInfo `json:"memory"`                        // 内存信息
	NatID                      string     `json:"nat_id,omitempty"`              // NAT ID
	NatType                    string     `json:"nat_type,omitempty"`            // NAT 类型
	NICInfos                   []NICInfo  `json:"nic_infos"`                     // 网卡信息列表
	CPUCoreCount               int        `json:"cpu_core_count"`                // CPU 物理核心数
	CPUThreadCount             int        `json:"cpu_thread_count"`              // CPU 线程数
	TotalDiskCapacityTB        float64    `json:"total_disk_capacity_tb"`        // 总磁盘容量（TiB，二进制计算）
	TotalDiskCapacityTBDecimal float64    `json:"total_disk_capacity_tb_decimal"`// 总磁盘容量（TB，十进制计算，厂商标注）
	LogicalCPUFrequencyMHz     float64    `json:"logical_cpu_frequency_mhz"`     // 逻辑 CPU 频率（MHz）
	TotalDiskCapacityBytes     uint64     `json:"total_disk_capacity_bytes"`     // 总磁盘容量（字节）
	PhysicalCPUFrequencyMHz    float64    `json:"physical_cpu_frequency_mhz"`    // 物理 CPU 频率（MHz）
	SiblingsNum                string     `json:"siblings_num"`                  // 每个物理 CPU 的逻辑处理器数
	NumCPUKernel               int        `json:"num_cpu_kernel"`                // 内核报告的 CPU 数量
}

// ============================================================================
// 磁盘 IO 测试相关结构
// ============================================================================

// DiskBenchmarkParams disk.benchmark 命令的参数
type DiskBenchmarkParams struct {
	Device     string `json:"device,omitempty"`     // 指定设备名（如 nvme0n1），为空则测试所有非系统盘
	TestSize   string `json:"test_size,omitempty"`  // 测试文件大小（默认 1G）
	Runtime    int    `json:"runtime,omitempty"`    // 每项测试运行时间秒（默认 30）
	BlockSize  string `json:"block_size,omitempty"` // 块大小（默认 4k）
	NumJobs    int    `json:"numjobs,omitempty"`    // 并发任务数（默认 1）
	IODepth    int    `json:"iodepth,omitempty"`    // IO 队列深度（默认 32）
	Concurrent bool   `json:"concurrent,omitempty"` // 是否并发测试多块磁盘（默认 false，顺序测试）
}

// DiskBenchmarkResult 单个磁盘的测试结果
type DiskBenchmarkResult struct {
	Device string `json:"device"` // 设备名
	Model  string `json:"model"`  // 磁盘型号
	Kind   string `json:"kind"`   // 磁盘类型（HDD/SSD/NVMe）

	// 顺序读
	SeqReadIOPS       float64 `json:"seq_read_iops"`        // 顺序读 IOPS
	SeqReadBWMBps     float64 `json:"seq_read_bw_mbps"`     // 顺序读带宽 MB/s
	SeqReadLatencyUs  float64 `json:"seq_read_latency_us"`  // 顺序读平均延迟 μs

	// 顺序写
	SeqWriteIOPS      float64 `json:"seq_write_iops"`       // 顺序写 IOPS
	SeqWriteBWMBps    float64 `json:"seq_write_bw_mbps"`    // 顺序写带宽 MB/s
	SeqWriteLatencyUs float64 `json:"seq_write_latency_us"` // 顺序写平均延迟 μs

	// 随机读 4K
	RandReadIOPS      float64 `json:"rand_read_iops"`       // 随机读 IOPS
	RandReadBWMBps    float64 `json:"rand_read_bw_mbps"`    // 随机读带宽 MB/s
	RandReadLatencyUs float64 `json:"rand_read_latency_us"` // 随机读平均延迟 μs

	// 随机写 4K
	RandWriteIOPS      float64 `json:"rand_write_iops"`       // 随机写 IOPS
	RandWriteBWMBps    float64 `json:"rand_write_bw_mbps"`    // 随机写带宽 MB/s
	RandWriteLatencyUs float64 `json:"rand_write_latency_us"` // 随机写平均延迟 μs

	// 混合随机读写 (70% 读 30% 写)
	MixedIOPS      float64 `json:"mixed_iops"`       // 混合 IOPS
	MixedBWMBps    float64 `json:"mixed_bw_mbps"`    // 混合带宽 MB/s
	MixedLatencyUs float64 `json:"mixed_latency_us"` // 混合平均延迟 μs

	// 测试信息
	TestPath  string `json:"test_path"`  // 测试路径
	TestSize  string `json:"test_size"`  // 测试大小
	Duration  int    `json:"duration"`   // 测试总耗时（秒）
	Error     string `json:"error,omitempty"` // 错误信息
}

// DiskBenchmarkResponse disk.benchmark 命令的完整响应
type DiskBenchmarkResponse struct {
	Success    bool                   `json:"success"`     // 是否成功
	Results    []*DiskBenchmarkResult `json:"results"`     // 各磁盘测试结果
	TotalDisks int                    `json:"total_disks"` // 测试磁盘总数
	TestedAt   string                 `json:"tested_at"`   // 测试时间
	Message    string                 `json:"message,omitempty"` // 消息
}

// ============================================================================
// 简化的磁盘 IOPS 检测相关结构（使用单一 FIO 命令）
// ============================================================================

// DiskIOPSParams disk.iops 命令的参数
type DiskIOPSParams struct {
	Device  string `json:"device,omitempty"`  // 指定设备名（如 nvme0n1），为空则测试所有非系统盘
	Runtime int    `json:"runtime,omitempty"` // 测试运行时间秒（默认 60）
}

// DiskIOPSResult 单个磁盘的 IOPS 测试结果
type DiskIOPSResult struct {
	Device    string  `json:"device"`              // 设备名
	Model     string  `json:"model"`               // 磁盘型号
	Kind      string  `json:"kind"`                // 磁盘类型（HDD/SSD/NVMe）
	ReadIOPS  float64 `json:"read_iops"`           // 读 IOPS
	WriteIOPS float64 `json:"write_iops"`          // 写 IOPS
	TestPath  string  `json:"test_path"`           // 测试路径
	Duration  int     `json:"duration"`            // 测试耗时（秒）
	Error     string  `json:"error,omitempty"`     // 错误信息
}

// DiskIOPSResponse disk.iops 命令的完整响应
type DiskIOPSResponse struct {
	Success    bool              `json:"success"`               // 是否成功
	Results    []*DiskIOPSResult `json:"results"`               // 各磁盘测试结果
	TotalDisks int               `json:"total_disks"`           // 测试磁盘总数
	TestedAt   string            `json:"tested_at"`             // 测试时间
	Message    string            `json:"message,omitempty"`     // 消息
}

// ============================================================================
// 发布系统相关结构
// ============================================================================

// ReleaseOperationType 发布操作类型
type ReleaseOperationType string

const (
	ReleaseOpDeploy    ReleaseOperationType = "deploy"    // 部署（自动判断安装/更新）
	ReleaseOpInstall   ReleaseOperationType = "install"   // 强制安装
	ReleaseOpUpdate    ReleaseOperationType = "update"    // 强制更新
	ReleaseOpRollback  ReleaseOperationType = "rollback"  // 回滚
	ReleaseOpUninstall ReleaseOperationType = "uninstall" // 卸载
)

// ReleaseExecuteParams release.execute 命令的参数
type ReleaseExecuteParams struct {
	ReleaseID   string               `json:"release_id"`            // 发布ID
	TargetID    string               `json:"target_id"`             // 目标ID
	Operation   ReleaseOperationType `json:"operation"`             // 操作类型
	Version     string               `json:"version"`               // 版本号
	Script      string               `json:"script"`                // 脚本内容（已解析变量）
	WorkDir     string               `json:"work_dir,omitempty"`    // 工作目录
	Environment map[string]string    `json:"environment,omitempty"` // 环境变量
	Timeout     int                  `json:"timeout,omitempty"`     // 超时时间（秒）
	Interpreter string               `json:"interpreter,omitempty"` // 脚本解释器（默认 /bin/bash）

	// 进程脱离选项
	DetachProcess bool   `json:"detach_process,omitempty"` // 是否脱离进程（脚本启动的进程独立于Client运行）
	DetachMethod  string `json:"detach_method,omitempty"`  // 脱离方式: setsid, nohup, systemd
}

// ReleaseExecuteResult release.execute 命令的结果
type ReleaseExecuteResult struct {
	Success    bool   `json:"success"`              // 是否成功
	ReleaseID  string `json:"release_id"`           // 发布ID
	TargetID   string `json:"target_id"`            // 目标ID
	Operation  string `json:"operation"`            // 操作类型
	ExitCode   int    `json:"exit_code"`            // 退出码
	Output     string `json:"output"`               // 标准输出
	Error      string `json:"error,omitempty"`      // 错误信息
	StartedAt  string `json:"started_at"`           // 开始时间
	FinishedAt string `json:"finished_at"`          // 完成时间
	Duration   int64  `json:"duration_ms"`          // 耗时（毫秒）
}

// ReleaseStatusParams release.status 命令的参数（状态上报）
type ReleaseStatusParams struct {
	ReleaseID   string `json:"release_id"`             // 发布ID
	TargetID    string `json:"target_id"`              // 目标ID
	Status      string `json:"status"`                 // 状态（running/success/failed）
	Progress    int    `json:"progress,omitempty"`     // 进度百分比（0-100）
	Message     string `json:"message,omitempty"`      // 状态消息
	Output      string `json:"output,omitempty"`       // 输出内容
	Error       string `json:"error,omitempty"`        // 错误信息
	Version     string `json:"version,omitempty"`      // 当前版本
	ReportedAt  string `json:"reported_at"`            // 上报时间
}

// ReleaseStatusResult release.status 命令的响应
type ReleaseStatusResult struct {
	Success bool   `json:"success"`          // 是否成功接收
	Message string `json:"message,omitempty"` // 响应消息
}

// ReleaseCheckParams release.check 命令的参数
type ReleaseCheckParams struct {
	WorkDir string `json:"work_dir"` // 工作目录
}

// ReleaseCheckResult release.check 命令的结果
type ReleaseCheckResult struct {
	Installed      bool   `json:"installed"`                 // 是否已安装
	Version        string `json:"version,omitempty"`         // 当前版本
	InstallPath    string `json:"install_path,omitempty"`    // 安装路径
	InstalledAt    string `json:"installed_at,omitempty"`    // 安装时间
	LastUpdatedAt  string `json:"last_updated_at,omitempty"` // 最后更新时间
	Error          string `json:"error,omitempty"`           // 错误信息
}

// ============================================================================
// 容器部署相关结构
// ============================================================================

// ContainerDeployParams container.deploy 命令的参数
type ContainerDeployParams struct {
	ReleaseID   string               `json:"release_id"`             // 发布ID
	TargetID    string               `json:"target_id"`              // 目标ID
	Operation   ReleaseOperationType `json:"operation"`              // 操作类型
	Version     string               `json:"version"`                // 版本号

	// 镜像配置
	Image            string `json:"image"`                         // 镜像地址
	Registry         string `json:"registry,omitempty"`            // 镜像仓库
	RegistryUser     string `json:"registry_user,omitempty"`       // 仓库用户
	RegistryPass     string `json:"registry_pass,omitempty"`       // 仓库密码
	ImagePullPolicy  string `json:"image_pull_policy,omitempty"`   // 拉取策略

	// 容器配置
	ContainerName string            `json:"container_name"`            // 容器名称
	Ports         []PortMappingCmd  `json:"ports,omitempty"`           // 端口映射
	Volumes       []VolumeMountCmd  `json:"volumes,omitempty"`         // 卷挂载
	Environment   map[string]string `json:"environment,omitempty"`     // 环境变量
	Networks      []string          `json:"networks,omitempty"`        // 网络
	RestartPolicy string            `json:"restart_policy,omitempty"`  // 重启策略
	Command       []string          `json:"command,omitempty"`         // 启动命令
	Entrypoint    []string          `json:"entrypoint,omitempty"`      // 入口点

	// 资源限制
	MemoryLimit   string `json:"memory_limit,omitempty"`
	CPULimit      string `json:"cpu_limit,omitempty"`

	// 健康检查
	HealthCheck *ContainerHealthCheckCmd `json:"health_check,omitempty"`

	// 部署选项
	StopTimeout    int  `json:"stop_timeout,omitempty"`     // 停止超时
	RemoveOld      bool `json:"remove_old,omitempty"`       // 移除旧容器
	PullBeforeStop bool `json:"pull_before_stop,omitempty"` // 先拉取再停止
	Timeout        int  `json:"timeout,omitempty"`          // 总超时时间
}

// PortMappingCmd 端口映射命令结构
type PortMappingCmd struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
	HostIP        string `json:"host_ip,omitempty"`
}

// VolumeMountCmd 卷挂载命令结构
type VolumeMountCmd struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only,omitempty"`
}

// ContainerHealthCheckCmd 容器健康检查命令结构
type ContainerHealthCheckCmd struct {
	Command     []string `json:"command"`
	Interval    int      `json:"interval,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
	Retries     int      `json:"retries,omitempty"`
	StartPeriod int      `json:"start_period,omitempty"`
}

// ContainerDeployResult container.deploy 命令的结果
type ContainerDeployResult struct {
	Success       bool   `json:"success"`                    // 是否成功
	ReleaseID     string `json:"release_id"`                 // 发布ID
	TargetID      string `json:"target_id"`                  // 目标ID
	Operation     string `json:"operation"`                  // 操作类型
	ContainerID   string `json:"container_id,omitempty"`     // 新容器ID
	ContainerName string `json:"container_name,omitempty"`   // 容器名称
	ImagePulled   bool   `json:"image_pulled"`               // 是否拉取了镜像
	OldRemoved    bool   `json:"old_removed"`                // 旧容器是否已移除
	Output        string `json:"output,omitempty"`           // 输出信息
	Error         string `json:"error,omitempty"`            // 错误信息
	StartedAt     string `json:"started_at"`                 // 开始时间
	FinishedAt    string `json:"finished_at"`                // 完成时间
	Duration      int64  `json:"duration_ms"`                // 耗时（毫秒）
}

// ============================================================================
// Git 拉取部署相关结构
// ============================================================================

// GitPullDeployParams gitpull.deploy 命令的参数
type GitPullDeployParams struct {
	ReleaseID   string               `json:"release_id"`             // 发布ID
	TargetID    string               `json:"target_id"`              // 目标ID
	Operation   ReleaseOperationType `json:"operation"`              // 操作类型
	Version     string               `json:"version"`                // 版本号

	// Git 仓库配置
	RepoURL    string `json:"repo_url"`               // 仓库地址
	Branch     string `json:"branch,omitempty"`       // 分支
	Tag        string `json:"tag,omitempty"`          // 标签
	Commit     string `json:"commit,omitempty"`       // 指定 commit
	Depth      int    `json:"depth,omitempty"`        // 克隆深度
	Submodules bool   `json:"submodules,omitempty"`   // 初始化子模块

	// 认证配置
	AuthType string `json:"auth_type,omitempty"` // none, ssh, token, basic
	SSHKey   string `json:"ssh_key,omitempty"`   // SSH 私钥
	Token    string `json:"token,omitempty"`     // Access Token
	Username string `json:"username,omitempty"`  // 用户名
	Password string `json:"password,omitempty"`  // 密码

	// 部署配置
	WorkDir      string `json:"work_dir"`                // 工作目录
	CleanBefore  bool   `json:"clean_before,omitempty"`  // 部署前清理
	BackupBefore bool   `json:"backup_before,omitempty"` // 部署前备份
	BackupDir    string `json:"backup_dir,omitempty"`    // 备份目录

	// 部署脚本
	PreScript     string            `json:"pre_script,omitempty"`     // 部署前脚本
	PostScript    string            `json:"post_script,omitempty"`    // 部署后脚本
	Environment   map[string]string `json:"environment,omitempty"`    // 环境变量
	Interpreter   string            `json:"interpreter,omitempty"`    // 脚本解释器

	// 超时配置
	CloneTimeout  int `json:"clone_timeout,omitempty"`  // 克隆超时
	ScriptTimeout int `json:"script_timeout,omitempty"` // 脚本超时
	Timeout       int `json:"timeout,omitempty"`        // 总超时时间
}

// GitPullDeployResult gitpull.deploy 命令的结果
type GitPullDeployResult struct {
	Success       bool   `json:"success"`                  // 是否成功
	ReleaseID     string `json:"release_id"`               // 发布ID
	TargetID      string `json:"target_id"`                // 目标ID
	Operation     string `json:"operation"`                // 操作类型
	GitOutput     string `json:"git_output,omitempty"`     // Git 命令输出
	ScriptOutput  string `json:"script_output,omitempty"`  // 脚本输出
	Commit        string `json:"commit,omitempty"`         // 当前 commit hash
	Branch        string `json:"branch,omitempty"`         // 当前分支
	BackupPath    string `json:"backup_path,omitempty"`    // 备份路径
	CleanedBefore bool   `json:"cleaned_before"`           // 是否清理过
	BackedUpBefore bool  `json:"backed_up_before"`         // 是否备份过
	Error         string `json:"error,omitempty"`          // 错误信息
	StartedAt     string `json:"started_at"`               // 开始时间
	FinishedAt    string `json:"finished_at"`              // 完成时间
	Duration      int64  `json:"duration_ms"`              // 耗时（毫秒）
}

// ============================================================================
// Git 版本信息相关结构
// ============================================================================

// GitVersionsParams git.versions 命令的参数
type GitVersionsParams struct {
	// Git 仓库配置
	RepoURL  string `json:"repo_url"`            // 仓库地址
	WorkDir  string `json:"work_dir,omitempty"`  // 工作目录（如果已 clone）

	// 认证配置
	AuthType string `json:"auth_type,omitempty"` // none, ssh, token, basic
	SSHKey   string `json:"ssh_key,omitempty"`   // SSH 私钥
	Token    string `json:"token,omitempty"`     // Access Token
	Username string `json:"username,omitempty"`  // 用户名
	Password string `json:"password,omitempty"`  // 密码

	// 查询选项
	MaxTags    int  `json:"max_tags,omitempty"`    // 最大返回 tag 数量，默认 20
	MaxCommits int  `json:"max_commits,omitempty"` // 最大返回 commit 数量，默认 10
	IncludeBranches bool `json:"include_branches,omitempty"` // 是否包含分支列表
}

// GitTag Git 标签信息
type GitTag struct {
	Name      string `json:"name"`                 // 标签名
	Commit    string `json:"commit"`               // 对应的 commit hash
	Message   string `json:"message,omitempty"`    // 标签消息（annotated tag）
	CreatedAt string `json:"created_at,omitempty"` // 创建时间
}

// GitBranch Git 分支信息
type GitBranch struct {
	Name      string `json:"name"`                 // 分支名
	Commit    string `json:"commit"`               // 最新 commit hash
	IsDefault bool   `json:"is_default,omitempty"` // 是否为默认分支
	IsRemote  bool   `json:"is_remote,omitempty"`  // 是否为远程分支
}

// GitCommit Git 提交信息
type GitCommit struct {
	Hash      string `json:"hash"`                 // commit hash (短)
	FullHash  string `json:"full_hash"`            // commit hash (完整)
	Author    string `json:"author"`               // 作者
	Email     string `json:"email,omitempty"`      // 作者邮箱
	Message   string `json:"message"`              // 提交消息
	CreatedAt string `json:"created_at"`           // 提交时间
}

// GitVersionsResult git.versions 命令的结果
type GitVersionsResult struct {
	Success       bool        `json:"success"`                  // 是否成功
	RepoURL       string      `json:"repo_url"`                 // 仓库地址
	DefaultBranch string      `json:"default_branch,omitempty"` // 默认分支
	Tags          []GitTag    `json:"tags,omitempty"`           // 标签列表（按时间倒序）
	Branches      []GitBranch `json:"branches,omitempty"`       // 分支列表
	RecentCommits []GitCommit `json:"recent_commits,omitempty"` // 最近提交列表
	CurrentCommit string      `json:"current_commit,omitempty"` // 当前 commit（如果已 clone）
	CurrentBranch string      `json:"current_branch,omitempty"` // 当前分支（如果已 clone）
	Error         string      `json:"error,omitempty"`          // 错误信息
}

// ============================================================================
// 进程采集和上报相关结构
// ============================================================================

// ProcessMatchRuleCmd 进程匹配规则
type ProcessMatchRuleCmd struct {
	Type    string `json:"type"`    // name, cmdline, pidfile, port
	Pattern string `json:"pattern"` // 匹配模式
	Name    string `json:"name"`    // 显示名称
}

// ProcessCollectParams process.collect 命令的参数
type ProcessCollectParams struct {
	ProjectID string                `json:"project_id,omitempty"` // 项目ID
	Rules     []ProcessMatchRuleCmd `json:"rules"`                // 匹配规则
	Timeout   int                   `json:"timeout,omitempty"`    // 超时时间（秒）
}

// ProcessInfoCmd 进程信息
type ProcessInfoCmd struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	Cmdline    string  `json:"cmdline"`
	StartTime  string  `json:"start_time"`
	Status     string  `json:"status"`      // running, sleeping, zombie
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   float64 `json:"memory_mb"`
	MemoryPct  float64 `json:"memory_pct"`
	MatchedBy  string  `json:"matched_by"`  // 匹配规则名称
}

// ProcessCollectResult process.collect 命令的结果
type ProcessCollectResult struct {
	Success   bool             `json:"success"`
	Processes []ProcessInfoCmd `json:"processes"`
	Error     string           `json:"error,omitempty"`
}

// ProcessReportParams process.report 命令的参数
type ProcessReportParams struct {
	ClientID   string           `json:"client_id"`
	ProjectID  string           `json:"project_id"`
	ReleaseID  string           `json:"release_id,omitempty"`
	Version    string           `json:"version,omitempty"`
	Processes  []ProcessInfoCmd `json:"processes"`
	ReportedAt string           `json:"reported_at"`
}

// ProcessReportResult process.report 命令的结果
type ProcessReportResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ============================================================================
// 容器采集和上报相关结构
// ============================================================================

// ContainerCollectParams container.collect 命令的参数
type ContainerCollectParams struct {
	Prefixes []string `json:"prefixes,omitempty"` // 容器名称前缀过滤
	All      bool     `json:"all"`                // 是否包含所有容器（包括已停止的）
	Timeout  int      `json:"timeout,omitempty"`  // 超时时间（秒）
}

// ContainerInfoCmd 容器信息
type ContainerInfoCmd struct {
	ContainerID   string  `json:"container_id"`
	ContainerName string  `json:"container_name"`
	Image         string  `json:"image"`
	Status        string  `json:"status"`     // running, exited, paused
	State         string  `json:"state"`      // created, running, paused, restarting, removing, exited, dead
	CreatedAt     string  `json:"created_at"`
	StartedAt     string  `json:"started_at"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   int64   `json:"memory_usage"`   // bytes
	MemoryLimit   int64   `json:"memory_limit"`   // bytes
	MemoryPercent float64 `json:"memory_percent"`
	NetworkRx     int64   `json:"network_rx"`     // bytes
	NetworkTx     int64   `json:"network_tx"`     // bytes
	MatchedPrefix string  `json:"matched_prefix,omitempty"`
	MatchedProject string `json:"matched_project,omitempty"`
}

// ContainerCollectResult container.collect 命令的结果
type ContainerCollectResult struct {
	Success       bool               `json:"success"`
	Containers    []ContainerInfoCmd `json:"containers"`
	DockerVersion string             `json:"docker_version,omitempty"`
	TotalCount    int                `json:"total_count"`
	RunningCount  int                `json:"running_count"`
	Error         string             `json:"error,omitempty"`
}

// ContainerReportParams container.report 命令的参数
type ContainerReportParams struct {
	ClientID      string             `json:"client_id"`
	ProjectID     string             `json:"project_id,omitempty"`
	Containers    []ContainerInfoCmd `json:"containers"`
	DockerVersion string             `json:"docker_version,omitempty"`
	TotalCount    int                `json:"total_count"`
	RunningCount  int                `json:"running_count"`
	ReportedAt    string             `json:"reported_at"`
}

// ContainerReportResult container.report 命令的结果
type ContainerReportResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// ContainerListParams container.list 命令的参数
type ContainerListParams struct {
	All      bool     `json:"all"`                // 是否包含所有容器
	Prefixes []string `json:"prefixes,omitempty"` // 名称前缀过滤
}

// ContainerListResult container.list 命令的结果
type ContainerListResult struct {
	Success    bool               `json:"success"`
	Containers []ContainerInfoCmd `json:"containers"`
	Error      string             `json:"error,omitempty"`
}

// ============================================================================
// K8s Pod 采集和上报相关结构
// ============================================================================

// K8sCollectParams k8s.collect 命令的参数
type K8sCollectParams struct {
	APIServer     string `json:"api_server,omitempty"`     // API Server 地址
	Token         string `json:"token,omitempty"`          // Bearer Token
	TokenFile     string `json:"token_file,omitempty"`     // Token 文件路径
	Namespace     string `json:"namespace,omitempty"`      // 命名空间（空表示所有）
	LabelSelector string `json:"label_selector,omitempty"` // 标签选择器
	InCluster     bool   `json:"in_cluster,omitempty"`     // 是否在集群内运行
	InsecureTLS   bool   `json:"insecure_tls,omitempty"`   // 跳过 TLS 验证
	Timeout       int    `json:"timeout,omitempty"`        // 超时时间（秒）
}

// K8sContainerStatusCmd 容器状态
type K8sContainerStatusCmd struct {
	Name         string `json:"name"`                   // 容器名称
	Image        string `json:"image"`                  // 镜像
	Ready        bool   `json:"ready"`                  // 是否就绪
	RestartCount int    `json:"restart_count"`          // 重启次数
	State        string `json:"state"`                  // 状态（running/waiting/terminated）
	StartedAt    string `json:"started_at,omitempty"`   // 启动时间
	Reason       string `json:"reason,omitempty"`       // 原因
	Message      string `json:"message,omitempty"`      // 消息
}

// K8sPodInfoCmd Pod 信息
type K8sPodInfoCmd struct {
	Name         string                  `json:"name"`                    // Pod 名称
	Namespace    string                  `json:"namespace"`               // 命名空间
	UID          string                  `json:"uid"`                     // UID
	Status       string                  `json:"status"`                  // 状态
	Phase        string                  `json:"phase"`                   // 阶段
	HostIP       string                  `json:"host_ip,omitempty"`       // 主机 IP
	PodIP        string                  `json:"pod_ip,omitempty"`        // Pod IP
	StartTime    string                  `json:"start_time,omitempty"`    // 启动时间
	Labels       map[string]string       `json:"labels,omitempty"`        // 标签
	Containers   []K8sContainerStatusCmd `json:"containers,omitempty"`    // 容器列表
	RestartCount int                     `json:"restart_count"`           // 总重启次数
	Ready        bool                    `json:"ready"`                   // 是否就绪
}

// K8sCollectResult k8s.collect 命令的结果
type K8sCollectResult struct {
	Success      bool            `json:"success"`                 // 是否成功
	Pods         []K8sPodInfoCmd `json:"pods,omitempty"`          // Pod 列表
	TotalCount   int             `json:"total_count"`             // 总数
	RunningCount int             `json:"running_count"`           // 运行中数量
	ReadyCount   int             `json:"ready_count"`             // 就绪数量
	PendingCount int             `json:"pending_count"`           // 等待中数量
	FailedCount  int             `json:"failed_count"`            // 失败数量
	Error        string          `json:"error,omitempty"`         // 错误信息
}

// K8sReportParams k8s.report 命令的参数
type K8sReportParams struct {
	ClientID     string          `json:"client_id"`               // 客户端 ID
	ProjectID    string          `json:"project_id,omitempty"`    // 项目 ID
	Namespace    string          `json:"namespace,omitempty"`     // 命名空间
	Pods         []K8sPodInfoCmd `json:"pods"`                    // Pod 列表
	TotalCount   int             `json:"total_count"`             // 总数
	RunningCount int             `json:"running_count"`           // 运行中数量
	ReadyCount   int             `json:"ready_count"`             // 就绪数量
	ReportedAt   string          `json:"reported_at"`             // 上报时间
}

// K8sReportResult k8s.report 命令的结果
type K8sReportResult struct {
	Success bool   `json:"success"`           // 是否成功
	Error   string `json:"error,omitempty"`   // 错误信息
}
