package filetransfer

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// FileTransfer 文件传输记录模型
type FileTransfer struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TaskID          uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null;comment:任务UUID" json:"task_id"`
	FileName        string         `gorm:"size:512;not null;comment:文件名" json:"file_name"`
	FilePath        string         `gorm:"type:text;not null;comment:文件路径" json:"file_path"`
	FileSize        int64          `gorm:"not null;comment:文件大小(字节)" json:"file_size"`
	FileHash        string         `gorm:"size:64;comment:SHA256哈希" json:"file_hash,omitempty"`
	TransferType    string         `gorm:"size:10;not null;comment:传输类型" json:"transfer_type"` // upload, download
	Status          string         `gorm:"size:20;not null;index;comment:状态" json:"status"`        // pending, transferring, paused, completed, failed, cancelled
	Progress        int            `gorm:"default:0;not null;comment:进度百分比" json:"progress"`
	Speed           int64          `gorm:"default:0;comment:传输速度(字节/秒)" json:"speed"`
	BytesTransferred int64         `gorm:"default:0;comment:已传输字节数" json:"bytes_transferred"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null;index;comment:用户UUID" json:"user_id"`
	ClientIP        string         `gorm:"type:inet;comment:客户端IP" json:"client_ip,omitempty"`
	ErrorMessage    string         `gorm:"type:text;comment:错误信息" json:"error_message,omitempty"`
	Metadata        datatypes.JSON `gorm:"type:jsonb;comment:元数据" json:"metadata,omitempty"`
	StartedAt       time.Time      `gorm:"not null;comment:开始时间" json:"started_at"`
	CompletedAt     *time.Time     `gorm:"comment:完成时间" json:"completed_at,omitempty"`
	CreatedAt       time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (FileTransfer) TableName() string {
	return "file_transfers"
}

// BeforeCreate GORM hook
func (ft *FileTransfer) BeforeCreate(tx *gorm.DB) error {
	if ft.ID == uuid.Nil {
		ft.ID = uuid.New()
	}
	if ft.TaskID == uuid.Nil {
		ft.TaskID = uuid.New()
	}
	now := time.Now()
	if ft.StartedAt.IsZero() {
		ft.StartedAt = now
	}
	return nil
}

// FileMetadata 文件元数据模型
type FileMetadata struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	FilePath      string         `gorm:"type:text;uniqueIndex;not null;comment:文件路径" json:"file_path"`
	FileName      string         `gorm:"size:512;not null;comment:文件名" json:"file_name"`
	FileSize      int64          `gorm:"not null;comment:文件大小" json:"file_size"`
	FileHash      string         `gorm:"size:64;uniqueIndex;not null;comment:SHA256哈希" json:"file_hash"`
	ContentType   string         `gorm:"size:100;comment:内容类型" json:"content_type,omitempty"`
	StoragePath   string         `gorm:"type:text;not null;comment:实际存储路径" json:"storage_path"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index;comment:上传用户UUID" json:"user_id"`
	UploadCount   int            `gorm:"default:1;comment:上传次数" json:"upload_count"`
	DownloadCount int            `gorm:"default:0;comment:下载次数" json:"download_count"`
	Tags          pq.StringArray `gorm:"type:text[];comment:标签" json:"tags,omitempty"`
	Description   string         `gorm:"type:text;comment:描述" json:"description,omitempty"`
	IsDeleted     bool           `gorm:"default:false;index;comment:是否删除" json:"is_deleted"`
	CreatedAt     time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (FileMetadata) TableName() string {
	return "file_metadata"
}

// BeforeCreate GORM hook
func (fm *FileMetadata) BeforeCreate(tx *gorm.DB) error {
	if fm.ID == uuid.Nil {
		fm.ID = uuid.New()
	}
	return nil
}

// AllFileTransferModels 所有文件传输模型列表
var AllFileTransferModels = []interface{}{
	&FileTransfer{},
	&FileMetadata{},
}

// AutoMigrateFileTransfer 自动迁移文件传输表
func AutoMigrateFileTransfer(db *gorm.DB) error {
	return db.AutoMigrate(AllFileTransferModels...)
}

// CreateIndexes 创建文件传输相关索引
func CreateIndexes(db *gorm.DB) error {
	indexes := []string{
		// file_transfers 表索引
		`CREATE INDEX IF NOT EXISTS idx_file_transfers_user_status
		 ON file_transfers(user_id, status)
		 WHERE status IN ('pending', 'transferring', 'paused')`,
		`CREATE INDEX IF NOT EXISTS idx_file_transfers_created_at
		 ON file_transfers(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_file_transfers_type_status
		 ON file_transfers(transfer_type, status)`,

		// file_metadata 表索引
		`CREATE INDEX IF NOT EXISTS idx_file_metadata_tags
		 ON file_metadata USING GIN(tags)`,
		`CREATE INDEX IF NOT EXISTS idx_file_metadata_created_at
		 ON file_metadata(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_file_metadata_user_deleted
		 ON file_metadata(user_id, is_deleted)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			// 索引可能已存在，忽略错误
			continue
		}
	}

	return nil
}
