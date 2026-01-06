package filetransfer

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// DownloadManager 下载管理器
type DownloadManager struct {
	manager *Manager
}

// NewDownloadManager 创建下载管理器
func NewDownloadManager(manager *Manager) *DownloadManager {
	return &DownloadManager{
		manager: manager,
	}
}

// RequestDownload 请求下载
func (dm *DownloadManager) RequestDownload(ctx context.Context, req *RequestDownloadRequest, userID, clientIP string) (*RequestDownloadResponse, error) {
	// 获取文件信息
	var fileMeta FileMeta
	var err error

	if req.FileID != "" {
		// 通过文件ID获取
		fileID, err := uuid.Parse(req.FileID)
		if err != nil {
			return nil, NewTransferError(ErrCodeInvalidParameters, "invalid file ID", err)
		}

		var dbMeta FileMetadata
		err = dm.manager.db.WithContext(ctx).
			Where("id = ? AND is_deleted = ?", fileID, false).
			First(&dbMeta).Error
		if err != nil {
			if err == ErrRecordNotFound {
				return nil, ErrFileNotFound
			}
			return nil, NewTransferError(ErrCodeStorageError, "failed to query file", err)
		}

		fileMeta = FileMeta{
			Name:     dbMeta.FileName,
			Path:     dbMeta.FilePath,
			Size:     dbMeta.FileSize,
			Checksum: dbMeta.FileHash,
		}
	} else if req.FilePath != "" {
		// 通过文件路径获取
		fileMeta, err = dm.manager.storage.Stat(ctx, req.FilePath)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, NewTransferError(ErrCodeInvalidParameters, "file_id or file_path is required", nil)
	}

	// 创建任务
	task := &TransferTask{
		Type:      TransferTypeDownload,
		FileName:  fileMeta.Name,
		SourcePath: fileMeta.Path,
		FileSize:  fileMeta.Size,
		Checksum:  fileMeta.Checksum,
		UserID:    userID,
		ClientIP:  clientIP,
		Options:   req.Options,
		Metadata:  make(map[string]interface{}),
		Status:    TaskStatusPending,
		Progress:  0,
		cancelChan: make(chan struct{}),
	}

	// 如果有本地路径
	if req.LocalPath != "" {
		task.DestPath = req.LocalPath
	}

	// 提交任务
	submittedTask, err := dm.manager.SubmitTask(task)
	if err != nil {
		return nil, err
	}

	return &RequestDownloadResponse{
		TaskID: submittedTask.ID,
		DownloadConfig: DownloadConfig{
			QUICUrl: fmt.Sprintf("quic://localhost:4242/download/%s", submittedTask.ID),
			FileInfo: FileInfo{
				FileName: fileMeta.Name,
				FilePath: fileMeta.Path,
				FileSize: fileMeta.Size,
				Checksum: fileMeta.Checksum,
			},
			ChunkSize: dm.manager.config.ChunkSize,
			Timeout:   600,
		},
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}, nil
}

// StartDownload 开始下载
func (dm *DownloadManager) StartDownload(ctx context.Context, taskID string) (io.ReadCloser, *FileInfo, error) {
	// 获取任务
	task, err := dm.manager.GetTask(taskID)
	if err != nil {
		return nil, nil, err
	}

	if task.Type != TransferTypeDownload {
		return nil, nil, NewTransferError(ErrCodeInvalidParameters, "task is not a download task", nil)
	}

	// 从存储检索文件
	reader, metadata, err := dm.manager.storage.Retrieve(ctx, task.SourcePath)
	if err != nil {
		return nil, nil, err
	}

	// 包装进度读取器
	progressReader := NewProgressReader(taskID, dm.manager.tracker, task.FileSize, reader)

	fileInfo := &FileInfo{
		FileName:    metadata.Name,
		FilePath:    metadata.Path,
		FileSize:    metadata.Size,
		ContentType: metadata.ContentType,
		Checksum:    metadata.Checksum,
	}

	// 返回带清理功能的 ReadCloser
	return &downloadReadCloser{
		reader:        progressReader,
		taskID:        taskID,
		manager:       dm.manager,
		originalReader: reader,
	}, fileInfo, nil
}

// downloadReadCloser 下载读取器（带清理功能）
type downloadReadCloser struct {
	reader         *ProgressReader
	taskID         string
	manager        *Manager
	originalReader io.ReadCloser
	closed         bool
}

// Read 实现 io.Reader
func (drc *downloadReadCloser) Read(p []byte) (int, error) {
	return drc.reader.Read(p)
}

// Close 实现 io.Closer
func (drc *downloadReadCloser) Close() error {
	if drc.closed {
		return nil
	}
	drc.closed = true

	// 关闭原始读取器
	err := drc.originalReader.Close()

	// 标记进度为完成
	drc.reader.Close()

	// 更新任务状态
	task, err := drc.manager.GetTask(drc.taskID)
	if err == nil {
		task.Status = TaskStatusCompleted
		now := time.Now()
		task.CompletedAt = &now
		drc.manager.updateTaskInDB(task)
	}

	return err
}

// GetDownloadProgress 获取下载进度
func (dm *DownloadManager) GetDownloadProgress(taskID string) (*TransferProgress, bool) {
	return dm.manager.tracker.Get(taskID)
}

// CancelDownload 取消下载
func (dm *DownloadManager) CancelDownload(ctx context.Context, taskID string) error {
	return dm.manager.CancelTask(taskID)
}

// ResumeDownload 恢复下载
func (dm *DownloadManager) ResumeDownload(ctx context.Context, taskID string, offset int64) error {
	// 检查任务是否存在
	task, err := dm.manager.GetTask(taskID)
	if err != nil {
		return err
	}

	if task.Type != TransferTypeDownload {
		return NewTransferError(ErrCodeInvalidParameters, "task is not a download task", nil)
	}

	// 检查是否支持断点续传
	if !dm.manager.config.ResumeSupport {
		return NewTransferError(ErrCodeInvalidParameters, "resume is not supported", nil)
	}

	// 设置偏移量
	task.Transferred = offset

	// 恢复任务
	return dm.manager.ResumeTask(taskID)
}

// VerifyDownload 验证下载
func (dm *DownloadManager) VerifyDownload(ctx context.Context, taskID string, localChecksum string) (bool, error) {
	// 获取任务
	task, err := dm.manager.GetTask(taskID)
	if err != nil {
		return false, err
	}

	// 如果没有预期校验和，跳过验证
	if task.Checksum == "" {
		return true, nil
	}

	// 比较校验和
	return compareChecksum(localChecksum, task.Checksum), nil
}

// ListFiles 列出文件
func (dm *DownloadManager) ListFiles(ctx context.Context, prefix string, limit int) ([]FileMeta, error) {
	return dm.manager.storage.List(ctx, prefix, limit)
}

// GetFileInfo 获取文件信息
func (dm *DownloadManager) GetFileInfo(ctx context.Context, filePath string) (*FileMeta, error) {
	meta, err := dm.manager.storage.Stat(ctx, filePath)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// GetQuotaInfo 获取配额信息
func (dm *DownloadManager) GetQuotaInfo(ctx context.Context, userID string) (*QuotaInfo, error) {
	return dm.manager.GetUserQuotaInfo(ctx, userID)
}

// ErrRecordNotFound 记录不存在错误
var ErrRecordNotFound = fmt.Errorf("record not found")
