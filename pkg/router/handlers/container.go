package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/container"
)

// ContainerCollect 采集容器信息
// 命令类型: container.collect
func ContainerCollect(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ContainerCollectParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 创建采集器
	collector := container.NewCollector("")

	// 检查 Docker 是否可用
	if !collector.IsDockerAvailable() {
		return json.Marshal(command.ContainerCollectResult{
			Success: false,
			Error:   "Docker is not available",
		})
	}

	// 采集容器信息
	infos, err := collector.Collect(params.All, params.Prefixes)
	if err != nil {
		return json.Marshal(command.ContainerCollectResult{
			Success: false,
			Error:   err.Error(),
		})
	}

	// 获取 Docker 版本
	dockerVersion, _ := collector.GetDockerVersion()

	// 统计
	summary := collector.GetSummary(infos)

	// 转换为命令结果格式
	var containers []command.ContainerInfoCmd
	for _, info := range infos {
		containers = append(containers, command.ContainerInfoCmd{
			ContainerID:   info.ContainerID,
			ContainerName: info.ContainerName,
			Image:         info.Image,
			Status:        info.Status,
			State:         info.State,
			CreatedAt:     info.CreatedAt.Format(time.RFC3339),
			StartedAt:     info.StartedAt.Format(time.RFC3339),
			CPUPercent:    info.CPUPercent,
			MemoryUsage:   info.MemoryUsage,
			MemoryLimit:   info.MemoryLimit,
			MemoryPercent: info.MemoryPercent,
			NetworkRx:     info.NetworkRx,
			NetworkTx:     info.NetworkTx,
			MatchedPrefix: info.MatchedPrefix,
			MatchedProject: info.MatchedProject,
		})
	}

	return json.Marshal(command.ContainerCollectResult{
		Success:       true,
		Containers:    containers,
		DockerVersion: dockerVersion,
		TotalCount:    summary.TotalCount,
		RunningCount:  summary.RunningCount,
	})
}

// ContainerReport 上报容器信息
// 命令类型: container.report
func ContainerReport(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ContainerReportParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 这个处理器在客户端侧用于触发上报
	// 实际上报逻辑需要调用服务端 API

	return json.Marshal(command.ContainerReportResult{
		Success: true,
	})
}

// ContainerList 列出容器
// 命令类型: container.list
func ContainerList(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ContainerListParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 创建采集器并采集
	collector := container.NewCollector("")
	infos, err := collector.Collect(params.All, params.Prefixes)
	if err != nil {
		return json.Marshal(command.ContainerListResult{
			Success: false,
			Error:   err.Error(),
		})
	}

	// 转换为命令结果格式
	var containers []command.ContainerInfoCmd
	for _, info := range infos {
		containers = append(containers, command.ContainerInfoCmd{
			ContainerID:   info.ContainerID,
			ContainerName: info.ContainerName,
			Image:         info.Image,
			Status:        info.Status,
			State:         info.State,
			CreatedAt:     info.CreatedAt.Format(time.RFC3339),
			StartedAt:     info.StartedAt.Format(time.RFC3339),
			CPUPercent:    info.CPUPercent,
			MemoryUsage:   info.MemoryUsage,
			MemoryLimit:   info.MemoryLimit,
			MemoryPercent: info.MemoryPercent,
			NetworkRx:     info.NetworkRx,
			NetworkTx:     info.NetworkTx,
			MatchedPrefix: info.MatchedPrefix,
			MatchedProject: info.MatchedProject,
		})
	}

	return json.Marshal(command.ContainerListResult{
		Success:    true,
		Containers: containers,
	})
}
