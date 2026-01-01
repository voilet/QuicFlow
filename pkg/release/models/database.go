package models

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultConfig 默认配置
func DefaultConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:    "localhost",
		Port:    5432,
		User:    "postgres",
		Password: "postgres",
		DBName:  "quic_release",
		SSLMode: "disable",
	}
}

// DSN 生成数据库连接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// InitDB 初始化数据库连接
func InitDB(config *DatabaseConfig) (*gorm.DB, error) {
	if config == nil {
		config = DefaultConfig()
	}

	db, err := gorm.Open(postgres.Open(config.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	return db, nil
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	// 启用 uuid-ossp 扩展
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	// 自动迁移所有模型
	if err := db.AutoMigrate(AllModels...); err != nil {
		return fmt.Errorf("failed to migrate models: %w", err)
	}

	// 创建索引
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes 创建额外索引
func createIndexes(db *gorm.DB) error {
	indexes := []string{
		// 状态上报复合索引
		`CREATE INDEX IF NOT EXISTS idx_status_reports_release_target
		 ON release_status_reports(release_id, target_id, reported_at DESC)`,

		// 发布记录索引
		`CREATE INDEX IF NOT EXISTS idx_releases_project_env
		 ON releases(project_id, environment_id, created_at DESC)`,

		// 目标安装状态索引
		`CREATE INDEX IF NOT EXISTS idx_target_installations_target_project
		 ON target_installations(target_id, project_id)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

// DropAllTables 删除所有表 (仅用于测试)
func DropAllTables(db *gorm.DB) error {
	tables := []string{
		"release_service_dependencies",
		"release_approvals",
		"release_status_reports",
		"target_installations",
		"releases",
		"pipelines",
		"variables",
		"targets",
		"environments",
		"projects",
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			return err
		}
	}

	return nil
}
