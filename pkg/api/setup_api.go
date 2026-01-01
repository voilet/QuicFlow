package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/voilet/quic-flow/pkg/config"
	"github.com/voilet/quic-flow/pkg/monitoring"
	releasemodels "github.com/voilet/quic-flow/pkg/release/models"
	"gorm.io/gorm"
)

// SetupAPI 数据库初始化引导 API
type SetupAPI struct {
	logger       *monitoring.Logger
	configPath   string
	db           *gorm.DB
	dbMu         sync.RWMutex
	initialized  bool
	onDBReady    func(*gorm.DB) // 数据库就绪回调
}

// NewSetupAPI 创建 SetupAPI
func NewSetupAPI(configPath string, logger *monitoring.Logger) *SetupAPI {
	return &SetupAPI{
		logger:     logger,
		configPath: configPath,
	}
}

// SetOnDBReady 设置数据库就绪回调
func (s *SetupAPI) SetOnDBReady(callback func(*gorm.DB)) {
	s.onDBReady = callback
}

// GetDB 获取数据库连接
func (s *SetupAPI) GetDB() *gorm.DB {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
	return s.db
}

// IsInitialized 检查是否已初始化
func (s *SetupAPI) IsInitialized() bool {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()
	return s.initialized
}

// RegisterRoutes 注册路由
func (s *SetupAPI) RegisterRoutes(r *gin.RouterGroup) {
	setup := r.Group("/setup")
	{
		setup.GET("/status", s.handleStatus)
		setup.POST("/test-connection", s.handleTestConnection)
		setup.POST("/initialize", s.handleInitialize)
	}
}

// DatabaseConfig 数据库配置请求
type DatabaseConfig struct {
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port" binding:"required"`
	User        string `json:"user" binding:"required"`
	Password    string `json:"password"`
	DBName      string `json:"dbname" binding:"required"`
	SSLMode     string `json:"sslmode"`
	AutoMigrate bool   `json:"auto_migrate"`
}

// SetupStatus 初始化状态响应
type SetupStatus struct {
	Initialized    bool   `json:"initialized"`
	DatabaseStatus string `json:"database_status"` // connected, disconnected, not_configured
	Message        string `json:"message,omitempty"`
}

// handleStatus 检查初始化状态
func (s *SetupAPI) handleStatus(c *gin.Context) {
	s.dbMu.RLock()
	defer s.dbMu.RUnlock()

	status := SetupStatus{
		Initialized: s.initialized,
	}

	if s.db != nil {
		// 测试连接
		sqlDB, err := s.db.DB()
		if err == nil && sqlDB.Ping() == nil {
			status.DatabaseStatus = "connected"
			status.Message = "Database is connected and ready"
		} else {
			status.DatabaseStatus = "disconnected"
			status.Message = "Database connection lost"
		}
	} else {
		status.DatabaseStatus = "not_configured"
		status.Message = "Database not configured"
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "status": status})
}

// handleTestConnection 测试数据库连接
func (s *SetupAPI) handleTestConnection(c *gin.Context) {
	var req DatabaseConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 设置默认值
	if req.SSLMode == "" {
		req.SSLMode = "disable"
	}

	// 构建配置
	dbConfig := &releasemodels.DatabaseConfig{
		Host:     req.Host,
		Port:     req.Port,
		User:     req.User,
		Password: req.Password,
		DBName:   req.DBName,
		SSLMode:  req.SSLMode,
	}

	s.logger.Info("Testing database connection",
		"host", dbConfig.Host,
		"port", dbConfig.Port,
		"dbname", dbConfig.DBName)

	// 尝试连接
	db, err := releasemodels.InitDB(dbConfig)
	if err != nil {
		s.logger.Error("Database connection test failed", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"success":   false,
			"connected": false,
			"error":     fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}

	// 关闭测试连接
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	s.logger.Info("Database connection test successful")
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"connected": true,
		"message":   "Connection successful",
	})
}

// handleInitialize 初始化数据库
func (s *SetupAPI) handleInitialize(c *gin.Context) {
	var req DatabaseConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	// 设置默认值
	if req.SSLMode == "" {
		req.SSLMode = "disable"
	}

	// 构建配置
	dbConfig := &releasemodels.DatabaseConfig{
		Host:     req.Host,
		Port:     req.Port,
		User:     req.User,
		Password: req.Password,
		DBName:   req.DBName,
		SSLMode:  req.SSLMode,
	}

	s.logger.Info("Initializing database",
		"host", dbConfig.Host,
		"port", dbConfig.Port,
		"dbname", dbConfig.DBName)

	// 连接数据库
	db, err := releasemodels.InitDB(dbConfig)
	if err != nil {
		s.logger.Error("Failed to connect to database", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"step":    "connect",
			"error":   fmt.Sprintf("Failed to connect: %v", err),
		})
		return
	}

	// 执行迁移
	s.logger.Info("Running database migrations...")
	if err := releasemodels.Migrate(db); err != nil {
		s.logger.Error("Database migration failed", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"step":    "migrate",
			"error":   fmt.Sprintf("Migration failed: %v", err),
		})
		return
	}

	// 保存配置到文件
	if err := s.saveConfig(req); err != nil {
		s.logger.Warn("Failed to save config file", "error", err)
		// 不返回错误，配置文件保存失败不影响运行
	}

	// 更新内部状态
	s.dbMu.Lock()
	s.db = db
	s.initialized = true
	s.dbMu.Unlock()

	// 触发回调
	if s.onDBReady != nil {
		s.onDBReady(db)
	}

	s.logger.Info("Database initialization completed")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database initialized successfully",
		"tables": []string{
			"projects",
			"environments",
			"targets",
			"variables",
			"pipelines",
			"releases",
			"target_installations",
			"release_status_reports",
			"release_approvals",
			"release_service_dependencies",
		},
	})
}

// saveConfig 保存配置到文件
func (s *SetupAPI) saveConfig(dbConfig DatabaseConfig) error {
	// 读取现有配置
	cfg, err := config.Load(s.configPath)
	if err != nil {
		// 如果配置文件不存在，使用默认配置
		cfg = config.DefaultConfig()
	}

	// 更新数据库配置
	cfg.Database.Enabled = true
	cfg.Database.Host = dbConfig.Host
	cfg.Database.Port = dbConfig.Port
	cfg.Database.User = dbConfig.User
	cfg.Database.Password = dbConfig.Password
	cfg.Database.DBName = dbConfig.DBName
	cfg.Database.SSLMode = dbConfig.SSLMode
	cfg.Database.AutoMigrate = dbConfig.AutoMigrate

	// 确保目录存在
	dir := filepath.Dir(s.configPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// 保存配置
	if err := config.GenerateConfig(s.configPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	s.logger.Info("Configuration saved", "path", s.configPath)
	return nil
}

// TryAutoConnect 尝试使用现有配置自动连接
func (s *SetupAPI) TryAutoConnect(cfg *config.ServerConfig) error {
	if !cfg.Database.Enabled {
		return fmt.Errorf("database not enabled in config")
	}

	dbConfig := &releasemodels.DatabaseConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	db, err := releasemodels.InitDB(dbConfig)
	if err != nil {
		return err
	}

	// 自动迁移
	if cfg.Database.AutoMigrate {
		if err := releasemodels.Migrate(db); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	s.dbMu.Lock()
	s.db = db
	s.initialized = true
	s.dbMu.Unlock()

	if s.onDBReady != nil {
		s.onDBReady(db)
	}

	return nil
}
