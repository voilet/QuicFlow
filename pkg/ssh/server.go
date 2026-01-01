package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/creack/pty"
	"github.com/quic-go/quic-go"
	"golang.org/x/crypto/ssh"
)

// Server SSH 服务器
// 运行在 QUIC 客户端（内网侧），接收来自公网的 SSH 连接请求
type Server struct {
	config    *ServerConfig
	sshConfig *ssh.ServerConfig

	// 状态
	running atomic.Bool

	// 会话管理
	sessions   map[string]*ServerSession
	sessionsMu sync.RWMutex

	// 日志
	logger Logger

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ServerSession SSH 服务器会话
type ServerSession struct {
	ID         string
	RemoteAddr net.Addr
	User       string
	Conn       *ssh.ServerConn
	Channels   <-chan ssh.NewChannel
	Requests   <-chan *ssh.Request
}

// ptyRequestPayload PTY 请求数据结构
type ptyRequestPayload struct {
	Term   string
	Cols   uint32
	Rows   uint32
	Width  uint32
	Height uint32
	Modes  string
}

// channelSession 通道会话（用于存储 PTY 状态）
type channelSession struct {
	ptyReq *ptyRequestPayload
	ptmx   *os.File // PTY master 文件描述符
	cmd    *exec.Cmd
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// defaultLogger 默认日志实现
type defaultLogger struct{}

func (l *defaultLogger) Debug(msg string, args ...any) {}
func (l *defaultLogger) Info(msg string, args ...any)  {}
func (l *defaultLogger) Warn(msg string, args ...any)  {}
func (l *defaultLogger) Error(msg string, args ...any) {}

// NewServer 创建新的 SSH 服务器
func NewServer(config *ServerConfig) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	sshConfig, err := config.BuildSSHConfig()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:    config,
		sshConfig: sshConfig,
		sessions:  make(map[string]*ServerSession),
		logger:    &defaultLogger{},
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// SetLogger 设置日志记录器
func (s *Server) SetLogger(logger Logger) {
	s.logger = logger
}

// HandleStream 处理 SSH 流
// 当 QUIC 客户端接收到 SSH 类型的流时调用此方法
func (s *Server) HandleStream(stream *quic.Stream, conn *quic.Conn) error {
	isRunning := s.running.Load()
	s.logger.Info("HandleStream called", "stream_id", stream.StreamID(), "running", isRunning)

	if !isRunning {
		s.logger.Error("SSH server not running, rejecting stream", "stream_id", stream.StreamID())
		return ErrServerNotRunning
	}

	// 创建 StreamConn 适配器
	streamConn := NewStreamConn(stream, conn)

	// 创建会话 ID
	sessionID := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), stream.StreamID())

	s.logger.Info("New SSH connection", "session_id", sessionID, "remote_addr", conn.RemoteAddr().String())

	// 启动 SSH 握手
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.handleSSHConnection(sessionID, streamConn); err != nil {
			s.logger.Error("SSH connection failed", "session_id", sessionID, "error", err)
		}
	}()

	return nil
}

// handleSSHConnection 处理单个 SSH 连接
func (s *Server) handleSSHConnection(sessionID string, conn net.Conn) error {
	defer conn.Close()

	s.logger.Debug("Starting SSH server handshake", "session_id", sessionID)

	// SSH 握手
	sshConn, channels, reqs, err := ssh.NewServerConn(conn, s.sshConfig)
	if err != nil {
		s.logger.Error("SSH server handshake failed", "session_id", sessionID, "error", err)
		return fmt.Errorf("SSH handshake failed: %w", err)
	}
	s.logger.Debug("SSH server handshake completed", "session_id", sessionID)
	defer sshConn.Close()

	s.logger.Info("SSH connection established",
		"session_id", sessionID,
		"user", sshConn.User(),
		"remote_addr", sshConn.RemoteAddr().String(),
	)

	// 创建并存储会话
	session := &ServerSession{
		ID:         sessionID,
		RemoteAddr: sshConn.RemoteAddr(),
		User:       sshConn.User(),
		Conn:       sshConn,
		Channels:   channels,
		Requests:   reqs,
	}

	s.sessionsMu.Lock()
	s.sessions[sessionID] = session
	s.sessionsMu.Unlock()

	defer func() {
		s.sessionsMu.Lock()
		delete(s.sessions, sessionID)
		s.sessionsMu.Unlock()
		s.logger.Info("SSH session closed", "session_id", sessionID)
	}()

	// 处理全局请求
	go s.handleGlobalRequests(session)

	// 处理通道
	for newChannel := range channels {
		s.wg.Add(1)
		go func(ch ssh.NewChannel) {
			defer s.wg.Done()
			s.handleChannel(session, ch)
		}(newChannel)
	}

	return nil
}

