package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/dispatcher"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/protocol"
	"github.com/voilet/quic-flow/pkg/router"
	"github.com/voilet/quic-flow/pkg/router/handlers"
	"github.com/voilet/quic-flow/pkg/transport/client"
)

// 服务端地址（需要外部启动服务端）
// 容器内使用 host.docker.internal 访问宿主机
// 宿主机本地使用 127.0.0.1
const testServerAddr = "127.0.0.1:8474"

// LoadTestConfig 负载测试配置
type LoadTestConfig struct {
	ClientCount        int           // 客户端数量
	CommandsPerClient  int           // 每客户端命令数
	ConnectionTimeout  time.Duration // 连接超时
	CommandTimeout     time.Duration // 命令超时
	RampUpDuration     time.Duration // 客户端启动间隔
	ConcurrencyLimit   int           // 并发连接限制
}

// LoadTestResult 负载测试结果
type LoadTestResult struct {
	TotalClients     int           `json:"total_clients"`
	ConnectedClients int64         `json:"connected_clients"`
	FailedClients    int64         `json:"failed_clients"`
	TotalCommands    int           `json:"total_commands"`
	SuccessCommands  int64         `json:"success_commands"`
	FailedCommands   int64         `json:"failed_commands"`
	Duration         time.Duration `json:"duration"`
	ConnectRate      float64       `json:"connect_rate"`      // 连接/秒
	CommandRate      float64       `json:"command_rate"`      // 命令/秒
	AvgLatency       time.Duration `json:"avg_latency"`       // 平均延迟
}

// TestConcurrentClientConnections 测试多个客户端并发连接
func TestConcurrentClientConnections(t *testing.T) {
	testCases := []struct {
		name        string
		clientCount int
	}{
		{"10_clients", 10},
		{"50_clients", 50},
		{"100_clients", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := testConcurrentConnections(t, tc.clientCount)
			t.Logf("Result: connected=%d, failed=%d, rate=%.2f/s",
				result.ConnectedClients, result.FailedClients, result.ConnectRate)
		})
	}
}

// testConcurrentConnections 测试指定数量的并发连接
func testConcurrentConnections(t *testing.T, clientCount int) *LoadTestResult {
	result := &LoadTestResult{
		TotalClients: clientCount,
	}

	var wg sync.WaitGroup
	clients := make([]*client.Client, clientCount)
	startTime := time.Now()

	// 并发创建客户端
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			clientID := fmt.Sprintf("test-client-%d-%d", time.Now().UnixNano(), idx)
			c, err := createTestClient(clientID)
			if err != nil {
				atomic.AddInt64(&result.FailedClients, 1)
				t.Logf("Failed to create client %s: %v", clientID, err)
				return
			}

			// 连接服务器
			if err := c.Connect(testServerAddr); err != nil {
				atomic.AddInt64(&result.FailedClients, 1)
				t.Logf("Failed to connect client %s: %v", clientID, err)
				return
			}

			clients[idx] = c
			atomic.AddInt64(&result.ConnectedClients, 1)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	if result.ConnectedClients > 0 {
		result.ConnectRate = float64(result.ConnectedClients) / result.Duration.Seconds()
	}

	t.Logf("Connection results: success=%d, failed=%d, time=%v",
		result.ConnectedClients, result.FailedClients, result.Duration)

	// 等待所有客户端注册
	time.Sleep(500 * time.Millisecond)

	// 清理客户端
	for _, c := range clients {
		if c != nil {
			c.Disconnect()
		}
	}

	// 等待断开完成
	time.Sleep(200 * time.Millisecond)

	if result.FailedClients > 0 {
		t.Errorf("Some connections failed: %d/%d", result.FailedClients, clientCount)
	}

	return result
}

// TestConcurrentCommands 测试并发向服务端发送命令
func TestConcurrentCommands(t *testing.T) {
	config := LoadTestConfig{
		ClientCount:       20,
		CommandsPerClient: 5,
		ConnectionTimeout: 10 * time.Second,
		CommandTimeout:    10 * time.Second,
	}

	result := runLoadTest(t, config)

	t.Logf("Command test results:")
	t.Logf("  Total commands: %d", result.TotalCommands)
	t.Logf("  Success: %d", result.SuccessCommands)
	t.Logf("  Failed: %d", result.FailedCommands)
	t.Logf("  Duration: %v", result.Duration)
	t.Logf("  Rate: %.2f commands/sec", result.CommandRate)
}

