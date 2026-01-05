package models

import (
	"time"

	"gorm.io/gorm"
)

// SysApi API模型
type SysApi struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Path        string `json:"path" gorm:"size:200;comment:api路径"`
	Description string `json:"description" gorm:"size:200;comment:api中文描述"`
	ApiGroup    string `json:"api_group" gorm:"size:50;comment:api组"`
	Method      string `json:"method" gorm:"size:10;default:POST;comment:方法"`
}

// TableName 指定表名
func (SysApi) TableName() string {
	return "sys_apis"
}
