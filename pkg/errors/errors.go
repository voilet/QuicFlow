package errors

import "errors"

// 连接相关错误
var (
	// ErrConnectionClosed 表示连接已关闭
	ErrConnectionClosed = errors.New("connection closed")

	// ErrClientNotConnected 表示客户端未连接或不存在
	ErrClientNotConnected = errors.New("client not connected")

	// ErrInvalidClientID 表示客户端 ID 无效或为空
	ErrInvalidClientID = errors.New("invalid client ID")
)

// 消息相关错误
var (
	// ErrPayloadTooLarge 表示消息负载超过 1MB 限制
	ErrPayloadTooLarge = errors.New("payload size exceeds 1MB limit")

	// ErrInvalidMessage 表示消息格式无效
	ErrInvalidMessage = errors.New("invalid message format")

	// ErrMessageTimeout 表示消息发送或接收超时
	ErrMessageTimeout = errors.New("message operation timeout")
)

// 回调相关错误
var (
	// ErrPromiseCapacityFull 表示 Promise 映射表已满（达到 50,000 上限）
	ErrPromiseCapacityFull = errors.New("promise capacity full (50,000 limit)")

	// ErrPromiseNotFound 表示未找到对应的 Promise 记录
	ErrPromiseNotFound = errors.New("promise not found")

	// ErrTimeout 表示操作超时
	ErrTimeout = errors.New("operation timeout")
)

// Dispatcher 相关错误
var (
	// ErrDispatcherFull 表示 Dispatcher 消息队列已满
	ErrDispatcherFull = errors.New("dispatcher message queue is full")

	// ErrHandlerNotFound 表示未找到对应的消息处理器
	ErrHandlerNotFound = errors.New("message handler not found")

	// ErrHandlerPanic 表示消息处理器发生 panic
	ErrHandlerPanic = errors.New("message handler panicked")
)

// 配置相关错误
var (
	// ErrInvalidConfig 表示配置无效
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrMissingTLSConfig 表示缺少 TLS 配置
	ErrMissingTLSConfig = errors.New("missing TLS configuration")

	// ErrInvalidAddress 表示网络地址无效
	ErrInvalidAddress = errors.New("invalid network address")
)

// 会话相关错误
var (
	// ErrSessionNotFound 表示会话不存在
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionAlreadyExists 表示会话已存在（客户端 ID 重复）
	ErrSessionAlreadyExists = errors.New("session already exists")

	// ErrHeartbeatTimeout 表示心跳超时
	ErrHeartbeatTimeout = errors.New("heartbeat timeout")
)

// 编解码相关错误
var (
	// ErrEncodeFailed 表示编码失败
	ErrEncodeFailed = errors.New("failed to encode message")

	// ErrDecodeFailed 表示解码失败
	ErrDecodeFailed = errors.New("failed to decode message")

	// ErrInvalidFrameType 表示帧类型无效
	ErrInvalidFrameType = errors.New("invalid frame type")
)
