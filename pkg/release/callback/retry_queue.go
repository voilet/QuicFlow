package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"
	"gorm.io/gorm"
)

// RetryTask 重试任务
type RetryTask struct {
	ID            string                 `json:"id"`
	CallbackID    string                 `json:"callback_id"`
	Channel       models.CallbackChannel `json:"channel"`
	Payload       models.CallbackPayload `json:"payload"`
	AttemptCount  int                    `json:"attempt_count"`
	MaxAttempts   int                    `json:"max_attempts"`
	NextRetryTime time.Time              `json:"next_retry_time"`
	CreatedAt     time.Time              `json:"created_at"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts    int           `json:"max_attempts"`    // 最大重试次数
	InitialBackoff time.Duration `json:"initial_backoff"` // 初始退避时间
	MaxBackoff     time.Duration `json:"max_backoff"`     // 最大退避时间
	BackoffFactor  float64       `json:"backoff_factor"`  // 退避因子
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:    5,
	InitialBackoff: 5 * time.Second,
	MaxBackoff:     10 * time.Minute,
	BackoffFactor:  2.0,
}

// RetryQueue 回调重试队列
type RetryQueue struct {
	db            *gorm.DB
	queue         chan *RetryTask
	config        RetryConfig
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	pendingTasks  map[string]*RetryTask
	workerCount   int
	taskHistory   map[string]int // 记录任务的历史尝试次数
	historyMu     sync.RWMutex
}

// NewRetryQueue 创建重试队列
func NewRetryQueue(db *gorm.DB, config RetryConfig) *RetryQueue {
	ctx, cancel := context.WithCancel(context.Background())

	rq := &RetryQueue{
		db:           db,
		queue:        make(chan *RetryTask, 1000),
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		pendingTasks: make(map[string]*RetryTask),
		taskHistory:  make(map[string]int),
		workerCount:  3, // 默认 3 个工作线程
	}

	return rq
}

// Start 启动重试队列
func (rq *RetryQueue) Start() {
	// 启动工作线程
	for i := 0; i < rq.workerCount; i++ {
		rq.wg.Add(1)
		go rq.worker(i)
	}

	// 启动定时检查线程，用于调度即将到期的任务
	rq.wg.Add(1)
	go rq.scheduler()
}

// Stop 停止重试队列
func (rq *RetryQueue) Stop() {
	rq.cancel()
	close(rq.queue)
	rq.wg.Wait()
}

// Enqueue 将回调任务加入重试队列
func (rq *RetryQueue) Enqueue(task *RetryTask) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// 检查任务是否已在队列中
	taskKey := rq.getTaskKey(task.CallbackID, string(task.Channel.Type), string(task.Payload.EventType))
	rq.historyMu.RLock()
	attempts := rq.taskHistory[taskKey]
	rq.historyMu.RUnlock()

	if attempts >= rq.config.MaxAttempts {
		return fmt.Errorf("task has reached max retry attempts: %d", attempts)
	}

	// 设置初始重试时间（如果未设置）
	if task.NextRetryTime.IsZero() {
		task.NextRetryTime = time.Now().Add(rq.calculateBackoff(task.AttemptCount))
	}

	task.ID = generateRetryTaskID()
	task.CreatedAt = time.Now()

	// 更新任务历史记录
	rq.historyMu.Lock()
	rq.taskHistory[taskKey] = attempts + 1
	rq.historyMu.Unlock()

	// 加入队列
	rq.mu.Lock()
	rq.pendingTasks[task.ID] = task
	rq.mu.Unlock()

	select {
	case rq.queue <- task:
		return nil
	default:
		// 队列已满
		rq.mu.Lock()
		delete(rq.pendingTasks, task.ID)
		rq.mu.Unlock()
		return fmt.Errorf("retry queue is full")
	}
}

// worker 工作线程，处理重试任务
func (rq *RetryQueue) worker(id int) {
	defer rq.wg.Done()

	for {
		select {
		case <-rq.ctx.Done():
			return
		case task, ok := <-rq.queue:
			if !ok {
				return
			}
			rq.processTask(task, id)
		}
	}
}

// scheduler 调度器，定期检查待处理的任务
func (rq *RetryQueue) scheduler() {
	defer rq.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rq.ctx.Done():
			return
		case <-ticker.C:
			rq.checkPendingTasks()
		}
	}
}

// checkPendingTasks 检查并处理到期的待处理任务
func (rq *RetryQueue) checkPendingTasks() {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	now := time.Now()
	for _, task := range rq.pendingTasks {
		if now.After(task.NextRetryTime) || now.Equal(task.NextRetryTime) {
			// 从待处理队列移除（避免重复处理）
			delete(rq.pendingTasks, task.ID)

			// 重新入队进行处理
			select {
			case rq.queue <- task:
			default:
				// 队列已满，放回待处理队列
				rq.pendingTasks[task.ID] = task
			}
		}
	}
}

// processTask 处理单个重试任务
func (rq *RetryQueue) processTask(task *RetryTask, workerID int) {
	// 记录重试历史
	rq.recordRetryHistory(task, workerID)

	// 创建回调管理器来执行回调
	callbackManager := NewManager(rq.db)
	err := callbackManager.SendCallback(task.Channel, task.Payload)

	if err == nil {
		// 回调成功，从历史记录中清除
		taskKey := rq.getTaskKey(task.CallbackID, string(task.Channel.Type), string(task.Payload.EventType))
		rq.historyMu.Lock()
		delete(rq.taskHistory, taskKey)
		rq.historyMu.Unlock()

		// 更新回调历史状态
		rq.updateHistoryStatus(task, "success", nil)
		return
	}

	// 回调失败，判断是否需要继续重试
	task.AttemptCount++

	if task.AttemptCount >= task.MaxAttempts {
		// 达到最大重试次数，放弃重试
		taskKey := rq.getTaskKey(task.CallbackID, string(task.Channel.Type), string(task.Payload.EventType))
		rq.historyMu.Lock()
		delete(rq.taskHistory, taskKey)
		rq.historyMu.Unlock()

		// 更新回调历史状态为失败
		rq.updateHistoryStatus(task, "failed", err)
		return
	}

	// 计算下次重试时间并重新入队
	task.NextRetryTime = time.Now().Add(rq.calculateBackoff(task.AttemptCount))

	rq.mu.Lock()
	rq.pendingTasks[task.ID] = task
	rq.mu.Unlock()

	// 更新回调历史状态为重试中
	rq.updateHistoryStatus(task, "retrying", err)
}

// calculateBackoff 计算退避时间（指数退避）
func (rq *RetryQueue) calculateBackoff(attempt int) time.Duration {
	backoff := float64(rq.config.InitialBackoff) * math.Pow(rq.config.BackoffFactor, float64(attempt))

	if backoff > float64(rq.config.MaxBackoff) {
		backoff = float64(rq.config.MaxBackoff)
	}

	return time.Duration(backoff)
}

// getTaskKey 生成任务唯一标识
func (rq *RetryQueue) getTaskKey(callbackID, channelType, eventType string) string {
	return fmt.Sprintf("%s:%s:%s", callbackID, channelType, eventType)
}

// recordRetryHistory 记录重试历史
func (rq *RetryQueue) recordRetryHistory(task *RetryTask, workerID int) {
	history := &models.CallbackHistory{
		ID:         generateHistoryID(),
		ConfigID:   task.CallbackID,
		EventType:  task.Payload.EventType,
		Channel:    task.Channel.Type,
		Status:     models.CallbackStatusRetrying,
		Request:    task.Payload, // 直接使用原始 payload
		Response:   fmt.Sprintf("Retry attempt %d by worker %d", task.AttemptCount+1, workerID),
		RetryCount: task.AttemptCount + 1,
		CreatedAt:  time.Now(),
	}

	rq.db.Create(history)
}

// updateHistoryStatus 更新历史记录状态
func (rq *RetryQueue) updateHistoryStatus(task *RetryTask, status string, err error) {
	// 查找最新的重试记录并更新
	var history models.CallbackHistory
	rq.db.Where("config_id = ? AND event_type = ? AND channel = ?", task.CallbackID, string(task.Payload.EventType), string(task.Channel.Type)).
		Order("created_at DESC").
		First(&history)

	if history.ID != "" {
		history.Status = models.CallbackStatus(status)
		if err != nil {
			history.Error = err.Error()
		}
		rq.db.Save(&history)
	}
}

// buildRequestHistory 构建请求历史记录
func (rq *RetryQueue) buildRequestHistory(task *RetryTask) map[string]interface{} {
	requestData := map[string]interface{}{
		"retry_attempt":   task.AttemptCount + 1,
		"max_attempts":    task.MaxAttempts,
		"next_retry_time": task.NextRetryTime,
		"created_at":      task.CreatedAt,
	}

	// 添加 payload 信息
	payloadBytes, _ := json.Marshal(task.Payload)
	requestData["payload"] = string(payloadBytes)

	return requestData
}

// GetQueueStatus 获取队列状态
func (rq *RetryQueue) GetQueueStatus() map[string]interface{} {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	rq.historyMu.RLock()
	totalHistory := len(rq.taskHistory)
	rq.historyMu.RUnlock()

	return map[string]interface{}{
		"pending_tasks":    len(rq.pendingTasks),
		"queue_capacity":   cap(rq.queue),
		"queue_length":     len(rq.queue),
		"worker_count":     rq.workerCount,
		"tracked_attempts": totalHistory,
	}
}

// generateRetryTaskID 生成重试任务 ID
func generateRetryTaskID() string {
	return fmt.Sprintf("retry-%d", time.Now().UnixNano())
}

// generateHistoryID 生成历史记录 ID
func generateHistoryID() string {
	return fmt.Sprintf("history-%d", time.Now().UnixNano())
}
