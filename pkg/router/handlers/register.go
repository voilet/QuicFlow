package handlers

import (
	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/router"
)

// Config 处理器配置
type Config struct {
	Logger  *monitoring.Logger
	Version string // 客户端版本号
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Logger:  monitoring.NewLogger(monitoring.LogLevelInfo, "text"),
		Version: "1.0.0",
	}
}

// ============================================================================
// 处理器注册辅助函数（供 cmd/client 选择性调用）
// ============================================================================

// RegisterShellHandler 注册 Shell 命令处理器
// 命令类型: exec_shell
func RegisterShellHandler(r *router.Router, logger *monitoring.Logger) {
	handler := NewShellHandler(logger)
	r.Register(command.CmdExecShell, handler.Handle)
	logger.Debug("Shell handler registered", "command", command.CmdExecShell)
}

// RegisterStatusHandler 注册状态查询处理器
// 命令类型: get_status
func RegisterStatusHandler(r *router.Router, logger *monitoring.Logger, version string) {
	handler := NewStatusHandler(logger, version)
	r.Register(command.CmdGetStatus, handler.Handle)
	logger.Debug("Status handler registered", "command", command.CmdGetStatus)
}

// ============================================================================
// 统一注册（推荐方式）
// ============================================================================

// RegisterBuiltinHandlers 注册所有内置处理器
// 统一注册保证 Server/Client 命令类型一致性
func RegisterBuiltinHandlers(r *router.Router, cfg *Config) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.Logger == nil {
		cfg.Logger = monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}

	// 注册所有内置处理器
	RegisterShellHandler(r, cfg.Logger)
	RegisterStatusHandler(r, cfg.Logger, cfg.Version)

	cfg.Logger.Info("All builtin handlers registered", "commands", r.ListCommands())
}

// ============================================================================
// 命令类型常量导出（方便 cmd/client 使用）
// ============================================================================

// 重新导出命令类型常量，方便使用
const (
	CmdExecShell  = command.CmdExecShell
	CmdGetStatus  = command.CmdGetStatus
	CmdSystemInfo = command.CmdSystemInfo
	CmdFileRead   = command.CmdFileRead
	CmdFileWrite  = command.CmdFileWrite
	CmdPing       = command.CmdPing
	CmdEcho       = command.CmdEcho
)
