package filetransfer

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ChecksumType 校验和类型
type ChecksumType string

const (
	ChecksumTypeSHA256 ChecksumType = "sha256"
	ChecksumTypeMD5    ChecksumType = "md5"
)

// ChecksumCalculator 校验和计算器
type ChecksumCalculator struct {
	hashType ChecksumType
}

// NewChecksumCalculator 创建校验和计算器
func NewChecksumCalculator(hashType ChecksumType) *ChecksumCalculator {
	return &ChecksumCalculator{
		hashType: hashType,
	}
}

// NewSHA256Calculator 创建 SHA256 校验和计算器
func NewSHA256Calculator() *ChecksumCalculator {
	return &ChecksumCalculator{
		hashType: ChecksumTypeSHA256,
	}
}

// Calculate 计算校验和
func (cc *ChecksumCalculator) Calculate(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return cc.formatHash(h.Sum(nil))
}

// CalculateReader 计算流的校验和
func (cc *ChecksumCalculator) CalculateReader(reader io.Reader) (string, error) {
	h := sha256.New()

	// 使用缓冲读取提高性能
	bufReader := bufio.NewReaderSize(reader, 64*1024) // 64KB buffer
	_, err := io.Copy(h, bufReader)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return cc.formatHash(h.Sum(nil)), nil
}

// CalculateFile 计算文件的校验和
func (cc *ChecksumCalculator) CalculateFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return cc.CalculateReader(file)
}

// Verify 验证校验和
func (cc *ChecksumCalculator) Verify(data []byte, expectedChecksum string) bool {
	actualChecksum := cc.Calculate(data)
	return compareChecksum(actualChecksum, expectedChecksum)
}

// VerifyReader 验证流的校验和
func (cc *ChecksumCalculator) VerifyReader(reader io.Reader, expectedChecksum string) (bool, error) {
	actualChecksum, err := cc.CalculateReader(reader)
	if err != nil {
		return false, err
	}
	return compareChecksum(actualChecksum, expectedChecksum), nil
}

// VerifyFile 验证文件的校验和
func (cc *ChecksumCalculator) VerifyFile(filePath string, expectedChecksum string) (bool, error) {
	actualChecksum, err := cc.CalculateFile(filePath)
	if err != nil {
		return false, err
	}
	return compareChecksum(actualChecksum, expectedChecksum), nil
}

// formatHash 格式化哈希值
func (cc *ChecksumCalculator) formatHash(hashBytes []byte) string {
	return fmt.Sprintf("%s:%s", cc.hashType, hex.EncodeToString(hashBytes))
}

// extractHash 提取哈希值（移除类型前缀）
func extractHash(checksum string) string {
	// 如果包含冒号，取冒号后的部分
	for i := 0; i < len(checksum); i++ {
		if checksum[i] == ':' {
			return checksum[i+1:]
		}
	}
	return checksum
}

// compareChecksum 比较校验和
func compareChecksum(actual, expected string) bool {
	// 移除可能的类型前缀
	actualHash := extractHash(actual)
	expectedHash := extractHash(expected)
	return actualHash == expectedHash
}

// CompareChecksum 导出的比较函数
func CompareChecksum(actual, expected string) bool {
	return compareChecksum(actual, expected)
}

// ParseChecksum 解析校验和字符串
func ParseChecksum(checksum string) (ChecksumType, string, error) {
	for i := 0; i < len(checksum); i++ {
		if checksum[i] == ':' {
			hashType := ChecksumType(checksum[:i])
			hashValue := checksum[i+1:]

			// 验证哈希类型
			switch hashType {
			case ChecksumTypeSHA256, ChecksumTypeMD5:
				return hashType, hashValue, nil
			default:
				return "", "", fmt.Errorf("unsupported checksum type: %s", hashType)
			}
		}
	}

	// 如果没有类型前缀，默认为 SHA256
	return ChecksumTypeSHA256, checksum, nil
}

// StreamingChecksum 流式校验和计算器
type StreamingChecksum struct {
	hash   *sha256Hash
	hasher *ChecksumCalculator
}

type sha256Hash struct {
	h *sha256HashWrapper
}

type sha256HashWrapper struct {
	hashImpl sha256Impl
}

type sha256Impl struct {
	data []byte
}

func newSHA256Hash() *sha256Hash {
	return &sha256Hash{}
}

func (s *sha256Hash) Write(p []byte) (int, error) {
	if s.h == nil {
		s.h = &sha256HashWrapper{}
	}
	s.h.hashImpl.data = append(s.h.hashImpl.data, p...)
	return len(p), nil
}

func (s *sha256Hash) Sum(p []byte) []byte {
	if s.h == nil {
		return sha256.New().Sum(p)
	}
	h := sha256.New()
	h.Write(s.h.hashImpl.data)
	return h.Sum(p)
}

func (s *sha256Hash) Reset() {
	if s.h != nil {
		s.h.hashImpl.data = nil
	}
}

// NewStreamingChecksum 创建流式校验和计算器
func NewStreamingChecksum(hashType ChecksumType) *StreamingChecksum {
	return &StreamingChecksum{
		hash:   newSHA256Hash(),
		hasher: &ChecksumCalculator{hashType: hashType},
	}
}

// Write 实现 io.Writer 接口
func (sc *StreamingChecksum) Write(p []byte) (int, error) {
	return sc.hash.Write(p)
}

// Sum 返回当前校验和
func (sc *StreamingChecksum) Sum() string {
	return sc.hasher.formatHash(sc.hash.Sum(nil))
}

// Reset 重置校验和
func (sc *StreamingChecksum) Reset() {
	sc.hash.Reset()
}

// TeeChecksum 带校验和的 Reader
type TeeChecksum struct {
	reader       io.Reader
	checksum     *StreamingChecksum
	totalRead    int64
}

// NewTeeChecksum 创建带校验和的 Reader
func NewTeeChecksum(reader io.Reader, hashType ChecksumType) *TeeChecksum {
	return &TeeChecksum{
		reader:   reader,
		checksum: NewStreamingChecksum(hashType),
	}
}

// Read 实现 io.Reader 接口
func (tc *TeeChecksum) Read(p []byte) (int, error) {
	n, err := tc.reader.Read(p)
	if n > 0 {
		tc.checksum.Write(p[:n])
		tc.totalRead += int64(n)
	}
	return n, err
}

// Checksum 返回当前校验和
func (tc *TeeChecksum) Checksum() string {
	return tc.checksum.Sum()
}

// TotalRead 返回已读取的字节数
func (tc *TeeChecksum) TotalRead() int64 {
	return tc.totalRead
}
