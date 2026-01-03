package models

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBType 数据库类型
type DBType string

const (
	DBTypePostgres DBType = "postgres"
	DBTypeMySQL    DBType = "mysql"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     DBType // 数据库类型: postgres, mysql
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string // PostgreSQL 专用
	Charset  string // MySQL 专用
}

// DefaultConfig 默认配置
func DefaultConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Type:     DBTypePostgres,
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "quic_release",
		SSLMode:  "disable",
		Charset:  "utf8mb4",
	}
}

// DSN 生成数据库连接字符串
func (c *DatabaseConfig) DSN() string {
	switch c.Type {
	case DBTypeMySQL:
		return c.mysqlDSN()
	default:
		return c.postgresDSN()
	}
}

// postgresDSN 生成 PostgreSQL 连接字符串
func (c *DatabaseConfig) postgresDSN() string {
	// 基础连接参数
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.DBName, c.SSLMode)

	// 只有在密码非空时才添加密码参数
	if c.Password != "" {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
	}

	return dsn
}

// mysqlDSN 生成 MySQL 连接字符串
func (c *DatabaseConfig) mysqlDSN() string {
	charset := c.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.DBName, charset,
	)
}

// systemDSN 生成系统数据库连接字符串(不指定具体数据库)
func (c *DatabaseConfig) systemDSN() string {
	switch c.Type {
	case DBTypeMySQL:
		charset := c.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local",
			c.User, c.Password, c.Host, c.Port, charset,
		)
	default:
		// PostgreSQL 连接到 postgres 系统数据库
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.SSLMode,
		)
	}
}

// InitDB 初始化数据库连接
func InitDB(config *DatabaseConfig) (*gorm.DB, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var dialector gorm.Dialector
	dsn := config.DSN()

	switch config.Type {
	case DBTypeMySQL:
		dialector = mysql.Open(dsn)
	default:
		dialector = postgres.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	return db, nil
}

// ListDatabases 列出服务器上所有可用的数据库
func ListDatabases(config *DatabaseConfig) ([]string, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var dialector gorm.Dialector
	switch config.Type {
	case DBTypeMySQL:
		dialector = mysql.Open(config.systemDSN())
	default:
		dialector = postgres.Open(config.systemDSN())
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database server: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	defer sqlDB.Close()

	var databases []string
	var query string

	switch config.Type {
	case DBTypeMySQL:
		query = "SHOW DATABASES"
		rows, err := db.Raw(query).Rows()
		if err != nil {
			return nil, fmt.Errorf("failed to list databases: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err != nil {
				continue
			}
			// 过滤系统数据库
			if dbName != "information_schema" && dbName != "mysql" &&
				dbName != "performance_schema" && dbName != "sys" {
				databases = append(databases, dbName)
			}
		}
	default:
		// PostgreSQL
		query = "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname"
		rows, err := db.Raw(query).Rows()
		if err != nil {
			return nil, fmt.Errorf("failed to list databases: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var dbName string
			if err := rows.Scan(&dbName); err != nil {
				continue
			}
			databases = append(databases, dbName)
		}
	}

	return databases, nil
}

// CheckDatabaseExists 检查数据库是否存在
func CheckDatabaseExists(config *DatabaseConfig, dbName string) (bool, error) {
	databases, err := ListDatabases(config)
	if err != nil {
		return false, err
	}

	for _, db := range databases {
		if db == dbName {
			return true, nil
		}
	}
	return false, nil
}

// CreateDatabase 创建数据库（如果不存在）
func CreateDatabase(config *DatabaseConfig) error {
	if config == nil {
		config = DefaultConfig()
	}

	// 先检查数据库是否已存在
	exists, err := CheckDatabaseExists(config, config.DBName)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		// 数据库已存在，无需创建
		return nil
	}

	// 连接到系统数据库
	var dialector gorm.Dialector
	switch config.Type {
	case DBTypeMySQL:
		dialector = mysql.Open(config.systemDSN())
	default:
		dialector = postgres.Open(config.systemDSN())
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database server: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	// 创建数据库
	var createSQL string
	switch config.Type {
	case DBTypeMySQL:
		charset := config.Charset
		if charset == "" {
			charset = "utf8mb4"
		}
		createSQL = fmt.Sprintf(
			"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s_unicode_ci",
			config.DBName, charset, charset,
		)
	default:
		// PostgreSQL
		createSQL = fmt.Sprintf(
			"CREATE DATABASE \"%s\" WITH ENCODING 'UTF8'",
			config.DBName,
		)
	}

	if err := db.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("failed to create database %s: %w", config.DBName, err)
	}

	return nil
}

// InitDBWithCreate 初始化数据库连接，如果数据库不存在则先创建
func InitDBWithCreate(config *DatabaseConfig) (*gorm.DB, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// 先尝试创建数据库（如果不存在）
	if err := CreateDatabase(config); err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// 然后连接到目标数据库
	return InitDB(config)
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	// 检测数据库类型并执行特定初始化
	dbType := detectDBType(db)
	fmt.Printf("Detected database type: %s\n", dbType)

	if dbType == DBTypePostgres {
		// PostgreSQL: 启用必要的扩展
		extensions := []string{
			`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
			`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`,
		}
		for _, ext := range extensions {
			if err := db.Exec(ext).Error; err != nil {
				// 忽略扩展创建错误（可能已存在或权限不足）
				fmt.Printf("Warning: failed to create extension: %v\n", err)
			}
		}
	}

	// 自动迁移所有模型
	fmt.Printf("Starting migration for %d models...\n", len(AllModels))
	for i, model := range AllModels {
		fmt.Printf("Migrating model %d: %T\n", i+1, model)
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}
	fmt.Println("All models migrated successfully")

	// 创建索引
	if err := createIndexes(db, dbType); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	fmt.Println("Database migration completed")
	return nil
}

// detectDBType 检测数据库类型
func detectDBType(db *gorm.DB) DBType {
	dialectorName := db.Dialector.Name()
	switch dialectorName {
	case "mysql":
		return DBTypeMySQL
	default:
		return DBTypePostgres
	}
}

// createIndexes 创建额外索引
func createIndexes(db *gorm.DB, dbType DBType) error {
	var indexes []string

	if dbType == DBTypeMySQL {
		// MySQL 索引语法
		indexes = []string{
			// 状态上报复合索引
			`CREATE INDEX IF NOT EXISTS idx_status_reports_release_target
			 ON release_status_reports(release_id, target_id, reported_at)`,

			// 发布记录索引
			`CREATE INDEX IF NOT EXISTS idx_releases_project_env
			 ON releases(project_id, environment_id, created_at)`,

			// 目标安装状态索引
			`CREATE INDEX IF NOT EXISTS idx_target_installations_target_project
			 ON target_installations(target_id, project_id)`,
		}
	} else {
		// PostgreSQL 索引语法 (支持 DESC)
		indexes = []string{
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
