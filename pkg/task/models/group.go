package models

import (
	"time"

	"gorm.io/gorm"
)

// TaskGroup 主机分组表
type TaskGroup struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:64;uniqueIndex;not null;comment:分组名称" json:"name"`
	Description string    `gorm:"size:256;comment:分组描述" json:"description"`
	Tags        string    `gorm:"size:256;comment:标签(逗号分隔)" json:"tags"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Tasks []Task `gorm:"many2many:tb_task_group_relation;" json:"tasks,omitempty"`
}

// TableName 指定表名
func (TaskGroup) TableName() string {
	return "tb_task_group"
}

// TaskGroupRelation 任务分组关联表
type TaskGroupRelation struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID    int64     `gorm:"not null;index:idx_task_group;comment:任务ID" json:"task_id"`
	GroupID   int64     `gorm:"not null;index:idx_task_group;comment:分组ID" json:"group_id"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 指定表名
func (TaskGroupRelation) TableName() string {
	return "tb_task_group_relation"
}
