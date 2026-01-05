package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
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
	serverAddr       string
	clientCount      int
	clientPrefix     string
	concurrency       int
	ratePerSecond     int
	insecure          bool
	keepAlive         bool
	reportInterval    int
	logLevel          string
	testDuration      int
	workloadType      string
	messageInterval   int
	autoReconnect     bool
	reconnectInterval int
)

// ClientStats 客户端统计
type ClientStats struct {
	Connected       int64
	Failed          int64
	Disconnected    int64
	Reconnected     int64
	MessagesRecv    int64
	MessagesSent    int64
	BytesSent       int64
	BytesRecv       int64
	CommandsRecv    int64
	CommandsExec    int64
	CommandsFail    int64
	LatencySumMs    int64
	MinLatencyMs    int64
	MaxLatencyMs    int64
}

// ConnTime 连接时间统计
type ConnTime struct {
	Time time.Time
	Dur  time.Duration
}

var (
	stats      = &ClientStats{}
	clients    = make(map[string]*client.Client)
	clientsMu  sync.RWMutex
	logger     *monitoring.Logger
	connTimes  = make([]time.Duration, 0, 10000)
	connTimesMu sync.Mutex
	startTime  time.Time
)

var rootCmd = &cobra.Command{
	Use:   "quic-loadtest",
	Short: "QUIC 负载测试工具",
	Long: `QUIC 协议服务器负载测试工具

支持功能：
  - 高并发连接测试（支持数万并发）
  - 连接速率控制（每秒连接数）
  - 多种工作负载模式
  - 自动重连测试
  - 详细的性能统计报告

示例:
  # 启动 1 万个客户端，每秒 100 个连接
  quic-loadtest -s 127.0.0.1:8474 -n 10000 -r 100

  # 压力测试：快速建立连接后不保持
  quic-loadtest -n 5000 -r 500 --keep-alive=false

  # 持续负载测试：发送心跳消息
  quic-loadtest -n 1000 -w heartbeat --message-interval 30

  # 自动重连测试
  quic-loadtest -n 500 -r 10 --auto-reconnect --reconnect-interval 60

  # 定时测试（运行 5 分钟后退出）
  quic-loadtest -n 1000 --duration 300`,
	Run: runLoadTest,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		version.Print("quic-loadtest")
	},
}

