package batch

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/voilet/quic-flow/pkg/callback"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/protocol"
)

// BatchExecutor 批量任务执行器
// 支持向大量客户端并发发送命令，带进度跟踪和流控
type BatchExecutor struct {
	// 服务器发送接口
	sender MessageSender

	// 配置
	config *BatchConfig

	// 运行状态
	activeJobs sync.Map // jobID -> *BatchJob
	jobCount   atomic.Int64

	// 日志
	logger *monitoring.Logger

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
}

// MessageSender 消息发送接口
type MessageSender interface {
	// SendTo 发送消息到指定客户端
	SendTo(clientID string, msg *protocol.DataMessage) error
	// SendToWithPromise 发送消息并等待响应
	SendToWithPromise(clientID string, msg *protocol.DataMessage, timeout time.Duration) (*callback.Promise, error)
	// ListClients 获取所有在线客户端
	ListClients() []string
}

// BatchConfig 批量执行器配置
type BatchConfig struct {
	// 并发控制
	MaxConcurrency int // 最大并发数（默认 5000）

	// 超时配置
	TaskTimeout time.Duration // 单任务超时（默认 30s）
	JobTimeout  time.Duration // 整体任务超时（默认 10min）

	// 重试配置
	MaxRetries    int           // 最大重试次数（默认 2）
	RetryInterval time.Duration // 重试间隔（默认 1s）

	// 进度回调
	OnProgress func(job *BatchJob)     // 进度更新回调
	OnComplete func(job *BatchJob)     // 任务完成回调
	OnError    func(clientID string, err error) // 单客户端错误回调

	// 日志
	Logger *monitoring.Logger
}

// BatchJob 批量任务
type BatchJob struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	// 任务内容
	Command     string          `json:"command"`
	Payload     json.RawMessage `json:"payload"`
	TargetClients []string      `json:"target_clients"` // 空表示所有客户端

	// 执行状态
	Status      BatchJobStatus `json:"status"`
	TotalCount  int64          `json:"total_count"`
	SuccessCount int64         `json:"success_count"`
	FailedCount int64          `json:"failed_count"`
	PendingCount int64         `json:"pending_count"`

	// 结果
	Results sync.Map // clientID -> *TaskResult
	Errors  sync.Map // clientID -> error

	// 内部控制
	startTime time.Time
	endTime   time.Time
	mu        sync.RWMutex
}

// BatchJobStatus 任务状态
type BatchJobStatus string

const (
	BatchJobPending   BatchJobStatus = "pending"
	BatchJobRunning   BatchJobStatus = "running"
	BatchJobCompleted BatchJobStatus = "completed"
	BatchJobFailed    BatchJobStatus = "failed"
	BatchJobCancelled BatchJobStatus = "cancelled"
)

