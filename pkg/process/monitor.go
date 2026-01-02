package process

import (
	"context"
	"sync"
	"time"
)

// MonitorConfig 监控配置
type MonitorConfig struct {
	Interval       time.Duration // 采集间隔
	Rules          []MatchRule   // 匹配规则
	ReportCallback func([]Info)  // 上报回调
	AlertCallback  func(Alert)   // 告警回调
}

// Alert 告警信息
type Alert struct {
	Type      AlertType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Process   *Info     `json:"process,omitempty"`
	Message   string    `json:"message"`
	Threshold float64   `json:"threshold,omitempty"`
	Value     float64   `json:"value,omitempty"`
}

// AlertType 告警类型
type AlertType string

const (
	AlertTypeProcessExit    AlertType = "process_exit"
	AlertTypeProcessStart   AlertType = "process_start"
	AlertTypeCPUHigh        AlertType = "cpu_high"
	AlertTypeMemoryHigh     AlertType = "memory_high"
	AlertTypeProcessRestart AlertType = "process_restart"
)

// AlertThresholds 告警阈值配置
type AlertThresholds struct {
	CPUPercent    float64 // CPU 使用率阈值 (%)
	MemoryPercent float64 // 内存使用率阈值 (%)
	MemoryMB      float64 // 内存使用量阈值 (MB)
}

// Monitor 进程监控器
type Monitor struct {
	config     MonitorConfig
	thresholds AlertThresholds
	collector  *Collector

	// 上次采集的进程状态
	lastProcesses map[int]Info
	mu            sync.RWMutex

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMonitor 创建进程监控器
func NewMonitor(config MonitorConfig) *Monitor {
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}

	return &Monitor{
		config:        config,
		collector:     NewCollector(config.Rules),
		lastProcesses: make(map[int]Info),
		thresholds: AlertThresholds{
			CPUPercent:    80,
			MemoryPercent: 80,
			MemoryMB:      0, // 不限制
		},
	}
}

// SetThresholds 设置告警阈值
func (m *Monitor) SetThresholds(thresholds AlertThresholds) {
	m.thresholds = thresholds
}

// Start 启动监控
func (m *Monitor) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	m.wg.Add(1)
	go m.monitorLoop()

	return nil
}

// Stop 停止监控
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}

// monitorLoop 监控循环
func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	// 首次采集
	m.collect()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collect()
		}
	}
}

// collect 执行一次采集
func (m *Monitor) collect() {
	processes, err := m.collector.Collect()
	if err != nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 构建当前进程映射
	currentProcesses := make(map[int]Info)
	for _, p := range processes {
		currentProcesses[p.PID] = p
	}

	// 检测进程退出
	for pid, lastInfo := range m.lastProcesses {
		if _, exists := currentProcesses[pid]; !exists {
			// 进程退出
			m.sendAlert(Alert{
				Type:      AlertTypeProcessExit,
				Timestamp: time.Now(),
				Process:   &lastInfo,
				Message:   "Process exited: " + lastInfo.Name,
			})
		}
	}

	// 检测新进程启动
	for pid, info := range currentProcesses {
		if _, exists := m.lastProcesses[pid]; !exists {
			// 新进程启动
			m.sendAlert(Alert{
				Type:      AlertTypeProcessStart,
				Timestamp: time.Now(),
				Process:   &info,
				Message:   "Process started: " + info.Name,
			})
		}
	}

	// 检测进程重启（相同名称但不同PID）
	m.detectRestarts(currentProcesses)

	// 检查资源阈值
	for _, info := range processes {
		m.checkThresholds(info)
	}

	// 更新状态
	m.lastProcesses = currentProcesses

	// 上报回调
	if m.config.ReportCallback != nil && len(processes) > 0 {
		m.config.ReportCallback(processes)
	}
}

// detectRestarts 检测进程重启
func (m *Monitor) detectRestarts(current map[int]Info) {
	// 按名称分组
	lastByName := make(map[string][]Info)
	for _, info := range m.lastProcesses {
		lastByName[info.Name] = append(lastByName[info.Name], info)
	}

	currentByName := make(map[string][]Info)
	for _, info := range current {
		currentByName[info.Name] = append(currentByName[info.Name], info)
	}

	// 检测重启：上次存在，当前也存在，但PID不同
	for name, lastInfos := range lastByName {
		currentInfos, exists := currentByName[name]
		if !exists {
			continue
		}

		// 检查是否有PID变化
		lastPIDs := make(map[int]bool)
		for _, info := range lastInfos {
			lastPIDs[info.PID] = true
		}

		for _, info := range currentInfos {
			if !lastPIDs[info.PID] {
				// 新的PID，可能是重启
				m.sendAlert(Alert{
					Type:      AlertTypeProcessRestart,
					Timestamp: time.Now(),
					Process:   &info,
					Message:   "Process restarted: " + name,
				})
			}
		}
	}
}

// checkThresholds 检查资源阈值
func (m *Monitor) checkThresholds(info Info) {
	// CPU 阈值
	if m.thresholds.CPUPercent > 0 && info.CPUPercent > m.thresholds.CPUPercent {
		m.sendAlert(Alert{
			Type:      AlertTypeCPUHigh,
			Timestamp: time.Now(),
			Process:   &info,
			Message:   "CPU usage high: " + info.Name,
			Threshold: m.thresholds.CPUPercent,
			Value:     info.CPUPercent,
		})
	}

	// 内存百分比阈值
	if m.thresholds.MemoryPercent > 0 && info.MemoryPct > m.thresholds.MemoryPercent {
		m.sendAlert(Alert{
			Type:      AlertTypeMemoryHigh,
			Timestamp: time.Now(),
			Process:   &info,
			Message:   "Memory usage high: " + info.Name,
			Threshold: m.thresholds.MemoryPercent,
			Value:     info.MemoryPct,
		})
	}

	// 内存MB阈值
	if m.thresholds.MemoryMB > 0 && info.MemoryMB > m.thresholds.MemoryMB {
		m.sendAlert(Alert{
			Type:      AlertTypeMemoryHigh,
			Timestamp: time.Now(),
			Process:   &info,
			Message:   "Memory usage high: " + info.Name,
			Threshold: m.thresholds.MemoryMB,
			Value:     info.MemoryMB,
		})
	}
}

// sendAlert 发送告警
func (m *Monitor) sendAlert(alert Alert) {
	if m.config.AlertCallback != nil {
		m.config.AlertCallback(alert)
	}
}

// GetLastProcesses 获取上次采集的进程列表
func (m *Monitor) GetLastProcesses() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Info, 0, len(m.lastProcesses))
	for _, info := range m.lastProcesses {
		result = append(result, info)
	}
	return result
}

// IsProcessRunning 检查进程是否运行
func (m *Monitor) IsProcessRunning(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, info := range m.lastProcesses {
		if info.Name == name {
			return true
		}
	}
	return false
}

// GetProcessByName 按名称获取进程信息
func (m *Monitor) GetProcessByName(name string) []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Info
	for _, info := range m.lastProcesses {
		if info.Name == name {
			result = append(result, info)
		}
	}
	return result
}

// CollectOnce 立即执行一次采集
func (m *Monitor) CollectOnce() ([]Info, error) {
	return m.collector.Collect()
}
