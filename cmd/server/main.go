package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/voilet/quic-flow/pkg/api"
	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/dispatcher"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/protocol"
	"github.com/voilet/quic-flow/pkg/router"
	"github.com/voilet/quic-flow/pkg/transport/server"
	"github.com/voilet/quic-flow/pkg/version"
)

var (
	// 命令行参数
	addr    string
	cert    string
	key     string
	apiAddr string
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "quic-server",
	Short: "QUIC Backbone Server",
	Long:  "QUIC Backbone Server - 高性能 QUIC 协议服务器",
	Run:   runServer,
}

// versionCmd 版本信息子命令
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  "显示服务器版本号、Git 提交、编译时间等信息",
	Run: func(cmd *cobra.Command, args []string) {
		version.Print("quic-server")
	},
}

func init() {
	// 定义命令行参数
	rootCmd.Flags().StringVarP(&addr, "addr", "a", ":8474", "服务器监听地址")
	rootCmd.Flags().StringVarP(&cert, "cert", "c", "certs/server-cert.pem", "TLS 证书文件路径")
	rootCmd.Flags().StringVarP(&key, "key", "k", "certs/server-key.pem", "TLS 私钥文件路径")
	rootCmd.Flags().StringVarP(&apiAddr, "api", "p", ":8475", "HTTP API 监听地址")

	// 添加子命令
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// 创建日志器
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")

	logger.Info("=== QUIC Backbone Server ===")
	logger.Info("Version", "version", version.String())
	logger.Info("Starting server", "addr", addr)

	// 创建服务器配置
	config := server.NewDefaultServerConfig(cert, key, addr)
	config.Logger = logger

	// 设置事件钩子
	config.Hooks = &monitoring.EventHooks{
		OnConnect: func(clientID string) {
			logger.Info("Client connected", "client_id", clientID)
		},
		OnDisconnect: func(clientID string, reason error) {
			logger.Info("Client disconnected", "client_id", clientID, "reason", reason)
		},
		OnHeartbeatTimeout: func(clientID string) {
			logger.Warn("Heartbeat timeout", "client_id", clientID)
		},
	}

	// 创建服务器
	srv, err := server.NewServer(config)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// 设置消息路由器
	msgRouter := SetupServerRouter(logger)

	// 创建 Dispatcher 并注册消息处理器
	disp := setupServerDispatcher(logger, msgRouter)

	// 设置 Dispatcher 到服务器
	srv.SetDispatcher(disp)
	logger.Info("Dispatcher attached to server")

	// 启动服务器
	if err := srv.Start(addr); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("Server started successfully")

	// 创建命令管理器
	commandManager := command.NewCommandManager(srv, logger)
	logger.Info("Command manager created")

	// 启动 HTTP API 服务器
	httpServer := api.NewHTTPServer(apiAddr, srv, commandManager, logger)
	if err := httpServer.Start(); err != nil {
		logger.Error("Failed to start HTTP API server", "error", err)
		os.Exit(1)
	}

	logger.Info("HTTP API server started", "addr", apiAddr)
	logger.Info("Command system enabled")
	logger.Info("Press Ctrl+C to stop")

	// 定期打印统计信息
	go printServerStatus(srv, msgRouter)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	shutdownServer(logger, disp, httpServer, srv)
}

// setupServerDispatcher 设置服务器消息分发器
func setupServerDispatcher(logger *monitoring.Logger, msgRouter *router.Router) *dispatcher.Dispatcher {
	dispatcherConfig := &dispatcher.DispatcherConfig{
		WorkerCount:    20,
		TaskQueueSize:  2000,
		HandlerTimeout: 30 * time.Second,
		Logger:         logger,
	}
	disp := dispatcher.NewDispatcher(dispatcherConfig)

	// 创建路由处理函数
	routeHandler := func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
		// 从payload中解析命令类型
		var cmdPayload struct {
			CommandType string          `json:"command_type"`
			Payload     json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(msg.Payload, &cmdPayload); err != nil {
			// 如果解析失败，尝试直接作为payload处理
			cmdPayload.CommandType = "unknown"
			cmdPayload.Payload = msg.Payload
		}

		// 使用路由器执行
		result, err := msgRouter.ExecuteWithContext(ctx, cmdPayload.CommandType, cmdPayload.Payload)
		if err != nil {
			return nil, err
		}

		// 构建响应消息
		return &protocol.DataMessage{
			MsgId:     msg.MsgId,
			SenderId:  "server",
			Type:      protocol.MessageType_MESSAGE_TYPE_RESPONSE,
			Payload:   result,
			Timestamp: time.Now().UnixMilli(),
		}, nil
	}

	// 注册消息类型处理器
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_EVENT, dispatcher.MessageHandlerFunc(routeHandler))
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_QUERY, dispatcher.MessageHandlerFunc(routeHandler))
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_RESPONSE, dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
		logger.Info("Received response from client", "msg_id", msg.MsgId, "sender", msg.SenderId)
		return nil, nil
	}))

	// 启动 Dispatcher
	disp.Start()
	logger.Info("Dispatcher started")

	return disp
}

// printServerStatus 定期打印服务器状态
func printServerStatus(srv *server.Server, msgRouter *router.Router) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics := srv.GetMetrics()
		clients := srv.ListClients()

		fmt.Printf("\n=== Server Status ===\n")
		fmt.Printf("Connected Clients: %d\n", len(clients))
		fmt.Printf("Total Connections: %d\n", metrics.ConnectedClients)
		fmt.Printf("Messages Sent: %d\n", metrics.MessageThroughput)
		fmt.Printf("Registered Routes: %v\n", msgRouter.ListCommands())

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
}

// shutdownServer 优雅关闭服务器
func shutdownServer(logger *monitoring.Logger, disp *dispatcher.Dispatcher, httpServer *api.HTTPServer, srv *server.Server) {
	logger.Info("Shutting down server...")

	// 停止 Dispatcher
	disp.Stop()
	logger.Info("Dispatcher stopped")

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