func init() {
	// 连接参数
	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "127.0.0.1:8474", "服务器地址")
	rootCmd.Flags().IntVarP(&clientCount, "count", "n", 10000, "客户端数量")
	rootCmd.Flags().StringVarP(&clientPrefix, "prefix", "p", "load-client", "客户端 ID 前缀")
	rootCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 500, "并发连接数（同时连接的客户端数）")
	rootCmd.Flags().IntVarP(&ratePerSecond, "rate", "r", 100, "连接速率（每秒建立连接数）")
	rootCmd.Flags().BoolVarP(&insecure, "insecure", "k", true, "跳过 TLS 证书验证")

	// 行为参数
	rootCmd.Flags().BoolVar(&keepAlive, "keep-alive", true, "保持连接（false 表示连接后立即断开）")
	rootCmd.Flags().BoolVar(&autoReconnect, "auto-reconnect", false, "启用自动重连")
	rootCmd.Flags().IntVar(&reconnectInterval, "reconnect-interval", 60, "重连间隔（秒）")
	rootCmd.Flags().IntVar(&testDuration, "duration", 0, "测试持续时间（秒），0 表示无限期")

	// 工作负载参数
	rootCmd.Flags().StringVar(&workloadType, "workload", "idle", "工作负载类型: idle(空闲), heartbeat(心跳), echo(回显), stress(压力)")
	rootCmd.Flags().IntVar(&messageInterval, "message-interval", 30, "消息发送间隔（秒）")

	// 输出参数
	rootCmd.Flags().IntVar(&reportInterval, "report-interval", 5, "状态报告间隔（秒）")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "warn", "日志级别 (debug/info/warn/error)")

	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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

	// 验证参数
	if clientCount <= 0 {
		fmt.Fprintf(os.Stderr, "错误: 客户端数量必须大于 0\n")
		os.Exit(1)
	}
	if concurrency <= 0 {
		concurrency = clientCount
	}
	if ratePerSecond <= 0 {
		ratePerSecond = 100
	}

	// 打印配置
	printHeader()

	// 启动统计报告
	stopReport := make(chan struct{})
	go reportStats(stopReport)

	// 创建速率限制器
	rateLimiter := time.NewTicker(time.Second / time.Duration(ratePerSecond))
	defer rateLimiter.Stop()

	// 测试超时控制
	var testTimer *time.Timer
	if testDuration > 0 {
		testTimer = time.NewTimer(time.Duration(testDuration) * time.Second)
		defer testTimer.Stop()
	}

	// 启动客户端
	startTime = time.Now()
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	connChan := make(chan int, clientCount)
	for i := 0; i < clientCount; i++ {
		connChan <- i
	}
	close(connChan)

	// 连接建立协程
	go func() {
		for idx := range connChan {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				// 速率限制
				<-rateLimiter.C

				// 并发控制
				sem <- struct{}{}
				defer func() { <-sem }()

				clientID := fmt.Sprintf("%s-%05d", clientPrefix, i)
				connStart := time.Now()

				if err := startClient(clientID); err != nil {
					atomic.AddInt64(&stats.Failed, 1)
					if level <= monitoring.LogLevelInfo {
						logger.Warn("连接失败", "client_id", clientID, "error", err)
					}
				} else {
					connDur := time.Since(connStart)
					connTimesMu.Lock()
					connTimes = append(connTimes, connDur)
					connTimesMu.Unlock()

					atomic.AddInt64(&stats.Connected, 1)
				}
			}(idx)
		}
	}()

	// 等待所有客户端连接完成
	wg.Wait()

	connDuration := time.Since(startTime)
	printConnectionSummary(connDuration)

	if !keepAlive {
		fmt.Println("\n正在断开所有连接...")
		disconnectAll()
		close(stopReport)
		return
	}

	// 根据工作负载类型执行
	fmt.Println("\n" + getWorkloadDescription())
	fmt.Println("测试运行中，按 Ctrl+C 停止...")

	// 启动工作负载
	var workloadStop chan struct{}
	workloadStop = make(chan struct{})
	go runWorkload(workloadType, messageInterval, workloadStop)

	// 等待停止信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if testTimer != nil {
		select {
		case <-sigChan:
			fmt.Println("\n收到停止信号...")
		case <-testTimer.C:
			fmt.Println("\n测试时间到达...")
		}
	} else {
		// 没有设置超时，只等待信号
		<-sigChan
		fmt.Println("\n收到停止信号...")
	}

	close(workloadStop)
	close(stopReport)

	fmt.Println("\n正在断开所有连接...")
	disconnectAll()

	// 打印最终统计
	printFinalStats()
}

// startClient 启动单个客户端
func startClient(clientID string) error {
	config := client.NewDefaultClientConfig(clientID)
	config.InsecureSkipVerify = insecure
	config.Logger = logger

	// 配置自动重连
	if autoReconnect {
		config.ReconnectEnabled = true
		config.InitialBackoff = time.Duration(reconnectInterval) * time.Second
		config.MaxBackoff = time.Duration(reconnectInterval) * time.Second * 4
	}

	// 配置心跳
	config.HeartbeatInterval = 30 * time.Second
	config.HeartbeatTimeout = 90 * time.Second

	c, err := client.NewClient(config)
	if err != nil {
		return fmt.Errorf("创建客户端: %w", err)
	}

	// 设置消息处理器
	setupClientDispatcher(c, clientID)

	// 连接服务器
	if err := c.Connect(serverAddr); err != nil {
		return fmt.Errorf("连接服务器: %w", err)
	}

	// 保存客户端引用
	clientsMu.Lock()
	clients[clientID] = c
	clientsMu.Unlock()

	// 连接成功后上报硬件信息
	go func() {
		time.Sleep(500 * time.Millisecond)
		if c.IsConnected() {
			reportHardwareInfo(c, clientID)
		}
	}()

	return nil
}

