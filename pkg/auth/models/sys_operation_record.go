package models

import (
	"time"

	"gorm.io/gorm"
)

// SysOperationRecord 操作记录
type SysOperationRecord struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Ip     string `json:"ip" gorm:"size:50;comment:客户端IP"`
	Method string `json:"method" gorm:"size:10;comment:请求方法"`
	Path   string `json:"path" gorm:"size:200;comment:请求路径"`
	Status int    `json:"status" gorm:"comment:响应状态码"`
	Latency int64  `json:"latency" gorm:"comment:延迟(毫秒)"`
	// Agent信息
	Agent string `json:"agent" gorm:"size:500;comment:用户代理"`
	// 用户信息
	UserId   uint   `json:"user_id" gorm:"index;comment:用户ID"`
	UserName string `json:"user_name" gorm:"size:50;comment:用户名"`
	// 错误信息
	ErrorMsg string `json:"error_msg" gorm:"type:text;comment:错误信息"`
}

// TableName 指定表名
func (SysOperationRecord) TableName() string {
	return "sys_operation_records"
}