// handleGlobalRequests 处理全局请求
func (s *Server) handleGlobalRequests(session *ServerSession) {
	for req := range session.Requests {
		switch req.Type {
		case "keepalive@openssh.com":
			// 心跳请求
			if req.WantReply {
				req.Reply(true, nil)
			}
		case "tcpip-forward":
			// TCP 端口转发请求
			if s.config.AllowTcpForwarding {
				// TODO: 实现端口转发
				req.Reply(false, nil)
			} else {
				req.Reply(false, nil)
			}
		case "cancel-tcpip-forward":
			// 取消端口转发
			req.Reply(false, nil)
		default:
			s.logger.Debug("Unknown global request", "type", req.Type)
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// handleChannel 处理通道请求
func (s *Server) handleChannel(session *ServerSession, newChannel ssh.NewChannel) {
	channelType := newChannel.ChannelType()

	switch channelType {
	case "session":
		s.handleSessionChannel(session, newChannel)
	case "direct-tcpip":
		if s.config.AllowTcpForwarding {
			s.handleDirectTCPIP(session, newChannel)
		} else {
			newChannel.Reject(ssh.Prohibited, "TCP forwarding disabled")
		}
	default:
		s.logger.Warn("Unknown channel type", "type", channelType)
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", channelType))
	}
}

// handleSessionChannel 处理 session 通道（Shell、Exec）
func (s *Server) handleSessionChannel(session *ServerSession, newChannel ssh.NewChannel) {
	channel, requests, err := newChannel.Accept()
	if err != nil {
		s.logger.Error("Failed to accept channel", "error", err)
		return
	}

	// 创建通道会话来存储 PTY 状态
	chSession := &channelSession{}

	// 处理通道请求
	for req := range requests {
		switch req.Type {
		case "shell":
			s.handleShellRequest(session, channel, req, chSession)
			// Shell 请求处理完后返回
			return
		case "exec":
			s.handleExecRequest(session, channel, req)
			// Exec 请求处理完后返回
			return
		case "pty-req":
			s.handlePtyRequest(session, channel, req, chSession)
		case "env":
			// 环境变量设置
			if req.WantReply {
				req.Reply(true, nil)
			}
		case "window-change":
			// 窗口大小变化
			s.handleWindowChange(chSession, req)
		default:
			s.logger.Debug("Unknown channel request", "type", req.Type)
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// handlePtyRequest 处理 PTY 请求
func (s *Server) handlePtyRequest(session *ServerSession, channel ssh.Channel, req *ssh.Request, chSession *channelSession) {
	if !s.config.AllowPty {
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	// 解析 PTY 请求
	ptyReq := &ptyRequestPayload{}
	if err := ssh.Unmarshal(req.Payload, ptyReq); err != nil {
		s.logger.Error("Failed to parse pty request", "error", err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	s.logger.Debug("PTY request",
		"term", ptyReq.Term,
		"cols", ptyReq.Cols,
		"rows", ptyReq.Rows,
	)

	// 保存 PTY 配置到通道会话
	chSession.ptyReq = ptyReq

	if req.WantReply {
		req.Reply(true, nil)
	}
}

// handleWindowChange 处理窗口大小变化
func (s *Server) handleWindowChange(chSession *channelSession, req *ssh.Request) {
	// 解析窗口变化请求
	var winChange struct {
		Cols   uint32
		Rows   uint32
		Width  uint32
		Height uint32
	}
	if err := ssh.Unmarshal(req.Payload, &winChange); err != nil {
		s.logger.Error("Failed to parse window-change request", "error", err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	// 如果有活动的 PTY，调整大小
	if chSession.ptmx != nil {
		if err := pty.Setsize(chSession.ptmx, &pty.Winsize{
			Rows: uint16(winChange.Rows),
			Cols: uint16(winChange.Cols),
		}); err != nil {
			s.logger.Error("Failed to resize PTY", "error", err)
		} else {
			s.logger.Debug("PTY resized", "cols", winChange.Cols, "rows", winChange.Rows)
		}
	}

	if req.WantReply {
		req.Reply(true, nil)
	}
}

// handleShellRequest 处理 Shell 请求
func (s *Server) handleShellRequest(session *ServerSession, channel ssh.Channel, req *ssh.Request, chSession *channelSession) {
	if req.WantReply {
		req.Reply(true, nil)
	}

	shell := s.config.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.CommandContext(s.ctx, shell)

	// 设置环境变量
	term := "xterm-256color"
	if chSession.ptyReq != nil && chSession.ptyReq.Term != "" {
		term = chSession.ptyReq.Term
	}

	cmd.Env = []string{
		fmt.Sprintf("TERM=%s", term),
		fmt.Sprintf("USER=%s", session.User),
		fmt.Sprintf("HOME=/home/%s", session.User),
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"LANG=en_US.UTF-8",
	}

	// 如果有 PTY 请求，使用真正的 PTY
	if chSession.ptyReq != nil && s.config.AllowPty {
		s.handleShellWithPTY(session, channel, cmd, chSession)
	} else {
		s.handleShellWithPipes(session, channel, cmd)
	}
}

// handleShellWithPTY 使用 PTY 处理 Shell
func (s *Server) handleShellWithPTY(session *ServerSession, channel ssh.Channel, cmd *exec.Cmd, chSession *channelSession) {
	// 启动命令并附加 PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		s.logger.Error("Failed to start shell with PTY", "error", err)
		channel.Close()
		return
	}
	defer ptmx.Close()

	// 保存 PTY 到会话以便后续 resize
	chSession.ptmx = ptmx
	chSession.cmd = cmd

	// 设置初始窗口大小
	if chSession.ptyReq != nil {
		pty.Setsize(ptmx, &pty.Winsize{
			Rows: uint16(chSession.ptyReq.Rows),
			Cols: uint16(chSession.ptyReq.Cols),
		})
	}

	s.logger.Info("Shell started with PTY",
		"user", session.User,
		"term", chSession.ptyReq.Term,
		"size", fmt.Sprintf("%dx%d", chSession.ptyReq.Cols, chSession.ptyReq.Rows),
	)

	// 双向复制数据
	done := make(chan struct{})

	// PTY -> SSH channel (stdout)
	go func() {
		io.Copy(channel, ptmx)
		close(done)
	}()

	// SSH channel -> PTY (stdin)
	go func() {
		io.Copy(ptmx, channel)
	}()

	// 等待命令完成或连接断开
	<-done
	cmd.Wait()

	// 发送退出状态
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Code uint32 }{uint32(exitCode)}))
	channel.Close()
}

// handleShellWithPipes 使用管道处理 Shell（无 PTY）
func (s *Server) handleShellWithPipes(session *ServerSession, channel ssh.Channel, cmd *exec.Cmd) {

	// 连接 stdin/stdout/stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.logger.Error("Failed to get stdin pipe", "error", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.logger.Error("Failed to get stdout pipe", "error", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.logger.Error("Failed to get stderr pipe", "error", err)
		return
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		s.logger.Error("Failed to start shell", "error", err)
		return
	}

	// 复制数据
	var wg sync.WaitGroup

	// stdin
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(stdin, channel)
		stdin.Close()
	}()

	// stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(channel, stdout)
	}()

	// stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(channel.Stderr(), stderr)
	}()

	// 等待命令完成
	cmd.Wait()
	wg.Wait()

	// 发送退出状态
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Code uint32 }{uint32(exitCode)}))
}

