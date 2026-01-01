package main

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"golang.org/x/crypto/ssh"

	"github.com/voilet/quic-flow/pkg/api"
	"github.com/voilet/quic-flow/pkg/monitoring"
	quicssh "github.com/voilet/quic-flow/pkg/ssh"
	"github.com/voilet/quic-flow/pkg/transport/server"
)

// SSHClientManager SSH 客户端管理器
// 在 QUIC 服务器侧管理到各个客户端的 SSH 连接
type SSHClientManager struct {
	server     *server.Server
	logger     *monitoring.Logger
	config     *SSHClientConfig
	mu         sync.RWMutex
	sshClients map[string]*ManagedSSHClient // clientID -> SSH client
}

// SSHClientConfig SSH 客户端配置
type SSHClientConfig struct {
	// DefaultUser 默认 SSH 用户名
	DefaultUser string

	// DefaultPassword 默认 SSH 密码
	DefaultPassword string

	// ConnectionTimeout 连接超时
	ConnectionTimeout time.Duration

	// CommandTimeout 命令执行超时
	CommandTimeout time.Duration
}

// ManagedSSHClient 被管理的 SSH 客户端
type ManagedSSHClient struct {
	ClientID    string
	SSHConn     *ssh.Client
	Stream      *quic.Stream
	ConnectedAt time.Time
	mu          sync.Mutex
}

// DefaultSSHClientConfig 默认 SSH 客户端配置
func DefaultSSHClientConfig() *SSHClientConfig {
	return &SSHClientConfig{
		DefaultUser:       "admin",
		DefaultPassword:   "admin123",
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    60 * time.Second,
	}
}

// NewSSHClientManager 创建 SSH 客户端管理器
func NewSSHClientManager(srv *server.Server, config *SSHClientConfig, logger *monitoring.Logger) *SSHClientManager {
	if config == nil {
		config = DefaultSSHClientConfig()
	}
	return &SSHClientManager{
		server:     srv,
		logger:     logger,
		config:     config,
		sshClients: make(map[string]*ManagedSSHClient),
	}
}

// Connect 连接到指定客户端的 SSH 服务
func (m *SSHClientManager) Connect(clientID string, user, password string) error {
	// 检查是否已存在连接
	m.mu.RLock()
	if _, exists := m.sshClients[clientID]; exists {
		m.mu.RUnlock()
		return fmt.Errorf("SSH connection to client %s already exists", clientID)
	}
	m.mu.RUnlock()

	// 获取客户端的 QUIC 连接
	quicConn := m.server.GetClientConnection(clientID)
	if quicConn == nil {
		return fmt.Errorf("client %s not found or not connected", clientID)
	}

	// 使用默认凭据
	if user == "" {
		user = m.config.DefaultUser
	}
	if password == "" {
		password = m.config.DefaultPassword
	}

	// 打开一个新的 QUIC 流用于 SSH
	ctx, cancel := context.WithTimeout(context.Background(), m.config.ConnectionTimeout)
	defer cancel()

	stream, err := quicConn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("failed to open QUIC stream: %w", err)
	}

	// 发送 SSH 流标识
	if err := quicssh.WriteHeader(stream, quicssh.StreamTypeSSH); err != nil {
		stream.Close()
		return fmt.Errorf("failed to write SSH header: %w", err)
	}

	// 创建 StreamConn 适配器
	streamConn := quicssh.NewStreamConn(stream, quicConn)

	// SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         m.config.ConnectionTimeout,
	}

	// SSH 握手
	sshConn, chans, reqs, err := ssh.NewClientConn(streamConn, quicConn.RemoteAddr().String(), sshConfig)
	if err != nil {
		stream.Close()
		return fmt.Errorf("SSH handshake failed: %w", err)
	}

	// 创建 SSH 客户端
	client := ssh.NewClient(sshConn, chans, reqs)

	// 保存连接
	managedClient := &ManagedSSHClient{
		ClientID:    clientID,
		SSHConn:     client,
		Stream:      stream,
		ConnectedAt: time.Now(),
	}

	m.mu.Lock()
	m.sshClients[clientID] = managedClient
	m.mu.Unlock()

	m.logger.Info("SSH connection established", "client_id", clientID, "user", user)
	return nil
}

