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
	"github.com/voilet/quic-flow/pkg/audit"
	"github.com/voilet/quic-flow/pkg/batch"
	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/config"
	"github.com/voilet/quic-flow/pkg/dispatcher"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/protocol"
	"github.com/voilet/quic-flow/pkg/recording"
	releaseapi "github.com/voilet/quic-flow/pkg/release/api"
	releasemodels "github.com/voilet/quic-flow/pkg/release/models"
	"github.com/voilet/quic-flow/pkg/router"
	"github.com/voilet/quic-flow/pkg/transport/server"
	"github.com/voilet/quic-flow/pkg/version"

	"gorm.io/gorm"
)

var (
	// 命令行参数
	configFile string // 配置文件路径
	genConfig  string // 生成配置文件路径
	highPerf   bool   // 高性能模式（用于生成配置）
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

// genConfigCmd 生成配置文件子命令
var genConfigCmd = &cobra.Command{
	Use:   "genconfig",
	Short: "生成配置文件",
	Long:  "生成默认或高性能模式的配置文件",
	Run: func(cmd *cobra.Command, args []string) {
		if genConfig == "" {
			genConfig = "config/server.yaml"
		}

		if err := config.GenerateDefaultConfig(genConfig, highPerf); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate config: %v\n", err)
			os.Exit(1)
		}

		mode := "standard"
		if highPerf {
			mode = "high-performance"
		}
		fmt.Printf("Configuration file generated: %s (mode: %s)\n", genConfig, mode)
	},
}