// handleExecRequest 处理 Exec 请求（执行单个命令）
func (s *Server) handleExecRequest(session *ServerSession, channel ssh.Channel, req *ssh.Request) {
	// 解析命令
	var payload struct {
		Command string
	}
	if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
		s.logger.Error("Failed to parse exec payload", "error", err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		channel.Close()
		return
	}

	// 先回复请求
	if req.WantReply {
		req.Reply(true, nil)
	}

	s.logger.Debug("Exec request", "command", payload.Command)

	shell := s.config.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.CommandContext(s.ctx, shell, "-c", payload.Command)
	cmd.Env = []string{
		"TERM=xterm-256color",
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		fmt.Sprintf("USER=%s", session.User),
	}

	// 直接执行命令并获取输出
	output, err := cmd.CombinedOutput()

	// 写入输出
	if len(output) > 0 {
		channel.Write(output)
	}

	// 发送退出状态
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	channel.SendRequest("exit-status", false, ssh.Marshal(struct{ Code uint32 }{uint32(exitCode)}))
	channel.Close()
}

// handleDirectTCPIP 处理直接 TCP/IP 通道（端口转发）
func (s *Server) handleDirectTCPIP(session *ServerSession, newChannel ssh.NewChannel) {
	// 解析目标地址
	var payload struct {
		DestHost   string
		DestPort   uint32
		OriginHost string
		OriginPort uint32
	}
	if err := ssh.Unmarshal(newChannel.ExtraData(), &payload); err != nil {
		s.logger.Error("Failed to parse direct-tcpip payload", "error", err)
		newChannel.Reject(ssh.ConnectionFailed, "failed to parse payload")
		return
	}

	s.logger.Debug("Direct TCP/IP request",
		"dest", fmt.Sprintf("%s:%d", payload.DestHost, payload.DestPort),
		"origin", fmt.Sprintf("%s:%d", payload.OriginHost, payload.OriginPort),
	)

	// 连接到目标
	dest := fmt.Sprintf("%s:%d", payload.DestHost, payload.DestPort)
	conn, err := net.Dial("tcp", dest)
	if err != nil {
		s.logger.Error("Failed to connect to destination", "dest", dest, "error", err)
		newChannel.Reject(ssh.ConnectionFailed, err.Error())
		return
	}

	// 接受通道
	channel, requests, err := newChannel.Accept()
	if err != nil {
		s.logger.Error("Failed to accept channel", "error", err)
		conn.Close()
		return
	}

	// 忽略通道请求
	go ssh.DiscardRequests(requests)

	// 双向复制数据
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(conn, channel)
		conn.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(channel, conn)
		channel.Close()
	}()

	wg.Wait()
}

