package handlers

import (
	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/router"
)

// Config 处理器配置
type Config struct {
	Version string                 // 客户端版本号
	Logger  *monitoring.Logger     // 可选，用于 Server 端处理器
}

// RegisterBuiltinHandlers 注册所有内置处理器
// 用法:
//
//	handlers.RegisterBuiltinHandlers(r, &handlers.Config{Version: "1.0.0"})
func RegisterBuiltinHandlers(r *router.Router, cfg *Config) {
	if cfg == nil {
		cfg = &Config{Version: "1.0.0"}
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}

	// 初始化状态（用于 GetStatus）
	InitStatus(cfg.Version)

	// 注册内置处理器（简洁的函数式风格）
	r.Register(command.CmdExecShell, ExecShell)
	r.Register(command.CmdGetStatus, GetStatus)

	// 网络相关处理器
	r.Register(command.CmdNetworkInterfaces, GetNetworkInterfaces)
	r.Register(command.CmdNetworkSpeed, GetNetworkSpeed)

	// 硬件信息处理器
	r.Register(command.CmdHardwareInfo, GetHardwareInfo)

	// 磁盘测试处理器
	r.Register(command.CmdDiskBenchmark, DiskBenchmark)
	r.Register(command.CmdDiskIOPS, DiskIOPS)

	// 进程采集处理器
	r.Register(command.CmdProcessCollect, ProcessCollect)
	r.Register(command.CmdProcessReport, ProcessReport)

	// 容器采集处理器
	r.Register(command.CmdContainerCollect, ContainerCollect)
	r.Register(command.CmdContainerReport, ContainerReport)
	r.Register(command.CmdContainerList, ContainerList)
	r.Register(command.CmdContainerDeploy, ContainerDeploy)
	r.Register(command.CmdContainerLogs, ContainerLogs)

	// K8s Pod 采集处理器
	r.Register(command.CmdK8sCollect, K8sCollect)
	r.Register(command.CmdK8sReport, K8sReport)
}

// ============================================================================
// 命令类型常量导出（方便直接使用）
// ============================================================================

const (
	CmdExecShell         = command.CmdExecShell
	CmdGetStatus         = command.CmdGetStatus
	CmdSystemInfo        = command.CmdSystemInfo
	CmdFileRead          = command.CmdFileRead
	CmdFileWrite         = command.CmdFileWrite
	CmdPing              = command.CmdPing
	CmdEcho              = command.CmdEcho
	CmdNetworkInterfaces = command.CmdNetworkInterfaces
	CmdNetworkSpeed      = command.CmdNetworkSpeed
	CmdHardwareInfo      = command.CmdHardwareInfo
	CmdDiskBenchmark     = command.CmdDiskBenchmark
	CmdDiskIOPS          = command.CmdDiskIOPS
	// 进程采集
	CmdProcessCollect = command.CmdProcessCollect
	CmdProcessReport  = command.CmdProcessReport
	// 容器采集
	CmdContainerCollect = command.CmdContainerCollect
	CmdContainerReport  = command.CmdContainerReport
	CmdContainerList    = command.CmdContainerList
	CmdContainerDeploy  = command.CmdContainerDeploy
	CmdContainerLogs    = command.CmdContainerLogs
	// K8s Pod 采集
	CmdK8sCollect = command.CmdK8sCollect
	CmdK8sReport  = command.CmdK8sReport
)
