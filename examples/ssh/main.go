// Package main 演示 SSH-over-QUIC 功能
// 这个示例展示了如何在 QUIC 隧道上运行 SSH 协议
//
// 架构说明：
//   - 传输层：UDP
//   - 隧道层：QUIC (多路复用、加密、拥塞控制)
//   - 应用层：SSH (权限控制、Shell 交互、文件传输)
//
// 使用场景：
//   - 内网穿透：内网客户端主动连接到公网服务器，建立 QUIC 长连接
//   - 反向 SSH：公网服务器可以通过已建立的 QUIC 连接访问内网机器
//
// 运行方式：
//   go run examples/ssh/main.go -mode server  # 在公网服务器运行
//   go run examples/ssh/main.go -mode client  # 在内网机器运行
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"

	quicssh "github.com/voilet/quic-flow/pkg/ssh"
)

var (
	mode     = flag.String("mode", "client", "运行模式: server 或 client")
	addr     = flag.String("addr", "127.0.0.1:4443", "服务器地址")
	user     = flag.String("user", "root", "SSH 用户名")
	password = flag.String("password", "password", "SSH 密码")
)

func main() {
	flag.Parse()

	switch *mode {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		log.Fatalf("未知模式: %s", *mode)
	}
}

// runServer 运行 QUIC 服务端 + SSH 客户端
// 在公网服务器运行，等待内网客户端连接
func runServer() {
	log.Println("=== SSH-over-QUIC 服务端 ===")
	log.Printf("监听地址: %s", *addr)
	log.Println("等待内网客户端连接...")
	log.Println()
	log.Println("当客户端连接后，可以通过 SSH 访问内网机器")
	log.Println("示例命令：")
	log.Println("  ssh -o ProxyCommand='quic-ssh %h' user@client-id")
	log.Println()

	// TODO: 这里应该启动 QUIC 服务器并监听连接
	// 当客户端连接后，可以打开 SSH 流连接到客户端的 SSH 服务

	// 演示 SSH 客户端配置
	clientConfig := &quicssh.ClientConfig{
		User:     *user,
		Password: *password,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应验证主机密钥
	}

	log.Printf("SSH 客户端配置: 用户=%s", clientConfig.User)

	// 等待中断信号
	waitForSignal()
}

// runClient 运行 QUIC 客户端 + SSH 服务端
// 在内网机器运行，主动连接到公网服务器
func runClient() {
	log.Println("=== SSH-over-QUIC 客户端 ===")
	log.Printf("连接到服务器: %s", *addr)
	log.Println()
	log.Println("内网 SSH 服务将通过 QUIC 隧道暴露给公网服务器")
	log.Println()

	// 创建 SSH 服务器配置
	serverConfig := quicssh.DefaultServerConfig()
	serverConfig.PasswordAuth = true
	serverConfig.PasswordCallback = func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		// 简单的密码验证（生产环境应使用更安全的方式）
		if conn.User() == "root" && string(password) == "password" {
			return nil, nil
		}
		return nil, fmt.Errorf("invalid password for %s", conn.User())
	}

	// 创建 SSH 管理器
	manager := quicssh.NewManager()
	if err := manager.InitServer(serverConfig); err != nil {
		log.Fatalf("初始化 SSH 服务器失败: %v", err)
	}

	// 启动 SSH 服务器
	if err := manager.StartServer(); err != nil {
		log.Fatalf("启动 SSH 服务器失败: %v", err)
	}

	log.Println("SSH 服务器已启动，等待来自公网的连接...")

	// TODO: 这里应该连接到 QUIC 服务器
	// 并在收到 SSH 类型的流时，调用 manager.HandleSSHStream()

	// 等待中断信号
	waitForSignal()

	// 清理
	manager.Stop()
	log.Println("SSH 服务器已停止")
}

func waitForSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("\n收到中断信号，正在关闭...")
}
