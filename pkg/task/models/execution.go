package models

import (
	"time"

	"gorm.io/gorm"
)

// ExecutionType 执行类型
type ExecutionType int

const (
	ExecutionTypeScheduled ExecutionType = 1 // 定时执行
	ExecutionTypeManual    ExecutionType = 2 // 手动触发
)

// ExecutionStatus 执行状态
type ExecutionStatus int

const (
	ExecutionStatusPending   ExecutionStatus = 1 // 待执行
	ExecutionStatusRunning   ExecutionStatus = 2 // 执行中
	ExecutionStatusSuccess   ExecutionStatus = 3 // 成功
	ExecutionStatusFailed    ExecutionStatus = 4 // 失败
	ExecutionStatusTimeout   ExecutionStatus = 5 // 超时
	ExecutionStatusCancelled ExecutionStatus = 6 // 已取消
)

// Execution 任务执行记录表
type Execution struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID        int64          `gorm:"not null;index:idx_task_id;comment:任务ID" json:"task_id"`
	TaskName      string         `gorm:"size:128;comment:任务名称(冗余)" json:"task_name"`
	ClientID      string         `gorm:"size:64;not null;index:idx_client_id;comment:客户端ID" json:"client_id"`
	GroupID       *int64         `gorm:"index:idx_group_id;comment:分组ID" json:"group_id,omitempty"`
	ExecutionType ExecutionType  `gorm:"not null;default:1;comment:执行类型:1=定时,2=手动" json:"execution_type"`
	Status        ExecutionStatus `gorm:"not null;index:idx_status;comment:状态:1=Pending,2=Running,3=Success,4=Failed,5=Timeout,6=Cancelled" json:"status"`
	StartTime     *time.Time     `gorm:"index:idx_start_time;comment:开始时间" json:"start_time,omitempty"`
	EndTime       *time.Time     `gorm:"comment:结束时间" json:"end_time,omitempty"`
	Duration      int            `gorm:"comment:执行耗时(毫秒)" json:"duration"`
	ExitCode      int            `gorm:"comment:退出码" json:"exit_code"`
	Output        string         `gorm:"type:text;comment:执行输出" json:"output"`
	ErrorMsg      string         `gorm:"type:text;comment:错误信息" json:"error_msg"`
	RetryCount    int            `gorm:"not null;default:0;comment:重试次数" json:"retry_count"`
	CreatedAt     time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

// TableName 指定表名
func (Execution) TableName() string {
	return "tb_execution"
}

// IsFinished 检查执行是否已完成
func (e *Execution) IsFinished() bool {
	return e.Status == ExecutionStatusSuccess ||
		e.Status == ExecutionStatusFailed ||
		e.Status == ExecutionStatusTimeout ||
		e.Status == ExecutionStatusCancelled
}

// CalculateDuration 计算执行耗时（毫秒）
func (e *Execution) CalculateDuration() int {
	if e.StartTime == nil || e.EndTime == nil {
		return 0
	}
	return int(e.EndTime.Sub(*e.StartTime).Milliseconds())
}
