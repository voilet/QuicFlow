package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Info 容器信息
type Info struct {
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	Image         string    `json:"image"`
	Status        string    `json:"status"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"created_at"`
	StartedAt     time.Time `json:"started_at"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryUsage   int64     `json:"memory_usage"`
	MemoryLimit   int64     `json:"memory_limit"`
	MemoryPercent float64   `json:"memory_percent"`
	NetworkRx     int64     `json:"network_rx"`
	NetworkTx     int64     `json:"network_tx"`
	MatchedPrefix string    `json:"matched_prefix,omitempty"`
	MatchedProject string   `json:"matched_project,omitempty"`
}

// Collector 容器采集器
type Collector struct {
	socketPath string
	client     *http.Client
	prefixes   map[string]string // prefix -> projectID
}

// NewCollector 创建容器采集器
func NewCollector(socketPath string) *Collector {
	if socketPath == "" {
		socketPath = "/var/run/docker.sock"
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	return &Collector{
		socketPath: socketPath,
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		prefixes: make(map[string]string),
	}
}

// SetPrefixes 设置容器名称前缀到项目ID的映射
func (c *Collector) SetPrefixes(prefixes map[string]string) {
	c.prefixes = prefixes
}

// Collect 采集容器信息
func (c *Collector) Collect(all bool, prefixFilter []string) ([]Info, error) {
	// 获取容器列表
	url := "http://localhost/containers/json"
	if all {
		url += "?all=true"
	}

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("docker API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker API returned status %d", resp.StatusCode)
	}

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var result []Info
	for _, container := range containers {
		info := c.convertToInfo(container)

		// 前缀过滤
		if len(prefixFilter) > 0 {
			matched := false
			for _, prefix := range prefixFilter {
				if strings.HasPrefix(info.ContainerName, prefix) {
					matched = true
					info.MatchedPrefix = prefix
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 匹配项目
		for prefix, projectID := range c.prefixes {
			if strings.HasPrefix(info.ContainerName, prefix) {
				info.MatchedPrefix = prefix
				info.MatchedProject = projectID
				break
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// GetDockerVersion 获取 Docker 版本
func (c *Collector) GetDockerVersion() (string, error) {
	resp, err := c.client.Get("http://localhost/version")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var version struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return "", err
	}

	return version.Version, nil
}

// GetContainerStats 获取容器资源统计
func (c *Collector) GetContainerStats(containerID string) (*containerStats, error) {
	url := fmt.Sprintf("http://localhost/containers/%s/stats?stream=false", containerID)
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stats API returned status %d", resp.StatusCode)
	}

	var stats containerStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// convertToInfo 转换 Docker API 响应为 Info
func (c *Collector) convertToInfo(container dockerContainer) Info {
	name := ""
	if len(container.Names) > 0 {
		name = strings.TrimPrefix(container.Names[0], "/")
	}

	info := Info{
		ContainerID:   container.ID[:12],
		ContainerName: name,
		Image:         container.Image,
		Status:        container.Status,
		State:         container.State,
		CreatedAt:     time.Unix(container.Created, 0),
	}

	// 尝试获取统计信息（可能会失败）
	stats, err := c.GetContainerStats(container.ID)
	if err == nil && stats != nil {
		info.CPUPercent = c.calculateCPUPercent(stats)
		info.MemoryUsage = int64(stats.MemoryStats.Usage)
		info.MemoryLimit = int64(stats.MemoryStats.Limit)
		if info.MemoryLimit > 0 {
			info.MemoryPercent = float64(info.MemoryUsage) / float64(info.MemoryLimit) * 100
		}

		// 网络统计
		for _, netStats := range stats.Networks {
			info.NetworkRx += int64(netStats.RxBytes)
			info.NetworkTx += int64(netStats.TxBytes)
		}
	}

	return info
}

// calculateCPUPercent 计算 CPU 使用率
func (c *Collector) calculateCPUPercent(stats *containerStats) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent := (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100
		return cpuPercent
	}
	return 0
}

// dockerContainer Docker API 容器响应结构
type dockerContainer struct {
	ID      string   `json:"Id"`
	Names   []string `json:"Names"`
	Image   string   `json:"Image"`
	Status  string   `json:"Status"`
	State   string   `json:"State"`
	Created int64    `json:"Created"`
}

// containerStats Docker API 统计响应结构
type containerStats struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage int64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
		OnlineCPUs     int   `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage int64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage int64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
}

// IsDockerAvailable 检查 Docker 是否可用
func (c *Collector) IsDockerAvailable() bool {
	resp, err := c.client.Get("http://localhost/_ping")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// CollectByPrefixes 按前缀采集容器（简化接口）
func (c *Collector) CollectByPrefixes(prefixes []string) ([]Info, error) {
	return c.Collect(true, prefixes)
}

// Summary 容器统计摘要
type Summary struct {
	TotalCount   int
	RunningCount int
	StoppedCount int
}

// GetSummary 获取容器统计摘要
func (c *Collector) GetSummary(containers []Info) Summary {
	summary := Summary{
		TotalCount: len(containers),
	}

	for _, container := range containers {
		if container.State == "running" {
			summary.RunningCount++
		} else {
			summary.StoppedCount++
		}
	}

	return summary
}

// ReadDockerVersion 从 /proc 读取 Docker 版本（备用方案）
func ReadDockerVersion() string {
	// 尝试从 docker version 命令获取
	// 这是一个简化版本，实际应该执行命令
	return "unknown"
}

// ParseDockerPS 解析 docker ps 输出（备用方案）
func ParseDockerPS(output string) []Info {
	var result []Info
	scanner := bufio.NewScanner(strings.NewReader(output))

	// 跳过标题行
	if scanner.Scan() {
		// 标题行
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		info := Info{
			ContainerID:   fields[0],
			Image:         fields[1],
			Status:        strings.Join(fields[4:len(fields)-2], " "),
			ContainerName: fields[len(fields)-1],
		}

		if strings.Contains(info.Status, "Up") {
			info.State = "running"
		} else {
			info.State = "exited"
		}

		result = append(result, info)
	}

	return result
}
