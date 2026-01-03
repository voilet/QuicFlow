package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/container"
	"github.com/voilet/quic-flow/pkg/release/executor"
	"github.com/voilet/quic-flow/pkg/release/models"
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

// ContainerDeploy 容器部署
// 命令类型: container.deploy
func ContainerDeploy(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ContainerDeployParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return json.Marshal(command.ContainerDeployResult{
			Success: false,
			Error:   fmt.Sprintf("invalid params: %v", err),
		})
	}

	result := &command.ContainerDeployResult{
		ContainerName: params.ContainerName,
	}

	// 验证必要参数
	if params.Image == "" {
		result.Error = "image is required"
		return json.Marshal(result)
	}
	if params.ContainerName == "" {
		params.ContainerName = "app"
		result.ContainerName = params.ContainerName
	}

	// 构建配置
	config := &models.ContainerDeployConfig{
		Image:           params.Image,
		ContainerName:   params.ContainerName,
		Registry:        params.Registry,
		RegistryUser:    params.RegistryUser,
		RegistryPass:    params.RegistryPass,
		ImagePullPolicy: params.ImagePullPolicy,
		RestartPolicy:   params.RestartPolicy,
		Command:         params.Command,
		Entrypoint:      params.Entrypoint,
		MemoryLimit:     params.MemoryLimit,
		CPULimit:        params.CPULimit,
		StopTimeout:     params.StopTimeout,
		RemoveOld:       params.RemoveOld,
		PullBeforeStop:  params.PullBeforeStop,
		Environment:     params.Environment,
		Networks:        params.Networks,
	}

	// 端口映射
	for _, p := range params.Ports {
		config.Ports = append(config.Ports, models.PortMapping{
			HostPort:      p.HostPort,
			ContainerPort: p.ContainerPort,
			Protocol:      p.Protocol,
			HostIP:        p.HostIP,
		})
	}

	// 卷挂载
	for _, v := range params.Volumes {
		config.Volumes = append(config.Volumes, models.VolumeMount{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      v.ReadOnly,
		})
	}

	// 健康检查
	if params.HealthCheck != nil {
		config.HealthCheck = &models.ContainerHealthCheck{
			Command:     params.HealthCheck.Command,
			Interval:    params.HealthCheck.Interval,
			Timeout:     params.HealthCheck.Timeout,
			Retries:     params.HealthCheck.Retries,
			StartPeriod: params.HealthCheck.StartPeriod,
		}
	}

	// 构建部署脚本
	builder := executor.NewDockerCommandBuilder(config)
	script, err := builder.BuildDeployScript(params.ContainerName)
	if err != nil {
		result.Error = fmt.Sprintf("build deploy script: %v", err)
		return json.Marshal(result)
	}

	// 执行部署脚本
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", script)
	output, err := cmd.CombinedOutput()
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("deploy failed: %v", err)
		return json.Marshal(result)
	}

	// 获取容器 ID
	inspectCmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Id}}", params.ContainerName)
	containerID, _ := inspectCmd.Output()
	result.ContainerID = strings.TrimSpace(string(containerID))

	result.Success = true
	result.ImagePulled = true
	result.OldRemoved = config.RemoveOld

	return json.Marshal(result)
}

// ContainerLogs 查看容器日志
// 命令类型: container.logs
func ContainerLogs(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.ContainerLogsParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return json.Marshal(command.ContainerLogsResult{
			Success: false,
			Error:   fmt.Sprintf("invalid params: %v", err),
		})
	}

	result := &command.ContainerLogsResult{}

	// 确定容器标识
	containerRef := params.ContainerName
	if containerRef == "" {
		containerRef = params.ContainerID
	}
	if containerRef == "" {
		result.Error = "container_id or container_name is required"
		return json.Marshal(result)
	}

	// 构建 docker logs 命令参数
	args := []string{"logs"}

	// 返回行数，默认 100
	tail := params.Tail
	if tail <= 0 {
		tail = 100
	}
	args = append(args, "--tail", fmt.Sprintf("%d", tail))

	// 时间范围
	if params.Since != "" {
		args = append(args, "--since", params.Since)
	}
	if params.Until != "" {
		args = append(args, "--until", params.Until)
	}

	// 时间戳
	if params.Timestamps {
		args = append(args, "--timestamps")
	}

	args = append(args, containerRef)

	// 执行命令
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("get logs failed: %v, output: %s", err, string(output))
		return json.Marshal(result)
	}

	logs := string(output)
	lineCount := strings.Count(logs, "\n")
	if len(logs) > 0 && !strings.HasSuffix(logs, "\n") {
		lineCount++
	}

	result.Success = true
	result.ContainerName = params.ContainerName
	result.ContainerID = params.ContainerID
	result.Logs = logs
	result.LineCount = lineCount

	return json.Marshal(result)
}
