package alert

import (
	"context"
	"fmt"

	"github.com/voilet/quic-flow/pkg/container"
)

// ContainerAlertConfig 容器告警配置
type ContainerAlertConfig struct {
	ClientID           string
	ProjectID          string
	CPUThreshold       float64  // CPU 使用率阈值 (%)
	MemoryThreshold    float64  // 内存使用率阈值 (%)
	AlertOnStop        bool     // 容器停止时告警
	AlertOnRestart     bool     // 容器重启时告警
	AlertOnUnhealthy   bool     // 容器不健康时告警
	MonitoredPrefixes  []string // 监控的容器名称前缀
}

// ContainerAlertHandler 容器告警处理器
type ContainerAlertHandler struct {
	manager        *Manager
	config         ContainerAlertConfig
	lastContainers map[string]container.Info // containerID -> Info
}

// NewContainerAlertHandler 创建容器告警处理器
func NewContainerAlertHandler(manager *Manager, config ContainerAlertConfig) *ContainerAlertHandler {
	return &ContainerAlertHandler{
		manager:        manager,
		config:         config,
		lastContainers: make(map[string]container.Info),
	}
}

// CheckContainers 检查容器状态并生成告警
func (h *ContainerAlertHandler) CheckContainers(containers []container.Info) {
	ctx := context.Background()
	currentContainers := make(map[string]container.Info)

	for _, c := range containers {
		currentContainers[c.ContainerID] = c

		// 检查容器状态
		h.checkContainerState(ctx, c)

		// 检查资源使用
		h.checkContainerResources(ctx, c)
	}

	// 检查容器停止
	if h.config.AlertOnStop {
		h.checkContainerStop(ctx, currentContainers)
	}

	// 更新状态
	h.lastContainers = currentContainers
}

// checkContainerState 检查容器状态
func (h *ContainerAlertHandler) checkContainerState(ctx context.Context, c container.Info) {
	labels := map[string]string{
		"container_id":   c.ContainerID[:12],
		"container_name": c.ContainerName,
		"image":          c.Image,
	}

	// 检查不健康状态
	if h.config.AlertOnUnhealthy && c.State != "running" {
		level := AlertLevelWarning
		if c.State == "dead" || c.State == "exited" {
			level = AlertLevelCritical
		}

		alert := &Alert{
			Type:      "container_unhealthy",
			Level:     level,
			Source:    "container",
			ClientID:  h.config.ClientID,
			ProjectID: h.config.ProjectID,
			Title:     "容器状态异常",
			Message:   fmt.Sprintf("容器 %s 状态: %s", c.ContainerName, c.State),
			Labels:    labels,
		}
		h.manager.Fire(ctx, alert)
	}

	// 检查重启
	if h.config.AlertOnRestart {
		if last, ok := h.lastContainers[c.ContainerID]; ok {
			// 简单通过启动时间判断是否重启
			if !last.StartedAt.IsZero() && !c.StartedAt.IsZero() && c.StartedAt.After(last.StartedAt) {
				alert := &Alert{
					Type:      "container_restart",
					Level:     AlertLevelWarning,
					Source:    "container",
					ClientID:  h.config.ClientID,
					ProjectID: h.config.ProjectID,
					Title:     "容器重启",
					Message:   fmt.Sprintf("容器 %s 已重启", c.ContainerName),
					Labels:    labels,
				}
				h.manager.Fire(ctx, alert)
			}
		}
	}
}

// checkContainerResources 检查容器资源使用
func (h *ContainerAlertHandler) checkContainerResources(ctx context.Context, c container.Info) {
	labels := map[string]string{
		"container_id":   c.ContainerID[:12],
		"container_name": c.ContainerName,
		"image":          c.Image,
	}

	// CPU 阈值检查
	if h.config.CPUThreshold > 0 && c.CPUPercent > h.config.CPUThreshold {
		alert := &Alert{
			Type:      "container_cpu_high",
			Level:     AlertLevelWarning,
			Source:    "container",
			ClientID:  h.config.ClientID,
			ProjectID: h.config.ProjectID,
			Title:     "容器CPU使用率过高",
			Message:   fmt.Sprintf("容器 %s CPU使用率 %.1f%% 超过阈值 %.1f%%", c.ContainerName, c.CPUPercent, h.config.CPUThreshold),
			Value:     c.CPUPercent,
			Threshold: h.config.CPUThreshold,
			Labels:    labels,
		}
		h.manager.Fire(ctx, alert)
	}

	// 内存阈值检查
	if h.config.MemoryThreshold > 0 && c.MemoryPercent > h.config.MemoryThreshold {
		alert := &Alert{
			Type:      "container_memory_high",
			Level:     AlertLevelWarning,
			Source:    "container",
			ClientID:  h.config.ClientID,
			ProjectID: h.config.ProjectID,
			Title:     "容器内存使用率过高",
			Message:   fmt.Sprintf("容器 %s 内存使用率 %.1f%% 超过阈值 %.1f%%", c.ContainerName, c.MemoryPercent, h.config.MemoryThreshold),
			Value:     c.MemoryPercent,
			Threshold: h.config.MemoryThreshold,
			Labels:    labels,
		}
		h.manager.Fire(ctx, alert)
	}
}

// checkContainerStop 检查容器停止
func (h *ContainerAlertHandler) checkContainerStop(ctx context.Context, current map[string]container.Info) {
	for id, last := range h.lastContainers {
		if _, exists := current[id]; !exists {
			// 容器不存在了，可能被删除或停止
			alert := &Alert{
				Type:      "container_stopped",
				Level:     AlertLevelCritical,
				Source:    "container",
				ClientID:  h.config.ClientID,
				ProjectID: h.config.ProjectID,
				Title:     "容器停止",
				Message:   fmt.Sprintf("容器 %s 已停止或被删除", last.ContainerName),
				Labels: map[string]string{
					"container_id":   last.ContainerID[:12],
					"container_name": last.ContainerName,
					"image":          last.Image,
				},
			}
			h.manager.Fire(ctx, alert)
		}
	}
}

// ContainerMonitor 容器监控器
type ContainerMonitor struct {
	collector *container.Collector
	handler   *ContainerAlertHandler
}

// NewContainerMonitor 创建容器监控器
func NewContainerMonitor(handler *ContainerAlertHandler, prefixes []string) *ContainerMonitor {
	return &ContainerMonitor{
		collector: container.NewCollector(""),
		handler:   handler,
	}
}

// Check 执行一次检查
func (m *ContainerMonitor) Check() error {
	if !m.collector.IsDockerAvailable() {
		return fmt.Errorf("docker not available")
	}

	containers, err := m.collector.Collect(true, m.handler.config.MonitoredPrefixes)
	if err != nil {
		return err
	}

	m.handler.CheckContainers(containers)
	return nil
}
