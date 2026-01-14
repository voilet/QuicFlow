package callback

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"
	"gorm.io/gorm"
)

// 默认并发配置
const (
	DefaultMaxConcurrentCallbacks = 10 // 默认最大并发回调数
	DefaultCallbackTimeout        = 30 * time.Second
)

// Manager 回调管理器
type Manager struct {
	db             *gorm.DB
	sender         *CallbackSender
	retryQueue     *RetryQueue
	retryConfig    RetryConfig
	semaphore      chan struct{} // 并发控制信号量
	maxConcurrent  int           // 最大并发数
	callbackTimeout time.Duration // 回调超时时间
}

// NewManager 创建回调管理器
func NewManager(db *gorm.DB) *Manager {
	m := &Manager{
		db:     db,
		sender: NewCallbackSender(),
		retryConfig: RetryConfig{
			MaxAttempts:    5,
			InitialBackoff: 5 * time.Second,
			MaxBackoff:     10 * time.Minute,
			BackoffFactor:  2.0,
		},
		maxConcurrent:   DefaultMaxConcurrentCallbacks,
		callbackTimeout: DefaultCallbackTimeout,
		semaphore:       make(chan struct{}, DefaultMaxConcurrentCallbacks),
	}

	// 创建并启动重试队列
	m.retryQueue = NewRetryQueue(db, m.retryConfig)
	m.retryQueue.Start()

	return m
}

// NewManagerWithRetryConfig 创建带自定义重试配置的回调管理器
func NewManagerWithRetryConfig(db *gorm.DB, retryConfig RetryConfig) *Manager {
	maxConcurrent := DefaultMaxConcurrentCallbacks
	if retryConfig.WorkerCount > 0 {
		maxConcurrent = retryConfig.WorkerCount * 2 // 并发数为worker数的2倍
	}

	m := &Manager{
		db:              db,
		sender:          NewCallbackSender(),
		retryConfig:     retryConfig,
		maxConcurrent:   maxConcurrent,
		callbackTimeout: DefaultCallbackTimeout,
		semaphore:       make(chan struct{}, maxConcurrent),
	}

	// 创建并启动重试队列
	m.retryQueue = NewRetryQueue(db, retryConfig)
	m.retryQueue.Start()

	return m
}

// Close 关闭管理器，释放资源
func (m *Manager) Close() {
	if m.retryQueue != nil {
		m.retryQueue.Stop()
	}
}

// GetRetryQueueStatus 获取重试队列状态
func (m *Manager) GetRetryQueueStatus() map[string]interface{} {
	if m.retryQueue != nil {
		return m.retryQueue.GetQueueStatus()
	}
	return nil
}

// SendCallback 公开方法：发送单个回调（供重试队列使用）
func (m *Manager) SendCallback(channel models.CallbackChannel, payload models.CallbackPayload) error {
	return m.sendCallback(channel, payload)
}

// TriggerCallback 触发回调
func (m *Manager) TriggerCallback(ctx context.Context, task *models.DeployTask, project *models.Project, version *models.Version, eventType models.CallbackEventType, environment string) error {
	// 查询项目的回调配置
	var configs []models.CallbackConfig
	if err := m.db.WithContext(ctx).
		Where("project_id = ? AND enabled = ?", task.ProjectID, true).
		Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to query callback configs: %w", err)
	}

	if len(configs) == 0 {
		return nil // 没有配置回调，直接返回
	}

	// 构造回调负载
	payload := m.buildPayload(task, project, version, eventType, environment)

	// 遍历所有配置
	for _, config := range configs {
		// 检查是否订阅了此事件
		if !m.isEventSubscribed(config, eventType) {
			continue
		}

		// 异步发送回调，避免阻塞部署流程
		go m.sendCallbacks(config, payload, task.ID)
	}

	return nil
}

// buildPayload 构造回调负载
func (m *Manager) buildPayload(task *models.DeployTask, project *models.Project, version *models.Version, eventType models.CallbackEventType, environment string) models.CallbackPayload {
	payload := models.CallbackPayload{
		EventType:   eventType,
		Timestamp:   time.Now(),
		Environment: environment,
		Project: models.CallbackProject{
			ID:          project.ID,
			Name:        project.Name,
			Description: project.Description,
		},
		Version: models.CallbackVersion{
			ID:          version.ID,
			Name:        version.Version,
			Description: version.Description,
		},
		Task: models.CallbackTask{
			ID:     task.ID,
			Type:   task.Operation,
			Status: task.Status,
		},
		Deployment: models.CallbackDeployment{
			TotalCount:     task.TotalCount,
			CompletedCount: task.SuccessCount + task.FailedCount,
			FailedCount:    task.FailedCount,
		},
	}

	// 添加主机列表
	if len(task.Results) > 0 {
		payload.Deployment.Hosts = make([]string, 0, len(task.Results))
		for _, r := range task.Results {
			payload.Deployment.Hosts = append(payload.Deployment.Hosts, r.ClientID)
		}
	}

	// 如果是金丝雀事件，添加金丝雀数量
	if task.CanaryEnabled && (eventType == models.CallbackEventCanaryStarted || eventType == models.CallbackEventCanaryCompleted) {
		canaryCount := len(task.ClientIDs) * task.CanaryPercent / 100
		if canaryCount < 1 {
			canaryCount = 1
		}
		payload.Deployment.CanaryCount = canaryCount
	}

	// 计算耗时
	if task.StartedAt != nil {
		if task.FinishedAt != nil {
			duration := task.FinishedAt.Sub(*task.StartedAt)
			payload.Duration = duration.String()
		} else {
			duration := time.Since(*task.StartedAt)
			payload.Duration = duration.String()
		}
	}

	return payload
}

