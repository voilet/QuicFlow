package ssh

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/quic-go/quic-go"
)

// Manager SSH 连接管理器
// 负责管理 SSH 服务器和客户端实例，处理 SSH 流的识别和路由
type Manager struct {
	// SSH 服务器（在 QUIC 客户端侧运行）
	server *Server

	// SSH 客户端连接（在 QUIC 服务端侧管理）
	clients   map[string]*Client
	clientsMu sync.RWMutex

	// 配置
	serverConfig *ServerConfig
	clientConfig *ClientConfig

	// 状态
	running atomic.Bool

	// 日志
	logger Logger

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager 创建新的 SSH 管理器
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		clients: make(map[string]*Client),
		logger:  &defaultLogger{},
		ctx:     ctx,
		cancel:  cancel,
	}
}

// SetLogger 设置日志记录器
func (m *Manager) SetLogger(logger Logger) {
	m.logger = logger
	if m.server != nil {
		m.server.SetLogger(logger)
	}
}

// InitServer 初始化 SSH 服务器（用于 QUIC 客户端侧）
func (m *Manager) InitServer(config *ServerConfig) error {
	if config == nil {
		config = DefaultServerConfig()
	}

	server, err := NewServer(config)
	if err != nil {
		return fmt.Errorf("failed to create SSH server: %w", err)
	}

	server.SetLogger(m.logger)
	m.server = server
	m.serverConfig = config

	return nil
}

// StartServer 启动 SSH 服务器
func (m *Manager) StartServer() error {
	if m.server == nil {
		return fmt.Errorf("SSH server not initialized")
	}
	return m.server.Start()
}

// StopServer 停止 SSH 服务器
func (m *Manager) StopServer() error {
	if m.server == nil {
		return nil
	}
	return m.server.Stop()
}

// HandleSSHStream 处理 SSH 类型的流（服务器端）
// 当接收到 SSH 流时，将其交给 SSH 服务器处理
func (m *Manager) HandleSSHStream(stream *quic.Stream, conn *quic.Conn) error {
	if m.server == nil {
		return fmt.Errorf("SSH server not initialized")
	}
	return m.server.HandleStream(stream, conn)
}

// CreateClient 创建 SSH 客户端连接（用于 QUIC 服务端侧）
// clientID: 目标 QUIC 客户端 ID
// stream: 已打开的 QUIC 流
// conn: QUIC 连接
// config: SSH 客户端配置
func (m *Manager) CreateClient(clientID string, stream *quic.Stream, conn *quic.Conn, config *ClientConfig) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}

	client.SetLogger(m.logger)

	// 连接
	if err := client.Connect(stream, conn); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// 存储客户端
	m.clientsMu.Lock()
	m.clients[clientID] = client
	m.clientsMu.Unlock()

	m.logger.Info("SSH client created", "client_id", clientID)

	return client, nil
}

// GetClient 获取指定的 SSH 客户端
func (m *Manager) GetClient(clientID string) (*Client, error) {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	client, ok := m.clients[clientID]
	if !ok {
		return nil, fmt.Errorf("SSH client not found: %s", clientID)
	}

	return client, nil
}

// RemoveClient 移除并关闭 SSH 客户端
func (m *Manager) RemoveClient(clientID string) error {
	m.clientsMu.Lock()
	client, ok := m.clients[clientID]
	if ok {
		delete(m.clients, clientID)
	}
	m.clientsMu.Unlock()

	if !ok {
		return fmt.Errorf("SSH client not found: %s", clientID)
	}

	return client.Close()
}

// ListClients 列出所有 SSH 客户端 ID
func (m *Manager) ListClients() []string {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	ids := make([]string, 0, len(m.clients))
	for id := range m.clients {
		ids = append(ids, id)
	}
	return ids
}

// RunCommand 在指定客户端上运行命令
func (m *Manager) RunCommand(clientID, command string) ([]byte, error) {
	client, err := m.GetClient(clientID)
	if err != nil {
		return nil, err
	}

	return client.RunCommand(command)
}

// Start 启动管理器
func (m *Manager) Start() error {
	if m.running.Swap(true) {
		return fmt.Errorf("manager already running")
	}

	m.logger.Info("SSH manager started")
	return nil
}

// Stop 停止管理器
func (m *Manager) Stop() error {
	if !m.running.Swap(false) {
		return nil
	}

	m.cancel()

	// 停止服务器
	if m.server != nil {
		m.server.Stop()
	}

	// 关闭所有客户端
	m.clientsMu.Lock()
	for _, client := range m.clients {
		client.Close()
	}
	m.clients = make(map[string]*Client)
	m.clientsMu.Unlock()

	m.wg.Wait()

	m.logger.Info("SSH manager stopped")
	return nil
}

// GetServer 获取 SSH 服务器实例
func (m *Manager) GetServer() *Server {
	return m.server
}

// IsServerRunning 检查 SSH 服务器是否正在运行
func (m *Manager) IsServerRunning() bool {
	if m.server == nil {
		return false
	}
	return m.server.IsRunning()
}

// StreamHandler 流处理器接口
// 用于与现有的流处理逻辑集成
type StreamHandler interface {
	// HandleStream 处理流，返回 true 表示已处理，false 表示未处理
	HandleStream(stream *quic.Stream, conn *quic.Conn) (bool, error)
}

// Ensure Manager implements StreamHandler
var _ StreamHandler = (*Manager)(nil)

// HandleStream 实现 StreamHandler 接口
// 尝试将流作为 SSH 流处理
// 注意：此方法假设流头部已被读取，直接作为 SSH 流处理
func (m *Manager) HandleStream(stream *quic.Stream, conn *quic.Conn) (bool, error) {
	// 直接交给 SSH 服务器处理
	if err := m.HandleSSHStream(stream, conn); err != nil {
		return true, err
	}

	return true, nil
}
