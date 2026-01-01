package ssh

import "errors"

// 错误定义
var (
	// ErrInvalidMagic 无效的魔数（不是 SSH 流）
	ErrInvalidMagic = errors.New("invalid magic number: not a SSH-over-QUIC stream")

	// ErrUnsupportedVersion 不支持的协议版本
	ErrUnsupportedVersion = errors.New("unsupported protocol version")

	// ErrServerNotRunning SSH 服务器未运行
	ErrServerNotRunning = errors.New("SSH server is not running")

	// ErrServerAlreadyRunning SSH 服务器已在运行
	ErrServerAlreadyRunning = errors.New("SSH server is already running")

	// ErrClientNotConnected SSH 客户端未连接
	ErrClientNotConnected = errors.New("SSH client is not connected")

	// ErrAuthenticationFailed 认证失败
	ErrAuthenticationFailed = errors.New("SSH authentication failed")

	// ErrSessionNotFound 会话未找到
	ErrSessionNotFound = errors.New("SSH session not found")

	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("invalid SSH configuration")

	// ErrHostKeyNotSet 主机密钥未设置
	ErrHostKeyNotSet = errors.New("host key not set")

	// ErrNoAuthMethods 没有配置认证方法
	ErrNoAuthMethods = errors.New("no authentication methods configured")

	// ErrConnectionClosed 连接已关闭
	ErrConnectionClosed = errors.New("connection closed")

	// ErrTimeout 操作超时
	ErrTimeout = errors.New("operation timed out")

	// ErrPortForwardFailed 端口转发失败
	ErrPortForwardFailed = errors.New("port forwarding failed")
)
