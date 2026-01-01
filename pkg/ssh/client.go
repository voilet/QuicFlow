package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/quic-go/quic-go"
	"golang.org/x/crypto/ssh"
)

// Client SSH 客户端
// 运行在 QUIC 服务端（公网侧），通过 QUIC 流连接到内网的 SSH 服务
type Client struct {
	config    *ClientConfig
	sshConfig *ssh.ClientConfig

	// 连接
	conn    *quic.Conn
	sshConn ssh.Conn

	// 状态
	connected atomic.Bool

	// 通道和请求
	channels <-chan ssh.NewChannel
	requests <-chan *ssh.Request

	// 会话管理
	sessions   map[string]*ClientSession
	sessionsMu sync.RWMutex

	// 日志
	logger Logger

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ClientSession SSH 客户端会话
type ClientSession struct {
	ID      string
	Session *ssh.Session
	Stdin   io.WriteCloser
	Stdout  io.Reader
	Stderr  io.Reader
}

// NewClient 创建新的 SSH 客户端
func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	sshConfig, err := config.BuildSSHConfig()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config:    config,
		sshConfig: sshConfig,
		sessions:  make(map[string]*ClientSession),
		logger:    &defaultLogger{},
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// SetLogger 设置日志记录器
func (c *Client) SetLogger(logger Logger) {
	c.logger = logger
}

// Connect 通过 QUIC 连接建立 SSH 连接
// 在调用此方法之前，需要先打开一个 QUIC 流
func (c *Client) Connect(stream *quic.Stream, conn *quic.Conn) error {
	if c.connected.Load() {
		return fmt.Errorf("already connected")
	}

	c.conn = conn

	// 创建 StreamConn 适配器
	streamConn := NewStreamConn(stream, conn)

	// 发送 SSH 流标识
	if err := WriteHeader(streamConn, StreamTypeSSH); err != nil {
		return fmt.Errorf("failed to write SSH header: %w", err)
	}

	// SSH 握手
	sshConn, channels, requests, err := ssh.NewClientConn(streamConn, conn.RemoteAddr().String(), c.sshConfig)
	if err != nil {
		return fmt.Errorf("SSH handshake failed: %w", err)
	}

	c.sshConn = sshConn
	c.channels = channels
	c.requests = requests
	c.connected.Store(true)

	c.logger.Info("SSH connection established",
		"remote_addr", conn.RemoteAddr().String(),
		"user", c.config.User,
	)

	// 处理请求和通道
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.handleRequests()
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.handleChannels()
	}()

	return nil
}

