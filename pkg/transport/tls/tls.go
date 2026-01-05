package tls

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	pkgerrors "github.com/voilet/quic-flow/pkg/errors"
)

const (
	// ALPN 协议标识
	ALPNProtocol = "quic-backbone-v1"

	// Session Ticket Key 配置
	sessionTicketKeyName = "quic-flow-ticket-key"
	sessionKeyRotateInterval = 24 * time.Hour // 24小时轮换一次
)

var (
	// 全局 session ticket keys (支持密钥轮换)
	sessionTicketKeys    [][32]byte
	sessionTicketKeysMu  sync.RWMutex
	sessionTicketKeysIdx int

	// Session cache (用于 stateful session resumption)
	sessionCache = tls.NewLRUClientSessionCache(1024) // 缓存 1024 个 session
)

func init() {
	// 初始化默认 session ticket key（生产环境应从配置加载）
	initializeSessionTicketKeys()
}

// initializeSessionTicketKeys 初始化 session ticket keys
// 支持从环境变量加载多个密钥（逗号分隔）
// 例如: QUIC_SESSION_TICKET_KEYS=key1,key2,key3
func initializeSessionTicketKeys() {
	sessionTicketKeysMu.Lock()
	defer sessionTicketKeysMu.Unlock()

	// 默认保留 3 个密钥（支持平滑轮换）
	sessionTicketKeys = make([][32]byte, 3)

	// 尝试从环境变量加载（支持多个密钥）
	if keysHex := os.Getenv("QUIC_SESSION_TICKET_KEYS"); keysHex != "" {
		// 解析逗号分隔的密钥列表
		keyList := parseKeyList(keysHex)
		if len(keyList) > 0 {
			for i := 0; i < len(keyList) && i < len(sessionTicketKeys); i++ {
				if key, err := hex.DecodeString(keyList[i]); err == nil && len(key) == 32 {
					copy(sessionTicketKeys[i][:], key)
				}
			}
			// 填充剩余密钥
			for i := len(keyList); i < len(sessionTicketKeys); i++ {
				rand.Read(sessionTicketKeys[i][:])
			}
			return
		}
	}

	// 兼容旧的环境变量（单个密钥）
	if keyHex := os.Getenv("QUIC_SESSION_TICKET_KEY"); keyHex != "" {
		if key, err := hex.DecodeString(keyHex); err == nil && len(key) == 32 {
			copy(sessionTicketKeys[0][:], key)
			// 生成其他密钥用于轮换
			for i := 1; i < len(sessionTicketKeys); i++ {
				rand.Read(sessionTicketKeys[i][:])
			}
			return
		}
	}

	// 使用随机密钥（仅用于开发环境）
	for i := 0; i < len(sessionTicketKeys); i++ {
		rand.Read(sessionTicketKeys[i][:])
	}
}

// parseKeyList 解析逗号分隔的密钥列表
func parseKeyList(input string) []string {
	var result []string
	current := ""
	for _, c := range input {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// rotateSessionTicketKey 轮换 session ticket key
func rotateSessionTicketKey() {
	sessionTicketKeysMu.Lock()
	defer sessionTicketKeysMu.Unlock()

	// 将当前密钥移到次位置
	sessionTicketKeys[1] = sessionTicketKeys[0]

	// 生成新密钥
	rand.Read(sessionTicketKeys[0][:])
}

// getSessionTicketKeys 获取当前 session ticket keys
func getSessionTicketKeys() [][32]byte {
	sessionTicketKeysMu.RLock()
	defer sessionTicketKeysMu.RUnlock()

	// 返回密钥的副本，第一个元素是最新的密钥
	keys := make([][32]byte, len(sessionTicketKeys))
	copy(keys, sessionTicketKeys[:])
	return keys
}

// SetSessionTicketKeys 设置 session ticket keys（用于配置）
func SetSessionTicketKeys(keys [][32]byte) {
	sessionTicketKeysMu.Lock()
	defer sessionTicketKeysMu.Unlock()

	if len(keys) > 0 {
		sessionTicketKeys = make([][32]byte, len(keys))
		for i, key := range keys {
			sessionTicketKeys[i] = key
		}
	}
}

// LoadServerTLSConfig 加载服务器端 TLS 配置（性能优化版本）
// certFile: 证书文件路径
// keyFile: 私钥文件路径
func LoadServerTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	// 加载证书和私钥
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load certificate: %v", pkgerrors.ErrMissingTLSConfig, err)
	}

	return CreateOptimizedServerTLSConfig(cert), nil
}

