package filetransfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LocalStorage 本地文件系统存储实现
type LocalStorage struct {
	rootPath    string
	pathTemplate string
	userQuota   int64
	db          *gorm.DB
	mu          sync.RWMutex
}

// NewLocalStorage 创建本地存储实例
func NewLocalStorage(rootPath, pathTemplate string, userQuota int64, db *gorm.DB) (*LocalStorage, error) {
	// 确保根路径存在
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage root: %w", err)
	}

	return &LocalStorage{
		rootPath:    rootPath,
		pathTemplate: pathTemplate,
		userQuota:   userQuota,
		db:          db,
	}, nil
}

// resolvePath 解析实际存储路径
func (ls *LocalStorage) resolvePath(user, project, filename string) string {
	path := ls.pathTemplate

	// 替换路径变量
	now := time.Now()
	replacements := map[string]string{
		"{date}":      now.Format("2006-01-02"),
		"{year}":      now.Format("2006"),
		"{month}":     now.Format("01"),
		"{day}":       now.Format("02"),
		"{user}":      user,
		"{project}":   project,
		"{timestamp}": fmt.Sprintf("%d", now.Unix()),
	}

	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}

	// 清理路径
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(ls.rootPath, path)
	}

	// 添加文件名
	return filepath.Join(path, filename)
}

// checkQuota 检查用户配额
func (ls *LocalStorage) checkQuota(ctx context.Context, userID string, size int64) error {
	// 匿名用户使用默认配额检查，不查询数据库
	if userID == "anonymous" {
		if ls.userQuota > 0 && size > ls.userQuota {
			return NewTransferError(ErrCodeStorageQuotaExceeded,
				fmt.Sprintf("File size %d exceeds anonymous quota %d", size, ls.userQuota), nil)
		}
		return nil
	}

	if ls.userQuota <= 0 {
		return nil // 无限制
	}

	// 查询用户已使用的存储空间
	var usedSpace int64
	err := ls.db.WithContext(ctx).
		Model(&FileMetadata{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&usedSpace).Error
	if err != nil {
		return fmt.Errorf("failed to check user quota: %w", err)
	}

	if usedSpace+size > ls.userQuota {
		return NewTransferError(ErrCodeStorageQuotaExceeded,
			fmt.Sprintf("Quota exceeded. Used: %d, Required: %d, Limit: %d",
				usedSpace, size, ls.userQuota), nil)
	}

	return nil
}

// Store 存储文件
func (ls *LocalStorage) Store(ctx context.Context, path string, reader io.Reader, metadata FileMeta) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// 检查配额
	if err := ls.checkQuota(ctx, metadata.UserID, metadata.Size); err != nil {
		return err
	}

	// 解析实际存储路径
	fullPath := ls.resolvePath(metadata.UserID, "", filepath.Base(path))

	// 确保目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewTransferError(ErrCodeStorageError, "Failed to create directory", err)
	}

	// 创建临时文件
	tempPath := fullPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return NewTransferError(ErrCodeStorageError, "Failed to create file", err)
	}
	defer file.Close()

	// 写入数据
	written, err := io.Copy(file, reader)
	if err != nil {
		os.Remove(tempPath)
		return NewTransferError(ErrCodeStorageError, "Failed to write file", err)
	}

	// 验证文件大小
	if written != metadata.Size {
		os.Remove(tempPath)
		return NewTransferError(ErrCodeInvalidParameters,
			fmt.Sprintf("Size mismatch: expected %d, got %d", metadata.Size, written), nil)
	}

	// 重命名为最终文件名
	if err := os.Rename(tempPath, fullPath); err != nil {
		os.Remove(tempPath)
		return NewTransferError(ErrCodeStorageError, "Failed to finalize file", err)
	}

	// 保存元数据到数据库
	// 处理匿名用户的 UUID
	var userID uuid.UUID
	if metadata.UserID == "anonymous" {
		userID = uuid.Nil // 使用 nil UUID 表示匿名用户
	} else {
		var err error
		userID, err = uuid.Parse(metadata.UserID)
		if err != nil {
			os.Remove(fullPath)
			return NewTransferError(ErrCodeInvalidParameters, "Invalid user ID", err)
		}
	}

	// 检查是否已存在相同哈希的文件
	var existingMeta FileMetadata
	checkErr := ls.db.WithContext(ctx).
		Where("file_hash = ? AND is_deleted = ?", metadata.Checksum, false).
		First(&existingMeta).Error

	if checkErr == nil {
		// 文件已存在，删除刚上传的文件并返回成功
		os.Remove(fullPath)
		// 返回已存在文件的信息
		return nil
	} else if checkErr != gorm.ErrRecordNotFound {
		os.Remove(fullPath)
		return NewTransferError(ErrCodeStorageError, "Failed to check existing file", checkErr)
	}

	// 文件不存在，创建新记录
	dbMeta := &FileMetadata{
		FilePath:    path,
		FileName:    metadata.Name,
		FileSize:    metadata.Size,
		FileHash:    metadata.Checksum,
		ContentType: metadata.ContentType,
		StoragePath: fullPath,
		UserID:      userID,
		Tags:       metadata.Tags,
		Description: metadata.Description,
	}

	if err := ls.db.WithContext(ctx).Create(dbMeta).Error; err != nil {
		// 数据库保存失败，删除已上传的文件
		os.Remove(fullPath)
		return NewTransferError(ErrCodeStorageError, "Failed to save metadata", err)
	}

	return nil
}