// Start 启动 SSH 服务器
func (s *Server) Start() error {
	if s.running.Swap(true) {
		return ErrServerAlreadyRunning
	}

	s.logger.Info("SSH server started")
	return nil
}

// Stop 停止 SSH 服务器
func (s *Server) Stop() error {
	wasRunning := s.running.Swap(false)
	s.logger.Info("SSH server Stop called", "was_running", wasRunning)

	if !wasRunning {
		return ErrServerNotRunning
	}

	s.cancel()

	// 关闭所有会话
	s.sessionsMu.Lock()
	for _, session := range s.sessions {
		session.Conn.Close()
	}
	s.sessionsMu.Unlock()

	// 等待所有 goroutine 完成
	s.wg.Wait()

	s.logger.Info("SSH server stopped")
	return nil
}

// ListSessions 列出所有活动会话
func (s *Server) ListSessions() []*ServerSession {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	sessions := make([]*ServerSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetSession 获取指定会话
func (s *Server) GetSession(sessionID string) (*ServerSession, error) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// CloseSession 关闭指定会话
func (s *Server) CloseSession(sessionID string) error {
	s.sessionsMu.Lock()
	defer s.sessionsMu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	session.Conn.Close()
	delete(s.sessions, sessionID)
	return nil
}

// IsRunning 检查服务器是否正在运行
func (s *Server) IsRunning() bool {
	return s.running.Load()
}
