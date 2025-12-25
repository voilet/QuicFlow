package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voilet/QuicFlow/pkg/api"
	"github.com/voilet/QuicFlow/pkg/command"
	"github.com/voilet/QuicFlow/pkg/monitoring"
	"github.com/voilet/QuicFlow/pkg/transport/server"
)

func main() {
	// 命令行参数
	addr := flag.String("addr", ":8474", "服务器监听地址")
	cert := flag.String("cert", "certs/server-cert.pem", "TLS 证书文件路径")
	key := flag.String("key", "certs/server-key.pem", "TLS 私钥文件路径")
	apiAddr := flag.String("api", ":8475", "HTTP API 监听地址")
	flag.Parse()

	// 创建日志器
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")

	logger.Info("=== QUIC Backbone Server ===")
	logger.Info("Starting server", "addr", *addr)

	// 创建服务器配置
	config := server.NewDefaultServerConfig(*cert, *key, *addr)
	config.Logger = logger

	// 设置事件钩子
	config.Hooks = &monitoring.EventHooks{
		OnConnect: func(clientID string) {
			logger.Info("✅ Client connected", "client_id", clientID)
		},
		OnDisconnect: func(clientID string, reason error) {
			logger.Info("❌ Client disconnected", "client_id", clientID, "reason", reason)
		},
		OnHeartbeatTimeout: func(clientID string) {
			logger.Warn("💔 Heartbeat timeout", "client_id", clientID)
		},
	}

	// 创建服务器
	srv, err := server.NewServer(config)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// 启动服务器
	if err := srv.Start(*addr); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("✅ Server started successfully")

	// 创建命令管理器
	commandManager := command.NewCommandManager(srv, logger)
	logger.Info("✅ Command manager created")

	// 启动 HTTP API 服务器
	httpServer := api.NewHTTPServer(*apiAddr, srv, commandManager, logger)
	if err := httpServer.Start(); err != nil {
		logger.Error("Failed to start HTTP API server", "error", err)
		os.Exit(1)
	}

	logger.Info("✅ HTTP API server started", "addr", *apiAddr)
	logger.Info("✅ Command system enabled")
	logger.Info("Press Ctrl+C to stop")

	// 定期打印统计信息
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			metrics := srv.GetMetrics()
			clients := srv.ListClients()

			fmt.Printf("\n=== Server Status ===\n")
			fmt.Printf("Connected Clients: %d\n", len(clients))
			fmt.Printf("Total Connections: %d\n", metrics.ConnectedClients)
			fmt.Printf("Messages Sent: %d\n", metrics.MessageThroughput)

			if len(clients) > 0 {
				fmt.Printf("Active Clients:\n")
				for _, clientID := range clients {
					info, err := srv.GetClientInfo(clientID)
					if err == nil {
						uptime := time.Since(time.UnixMilli(info.ConnectedAt))
						fmt.Printf("  - %s (uptime: %v)\n", clientID, uptime.Round(time.Second))
					}
				}
			}
			fmt.Println()
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 停止 HTTP API 服务器
	if err := httpServer.Stop(ctx); err != nil {
		logger.Error("Error stopping HTTP API server", "error", err)
	}

	if err := srv.Stop(ctx); err != nil {
		logger.Error("Error during shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server stopped gracefully")
}
