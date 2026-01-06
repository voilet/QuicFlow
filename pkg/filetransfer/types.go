package filetransfer

import (
	"context"
	"io"
	"time"
)

// TransferType 传输类型
type TransferType string

const (
	TransferTypeUpload   TransferType = "upload"
	TransferTypeDownload TransferType = "download"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending      TaskStatus = "pending"
	TaskStatusTransferring TaskStatus = "transferring"
	TaskStatusPaused       TaskStatus = "paused"
	TaskStatusCompleted    TaskStatus = "completed"
	TaskStatusFailed       TaskStatus = "failed"
	TaskStatusCancelled    TaskStatus = "cancelled"
)

// Config 文件传输配置
type Config struct {
	// 存储配置
	StorageRoot   string `yaml:"storage_root" json:"storage_root"`         // 存储根路径
	PathTemplate  string `yaml:"path_template" json:"path_template"`       // 路径模板，如 "{date}/{user}/{project}"

	// 限制配置
	MaxFileSize           int64  `yaml:"max_file_size" json:"max_file_size"`                       // 单文件大小限制（字节，0=无限制）
	StorageQuota          int64  `yaml:"storage_quota" json:"storage_quota"`                         // 总存储配额（字节）
	UserQuota             int64  `yaml:"user_quota" json:"user_quota"`                               // 用户配额（字节）
	MaxConcurrentTransfers int   `yaml:"max_concurrent_transfers" json:"max_concurrent_transfers"`   // 最大并发传输数

	// 性能配置
	BufferSize      int    `yaml:"buffer_size" json:"buffer_size"`           // 传输缓冲区大小（字节）
	ChunkSize       int    `yaml:"chunk_size" json:"chunk_size"`             // 分块大小（字节）
	UploadThreads   int    `yaml:"upload_threads" json:"upload_threads"`     // 上传线程数
	DownloadThreads int    `yaml:"download_threads" json:"download_threads"` // 下载线程数

	// 功能开关
	Compression     bool `yaml:"compression" json:"compression"`           // 是否启用压缩
	ChecksumVerify  bool `yaml:"checksum_verify" json:"checksum_verify"`   // 是否校验校验和
	ResumeSupport   bool `yaml:"resume_support" json:"resume_support"`     // 是否支持断点续传

	// 保留策略
	RetentionDays int  `yaml:"retention_days" json:"retention_days"` // 保留天数（0=永久）
	AutoCleanup   bool `yaml:"auto_cleanup" json:"auto_cleanup"`     // 是否自动清理过期文件
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		StorageRoot:           "/data/quic-files",
		PathTemplate:          "{date}/{user}",
		MaxFileSize:           10 * 1024 * 1024 * 1024, // 10GB
		StorageQuota:          1024 * 1024 * 1024 * 1024, // 1TB
		UserQuota:             100 * 1024 * 1024 * 1024, // 100GB
		MaxConcurrentTransfers: 100,
		BufferSize:            64 * 1024,  // 64KB
		ChunkSize:             1024 * 1024, // 1MB
		UploadThreads:         4,
		DownloadThreads:       4,
		Compression:           false,
		ChecksumVerify:        true,
		ResumeSupport:         true,
		RetentionDays:         30,
		AutoCleanup:           false,
	}
}

// TransferTask 传输任务
type TransferTask struct {
	ID              string
	Type            TransferType
	FileName        string
	SourcePath      string
	DestPath        string
	FileSize        int64
	Transferred     int64
	Status          TaskStatus
	Speed           int64 // bytes/sec
	Progress        float64
	CreatedAt       time.Time
	StartedAt       *time.Time
	CompletedAt     *time.Time
	Error           error
	Checksum        string // SHA256
	UserID          string
	ClientIP        string
	Metadata        map[string]interface{}
	Options         TransferOptions
	cancelChan      chan struct{}
	doneChan        chan struct{} // 用于通知任务完成
	progressChan    chan ProgressUpdate
}

// TransferOptions 传输选项
type TransferOptions struct {
	Overwrite      bool   `json:"overwrite"`       // 是否覆盖已存在文件
	Encryption     bool   `json:"encryption"`      // 是否加密传输
	Compression    bool   `json:"compression"`     // 是否压缩
	VerifyChecksum bool   `json:"verify_checksum"` // 是否验证校验和
	Resume         bool   `json:"resume"`          // 是否支持断点续传
	Threads        int    `json:"threads"`         // 并发线程数
}

// InitUploadRequest 初始化上传请求
type InitUploadRequest struct {
	Filename    string                 `json:"filename" binding:"required"`
	FileSize    int64                  `json:"file_size" binding:"required,min=1"`
	Checksum    string                 `json:"checksum,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
	Path        string                 `json:"path,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Options     TransferOptions        `json:"options,omitempty"`
}