// runLoadTest 运行负载测试
func runLoadTest(t *testing.T, config LoadTestConfig) *LoadTestResult {
	result := &LoadTestResult{
		TotalClients:  config.ClientCount,
		TotalCommands: config.ClientCount * config.CommandsPerClient,
	}

	// 创建并连接客户端
	clients := make([]*client.Client, config.ClientCount)

	for i := 0; i < config.ClientCount; i++ {
		clientID := fmt.Sprintf("load-client-%d-%d", time.Now().UnixNano(), i)
		c, err := createTestClient(clientID)
		if err != nil {
			t.Fatalf("Failed to create client %s: %v", clientID, err)
		}

		// 设置命令处理器
		setupClientDispatcher(c)

		if err := c.Connect(testServerAddr); err != nil {
			t.Fatalf("Failed to connect client %s: %v", clientID, err)
		}
		clients[i] = c
		atomic.AddInt64(&result.ConnectedClients, 1)
	}

	// 等待连接建立
	time.Sleep(500 * time.Millisecond)
	t.Logf("All %d clients connected", config.ClientCount)

	// 并发发送命令
	var wg sync.WaitGroup
	var totalLatency int64

	startTime := time.Now()

	for i := 0; i < config.ClientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			c := clients[idx]
			if c == nil {
				return
			}

			for j := 0; j < config.CommandsPerClient; j++ {
				cmdStart := time.Now()

				// 构造上报消息
				reportPayload := map[string]interface{}{
					"client_id": c.GetClientID(),
					"status":    "running",
					"timestamp": time.Now().UnixMilli(),
					"metrics": map[string]interface{}{
						"cpu":    50.5,
						"memory": 60.2,
					},
				}
				payloadBytes, _ := json.Marshal(reportPayload)

				cmdPayload := &command.CommandPayload{
					CommandType: "report.status",
					Payload:     payloadBytes,
				}
				cmdBytes, _ := json.Marshal(cmdPayload)

				msg := &protocol.DataMessage{
					MsgId:      uuid.New().String(),
					SenderId:   c.GetClientID(),
					ReceiverId: "server",
					Type:       protocol.MessageType_MESSAGE_TYPE_COMMAND,
					Payload:    cmdBytes,
					Timestamp:  time.Now().UnixMilli(),
					WaitAck:    true,
				}

				// 发送命令并等待响应
				ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
				ack, err := c.SendMessage(ctx, msg, true, config.CommandTimeout)
				cancel()

				latency := time.Since(cmdStart)
				atomic.AddInt64(&totalLatency, int64(latency))

				if err != nil {
					atomic.AddInt64(&result.FailedCommands, 1)
				} else if ack != nil && ack.Status == protocol.AckStatus_ACK_STATUS_SUCCESS {
					atomic.AddInt64(&result.SuccessCommands, 1)
				} else {
					atomic.AddInt64(&result.FailedCommands, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	// 计算统计数据
	totalCmds := result.SuccessCommands + result.FailedCommands
	if totalCmds > 0 {
		result.CommandRate = float64(result.SuccessCommands) / result.Duration.Seconds()
		result.AvgLatency = time.Duration(totalLatency / totalCmds)
	}

	// 清理
	for _, c := range clients {
		if c != nil {
			c.Disconnect()
		}
	}

	return result
}

// TestConcurrentBroadcast 测试接收服务端广播消息
func TestConcurrentBroadcast(t *testing.T) {
	clientCount := 30
	clients := make([]*client.Client, clientCount)
	receivedMsgs := make([]int32, clientCount)

	for i := 0; i < clientCount; i++ {
		idx := i
		clientID := fmt.Sprintf("broadcast-client-%d-%d", time.Now().UnixNano(), i)
		c, err := createTestClient(clientID)
		if err != nil {
			t.Fatalf("Failed to create client %s: %v", clientID, err)
		}

		// 设置消息接收计数器
		setupClientDispatcherWithCounter(c, &receivedMsgs[idx])

		if err := c.Connect(testServerAddr); err != nil {
			t.Fatalf("Failed to connect client %s: %v", clientID, err)
		}
		clients[i] = c
	}

	time.Sleep(500 * time.Millisecond)
	t.Logf("All %d clients connected, waiting for broadcast messages...", clientCount)

	// 等待接收广播消息（需要服务端发送广播）
	time.Sleep(5 * time.Second)

	// 统计接收情况
	var totalReceived int32
	for i := 0; i < clientCount; i++ {
		totalReceived += receivedMsgs[i]
	}

	t.Logf("Broadcast results: total received=%d messages across %d clients", totalReceived, clientCount)

	// 清理
	for _, c := range clients {
		if c != nil {
			c.Disconnect()
		}
	}
}

// TestHighConcurrencyConnections 高并发连接测试
func TestHighConcurrencyConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")
	}

	testCases := []struct {
		name        string
		clientCount int
	}{
		{"500_clients", 500},
		{"1000_clients", 1000},
		{"2000_clients", 2000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := testHighConcurrencyConnections(t, tc.clientCount)
			successRate := float64(result.ConnectedClients) / float64(tc.clientCount) * 100
			t.Logf("Result: connected=%d/%d (%.1f%%), rate=%.2f/s",
				result.ConnectedClients, tc.clientCount, successRate, result.ConnectRate)

			// 至少 90% 成功率
			if successRate < 90 {
				t.Errorf("Connection success rate too low: %.1f%%", successRate)
			}
		})
	}
}