// TaskResult 单任务结果
type TaskResult struct {
	ClientID   string          `json:"client_id"`
	Success    bool            `json:"success"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	Duration   time.Duration   `json:"duration"`
	RetryCount int             `json:"retry_count"`
}

// NewBatchExecutor 创建批量执行器
func NewBatchExecutor(sender MessageSender, config *BatchConfig) *BatchExecutor {
	if config == nil {
		config = &BatchConfig{}
	}

	// 默认值
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 5000
	}
	if config.TaskTimeout <= 0 {
		config.TaskTimeout = 30 * time.Second
	}
	if config.JobTimeout <= 0 {
		config.JobTimeout = 10 * time.Minute
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 2
	}
	if config.RetryInterval <= 0 {
		config.RetryInterval = time.Second
	}
	if config.Logger == nil {
		config.Logger = monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BatchExecutor{
		sender: sender,
		config: config,
		logger: config.Logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Execute 执行批量任务
// command: 命令类型
// payload: 命令参数
// targetClients: 目标客户端列表（空表示所有在线客户端）
// waitForResult: 是否等待执行结果
func (e *BatchExecutor) Execute(command string, payload json.RawMessage, targetClients []string, waitForResult bool) (*BatchJob, error) {
	// 确定目标客户端
	if len(targetClients) == 0 {
		targetClients = e.sender.ListClients()
	}

	if len(targetClients) == 0 {
		return nil, fmt.Errorf("no target clients available")
	}

	// 创建任务
	job := &BatchJob{
		ID:            uuid.New().String(),
		CreatedAt:     time.Now(),
		Command:       command,
		Payload:       payload,
		TargetClients: targetClients,
		Status:        BatchJobPending,
		TotalCount:    int64(len(targetClients)),
		PendingCount:  int64(len(targetClients)),
	}

	// 存储任务
	e.activeJobs.Store(job.ID, job)
	e.jobCount.Add(1)

	e.logger.Info("Batch job created",
		"job_id", job.ID,
		"command", command,
		"target_count", len(targetClients),
		"wait_for_result", waitForResult)

	// 异步执行
	go e.executeJob(job, waitForResult)

	return job, nil
}

// ExecuteSync 同步执行批量任务（等待完成）
func (e *BatchExecutor) ExecuteSync(ctx context.Context, command string, payload json.RawMessage, targetClients []string) (*BatchJob, error) {
	job, err := e.Execute(command, payload, targetClients, true)
	if err != nil {
		return nil, err
	}

	// 等待完成
	for {
		select {
		case <-ctx.Done():
			e.CancelJob(job.ID)
			return job, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			job.mu.RLock()
			status := job.Status
			job.mu.RUnlock()

			if status == BatchJobCompleted || status == BatchJobFailed || status == BatchJobCancelled {
				return job, nil
			}
		}
	}
}

// executeJob 执行批量任务
func (e *BatchExecutor) executeJob(job *BatchJob, waitForResult bool) {
	job.mu.Lock()
	job.Status = BatchJobRunning
	job.startTime = time.Now()
	job.mu.Unlock()

	// 创建任务超时 context
	ctx, cancel := context.WithTimeout(e.ctx, e.config.JobTimeout)
	defer cancel()

	// 使用信号量控制并发
	sem := make(chan struct{}, e.config.MaxConcurrency)
	var wg sync.WaitGroup

	e.logger.Info("Batch job started",
		"job_id", job.ID,
		"total_clients", len(job.TargetClients),
		"max_concurrency", e.config.MaxConcurrency)

	// 向所有目标客户端发送任务
	for _, clientID := range job.TargetClients {
		select {
		case <-ctx.Done():
			e.logger.Warn("Batch job timeout or cancelled", "job_id", job.ID)
			job.mu.Lock()
			job.Status = BatchJobCancelled
			job.mu.Unlock()
			goto done
		case sem <- struct{}{}:
			wg.Add(1)
			go func(cid string) {
				defer wg.Done()
				defer func() { <-sem }()

				e.executeTask(ctx, job, cid, waitForResult)
			}(clientID)
		}
	}

	wg.Wait()

done:
	// 更新最终状态
	job.mu.Lock()
	job.endTime = time.Now()
	if job.Status == BatchJobRunning {
		if job.FailedCount == 0 {
			job.Status = BatchJobCompleted
		} else if job.SuccessCount == 0 {
			job.Status = BatchJobFailed
		} else {
			job.Status = BatchJobCompleted // 部分成功也算完成
		}
	}
	job.mu.Unlock()

	duration := job.endTime.Sub(job.startTime)
	e.logger.Info("Batch job completed",
		"job_id", job.ID,
		"status", job.Status,
		"total", job.TotalCount,
		"success", job.SuccessCount,
		"failed", job.FailedCount,
		"duration", duration,
		"rate", fmt.Sprintf("%.2f/s", float64(job.SuccessCount+job.FailedCount)/duration.Seconds()))

	// 完成回调
	if e.config.OnComplete != nil {
		e.config.OnComplete(job)
	}

	// 清理任务（延迟删除，保留一段时间供查询）
	go func() {
		time.Sleep(5 * time.Minute)
		e.activeJobs.Delete(job.ID)
		e.jobCount.Add(-1)
	}()
}

// executeTask 执行单个任务
func (e *BatchExecutor) executeTask(ctx context.Context, job *BatchJob, clientID string, waitForResult bool) {
	startTime := time.Now()
	result := &TaskResult{
		ClientID: clientID,
	}

	var lastErr error
	for retry := 0; retry <= e.config.MaxRetries; retry++ {
		if retry > 0 {
			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				goto failed
			case <-time.After(e.config.RetryInterval):
			}
		}

		result.RetryCount = retry

		// 构造命令消息
		msg := &protocol.DataMessage{
			MsgId:      uuid.New().String(),
			SenderId:   "server",
			ReceiverId: clientID,
			Type:       protocol.MessageType_MESSAGE_TYPE_COMMAND,
			Payload:    e.buildPayload(job.Command, job.Payload),
			Timestamp:  time.Now().UnixMilli(),
			WaitAck:    waitForResult,
		}

		if waitForResult {
			// 等待响应
			promise, err := e.sender.SendToWithPromise(clientID, msg, e.config.TaskTimeout)
			if err != nil {
				lastErr = err
				continue
			}

			// 等待结果
			select {
			case <-ctx.Done():
				lastErr = ctx.Err()
				goto failed
			case resp := <-promise.RespChan:
				if resp.Error != nil {
					lastErr = resp.Error
					continue
				}
				if resp.AckMessage != nil {
					result.Success = resp.AckMessage.Status == protocol.AckStatus_ACK_STATUS_SUCCESS
					result.Result = resp.AckMessage.Result
					if !result.Success && resp.AckMessage.Error != "" {
						lastErr = fmt.Errorf(resp.AckMessage.Error)
						continue
					}
				}
				goto success
			case <-time.After(e.config.TaskTimeout):
				lastErr = fmt.Errorf("timeout waiting for response")
				continue
			}
		} else {
			// 不等待响应
			if err := e.sender.SendTo(clientID, msg); err != nil {
				lastErr = err
				continue
			}
			result.Success = true
			goto success
		}
	}

failed:
	result.Success = false
	result.Error = lastErr.Error()
	result.Duration = time.Since(startTime)
	job.Results.Store(clientID, result)
	job.Errors.Store(clientID, lastErr)
	atomic.AddInt64(&job.FailedCount, 1)
	atomic.AddInt64(&job.PendingCount, -1)

	if e.config.OnError != nil {
		e.config.OnError(clientID, lastErr)
	}

	e.reportProgress(job)
	return

success:
	result.Duration = time.Since(startTime)
	job.Results.Store(clientID, result)
	atomic.AddInt64(&job.SuccessCount, 1)
	atomic.AddInt64(&job.PendingCount, -1)

	e.reportProgress(job)
}

// buildPayload 构建命令 payload
func (e *BatchExecutor) buildPayload(command string, payload json.RawMessage) []byte {
	data := map[string]interface{}{
		"command_type": command,
		"payload":      payload,
	}
	result, _ := json.Marshal(data)
	return result
}

// reportProgress 报告进度
func (e *BatchExecutor) reportProgress(job *BatchJob) {
	if e.config.OnProgress == nil {
		return
	}

	// 每 1% 或每 100 个任务报告一次
	completed := atomic.LoadInt64(&job.SuccessCount) + atomic.LoadInt64(&job.FailedCount)
	if completed%100 == 0 || completed == job.TotalCount {
		e.config.OnProgress(job)
	}
}

// GetJob 获取任务状态
func (e *BatchExecutor) GetJob(jobID string) (*BatchJob, bool) {
	if val, ok := e.activeJobs.Load(jobID); ok {
		return val.(*BatchJob), true
	}
	return nil, false
}

// CancelJob 取消任务
func (e *BatchExecutor) CancelJob(jobID string) bool {
	if val, ok := e.activeJobs.Load(jobID); ok {
		job := val.(*BatchJob)
		job.mu.Lock()
		if job.Status == BatchJobPending || job.Status == BatchJobRunning {
			job.Status = BatchJobCancelled
			job.mu.Unlock()
			e.logger.Info("Batch job cancelled", "job_id", jobID)
			return true
		}
		job.mu.Unlock()
	}
	return false
}

// ListJobs 获取所有活跃任务
func (e *BatchExecutor) ListJobs() []*BatchJob {
	var jobs []*BatchJob
	e.activeJobs.Range(func(key, value interface{}) bool {
		jobs = append(jobs, value.(*BatchJob))
		return true
	})
	return jobs
}

// GetStats 获取统计信息
func (e *BatchExecutor) GetStats() *BatchStats {
	stats := &BatchStats{
		ActiveJobs: e.jobCount.Load(),
	}

	e.activeJobs.Range(func(key, value interface{}) bool {
		job := value.(*BatchJob)
		switch job.Status {
		case BatchJobPending:
			stats.PendingJobs++
		case BatchJobRunning:
			stats.RunningJobs++
		case BatchJobCompleted:
			stats.CompletedJobs++
		case BatchJobFailed:
			stats.FailedJobs++
		}
		stats.TotalTasks += job.TotalCount
		stats.CompletedTasks += job.SuccessCount + job.FailedCount
		return true
	})

	return stats
}

// BatchStats 批量执行统计
type BatchStats struct {
	ActiveJobs     int64 `json:"active_jobs"`
	PendingJobs    int64 `json:"pending_jobs"`
	RunningJobs    int64 `json:"running_jobs"`
	CompletedJobs  int64 `json:"completed_jobs"`
	FailedJobs     int64 `json:"failed_jobs"`
	TotalTasks     int64 `json:"total_tasks"`
	CompletedTasks int64 `json:"completed_tasks"`
}

// Stop 停止执行器
func (e *BatchExecutor) Stop() {
	e.cancel()
	e.logger.Info("Batch executor stopped")
}