// InitUploadResponse 初始化上传响应
type InitUploadResponse struct {
	TaskID       string         `json:"task_id"`
	UploadConfig UploadConfig   `json:"upload_config"`
	Status       TaskStatus     `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
}

// UploadConfig 上传配置
type UploadConfig struct {
	QUICUrl      string `json:"quic_url"`
	ChunkSize    int    `json:"chunk_size"`
	MaxRetries   int    `json:"max_retries"`
	Timeout      int    `json:"timeout"`
}

// UploadChunkRequest 上传分块请求
type UploadChunkRequest struct {
	TaskID   string `json:"task_id" binding:"required"`
	Offset   int64  `json:"offset" binding:"required,min=0"`
	Sequence int64  `json:"sequence" binding:"required,min=0"`
	Data     []byte `json:"data" binding:"required"`
	Checksum string `json:"checksum,omitempty"`
}

// UploadChunkResponse 上传分块响应
type UploadChunkResponse struct {
	Ack           bool    `json:"ack"`
	Received      int64   `json:"received"`
	TotalReceived int64   `json:"total_received"`
	Progress      float64 `json:"progress"`
}

// CompleteUploadRequest 完成上传请求
type CompleteUploadRequest struct {
	TaskID   string `json:"task_id" binding:"required"`
	Checksum string `json:"checksum,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CompleteUploadResponse 完成上传响应
type CompleteUploadResponse struct {
	TaskID         string      `json:"task_id"`
	Status         TaskStatus  `json:"status"`
	FileInfo       FileInfo    `json:"file_info"`
	TransferStats  TransferStats `json:"transfer_stats"`
	CompletedAt    time.Time   `json:"completed_at"`
}

// FileInfo 文件信息
type FileInfo struct {
	FileID      string `json:"file_id,omitempty"`
	FileName    string `json:"file_name"`
	FilePath    string `json:"file_path"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type,omitempty"`
	Checksum    string `json:"checksum,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

// TransferStats 传输统计
type TransferStats struct {
	Duration    int64  `json:"duration_ms"`
	AvgSpeed    string `json:"average_speed"`
	PeakSpeed   string `json:"peak_speed,omitempty"`
	TotalBytes  int64  `json:"total_bytes"`
}

// RequestDownloadRequest 请求下载请求
type RequestDownloadRequest struct {
	FileID    string          `json:"file_id,omitempty"`
	FilePath  string          `json:"file_path,omitempty"`
	LocalPath string          `json:"local_path,omitempty"`
	Offset    int64           `json:"offset,omitempty"`
	Options   TransferOptions `json:"options,omitempty"`
}

// RequestDownloadResponse 请求下载响应
type RequestDownloadResponse struct {
	TaskID         string       `json:"task_id"`
	DownloadConfig DownloadConfig `json:"download_config"`
	Status         TaskStatus   `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
}

// DownloadConfig 下载配置
type DownloadConfig struct {
	QUICUrl   string   `json:"quic_url"`
	FileInfo  FileInfo `json:"file_info"`
	ChunkSize int      `json:"chunk_size"`
	Timeout   int      `json:"timeout"`
}

// ProgressUpdate 进度更新
type ProgressUpdate struct {
	TaskID      string  `json:"task_id"`
	Status      TaskStatus `json:"status"`
	Progress    float64 `json:"progress"`
	Transferred int64   `json:"transferred"`
	Total       int64   `json:"total"`
	Speed       string  `json:"speed"`
	ETA         string  `json:"eta"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// QuotaInfo 配额信息
type QuotaInfo struct {
	Total            int64           `json:"total"`
	Used             int64           `json:"used"`
	Available        int64           `json:"available"`
	UsagePercentage  float64         `json:"usage_percentage"`
	Formatted        QuotaFormatted  `json:"formatted"`
}

// QuotaFormatted 格式化的配额信息
type QuotaFormatted struct {
	Total     string `json:"total"`
	Used      string `json:"used"`
	Available string `json:"available"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	Upload    UploadConfigLimits    `json:"upload"`
	Download  DownloadConfigLimits  `json:"download"`
	Storage   StorageConfig         `json:"storage"`
	Quotas    QuotaConfig           `json:"quotas"`
}

// UploadConfigLimits 上传配置限制
type UploadConfigLimits struct {
	MaxFileSize          int64    `json:"max_file_size"`
	MaxConcurrentUploads int      `json:"max_concurrent_uploads"`
	ChunkSize            int      `json:"chunk_size"`
	SupportedFormats     []string `json:"supported_formats"`
	ChecksumRequired     bool     `json:"checksum_required"`
}

// DownloadConfigLimits 下载配置限制
type DownloadConfigLimits struct {
	MaxConcurrentDownloads int  `json:"max_concurrent_downloads"`
	ChunkSize              int  `json:"chunk_size"`
	ResumeSupport          bool `json:"resume_support"`
	MultiThreadSupport     bool `json:"multi_thread_support"`
	MaxThreads             int  `json:"max_threads"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	RetentionDays    int  `json:"retention_days"`
	AutoCleanup      bool `json:"auto_cleanup"`
	CompressionAvailable bool `json:"compression_available"`
}

// QuotaConfig 配额配置
type QuotaConfig struct {
	UserQuota    int64 `json:"user_quota"`
	ProjectQuota int64 `json:"project_quota"`
}

// StorageBackend 存储后端接口
type StorageBackend interface {
	// Store 存储文件
	Store(ctx context.Context, path string, reader io.Reader, metadata FileMeta) error

	// Retrieve 检索文件
	Retrieve(ctx context.Context, path string) (io.ReadCloser, FileMeta, error)

	// Delete 删除文件
	Delete(ctx context.Context, path string) error

	// Exists 检查文件是否存在
	Exists(ctx context.Context, path string) (bool, error)

	// Stat 获取文件信息
	Stat(ctx context.Context, path string) (FileMeta, error)

	// List 列出文件
	List(ctx context.Context, prefix string, limit int) ([]FileMeta, error)

	// CheckQuota 检查配额
	CheckQuota(ctx context.Context, userID string, size int64) error
}

// FileMeta 文件元数据
type FileMeta struct {
	Name        string
	Path        string
	Size        int64
	ModTime     time.Time
	ContentType string
	Checksum    string
	UserID      string
	Tags        []string
	Description string
}

// StorageStorage 存储统计
type StorageStats struct {
	TotalFiles    int64  `json:"total_files"`
	TotalSize     int64  `json:"total_size"`
	AvailableSize int64  `json:"available_size"`
}