// reportHardwareInfo 上报硬件信息到服务器
func reportHardwareInfo(c *client.Client, clientID string) {
	// 构建硬件信息结构（与 command.HardwareInfoResult 兼容）
	hwInfo := map[string]interface{}{
		"host": map[string]interface{}{
			"hostname":         clientID,
			"os":               "linux",
			"platform":         "linux",
			"platform_version": "load-test",
			"kernel_version":   "5.10.0-loadtest",
			"kernel_arch":      "x86_64",
			"uptime":           86400,
			"host_id":          clientID,
		},
		"model_name":       "LoadTest CPU",
		"cpu_core_count":   4,
		"cpu_thread_count": 8,
		"memory": map[string]interface{}{
			"total_gb_rounded": 16,
			"total_bytes":      17179869184,
			"total_gb":         16.0,
		},
		"disks": []map[string]interface{}{
			{
				"device":            "sda1",
				"model":             "LoadTest Disk",
				"size_rounded_bytes": 107374182400,
				"size_tb":           0.1,
				"kind":              "ssd",
			},
		},
		"total_disk_capacity_tb_decimal": 0.1,
		"mac": "00:00:00:00:00:00",
		"nic_infos": []map[string]interface{}{
			{
				"name":        "eth0",
				"mac_address": "00:00:00:00:00:00",
				"ip_address":  "192.168.1.100",
				"status":      "up",
			},
		},
	}

	// 构建内层上报消息（使用 hardware.info 命令格式）
	reportMsg := map[string]interface{}{
		"client_id":     clientID,
		"command_type":  "hardware.info",
		"hardware_info": hwInfo,
		"report_type":   "auto_sync",
	}

	// 构建外层命令载荷（使用 router 的命令格式）
	cmdPayload := map[string]interface{}{
		"command_type": "report.hardware",
		"payload":      reportMsg,
	}
	payloadBytes, _ := json.Marshal(cmdPayload)

	msg := &protocol.DataMessage{
		MsgId:      fmt.Sprintf("hw-%d", time.Now().UnixNano()),
		SenderId:   clientID,
		ReceiverId: "server",
		Type:       protocol.MessageType_MESSAGE_TYPE_EVENT,
		Payload:    payloadBytes,
		Timestamp:  time.Now().UnixMilli(),
	}

	if err := c.SendMessageAsync(msg); err != nil {
		logger.Warn("Failed to report hardware info", "client_id", clientID, "error", err)
	}
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
			start := time.Now()
			atomic.AddInt64(&stats.CommandsRecv, 1)

			resp, err := commandHandler.HandleCommand(ctx, msg)
			latency := time.Since(start)
			latencyMs := latency.Milliseconds()

			// 更新延迟统计
			updateLatencyStats(latencyMs)

			if err != nil {
				atomic.AddInt64(&stats.CommandsFail, 1)
				return resp, err
			}

			atomic.AddInt64(&stats.CommandsExec, 1)
			return resp, nil
		}))

	// 注册事件处理器（支持心跳和消息接收）
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_EVENT,
		dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
			atomic.AddInt64(&stats.MessagesRecv, 1)
			atomic.AddInt64(&stats.BytesRecv, int64(len(msg.Payload)))
			return nil, nil
		}))

	// 注册查询处理器
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_QUERY,
		dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
			atomic.AddInt64(&stats.CommandsRecv, 1)
			// 对于查询，返回一个简单的响应
			return &protocol.DataMessage{
				MsgId:      msg.MsgId,
				SenderId:   clientID,
				ReceiverId: msg.SenderId,
				Type:       protocol.MessageType_MESSAGE_TYPE_RESPONSE,
				Payload:    []byte(`{"status":"ok"}`),
				Timestamp:  time.Now().UnixMilli(),
			}, nil
		}))

	disp.Start()
	c.SetDispatcher(disp)
}

