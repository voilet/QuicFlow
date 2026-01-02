package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/process"
)

// ProcessCollect 采集进程信息
// 命令类型: process.collect
func ProcessCollect(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ProcessCollectParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 转换匹配规则
	var rules []process.MatchRule
	for _, r := range params.Rules {
		rules = append(rules, process.MatchRule{
			Type:    r.Type,
			Pattern: r.Pattern,
			Name:    r.Name,
		})
	}

	// 创建采集器并采集
	collector := process.NewCollector(rules)
	infos, err := collector.Collect()
	if err != nil {
		return json.Marshal(command.ProcessCollectResult{
			Success: false,
			Error:   err.Error(),
		})
	}

	// 转换为命令结果格式
	var processes []command.ProcessInfoCmd
	for _, info := range infos {
		processes = append(processes, command.ProcessInfoCmd{
			PID:        info.PID,
			Name:       info.Name,
			Cmdline:    info.Cmdline,
			StartTime:  info.StartTime.Format(time.RFC3339),
			Status:     info.Status,
			CPUPercent: info.CPUPercent,
			MemoryMB:   info.MemoryMB,
			MemoryPct:  info.MemoryPct,
			MatchedBy:  info.MatchedBy,
		})
	}

	return json.Marshal(command.ProcessCollectResult{
		Success:   true,
		Processes: processes,
	})
}

// ProcessReport 上报进程信息（客户端主动上报）
// 命令类型: process.report
func ProcessReport(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ProcessReportParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 这个处理器在客户端侧用于触发上报
	// 实际上报逻辑需要调用服务端 API
	// 此处返回成功表示参数解析正确

	return json.Marshal(command.ProcessReportResult{
		Success: true,
	})
}
