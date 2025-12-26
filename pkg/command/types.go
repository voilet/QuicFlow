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
	CmdNetworkPing  = "network.ping"  // Ping 测试
	CmdNetworkTrace = "network.trace" // 路由追踪

	// 通用
	CmdPing = "ping" // 简单存活检测
	CmdEcho = "echo" // 回显测试
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
	Success      bool                   `json:"success"`       // 整体是否成功（所有命令都发送成功）
	Total        int                    `json:"total"`         // 总客户端数
	SuccessCount int                    `json:"success_count"` // 成功发送的数量
	FailedCount  int                    `json:"failed_count"`  // 发送失败的数量
	Results      []*ClientCommandResult `json:"results"`       // 各客户端的结果
	Message      string                 `json:"message"`       // 摘要信息
}
