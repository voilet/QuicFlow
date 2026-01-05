package models

import (
	"time"

	"gorm.io/gorm"
)

// SysJwtBlacklist JWT黑名单
type SysJwtBlacklist struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Jwt string `json:"jwt" gorm:"size:1000;comment:jwt"`
}

// TableName 指定表名
func (SysJwtBlacklist) TableName() string {
	return "sys_jwt_blacklists"
}

// IsInBlacklist 检查JWT是否在黑名单中
func IsInBlacklist(db *gorm.DB, jwt string) bool {
	var count int64
	db.Model(&SysJwtBlacklist{}).Where("jwt = ? AND deleted_at is null", jwt).Count(&count)
	return count > 0
}

// JoinBlacklist 将JWT加入黑名单
func JoinBlacklist(db *gorm.DB, jwt string) error {
	return db.Create(&SysJwtBlacklist{Jwt: jwt}).Error
}

// GetRedisJWT 从Redis获取JWT (用于多点登录)
func GetRedisJWT(userName string) (string, error) {
	// TODO: 实现Redis获取
	return "", nil
}

// SetRedisJWT 设置JWT到Redis
func SetRedisJWT(userName, jwt string) error {
	// TODO: 实现Redis设置
	return nil
}
