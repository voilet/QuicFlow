package models

import (
	"time"

	"gorm.io/gorm"
)

// SysAuthority 角色模型
type SysAuthority struct {
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	AuthorityId   uint           `json:"authority_id" gorm:"primaryKey;comment:角色ID"`
	AuthorityName string         `json:"authority_name" gorm:"size:100;uniqueIndex;comment:角色名"`
	ParentId      *uint          `json:"parent_id" gorm:"comment:父角色ID"`
	DefaultRouter string         `json:"default_router" gorm:"size:100;default:dashboard;comment:默认菜单"`

	// 关联
	DataAuthorityId []*SysAuthority `json:"data_authority_id" gorm:"many2many:sys_data_authority_ids;"`
	Children        []SysAuthority  `json:"children" gorm:"-"`
	SysBaseMenus    []SysBaseMenu   `json:"menus" gorm:"many2many:sys_authority_menus;"`
	// Users 字段不需要，它通过 many2many 关联，不创建外键
}

// TableName 指定表名
func (SysAuthority) TableName() string {
	return "sys_authorities"
}

// SysAuthorityMenu 角色菜单关联表
type SysAuthorityMenu struct {
	SysAuthorityAuthorityId uint `gorm:"primaryKey;comment:角色ID"`
	SysBaseMenuId            uint `gorm:"primaryKey;comment:菜单ID"`
}

// TableName 指定表名
func (SysAuthorityMenu) TableName() string {
	return "sys_authority_menus"
}
