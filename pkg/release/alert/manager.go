package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusFiring   AlertStatus = "firing"
	AlertStatusResolved AlertStatus = "resolved"
)

// Alert 通用告警结构
type Alert struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Level       AlertLevel        `json:"level"`
	Status      AlertStatus       `json:"status"`
	Source      string            `json:"source"`       // process, container, k8s
	ClientID    string            `json:"client_id"`
	ProjectID   string            `json:"project_id,omitempty"`
	Title       string            `json:"title"`
	Message     string            `json:"message"`
	Labels      map[string]string `json:"labels,omitempty"`
	Value       float64           `json:"value,omitempty"`
	Threshold   float64           `json:"threshold,omitempty"`
	FiredAt     time.Time         `json:"fired_at"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
	Fingerprint string            `json:"fingerprint"` // 用于去重
}

// NotificationChannel 通知渠道接口
type NotificationChannel interface {
	Send(ctx context.Context, alert *Alert) error
	Name() string
}

// Manager 告警管理器
type Manager struct {
	channels      []NotificationChannel
	activeAlerts  map[string]*Alert // fingerprint -> alert
	alertHistory  []*Alert
	historyLimit  int
	mu            sync.RWMutex

	// 告警抑制
	silences      map[string]time.Time // fingerprint -> 抑制到期时间
	silenceMu     sync.RWMutex

	// 告警聚合
	aggregateWindow time.Duration
	pendingAlerts   map[string][]*Alert // fingerprint前缀 -> alerts
	aggregateMu     sync.Mutex
}

// NewManager 创建告警管理器
func NewManager() *Manager {
	return &Manager{
		channels:        make([]NotificationChannel, 0),
		activeAlerts:    make(map[string]*Alert),
		alertHistory:    make([]*Alert, 0),
		historyLimit:    1000,
		silences:        make(map[string]time.Time),
		aggregateWindow: 30 * time.Second,
		pendingAlerts:   make(map[string][]*Alert),
	}
}

// AddChannel 添加通知渠道
func (m *Manager) AddChannel(channel NotificationChannel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channels = append(m.channels, channel)
}

// Fire 触发告警
func (m *Manager) Fire(ctx context.Context, alert *Alert) error {
	if alert.ID == "" {
		alert.ID = generateAlertID()
	}
	if alert.FiredAt.IsZero() {
		alert.FiredAt = time.Now()
	}
	if alert.Fingerprint == "" {
		alert.Fingerprint = generateFingerprint(alert)
	}
	alert.Status = AlertStatusFiring

	// 检查是否被抑制
	if m.isSilenced(alert.Fingerprint) {
		return nil
	}

	// 检查是否已有相同告警
	m.mu.Lock()
	if existing, ok := m.activeAlerts[alert.Fingerprint]; ok {
		// 更新现有告警
		existing.Value = alert.Value
		m.mu.Unlock()
		return nil
	}
	m.activeAlerts[alert.Fingerprint] = alert
	m.addToHistory(alert)
	m.mu.Unlock()

	// 发送通知
	return m.notify(ctx, alert)
}

// Resolve 解决告警
func (m *Manager) Resolve(ctx context.Context, fingerprint string) error {
	m.mu.Lock()
	alert, ok := m.activeAlerts[fingerprint]
	if !ok {
		m.mu.Unlock()
		return nil
	}

	now := time.Now()
	alert.Status = AlertStatusResolved
	alert.ResolvedAt = &now
	delete(m.activeAlerts, fingerprint)
	m.mu.Unlock()

	// 发送解决通知
	return m.notify(ctx, alert)
}

// Silence 抑制告警
func (m *Manager) Silence(fingerprint string, duration time.Duration) {
	m.silenceMu.Lock()
	defer m.silenceMu.Unlock()
	m.silences[fingerprint] = time.Now().Add(duration)
}

// isSilenced 检查是否被抑制
func (m *Manager) isSilenced(fingerprint string) bool {
	m.silenceMu.RLock()
	defer m.silenceMu.RUnlock()

	if expireAt, ok := m.silences[fingerprint]; ok {
		if time.Now().Before(expireAt) {
			return true
		}
		// 过期，清理
		delete(m.silences, fingerprint)
	}
	return false
}

// notify 发送通知
func (m *Manager) notify(ctx context.Context, alert *Alert) error {
	m.mu.RLock()
	channels := m.channels
	m.mu.RUnlock()

	var errs []string
	for _, ch := range channels {
		if err := ch.Send(ctx, alert); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", ch.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// addToHistory 添加到历史
func (m *Manager) addToHistory(alert *Alert) {
	m.alertHistory = append(m.alertHistory, alert)
	if len(m.alertHistory) > m.historyLimit {
		m.alertHistory = m.alertHistory[1:]
	}
}

// GetActiveAlerts 获取活跃告警
func (m *Manager) GetActiveAlerts() []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Alert, 0, len(m.activeAlerts))
	for _, alert := range m.activeAlerts {
		result = append(result, alert)
	}
	return result
}

// GetAlertHistory 获取告警历史
func (m *Manager) GetAlertHistory(limit int) []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.alertHistory) {
		limit = len(m.alertHistory)
	}

	// 返回最近的告警
	start := len(m.alertHistory) - limit
	result := make([]*Alert, limit)
	copy(result, m.alertHistory[start:])
	return result
}

// generateAlertID 生成告警ID
func generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().UnixNano())
}

// generateFingerprint 生成告警指纹
func generateFingerprint(alert *Alert) string {
	return fmt.Sprintf("%s:%s:%s:%s", alert.Source, alert.Type, alert.ClientID, alert.Title)
}

// ============================================================================
// 通知渠道实现
// ============================================================================

// WebhookChannel Webhook 通知渠道
type WebhookChannel struct {
	URL     string
	Headers map[string]string
	client  *http.Client
}

// NewWebhookChannel 创建 Webhook 渠道
func NewWebhookChannel(url string, headers map[string]string) *WebhookChannel {
	return &WebhookChannel{
		URL:     url,
		Headers: headers,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name 返回渠道名称
func (w *WebhookChannel) Name() string {
	return "webhook"
}

// Send 发送告警
func (w *WebhookChannel) Send(ctx context.Context, alert *Alert) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.URL, strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// LogChannel 日志通知渠道（用于调试）
type LogChannel struct {
	Logger func(format string, args ...interface{})
}

// NewLogChannel 创建日志渠道
func NewLogChannel(logger func(format string, args ...interface{})) *LogChannel {
	if logger == nil {
		logger = func(format string, args ...interface{}) {
			fmt.Printf(format+"\n", args...)
		}
	}
	return &LogChannel{Logger: logger}
}

// Name 返回渠道名称
func (l *LogChannel) Name() string {
	return "log"
}

// Send 发送告警
func (l *LogChannel) Send(ctx context.Context, alert *Alert) error {
	l.Logger("[ALERT] [%s] [%s] %s: %s (value=%.2f, threshold=%.2f)",
		alert.Level, alert.Status, alert.Type, alert.Message, alert.Value, alert.Threshold)
	return nil
}

// CallbackChannel 回调通知渠道
type CallbackChannel struct {
	Callback func(alert *Alert)
}

// NewCallbackChannel 创建回调渠道
func NewCallbackChannel(callback func(alert *Alert)) *CallbackChannel {
	return &CallbackChannel{Callback: callback}
}

// Name 返回渠道名称
func (c *CallbackChannel) Name() string {
	return "callback"
}

// Send 发送告警
func (c *CallbackChannel) Send(ctx context.Context, alert *Alert) error {
	if c.Callback != nil {
		c.Callback(alert)
	}
	return nil
}
