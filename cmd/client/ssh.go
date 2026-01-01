package main

import (
	"context"
	"strings"

	"github.com/quic-go/quic-go"
	"golang.org/x/crypto/ssh"

	"github.com/voilet/quic-flow/pkg/monitoring"
	quicssh "github.com/voilet/quic-flow/pkg/ssh"
)

// SSHIntegration SSH 集成组件
// 在 QUIC 客户端侧运行 SSH 服务器，允许公网服务器通过已建立的 QUIC 连接访问内网机器
type SSHIntegration struct {
	server *quicssh.Server
	logger *monitoring.Logger
	config *SSHConfig
}

// SSHConfig SSH 配置
type SSHConfig struct {
	// Enabled 是否启用 SSH 服务
	Enabled bool

	// User SSH 用户名
	User string

	// Password SSH 密码（简单认证，生产环境建议使用密钥）
	Password string

	// Shell 默认 Shell
	Shell string

	// AllowPortForward 允许端口转发
	AllowPortForward bool
}

// DefaultSSHConfig 默认 SSH 配置
func DefaultSSHConfig() *SSHConfig {
	return &SSHConfig{
		Enabled:          true,
		User:             "admin",
		Password:         "admin123",
		Shell:            "/bin/sh",
		AllowPortForward: true,
	}
}

// NewSSHIntegration 创建 SSH 集成组件
func NewSSHIntegration(config *SSHConfig, logger *monitoring.Logger) (*SSHIntegration, error) {
	if config == nil {
		config = DefaultSSHConfig()
	}

	// 创建 SSH 服务器配置
	serverConfig := quicssh.DefaultServerConfig()
	serverConfig.Shell = config.Shell
	serverConfig.AllowTcpForwarding = config.AllowPortForward
	serverConfig.PasswordAuth = true
	serverConfig.PasswordCallback = func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		if conn.User() == config.User && string(password) == config.Password {
			logger.Info("SSH authentication successful", "user", conn.User())
			return &ssh.Permissions{
				Extensions: map[string]string{
					"user": conn.User(),
				},
			}, nil
		}
		logger.Warn("SSH authentication failed", "user", conn.User())
		return nil, quicssh.ErrAuthenticationFailed
	}

	// 创建 SSH 服务器
	server, err := quicssh.NewServer(serverConfig)
	if err != nil {
		return nil, err
	}

	// 设置日志
	server.SetLogger(&sshLogger{logger: logger})

	return &SSHIntegration{
		server: server,
		logger: logger,
		config: config,
	}, nil
}

// Start 启动 SSH 服务
func (s *SSHIntegration) Start() error {
	if err := s.server.Start(); err != nil {
		return err
	}
	s.logger.Info("SSH server started",
		"user", s.config.User,
		"shell", s.config.Shell,
		"port_forward", s.config.AllowPortForward)
	return nil
}

// Stop 停止 SSH 服务
func (s *SSHIntegration) Stop() error {
	return s.server.Stop()
}

// HandleStream 处理 SSH 流
// 当收到 SSH 类型的流时调用此方法
func (s *SSHIntegration) HandleStream(stream *quic.Stream, conn *quic.Conn) error {
	return s.server.HandleStream(stream, conn)
}

// AcceptSSHStreams 在 QUIC 连接上接受 SSH 流
// 这个方法在一个单独的 goroutine 中运行，持续监听来自服务器的 SSH 流请求
func (s *SSHIntegration) AcceptSSHStreams(ctx context.Context, conn *quic.Conn) {
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // 上下文取消，正常退出
			}
			if strings.Contains(err.Error(), "Application error 0x0") {
				return // 连接正常关闭
			}
			s.logger.Error("Failed to accept stream", "error", err)
			return
		}

		// 读取流头部以识别类型
		header, err := quicssh.ReadHeader(stream)
		if err != nil {
			s.logger.Debug("Failed to read stream header", "error", err)
			stream.Close()
			continue
		}

		// 检查是否是 SSH 流
		if header.Type == quicssh.StreamTypeSSH {
			s.logger.Info("Received SSH stream, starting handler", "stream_id", stream.StreamID())
			go func(st *quic.Stream) {
				s.logger.Debug("SSH stream handler goroutine started", "stream_id", st.StreamID())
				if err := s.HandleStream(st, conn); err != nil {
					s.logger.Error("SSH stream handling failed", "stream_id", st.StreamID(), "error", err)
					// 如果处理失败，确保关闭流
					st.Close()
				}
			}(stream)
		} else {
			s.logger.Debug("Non-SSH stream received", "type", header.Type)
			stream.Close()
		}
	}
}

// sshLogger SSH 日志适配器
type sshLogger struct {
	logger *monitoring.Logger
}

func (l *sshLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *sshLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *sshLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *sshLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}
