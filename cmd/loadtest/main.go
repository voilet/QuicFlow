package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/dispatcher"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/protocol"
	"github.com/voilet/quic-flow/pkg/router"
	"github.com/voilet/quic-flow/pkg/router/handlers"
	"github.com/voilet/quic-flow/pkg/transport/client"
	"github.com/voilet/quic-flow/pkg/version"
)

var (
	// 命令行参数
	serverAddr     string
	clientCount    int
	clientPrefix   string
	concurrency    int
	insecure       bool
	keepAlive      bool
	reportInterval int
	logLevel       string
)

// ClientStats 客户端统计
type ClientStats struct {
	Connected     int64
	Failed        int64
	Disconnected  int64
	CommandsRecv  int64
	CommandsExec  int64
	CommandsFail  int64
}

var stats = &ClientStats{}
var clients = make(map[string]*client.Client)
var clientsMu sync.RWMutex
var logger *monitoring.Logger

var rootCmd = &cobra.Command{
	Use:   "quic-loadtest",
	Short: "QUIC Load Test Tool",
	Long: `批量启动客户端连接到 QUIC 服务器进行负载测试

示例:
  # 启动 1 万个客户端并保持连接
  quic-loadtest -s 127.0.0.1:8474 -n 10000 -c 200

  # 生成客户端 ID 列表到文件
  quic-loadtest generate -n 10000 -o clients.txt

  # 连接后不保持（压力测试）
  quic-loadtest -n 1000 --keep-alive=false`,
	Run: runLoadTest,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		version.Print("quic-loadtest")
	},
}

var (
	outputFile string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "生成客户端 ID 列表到文件",
	Long:  "生成客户端 ID 列表并保存到文件，用于后续导入或批量下发",
	Run:   runGenerate,
}

func init() {
	// 主命令参数
	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "127.0.0.1:8474", "服务器地址")
	rootCmd.Flags().IntVarP(&clientCount, "count", "n", 10000, "客户端数量")
	rootCmd.Flags().StringVarP(&clientPrefix, "prefix", "p", "load-client", "客户端 ID 前缀")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 100, "并发连接数")
	rootCmd.Flags().BoolVarP(&insecure, "insecure", "k", true, "跳过 TLS 验证")
	rootCmd.Flags().BoolVar(&keepAlive, "keep-alive", true, "保持连接")
	rootCmd.Flags().IntVar(&reportInterval, "report-interval", 5, "状态报告间隔（秒）")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "warn", "日志级别 (debug/info/warn/error)")

	// 生成命令参数
	generateCmd.Flags().IntVarP(&clientCount, "count", "n", 10000, "客户端数量")
	generateCmd.Flags().StringVarP(&clientPrefix, "prefix", "p", "load-client", "客户端 ID 前缀")
	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "输出文件路径（默认输出到标准输出）")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(generateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runGenerate 生成客户端 ID 列表
func runGenerate(cmd *cobra.Command, args []string) {
	// 生成 ID 列表
	var ids []string
	for i := 0; i < clientCount; i++ {
		ids = append(ids, fmt.Sprintf("%s-%05d", clientPrefix, i))
	}

	// 输出到文件或标准输出
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建文件失败: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		for _, id := range ids {
			fmt.Fprintln(f, id)
		}
		fmt.Printf("已生成 %d 个客户端 ID 到文件: %s\n", clientCount, outputFile)
	} else {
		fmt.Printf("# 生成 %d 个客户端 ID (前缀: %s)\n", clientCount, clientPrefix)
		for _, id := range ids {
			fmt.Println(id)
		}
	}
}

// runLoadTest 运行负载测试
func runLoadTest(cmd *cobra.Command, args []string) {
	// 初始化日志
	level := monitoring.LogLevelWarn
	switch logLevel {
	case "debug":
		level = monitoring.LogLevelDebug
	case "info":
		level = monitoring.LogLevelInfo
	case "error":
		level = monitoring.LogLevelError
	}
	logger = monitoring.NewLogger(level, "text")

	fmt.Println("=== QUIC Load Test Tool ===")
	fmt.Printf("Server:      %s\n", serverAddr)
	fmt.Printf("Clients:     %d\n", clientCount)
	fmt.Printf("Prefix:      %s\n", clientPrefix)
	fmt.Printf("Concurrency: %d\n", concurrency)
	fmt.Printf("Keep-Alive:  %v\n", keepAlive)
	fmt.Println()

	// 启动状态报告
	go reportStats()

	// 创建信号量控制并发
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	startTime := time.Now()

	// 并发启动客户端
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			clientID := fmt.Sprintf("%s-%05d", clientPrefix, idx)
			if err := startClient(clientID); err != nil {
				atomic.AddInt64(&stats.Failed, 1)
				if level <= monitoring.LogLevelInfo {
					logger.Error("Failed to start client", "client_id", clientID, "error", err)
				}
			} else {
				atomic.AddInt64(&stats.Connected, 1)
			}
		}(i)
	}

	// 等待所有客户端启动完成
	wg.Wait()

	duration := time.Since(startTime)
	fmt.Printf("\n=== 连接完成 ===\n")
	fmt.Printf("总耗时:   %v\n", duration.Round(time.Millisecond))
	fmt.Printf("成功:     %d\n", stats.Connected)
	fmt.Printf("失败:     %d\n", stats.Failed)
	fmt.Printf("连接速率: %.2f/s\n", float64(stats.Connected)/duration.Seconds())

	if !keepAlive {
		fmt.Println("\n正在断开所有连接...")
		disconnectAll()
		return
	}

	fmt.Println("\n客户端保持连接中，按 Ctrl+C 退出...")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在断开所有连接...")
	disconnectAll()

	fmt.Printf("\n=== 最终统计 ===\n")
	fmt.Printf("命令接收: %d\n", stats.CommandsRecv)
	fmt.Printf("命令执行: %d\n", stats.CommandsExec)
	fmt.Printf("命令失败: %d\n", stats.CommandsFail)
}

