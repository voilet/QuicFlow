package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServerConfig SSH 服务器配置
// 用于在 QUIC 客户端（内网侧）运行 SSH 服务
type ServerConfig struct {
	// HostKeyPath 主机密钥文件路径
	// 如果未设置，将自动生成临时密钥（仅用于测试）
	HostKeyPath string

	// HostKey 主机私钥（如果已加载）
	// 优先级高于 HostKeyPath
	HostKey ssh.Signer

	// AuthorizedKeysPath 授权公钥文件路径
	AuthorizedKeysPath string

	// PasswordAuth 启用密码认证
	PasswordAuth bool

	// PasswordCallback 密码验证回调
	// 返回 nil 表示认证成功，返回 error 表示失败
	PasswordCallback func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)

	// PublicKeyCallback 公钥验证回调
	PublicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

	// NoClientAuth 禁用客户端认证（不安全，仅用于测试）
	NoClientAuth bool

	// Shell 默认 Shell 路径
	Shell string

	// IdleTimeout 空闲超时时间
	IdleTimeout time.Duration

	// MaxAuthTries 最大认证尝试次数
	MaxAuthTries int

	// Banner 连接横幅消息
	Banner string

	// AllowAgentForwarding 允许 SSH Agent 转发
	AllowAgentForwarding bool

	// AllowTcpForwarding 允许 TCP 端口转发
	AllowTcpForwarding bool

	// AllowPty 允许分配伪终端
	AllowPty bool
}

// DefaultServerConfig 返回默认的服务器配置
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Shell:                "/bin/sh",
		IdleTimeout:          30 * time.Minute,
		MaxAuthTries:         3,
		AllowAgentForwarding: false,
		AllowTcpForwarding:   true,
		AllowPty:             true,
		PasswordAuth:         true,
	}
}

// Validate 验证服务器配置
func (c *ServerConfig) Validate() error {
	if c.HostKey == nil && c.HostKeyPath == "" {
		// 允许自动生成密钥（用于测试）
	}

	if !c.PasswordAuth && c.PublicKeyCallback == nil && !c.NoClientAuth {
		return fmt.Errorf("%w: at least one authentication method must be enabled", ErrNoAuthMethods)
	}

	if c.MaxAuthTries <= 0 {
		c.MaxAuthTries = 3
	}

	return nil
}

// BuildSSHConfig 构建 SSH 服务器配置
func (c *ServerConfig) BuildSSHConfig() (*ssh.ServerConfig, error) {
	config := &ssh.ServerConfig{
		MaxAuthTries: c.MaxAuthTries,
	}

	// 设置密码认证
	if c.PasswordAuth && c.PasswordCallback != nil {
		config.PasswordCallback = c.PasswordCallback
	}

	// 设置公钥认证
	if c.PublicKeyCallback != nil {
		config.PublicKeyCallback = c.PublicKeyCallback
	}

	// 禁用认证（不安全）
	if c.NoClientAuth {
		config.NoClientAuth = true
	}

	// 设置横幅
	if c.Banner != "" {
		config.BannerCallback = func(conn ssh.ConnMetadata) string {
			return c.Banner
		}
	}

	// 加载或生成主机密钥
	hostKey, err := c.getHostKey()
	if err != nil {
		return nil, err
	}
	config.AddHostKey(hostKey)

	return config, nil
}

// getHostKey 获取主机密钥
func (c *ServerConfig) getHostKey() (ssh.Signer, error) {
	// 优先使用已设置的密钥
	if c.HostKey != nil {
		return c.HostKey, nil
	}

	// 从文件加载
	if c.HostKeyPath != "" {
		keyBytes, err := os.ReadFile(c.HostKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read host key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key: %w", err)
		}
		return signer, nil
	}

	// 生成临时密钥（仅用于测试）
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate host key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}
	return signer, nil
}

// ClientConfig SSH 客户端配置
// 用于在 QUIC 服务端（公网侧）连接到内网 SSH 服务
type ClientConfig struct {
	// User 用户名
	User string

	// Password 密码（密码认证）
	Password string

	// PrivateKeyPath 私钥文件路径（公钥认证）
	PrivateKeyPath string

	// PrivateKey 私钥数据（如果已加载）
	PrivateKey []byte

	// HostKeyCallback 主机密钥验证回调
	// 设置为 nil 将使用 InsecureIgnoreHostKey（不安全）
	HostKeyCallback ssh.HostKeyCallback

	// Timeout 连接超时
	Timeout time.Duration

	// KeepAlive 保活间隔
	KeepAlive time.Duration

	// ClientVersion 客户端版本字符串
	ClientVersion string
}

// DefaultClientConfig 返回默认的客户端配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		User:          "root",
		Timeout:       30 * time.Second,
		KeepAlive:     30 * time.Second,
		ClientVersion: "SSH-2.0-QUIC-SSH-1.0",
	}
}

// Validate 验证客户端配置
func (c *ClientConfig) Validate() error {
	if c.User == "" {
		return fmt.Errorf("%w: user is required", ErrInvalidConfig)
	}

	if c.Password == "" && c.PrivateKeyPath == "" && len(c.PrivateKey) == 0 {
		return fmt.Errorf("%w: password or private key is required", ErrInvalidConfig)
	}

	return nil
}

// BuildSSHConfig 构建 SSH 客户端配置
func (c *ClientConfig) BuildSSHConfig() (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:          c.User,
		Timeout:       c.Timeout,
		ClientVersion: c.ClientVersion,
	}

	// 设置认证方法
	var authMethods []ssh.AuthMethod

	// 密码认证
	if c.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.Password))
	}

	// 公钥认证
	if c.PrivateKeyPath != "" || len(c.PrivateKey) > 0 {
		signer, err := c.getPrivateKeySigner()
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if len(authMethods) == 0 {
		return nil, ErrNoAuthMethods
	}
	config.Auth = authMethods

	// 设置主机密钥验证
	if c.HostKeyCallback != nil {
		config.HostKeyCallback = c.HostKeyCallback
	} else {
		// 不安全：忽略主机密钥验证（生产环境应该设置正确的回调）
		config.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	return config, nil
}

// getPrivateKeySigner 获取私钥签名器
func (c *ClientConfig) getPrivateKeySigner() (ssh.Signer, error) {
	var keyBytes []byte
	var err error

	if len(c.PrivateKey) > 0 {
		keyBytes = c.PrivateKey
	} else if c.PrivateKeyPath != "" {
		keyBytes, err = os.ReadFile(c.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no private key provided")
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return signer, nil
}

// SessionConfig SSH 会话配置
type SessionConfig struct {
	// Env 环境变量
	Env map[string]string

	// Term 终端类型
	Term string

	// Width 终端宽度
	Width int

	// Height 终端高度
	Height int

	// Modes 终端模式
	Modes ssh.TerminalModes
}

// DefaultSessionConfig 返回默认的会话配置
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		Term:   "xterm-256color",
		Width:  80,
		Height: 24,
		Modes: ssh.TerminalModes{
			ssh.ECHO:          1,     // 启用回显
			ssh.TTY_OP_ISPEED: 14400, // 输入波特率
			ssh.TTY_OP_OSPEED: 14400, // 输出波特率
		},
	}
}
