package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/voilet/quic-flow/pkg/auth/api"
	authinit "github.com/voilet/quic-flow/pkg/auth/init"
	"github.com/voilet/quic-flow/pkg/auth/middleware"
	"github.com/voilet/quic-flow/pkg/auth/models"
)

// Manager 权限系统管理器
type Manager struct {
	db                 *gorm.DB
	jwtMiddleware      *middleware.JWTAuthMiddleware
	casbinMiddleware   *middleware.CasbinMiddleware
	authAPI            *api.AuthAPI
	jwtConfig          *middleware.JWTConfig
	casbinConfig       *middleware.CasbinConfig
}

// Config 权限系统配置
type Config struct {
	JWTSigningKey string        // JWT 签名密钥
	JWTExpires    string        // JWT 过期时间，如 "7d"
	BufferTime    string        // JWT 缓冲时间，如 "1h"
	RouterPrefix  string        // API 路由前缀
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	JWTSigningKey: "quic-flow-secret-key-change-in-production",
	JWTExpires:    "7d",
	BufferTime:    "1h",
	RouterPrefix:  "/api",
}

// NewManager 创建权限系统管理器
func NewManager(db *gorm.DB, config *Config) (*Manager, error) {
	if config == nil {
		config = &DefaultConfig
	}

	// 解析时间
	expiresTime, _ := parseDuration(config.JWTExpires)
	bufferTime, _ := parseDuration(config.BufferTime)

	// 创建 JWT 配置
	jwtConfig := &middleware.JWTConfig{
		SigningKey:  config.JWTSigningKey,
		ExpiresTime: expiresTime,
		BufferTime:  bufferTime,
		Issuer:      "quic-flow",
	}

	// 创建 Casbin 配置
	casbinConfig := &middleware.CasbinConfig{
		RouterPrefix: config.RouterPrefix,
		ModelText: middleware.DefaultCasbinConfig.ModelText,
	}

	// 创建中间件
	jwtMiddleware := middleware.NewJWTAuthMiddleware(db, jwtConfig)
	casbinMiddleware, err := middleware.NewCasbinMiddleware(db, casbinConfig)
	if err != nil {
		return nil, err
	}

	// 创建 API
	authAPI := api.NewAuthAPI(db, jwtConfig)

	return &Manager{
		db:               db,
		jwtMiddleware:    jwtMiddleware,
		casbinMiddleware: casbinMiddleware,
		authAPI:          authAPI,
		jwtConfig:        jwtConfig,
		casbinConfig:     casbinConfig,
	}, nil
}

// Initialize 初始化权限系统（迁移数据库并创建种子数据）
func (m *Manager) Initialize() error {
	// 迁移数据库表
	if err := models.AutoMigrateAuth(m.db); err != nil {
		return err
	}

	// 创建种子数据
	return authinit.SeedAuthData(m.db)
}

// RegisterRoutes 注册权限相关路由
func (m *Manager) RegisterRoutes(r *gin.RouterGroup) {
	// 注册 Casbin 权限中间件（使用 Gin 包装器）
	casbinHandler := middleware.GinCasbinMiddleware(
		m.casbinMiddleware,
		middleware.GetClaims,
	)

	// 使用 JWT 和 Casbin 双重中间件
	r.Use(m.jwtMiddleware.Handler())
	r.Use(casbinHandler)

	// 注册 API
	m.authAPI.RegisterRoutes(r, m.jwtMiddleware)
}

// RegisterPublicRoutes 注册公开路由（不需要认证）
func (m *Manager) RegisterPublicRoutes(r *gin.RouterGroup) {
	m.authAPI.RegisterRoutes(r, m.jwtMiddleware)
}

// GetJWTMiddleware 获取 JWT 中间件
func (m *Manager) GetJWTMiddleware() *middleware.JWTAuthMiddleware {
	return m.jwtMiddleware
}

// GetCasbinMiddleware 获取 Casbin 中间件
func (m *Manager) GetCasbinMiddleware() *middleware.CasbinMiddleware {
	return m.casbinMiddleware
}

// GetAuthAPI 获取认证 API
func (m *Manager) GetAuthAPI() *api.AuthAPI {
	return m.authAPI
}

// AddWhitelist 添加白名单路径
func (m *Manager) AddWhitelist(paths ...string) {
	middleware.WhiteList = append(middleware.WhiteList, paths...)
}

// SetWhitelist 设置白名单路径
func (m *Manager) SetWhitelist(paths []string) {
	middleware.WhiteList = paths
}

// parseDuration 解析时间字符串
func parseDuration(s string) (dur time.Duration, err error) {
	// 尝试解析常见格式
	if s == "" {
		return 7 * 24 * time.Hour, nil
	}

	// 简单处理：如果是数字则按小时计算
	var d int
	if _, err := fmt.Sscanf(s, "%d", &d); err == nil {
		if strings.HasSuffix(s, "h") {
			return time.Duration(d) * time.Hour, nil
		}
		if strings.HasSuffix(s, "d") {
			return time.Duration(d) * 24 * time.Hour, nil
		}
	}

	return time.ParseDuration(s)
}