// Disconnect 断开到指定客户端的 SSH 连接
func (m *SSHClientManager) Disconnect(clientID string) error {
	m.mu.Lock()
	client, exists := m.sshClients[clientID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("SSH connection to client %s not found", clientID)
	}
	delete(m.sshClients, clientID)
	m.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	if client.SSHConn != nil {
		client.SSHConn.Close()
	}
	if client.Stream != nil {
		client.Stream.Close()
	}

	m.logger.Info("SSH connection closed", "client_id", clientID)
	return nil
}

// ExecuteCommand 在指定客户端上执行命令
func (m *SSHClientManager) ExecuteCommand(clientID, command string) (string, error) {
	m.mu.RLock()
	client, exists := m.sshClients[clientID]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("SSH connection to client %s not found, please connect first", clientID)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	// 创建 SSH 会话
	session, err := client.SSHConn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// 执行命令
	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

// ExecuteCommandOneShot 一次性连接并执行命令（不保持连接）
func (m *SSHClientManager) ExecuteCommandOneShot(clientID, user, password, command string) (string, error) {
	// 获取客户端的 QUIC 连接
	quicConn := m.server.GetClientConnection(clientID)
	if quicConn == nil {
		return "", fmt.Errorf("client %s not found or not connected", clientID)
	}

	// 使用默认凭据
	if user == "" {
		user = m.config.DefaultUser
	}
	if password == "" {
		password = m.config.DefaultPassword
	}

	// 打开一个新的 QUIC 流用于 SSH
	ctx, cancel := context.WithTimeout(context.Background(), m.config.ConnectionTimeout)
	defer cancel()

	stream, err := quicConn.OpenStreamSync(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open QUIC stream: %w", err)
	}
	defer stream.Close()

	// 发送 SSH 流标识
	if err := quicssh.WriteHeader(stream, quicssh.StreamTypeSSH); err != nil {
		return "", fmt.Errorf("failed to write SSH header: %w", err)
	}

	// 创建 StreamConn 适配器
	streamConn := quicssh.NewStreamConn(stream, quicConn)

	// SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         m.config.ConnectionTimeout,
	}

	// SSH 握手
	sshConn, chans, reqs, err := ssh.NewClientConn(streamConn, quicConn.RemoteAddr().String(), sshConfig)
	if err != nil {
		return "", fmt.Errorf("SSH handshake failed: %w", err)
	}
	defer sshConn.Close()

	// 创建 SSH 客户端
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	// 创建 SSH 会话
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// 执行命令
	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

// StartShell 在指定客户端上启动交互式 Shell
func (m *SSHClientManager) StartShell(clientID string, stdin io.Reader, stdout, stderr io.Writer) error {
	m.mu.RLock()
	client, exists := m.sshClients[clientID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("SSH connection to client %s not found, please connect first", clientID)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	// 创建 SSH 会话
	session, err := client.SSHConn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// 请求 PTY
	if err := session.RequestPty("xterm-256color", 80, 40, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		return fmt.Errorf("failed to request pty: %w", err)
	}

	// 设置 IO
	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	// 启动 Shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// 等待 Shell 结束
	return session.Wait()
}

// ListConnections 列出所有 SSH 连接
func (m *SSHClientManager) ListConnections() []SSHConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var connections []SSHConnectionInfo
	for clientID, client := range m.sshClients {
		connections = append(connections, SSHConnectionInfo{
			ClientID:    clientID,
			ConnectedAt: client.ConnectedAt,
			Uptime:      time.Since(client.ConnectedAt),
		})
	}
	return connections
}

// IsConnected 检查是否已建立 SSH 连接
func (m *SSHClientManager) IsConnected(clientID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.sshClients[clientID]
	return exists
}

// GetConnection 获取指定客户端的 SSH 连接信息
func (m *SSHClientManager) GetConnection(clientID string) (*SSHConnectionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.sshClients[clientID]
	if !exists {
		return nil, fmt.Errorf("SSH connection to client %s not found", clientID)
	}

	return &SSHConnectionInfo{
		ClientID:    clientID,
		ConnectedAt: client.ConnectedAt,
		Uptime:      time.Since(client.ConnectedAt),
	}, nil
}

// Close 关闭所有 SSH 连接
func (m *SSHClientManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for clientID, client := range m.sshClients {
		client.mu.Lock()
		if client.SSHConn != nil {
			client.SSHConn.Close()
		}
		if client.Stream != nil {
			client.Stream.Close()
		}
		client.mu.Unlock()
		m.logger.Info("SSH connection closed", "client_id", clientID)
	}
	m.sshClients = make(map[string]*ManagedSSHClient)
}

// SSHConnectionInfo SSH 连接信息
type SSHConnectionInfo struct {
	ClientID    string        `json:"client_id"`
	ConnectedAt time.Time     `json:"connected_at"`
	Uptime      time.Duration `json:"uptime"`
}

// ToAPIFormat 转换为 API 格式
func (info SSHConnectionInfo) ToAPIFormat() map[string]interface{} {
	return map[string]interface{}{
		"client_id":    info.ClientID,
		"connected_at": info.ConnectedAt.Format(time.RFC3339),
		"uptime":       info.Uptime.String(),
	}
}

// SSHClientManagerAPIAdapter 适配器，用于将 SSHClientManager 适配到 API 接口
type SSHClientManagerAPIAdapter struct {
	manager *SSHClientManager
}

// NewSSHClientManagerAPIAdapter 创建 API 适配器
func NewSSHClientManagerAPIAdapter(manager *SSHClientManager) *SSHClientManagerAPIAdapter {
	return &SSHClientManagerAPIAdapter{manager: manager}
}

// Connect 连接到指定客户端的 SSH 服务
func (a *SSHClientManagerAPIAdapter) Connect(clientID string, user, password string) error {
	return a.manager.Connect(clientID, user, password)
}

// Disconnect 断开到指定客户端的 SSH 连接
func (a *SSHClientManagerAPIAdapter) Disconnect(clientID string) error {
	return a.manager.Disconnect(clientID)
}

// ExecuteCommand 在指定客户端上执行命令
func (a *SSHClientManagerAPIAdapter) ExecuteCommand(clientID, command string) (string, error) {
	return a.manager.ExecuteCommand(clientID, command)
}

// ExecuteCommandOneShot 一次性连接并执行命令（不保持连接）
func (a *SSHClientManagerAPIAdapter) ExecuteCommandOneShot(clientID, user, password, command string) (string, error) {
	return a.manager.ExecuteCommandOneShot(clientID, user, password, command)
}

// GetConnectionInfos 获取所有 SSH 连接信息
func (a *SSHClientManagerAPIAdapter) GetConnectionInfos() []api.ConnectionInfoData {
	connections := a.manager.ListConnections()
	result := make([]api.ConnectionInfoData, len(connections))
	for i, conn := range connections {
		result[i] = api.ConnectionInfoData{
			ClientID:    conn.ClientID,
			ConnectedAt: conn.ConnectedAt,
			Uptime:      conn.Uptime,
		}
	}
	return result
}

// IsConnected 检查是否已建立 SSH 连接
func (a *SSHClientManagerAPIAdapter) IsConnected(clientID string) bool {
	return a.manager.IsConnected(clientID)
}

// ===== PTY Session 管理（用于 WebSocket 终端） =====

// PTYSession 交互式 PTY 会话
type PTYSession struct {
	ID        string
	ClientID  string
	Session   *ssh.Session
	SSHClient *ssh.Client
	Stream    *quic.Stream
	Stdin     io.WriteCloser
	Stdout    io.Reader
	Cols      int
	Rows      int
	CreatedAt time.Time
	Done      chan struct{}     // 会话结束信号
	CleanedUp chan struct{}     // 清理完成信号
	closed    bool
	mu        sync.Mutex
}

// ptySessions 存储活动的 PTY 会话
var (
	ptySessions   = make(map[string]*PTYSession)
	ptySessionsMu sync.RWMutex
)

// StartPTYSession 启动交互式 PTY 会话
func (m *SSHClientManager) StartPTYSession(clientID string, cols, rows int) (*PTYSession, error) {
	// 获取客户端的 QUIC 连接
	quicConn := m.server.GetClientConnection(clientID)
	if quicConn == nil {
		return nil, fmt.Errorf("client %s not found or not connected", clientID)
	}

	// 打开一个新的 QUIC 流用于 SSH
	ctx, cancel := context.WithTimeout(context.Background(), m.config.ConnectionTimeout)
	defer cancel()

	m.logger.Info("Opening QUIC stream for PTY session", "client_id", clientID, "cols", cols, "rows", rows)
	stream, err := quicConn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open QUIC stream: %w", err)
	}
	m.logger.Info("QUIC stream opened", "client_id", clientID, "stream_id", stream.StreamID())

	// 发送 SSH 流标识
	m.logger.Debug("Writing SSH header to stream", "client_id", clientID, "stream_id", stream.StreamID())
	if err := quicssh.WriteHeader(stream, quicssh.StreamTypeSSH); err != nil {
		stream.Close()
		return nil, fmt.Errorf("failed to write SSH header: %w", err)
	}
	m.logger.Debug("SSH header written successfully", "client_id", clientID)

	// 创建 StreamConn 适配器
	streamConn := quicssh.NewStreamConn(stream, quicConn)

	// SSH 客户端配置
	sshConfig := &ssh.ClientConfig{
		User: m.config.DefaultUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(m.config.DefaultPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         m.config.ConnectionTimeout,
	}

	// SSH 握手
	m.logger.Debug("Starting SSH client handshake", "client_id", clientID, "user", m.config.DefaultUser)
	sshConn, chans, reqs, err := ssh.NewClientConn(streamConn, quicConn.RemoteAddr().String(), sshConfig)
	if err != nil {
		stream.Close()
		m.logger.Error("SSH client handshake failed", "client_id", clientID, "error", err)
		return nil, fmt.Errorf("SSH handshake failed: %w", err)
	}
	m.logger.Debug("SSH client handshake completed", "client_id", clientID)

	// 创建 SSH 客户端
	client := ssh.NewClient(sshConn, chans, reqs)

	// 创建 SSH 会话
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		stream.Close()
		return nil, fmt.Errorf("failed to create SSH session: %w", err)
	}

	// 请求 PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		client.Close()
		stream.Close()
		return nil, fmt.Errorf("failed to request pty: %w", err)
	}

	// 获取 stdin pipe
	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		stream.Close()
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	// 获取 stdout pipe
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		stream.Close()
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// 启动 Shell
	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		stream.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	// 创建 PTY 会话
	sessionID := fmt.Sprintf("pty-%s-%d", clientID, time.Now().UnixNano())
	ptySession := &PTYSession{
		ID:        sessionID,
		ClientID:  clientID,
		Session:   session,
		SSHClient: client,
		Stream:    stream,
		Stdin:     stdin,
		Stdout:    stdout,
		Cols:      cols,
		Rows:      rows,
		CreatedAt: time.Now(),
		Done:      make(chan struct{}),
		CleanedUp: make(chan struct{}),
	}

	// 保存会话
	ptySessionsMu.Lock()
	ptySessions[sessionID] = ptySession
	ptySessionsMu.Unlock()

	m.logger.Info("PTY session started",
		"session_id", sessionID,
		"client_id", clientID,
		"cols", cols,
		"rows", rows,
	)

	// 监控会话结束并清理资源
	go func() {
		defer close(ptySession.CleanedUp) // 标记清理完成

		session.Wait()

		// 标记会话已关闭
		ptySession.mu.Lock()
		if ptySession.closed {
			ptySession.mu.Unlock()
			// 已经被 ClosePTYSession 关闭，但仍然需要清理资源
			return
		}
		ptySession.closed = true
		ptySession.mu.Unlock()

		// 关闭 Done 通道（通知其他等待者）
		select {
		case <-ptySession.Done:
			// 已经关闭
		default:
			close(ptySession.Done)
		}

		// 从 map 中删除
		ptySessionsMu.Lock()
		delete(ptySessions, sessionID)
		ptySessionsMu.Unlock()

		// 按顺序关闭资源
		if session != nil {
			session.Close()
		}
		if client != nil {
			client.Close()
		}
		if stream != nil {
			stream.Close()
		}

		m.logger.Info("PTY session ended", "session_id", sessionID)
	}()

	return ptySession, nil
}

