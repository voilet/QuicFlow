package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/voilet/QuicFlow/pkg/command"
	"github.com/voilet/QuicFlow/pkg/dispatcher"
	"github.com/voilet/QuicFlow/pkg/monitoring"
	"github.com/voilet/QuicFlow/pkg/protocol"
	"github.com/voilet/QuicFlow/pkg/transport/client"
)

// SimpleCommandExecutor 简单的命令执行器
type SimpleCommandExecutor struct {
	logger *monitoring.Logger
}

// Execute 执行命令
func (e *SimpleCommandExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
	e.logger.Info("Executing command", "command_type", commandType)

	switch commandType {
	case "exec_shell":
		return e.executeShell(payload)
	case "get_status":
		return e.executeGetStatus(payload)
	default:
		return nil, fmt.Errorf("unknown command type: %s", commandType)
	}
}

// executeShell 执行Shell命令
func (e *SimpleCommandExecutor) executeShell(payload []byte) ([]byte, error) {
	var params struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout,omitempty"` // 超时时间（秒），默认30秒
	}
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid exec_shell params: %w", err)
	}

	// 验证命令
	if strings.TrimSpace(params.Command) == "" {
		return nil, fmt.Errorf("command is empty")
	}

	e.logger.Info("执行Shell命令", "command", params.Command)

	// 设置默认超时
	timeout := 30
	if params.Timeout > 0 {
		timeout = params.Timeout
	}
	// 限制最大超时为5分钟
	if timeout > 300 {
		timeout = 300
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 执行Shell命令（使用 sh -c 以支持管道和复杂命令）
	cmd := exec.CommandContext(ctx, "sh", "-c", params.Command)

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()

	// 限制输出大小（最大10KB）
	const maxOutputSize = 10 * 1024
	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	if len(stdoutStr) > maxOutputSize {
		stdoutStr = stdoutStr[:maxOutputSize] + "... (truncated)"
	}
	if len(stderrStr) > maxOutputSize {
		stderrStr = stderrStr[:maxOutputSize] + "... (truncated)"
	}

	// 构建结果
	result := map[string]interface{}{
		"success":   err == nil,
		"exit_code": 0,
		"stdout":    stdoutStr,
		"stderr":    stderrStr,
		"message":   "命令执行成功",
	}

	// 处理错误
	if err != nil {
		result["message"] = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result["message"] = fmt.Sprintf("命令执行超时（%d秒）", timeout)
			result["exit_code"] = -1
		} else {
			result["exit_code"] = -1
		}
	}

	e.logger.Info("Shell命令执行完成",
		"command", params.Command,
		"success", result["success"],
		"exit_code", result["exit_code"],
		"stdout_len", len(stdoutStr),
		"stderr_len", len(stderrStr),
	)

	return json.Marshal(result)
}

// executeGetStatus 执行获取状态命令
func (e *SimpleCommandExecutor) executeGetStatus(payload []byte) ([]byte, error) {
	result := map[string]interface{}{
		"status":  "running",
		"uptime":  3600,
		"version": "1.0.0",
	}

	return json.Marshal(result)
}

func main() {
	// 命令行参数
	serverAddr := flag.String("server", "localhost:8474", "服务器地址")
	clientID := flag.String("id", "client-001", "客户端 ID")
	insecure := flag.Bool("insecure", true, "跳过 TLS 证书验证（仅开发环境）")
	flag.Parse()

	// 创建日志器
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")

	logger.Info("=== QUIC Backbone Client ===")
	logger.Info("Connecting to server", "server", *serverAddr, "client_id", *clientID)

	// 创建客户端配置
	config := client.NewDefaultClientConfig(*clientID)
	config.InsecureSkipVerify = *insecure
	config.Logger = logger

	// 设置事件钩子
	config.Hooks = &monitoring.EventHooks{
		OnConnect: func(clientID string) {
			logger.Info("✅ Connected to server", "client_id", clientID)
		},
		OnDisconnect: func(clientID string, reason error) {
			logger.Warn("❌ Disconnected from server", "client_id", clientID, "reason", reason)
		},
		OnReconnect: func(clientID string, attemptCount int) {
			logger.Info("🔄 Reconnected to server", "client_id", clientID, "attempts", attemptCount)
		},
	}

	// 创建客户端
	c, err := client.NewClient(config)
	if err != nil {
		logger.Error("Failed to create client", "error", err)
		os.Exit(1)
	}

	// 创建命令执行器
	executor := &SimpleCommandExecutor{logger: logger}

	// 创建命令处理器
	commandHandler := command.NewCommandHandler(c, executor, logger)

	// 创建 Dispatcher 并注册命令处理器
	dispatcherConfig := &dispatcher.DispatcherConfig{
		WorkerCount:    10,
		TaskQueueSize:  1000,
		HandlerTimeout: 30 * time.Second,
		Logger:         logger,
	}
	disp := dispatcher.NewDispatcher(dispatcherConfig)

	// 注册命令处理器（包装为 MessageHandler）
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_COMMAND, dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
		return commandHandler.HandleCommand(ctx, msg)
	}))

	// 启动 Dispatcher
	disp.Start()
	logger.Info("✅ Dispatcher started with command handler")

	// 设置 Dispatcher 到客户端（必须在连接之前设置）
	c.SetDispatcher(disp)
	logger.Info("✅ Dispatcher attached to client")

	// 连接到服务器
	if err := c.Connect(*serverAddr); err != nil {
		logger.Error("Failed to connect", "error", err)
		// 不退出，因为启用了自动重连
	}

	logger.Info("Client started (auto-reconnect enabled)")
	logger.Info("🎯 Ready to receive and execute commands")
	logger.Info("Press Ctrl+C to stop")

	// 定期打印状态
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			state := c.GetState()
			metrics := c.GetMetrics()
			lastPong := c.GetTimeSinceLastPong()

			fmt.Printf("\n=== Client Status ===\n")
			fmt.Printf("State: %v\n", state)
			fmt.Printf("Connected: %v\n", c.IsConnected())
			fmt.Printf("Last Pong: %v ago\n", lastPong.Round(time.Second))
			fmt.Printf("Heartbeats Sent: %d\n", metrics.ConnectedClients)
			fmt.Println()
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	logger.Info("Shutting down client...")

	// 停止 Dispatcher
	disp.Stop()
	logger.Info("Dispatcher stopped")

	// 断开连接
	logger.Info("Disconnecting from server...")
	if err := c.Disconnect(); err != nil {
		logger.Error("Error during disconnect", "error", err)
		os.Exit(1)
	}

	logger.Info("Client stopped gracefully")
}
