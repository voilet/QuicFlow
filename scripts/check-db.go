package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"

	"github.com/voilet/quic-flow/pkg/release/models"
)

func readPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\n读取密码失败:", err)
		os.Exit(1)
	}
	fmt.Println()
	return strings.TrimSpace(string(bytePassword))
}

func main() {
	var host, user, password, dbname string
	var port int

	// 从环境变量或命令行参数获取配置
	if len(os.Args) >= 4 {
		host = os.Args[1]
		if p, err := strconv.Atoi(os.Args[2]); err == nil {
			port = p
		} else {
			fmt.Printf("错误: 无效的端口号: %s\n", os.Args[2])
			os.Exit(1)
		}
		user = os.Args[3]
		if len(os.Args) >= 5 {
			password = os.Args[4]
		}
		if len(os.Args) >= 6 {
			dbname = os.Args[5]
		}
	} else {
		// 交互式输入
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("数据库主机 [192.168.110.104]: ")
		host, _ = reader.ReadString('\n')
		host = strings.TrimSpace(host)
		if host == "" {
			host = "192.168.110.104"
		}

		fmt.Print("端口 [15432]: ")
		portStr, _ := reader.ReadString('\n')
		portStr = strings.TrimSpace(portStr)
		if portStr == "" {
			port = 15432
		} else {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			} else {
				fmt.Printf("错误: 无效的端口号: %s\n", portStr)
				os.Exit(1)
			}
		}

		fmt.Print("用户名 [postgres]: ")
		user, _ = reader.ReadString('\n')
		user = strings.TrimSpace(user)
		if user == "" {
			user = "postgres"
		}
	}

	// 密码从环境变量或交互式输入
	if password == "" {
		if envPass := os.Getenv("DB_PASSWORD"); envPass != "" {
			password = envPass
		} else {
			password = readPassword("密码: ")
		}
	}

	if dbname == "" {
		dbname = "quic_release"
	}

	fmt.Printf("正在检查数据库连接...\n")
	fmt.Printf("主机: %s:%d\n", host, port)
	fmt.Printf("用户: %s\n", user)
	fmt.Printf("数据库: %s\n\n", dbname)

	// 创建配置
	config := &models.DatabaseConfig{
		Type:     models.DBTypePostgres,
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbname,
		SSLMode:  "disable",
	}

	// 1. 测试连接到 PostgreSQL 服务器（使用 postgres 系统数据库）
	fmt.Println("步骤 1: 测试连接到 PostgreSQL 服务器...")
	systemConfig := *config
	systemConfig.DBName = "postgres"
	
	db, err := models.InitDB(&systemConfig)
	if err != nil {
		fmt.Printf("❌ 无法连接到 PostgreSQL 服务器: %v\n", err)
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
	fmt.Println("✅ 成功连接到 PostgreSQL 服务器")

	// 2. 列出所有数据库
	fmt.Println("\n步骤 2: 列出所有数据库...")
	databases, err := models.ListDatabases(config)
	if err != nil {
		fmt.Printf("❌ 无法列出数据库: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("找到 %d 个数据库:\n", len(databases))
	for _, db := range databases {
		if db == dbname {
			fmt.Printf("  ✅ %s (已存在)\n", db)
		} else {
			fmt.Printf("  - %s\n", db)
		}
	}

	// 3. 检查目标数据库是否存在
	fmt.Printf("\n步骤 3: 检查数据库 '%s' 是否存在...\n", dbname)
	exists, err := models.CheckDatabaseExists(config, dbname)
	if err != nil {
		fmt.Printf("❌ 检查失败: %v\n", err)
		os.Exit(1)
	}

	if exists {
		fmt.Printf("✅ 数据库 '%s' 已存在\n", dbname)
	} else {
		fmt.Printf("❌ 数据库 '%s' 不存在\n", dbname)
		fmt.Printf("\n步骤 4: 创建数据库 '%s'...\n", dbname)
		if err := models.CreateDatabase(config); err != nil {
			fmt.Printf("❌ 创建数据库失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ 数据库 '%s' 创建成功\n", dbname)
	}

	// 4. 测试连接到目标数据库
	fmt.Printf("\n步骤 5: 测试连接到数据库 '%s'...\n", dbname)
	db, err = models.InitDB(config)
	if err != nil {
		fmt.Printf("❌ 无法连接到数据库 '%s': %v\n", dbname, err)
		os.Exit(1)
	}
	sqlDB, _ = db.DB()
	if err := sqlDB.Ping(); err != nil {
		fmt.Printf("❌ Ping 失败: %v\n", err)
		os.Exit(1)
	}
	sqlDB.Close()
	fmt.Printf("✅ 成功连接到数据库 '%s'\n", dbname)

	fmt.Println("\n✅ 所有检查通过！数据库已准备就绪。")
}

