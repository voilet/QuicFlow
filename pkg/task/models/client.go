package models

import (
	"time"

	"gorm.io/gorm"
)

// Client 客户端表（扩展现有表结构）
type Client struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ClientID     string    `gorm:"size:64;uniqueIndex;not null;comment:客户端ID" json:"client_id"`
	GroupID      *int64    `gorm:"index:idx_group_id;comment:所属分组ID" json:"group_id,omitempty"`
	Hostname     string    `gorm:"size:128;comment:主机名" json:"hostname"`
	IP           string    `gorm:"size:64;comment:IP地址" json:"ip"`
	TaskVersion  int64     `gorm:"not null;default:0;comment:任务配置版本" json:"task_version"`
	LastHeartbeat *time.Time `gorm:"comment:最后心跳时间" json:"last_heartbeat,omitempty"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Group *TaskGroup `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// TableName 指定表名
func (Client) TableName() string {
	return "tb_client"
}
