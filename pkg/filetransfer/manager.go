package filetransfer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Manager 传输管理器
type Manager struct {
	config       *Config
	storage      StorageBackend
	tracker      *ProgressTracker
	db           *gorm.DB
	activeTasks  sync.Map // taskID -> *TransferTask
	taskQueue    chan *TransferTask
	workers      int
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
}

// NewManager 创建传输管理器
func NewManager(config *Config, storage StorageBackend, db *gorm.DB) (*Manager, error) {
	fmt.Printf("[DEBUG] NewManager: starting\n")
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		config:    config,
		storage:   storage,
		tracker:   NewProgressTracker(),
		db:        db,
		taskQueue: make(chan *TransferTask, 100),
		workers:   config.MaxConcurrentTransfers,
		ctx:       ctx,
		cancel:    cancel,
	}

	fmt.Printf("[DEBUG] NewManager: starting %d workers\n", m.workers)
	// 启动工作协程
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}
	fmt.Printf("[DEBUG] NewManager: workers started\n")

	return m, nil
}

// Start 启动管理器
func (m *Manager) Start() error {
	return m.restorePendingTasks()
}

// Stop 停止管理器
func (m *Manager) Stop() {
	m.cancel()
	close(m.taskQueue)
	m.wg.Wait()
	m.tracker.Close()
}

// SubmitTask 提交传输任务
func (m *Manager) SubmitTask(task *TransferTask) (*TransferTask, error) {
	fmt.Printf("[DEBUG] SubmitTask: starting\n")

	// 验证任务
	fmt.Printf("[DEBUG] SubmitTask: validating task...\n")
	if err := m.validateTask(task); err != nil {
		return nil, err
	}
	fmt.Printf("[DEBUG] SubmitTask: validation passed\n")

	// 设置任务ID
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	fmt.Printf("[DEBUG] SubmitTask: task ID=%s\n", task.ID)

	// 创建进度追踪
	m.tracker.Create(task.ID, task.FileSize)
	fmt.Printf("[DEBUG] SubmitTask: tracker created\n")

	// 保存到数据库
	fmt.Printf("[DEBUG] SubmitTask: saving to DB...\n")
	if err := m.saveTaskToDB(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}
	fmt.Printf("[DEBUG] SubmitTask: saved to DB\n")

	// 添加到活跃任务
	m.activeTasks.Store(task.ID, task)

	// 加入队列
	fmt.Printf("[DEBUG] SubmitTask: adding to queue...\n")
	select {
	case m.taskQueue <- task:
		task.Status = TaskStatusPending
		m.tracker.SetStatus(task.ID, TaskStatusPending)
		fmt.Printf("[DEBUG] SubmitTask: added to queue\n")
	default:
		return nil, fmt.Errorf("task queue is full")
	}

	return task, nil
}

// GetTask 获取任务
func (m *Manager) GetTask(taskID string) (*TransferTask, error) {
	if task, ok := m.activeTasks.Load(taskID); ok {
		return task.(*TransferTask), nil
	}

	// 从数据库加载
	return m.loadTaskFromDB(taskID)
}

// CancelTask 取消任务
func (m *Manager) CancelTask(taskID string) error {
	if task, ok := m.activeTasks.Load(taskID); ok {
		t := task.(*TransferTask)
		if t.cancelChan != nil {
			close(t.cancelChan)
		}
		t.Status = TaskStatusCancelled
		m.tracker.SetStatus(taskID, TaskStatusCancelled)
		m.updateTaskInDB(t)
		m.activeTasks.Delete(taskID)
		return nil
	}
	return ErrTaskNotFound
}

// PauseTask 暂停任务
func (m *Manager) PauseTask(taskID string) error {
	if task, ok := m.activeTasks.Load(taskID); ok {
		t := task.(*TransferTask)
		if t.Status != TaskStatusTransferring {
			return fmt.Errorf("task is not transferring")
		}
		t.Status = TaskStatusPaused
		m.tracker.SetStatus(taskID, TaskStatusPaused)
		m.updateTaskInDB(t)
		return nil
	}
	return ErrTaskNotFound
}

