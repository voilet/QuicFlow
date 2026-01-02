package alert

import (
	"context"
	"fmt"

	"github.com/voilet/quic-flow/pkg/process"
)

// ProcessAlertConfig 进程告警配置
type ProcessAlertConfig struct {
	ClientID        string
	ProjectID       string
	CPUThreshold    float64 // CPU 使用率阈值 (%)
	MemoryThreshold float64 // 内存使用率阈值 (%)
	MemoryMBLimit   float64 // 内存使用量限制 (MB)
	AlertOnExit     bool    // 进程退出时告警
	AlertOnStart    bool    // 进程启动时告警
	AlertOnRestart  bool    // 进程重启时告警
}

// ProcessAlertHandler 进程告警处理器
type ProcessAlertHandler struct {
	manager *Manager
	config  ProcessAlertConfig
}

// NewProcessAlertHandler 创建进程告警处理器
func NewProcessAlertHandler(manager *Manager, config ProcessAlertConfig) *ProcessAlertHandler {
	return &ProcessAlertHandler{
		manager: manager,
		config:  config,
	}
}

// HandleProcessAlert 处理来自进程监控器的告警
func (h *ProcessAlertHandler) HandleProcessAlert(processAlert process.Alert) {
	ctx := context.Background()

	alert := &Alert{
		Source:    "process",
		ClientID:  h.config.ClientID,
		ProjectID: h.config.ProjectID,
		Labels:    make(map[string]string),
	}

	if processAlert.Process != nil {
		alert.Labels["process_name"] = processAlert.Process.Name
		alert.Labels["pid"] = fmt.Sprintf("%d", processAlert.Process.PID)
	}

	switch processAlert.Type {
	case process.AlertTypeProcessExit:
		if !h.config.AlertOnExit {
			return
		}
		alert.Type = "process_exit"
		alert.Level = AlertLevelCritical
		alert.Title = "进程退出"
		alert.Message = processAlert.Message

	case process.AlertTypeProcessStart:
		if !h.config.AlertOnStart {
			return
		}
		alert.Type = "process_start"
		alert.Level = AlertLevelInfo
		alert.Title = "进程启动"
		alert.Message = processAlert.Message

	case process.AlertTypeProcessRestart:
		if !h.config.AlertOnRestart {
			return
		}
		alert.Type = "process_restart"
		alert.Level = AlertLevelWarning
		alert.Title = "进程重启"
		alert.Message = processAlert.Message

	case process.AlertTypeCPUHigh:
		alert.Type = "process_cpu_high"
		alert.Level = AlertLevelWarning
		alert.Title = "进程CPU使用率过高"
		alert.Message = processAlert.Message
		alert.Value = processAlert.Value
		alert.Threshold = processAlert.Threshold

	case process.AlertTypeMemoryHigh:
		alert.Type = "process_memory_high"
		alert.Level = AlertLevelWarning
		alert.Title = "进程内存使用过高"
		alert.Message = processAlert.Message
		alert.Value = processAlert.Value
		alert.Threshold = processAlert.Threshold

	default:
		return
	}

	h.manager.Fire(ctx, alert)
}

// ProcessReportToAlerts 从进程上报数据检查告警
func (h *ProcessAlertHandler) ProcessReportToAlerts(processes []process.Info) {
	ctx := context.Background()

	for _, proc := range processes {
		// CPU 阈值检查
		if h.config.CPUThreshold > 0 && proc.CPUPercent > h.config.CPUThreshold {
			alert := &Alert{
				Type:      "process_cpu_high",
				Level:     AlertLevelWarning,
				Source:    "process",
				ClientID:  h.config.ClientID,
				ProjectID: h.config.ProjectID,
				Title:     "进程CPU使用率过高",
				Message:   fmt.Sprintf("进程 %s (PID: %d) CPU使用率 %.1f%% 超过阈值 %.1f%%", proc.Name, proc.PID, proc.CPUPercent, h.config.CPUThreshold),
				Value:     proc.CPUPercent,
				Threshold: h.config.CPUThreshold,
				Labels: map[string]string{
					"process_name": proc.Name,
					"pid":          fmt.Sprintf("%d", proc.PID),
				},
			}
			h.manager.Fire(ctx, alert)
		}

		// 内存百分比阈值检查
		if h.config.MemoryThreshold > 0 && proc.MemoryPct > h.config.MemoryThreshold {
			alert := &Alert{
				Type:      "process_memory_high",
				Level:     AlertLevelWarning,
				Source:    "process",
				ClientID:  h.config.ClientID,
				ProjectID: h.config.ProjectID,
				Title:     "进程内存使用率过高",
				Message:   fmt.Sprintf("进程 %s (PID: %d) 内存使用率 %.1f%% 超过阈值 %.1f%%", proc.Name, proc.PID, proc.MemoryPct, h.config.MemoryThreshold),
				Value:     proc.MemoryPct,
				Threshold: h.config.MemoryThreshold,
				Labels: map[string]string{
					"process_name": proc.Name,
					"pid":          fmt.Sprintf("%d", proc.PID),
				},
			}
			h.manager.Fire(ctx, alert)
		}

		// 内存MB阈值检查
		if h.config.MemoryMBLimit > 0 && proc.MemoryMB > h.config.MemoryMBLimit {
			alert := &Alert{
				Type:      "process_memory_high",
				Level:     AlertLevelWarning,
				Source:    "process",
				ClientID:  h.config.ClientID,
				ProjectID: h.config.ProjectID,
				Title:     "进程内存使用过高",
				Message:   fmt.Sprintf("进程 %s (PID: %d) 内存使用 %.1fMB 超过阈值 %.1fMB", proc.Name, proc.PID, proc.MemoryMB, h.config.MemoryMBLimit),
				Value:     proc.MemoryMB,
				Threshold: h.config.MemoryMBLimit,
				Labels: map[string]string{
					"process_name": proc.Name,
					"pid":          fmt.Sprintf("%d", proc.PID),
				},
			}
			h.manager.Fire(ctx, alert)
		}
	}
}