// ResizePTY 调整 PTY 会话窗口大小
func (m *SSHClientManager) ResizePTY(sessionID string, cols, rows int) error {
	ptySessionsMu.RLock()
	ptySession, exists := ptySessions[sessionID]
	ptySessionsMu.RUnlock()

	if !exists {
		return fmt.Errorf("PTY session %s not found", sessionID)
	}

	// 发送窗口变化请求
	if err := ptySession.Session.WindowChange(rows, cols); err != nil {
		return fmt.Errorf("failed to resize PTY: %w", err)
	}

	ptySession.Cols = cols
	ptySession.Rows = rows

	m.logger.Debug("PTY resized", "session_id", sessionID, "cols", cols, "rows", rows)
	return nil
}

// ClosePTYSession 关闭 PTY 会话
func (m *SSHClientManager) ClosePTYSession(sessionID string) error {
	m.logger.Info("ClosePTYSession called", "session_id", sessionID)

	ptySessionsMu.Lock()
	ptySession, exists := ptySessions[sessionID]
	if exists {
		delete(ptySessions, sessionID)
	}
	ptySessionsMu.Unlock()

	if !exists {
		m.logger.Info("PTY session not found (already cleaned up)", "session_id", sessionID)
		return nil
	}

	// 标记会话已关闭
	ptySession.mu.Lock()
	alreadyClosed := ptySession.closed
	ptySession.closed = true
	ptySession.mu.Unlock()

	m.logger.Info("ClosePTYSession marking closed", "session_id", sessionID, "already_closed", alreadyClosed)

	if alreadyClosed {
		// 等待清理完成
		m.logger.Info("Waiting for cleanup (already closed)", "session_id", sessionID)
		<-ptySession.CleanedUp
		return nil
	}

	// 关闭 Done 通道（通知其他等待者）
	select {
	case <-ptySession.Done:
		// 已经关闭
	default:
		close(ptySession.Done)
	}

	// 关闭 SSH session 触发清理
	m.logger.Info("Closing SSH session", "session_id", sessionID)
	if ptySession.Session != nil {
		ptySession.Session.Close()
	}

	// 等待清理 goroutine 完成（最多等待 5 秒）
	m.logger.Info("Waiting for cleanup goroutine", "session_id", sessionID)
	select {
	case <-ptySession.CleanedUp:
		m.logger.Info("Cleanup completed", "session_id", sessionID)
	case <-time.After(5 * time.Second):
		m.logger.Warn("PTY session cleanup timeout, forcing cleanup", "session_id", sessionID)
		// 强制清理
		if ptySession.SSHClient != nil {
			ptySession.SSHClient.Close()
		}
		if ptySession.Stream != nil {
			ptySession.Stream.Close()
		}
	}

	m.logger.Info("PTY session closed", "session_id", sessionID)
	return nil
}