// handleRequests 处理全局请求
func (c *Client) handleRequests() {
	for req := range c.requests {
		switch req.Type {
		case "keepalive@openssh.com":
			if req.WantReply {
				req.Reply(true, nil)
			}
		default:
			c.logger.Debug("Unknown request", "type", req.Type)
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// handleChannels 处理通道请求
func (c *Client) handleChannels() {
	for ch := range c.channels {
		c.logger.Debug("Received channel", "type", ch.ChannelType())
		// 通常客户端不需要处理反向通道
		ch.Reject(ssh.UnknownChannelType, "not supported")
	}
}

// NewSession 创建新的 SSH 会话
func (c *Client) NewSession() (*ssh.Session, error) {
	if !c.connected.Load() {
		return nil, ErrClientNotConnected
	}

	client := ssh.NewClient(c.sshConn, c.channels, c.requests)
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// RunCommand 运行单个命令并返回输出
func (c *Client) RunCommand(command string) ([]byte, error) {
	session, err := c.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}

	return output, nil
}

// StartShell 启动交互式 Shell
func (c *Client) StartShell(sessionConfig *SessionConfig) (*ClientSession, error) {
	if sessionConfig == nil {
		sessionConfig = DefaultSessionConfig()
	}

	session, err := c.NewSession()
	if err != nil {
		return nil, err
	}

	// 请求 PTY
	if err := session.RequestPty(
		sessionConfig.Term,
		sessionConfig.Height,
		sessionConfig.Width,
		sessionConfig.Modes,
	); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to request PTY: %w", err)
	}

	// 设置环境变量
	for key, value := range sessionConfig.Env {
		if err := session.Setenv(key, value); err != nil {
			c.logger.Warn("Failed to set env", "key", key, "error", err)
		}
	}

	// 获取 stdin/stdout/stderr
	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to get stdin: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to get stderr: %w", err)
	}

	// 启动 Shell
	if err := session.Shell(); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	clientSession := &ClientSession{
		ID:      fmt.Sprintf("session-%p", session),
		Session: session,
		Stdin:   stdin,
		Stdout:  stdout,
		Stderr:  stderr,
	}

	c.sessionsMu.Lock()
	c.sessions[clientSession.ID] = clientSession
	c.sessionsMu.Unlock()

	return clientSession, nil
}

// CloseSession 关闭指定会话
func (c *Client) CloseSession(sessionID string) error {
	c.sessionsMu.Lock()
	defer c.sessionsMu.Unlock()

	session, ok := c.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	if err := session.Session.Close(); err != nil {
		return err
	}

	delete(c.sessions, sessionID)
	return nil
}

// Close 关闭 SSH 连接
func (c *Client) Close() error {
	if !c.connected.Swap(false) {
		return nil
	}

	c.cancel()

	// 关闭所有会话
	c.sessionsMu.Lock()
	for _, session := range c.sessions {
		session.Session.Close()
	}
	c.sessions = make(map[string]*ClientSession)
	c.sessionsMu.Unlock()

	// 关闭 SSH 连接
	if c.sshConn != nil {
		c.sshConn.Close()
	}

	c.wg.Wait()

	c.logger.Info("SSH connection closed")
	return nil
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

// LocalPortForward 创建本地端口转发
// 监听本地端口，将连接转发到远程地址
func (c *Client) LocalPortForward(localAddr, remoteAddr string) (net.Listener, error) {
	if !c.connected.Load() {
		return nil, ErrClientNotConnected
	}

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", localAddr, err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.handleLocalForward(listener, remoteAddr)
	}()

	c.logger.Info("Local port forward started", "local", localAddr, "remote", remoteAddr)
	return listener, nil
}

// handleLocalForward 处理本地端口转发
func (c *Client) handleLocalForward(listener net.Listener, remoteAddr string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			default:
				c.logger.Error("Failed to accept connection", "error", err)
				continue
			}
		}

		c.wg.Add(1)
		go func(conn net.Conn) {
			defer c.wg.Done()
			c.forwardConnection(conn, remoteAddr)
		}(conn)
	}
}

// forwardConnection 转发单个连接
func (c *Client) forwardConnection(localConn net.Conn, remoteAddr string) {
	defer localConn.Close()

	host, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		c.logger.Error("Invalid remote address", "addr", remoteAddr, "error", err)
		return
	}

	// 通过 SSH 连接到远程
	client := ssh.NewClient(c.sshConn, c.channels, c.requests)

	// 注意：这里需要使用 DialContext，但标准库不支持
	// 使用 Dial 替代
	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		c.logger.Error("Failed to dial remote", "addr", remoteAddr, "error", err)
		return
	}
	defer remoteConn.Close()

	c.logger.Debug("Forwarding connection", "local", localConn.RemoteAddr(), "remote", fmt.Sprintf("%s:%s", host, port))

	// 双向复制
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(remoteConn, localConn)
	}()

	go func() {
		defer wg.Done()
		io.Copy(localConn, remoteConn)
	}()

	wg.Wait()
}

// Dial 通过 SSH 隧道拨号
// 支持 "tcp" 和 "unix" 网络类型
func (c *Client) Dial(network, addr string) (net.Conn, error) {
	if !c.connected.Load() {
		return nil, ErrClientNotConnected
	}

	client := ssh.NewClient(c.sshConn, c.channels, c.requests)
	return client.Dial(network, addr)
}
