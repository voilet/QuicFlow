package filetransfer

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FileTransferConfig 文件传输配置结构
type FileTransferConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// 存储配置
	StorageRoot   string `yaml:"storage_root" json:"storage_root"`
	PathTemplate  string `yaml:"path_template" json:"path_template"`
	TempDir       string `yaml:"temp_dir" json:"temp_dir"`
	// 限制配置
	MaxFileSize           int64  `yaml:"max_file_size" json:"max_file_size"`
	StorageQuota          int64  `yaml:"storage_quota" json:"storage_quota"`
	UserQuota             int64  `yaml:"user_quota" json:"user_quota"`
	MaxConcurrentTransfers int    `yaml:"max_concurrent_transfers" json:"max_concurrent_transfers"`
	// 性能配置
	BufferSize      int `yaml:"buffer_size" json:"buffer_size"`
	ChunkSize       int `yaml:"chunk_size" json:"chunk_size"`
	UploadThreads   int `yaml:"upload_threads" json:"upload_threads"`
	DownloadThreads int `yaml:"download_threads" json:"download_threads"`
	// 功能开关
	Compression    bool `yaml:"compression" json:"compression"`
	ChecksumVerify bool `yaml:"checksum_verify" json:"checksum_verify"`
	ResumeSupport  bool `yaml:"resume_support" json:"resume_support"`
	// 保留策略
	RetentionDays int  `yaml:"retention_days" json:"retention_days"`
	AutoCleanup   bool `yaml:"auto_cleanup" json:"auto_cleanup"`
}

// LoadFromFile 从文件加载配置
func LoadFromFile(configPath string) (*FileTransferConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		FileTransfer *FileTransferConfig `yaml:"filetransfer"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.FileTransfer == nil {
		return nil, fmt.Errorf("filetransfer section not found in config")
	}

	// 设置默认值
	setDefaults(config.FileTransfer)

	return config.FileTransfer, nil
}

// LoadFromBytes 从字节数组加载配置
func LoadFromBytes(data []byte) (*FileTransferConfig, error) {
	var config struct {
		FileTransfer *FileTransferConfig `yaml:"filetransfer"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.FileTransfer == nil {
		return nil, fmt.Errorf("filetransfer section not found")
	}

	setDefaults(config.FileTransfer)

	return config.FileTransfer, nil
}

// setDefaults 设置默认值
func setDefaults(config *FileTransferConfig) {
	if config.StorageRoot == "" {
		config.StorageRoot = "/data/quic-files"
	}
	if config.PathTemplate == "" {
		config.PathTemplate = "{date}/{user}"
	}
	if config.TempDir == "" {
		config.TempDir = "/tmp/quic-upload"
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize = 10 * 1024 * 1024 * 1024 // 10GB
	}
	if config.StorageQuota == 0 {
		config.StorageQuota = 1024 * 1024 * 1024 * 1024 // 1TB
	}
	if config.UserQuota == 0 {
		config.UserQuota = 100 * 1024 * 1024 * 1024 // 100GB
	}
	if config.MaxConcurrentTransfers == 0 {
		config.MaxConcurrentTransfers = 100
	}
	if config.BufferSize == 0 {
		config.BufferSize = 64 * 1024 // 64KB
	}
	if config.ChunkSize == 0 {
		config.ChunkSize = 1024 * 1024 // 1MB
	}
	if config.UploadThreads == 0 {
		config.UploadThreads = 4
	}
	if config.DownloadThreads == 0 {
		config.DownloadThreads = 4
	}
}

// ToConfig 转换为 Config
func (fc *FileTransferConfig) ToConfig() *Config {
	return &Config{
		StorageRoot:           fc.StorageRoot,
		PathTemplate:          fc.PathTemplate,
		MaxFileSize:           fc.MaxFileSize,
		StorageQuota:          fc.StorageQuota,
		UserQuota:             fc.UserQuota,
		MaxConcurrentTransfers: fc.MaxConcurrentTransfers,
		BufferSize:            fc.BufferSize,
		ChunkSize:             fc.ChunkSize,
		UploadThreads:         fc.UploadThreads,
		DownloadThreads:       fc.DownloadThreads,
		Compression:           fc.Compression,
		ChecksumVerify:        fc.ChecksumVerify,
		ResumeSupport:         fc.ResumeSupport,
		RetentionDays:         fc.RetentionDays,
		AutoCleanup:           fc.AutoCleanup,
	}
}

// Validate 验证配置
func (fc *FileTransferConfig) Validate() error {
	// 检查路径
	if fc.StorageRoot == "" {
		return fmt.Errorf("storage_root cannot be empty")
	}

	if fc.TempDir == "" {
		return fmt.Errorf("temp_dir cannot be empty")
	}

	// 检查大小限制
	if fc.MaxFileSize < 0 {
		return fmt.Errorf("max_file_size cannot be negative")
	}

	if fc.StorageQuota < 0 {
		return fmt.Errorf("storage_quota cannot be negative")
	}

	if fc.UserQuota < 0 {
		return fmt.Errorf("user_quota cannot be negative")
	}

	// 检查并发数
	if fc.MaxConcurrentTransfers <= 0 {
		return fmt.Errorf("max_concurrent_transfers must be positive")
	}

	// 检查缓冲区和块大小
	if fc.BufferSize <= 0 {
		return fmt.Errorf("buffer_size must be positive")
	}

	if fc.ChunkSize <= 0 {
		return fmt.Errorf("chunk_size must be positive")
	}

	if fc.ChunkSize%fc.BufferSize != 0 {
		return fmt.Errorf("chunk_size should be a multiple of buffer_size")
	}

	// 检查线程数
	if fc.UploadThreads <= 0 {
		return fmt.Errorf("upload_threads must be positive")
	}

	if fc.DownloadThreads <= 0 {
		return fmt.Errorf("download_threads must be positive")
	}

	return nil
}

// EnsureDirectories 确保必要的目录存在
func (fc *FileTransferConfig) EnsureDirectories() error {
	// 创建存储根目录
	if err := os.MkdirAll(fc.StorageRoot, 0755); err != nil {
		return fmt.Errorf("failed to create storage_root: %w", err)
	}

	// 创建临时目录
	if err := os.MkdirAll(fc.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp_dir: %w", err)
	}

	return nil
}
