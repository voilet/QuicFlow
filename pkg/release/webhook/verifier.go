package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
)

var (
	// ErrInvalidSignature 签名无效
	ErrInvalidSignature = errors.New("invalid signature")
	// ErrMissingSignature 缺少签名
	ErrMissingSignature = errors.New("missing signature")
	// ErrUnsupportedSignature 不支持的签名算法
	ErrUnsupportedSignature = errors.New("unsupported signature type")
)

// SignatureType 签名类型
type SignatureType string

const (
	// SignatureTypeGitHub GitHub 签名 (SHA1)
	SignatureTypeGitHub SignatureType = "github"
	// SignatureTypeGitLab GitLab 签名 (SHA1 或 SHA256)
	SignatureTypeGitLab SignatureType = "gitlab"
	// SignatureTypeGitee Gitee 签名 (SHA256)
	SignatureTypeGitee SignatureType = "gitee"
)

// Verifier Webhook 签名验证器
type Verifier struct {
	secret string
}

// NewVerifier 创建签名验证器
func NewVerifier(secret string) *Verifier {
	return &Verifier{
		secret: secret,
	}
}

// Verify 验证签名
func (v *Verifier) Verify(payload []byte, signature string, sigType SignatureType) error {
	if signature == "" {
		return ErrMissingSignature
	}

	expectedSig, err := v.ComputeSignature(payload, sigType)
	if err != nil {
		return err
	}

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return ErrInvalidSignature
	}

	return nil
}

// ComputeSignature 计算签名
func (v *Verifier) ComputeSignature(payload []byte, sigType SignatureType) (string, error) {
	var h func() hash.Hash

	switch sigType {
	case SignatureTypeGitHub:
		// GitHub 使用 SHA1-HMAC
		h = sha1.New
	case SignatureTypeGitLab:
		// GitLab 使用 SHA256-HMAC
		h = sha256.New
	case SignatureTypeGitee:
		// Gitee 使用 SHA256-HMAC
		h = sha256.New
	default:
		return "", ErrUnsupportedSignature
	}

	mac := hmac.New(h, []byte(v.secret))
	mac.Write(payload)

	return fmt.Sprintf("%s=%s", sigType, hex.EncodeToString(mac.Sum(nil))), nil
}

// VerifyGitHub 验证 GitHub Webhook 签名
// 格式: sha1=<hex_signature>
func (v *Verifier) VerifyGitHub(payload []byte, signature string) error {
	return v.Verify(payload, signature, SignatureTypeGitHub)
}

// VerifyGitLab 验证 GitLab Webhook 签名
// 格式: sha256=<hex_signature>
func (v *Verifier) VerifyGitLab(payload []byte, signature string) error {
	return v.Verify(payload, signature, SignatureTypeGitLab)
}

// VerifyGitee 验证 Gitee Webhook 签名
// 格式: sha256=<hex_signature>
func (v *Verifier) VerifyGitee(payload []byte, signature string) error {
	return v.Verify(payload, signature, SignatureTypeGitee)
}

// ExtractSignature 从签名头中提取签名值
// 输入: "sha1=abc123..." 或 "sha256=def456..."
// 输出: "abc123..." 或 "def456..."
func ExtractSignature(signatureHeader string) (string, error) {
	if signatureHeader == "" {
		return "", ErrMissingSignature
	}

	// GitHub/GitLab/Gitee 格式: <algorithm>=<hex_signature>
	// 使用 SplitN 限制分割次数为 2
	var sig string
	n, err := fmt.Sscanf(signatureHeader, "%*s=%s", &sig)
	if err != nil || n != 1 {
		return "", ErrInvalidSignature
	}

	return sig, nil
}
