package models

import (
	"time"

	"gorm.io/gorm"
)

// SysBaseMenu 菜单模型
type SysBaseMenu struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	MenuLevel uint   `json:"-" gorm:"comment:菜单层级"`
	ParentId  uint   `json:"parent_id" gorm:"default:0;comment:父菜单ID"`
	Path      string `json:"path" gorm:"size:200;comment:路由path"`
	Name      string `json:"name" gorm:"size:50;comment:路由name"`
	Hidden    bool   `json:"hidden" gorm:"default:false;comment:是否在列表隐藏"`
	Component string `json:"component" gorm:"size:200;comment:对应前端文件路径"`
	Sort      int    `json:"sort" gorm:"default:0;comment:排序标记"`

	// Meta 信息 (内嵌字段)
	Title       string `json:"title" gorm:"size:50;comment:菜单名"`
	Icon        string `json:"icon" gorm:"size:50;comment:菜单图标"`
	KeepAlive   bool   `json:"keep_alive" gorm:"default:false;comment:是否缓存"`
	DefaultMenu bool   `json:"default_menu" gorm:"default:false;comment:是否是基础路由"`
	CloseTab    bool   `json:"close_tab" gorm:"default:false;comment:自动关闭tab"`

	// 关联
	SysAuthoritys []SysAuthority        `json:"authoritys" gorm:"many2many:sys_authority_menus;"`
	Children      []SysBaseMenu         `json:"children" gorm:"-"`
	Parameters    []SysBaseMenuParameter `json:"parameters" gorm:"-"`
}

// TableName 指定表名
func (SysBaseMenu) TableName() string {
	return "sys_base_menus"
}

// SysBaseMenuParameter 菜单参数
type SysBaseMenuParameter struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	SysBaseMenuID uint   `json:"sys_base_menu_id" gorm:"comment:菜单ID"`
	Type          string `json:"type" gorm:"size:10;comment:地址栏携带参数为params还是query"`
	Key           string `json:"key" gorm:"size:100;comment:地址栏携带参数的key"`
	Value         string `json:"value" gorm:"size:200;comment:地址栏携带参数的value"`
}

// TableName 指定表名
func (SysBaseMenuParameter) TableName() string {
	return "sys_base_menu_parameters"
}
