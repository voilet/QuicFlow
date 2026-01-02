package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/k8s"
)

// K8sCollect 采集 K8s Pod 信息
// 命令类型: k8s.collect
func K8sCollect(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.K8sCollectParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 创建采集器
	config := k8s.CollectorConfig{
		APIServer:   params.APIServer,
		Token:       params.Token,
		TokenFile:   params.TokenFile,
		Namespace:   params.Namespace,
		InCluster:   params.InCluster,
		InsecureTLS: params.InsecureTLS,
	}

	collector, err := k8s.NewCollector(config)
	if err != nil {
		return json.Marshal(command.K8sCollectResult{
			Success: false,
			Error:   err.Error(),
		})
	}

	// 检查可用性
	if !collector.IsAvailable() {
		return json.Marshal(command.K8sCollectResult{
			Success: false,
			Error:   "Kubernetes API is not available",
		})
	}

	// 采集 Pod 信息
	pods, err := collector.Collect(params.LabelSelector)
	if err != nil {
		return json.Marshal(command.K8sCollectResult{
			Success: false,
			Error:   err.Error(),
		})
	}

	// 获取摘要
	summary := collector.GetSummary(pods)

	// 转换为命令结果格式
	var podInfos []command.K8sPodInfoCmd
	for _, pod := range pods {
		var containers []command.K8sContainerStatusCmd
		for _, c := range pod.Containers {
			containers = append(containers, command.K8sContainerStatusCmd{
				Name:         c.Name,
				Image:        c.Image,
				Ready:        c.Ready,
				RestartCount: c.RestartCount,
				State:        c.State,
				StartedAt:    formatTime(c.StartedAt),
				Reason:       c.Reason,
				Message:      c.Message,
			})
		}

		podInfos = append(podInfos, command.K8sPodInfoCmd{
			Name:         pod.Name,
			Namespace:    pod.Namespace,
			UID:          pod.UID,
			Status:       pod.Status,
			Phase:        pod.Phase,
			HostIP:       pod.HostIP,
			PodIP:        pod.PodIP,
			StartTime:    formatTime(pod.StartTime),
			Labels:       pod.Labels,
			Containers:   containers,
			RestartCount: pod.RestartCount,
			Ready:        pod.Ready,
		})
	}

	return json.Marshal(command.K8sCollectResult{
		Success:      true,
		Pods:         podInfos,
		TotalCount:   summary.Total,
		RunningCount: summary.Running,
		ReadyCount:   summary.Ready,
		PendingCount: summary.Pending,
		FailedCount:  summary.Failed,
	})
}

// K8sReport 上报 K8s Pod 信息
// 命令类型: k8s.report
func K8sReport(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.K8sReportParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 这个处理器在客户端侧用于触发上报
	// 实际上报逻辑需要调用服务端 API

	return json.Marshal(command.K8sReportResult{
		Success: true,
	})
}

// formatTime 格式化时间
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