// testHighConcurrencyConnections 测试高并发连接
func testHighConcurrencyConnections(t *testing.T, clientCount int) *LoadTestResult {
	result := &LoadTestResult{
		TotalClients: clientCount,
	}

	// 使用信号量控制并发
	concurrencyLimit := 100
	sem := make(chan struct{}, concurrencyLimit)

	var wg sync.WaitGroup
	clients := make([]*client.Client, clientCount)
	startTime := time.Now()

	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			clientID := fmt.Sprintf("high-load-client-%d-%d", time.Now().UnixNano(), idx)
			c, err := createTestClient(clientID)
			if err != nil {
				atomic.AddInt64(&result.FailedClients, 1)
				return
			}

			if err := c.Connect(testServerAddr); err != nil {
				atomic.AddInt64(&result.FailedClients, 1)
				return
			}

			clients[idx] = c
			atomic.AddInt64(&result.ConnectedClients, 1)
		}(i)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	if result.ConnectedClients > 0 {
		result.ConnectRate = float64(result.ConnectedClients) / result.Duration.Seconds()
	}

	t.Logf("High concurrency results: success=%d, failed=%d, time=%v, rate=%.2f/s",
		result.ConnectedClients, result.FailedClients, result.Duration, result.ConnectRate)

	// 等待稳定
	time.Sleep(time.Second)

	// 清理客户端
	for _, c := range clients {
		if c != nil {
			c.Disconnect()
		}
	}

	time.Sleep(500 * time.Millisecond)

	return result
}

// BenchmarkConcurrentConnections 基准测试并发连接
func BenchmarkConcurrentConnections(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("bench-client-%d-%d", time.Now().UnixNano(), i)
		c, err := createTestClient(clientID)
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}

		if err := c.Connect(testServerAddr); err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}

		c.Disconnect()
	}
}

// BenchmarkCommandSend 基准测试命令发送
func BenchmarkCommandSend(b *testing.B) {
	// 创建一个客户端
	clientID := fmt.Sprintf("bench-cmd-client-%d", time.Now().UnixNano())
	c, err := createTestClient(clientID)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	setupClientDispatcher(c)

	if err := c.Connect(testServerAddr); err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect()

	time.Sleep(500 * time.Millisecond)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reportPayload := map[string]interface{}{
			"client_id": clientID,
			"status":    "running",
			"timestamp": time.Now().UnixMilli(),
		}
		payloadBytes, _ := json.Marshal(reportPayload)

		cmdPayload := &command.CommandPayload{
			CommandType: "report.status",
			Payload:     payloadBytes,
		}
		cmdBytes, _ := json.Marshal(cmdPayload)

		msg := &protocol.DataMessage{
			MsgId:      uuid.New().String(),
			SenderId:   clientID,
			ReceiverId: "server",
			Type:       protocol.MessageType_MESSAGE_TYPE_COMMAND,
			Payload:    cmdBytes,
			Timestamp:  time.Now().UnixMilli(),
			WaitAck:    true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := c.SendMessage(ctx, msg, true, 10*time.Second)
		cancel()

		if err != nil {
			b.Logf("Command failed: %v", err)
		}
	}
}

// ============================================================================
// Helper functions
// ============================================================================

func createTestClient(clientID string) (*client.Client, error) {
	logger := monitoring.NewLogger(monitoring.LogLevelError, "text")

	config := client.NewDefaultClientConfig(clientID)
	config.InsecureSkipVerify = true
	config.Logger = logger
	config.ReconnectEnabled = false // 测试时禁用自动重连
	config.HeartbeatInterval = 10 * time.Second

	return client.NewClient(config)
}

func setupClientDispatcher(c *client.Client) {
	logger := monitoring.NewLogger(monitoring.LogLevelError, "text")

	// 创建命令路由器
	cmdRouter := router.NewRouter(logger)
	handlers.RegisterBuiltinHandlers(cmdRouter, &handlers.Config{Version: "test"})

	// 创建 Dispatcher
	dispConfig := &dispatcher.DispatcherConfig{
		WorkerCount:    5,
		TaskQueueSize:  100,
		HandlerTimeout: 30 * time.Second,
		Logger:         logger,
	}
	disp := dispatcher.NewDispatcher(dispConfig)

	// 创建命令处理器
	commandHandler := command.NewCommandHandler(c, cmdRouter, logger)

	// 注册处理器
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_COMMAND,
		dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
			return commandHandler.HandleCommand(ctx, msg)
		}))

	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_EVENT,
		dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
			return nil, nil
		}))

	disp.Start()
	c.SetDispatcher(disp)
}

func setupClientDispatcherWithCounter(c *client.Client, counter *int32) {
	logger := monitoring.NewLogger(monitoring.LogLevelError, "text")

	dispConfig := &dispatcher.DispatcherConfig{
		WorkerCount:    5,
		TaskQueueSize:  100,
		HandlerTimeout: 30 * time.Second,
		Logger:         logger,
	}
	disp := dispatcher.NewDispatcher(dispConfig)

	// 注册事件处理器，计数接收到的消息
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_EVENT,
		dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
			atomic.AddInt32(counter, 1)
			return nil, nil
		}))

	disp.Start()
	c.SetDispatcher(disp)
}