// CreateOptimizedServerTLSConfig 创建优化的服务器 TLS 配置
// 这个函数应用了所有性能优化设置
func CreateOptimizedServerTLSConfig(cert tls.Certificate) *tls.Config {
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},

		// ALPN 协议标识
		NextProtos: []string{ALPNProtocol},

		// 强制 TLS 1.3（性能最优）
		MinVersion: tls.VersionTLS13,
		MaxVersion: tls.VersionTLS13,

		// ========== 性能优化配置 ==========

		// Client Session Cache - 用于 stateful session resumption
		// QUIC 会使用 session tickets 进行会话恢复
		ClientSessionCache: sessionCache,

		// 不要求客户端证书（如需双向 TLS 可改为 RequestClientCert）
		ClientAuth: tls.NoClientCert,

		// 优化 cipher suites 选择
		PreferServerCipherSuites: true,
	}

	return config
}

// GetSessionTicketKeys 获取 session ticket keys（用于 QUIC 配置）
func GetSessionTicketKeys() [][32]byte {
	return getSessionTicketKeys()
}

// LoadClientTLSConfig 加载客户端 TLS 配置
// certFile: 客户端证书文件路径（双向 TLS 时使用，可选）
// keyFile: 客户端私钥文件路径（双向 TLS 时使用，可选）
// caCertFile: CA 证书文件路径（用于验证服务器证书，可选）
// insecureSkipVerify: 是否跳过服务器证书验证（仅用于开发环境）
func LoadClientTLSConfig(certFile, keyFile, caCertFile string, insecureSkipVerify bool) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		NextProtos:         []string{ALPNProtocol}, // ALPN 协议标识
		MinVersion:         tls.VersionTLS13,       // 强制 TLS 1.3
		MaxVersion:         tls.VersionTLS13,
		InsecureSkipVerify: insecureSkipVerify, // 生产环境必须为 false
	}

	// 加载客户端证书（双向 TLS）
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to load client certificate: %v", pkgerrors.ErrMissingTLSConfig, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 加载 CA 证书（用于验证服务器）
	if caCertFile != "" {
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to read CA certificate: %v", pkgerrors.ErrMissingTLSConfig, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("%w: failed to parse CA certificate", pkgerrors.ErrMissingTLSConfig)
		}

		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// NewInsecureClientTLSConfig 创建不验证服务器证书的客户端 TLS 配置
// ⚠️ 警告：仅用于开发环境，生产环境禁止使用
func NewInsecureClientTLSConfig() *tls.Config {
	return &tls.Config{
		NextProtos:         []string{ALPNProtocol},
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true, // 跳过证书验证
	}
}

// ValidateTLSConfig 验证 TLS 配置的有效性
func ValidateTLSConfig(config *tls.Config) error {
	if config == nil {
		return fmt.Errorf("%w: TLS config is nil", pkgerrors.ErrMissingTLSConfig)
	}

	// 检查 ALPN 协议
	found := false
	for _, proto := range config.NextProtos {
		if proto == ALPNProtocol {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: ALPN protocol %s not found", pkgerrors.ErrInvalidConfig, ALPNProtocol)
	}

	// 检查 TLS 版本
	if config.MinVersion != tls.VersionTLS13 {
		return fmt.Errorf("%w: TLS 1.3 is required, got min version %d", pkgerrors.ErrInvalidConfig, config.MinVersion)
	}

	return nil
}
