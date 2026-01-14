package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/voilet/quic-flow/pkg/task/models"
	"github.com/voilet/quic-flow/pkg/monitoring"
)

func TestCronScheduler_AddTask(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	dispatcher := &TaskDispatcher{} // Mock dispatcher
	scheduler := NewCronScheduler(logger, dispatcher)

	task := &models.Task{
		ID:             1,
		Name:           "test-task",
		CronExpr:       "*/5 * * * * *", // 每5秒执行
		Status:         models.TaskStatusEnabled,
		ExecutorType:   models.ExecutorTypeShell,
		ExecutorConfig: `{"command": "echo hello"}`,
	}

	entryID, err := scheduler.AddTask(task)
	assert.NoError(t, err)
	assert.NotZero(t, entryID)
}

func TestCronScheduler_RemoveTask(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	dispatcher := &TaskDispatcher{}
	scheduler := NewCronScheduler(logger, dispatcher)

	task := &models.Task{
		ID:             1,
		Name:           "test-task",
		CronExpr:       "*/5 * * * * *",
		Status:         models.TaskStatusEnabled,
		ExecutorType:   models.ExecutorTypeShell,
		ExecutorConfig: `{"command": "echo hello"}`,
	}

	entryID, err := scheduler.AddTask(task)
	assert.NoError(t, err)
	assert.NotZero(t, entryID)

	err = scheduler.RemoveTask(task.ID)
	assert.NoError(t, err)

	// 再次移除应该返回错误
	err = scheduler.RemoveTask(task.ID)
	assert.Error(t, err)
}

func TestCronScheduler_UpdateTask(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	dispatcher := &TaskDispatcher{}
	scheduler := NewCronScheduler(logger, dispatcher)

	task := &models.Task{
		ID:             1,
		Name:           "test-task",
		CronExpr:       "*/5 * * * * *",
		Status:         models.TaskStatusEnabled,
		ExecutorType:   models.ExecutorTypeShell,
		ExecutorConfig: `{"command": "echo hello"}`,
	}

	_, err := scheduler.AddTask(task)
	assert.NoError(t, err)

	// 更新任务
	task.CronExpr = "*/10 * * * * *"
	err = scheduler.UpdateTask(task)
	assert.NoError(t, err)
}

func TestCronScheduler_GetNextRunTime(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	dispatcher := &TaskDispatcher{}
	scheduler := NewCronScheduler(logger, dispatcher)
	scheduler.Start()
	defer scheduler.Stop()

	task := &models.Task{
		ID:             1,
		Name:           "test-task",
		CronExpr:       "*/5 * * * * *", // 每5秒执行
		Status:         models.TaskStatusEnabled,
		ExecutorType:   models.ExecutorTypeShell,
		ExecutorConfig: `{"command": "echo hello"}`,
	}

	_, err := scheduler.AddTask(task)
	assert.NoError(t, err)

	// 等待一下让调度器启动
	time.Sleep(100 * time.Millisecond)

	nextRun, err := scheduler.GetNextRunTime(task.ID)
	assert.NoError(t, err)
	assert.False(t, nextRun.IsZero())
	assert.True(t, nextRun.After(time.Now()))
}

func TestCronScheduler_ValidateCronExpr(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	dispatcher := &TaskDispatcher{}
	scheduler := NewCronScheduler(logger, dispatcher)

	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "valid cron expression",
			expr:    "0 * * * * *",
			wantErr: false,
		},
		{
			name:    "invalid cron expression",
			expr:    "invalid",
			wantErr: true,
		},
		{
			name:    "empty expression",
			expr:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scheduler.validateCronExpr(tt.expr)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCronScheduler_DisabledTask(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	dispatcher := &TaskDispatcher{}
	scheduler := NewCronScheduler(logger, dispatcher)

	task := &models.Task{
		ID:             1,
		Name:           "test-task",
		CronExpr:       "*/5 * * * * *",
		Status:         models.TaskStatusDisabled, // 禁用状态
		ExecutorType:   models.ExecutorTypeShell,
		ExecutorConfig: `{"command": "echo hello"}`,
	}

	_, err := scheduler.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}