// updateLatencyStats 更新延迟统计
func updateLatencyStats(latencyMs int64) {
	atomic.AddInt64(&stats.LatencySumMs, latencyMs)

	// 更新最小值
	for {
		min := atomic.LoadInt64(&stats.MinLatencyMs)
		if min == 0 || latencyMs >= min {
			if atomic.CompareAndSwapInt64(&stats.MinLatencyMs, min, latencyMs) {
				break
			}
		} else {
			break
		}
	}

	// 更新最大值
	for {
		max := atomic.LoadInt64(&stats.MaxLatencyMs)
		if latencyMs <= max {
			if atomic.CompareAndSwapInt64(&stats.MaxLatencyMs, max, latencyMs) {
				break
			}
		} else {
			break
		}
	}
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
	sem := make(chan struct{}, 200)

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

// runWorkload 运行工作负载
func runWorkload(workload string, interval int, stop chan struct{}) {
	if interval <= 0 {
		interval = 30
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	switch workload {
	case "idle":
		// 空闲模式：只保持连接
		<-stop

	case "heartbeat":
		// 心跳模式：定期发送心跳消息
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				sendHeartbeat()
			}
		}

	case "echo":
		// 回显模式：发送消息并等待响应
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				sendEchoMessage()
			}
		}

	case "stress":
		// 压力模式：高频发送消息
		fastTicker := time.NewTicker(1 * time.Second)
		defer fastTicker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-fastTicker.C:
				sendStressMessage()
			}
		}
	}
}

// sendHeartbeat 发送心跳消息
func sendHeartbeat() {
	payload := map[string]interface{}{
		"type":      "heartbeat",
		"timestamp": time.Now().Unix(),
		"sequence":  atomic.LoadInt64(&stats.MessagesSent),
	}

	broadcastMessage("heartbeat", payload)
}

// sendEchoMessage 发送回显消息
func sendEchoMessage() {
	payload := map[string]interface{}{
		"type":      "echo",
		"timestamp": time.Now().Unix(),
		"data":      generateRandomData(128),
	}

	broadcastMessage("echo", payload)
}

// sendStressMessage 发送压力测试消息
func sendStressMessage() {
	payload := map[string]interface{}{
		"type":      "stress",
		"timestamp": time.Now().Unix(),
		"data":      generateRandomData(1024),
	}

	broadcastMessage("stress", payload)
}

// broadcastMessage 向所有客户端广播消息
func broadcastMessage(msgType string, payload map[string]interface{}) {
	clientsMu.RLock()
	clientList := make([]*client.Client, 0, len(clients))
	for _, c := range clients {
		clientList = append(clientList, c)
	}
	clientsMu.RUnlock()

	payload["client_id"] = "loadtest"
	payloadBytes, _ := json.Marshal(payload)

	cmdPayload := &command.CommandPayload{
		CommandType: msgType,
		Payload:     json.RawMessage(payloadBytes),
	}
	cmdBytes, _ := json.Marshal(cmdPayload)

	for _, c := range clientList {
		msg := &protocol.DataMessage{
			MsgId:      fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			SenderId:   "loadtest",
			Type:       protocol.MessageType_MESSAGE_TYPE_EVENT,
			Payload:    cmdBytes,
			Timestamp:  time.Now().UnixMilli(),
		}

		c.SendMessageAsync(msg)

		atomic.AddInt64(&stats.MessagesSent, 1)
		atomic.AddInt64(&stats.BytesSent, int64(len(cmdBytes)))
	}
}

