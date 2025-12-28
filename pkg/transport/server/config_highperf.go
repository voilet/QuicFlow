package server

import (
	"runtime"
	"time"

	"github.com/voilet/quic-flow/pkg/monitoring"
)

// HighPerformanceConfig 高性能配置预设
// 目标：支持 10W 客户端连接，5W 并发任务
type HighPerformanceConfig struct {
	// 目标客户端数量
	TargetClients int64

	// 目标并发任务数量
	TargetConcurrentTasks int64

	// 可选的自定义配置
	CustomLogger *monitoring.Logger
	CustomHooks  *monitoring.EventHooks
}

// NewHighPerformanceServerConfig 创建高性能服务器配置
// 针对 10W 连接 + 5W 并发任务优化
func NewHighPerformanceServerConfig(certFile, keyFile, listenAddr string) *ServerConfig {
	return NewHighPerformanceServerConfigWithTarget(certFile, keyFile, listenAddr, &HighPerformanceConfig{
		TargetClients:         100000,
		TargetConcurrentTasks: 50000,
	})
}

// NewHighPerformanceServerConfigWithTarget 创建指定目标的高性能配置
func NewHighPerformanceServerConfigWithTarget(certFile, keyFile, listenAddr string, perf *HighPerformanceConfig) *ServerConfig {
	if perf == nil {
		perf = &HighPerformanceConfig{
			TargetClients:         100000,
			TargetConcurrentTasks: 50000,
		}
	}

	// 计算配置值
	// MaxClients = 目标客户端数 * 1.5 (安全边界)
	maxClients := int64(float64(perf.TargetClients) * 1.5)

	// MaxIncomingStreams = 每客户端并发流数 (至少 100)
	maxStreams := int64(100)
	if perf.TargetConcurrentTasks > 0 {
		// 假设任务分布在所有客户端上
		streamsPerClient := perf.TargetConcurrentTasks / perf.TargetClients
		if streamsPerClient < 100 {
			streamsPerClient = 100
		}
		maxStreams = streamsPerClient * 10 // 10 倍余量
	}

	// MaxPromises = 目标并发任务数 * 3 (支持重试和堆积)
	maxPromises := perf.TargetConcurrentTasks * 3
	if maxPromises < 100000 {
		maxPromises = 100000
	}

	// 缓冲区大小（大连接数需要更大缓冲区）
	// 初始流窗口: 1MB
	// 最大流窗口: 16MB
	// 初始连接窗口: 2MB
	// 最大连接窗口: 32MB
	initialStreamWindow := uint64(1 * 1024 * 1024)      // 1MB
	maxStreamWindow := uint64(16 * 1024 * 1024)         // 16MB
	initialConnWindow := uint64(2 * 1024 * 1024)        // 2MB
	maxConnWindow := uint64(32 * 1024 * 1024)           // 32MB

	logger := perf.CustomLogger
	if logger == nil {
		logger = monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	}

	return &ServerConfig{
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
		ListenAddr:  listenAddr,

		// QUIC 高性能配置
		MaxIdleTimeout:                 120 * time.Second,  // 增加空闲超时
		MaxIncomingStreams:             maxStreams,
		MaxIncomingUniStreams:          1000,
		InitialStreamReceiveWindow:     initialStreamWindow,
		MaxStreamReceiveWindow:         maxStreamWindow,
		InitialConnectionReceiveWindow: initialConnWindow,
		MaxConnectionReceiveWindow:     maxConnWindow,

		// 会话管理配置
		MaxClients:             maxClients,
		HeartbeatInterval:      30 * time.Second,  // 增加心跳间隔减少开销
		HeartbeatTimeout:       90 * time.Second,  // 相应增加超时
		HeartbeatCheckInterval: 10 * time.Second,  // 增加检查间隔
		MaxTimeoutCount:        3,

		// Promise 配置
		MaxPromises:           maxPromises,
		PromiseWarnThreshold:  int64(float64(maxPromises) * 0.8),
		DefaultMessageTimeout: 60 * time.Second, // 增加消息超时

		// 监控
		Hooks:  perf.CustomHooks,
		Logger: logger,
	}
}

// CalculateOptimalWorkerCount 计算最佳 Worker 数量
// 基于 CPU 核心数和目标并发任务数
func CalculateOptimalWorkerCount(targetConcurrentTasks int64) int {
	cpuCount := runtime.NumCPU()

	// 基础值：每核心 4 个 worker
	baseWorkers := cpuCount * 4

	// 根据目标任务数调整
	// 每 1000 任务增加 1 个 worker，最多增加到基础值的 10 倍
	taskBasedWorkers := int(targetConcurrentTasks / 1000)
	if taskBasedWorkers > baseWorkers*10 {
		taskBasedWorkers = baseWorkers * 10
	}

	// 取较大值，但不超过 1000
	workers := baseWorkers + taskBasedWorkers
	if workers > 1000 {
		workers = 1000
	}
	if workers < 20 {
		workers = 20
	}

	return workers
}

// CalculateOptimalQueueSize 计算最佳队列大小
func CalculateOptimalQueueSize(targetConcurrentTasks int64) int {
	// 队列大小 = 目标并发任务数 * 2
	queueSize := int(targetConcurrentTasks * 2)
	if queueSize < 10000 {
		queueSize = 10000
	}
	if queueSize > 500000 {
		queueSize = 500000
	}
	return queueSize
}

// GetHighPerfDispatcherConfig 获取高性能 Dispatcher 配置
func GetHighPerfDispatcherConfig(targetTasks int64, logger *monitoring.Logger) (workerCount int, queueSize int, timeout time.Duration) {
	workerCount = CalculateOptimalWorkerCount(targetTasks)
	queueSize = CalculateOptimalQueueSize(targetTasks)
	timeout = 60 * time.Second
	return
}
