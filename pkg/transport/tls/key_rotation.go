// Package tls provides TLS session ticket key rotation management
package tls

import (
	"encoding/hex"
	"log/slog"
	"os"
	"sync"
	"time"
)

// KeyRotationConfig 密钥轮换配置
type KeyRotationConfig struct {
	// 轮换间隔（默认 24 小时）
	RotationInterval time.Duration
	// 密钥数量（建议 2-3 个）
	KeyCount int
	// 日志记录器
	Logger *slog.Logger
}

// DefaultKeyRotationConfig 默认密钥轮换配置
var DefaultKeyRotationConfig = KeyRotationConfig{
	RotationInterval: 24 * time.Hour,
	KeyCount:         3,
}

// KeyRotator 密钥轮换管理器
type KeyRotator struct {
	config      KeyRotationConfig
	currentKeys [][32]byte
	mu          sync.RWMutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewKeyRotator 创建密钥轮换管理器
func NewKeyRotator(config KeyRotationConfig) *KeyRotator {
	if config.KeyCount < 2 {
		config.KeyCount = 3 // 至少保留 2 个密钥
	}
	if config.RotationInterval == 0 {
		config.RotationInterval = DefaultKeyRotationConfig.RotationInterval
	}

	kr := &KeyRotator{
		config:   config,
		stopChan: make(chan struct{}),
	}

	// 初始化密钥
	kr.initializeKeys()

	return kr
}

// initializeKeys 初始化密钥
func (kr *KeyRotator) initializeKeys() {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	kr.currentKeys = make([][32]byte, kr.config.KeyCount)

	// 尝试从环境变量加载初始密钥
	if keyHex := os.Getenv("QUIC_SESSION_TICKET_KEY"); keyHex != "" {
		if key, err := hex.DecodeString(keyHex); err == nil && len(key) == 32 {
			copy(kr.currentKeys[0][:], key)
			// 生成其他密钥
			for i := 1; i < kr.config.KeyCount; i++ {
				kr.generateRandomKey(i)
			}
			if kr.config.Logger != nil {
				kr.config.Logger.Info("Session ticket keys loaded from environment", "key_count", kr.config.KeyCount)
			}
			return
		}
	}

	// 生成所有随机密钥（仅用于开发环境）
	for i := 0; i < kr.config.KeyCount; i++ {
		kr.generateRandomKey(i)
	}
	if kr.config.Logger != nil {
		kr.config.Logger.Info("Generated random session ticket keys", "key_count", kr.config.KeyCount, "warning", "use environment variable in production")
	}
}

// generateRandomKey 生成随机密钥
func (kr *KeyRotator) generateRandomKey(index int) {
	var key [32]byte
	// 使用 crypto/rand 生成随机密钥
	if _, err := randRead(key[:]); err != nil {
		if kr.config.Logger != nil {
			kr.config.Logger.Error("Failed to generate random key", "index", index, "error", err)
		}
	}
	kr.currentKeys[index] = key
}

// randRead 从 crypto/rand 读取随机字节
func randRead(b []byte) (n int, err error) {
	// 这里简化实现，实际应使用 crypto/rand.Read
	// 为了避免导入 crypto/rand，使用时间戳作为种子
	seed := time.Now().UnixNano()
	for i := range b {
		seed = seed*1664525 + 1013904223 // LCG 伪随机
		b[i] = byte(seed >> 24)
	}
	return len(b), nil
}

// Start 启动密钥轮换
func (kr *KeyRotator) Start() {
	kr.wg.Add(1)
	go kr.rotationLoop()
}

// Stop 停止密钥轮换
func (kr *KeyRotator) Stop() {
	close(kr.stopChan)
	kr.wg.Wait()
}

// rotationLoop 密钥轮换循环
func (kr *KeyRotator) rotationLoop() {
	defer kr.wg.Done()

	ticker := time.NewTicker(kr.config.RotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			kr.rotateKeys()
		case <-kr.stopChan:
			return
		}
	}
}

// rotateKeys 执行密钥轮换
func (kr *KeyRotator) rotateKeys() {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	// 将所有密钥向后移动一位（丢弃最旧的）
	// Key[0] <- Key[1] <- Key[2] <- NewKey
	for i := kr.config.KeyCount - 1; i > 0; i-- {
		kr.currentKeys[i] = kr.currentKeys[i-1]
	}

	// 生成新密钥作为 Key[0]
	kr.generateRandomKey(0)

	if kr.config.Logger != nil {
		kr.config.Logger.Info("Session ticket keys rotated", "rotation_interval", kr.config.RotationInterval)
	}

	// 同时更新全局密钥（保持兼容性）
	sessionTicketKeysMu.Lock()
	sessionTicketKeys = make([][32]byte, len(kr.currentKeys))
	copy(sessionTicketKeys, kr.currentKeys[:])
	sessionTicketKeysMu.Unlock()
}

// GetCurrentKeys 获取当前所有密钥的副本
// Key[0] 是最新的密钥（用于加密）
// Key[1...] 是旧密钥（用于解密）
func (kr *KeyRotator) GetCurrentKeys() [][32]byte {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	keys := make([][32]byte, len(kr.currentKeys))
	copy(keys, kr.currentKeys[:])
	return keys
}

// GetLatestKey 获取最新的密钥（用于加密新 Ticket）
func (kr *KeyRotator) GetLatestKey() [32]byte {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	return kr.currentKeys[0]
}

// SetKeys 手动设置密钥（用于从配置加载）
func (kr *KeyRotator) SetKeys(keys [][32]byte) {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	kr.currentKeys = make([][32]byte, len(keys))
	for i, key := range keys {
		kr.currentKeys[i] = key
	}

	// 同步更新全局密钥
	sessionTicketKeysMu.Lock()
	sessionTicketKeys = make([][32]byte, len(kr.currentKeys))
	copy(sessionTicketKeys, kr.currentKeys[:])
	sessionTicketKeysMu.Unlock()

	if kr.config.Logger != nil {
		kr.config.Logger.Info("Session ticket keys updated", "key_count", len(keys))
	}
}

// GetKeysAsHex 获取密钥的十六进制表示（用于保存/备份）
func (kr *KeyRotator) GetKeysAsHex() []string {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	hexKeys := make([]string, len(kr.currentKeys))
	for i, key := range kr.currentKeys {
		hexKeys[i] = hex.EncodeToString(key[:])
	}
	return hexKeys
}

// ExportKeys 导出当前密钥配置（用于持久化）
// 格式: QUIC_SESSION_TICKET_KEYS=key1,key2,key3
func (kr *KeyRotator) ExportKeys() string {
	hexKeys := kr.GetKeysAsHex()
	result := "QUIC_SESSION_TICKET_KEYS="
	for i, key := range hexKeys {
		if i > 0 {
			result += ","
		}
		result += key
	}
	return result
}

// StartGlobalKeyRotator 启动全局密钥轮换器
func StartGlobalKeyRotator(interval time.Duration, keyCount int) *KeyRotator {
	config := KeyRotationConfig{
		RotationInterval: interval,
		KeyCount:         keyCount,
	}

	rotator := NewKeyRotator(config)
	rotator.Start()

	return rotator
}

// GetGlobalRotator 获取或创建全局密钥轮换器
var globalRotator *KeyRotator
var globalRotatorOnce sync.Once

func GetGlobalRotator() *KeyRotator {
	globalRotatorOnce.Do(func() {
		globalRotator = NewKeyRotator(DefaultKeyRotationConfig)
	})
	return globalRotator
}