// generateRandomData 生成随机数据
func generateRandomData(size int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// reportStats 定期报告统计信息
func reportStats(stop chan struct{}) {
	ticker := time.NewTicker(time.Duration(reportInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			printStats()
		}
	}
}

// printStats 打印实时统计
func printStats() {
	connected := atomic.LoadInt64(&stats.Connected)
	failed := atomic.LoadInt64(&stats.Failed)
	disconnected := atomic.LoadInt64(&stats.Disconnected)
	reconnected := atomic.LoadInt64(&stats.Reconnected)
	msgRecv := atomic.LoadInt64(&stats.MessagesRecv)
	msgSent := atomic.LoadInt64(&stats.MessagesSent)
	cmdRecv := atomic.LoadInt64(&stats.CommandsRecv)
	cmdExec := atomic.LoadInt64(&stats.CommandsExec)
	cmdFail := atomic.LoadInt64(&stats.CommandsFail)

	// 获取当前活跃连接数
	clientsMu.RLock()
	activeCount := len(clients)
	clientsMu.RUnlock()

	// 计算延迟统计
	totalCmds := cmdRecv
	var avgLatency float64
	if totalCmds > 0 {
		avgLatency = float64(atomic.LoadInt64(&stats.LatencySumMs)) / float64(totalCmds)
	}

	elapsed := time.Since(startTime)
	rate := float64(connected) / elapsed.Seconds()

	fmt.Printf("[%s] 连接: %d/%d (失败: %d 断开: %d) | 活跃: %d | 重连: %d | 速率: %.1f/s\n",
		time.Now().Format("15:04:05"),
		connected, clientCount, failed, disconnected, activeCount, reconnected, rate)
	fmt.Printf("       消息: 发送=%d 接收=%d | 命令: 收=%d 执行=%d 失败=%d | 延迟: %.1fms\n",
		msgSent, msgRecv, cmdRecv, cmdExec, cmdFail, avgLatency)
}

// printHeader 打印配置信息
func printHeader() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   QUIC 负载测试工具                            ║")
	fmt.Println("║                                                              ║")
	fmt.Printf("║  服务器:       %-20s                           ║\n", serverAddr)
	fmt.Printf("║  客户端数量:   %-20d                           ║\n", clientCount)
	fmt.Printf("║  并发连接:     %-20d                           ║\n", concurrency)
	fmt.Printf("║  连接速率:     %-20d 连接/秒                    ║\n", ratePerSecond)
	fmt.Printf("║  保持连接:     %-20v                           ║\n", keepAlive)
	fmt.Printf("║  工作负载:     %-20s                           ║\n", workloadType)
	fmt.Printf("║  自动重连:     %-20v                           ║\n", autoReconnect)
	if testDuration > 0 {
		fmt.Printf("║  测试时长:     %-20d 秒                        ║\n", testDuration)
	}
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// printConnectionSummary 打印连接摘要
func printConnectionSummary(duration time.Duration) {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                       连接完成                                 ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  总耗时:       %-20s                           ║\n", duration.Round(time.Millisecond))
	fmt.Printf("║  成功连接:     %-20d                           ║\n", atomic.LoadInt64(&stats.Connected))
	fmt.Printf("║  连接失败:     %-20d                           ║\n", atomic.LoadInt64(&stats.Failed))
	fmt.Printf("║  连接速率:     %-20.2f 连接/秒                 ║\n", float64(atomic.LoadInt64(&stats.Connected))/duration.Seconds())

	// 计算连接时间分位数
	if len(connTimes) > 0 {
		connTimesMu.Lock()
		times := make([]time.Duration, len(connTimes))
		copy(times, connTimes)
		connTimesMu.Unlock()

		// 简单排序计算百分位
		for i := 0; i < len(times); i++ {
			for j := i + 1; j < len(times); j++ {
				if times[i] > times[j] {
					times[i], times[j] = times[j], times[i]
				}
			}
		}

		p50 := times[len(times)*50/100]
		p95 := times[len(times)*95/100]
		p99 := times[len(times)*99/100]

		fmt.Println("║  连接延迟:                                                    ║")
		fmt.Printf("║    P50:        %-20s                          ║\n", p50.Round(time.Millisecond))
		fmt.Printf("║    P95:        %-20s                          ║\n", p95.Round(time.Millisecond))
		fmt.Printf("║    P99:        %-20s                          ║\n", p99.Round(time.Millisecond))
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

// printFinalStats 打印最终统计
func printFinalStats() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                       最终统计                                 ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	totalDuration := time.Since(startTime)
	fmt.Printf("║  总运行时间:   %-20s                           ║\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("║  连接成功:     %-20d                           ║\n", atomic.LoadInt64(&stats.Connected))
	fmt.Printf("║  连接失败:     %-20d                           ║\n", atomic.LoadInt64(&stats.Failed))
	fmt.Printf("║  断开连接:     %-20d                           ║\n", atomic.LoadInt64(&stats.Disconnected))
	fmt.Printf("║  重连次数:     %-20d                           ║\n", atomic.LoadInt64(&stats.Reconnected))

	fmt.Println("║                                                              ║")
	fmt.Printf("║  消息发送:     %-20d                           ║\n", atomic.LoadInt64(&stats.MessagesSent))
	fmt.Printf("║  消息接收:     %-20d                           ║\n", atomic.LoadInt64(&stats.MessagesRecv))
	fmt.Printf("║  字节发送:     %-20d                           ║\n", atomic.LoadInt64(&stats.BytesSent))
	fmt.Printf("║  字节接收:     %-20d                           ║\n", atomic.LoadInt64(&stats.BytesRecv))

	fmt.Println("║                                                              ║")
	fmt.Printf("║  命令接收:     %-20d                           ║\n", atomic.LoadInt64(&stats.CommandsRecv))
	fmt.Printf("║  命令执行:     %-20d                           ║\n", atomic.LoadInt64(&stats.CommandsExec))
	fmt.Printf("║  命令失败:     %-20d                           ║\n", atomic.LoadInt64(&stats.CommandsFail))

	// 延迟统计
	totalCmds := atomic.LoadInt64(&stats.CommandsRecv)
	if totalCmds > 0 {
		avgLatency := float64(atomic.LoadInt64(&stats.LatencySumMs)) / float64(totalCmds)
		minLatency := atomic.LoadInt64(&stats.MinLatencyMs)
		maxLatency := atomic.LoadInt64(&stats.MaxLatencyMs)

		fmt.Println("║                                                              ║")
		fmt.Println("║  命令延迟:                                                    ║")
		fmt.Printf("║    平均:       %-20.2f ms                       ║\n", avgLatency)
		if minLatency > 0 {
			fmt.Printf("║    最小:       %-20d ms                        ║\n", minLatency)
		}
		if maxLatency > 0 {
			fmt.Printf("║    最大:       %-20d ms                        ║\n", maxLatency)
		}
	}

	// 吞吐量统计
	if totalDuration.Seconds() > 0 {
		fmt.Println("║                                                              ║")
		msgSent := atomic.LoadInt64(&stats.MessagesSent)
		msgRecv := atomic.LoadInt64(&stats.MessagesRecv)
		bytesSent := atomic.LoadInt64(&stats.BytesSent)
		bytesRecv := atomic.LoadInt64(&stats.BytesRecv)

		fmt.Printf("║  消息吞吐:     %.2f msg/s (发送) / %.2f msg/s (接收)       ║\n",
			float64(msgSent)/totalDuration.Seconds(),
			float64(msgRecv)/totalDuration.Seconds())
		fmt.Printf("║  数据吞吐:     %.2f KB/s (发送) / %.2f KB/s (接收)       ║\n",
			float64(bytesSent)/1024/totalDuration.Seconds(),
			float64(bytesRecv)/1024/totalDuration.Seconds())
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

// getWorkloadDescription 获取工作负载描述
func getWorkloadDescription() string {
	descriptions := map[string]string{
		"idle":     "工作负载: 空闲 (仅保持连接)",
		"heartbeat": "工作负载: 心跳 (定期发送心跳消息)",
		"echo":     "工作负载: 回显 (发送消息并等待响应)",
		"stress":   "工作负载: 压力 (高频发送消息)",
	}
	if desc, ok := descriptions[workloadType]; ok {
		return desc
	}
	return "工作负载: 未知"
}
