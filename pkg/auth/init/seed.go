package init

import (
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/voilet/quic-flow/pkg/auth/models"
)

// SeedAuthData 初始化权限系统种子数据
func SeedAuthData(db *gorm.DB) error {
	// 首先检查数据完整性，清理可能存在的孤儿数据
	// 如果有用户引用了不存在的 authority_id，需要先删除这些用户
	db.Exec("DELETE FROM sys_users WHERE authority_id NOT IN (SELECT authority_id FROM sys_authorities WHERE deleted_at IS NULL)")

	// 检查是否已有管理员用户
	var adminUser models.SysUser
	result := db.Where("username = ?", "admin").First(&adminUser)

	if result.Error == nil {
		// 管理员已存在，检查角色是否存在
		var authority models.SysAuthority
		if err := db.First(&authority, adminUser.AuthorityID).Error; err == nil {
			// 数据完整，跳过初始化
			return nil
		}
		// 角色不存在，需要清理并重新初始化
		fmt.Println("检测到不完整的数据，正在清理...")
		clearAuthData(db)
	} else if result.Error != gorm.ErrRecordNotFound {
		// 数据库查询错误
		return result.Error
	}

	// 使用事务
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 先创建角色（必须在用户之前）
		authorities := []models.SysAuthority{
			{
				AuthorityId:   1,
				AuthorityName: "超级管理员",
				DefaultRouter: "dashboard",
			},
			{
				AuthorityId:   888,
				AuthorityName: "普通用户",
				DefaultRouter: "dashboard",
			},
		}

		for _, authority := range authorities {
			// 使用 OnConflict 处理已存在的情况
			if err := tx.Where("authority_id = ?", authority.AuthorityId).
				Assign(authority).
				FirstOrCreate(&authority).Error; err != nil {
				return fmt.Errorf("创建角色失败: %w", err)
			}
		}

		// 2. 创建默认管理员用户
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		adminUser = models.SysUser{
			UUID:        uuid.New(),
			Username:    "admin",
			Password:    string(hashedPassword),
			NickName:    "超级管理员",
			HeaderImg:   "",
			AuthorityID: 1,
			Enable:      1,
		}

		if err := tx.Where("username = ?", "admin").
			Assign(adminUser).
			FirstOrCreate(&adminUser).Error; err != nil {
			return fmt.Errorf("创建管理员用户失败: %w", err)
		}

		// 3. 创建基础菜单
		menus := []models.SysBaseMenu{
			{
				ParentId:    0,
				Path:        "/dashboard",
				Name:        "Dashboard",
				Hidden:      false,
				Component:   "views/Dashboard.vue",
				Sort:        1,
				Title:       "控制台",
				Icon:        "Odometer",
				KeepAlive:   true,
				DefaultMenu: true,
			},
			{
				ParentId:    0,
				Path:        "/clients",
				Name:        "ClientList",
				Hidden:      false,
				Component:   "views/ClientList.vue",
				Sort:        2,
				Title:       "客户端管理",
				Icon:        "Monitor",
				KeepAlive:   true,
			},
			{
				ParentId:  0,
				Path:       "/command",
				Name:       "Command",
				Hidden:     false,
				Component:  "views/CommandSend.vue",
				Sort:       3,
				Title:      "命令管理",
				Icon:       "Terminal",
				KeepAlive:  false,
			},
			{
				ParentId:  3,
				Path:       "/command/send",
				Name:       "CommandSend",
				Hidden:     false,
				Component:  "views/CommandSend.vue",
				Sort:       1,
				Title:      "命令发送",
				Icon:       "Promotion",
				KeepAlive:  false,
			},
			{
				ParentId:  3,
				Path:       "/command/history",
				Name:       "CommandHistory",
				Hidden:     false,
				Component:  "views/CommandHistory.vue",
				Sort:       2,
				Title:      "命令历史",
				Icon:       "Clock",
				KeepAlive:  true,
			},
			{
				ParentId:    0,
				Path:        "/terminal",
				Name:        "Terminal",
				Hidden:      false,
				Component:   "views/Terminal.vue",
				Sort:        4,
				Title:       "SSH终端",
				Icon:        "Platform",
				KeepAlive:   false,
			},
			{
				ParentId:    0,
				Path:        "/audit",
				Name:        "Audit",
				Hidden:      false,
				Component:   "views/AuditLog.vue",
				Sort:        5,
				Title:       "命令审计",
				Icon:        "Document",
				KeepAlive:   true,
			},
			{
				ParentId:    0,
				Path:        "/release",
				Name:        "Release",
				Hidden:      false,
				Component:   "views/Release.vue",
				Sort:        6,
				Title:       "发布管理",
				Icon:        "Rocket",
				KeepAlive:   true,
			},
		}

		for _, menu := range menus {
			if err := tx.Where("path = ? AND name = ?", menu.Path, menu.Name).
				Assign(menu).
				FirstOrCreate(&menu).Error; err != nil {
				return fmt.Errorf("创建菜单失败: %w", err)
			}
		}

		// 重新获取菜单以获得正确的ID
		var createdMenus []models.SysBaseMenu
		if err := tx.Order("id").Find(&createdMenus).Error; err != nil {
			return fmt.Errorf("查询菜单失败: %w", err)
		}

		// 4. 为超级管理员分配所有菜单
		for _, menu := range createdMenus {
			var count int64
			tx.Exec(
				"SELECT COUNT(*) FROM sys_authority_menus WHERE sys_authority_authority_id = ? AND sys_base_menu_id = ?",
				1, menu.ID,
			).Count(&count)

			if count == 0 {
				if err := tx.Exec(
					"INSERT INTO sys_authority_menus (sys_authority_authority_id, sys_base_menu_id) VALUES (?, ?)",
					1, menu.ID,
				).Error; err != nil {
					return fmt.Errorf("分配菜单权限失败: %w", err)
				}
			}
		}

		// 5. 创建基础API记录
		apis := []models.SysApi{
			{Path: "/base/login", Description: "用户登录", ApiGroup: "认证", Method: "POST"},
			{Path: "/user/logout", Description: "用户登出", ApiGroup: "认证", Method: "POST"},
			{Path: "/user/info", Description: "获取用户信息", ApiGroup: "用户", Method: "GET"},
			{Path: "/user/list", Description: "用户列表", ApiGroup: "用户", Method: "GET"},
			{Path: "/user/create", Description: "创建用户", ApiGroup: "用户", Method: "POST"},
			{Path: "/user/update", Description: "更新用户", ApiGroup: "用户", Method: "PUT"},
			{Path: "/user/delete", Description: "删除用户", ApiGroup: "用户", Method: "DELETE"},
			{Path: "/authority/list", Description: "角色列表", ApiGroup: "角色", Method: "GET"},
			{Path: "/authority/create", Description: "创建角色", ApiGroup: "角色", Method: "POST"},
			{Path: "/authority/update", Description: "更新角色", ApiGroup: "角色", Method: "PUT"},
			{Path: "/authority/delete", Description: "删除角色", ApiGroup: "角色", Method: "DELETE"},
			{Path: "/menu/list", Description: "菜单列表", ApiGroup: "菜单", Method: "GET"},
			{Path: "/menu/create", Description: "创建菜单", ApiGroup: "菜单", Method: "POST"},
			{Path: "/menu/update", Description: "更新菜单", ApiGroup: "菜单", Method: "PUT"},
			{Path: "/menu/delete", Description: "删除菜单", ApiGroup: "菜单", Method: "DELETE"},
			{Path: "/clients", Description: "客户端列表", ApiGroup: "客户端", Method: "GET"},
			{Path: "/command", Description: "发送命令", ApiGroup: "命令", Method: "POST"},
			{Path: "/command/multi", Description: "多播命令", ApiGroup: "命令", Method: "POST"},
			{Path: "/commands", Description: "命令列表", ApiGroup: "命令", Method: "GET"},
			{Path: "/terminal/*", Description: "终端接口", ApiGroup: "终端", Method: "GET"},
			{Path: "/audit/*", Description: "审计接口", ApiGroup: "审计", Method: "GET"},
			{Path: "/recordings/*", Description: "录像接口", ApiGroup: "录像", Method: "GET"},
			{Path: "/release/*", Description: "发布接口", ApiGroup: "发布", Method: "GET"},
			{Path: "/release/*", Description: "发布接口", ApiGroup: "发布", Method: "POST"},
			{Path: "/release/*", Description: "发布接口", ApiGroup: "发布", Method: "PUT"},
			{Path: "/release/*", Description: "发布接口", ApiGroup: "发布", Method: "DELETE"},
		}

		for _, api := range apis {
			if err := tx.Where("path = ? AND method = ?", api.Path, api.Method).
				Assign(api).
				FirstOrCreate(&api).Error; err != nil {
				return fmt.Errorf("创建API记录失败: %w", err)
			}
		}

		return nil
	})
}

// ClearAuthData 清理权限系统数据（导出供外部使用）
func ClearAuthData(db *gorm.DB) {
	clearAuthData(db)
}

// clearAuthData 清理权限系统数据
func clearAuthData(db *gorm.DB) {
	// 按依赖关系逆序删除
	db.Exec("DELETE FROM sys_authority_menus")
	db.Exec("DELETE FROM sys_user_authorities")
	db.Exec("DELETE FROM sys_users")
	db.Exec("DELETE FROM sys_authorities")
	db.Exec("DELETE FROM sys_base_menus")
	db.Exec("DELETE FROM sys_base_menu_parameters")
	db.Exec("DELETE FROM sys_apis")
	db.Exec("DELETE FROM casbin_rule")
}

// GetSeededAuthUser 获取种子管理员用户（用于首次登录）
func GetSeededAuthUser(db *gorm.DB) *models.SysUser {
	var user models.SysUser
	db.Where("username = ?", "admin").First(&user)
	return &user
}

// IsSeededDataExists 检查种子数据是否已存在
func IsSeededDataExists(db *gorm.DB) bool {
	var count int64
	db.Model(&models.SysUser{}).Count(&count)
	return count > 0
}