// GetPTYSession 获取 PTY 会话
func (m *SSHClientManager) GetPTYSession(sessionID string) (*PTYSession, error) {
	ptySessionsMu.RLock()
	defer ptySessionsMu.RUnlock()

	ptySession, exists := ptySessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("PTY session %s not found", sessionID)
	}
	return ptySession, nil
}

// ListPTYSessions 列出所有 PTY 会话
func (m *SSHClientManager) ListPTYSessions() []*PTYSession {
	ptySessionsMu.RLock()
	defer ptySessionsMu.RUnlock()

	sessions := make([]*PTYSession, 0, len(ptySessions))
	for _, session := range ptySessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// ===== TerminalManagerAPI 接口适配器 =====

// SSHTerminalAdapter 将 SSHClientManager 适配到 TerminalManagerAPI 接口
type SSHTerminalAdapter struct {
	manager *SSHClientManager
}

// NewSSHTerminalAdapter 创建终端适配器
func NewSSHTerminalAdapter(manager *SSHClientManager) *SSHTerminalAdapter {
	return &SSHTerminalAdapter{manager: manager}
}

// StartPTYSession 启动 PTY 会话
func (a *SSHTerminalAdapter) StartPTYSession(clientID string, cols, rows int) (api.PTYSessionInfo, error) {
	session, err := a.manager.StartPTYSession(clientID, cols, rows)
	if err != nil {
		return api.PTYSessionInfo{}, err
	}
	return api.PTYSessionInfo{
		ID:        session.ID,
		ClientID:  session.ClientID,
		Cols:      session.Cols,
		Rows:      session.Rows,
		CreatedAt: session.CreatedAt,
	}, nil
}

// ResizePTY 调整 PTY 大小
func (a *SSHTerminalAdapter) ResizePTY(sessionID string, cols, rows int) error {
	return a.manager.ResizePTY(sessionID, cols, rows)
}

// ClosePTYSession 关闭 PTY 会话
func (a *SSHTerminalAdapter) ClosePTYSession(sessionID string) error {
	return a.manager.ClosePTYSession(sessionID)
}

// GetPTYSessionReader 获取 PTY 会话的输出 Reader
func (a *SSHTerminalAdapter) GetPTYSessionReader(sessionID string) (io.Reader, error) {
	session, err := a.manager.GetPTYSession(sessionID)
	if err != nil {
		return nil, err
	}
	return session.Stdout, nil
}

// GetPTYSessionWriter 获取 PTY 会话的输入 Writer
func (a *SSHTerminalAdapter) GetPTYSessionWriter(sessionID string) (io.WriteCloser, error) {
	session, err := a.manager.GetPTYSession(sessionID)
	if err != nil {
		return nil, err
	}
	return session.Stdin, nil
}

// GetPTYSessionDone 获取 PTY 会话完成通道
func (a *SSHTerminalAdapter) GetPTYSessionDone(sessionID string) (<-chan struct{}, error) {
	session, err := a.manager.GetPTYSession(sessionID)
	if err != nil {
		return nil, err
	}
	return session.Done, nil
}

// ListPTYSessions 列出所有 PTY 会话
func (a *SSHTerminalAdapter) ListPTYSessions() []api.PTYSessionInfo {
	sessions := a.manager.ListPTYSessions()
	result := make([]api.PTYSessionInfo, len(sessions))
	for i, s := range sessions {
		result[i] = api.PTYSessionInfo{
			ID:        s.ID,
			ClientID:  s.ClientID,
			Cols:      s.Cols,
			Rows:      s.Rows,
			CreatedAt: s.CreatedAt,
		}
	}
	return result
}
