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
	// 全局参数
	serverAddr string
	clientID   string
	insecure   bool

	// hwinfo 参数
	hwinfoFormat string
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "quic-client",
	Short: "QUIC Backbone Client",
	Long:  "QUIC Backbone Client - 高性能 QUIC 协议客户端",
	Run:   runClient,
}

// hwinfoCmd 硬件信息子命令
var hwinfoCmd = &cobra.Command{
	Use:   "hwinfo",
	Short: "获取本机硬件信息",
	Long:  "获取本机硬件信息，包括 CPU、内存、磁盘、网卡、DMI 等",
	Run:   runHwinfo,
}

// versionCmd 版本信息子命令
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Long:  "显示客户端版本号、Git 提交、编译时间等信息",
	Run: func(cmd *cobra.Command, args []string) {
		version.Print("quic-client")
	},
}

func init() {
	// 根命令参数
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "server", "s", "localhost:8474", "服务器地址")
	rootCmd.PersistentFlags().StringVarP(&clientID, "id", "i", "client-001", "客户端 ID")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", true, "跳过 TLS 证书验证（仅开发环境）")

	// hwinfo 子命令参数
	hwinfoCmd.Flags().StringVarP(&hwinfoFormat, "format", "f", "json", "输出格式 (json|text)")

	// 添加子命令
	rootCmd.AddCommand(hwinfoCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runClient 运行客户端（连接服务器模式）
func runClient(cmd *cobra.Command, args []string) {
	// 创建日志器
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")

	logger.Info("=== QUIC Backbone Client ===")
	logger.Info("Version", "version", version.String())
	logger.Info("Connecting to server", "server", serverAddr, "client_id", clientID)

	// 创建客户端配置
	config := client.NewDefaultClientConfig(clientID)
	config.InsecureSkipVerify = insecure
	config.Logger = logger

	// 设置事件钩子
	config.Hooks = &monitoring.EventHooks{
		OnConnect: func(clientID string) {
			logger.Info("Connected to server", "client_id", clientID)
		},
		OnDisconnect: func(clientID string, reason error) {
			logger.Warn("Disconnected from server", "client_id", clientID, "reason", reason)
		},
		OnReconnect: func(clientID string, attemptCount int) {
			logger.Info("Reconnected to server", "client_id", clientID, "attempts", attemptCount)
		},
	}

	// 创建客户端
	c, err := client.NewClient(config)
	if err != nil {
		logger.Error("Failed to create client", "error", err)
		os.Exit(1)
	}

	// 设置命令路由器
	cmdRouter := SetupClientRouter(logger)

	// 创建 Dispatcher 并注册消息处理器
	disp := setupDispatcher(logger, c, cmdRouter)

	// 设置 Dispatcher 到客户端
	c.SetDispatcher(disp)
	logger.Info("Dispatcher attached to client")

	// 连接到服务器
	if err := c.Connect(serverAddr); err != nil {
		logger.Error("Failed to connect", "error", err)
	}

	logger.Info("Client started (auto-reconnect enabled)")
	logger.Info("Ready to receive and execute commands")
	logger.Info("Press Ctrl+C to stop")

	// 定期打印状态
	go printStatus(c, cmdRouter)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	shutdown(logger, disp, c)
}

// runHwinfo 运行硬件信息获取（本地模式）
func runHwinfo(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 调用硬件信息获取函数
	result, err := handlers.GetHardwareInfo(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取硬件信息失败: %v\n", err)
		os.Exit(1)
	}

	if hwinfoFormat == "text" {
		// 文本格式输出
		var hwInfo command.HardwareInfoResult
		if err := json.Unmarshal(result, &hwInfo); err != nil {
			fmt.Fprintf(os.Stderr, "解析硬件信息失败: %v\n", err)
			os.Exit(1)
		}
		printHwinfoText(&hwInfo)
	} else {
		// JSON 格式输出（美化）
		var prettyJSON map[string]interface{}
		json.Unmarshal(result, &prettyJSON)
		output, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Println(string(output))
	}
}

// printHwinfoText 以文本格式打印硬件信息
func printHwinfoText(info *command.HardwareInfoResult) {
	fmt.Println("================== 硬件信息 ==================")
	fmt.Println()

	// 主机信息
	fmt.Println("【主机信息】")
	fmt.Printf("  主机名:     %s\n", info.Host.Hostname)
	fmt.Printf("  操作系统:   %s\n", info.Host.OS)
	fmt.Printf("  平台:       %s %s\n", info.Host.Platform, info.Host.PlatformVersion)
	fmt.Printf("  内核版本:   %s\n", info.Host.KernelVersion)
	fmt.Printf("  架构:       %s\n", info.Host.KernelArch)
	fmt.Printf("  运行时间:   %s\n", formatUptime(info.Host.Uptime))
	fmt.Printf("  主机ID:     %s\n", info.Host.HostID)
	if info.Host.VirtualizationSystem != "" {
		fmt.Printf("  虚拟化:     %s (%s)\n", info.Host.VirtualizationSystem, info.Host.VirtualizationRole)
	}
	fmt.Println()

	// CPU 信息
	fmt.Println("【CPU 信息】")
	fmt.Printf("  型号:       %s\n", info.ModelName)
	fmt.Printf("  物理核心:   %d\n", info.CPUCoreCount)
	fmt.Printf("  逻辑处理器: %d\n", info.CPUThreadCount)
	if info.PhysicalCPUFrequencyMHz > 0 {
		fmt.Printf("  频率:       %.0f MHz\n", info.PhysicalCPUFrequencyMHz)
	}
	fmt.Println()

	// 内存信息
	fmt.Println("【内存信息】")
	fmt.Printf("  总容量:     %d GB\n", info.Memory.TotalGBRounded)
	if info.Memory.Count > 0 {
		fmt.Printf("  内存条数:   %d\n", info.Memory.Count)
	}
	fmt.Println()

	// 磁盘信息
	fmt.Println("【磁盘信息】")
	fmt.Printf("  总容量:     %.2f TB\n", info.TotalDiskCapacityTB)
	for _, disk := range info.Disks {
		sizeStr := fmt.Sprintf("%.0f GB", float64(disk.SizeRoundedBytes)/1024/1024/1024)
		if disk.SizeRoundedTB >= 1 {
			sizeStr = fmt.Sprintf("%.2f TB", disk.SizeRoundedTB)
		}
		sysFlag := ""
		if disk.IsSystemDisk {
			sysFlag = " [系统盘]"
		}
		fmt.Printf("  - %s: %s %s (%s)%s\n", disk.Device, disk.Model, sizeStr, disk.Kind, sysFlag)
	}
	fmt.Println()

	// 网卡信息
	fmt.Println("【网卡信息】")
	fmt.Printf("  主MAC:      %s\n", info.MAC)
	for _, nic := range info.NICInfos {
		fmt.Printf("  - %s: %s (IP: %s, 状态: %s)\n", nic.Name, nic.MACAddress, nic.IPAddress, nic.Status)
	}
	fmt.Println()

	// DMI 信息
	fmt.Println("【DMI/BIOS 信息】")
	fmt.Printf("  系统厂商:   %s\n", info.DMI.SysVendor)
	fmt.Printf("  产品名称:   %s\n", info.DMI.ProductName)
	fmt.Printf("  产品UUID:   %s\n", info.DMI.ProductUUID)
	fmt.Printf("  BIOS厂商:   %s\n", info.DMI.BiosVendor)
	fmt.Printf("  BIOS版本:   %s\n", info.DMI.BiosVersion)
	fmt.Println()
	fmt.Println("================================================")
}

// formatUptime 格式化运行时间
func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

// setupDispatcher 设置消息分发器
func setupDispatcher(logger *monitoring.Logger, c *client.Client, cmdRouter *router.Router) *dispatcher.Dispatcher {
	dispatcherConfig := &dispatcher.DispatcherConfig{
		WorkerCount:    10,
		TaskQueueSize:  1000,
		HandlerTimeout: 30 * time.Second,
		Logger:         logger,
	}
	disp := dispatcher.NewDispatcher(dispatcherConfig)

	// 创建命令处理器
	commandHandler := command.NewCommandHandler(c, cmdRouter, logger)

	// 注册 MESSAGE_TYPE_COMMAND 处理器
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_COMMAND, dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
		return commandHandler.HandleCommand(ctx, msg)
	}))

	// 处理 Server 推送的事件
	disp.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_EVENT, dispatcher.MessageHandlerFunc(func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
		logger.Info("Received event from server", "msg_id", msg.MsgId)
		return nil, nil
	}))

	// 启动 Dispatcher
	disp.Start()
	logger.Info("Dispatcher started with command handler")

	return disp
}

// printStatus 定期打印状态
func printStatus(c *client.Client, cmdRouter interface{ ListCommands() []string }) {
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
		fmt.Printf("Registered Commands: %v\n", cmdRouter.ListCommands())
		fmt.Println()
	}
}

// shutdown 优雅关闭
func shutdown(logger *monitoring.Logger, disp *dispatcher.Dispatcher, c *client.Client) {
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