// ResumeTask 恢复任务
func (m *Manager) ResumeTask(taskID string) error {
	if task, ok := m.activeTasks.Load(taskID); ok {
		t := task.(*TransferTask)
		if t.Status != TaskStatusPaused {
			return fmt.Errorf("task is not paused")
		}
		t.Status = TaskStatusPending
		m.tracker.SetStatus(taskID, TaskStatusPending)
		m.updateTaskInDB(t)
		// 重新加入队列
		select {
		case m.taskQueue <- t:
		default:
			return fmt.Errorf("task queue is full")
		}
		return nil
	}
	return ErrTaskNotFound
}

// GetProgress 获取任务进度
func (m *Manager) GetProgress(taskID string) (ProgressUpdate, bool) {
	return m.tracker.GetProgressUpdate(taskID)
}

// SubscribeProgress 订阅任务进度
func (m *Manager) SubscribeProgress(taskID string) <-chan ProgressUpdate {
	return m.tracker.Subscribe(taskID)
}

// UnsubscribeProgress 取消订阅
func (m *Manager) UnsubscribeProgress(taskID string, ch <-chan ProgressUpdate) {
	m.tracker.Unsubscribe(taskID, ch)
}

// worker 工作协程
func (m *Manager) worker(id int) {
	fmt.Printf("[DEBUG] Worker %d: starting\n", id)
	defer m.wg.Done()

	for {
		select {
		case task, ok := <-m.taskQueue:
			if !ok {
				fmt.Printf("[DEBUG] Worker %d: channel closed, exiting\n", id)
				return
			}
			fmt.Printf("[DEBUG] Worker %d: received task %s\n", id, task.ID)
			m.processTask(task)
			fmt.Printf("[DEBUG] Worker %d: finished task %s\n", id, task.ID)
		case <-m.ctx.Done():
			fmt.Printf("[DEBUG] Worker %d: context done, exiting\n", id)
			return
		}
	}
}

// processTask 处理任务
func (m *Manager) processTask(task *TransferTask) {
	fmt.Printf("[DEBUG] processTask: starting task %s\n", task.ID)

	// 检查是否已取消或暂停
	if task.Status == TaskStatusCancelled || task.Status == TaskStatusPaused {
		fmt.Printf("[DEBUG] processTask: task %s is cancelled/paused, returning\n", task.ID)
		return
	}

	fmt.Printf("[DEBUG] processTask: setting status to Transferring...\n")
	task.Status = TaskStatusTransferring
	now := time.Now()
	task.StartedAt = &now

	fmt.Printf("[DEBUG] processTask: calling tracker.SetStatus...\n")
	m.tracker.SetStatus(task.ID, TaskStatusTransferring)
	fmt.Printf("[DEBUG] processTask: tracker.SetStatus returned\n")

	var err error
	fmt.Printf("[DEBUG] processTask: processing %s task...\n", task.Type)
	switch task.Type {
	case TransferTypeUpload:
		err = m.processUpload(task)
	case TransferTypeDownload:
		err = m.processDownload(task)
	}
	fmt.Printf("[DEBUG] processTask: processing completed with error: %v\n", err)

	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err
		fmt.Printf("[DEBUG] processTask: calling tracker.SetError...\n")
		m.tracker.SetError(task.ID)
		fmt.Printf("[DEBUG] processTask: tracker.SetError returned\n")
	} else {
		task.Status = TaskStatusCompleted
		now = time.Now()
		task.CompletedAt = &now
		fmt.Printf("[DEBUG] processTask: calling tracker.SetComplete...\n")
		m.tracker.SetComplete(task.ID)
		fmt.Printf("[DEBUG] processTask: tracker.SetComplete returned\n")
	}

	fmt.Printf("[DEBUG] processTask: updating task in DB...\n")
	m.updateTaskInDB(task)
	fmt.Printf("[DEBUG] processTask: deleting from activeTasks...\n")
	m.activeTasks.Delete(task.ID)
	fmt.Printf("[DEBUG] processTask: task %s completed\n", task.ID)
}

// processUpload 处理上传任务
func (m *Manager) processUpload(task *TransferTask) error {
	fmt.Printf("[DEBUG] processUpload: task %s, waiting for chunks via QUIC/HTTP\n", task.ID)

	// 上传任务通过 QUIC/HTTP 协议处理分块
	// 这个函数等待 doneChan 信号，实际的上传由 UploadManager 处理
	// 当所有分块接收完成后，UploadManager.CompleteUpload 会关闭 doneChan

	// 等待上传完成、取消或超时
	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	select {
	case <-timeout.C:
		return fmt.Errorf("upload timeout: no chunks received within 5 minutes")
	case <-task.cancelChan:
		return fmt.Errorf("upload cancelled")
	case <-task.doneChan:
		fmt.Printf("[DEBUG] processUpload: task %s completed via doneChan\n", task.ID)
		return nil
	}
}