// Retrieve 检索文件
func (ls *LocalStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, FileMeta, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	// 从数据库查询元数据
	var dbMeta FileMetadata
	err := ls.db.WithContext(ctx).
		Where("file_path = ? AND is_deleted = ?", path, false).
		First(&dbMeta).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, FileMeta{}, ErrFileNotFound
		}
		return nil, FileMeta{}, NewTransferError(ErrCodeStorageError, "Failed to query metadata", err)
	}

	// 打开文件
	file, err := os.Open(dbMeta.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, FileMeta{}, ErrFileNotFound
		}
		return nil, FileMeta{}, NewTransferError(ErrCodeStorageError, "Failed to open file", err)
	}

	// 构造元数据
	metadata := FileMeta{
		Name:        dbMeta.FileName,
		Path:        dbMeta.FilePath,
		Size:        dbMeta.FileSize,
		ModTime:     dbMeta.CreatedAt,
		ContentType: dbMeta.ContentType,
		Checksum:    dbMeta.FileHash,
		UserID:      dbMeta.UserID.String(),
		Tags:        dbMeta.Tags,
		Description: dbMeta.Description,
	}

	return file, metadata, nil
}

// Delete 删除文件
func (ls *LocalStorage) Delete(ctx context.Context, path string) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	// 从数据库查询元数据
	var dbMeta FileMetadata
	err := ls.db.WithContext(ctx).
		Where("file_path = ? AND is_deleted = ?", path, false).
		First(&dbMeta).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrFileNotFound
		}
		return NewTransferError(ErrCodeStorageError, "Failed to query metadata", err)
	}

	// 删除物理文件
	if err := os.Remove(dbMeta.StoragePath); err != nil && !os.IsNotExist(err) {
		return NewTransferError(ErrCodeStorageError, "Failed to delete file", err)
	}

	// 标记为已删除
	if err := ls.db.WithContext(ctx).
		Model(&dbMeta).
		Update("is_deleted", true).Error; err != nil {
		return NewTransferError(ErrCodeStorageError, "Failed to update metadata", err)
	}

	return nil
}

// Exists 检查文件是否存在
func (ls *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	var count int64
	err := ls.db.WithContext(ctx).
		Model(&FileMetadata{}).
		Where("file_path = ? AND is_deleted = ?", path, false).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Stat 获取文件信息
func (ls *LocalStorage) Stat(ctx context.Context, path string) (FileMeta, error) {
	var dbMeta FileMetadata
	err := ls.db.WithContext(ctx).
		Where("file_path = ? AND is_deleted = ?", path, false).
		First(&dbMeta).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return FileMeta{}, ErrFileNotFound
		}
		return FileMeta{}, NewTransferError(ErrCodeStorageError, "Failed to query metadata", err)
	}

	return FileMeta{
		Name:        dbMeta.FileName,
		Path:        dbMeta.FilePath,
		Size:        dbMeta.FileSize,
		ModTime:     dbMeta.CreatedAt,
		ContentType: dbMeta.ContentType,
		Checksum:    dbMeta.FileHash,
		UserID:      dbMeta.UserID.String(),
		Tags:        dbMeta.Tags,
		Description: dbMeta.Description,
	}, nil
}

// List 列出文件
func (ls *LocalStorage) List(ctx context.Context, prefix string, limit int) ([]FileMeta, error) {
	query := ls.db.WithContext(ctx).
		Model(&FileMetadata{}).
		Where("is_deleted = ?", false)

	if prefix != "" && prefix != "/" {
		query = query.Where("file_path LIKE ?", prefix+"%")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var dbMetas []FileMetadata
	if err := query.Find(&dbMetas).Error; err != nil {
		return nil, NewTransferError(ErrCodeStorageError, "Failed to list files", err)
	}

	metas := make([]FileMeta, len(dbMetas))
	for i, dbMeta := range dbMetas {
		metas[i] = FileMeta{
			Name:        dbMeta.FileName,
			Path:        dbMeta.FilePath,
			Size:        dbMeta.FileSize,
			ModTime:     dbMeta.CreatedAt,
			ContentType: dbMeta.ContentType,
			Checksum:    dbMeta.FileHash,
			UserID:      dbMeta.UserID.String(),
			Tags:        dbMeta.Tags,
			Description: dbMeta.Description,
		}
	}

	return metas, nil
}

// CheckQuota 检查配额
func (ls *LocalStorage) CheckQuota(ctx context.Context, userID string, size int64) error {
	return ls.checkQuota(ctx, userID, size)
}

// GetUserQuotaInfo 获取用户配额信息
func (ls *LocalStorage) GetUserQuotaInfo(ctx context.Context, userID string) (*QuotaInfo, error) {
	if ls.userQuota <= 0 {
		return &QuotaInfo{
			Total:           -1, // 无限制
			Used:            0,
			Available:       -1,
			UsagePercentage: 0,
			Formatted: QuotaFormatted{
				Total:     "Unlimited",
				Used:      "0 B",
				Available: "Unlimited",
			},
		}, nil
	}

	var usedSpace int64
	err := ls.db.WithContext(ctx).
		Model(&FileMetadata{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&usedSpace).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get user quota info: %w", err)
	}

	available := ls.userQuota - usedSpace
	usagePercent := float64(usedSpace) / float64(ls.userQuota) * 100

	return &QuotaInfo{
		Total:           ls.userQuota,
		Used:            usedSpace,
		Available:       available,
		UsagePercentage: usagePercent,
		Formatted: QuotaFormatted{
			Total:     formatBytes(ls.userQuota),
			Used:      formatBytes(usedSpace),
			Available: formatBytes(available),
		},
	}, nil
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