func init() {
	// 主命令参数（默认使用 config/server.yaml）
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config/server.yaml", "配置文件路径")

	// 生成配置命令参数
	genConfigCmd.Flags().StringVarP(&genConfig, "output", "o", "config/server.yaml", "输出配置文件路径")
	genConfigCmd.Flags().BoolVar(&highPerf, "high-perf", false, "生成高性能模式配置")

	// 添加子命令
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(genConfigCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// 加载配置
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 创建日志器
	logLevel := monitoring.LogLevelInfo
	switch cfg.Log.Level {
	case "debug":
		logLevel = monitoring.LogLevelDebug
	case "warn":
		logLevel = monitoring.LogLevelWarn
	case "error":
		logLevel = monitoring.LogLevelError
	}
	logger := monitoring.NewLogger(logLevel, cfg.Log.Format)

	logger.Info("=== QUIC Backbone Server ===")
	logger.Info("Version", "version", version.String())
	if configFile != "" {
		logger.Info("Config file loaded", "path", configFile)
	}

	// 初始化数据库（如果启用）
	var releaseDB *gorm.DB
	if cfg.Database.Enabled {
		releaseDB, err = initDatabase(cfg, logger)
		if err != nil {
			logger.Error("Failed to initialize database", "error", err)
			logger.Warn("Release system will run without database")
		}
	} else {
		logger.Info("Database disabled, release system will be limited")
	}

	logger.Info("Starting server",
		"addr", cfg.Server.Addr,
		"high_perf", cfg.Server.HighPerf,
		"max_clients", cfg.Server.MaxClients)

	// 创建服务器配置
	serverConfig := buildServerConfig(cfg, logger)

	// 创建服务器
	srv, err := server.NewServer(serverConfig)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// 设置消息路由器
	msgRouter := SetupServerRouter(logger)

	// 创建 Dispatcher 并注册消息处理器
	disp := setupServerDispatcherWithConfig(logger, msgRouter,
		cfg.Message.WorkerCount,
		cfg.Message.TaskQueueSize,
		cfg.GetHandlerTimeout())

	logger.Info("Dispatcher created",
		"workers", cfg.Message.WorkerCount,
		"queue_size", cfg.Message.TaskQueueSize)

	// 设置 Dispatcher 到服务器
	srv.SetDispatcher(disp)
	logger.Info("Dispatcher attached to server")

	// 启动服务器
	if err := srv.Start(cfg.Server.Addr); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	logger.Info("Server started successfully")

	// 创建命令管理器
	commandManager := command.NewCommandManager(srv, logger)
	logger.Info("Command manager created")

	// 创建批量执行器
	var batchExecutor *batch.BatchExecutor
	if cfg.Batch.Enabled {
		batchConfig := &batch.BatchConfig{
			MaxConcurrency: cfg.Batch.MaxConcurrency,
			TaskTimeout:    cfg.GetTaskTimeout(),
			JobTimeout:     cfg.GetJobTimeout(),
			MaxRetries:     cfg.Batch.MaxRetries,
			RetryInterval:  cfg.GetRetryInterval(),
			Logger:         logger,
			OnProgress: func(job *batch.BatchJob) {
				progress := float64(job.SuccessCount+job.FailedCount) / float64(job.TotalCount) * 100
				logger.Info("Batch progress", "job_id", job.ID, "progress", fmt.Sprintf("%.1f%%", progress))
			},
		}
		batchExecutor = batch.NewBatchExecutor(srv, batchConfig)
		logger.Info("Batch executor created", "max_concurrency", batchConfig.MaxConcurrency)
	}

	// 启动 HTTP API 服务器
	httpServer := api.NewHTTPServer(cfg.Server.APIAddr, srv, commandManager, logger)

	// 添加数据库初始化引导 API
	setupAPI := api.NewSetupAPI(configFile, logger)
	httpServer.AddSetupRoutes(setupAPI)

	// 尝试自动连接数据库（如果配置了）
	if cfg.Database.Enabled {
		if err := setupAPI.TryAutoConnect(cfg); err != nil {
			logger.Warn("Auto-connect to database failed", "error", err)
			logger.Info("Visit /setup to configure database")
		} else {
			logger.Info("Database auto-connected successfully")
			releaseDB = setupAPI.GetDB()
		}
	}

	// 添加批量执行 API
	if batchExecutor != nil {
		httpServer.AddBatchRoutes(batchExecutor)
	}

	// 添加流式 API（SSE）
	httpServer.AddStreamRoutes()

	// 创建 SSH 客户端管理器
	sshManager := NewSSHClientManager(srv, nil, logger)
	sshAPIAdapter := NewSSHClientManagerAPIAdapter(sshManager)
	httpServer.AddSSHRoutes(sshAPIAdapter)
	logger.Info("SSH client manager created")

	// 创建审计存储（每个会话一个文件）
	auditStore, err := audit.NewSessionFileStore("data/audit")
	if err != nil {
		logger.Error("Failed to create audit store", "error", err)
		os.Exit(1)
	}
	logger.Info("Audit store created", "path", "data/audit")

	// 创建录像存储
	recordingStore, err := recording.NewStore("data/recordings")
	if err != nil {
		logger.Error("Failed to create recording store", "error", err)
		os.Exit(1)
	}
	logger.Info("Recording store created", "path", "data/recordings")

	// 创建录像配置
	recordingConfig := &recording.Config{
		Enabled:     true,
		StorePath:   "data/recordings",
		RecordInput: true,
	}

	// 创建终端管理器（WebSocket SSH 终端）带审计和录像
	terminalAdapter := NewSSHTerminalAdapter(sshManager)
	terminalManager := api.NewTerminalManagerWithRecording(terminalAdapter, logger, auditStore, recordingConfig)
	httpServer.AddTerminalRoutes(terminalManager)
	logger.Info("Terminal WebSocket routes added")

	// 添加审计 API 路由
	auditAPI := api.NewAuditAPI(auditStore, logger)
	httpServer.AddAuditRoutes(auditAPI)

	// 添加录像 API 路由
	recordingAPI := api.NewRecordingAPI(recordingStore, logger)
	httpServer.AddRecordingRoutes(recordingAPI)

	// 添加发布系统 API 路由
	releaseAPI := releaseapi.NewReleaseAPIWithRemote(releaseDB, commandManager)
	httpServer.AddReleaseRoutes(releaseAPI)

	// 设置数据库初始化回调，当通过 setup 页面初始化数据库后更新 release API
	setupAPI.SetOnDBReady(func(db *gorm.DB) {
		releaseAPI.SetDB(db)
		releaseAPI.SetRemoteExecutor(commandManager)
		logger.Info("Release API database updated via setup")
	})

	if releaseDB != nil {
		logger.Info("Release API routes added with database")
	} else {
		logger.Info("Release API routes added (database not configured, visit /setup)")
	}

	if err := httpServer.Start(); err != nil {
		logger.Error("Failed to start HTTP API server", "error", err)
		os.Exit(1)
	}

	logger.Info("HTTP API server started", "addr", cfg.Server.APIAddr)
	logger.Info("Command system enabled")
	if batchExecutor != nil {
		logger.Info("Batch execution enabled")
	}
	logger.Info("SSH over QUIC enabled")
	logger.Info("Press Ctrl+C to stop")

	// 定期打印统计信息
	go printServerStatus(srv, msgRouter)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	if batchExecutor != nil {
		batchExecutor.Stop()
	}
	sshManager.Close()
	auditStore.Close()
	shutdownServer(logger, disp, httpServer, srv)
}

// buildServerConfig 从配置文件构建服务器配置
func buildServerConfig(cfg *config.ServerConfig, logger *monitoring.Logger) *server.ServerConfig {
	serverConfig := &server.ServerConfig{
		TLSCertFile: cfg.TLS.CertFile,
		TLSKeyFile:  cfg.TLS.KeyFile,
		ListenAddr:  cfg.Server.Addr,

		// QUIC 配置
		MaxIdleTimeout:                 cfg.GetMaxIdleTimeout(),
		MaxIncomingStreams:             cfg.QUIC.MaxIncomingStreams,
		MaxIncomingUniStreams:          cfg.QUIC.MaxIncomingUniStreams,
		InitialStreamReceiveWindow:     cfg.QUIC.InitialStreamReceiveWindow,
		MaxStreamReceiveWindow:         cfg.QUIC.MaxStreamReceiveWindow,
		InitialConnectionReceiveWindow: cfg.QUIC.InitialConnectionReceiveWindow,
		MaxConnectionReceiveWindow:     cfg.QUIC.MaxConnectionReceiveWindow,

		// 会话管理配置
		MaxClients:             cfg.Server.MaxClients,
		HeartbeatInterval:      cfg.GetHeartbeatInterval(),
		HeartbeatTimeout:       cfg.GetHeartbeatTimeout(),
		HeartbeatCheckInterval: cfg.GetHeartbeatCheckInterval(),
		MaxTimeoutCount:        cfg.Session.MaxTimeoutCount,

		// Promise 配置
		MaxPromises:           cfg.Message.MaxPromises,
		PromiseWarnThreshold:  cfg.Message.PromiseWarnThreshold,
		DefaultMessageTimeout: cfg.GetDefaultMessageTimeout(),

		// 监控
		Logger: logger,
	}

	// 设置事件钩子
	serverConfig.Hooks = &monitoring.EventHooks{
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

	return serverConfig
}

// setupServerDispatcherWithConfig 使用自定义配置设置服务器消息分发器
func setupServerDispatcherWithConfig(logger *monitoring.Logger, msgRouter *router.Router, workerCount int, queueSize int, timeout time.Duration) *dispatcher.Dispatcher {
	dispatcherConfig := &dispatcher.DispatcherConfig{
		WorkerCount:    workerCount,
		TaskQueueSize:  queueSize,
		HandlerTimeout: timeout,
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

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.ServerConfig, logger *monitoring.Logger) (*gorm.DB, error) {
	dbConfig := &releasemodels.DatabaseConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	}

	logger.Info("Connecting to database",
		"host", dbConfig.Host,
		"port", dbConfig.Port,
		"dbname", dbConfig.DBName)

	db, err := releasemodels.InitDB(dbConfig)
	if err != nil {
		return nil, err
	}

	logger.Info("Database connected successfully")

	// 自动迁移表结构
	if cfg.Database.AutoMigrate {
		logger.Info("Running database migrations...")
		if err := releasemodels.Migrate(db); err != nil {
			return nil, fmt.Errorf("database migration failed: %w", err)
		}
		logger.Info("Database migrations completed")
	}

	return db, nil
}