// isEventSubscribed 检查配置是否订阅了指定事件
func (m *Manager) isEventSubscribed(config models.CallbackConfig, eventType models.CallbackEventType) bool {
	for _, event := range config.Events {
		if event == string(eventType) {
			return true
		}
	}
	return false
}

// sendCallbacks 发送回调到所有启用的渠道（带并发控制）
func (m *Manager) sendCallbacks(config models.CallbackConfig, payload models.CallbackPayload, taskID string) {
	var wg sync.WaitGroup

	for _, channel := range config.Channels {
		if !channel.Enabled {
			continue
		}

		// 复制channel变量用于goroutine
		ch := channel

		wg.Add(1)
		go func() {
			defer wg.Done()

			// 获取信号量（限制并发）
			m.semaphore <- struct{}{}
			defer func() { <-m.semaphore }()

			// 发送带超时控制的回调
			m.sendCallbackWithHistory(config, ch, payload, taskID)
		}()
	}

	// 等待所有回调完成
	wg.Wait()
}

// sendCallbackWithHistory 发送单个回调并记录历史
func (m *Manager) sendCallbackWithHistory(config models.CallbackConfig, channel models.CallbackChannel, payload models.CallbackPayload, taskID string) {
	// 记录回调历史（开始）
	history := &models.CallbackHistory{
		TaskID:    taskID,
		ConfigID:  config.ID,
		EventType: payload.EventType,
		Channel:   channel.Type,
		Status:    models.CallbackStatusPending,
		Request:   payload,
	}

	// 保存历史记录
	m.db.Create(history)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), m.callbackTimeout)
	defer cancel()

	// 使用channel执行回调，支持超时
	resultChan := make(chan error, 1)
	startTime := time.Now()

	go func() {
		resultChan <- m.sendCallback(channel, payload)
	}()

	var err error
	select {
	case err = <-resultChan:
		// 正常完成
	case <-ctx.Done():
		// 超时
		err = fmt.Errorf("callback timeout after %v", m.callbackTimeout)
	}

	duration := time.Since(startTime)

	// 更新历史记录
	history.Duration = int(duration.Milliseconds())
	if err != nil {
		history.Status = models.CallbackStatusFailed
		history.Error = err.Error()

		// 回调失败，加入重试队列
		if m.retryQueue != nil {
			retryTask := &RetryTask{
				TaskID:       taskID,
				CallbackID:   config.ID,
				Channel:      channel,
				Payload:      payload,
				AttemptCount: 0,
				MaxAttempts:  m.retryConfig.MaxAttempts,
			}

			if enqueueErr := m.retryQueue.Enqueue(retryTask); enqueueErr != nil {
				// 记录重试队列错误
				history.Error = fmt.Sprintf("%s (retry enqueue failed: %s)", err.Error(), enqueueErr.Error())
			} else {
				history.Status = models.CallbackStatusRetrying
			}
		}
	} else {
		history.Status = models.CallbackStatusSuccess
	}
	m.db.Save(history)
}

// sendCallback 发送单个回调
func (m *Manager) sendCallback(channel models.CallbackChannel, payload models.CallbackPayload) error {
	switch channel.Type {
	case models.CallbackTypeFeishu:
		// 从 channel.Config 中提取配置
		if config, ok := channel.Config.(*models.FeishuCallbackConfig); ok {
			notifier := NewFeishuNotifier(config)
			return notifier.Send(payload)
		}
		return fmt.Errorf("invalid feishu config")

	case models.CallbackTypeDingTalk:
		if config, ok := channel.Config.(*models.DingTalkCallbackConfig); ok {
			notifier := NewDingTalkNotifier(config)
			return notifier.Send(payload)
		}
		return fmt.Errorf("invalid dingtalk config")

	case models.CallbackTypeWeChat:
		if config, ok := channel.Config.(*models.WeChatCallbackConfig); ok {
			notifier := NewWeChatNotifier(config)
			return notifier.Send(payload)
		}
		return fmt.Errorf("invalid wechat config")

	case models.CallbackTypeCustom:
		// 自定义 HTTP 回调
		if config, ok := channel.Config.(*models.CustomCallbackConfig); ok {
			notifier := NewCustomNotifier(config)
			return notifier.Send(payload)
		}
		return fmt.Errorf("invalid custom callback config")

	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// TriggerCanaryStarted 触发金丝雀开始回调
func (m *Manager) TriggerCanaryStarted(ctx context.Context, task *models.DeployTask, project *models.Project, version *models.Version, environment string) error {
	return m.TriggerCallback(ctx, task, project, version, models.CallbackEventCanaryStarted, environment)
}

// TriggerCanaryCompleted 触发金丝雀完成回调
func (m *Manager) TriggerCanaryCompleted(ctx context.Context, task *models.DeployTask, project *models.Project, version *models.Version, environment string) error {
	return m.TriggerCallback(ctx, task, project, version, models.CallbackEventCanaryCompleted, environment)
}

// TriggerFullCompleted 触发全部发布完成回调
func (m *Manager) TriggerFullCompleted(ctx context.Context, task *models.DeployTask, project *models.Project, version *models.Version, environment string) error {
	return m.TriggerCallback(ctx, task, project, version, models.CallbackEventFullCompleted, environment)
}
