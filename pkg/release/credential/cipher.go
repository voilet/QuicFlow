package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/voilet/quic-flow/pkg/release/models"
)

var (
	// ErrInvalidCiphertext 密文格式无效
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
	// ErrInvalidKey 密钥无效
	ErrInvalidKey = errors.New("invalid encryption key")
)

// Cipher 加密/解密接口
type Cipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	EncryptData(data *models.CredentialData) (string, error)
	DecryptData(ciphertext string) (*models.CredentialData, error)
}

// AESGCMCipher AES-256-GCM 加密实现
type AESGCMCipher struct {
	key []byte // 32 字节密钥
}

// NewCipher 创建加密器
func NewCipher(secretKey string) (*AESGCMCipher, error) {
	if secretKey == "" {
		return nil, ErrInvalidKey
	}

	// 使用 SHA256 派生 32 字节密钥
	key := sha256.Sum256([]byte(secretKey))

	return &AESGCMCipher{
		key: key[:],
	}, nil
}

// Encrypt 加密明文
// 格式: base64(nonce + ciphertext)
func (c *AESGCMCipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建 AES-GCM cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce (12 字节是 GCM 推荐值)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密密文
func (c *AESGCMCipher) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidCiphertext, err)
	}

	// 创建 AES-GCM cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	// 分离 nonce 和密文
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptData 加密凭证数据
func (c *AESGCMCipher) EncryptData(data *models.CredentialData) (string, error) {
	if data == nil {
		return "", nil
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal credential data: %w", err)
	}

	// 加密
	return c.Encrypt(string(jsonData))
}

// DecryptData 解密凭证数据
func (c *AESGCMCipher) DecryptData(ciphertext string) (*models.CredentialData, error) {
	if ciphertext == "" {
		return nil, nil
	}

	// 解密
	plaintext, err := c.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	// 反序列化 JSON
	var data models.CredentialData
	if err := json.Unmarshal([]byte(plaintext), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential data: %w", err)
	}

	return &data, nil
}

// GenerateSecretKey 生成随机密钥（用于初始化）
func GenerateSecretKey() (string, error) {
	key := make([]byte, 32) // 256 位
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
