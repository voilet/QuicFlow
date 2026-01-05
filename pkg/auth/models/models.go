package models

import (
	"gorm.io/gorm"
)

// AllAuthModels 所有权限系统模型列表
var AllAuthModels = []interface{}{
	&SysUser{},
	&SysAuthority{},
	&SysUserAuthority{},
	&SysBaseMenu{},
	&SysBaseMenuParameter{},
	&SysAuthorityMenu{},
	&SysApi{},
	&SysJwtBlacklist{},
	&SysOperationRecord{},
}

// AutoMigrateAuth 自动迁移权限系统表
func AutoMigrateAuth(db *gorm.DB) error {
	return db.AutoMigrate(AllAuthModels...)
}
