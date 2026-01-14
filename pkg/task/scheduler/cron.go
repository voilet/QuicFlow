package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/voilet/quic-flow/pkg/task/models"
	"github.com/voilet/quic-flow/pkg/monitoring"
)

// CronScheduler 基于 robfig/cron 的调度器
type CronScheduler struct {
	cron           *cron.Cron
	logger         *monitoring.Logger
	taskDispatcher *TaskDispatcher
	jobRegistry    map[cron.EntryID]int64 // EntryID -> TaskID
	registryMu     sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewCronScheduler 创建调度器
func NewCronScheduler(logger *monitoring.Logger, dispatcher *TaskDispatcher) *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// 使用秒级精度，支持 6 段 Cron 表达式
	c := cron.New(
		cron.WithSeconds(), // 支持秒级调度
	)

	return &CronScheduler{
		cron:           c,
		logger:         logger,
		taskDispatcher: dispatcher,
		jobRegistry:    make(map[cron.EntryID]int64),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start 启动调度器
func (s *CronScheduler) Start() {
	s.cron.Start()
	s.logger.Info("Cron scheduler started")
}

// Stop 停止调度器
func (s *CronScheduler) Stop() {
	s.cancel()
	ctx := s.cron.Stop()

	select {
	case <-ctx.Done():
		s.logger.Info("Cron scheduler stopped gracefully")
	case <-time.After(10 * time.Second):
		s.logger.Warn("Cron scheduler stop timeout")
	}
}

// AddTask 添加任务到调度器
func (s *CronScheduler) AddTask(task *models.Task) (cron.EntryID, error) {
	if task.Status != models.TaskStatusEnabled {
		return 0, fmt.Errorf("task %d is disabled", task.ID)
	}

	// 验证 Cron 表达式
	if err := s.validateCronExpr(task.CronExpr); err != nil {
		return 0, fmt.Errorf("invalid cron expression: %w", err)
	}

	// 创建 Job
	job := s.createJob(task)

	// 添加到调度器
	entryID, err := s.cron.AddFunc(task.CronExpr, job)
	if err != nil {
		return 0, fmt.Errorf("failed to add cron job: %w", err)
	}

	// 注册
	s.registryMu.Lock()
	s.jobRegistry[entryID] = task.ID
	s.registryMu.Unlock()

	s.logger.Info("Task added to scheduler",
		"task_id", task.ID,
		"task_name", task.Name,
		"cron_expr", task.CronExpr)

	return entryID, nil
}

// RemoveTask 从调度器移除任务
func (s *CronScheduler) RemoveTask(taskID int64) error {
	s.registryMu.Lock()
	defer s.registryMu.Unlock()

	// 查找 EntryID
	var targetID cron.EntryID
	for entryID, id := range s.jobRegistry {
		if id == taskID {
			targetID = entryID
			delete(s.jobRegistry, entryID)
			break
		}
	}

	if targetID == 0 {
		return fmt.Errorf("task %d not found in registry", taskID)
	}

	s.cron.Remove(targetID)
	s.logger.Info("Task removed from scheduler", "task_id", taskID)

	return nil
}

// UpdateTask 更新任务
func (s *CronScheduler) UpdateTask(task *models.Task) error {
	// 先移除旧任务
	if err := s.RemoveTask(task.ID); err != nil {
		// 如果任务不在注册表中，忽略错误
		s.logger.Debug("Task not in registry, skipping remove", "task_id", task.ID)
	}

	// 如果启用则重新添加
	if task.Status == models.TaskStatusEnabled {
		_, err := s.AddTask(task)
		return err
	}

	return nil
}

// createJob 创建执行函数
func (s *CronScheduler) createJob(task *models.Task) func() {
	return func() {
		// 在独立 goroutine 中执行，避免阻塞调度器
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("Task panic",
						"task_id", task.ID,
						"panic", r)
				}
			}()

			// 执行任务分发
			if err := s.taskDispatcher.Dispatch(s.ctx, task); err != nil {
				s.logger.Error("Failed to dispatch task",
					"task_id", task.ID,
					"error", err)
			}
		}()
	}
}

// validateCronExpr 验证 Cron 表达式
func (s *CronScheduler) validateCronExpr(expr string) error {
	// 使用 cron 解析器验证（支持秒级表达式）
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := parser.Parse(expr)
	return err
}

// GetNextRunTime 获取任务下次执行时间
func (s *CronScheduler) GetNextRunTime(taskID int64) (time.Time, error) {
	s.registryMu.RLock()
	defer s.registryMu.RUnlock()

	for entryID, id := range s.jobRegistry {
		if id == taskID {
			entry := s.cron.Entry(entryID)
			if entry.ID == 0 {
				return time.Time{}, fmt.Errorf("entry not found")
			}
			return entry.Next, nil
		}
	}

	return time.Time{}, fmt.Errorf("task %d not found in registry", taskID)
}
