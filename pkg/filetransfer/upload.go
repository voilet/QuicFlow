package filetransfer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UploadSession 上传会话
type UploadSession struct {
	manager      *Manager
	task         *TransferTask
	tempFile     *os.File
	tempPath     string
	checksum     *StreamingChecksum
	chunks       map[int64][]byte // 接收到的分块
	committed    int64            // 已提交的字节数
	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// UploadManager 上传管理器
type UploadManager struct {
	manager   *Manager
	sessions  sync.Map // taskID -> *UploadSession
	tempDir   string
}

// NewUploadManager 创建上传管理器
func NewUploadManager(manager *Manager, tempDir string) (*UploadManager, error) {
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &UploadManager{
		manager:  manager,
		tempDir:  tempDir,
	}, nil
}

// InitUpload 初始化上传
func (um *UploadManager) InitUpload(ctx context.Context, req *InitUploadRequest, userID, clientIP string) (*InitUploadResponse, error) {
	fmt.Printf("[DEBUG] InitUpload: starting\n")

	// 验证文件大小
	if um.manager.config.MaxFileSize > 0 && req.FileSize > um.manager.config.MaxFileSize {
		return nil, NewTransferError(ErrCodeFileTooLarge,
			fmt.Sprintf("file size %d exceeds maximum %d", req.FileSize, um.manager.config.MaxFileSize), nil)
	}
	fmt.Printf("[DEBUG] InitUpload: file size validation passed\n")

	// 检查配额
	fmt.Printf("[DEBUG] InitUpload: checking quota...\n")
	if err := um.manager.storage.CheckQuota(ctx, userID, req.FileSize); err != nil {
		return nil, err
	}
	fmt.Printf("[DEBUG] InitUpload: quota check passed\n")

	// 创建任务
	fmt.Printf("[DEBUG] InitUpload: creating task...\n")
	task := &TransferTask{
		Type:         TransferTypeUpload,
		FileName:     req.Filename,
		FileSize:     req.FileSize,
		Checksum:     req.Checksum,
		UserID:       userID,
		ClientIP:     clientIP,
		Options:      req.Options,
		Metadata:     req.Metadata,
		Status:       TaskStatusPending,
		Progress:     0,
		cancelChan:   make(chan struct{}),
		doneChan:     make(chan struct{}),
	}

	// 解析目标路径
	if req.Path != "" {
		task.DestPath = filepath.Join(req.Path, req.Filename)
	} else {
		task.DestPath = task.FileName
	}

	// 提交任务
	fmt.Printf("[DEBUG] InitUpload: submitting task...\n")
	submittedTask, err := um.manager.SubmitTask(task)
	if err != nil {
		fmt.Printf("[DEBUG] InitUpload: SubmitTask failed: %v\n", err)
		return nil, err
	}
	fmt.Printf("[DEBUG] InitUpload: task submitted, ID=%s\n", submittedTask.ID)

	// 创建上传会话
	// 注意：不能使用请求上下文作为父上下文，因为请求完成后会被取消
	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	session := &UploadSession{
		manager:   um.manager,
		task:      submittedTask,
		chunks:    make(map[int64][]byte),
		checksum:  NewStreamingChecksum(ChecksumTypeSHA256),
		ctx:       sessionCtx,
		cancel:    sessionCancel,
	}

	// 创建临时文件
	tempPath := filepath.Join(um.tempDir, submittedTask.ID+".tmp")
	file, err := os.Create(tempPath)
	if err != nil {
		sessionCancel()
		return nil, NewTransferError(ErrCodeStorageError, "failed to create temp file", err)
	}
	session.tempFile = file
	session.tempPath = tempPath

	um.sessions.Store(submittedTask.ID, session)

	return &InitUploadResponse{
		TaskID: submittedTask.ID,
		UploadConfig: UploadConfig{
			QUICUrl:    fmt.Sprintf("quic://localhost:4242/upload/%s", submittedTask.ID),
			ChunkSize:  um.manager.config.ChunkSize,
			MaxRetries: 3,
			Timeout:    300,
		},
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}, nil
}

// UploadChunk 上传分块
func (um *UploadManager) UploadChunk(ctx context.Context, req *UploadChunkRequest) (*UploadChunkResponse, error) {
	fmt.Printf("[DEBUG] UploadChunk: task_id=%s, offset=%d, len=%d\n", req.TaskID, req.Offset, len(req.Data))

	// 获取会话
	sessionValue, ok := um.sessions.Load(req.TaskID)
	if !ok {
		fmt.Printf("[DEBUG] UploadChunk: session not found for task_id=%s\n", req.TaskID)
		return nil, ErrTaskNotFound
	}
	session := sessionValue.(*UploadSession)

	session.mu.Lock()
	defer session.mu.Unlock()

	// 检查是否已取消
	select {
	case <-session.ctx.Done():
		fmt.Printf("[DEBUG] UploadChunk: session context cancelled for task_id=%s\n", req.TaskID)
		return nil, NewTransferError(ErrCodeTransferFailed, "upload cancelled", nil)
	default:
		fmt.Printf("[DEBUG] UploadChunk: session context OK for task_id=%s\n", req.TaskID)
	}

	// 验证偏移量
	if req.Offset < session.committed {
		return nil, NewTransferError(ErrCodeInvalidOffset,
			fmt.Sprintf("offset %d is less than committed %d", req.Offset, session.committed), nil)
	}

	// 写入分块数据
	if _, err := session.tempFile.Write(req.Data); err != nil {
		return nil, NewTransferError(ErrCodeStorageError, "failed to write chunk", err)
	}

	// 更新校验和
	session.checksum.Write(req.Data)

	// 更新已提交字节数
	session.committed = req.Offset + int64(len(req.Data))

	// 更新进度
	um.manager.tracker.Update(req.TaskID, session.committed, TaskStatusTransferring)

	return &UploadChunkResponse{
		Ack:           true,
		Received:      int64(len(req.Data)),
		TotalReceived: session.committed,
		Progress:      float64(session.committed) / float64(session.task.FileSize) * 100,
	}, nil
}

// CompleteUpload 完成上传
func (um *UploadManager) CompleteUpload(ctx context.Context, req *CompleteUploadRequest) (*CompleteUploadResponse, error) {
	fmt.Printf("[DEBUG] CompleteUpload: task_id=%s\n", req.TaskID)

	// 获取会话
	sessionValue, ok := um.sessions.Load(req.TaskID)
	if !ok {
		fmt.Printf("[DEBUG] CompleteUpload: session not found\n")
		return nil, ErrTaskNotFound
	}
	session := sessionValue.(*UploadSession)

	session.mu.Lock()
	defer session.mu.Unlock()

	fmt.Printf("[DEBUG] CompleteUpload: got session, tempPath=%s\n", session.tempPath)

	// 获取文件大小（在关闭文件之前）
	fileInfo, err := session.tempFile.Stat()
	if err != nil {
		fmt.Printf("[DEBUG] CompleteUpload: stat failed: %v\n", err)
		os.Remove(session.tempPath)
		return nil, NewTransferError(ErrCodeStorageError, "failed to stat temp file", err)
	}

	// 关闭临时文件
	if err := session.tempFile.Close(); err != nil {
		fmt.Printf("[DEBUG] CompleteUpload: close failed: %v\n", err)
		return nil, NewTransferError(ErrCodeStorageError, "failed to close temp file", err)
	}

	fmt.Printf("[DEBUG] CompleteUpload: file closed, size=%d\n", fileInfo.Size())

	// 验证文件大小
	if fileInfo.Size() != session.task.FileSize {
		fmt.Printf("[DEBUG] CompleteUpload: size mismatch: expected %d, got %d\n", session.task.FileSize, fileInfo.Size())
		os.Remove(session.tempPath)
		return nil, NewTransferError(ErrCodeInvalidParameters,
			fmt.Sprintf("file size mismatch: expected %d, got %d",
				session.task.FileSize, fileInfo.Size()), nil)
	}

	// 计算校验和
	actualChecksum := session.checksum.Sum()

	// 验证校验和（如果提供了）
	if session.task.Checksum != "" && req.Checksum != "" {
		if !compareChecksum(actualChecksum, session.task.Checksum) {
			os.Remove(session.tempPath)
			return nil, ErrInvalidChecksum
		}
	}

	// 打开临时文件进行传输
	tempFile, err := os.Open(session.tempPath)
	if err != nil {
		os.Remove(session.tempPath)
		return nil, NewTransferError(ErrCodeStorageError, "failed to open temp file", err)
	}
	defer tempFile.Close()

	// 创建进度读取器
	progressReader := NewProgressReader(req.TaskID, um.manager.tracker, session.task.FileSize, tempFile)

	// 存储文件
	metadata := FileMeta{
		Name:        session.task.FileName,
		Size:        session.task.FileSize,
		Checksum:    extractHash(actualChecksum),
		ContentType: "", // TODO: 从请求中获取
		UserID:      session.task.UserID,
		Description: "",
		Tags:        []string{},
	}

	if err := um.manager.storage.Store(ctx, session.task.DestPath, progressReader, metadata); err != nil {
		os.Remove(session.tempPath)
		return nil, err
	}

	// 更新任务状态
	session.task.Status = TaskStatusCompleted
	session.task.Transferred = session.task.FileSize
	session.task.Progress = 100
	now := time.Now()
	session.task.CompletedAt = &now

	um.manager.tracker.SetComplete(req.TaskID)

	// 通知 worker 任务完成
	close(session.task.doneChan)

	// 保存到数据库
	if err := um.manager.updateTaskInDB(session.task); err != nil {
		// 已完成，仅记录错误
		fmt.Printf("Warning: failed to update task in database: %v\n", err)
	}

	// 清理会话
	um.sessions.Delete(req.TaskID)
	os.Remove(session.tempPath)

	return &CompleteUploadResponse{
		TaskID: req.TaskID,
		Status: TaskStatusCompleted,
		FileInfo: FileInfo{
			FileName: session.task.FileName,
			FilePath: session.task.DestPath,
			FileSize: session.task.FileSize,
			Checksum: actualChecksum,
		},
		TransferStats: TransferStats{
			Duration:   time.Since(*session.task.StartedAt).Milliseconds(),
			AvgSpeed:   "", // TODO: 计算平均速度
			TotalBytes: session.task.FileSize,
		},
		CompletedAt: time.Now(),
	}, nil
}

// CancelUpload 取消上传
func (um *UploadManager) CancelUpload(ctx context.Context, taskID string) error {
	sessionValue, ok := um.sessions.Load(taskID)
	if !ok {
		return ErrTaskNotFound
	}
	session := sessionValue.(*UploadSession)

	session.mu.Lock()
	defer session.mu.Unlock()

	// 取消上下文
	session.cancel()

	// 关闭并删除临时文件
	session.tempFile.Close()
	os.Remove(session.tempPath)

	// 更新任务状态
	session.task.Status = TaskStatusCancelled
	um.manager.tracker.SetStatus(taskID, TaskStatusCancelled)
	um.manager.updateTaskInDB(session.task)

	// 删除会话
	um.sessions.Delete(taskID)

	return nil
}

// GetUploadProgress 获取上传进度
func (um *UploadManager) GetUploadProgress(taskID string) (*TransferProgress, bool) {
	return um.manager.tracker.Get(taskID)
}

// ResumeUpload 恢复上传
func (um *UploadManager) ResumeUpload(ctx context.Context, taskID string) error {
	// 检查任务是否存在
	task, err := um.manager.GetTask(taskID)
	if err != nil {
		return err
	}

	if task.Type != TransferTypeUpload {
		return NewTransferError(ErrCodeInvalidParameters, "task is not an upload task", nil)
	}

	// 检查是否支持断点续传
	if !um.manager.config.ResumeSupport {
		return NewTransferError(ErrCodeInvalidParameters, "resume is not supported", nil)
	}

	// 恢复任务
	return um.manager.ResumeTask(taskID)
}

// CleanupExpiredSessions 清理过期会话
func (um *UploadManager) CleanupExpiredSessions(timeout time.Duration) {
	um.sessions.Range(func(key, value interface{}) bool {
		session := value.(*UploadSession)

		// 检查会话是否超时
		if time.Since(*session.task.StartedAt) > timeout {
			// 取消会话
			session.cancel()
			session.tempFile.Close()
			os.Remove(session.tempPath)
			um.sessions.Delete(key)
		}

		return true
	})
}

// GetActiveUploadsCount 获取活跃上传数量
func (um *UploadManager) GetActiveUploadsCount() int {
	count := 0
	um.sessions.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
