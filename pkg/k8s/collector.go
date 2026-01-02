package k8s

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// PodInfo Pod 信息
type PodInfo struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	UID            string            `json:"uid"`
	Status         string            `json:"status"` // Running, Pending, Succeeded, Failed, Unknown
	Phase          string            `json:"phase"`
	HostIP         string            `json:"host_ip"`
	PodIP          string            `json:"pod_ip"`
	StartTime      time.Time         `json:"start_time"`
	Labels         map[string]string `json:"labels,omitempty"`
	Containers     []ContainerStatus `json:"containers"`
	RestartCount   int               `json:"restart_count"`
	Ready          bool              `json:"ready"`
	MatchedProject string            `json:"matched_project,omitempty"`
}

// ContainerStatus 容器状态
type ContainerStatus struct {
	Name         string    `json:"name"`
	Image        string    `json:"image"`
	Ready        bool      `json:"ready"`
	RestartCount int       `json:"restart_count"`
	State        string    `json:"state"` // running, waiting, terminated
	StartedAt    time.Time `json:"started_at,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	Message      string    `json:"message,omitempty"`
}

// Collector K8s Pod 采集器
type Collector struct {
	apiServer  string
	token      string
	caCert     string
	namespace  string
	httpClient *http.Client
}

// CollectorConfig 采集器配置
type CollectorConfig struct {
	APIServer   string // API Server 地址，默认使用集群内地址
	Token       string // Bearer Token
	TokenFile   string // Token 文件路径
	CACert      string // CA 证书路径
	Namespace   string // 监控的命名空间，空表示所有
	InCluster   bool   // 是否在集群内运行
	InsecureTLS bool   // 跳过TLS验证
}

// NewCollector 创建 K8s Pod 采集器
func NewCollector(config CollectorConfig) (*Collector, error) {
	c := &Collector{
		apiServer: config.APIServer,
		token:     config.Token,
		caCert:    config.CACert,
		namespace: config.Namespace,
	}

	// 如果在集群内运行，使用默认配置
	if config.InCluster {
		c.apiServer = "https://kubernetes.default.svc"

		// 读取 ServiceAccount Token
		if config.TokenFile == "" {
			config.TokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		}
		tokenBytes, err := os.ReadFile(config.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("read token file: %w", err)
		}
		c.token = strings.TrimSpace(string(tokenBytes))

		// CA 证书
		if config.CACert == "" {
			c.caCert = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
		}
	}

	// 创建 HTTP 客户端
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureTLS,
	}

	c.httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return c, nil
}

// IsAvailable 检查 K8s API 是否可用
func (c *Collector) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1", c.apiServer)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Collect 采集 Pod 信息
func (c *Collector) Collect(labelSelector string) ([]PodInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 构建 URL
	var url string
	if c.namespace != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/pods", c.apiServer, c.namespace)
	} else {
		url = fmt.Sprintf("%s/api/v1/pods", c.apiServer)
	}

	if labelSelector != "" {
		url += "?labelSelector=" + labelSelector
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var podList PodList
	if err := json.NewDecoder(resp.Body).Decode(&podList); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 转换为 PodInfo
	var result []PodInfo
	for _, item := range podList.Items {
		info := c.parsePod(item)
		result = append(result, info)
	}

	return result, nil
}

// GetPod 获取单个 Pod 信息
func (c *Collector) GetPod(namespace, name string) (*PodInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", c.apiServer, namespace, name)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var pod Pod
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	info := c.parsePod(pod)
	return &info, nil
}

// parsePod 解析 Pod 信息
func (c *Collector) parsePod(pod Pod) PodInfo {
	info := PodInfo{
		Name:      pod.Metadata.Name,
		Namespace: pod.Metadata.Namespace,
		UID:       pod.Metadata.UID,
		Labels:    pod.Metadata.Labels,
		Phase:     pod.Status.Phase,
		HostIP:    pod.Status.HostIP,
		PodIP:     pod.Status.PodIP,
	}

	// 解析启动时间
	if pod.Status.StartTime != "" {
		info.StartTime, _ = time.Parse(time.RFC3339, pod.Status.StartTime)
	}

	// 确定状态
	info.Status = c.determinePodStatus(pod)
	info.Ready = c.isPodReady(pod)

	// 解析容器状态
	for _, cs := range pod.Status.ContainerStatuses {
		container := ContainerStatus{
			Name:         cs.Name,
			Image:        cs.Image,
			Ready:        cs.Ready,
			RestartCount: cs.RestartCount,
		}

		info.RestartCount += cs.RestartCount

		// 解析容器状态
		if cs.State.Running != nil {
			container.State = "running"
			container.StartedAt, _ = time.Parse(time.RFC3339, cs.State.Running.StartedAt)
		} else if cs.State.Waiting != nil {
			container.State = "waiting"
			container.Reason = cs.State.Waiting.Reason
			container.Message = cs.State.Waiting.Message
		} else if cs.State.Terminated != nil {
			container.State = "terminated"
			container.Reason = cs.State.Terminated.Reason
			container.Message = cs.State.Terminated.Message
		}

		info.Containers = append(info.Containers, container)
	}

	return info
}

// determinePodStatus 确定 Pod 状态
func (c *Collector) determinePodStatus(pod Pod) string {
	// 检查是否正在删除
	if pod.Metadata.DeletionTimestamp != "" {
		return "Terminating"
	}

	// 检查条件
	for _, cond := range pod.Status.Conditions {
		if cond.Type == "Ready" && cond.Status == "True" {
			return "Running"
		}
	}

	// 检查容器状态
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			reason := cs.State.Waiting.Reason
			if reason == "CrashLoopBackOff" || reason == "Error" || reason == "ImagePullBackOff" {
				return reason
			}
		}
		if cs.State.Terminated != nil {
			return "Terminated"
		}
	}

	return pod.Status.Phase
}

// isPodReady 判断 Pod 是否就绪
func (c *Collector) isPodReady(pod Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == "Ready" {
			return cond.Status == "True"
		}
	}
	return false
}

// GetSummary 获取 Pod 摘要统计
func (c *Collector) GetSummary(pods []PodInfo) PodSummary {
	summary := PodSummary{
		Total: len(pods),
	}

	for _, pod := range pods {
		switch pod.Status {
		case "Running":
			summary.Running++
		case "Pending":
			summary.Pending++
		case "Succeeded":
			summary.Succeeded++
		case "Failed", "CrashLoopBackOff", "Error":
			summary.Failed++
		default:
			summary.Unknown++
		}

		if pod.Ready {
			summary.Ready++
		}
	}

	return summary
}

// PodSummary Pod 统计摘要
type PodSummary struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Unknown   int `json:"unknown"`
	Ready     int `json:"ready"`
}

// ============================================================================
// K8s API 响应结构
// ============================================================================

// PodList Pod 列表
type PodList struct {
	Items []Pod `json:"items"`
}

// Pod Pod 对象
type Pod struct {
	Metadata PodMetadata `json:"metadata"`
	Status   PodStatus   `json:"status"`
}

// PodMetadata Pod 元数据
type PodMetadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	UID               string            `json:"uid"`
	Labels            map[string]string `json:"labels"`
	DeletionTimestamp string            `json:"deletionTimestamp,omitempty"`
}

// PodStatus Pod 状态
type PodStatus struct {
	Phase             string             `json:"phase"`
	HostIP            string             `json:"hostIP"`
	PodIP             string             `json:"podIP"`
	StartTime         string             `json:"startTime"`
	Conditions        []PodCondition     `json:"conditions"`
	ContainerStatuses []K8sContainerStatus `json:"containerStatuses"`
}

// PodCondition Pod 条件
type PodCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

// K8sContainerStatus 容器状态
type K8sContainerStatus struct {
	Name         string         `json:"name"`
	Image        string         `json:"image"`
	Ready        bool           `json:"ready"`
	RestartCount int            `json:"restartCount"`
	State        ContainerState `json:"state"`
}

// ContainerState 容器状态详情
type ContainerState struct {
	Running    *ContainerStateRunning    `json:"running,omitempty"`
	Waiting    *ContainerStateWaiting    `json:"waiting,omitempty"`
	Terminated *ContainerStateTerminated `json:"terminated,omitempty"`
}

// ContainerStateRunning 运行状态
type ContainerStateRunning struct {
	StartedAt string `json:"startedAt"`
}

// ContainerStateWaiting 等待状态
type ContainerStateWaiting struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ContainerStateTerminated 终止状态
type ContainerStateTerminated struct {
	Reason   string `json:"reason"`
	Message  string `json:"message"`
	ExitCode int    `json:"exitCode"`
}
