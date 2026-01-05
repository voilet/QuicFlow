package auth

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/voilet/quic-flow/pkg/auth/captcha"
	authinit "github.com/voilet/quic-flow/pkg/auth/init"
	"github.com/voilet/quic-flow/pkg/auth/middleware"
	"github.com/voilet/quic-flow/pkg/auth/models"
	"github.com/voilet/quic-flow/pkg/auth/service"
	"gorm.io/gorm"
)

// 全局验证码验证函数
var captchaVerifyFunc func(id, code string) bool

// SetCaptchaVerify 设置验证码验证函数
func SetCaptchaVerify(fn func(id, code string) bool) {
	captchaVerifyFunc = fn
}

// InitDB 初始化权限系统数据库
func InitDB(db *gorm.DB) error {
	fmt.Println("=== 权限系统数据库初始化 ===")

	// 1. 迁移表结构
	fmt.Println("1. 迁移数据库表结构...")
	if err := models.AutoMigrateAuth(db); err != nil {
		return fmt.Errorf("表结构迁移失败: %w", err)
	}
	fmt.Println("   ✓ 表结构迁移完成")

	// 2. 创建种子数据
	fmt.Println("2. 创建种子数据...")
	if err := authinit.SeedAuthData(db); err != nil {
		// 如果种子数据创建失败，可能是数据不完整，清理后重试
		fmt.Println("   清理不完整的数据并重试...")
		authinit.ClearAuthData(db)
		if err := authinit.SeedAuthData(db); err != nil {
			return fmt.Errorf("种子数据创建失败: %w", err)
		}
	}
	fmt.Println("   ✓ 种子数据创建完成")

	// 3. 显示默认账户
	fmt.Println("\n=== 默认账户信息 ===")
	fmt.Println("用户名: admin")
	fmt.Println("密码: admin123")
	fmt.Println("角色: 超级管理员")
	fmt.Println("\n请登录后立即修改密码！")
	fmt.Println("==========================\n")

	return nil
}

// CheckInit 检查数据库是否已初始化
func CheckInit(db *gorm.DB) (bool, error) {
	// 检查 sys_users 表是否存在
	if !db.Migrator().HasTable(&models.SysUser{}) {
		return false, nil
	}

	// 检查是否有数据
	var count int64
	db.Model(&models.SysUser{}).Count(&count)
	return count > 0, nil
}

// RunInitScript 运行初始化脚本（命令行工具）
func RunInitScript(dbConfig string) error {
	fmt.Println("=== QUIC Flow 权限系统初始化 ===")
	fmt.Println()

	// 连接数据库
	fmt.Println("正在连接数据库...")
	db, err := connectDatabase(dbConfig)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 检查是否已初始化
	hasData, err := CheckInit(db)
	if err != nil {
		return fmt.Errorf("检查数据库状态失败: %w", err)
	}

	if hasData {
		fmt.Println("数据库已包含权限数据，是否要重新初始化？")
		fmt.Println("这将删除所有现有用户和权限数据！")
		fmt.Print("确认重新初始化？(yes/no): ")

		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" && confirm != "y" {
			fmt.Println("初始化已取消")
			return nil
		}

		// 清空现有数据
		fmt.Println("清空现有数据...")
		db.Exec("DELETE FROM sys_authority_menus")
		db.Exec("DELETE FROM sys_user_authorities")
		db.Exec("DELETE FROM sys_users")
		db.Exec("DELETE FROM sys_authorities")
		db.Exec("DELETE FROM sys_base_menus")
		db.Exec("DELETE FROM sys_base_menu_parameters")
		db.Exec("DELETE FROM sys_apis")
		db.Exec("DELETE FROM sys_jwt_blacklists")
		db.Exec("DELETE FROM sys_operation_records")
		db.Exec("DELETE FROM casbin_rule")
	}

	// 初始化
	return InitDB(db)
}

// connectDatabase 连接数据库（简化版，用于初始化脚本）
func connectDatabase(configStr string) (*gorm.DB, error) {
	// 这里应该解析配置字符串并连接数据库
	// 为简化，使用现有的数据库连接逻辑
	return nil, fmt.Errorf("请通过主服务器初始化数据库")
}

// SetupAuthRoutes 设置权限系统路由
func SetupAuthRoutes(db *gorm.DB, routerGroup *gin.RouterGroup, jwtConfig *middleware.JWTConfig) (*Manager, error) {
	// 设置验证码验证函数
	if captchaVerifyFunc != nil {
		service.SetCaptchaVerify(captchaVerifyFunc)
	} else {
		// 默认验证码验证函数
		service.SetCaptchaVerify(func(id, code string) bool {
			return captcha.GetCodeStore().Verify(id, code)
		})
	}

	// 创建验证码实例
	captch := captcha.NewCaptcha(nil)
	captch.RegisterRoutes(routerGroup.Group("/base"))

	// 创建权限管理器
	authManager, err := NewManager(db, &Config{
		JWTSigningKey: jwtConfig.SigningKey,
		JWTExpires:    jwtConfig.ExpiresTime.String(),
		BufferTime:    jwtConfig.BufferTime.String(),
		RouterPrefix:  "/api",
	})
	if err != nil {
		return nil, err
	}

	// 初始化权限系统
	if err := authManager.Initialize(); err != nil {
		return nil, err
	}

	// 注册路由
	authManager.RegisterRoutes(routerGroup)

	return authManager, nil
}