// processDownload 处理下载任务
func (m *Manager) processDownload(task *TransferTask) error {
	// TODO: 实现下载逻辑
	// 这里会在 download.go 中实现
	return fmt.Errorf("download not implemented")
}

// validateTask 验证任务
func (m *Manager) validateTask(task *TransferTask) error {
	if task.FileName == "" {
		return NewTransferError(ErrCodeInvalidParameters, "filename is required", nil)
	}

	if task.FileSize <= 0 {
		return NewTransferError(ErrCodeInvalidParameters, "file size must be positive", nil)
	}

	// 检查文件大小限制
	if m.config.MaxFileSize > 0 && task.FileSize > m.config.MaxFileSize {
		return NewTransferError(ErrCodeFileTooLarge,
			fmt.Sprintf("file size %d exceeds maximum %d", task.FileSize, m.config.MaxFileSize), nil)
	}

	return nil
}

// saveTaskToDB 保存任务到数据库
func (m *Manager) saveTaskToDB(task *TransferTask) error {
	// 解析用户ID
	var userID uuid.UUID
	var err error
	if task.UserID == "anonymous" {
		userID = uuid.Nil
	} else {
		userID, err = uuid.Parse(task.UserID)
		if err != nil {
			return fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// 解析任务ID
	taskID, err := uuid.Parse(task.ID)
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	// 准备数据库记录
	now := time.Now()

	// 将 map[string]interface{} 转换为 datatypes.JSON
	var metadataJSON datatypes.JSON
	if task.Metadata != nil {
		metadataBytes, _ := json.Marshal(task.Metadata)
		metadataJSON = datatypes.JSON(metadataBytes)
	}

	ft := &FileTransfer{
		TaskID:          taskID,
		FileName:        task.FileName,
		FilePath:        task.DestPath,
		FileSize:        task.FileSize,
		FileHash:        task.Checksum,
		TransferType:    string(task.Type),
		Status:          string(task.Status),
		Progress:        int(task.Progress),
		BytesTransferred: task.Transferred,
		UserID:          userID,
		ClientIP:        task.ClientIP,
		Metadata:        metadataJSON,
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if task.StartedAt != nil {
		ft.StartedAt = *task.StartedAt
	}
	if task.CompletedAt != nil {
		ft.CompletedAt = task.CompletedAt
	}
	if task.Error != nil {
		ft.ErrorMessage = task.Error.Error()
	}

	// 保存到数据库
	if err := m.db.Create(ft).Error; err != nil {
		return fmt.Errorf("failed to save task to database: %w", err)
	}

	return nil
}

// updateTaskInDB 更新数据库中的任务
func (m *Manager) updateTaskInDB(task *TransferTask) error {
	updates := map[string]interface{}{
		"status":           string(task.Status),
		"progress":         int(task.Progress),
		"bytes_transferred": task.Transferred,
		"updated_at":       time.Now(),
	}

	if task.Status == TaskStatusCompleted && task.CompletedAt != nil {
		updates["completed_at"] = *task.CompletedAt
	}

	if task.Status == TaskStatusFailed && task.Error != nil {
		updates["error_message"] = task.Error.Error()
	}

	return m.db.Model(&FileTransfer{}).
		Where("task_id = ?", uuid.MustParse(task.ID)).
		Updates(updates).Error
}

// loadTaskFromDB 从数据库加载任务
func (m *Manager) loadTaskFromDB(taskID string) (*TransferTask, error) {
	var ft FileTransfer
	err := m.db.Where("task_id = ?", uuid.MustParse(taskID)).First(&ft).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to load task: %w", err)
	}

	// 解析 Metadata 从 datatypes.JSON 到 map[string]interface{}
	var metadata map[string]interface{}
	if len(ft.Metadata) > 0 {
		if err := json.Unmarshal(ft.Metadata, &metadata); err != nil {
			// 如果解析失败，使用空 map
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	task := &TransferTask{
		ID:          ft.TaskID.String(),
		Type:        TransferType(ft.TransferType),
		FileName:    ft.FileName,
		DestPath:    ft.FilePath,
		FileSize:    ft.FileSize,
		Transferred: ft.BytesTransferred,
		Status:      TaskStatus(ft.Status),
		Checksum:    ft.FileHash,
		UserID:      ft.UserID.String(),
		ClientIP:    ft.ClientIP,
		Metadata:    metadata,
		Progress:    float64(ft.Progress),
	}

	if !ft.StartedAt.IsZero() {
		task.StartedAt = &ft.StartedAt
	}
	if ft.CompletedAt != nil {
		task.CompletedAt = ft.CompletedAt
	}
	if ft.ErrorMessage != "" {
		task.Error = fmt.Errorf(ft.ErrorMessage)
	}

	return task, nil
}

// restorePendingTasks 恢复未完成的任务
func (m *Manager) restorePendingTasks() error {
	var fts []FileTransfer
	err := m.db.Where("status IN (?)", []string{
		string(TaskStatusPending),
		string(TaskStatusTransferring),
		string(TaskStatusPaused),
	}).Find(&fts).Error
	if err != nil {
		return fmt.Errorf("failed to restore pending tasks: %w", err)
	}

	for _, ft := range fts {
		// 解析 Metadata
		var metadata map[string]interface{}
		if len(ft.Metadata) > 0 {
			if err := json.Unmarshal(ft.Metadata, &metadata); err != nil {
				metadata = make(map[string]interface{})
			}
		} else {
			metadata = make(map[string]interface{})
		}

		task := &TransferTask{
			ID:          ft.TaskID.String(),
			Type:        TransferType(ft.TransferType),
			FileName:    ft.FileName,
			DestPath:    ft.FilePath,
			FileSize:    ft.FileSize,
			Transferred: ft.BytesTransferred,
			Status:      TaskStatus(ft.Status),
			Checksum:    ft.FileHash,
			UserID:      ft.UserID.String(),
			ClientIP:    ft.ClientIP,
			Metadata:    metadata,
			cancelChan:  make(chan struct{}),
		}

		m.activeTasks.Store(task.ID, task)
		m.tracker.Create(task.ID, ft.FileSize)

		// 如果是pending或transferring状态，重新加入队列
		if ft.Status == string(TaskStatusPending) ||
			ft.Status == string(TaskStatusTransferring) {
			select {
			case m.taskQueue <- task:
			default:
				// 队列满，标记为失败
				task.Status = TaskStatusFailed
				task.Error = fmt.Errorf("failed to restore task: queue full")
				m.updateTaskInDB(task)
			}
		}
	}

	return nil
}

// GetActiveTasksCount 获取活跃任务数量
func (m *Manager) GetActiveTasksCount() int {
	count := 0
	m.activeTasks.Range(func(_, value interface{}) bool {
		task := value.(*TransferTask)
		if task.Status == TaskStatusTransferring ||
			task.Status == TaskStatusPending {
			count++
		}
		return true
	})
	return count
}

// GetUserQuotaInfo 获取用户配额信息
func (m *Manager) GetUserQuotaInfo(ctx context.Context, userID string) (*QuotaInfo, error) {
	if ls, ok := m.storage.(*LocalStorage); ok {
		return ls.GetUserQuotaInfo(ctx, userID)
	}
	return nil, fmt.Errorf("quota info not supported for this storage backend")
}

// GetSystemConfig 获取系统配置
func (m *Manager) GetSystemConfig() *SystemConfig {
	return &SystemConfig{
		Upload: UploadConfigLimits{
			MaxFileSize:          m.config.MaxFileSize,
			MaxConcurrentUploads: m.workers,
			ChunkSize:            m.config.ChunkSize,
			SupportedFormats:     []string{"*"},
			ChecksumRequired:     m.config.ChecksumVerify,
		},
		Download: DownloadConfigLimits{
			MaxConcurrentDownloads: m.workers,
			ChunkSize:             m.config.ChunkSize,
			ResumeSupport:         m.config.ResumeSupport,
			MultiThreadSupport:    m.config.DownloadThreads > 1,
			MaxThreads:            m.config.DownloadThreads,
		},
		Storage: StorageConfig{
			RetentionDays:       m.config.RetentionDays,
			AutoCleanup:         m.config.AutoCleanup,
			CompressionAvailable: m.config.Compression,
		},
		Quotas: QuotaConfig{
			UserQuota:    m.config.UserQuota,
			ProjectQuota: m.config.StorageQuota,
		},
	}
}
