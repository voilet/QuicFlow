package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SysUser 用户模型
type SysUser struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UUID        uuid.UUID `json:"uuid" gorm:"index;unique;comment:用户UUID"`
	Username    string    `json:"username" gorm:"size:50;uniqueIndex;not null;comment:用户登录名"`
	Password    string    `json:"-" gorm:"size:255;not null;comment:用户登录密码"`
	NickName    string    `json:"nick_name" gorm:"size:50;default:系统用户;comment:用户昵称"`
	HeaderImg   string    `json:"header_img" gorm:"size:500;default:;comment:用户头像"`
	AuthorityID uint      `json:"authority_id" gorm:"default:888;comment:用户角色ID"`
	Phone       string    `json:"phone" gorm:"size:20;comment:用户手机号"`
	Email       string    `json:"email" gorm:"size:100;comment:用户邮箱"`
	Enable      uint      `json:"enable" gorm:"default:1;comment:用户是否被冻结 1正常 2冻结"`

	// 关联 - 不使用 struct 引用避免 GORM 创建错误的外键
	// Authority   SysAuthority   `gorm:"-"`  // 需要时使用 Preload 加载
	// Authorities []SysAuthority `gorm:"-"`  // many2many 关联已通过 SysUserAuthority 显式定义
}

// TableName 指定表名
func (SysUser) TableName() string {
	return "sys_users"
}

// Login 登录接口
type Login interface {
	GetUsername() string
	GetNickname() string
	GetUUID() uuid.UUID
	GetUserId() uint
	GetAuthorityId() uint
}

var _ Login = new(SysUser)

// GetUsername 获取用户名
func (s *SysUser) GetUsername() string {
	return s.Username
}

// GetNickname 获取昵称
func (s *SysUser) GetNickname() string {
	return s.NickName
}

// GetUUID 获取UUID
func (s *SysUser) GetUUID() uuid.UUID {
	return s.UUID
}

// GetUserId 获取用户ID
func (s *SysUser) GetUserId() uint {
	return s.ID
}

// GetAuthorityId 获取角色ID
func (s *SysUser) GetAuthorityId() uint {
	return s.AuthorityID
}

// SysUserAuthority 用户角色关联表
type SysUserAuthority struct {
	SysUserId              uint `gorm:"primaryKey;comment:用户ID"`
	SysAuthorityAuthorityId uint `gorm:"primaryKey;comment:角色ID"`
}

// TableName 指定表名
func (SysUserAuthority) TableName() string {
	return "sys_user_authorities"
}
