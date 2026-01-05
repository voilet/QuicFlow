package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/quic-go/quic-go"
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
	hwinfoFormat      string
	hwinfoForceRefresh bool

	// diskbench 参数
	benchDevice     string
	benchTestSize   string
	benchRuntime    int
	benchFormat     string
	benchConcurrent bool

	// SSH 参数
	sshEnabled      bool
	sshUser         string
	sshPassword     string
	sshShell        string
	sshPortForward  bool
)

// 硬件信息缓存
var (
	hwCache      *command.HardwareInfoResult
	hwCacheTime  time.Time
	hwCacheMu    sync.RWMutex
	hwCacheTTL   = 3 * time.Minute // 默认缓存3分钟
	forceRefresh bool // 强制刷新标志
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

// diskbenchCmd 磁盘基准测试子命令
var diskbenchCmd = &cobra.Command{
	Use:   "diskbench",
	Short: "执行磁盘 IO 读写测试",
	Long:  "使用 FIO 对非系统盘进行 IO 读写性能测试，包括顺序读写、随机读写等",
	Run:   runDiskbench,
}

func init() {
	// 根命令参数
	rootCmd.PersistentFlags().StringVarP(&serverAddr, "server", "s", "localhost:8474", "服务器地址")
	rootCmd.PersistentFlags().StringVarP(&clientID, "id", "i", "client-001", "客户端 ID")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", true, "跳过 TLS 证书验证（仅开发环境）")

	// SSH 参数
	rootCmd.Flags().BoolVar(&sshEnabled, "ssh", true, "启用 SSH 服务（允许服务器通过 QUIC 连接 SSH 到本机）")
	rootCmd.Flags().StringVar(&sshUser, "ssh-user", "admin", "SSH 用户名")
	rootCmd.Flags().StringVar(&sshPassword, "ssh-password", "admin123", "SSH 密码")
	rootCmd.Flags().StringVar(&sshShell, "ssh-shell", "/bin/sh", "SSH 默认 Shell")
	rootCmd.Flags().BoolVar(&sshPortForward, "ssh-port-forward", true, "允许 SSH 端口转发")

	// hwinfo 子命令参数
	hwinfoCmd.Flags().StringVarP(&hwinfoFormat, "format", "f", "json", "输出格式 (json|text)")
	hwinfoCmd.Flags().BoolVarP(&hwinfoForceRefresh, "force-refresh", "F", false, "强制刷新硬件信息（忽略缓存）")

	// diskbench 子命令参数
	diskbenchCmd.Flags().StringVarP(&benchDevice, "device", "d", "", "指定测试设备（如 nvme0n1），为空则测试所有非系统盘")
	diskbenchCmd.Flags().StringVarP(&benchTestSize, "size", "S", "1G", "测试文件大小（如 1G, 512M）")
	diskbenchCmd.Flags().IntVarP(&benchRuntime, "runtime", "t", 60, "每项测试运行时间（秒）")
	diskbenchCmd.Flags().StringVarP(&benchFormat, "format", "f", "text", "输出格式 (json|text)")
	diskbenchCmd.Flags().BoolVarP(&benchConcurrent, "concurrent", "c", false, "并发测试多块磁盘（默认顺序测试）")

	// 添加子命令
	rootCmd.AddCommand(hwinfoCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(diskbenchCmd)
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

	// SSH 集成
	var sshIntegration *SSHIntegration

	if sshEnabled {
		logger.Info("SSH service enabled", "user", sshUser, "shell", sshShell)
		var err error
		sshIntegration, err = NewSSHIntegration(&SSHConfig{
			Enabled:          true,
			User:             sshUser,
			Password:         sshPassword,
			Shell:            sshShell,
			AllowPortForward: sshPortForward,
		}, logger)
		if err != nil {
			logger.Error("Failed to create SSH integration", "error", err)
			os.Exit(1)
		}
		if err := sshIntegration.Start(); err != nil {
			logger.Error("Failed to start SSH server", "error", err)
			os.Exit(1)
		}
		logger.Info("SSH server ready (will handle streams via receiveLoop)")
	}

	// 创建客户端配置
	config := client.NewDefaultClientConfig(clientID)
	config.InsecureSkipVerify = insecure
	config.Logger = logger

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

	// 设置 SSH 处理器（如果启用）
	if sshEnabled && sshIntegration != nil {
		c.SetSSHHandler(func(stream *quic.Stream, conn *quic.Conn) error {
			return sshIntegration.HandleStream(stream, conn)
		})
		logger.Info("SSH handler attached to client")
	}

	// 连接到服务器
	if err := c.Connect(serverAddr); err != nil {
		logger.Error("Failed to connect", "error", err)
	}

	logger.Info("Client started (auto-reconnect enabled)")
	logger.Info("Ready to receive and execute commands")
	if sshEnabled {
		logger.Info("SSH service available for remote access (via receiveLoop)")
	}
	logger.Info("Press Ctrl+C to stop")

	// 连接成功后自动上报硬件信息
	go func() {
		time.Sleep(1 * time.Second) // 等待连接完全建立
		if c.IsConnected() {
			reportHardwareInfo(c, logger)
		}
	}()

	// 注意：SSH 流现在由 receiveLoop 中的 SSH handler 处理，不再需要单独的 AcceptSSHStreams

	// 定期打印状态
	go printStatus(c, cmdRouter, sshEnabled)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	shutdown(logger, disp, c, sshIntegration)
}

// runHwinfo 运行硬件信息获取（本地模式）
func runHwinfo(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 使用缓存的硬件信息（除非强制刷新）
	hwInfo, err := getCachedHardwareInfo(ctx, hwinfoForceRefresh)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取硬件信息失败: %v\n", err)
		os.Exit(1)
	}

	if hwinfoFormat == "text" {
		printHwinfoText(hwInfo)
	} else {
		// JSON 格式输出（美化）
		output, _ := json.MarshalIndent(hwInfo, "", "  ")
		fmt.Println(string(output))
	}
}

// runDiskbench 运行磁盘基准测试（本地模式）
func runDiskbench(cmd *cobra.Command, args []string) {
	fmt.Println("================== 磁盘 IO 性能测试 ==================")
	fmt.Println()
	fmt.Println("正在准备测试...")
	fmt.Printf("  测试设备: %s\n", func() string {
		if benchDevice == "" {
			return "所有非系统盘"
		}
		return benchDevice
	}())
	fmt.Printf("  测试大小: %s\n", benchTestSize)
	fmt.Printf("  运行时间: %d 秒/项\n", benchRuntime)
	fmt.Printf("  并发模式: %v\n", benchConcurrent)
	fmt.Println()

	// 执行测试
	response, err := handlers.RunLocalBenchmark(benchDevice, benchTestSize, benchRuntime, benchConcurrent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "执行测试失败: %v\n", err)
		os.Exit(1)
	}

	if !response.Success {
		fmt.Fprintf(os.Stderr, "测试失败: %s\n", response.Message)
		os.Exit(1)
	}

	if benchFormat == "json" {
		// JSON 格式输出
		output, _ := json.MarshalIndent(response, "", "  ")
		fmt.Println(string(output))
	} else {
		// 文本格式输出
		fmt.Printf("测试完成时间: %s\n", response.TestedAt)
		fmt.Printf("测试磁盘数量: %d\n", response.TotalDisks)
		fmt.Println()

		for i, result := range response.Results {
			if i > 0 {
				fmt.Println("--------------------------------------------------")
			}
			fmt.Println(handlers.FormatBenchmarkResult(result))
		}

		fmt.Println("====================================================")
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
func printStatus(c *client.Client, cmdRouter interface{ ListCommands() []string }, sshEnabled bool) {
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
		if sshEnabled {
			fmt.Printf("SSH Service: enabled\n")
		}
		fmt.Println()
	}
}

// shutdown 优雅关闭
func shutdown(logger *monitoring.Logger, disp *dispatcher.Dispatcher, c *client.Client, sshIntegration *SSHIntegration) {
	logger.Info("Shutting down client...")

	// 停止 SSH 服务
	if sshIntegration != nil {
		if err := sshIntegration.Stop(); err != nil {
			logger.Error("Error stopping SSH server", "error", err)
		} else {
			logger.Info("SSH server stopped")
		}
	}

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

// getCachedHardwareInfo 获取缓存的硬件信息
func getCachedHardwareInfo(ctx context.Context, force bool) (*command.HardwareInfoResult, error) {
	hwCacheMu.RLock()
	// 检查缓存是否存在且未过期
	cached := hwCache != nil && time.Since(hwCacheTime) < hwCacheTTL
	if cached && !force {
		// 返回缓存副本
		result := *hwCache
		hwCacheMu.RUnlock()
		return &result, nil
	}
	hwCacheMu.RUnlock()

	// 采集新的硬件信息
	result, err := handlers.GetHardwareInfo(ctx, nil)
	if err != nil {
		return nil, err
	}

	var hwInfo command.HardwareInfoResult
	if err := json.Unmarshal(result, &hwInfo); err != nil {
		return nil, err
	}

	// 更新缓存
	hwCacheMu.Lock()
	hwCache = &hwInfo
	hwCacheTime = time.Now()
	hwCacheMu.Unlock()

	return &hwInfo, nil
}

// reportHardwareInfo 上报硬件信息到服务器
func reportHardwareInfo(c *client.Client, logger *monitoring.Logger) {
	// 等待一小段时间确保连接完全建立
	time.Sleep(500 * time.Millisecond)

	ctx := context.Background()

	// 获取缓存的硬件信息（首次采集会缓存，后续使用缓存）
	hwInfo, err := getCachedHardwareInfo(ctx, forceRefresh)
	if err != nil {
		logger.Warn("Failed to get hardware info for reporting", "error", err)
		return
	}

	// 构建上报消息（使用 report.hardware 命令格式）
	reportMsg := map[string]interface{}{
		"client_id":      c.GetClientID(),
		"command_type":   "hardware.info",
		"hardware_info":  hwInfo, // 使用解析后的对象，而不是 []byte
		"report_type":    "auto_sync", // 标识为自动同步
	}

	// 构建命令载荷（使用 router 的命令格式）
	cmdPayload := map[string]interface{}{
		"command_type": "report.hardware",
		"payload":      reportMsg,
	}
	payloadBytes, _ := json.Marshal(cmdPayload)

	msg := &protocol.DataMessage{
		MsgId:      generateMsgID(),
		SenderId:   c.GetClientID(),
		ReceiverId: "server",
		Type:       protocol.MessageType_MESSAGE_TYPE_EVENT,
		Payload:    payloadBytes,
		Timestamp:  time.Now().UnixMilli(),
	}

	// 发送消息
	if err := c.SendMessageAsync(msg); err != nil {
		logger.Warn("Failed to report hardware info", "error", err)
	} else {
		logger.Info("Hardware info reported to server")
	}
}

// generateMsgID 生成消息ID
func generateMsgID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