// startClient 启动单个客户端
func startClient(clientID string) error {
	config := client.NewDefaultClientConfig(clientID)
	config.InsecureSkipVerify = insecure
	config.Logger = logger
	config.ReconnectEnabled = true
	config.HeartbeatInterval = 30 * time.Second
	config.HeartbeatTimeout = 90 * time.Second

	c, err := client.NewClient(config)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// 设置消息处理器
	setupClientDispatcher(c, clientID)

	// 连接服务器
	if err := c.Connect(serverAddr); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	// 保存客户端引用
	clientsMu.Lock()
	clients[clientID] = c
	clientsMu.Unlock()

	return nil
}

// setupClientDispatcher 设置客户端消息处理器
func setupClientDispatcher(c *client.Client, clientID string) {
	// 创建命令路由器
	cmdRouter := router.NewRouter(logger)
	handlers.RegisterBuiltinHandlers(cmdRouter, &handlers.Config{
		Version: version.String(),
	})

	// 创建 Dispatcher
	dispConfig := &dispatcher.DispatcherConfig{
		WorkerCount:    2,
		TaskQueueSize:  50,
		HandlerTimeout: 30 * time.Second,
		Logger:         logger,
	}
	disp := dispatcher.NewDispatcher(dispConfig)

	// 创建命令处理器
	commandHandler := command.NewCommandHandler(c, cmdRouter, logger)

	// 注册命令处理器
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_COMMAND,
		dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
			atomic.AddInt64(&stats.CommandsRecv, 1)

			resp, err := commandHandler.HandleCommand(ctx, msg)
			if err != nil {
				atomic.AddInt64(&stats.CommandsFail, 1)
				return resp, err
			}

			atomic.AddInt64(&stats.CommandsExec, 1)
			return resp, nil
		}))

	// 注册事件处理器
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_EVENT,
		dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
			atomic.AddInt64(&stats.CommandsRecv, 1)
			return nil, nil
		}))

	disp.Start()
	c.SetDispatcher(disp)
}

// disconnectAll 断开所有客户端
func disconnectAll() {
	clientsMu.RLock()
	clientList := make([]*client.Client, 0, len(clients))
	for _, c := range clients {
		clientList = append(clientList, c)
	}
	clientsMu.RUnlock()

	var wg sync.WaitGroup
	sem := make(chan struct{}, 100)

	for _, c := range clientList {
		wg.Add(1)
		go func(cli *client.Client) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			cli.Disconnect()
			atomic.AddInt64(&stats.Disconnected, 1)
		}(c)
	}

	wg.Wait()
}

// reportStats 定期报告统计信息
func reportStats() {
	ticker := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		connected := atomic.LoadInt64(&stats.Connected)
		failed := atomic.LoadInt64(&stats.Failed)
		cmdRecv := atomic.LoadInt64(&stats.CommandsRecv)
		cmdExec := atomic.LoadInt64(&stats.CommandsExec)
		cmdFail := atomic.LoadInt64(&stats.CommandsFail)

		// 获取当前活跃连接数
		clientsMu.RLock()
		activeCount := len(clients)
		clientsMu.RUnlock()

		fmt.Printf("[%s] 连接: %d/%d (失败: %d) | 活跃: %d | 命令: 收=%d 成功=%d 失败=%d\n",
			time.Now().Format("15:04:05"),
			connected, clientCount, failed,
			activeCount,
			cmdRecv, cmdExec, cmdFail)
	}
}

// GetClientIDs 获取所有客户端 ID（供外部调用）
func GetClientIDs() []string {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	ids := make([]string, 0, len(clients))
	for id := range clients {
		ids = append(ids, id)
	}
	return ids
}

// SendCommandToAll 向所有客户端发送命令（供外部调用）
func SendCommandToAll(cmdType string, payload json.RawMessage) (int, int) {
	clientsMu.RLock()
	clientList := make([]*client.Client, 0, len(clients))
	for _, c := range clients {
		clientList = append(clientList, c)
	}
	clientsMu.RUnlock()

	var success, failed int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, 100)

	for _, c := range clientList {
		wg.Add(1)
		go func(cli *client.Client) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			cmdPayload := &command.CommandPayload{
				CommandType: cmdType,
				Payload:     payload,
			}
			cmdBytes, _ := json.Marshal(cmdPayload)

			msg := &protocol.DataMessage{
				MsgId:     fmt.Sprintf("cmd-%d", time.Now().UnixNano()),
				SenderId:  "loadtest",
				Type:      protocol.MessageType_MESSAGE_TYPE_COMMAND,
				Payload:   cmdBytes,
				Timestamp: time.Now().UnixMilli(),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err := cli.SendMessage(ctx, msg, false, 0)
			if err != nil {
				atomic.AddInt64(&failed, 1)
			} else {
				atomic.AddInt64(&success, 1)
			}
		}(c)
	}

	wg.Wait()
	return int(success), int(failed)
}
