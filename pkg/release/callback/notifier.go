package callback

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"
)

// Notifier 回调通知器接口
type Notifier interface {
	// Send 发送回调通知
	Send(payload models.CallbackPayload) error
	// GetType 获取通知器类型
	GetType() models.CallbackType
}

// CallbackSender 回调发送器
type CallbackSender struct {
	httpClient *http.Client
	notifiers  map[models.CallbackType]Notifier
}

// NewCallbackSender 创建回调发送器
func NewCallbackSender() *CallbackSender {
	return &CallbackSender{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		notifiers: make(map[models.CallbackType]Notifier),
	}
}

// RegisterNotifier 注册通知器
func (s *CallbackSender) RegisterNotifier(notifier Notifier) {
	s.notifiers[notifier.GetType()] = notifier
}

// Send 发送回调通知
func (s *CallbackSender) Send(config *models.CallbackConfig, payload models.CallbackPayload, channelType models.CallbackType) error {
	notifier, ok := s.notifiers[channelType]
	if !ok {
		return fmt.Errorf("no notifier registered for channel type: %s", channelType)
	}

	return notifier.Send(payload)
}

// ==================== 飞书通知器 ====================

// FeishuNotifier 飞书通知器
type FeishuNotifier struct {
	httpClient     *http.Client
	config         *models.FeishuCallbackConfig
	templateEngine *TemplateEngine
}

// NewFeishuNotifier 创建飞书通知器
func NewFeishuNotifier(config *models.FeishuCallbackConfig) *FeishuNotifier {
	return &FeishuNotifier{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config:         config,
		templateEngine: NewTemplateEngine(),
	}
}

// GetType 获取通知器类型
func (n *FeishuNotifier) GetType() models.CallbackType {
	return models.CallbackTypeFeishu
}

// Send 发送飞书通知
func (n *FeishuNotifier) Send(payload models.CallbackPayload) error {
	msg := n.buildMessage(payload)

	// 构造请求体
	body := map[string]interface{}{
		"msg_type": "interactive",
		"card":     msg,
	}

	// 如果配置了签名密钥，添加签名
	if n.config.SignSecret != "" {
		timestamp := time.Now().Unix()
		sign := n.generateSign(timestamp, body)
		body["timestamp"] = fmt.Sprintf("%d", timestamp)
		body["sign"] = sign
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", n.config.WebhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应检查错误码
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu API error (code %d): %s", result.Code, result.Msg)
	}

	return nil
}

// buildMessage 构造飞书消息卡片
func (n *FeishuNotifier) buildMessage(payload models.CallbackPayload) map[string]interface{} {
	// 1. 如果配置了自定义模板，使用模板渲染
	if n.config.MsgTemplate != "" {
		card, err := n.renderCustomTemplate(payload)
		if err == nil {
			return card
		}
		// 模板渲染失败，回退到默认卡片构建
	}

	// 2. 使用卡片构建器构建默认卡片
	builder := NewFeishuCardBuilder(payload)
	return builder.BuildCard()
}

// renderCustomTemplate 渲染自定义模板
func (n *FeishuNotifier) renderCustomTemplate(payload models.CallbackPayload) (map[string]interface{}, error) {
	// 渲染模板
	rendered, err := n.templateEngine.Render(n.config.MsgTemplate, payload)
	if err != nil {
		return nil, err
	}

	// 解析为 JSON 对象
	var card map[string]interface{}
	if err := json.Unmarshal([]byte(rendered), &card); err != nil {
		return nil, fmt.Errorf("invalid card JSON: %w", err)
	}

	return card, nil
}

// generateSign 生成签名
func (n *FeishuNotifier) generateSign(timestamp int64, body map[string]interface{}) string {
	key := []byte(n.config.SignSecret)

	// 将 body 转换为 JSON 字符串
	bodyBytes, _ := json.Marshal(body)
	bodyStr := string(bodyBytes)

	// 构造签名串: timestamp + "\n" + body
	signStr := fmt.Sprintf("%d\n%s", timestamp, bodyStr)

	// 计算 HMAC-SHA256
	h := hmac.New(sha256.New, key)
	h.Write([]byte(signStr))

	// Base64 编码
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ==================== 钉钉通知器 ====================

// DingTalkNotifier 钉钉通知器
type DingTalkNotifier struct {
	httpClient *http.Client
	config     *models.DingTalkCallbackConfig
}

// NewDingTalkNotifier 创建钉钉通知器
func NewDingTalkNotifier(config *models.DingTalkCallbackConfig) *DingTalkNotifier {
	return &DingTalkNotifier{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// GetType 获取通知器类型
func (n *DingTalkNotifier) GetType() models.CallbackType {
	return models.CallbackTypeDingTalk
}

// Send 发送钉钉通知
func (n *DingTalkNotifier) Send(payload models.CallbackPayload) error {
	msg := n.buildMessage(payload)

	// 构造请求体
	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": "发布通知",
			"text":  msg,
		},
	}

	// 如果配置了签名密钥，添加签名
	if n.config.SignSecret != "" {
		timestamp := time.Now().UnixMilli()
		sign := n.generateSign(timestamp)
		body["timestamp"] = timestamp
		body["sign"] = sign
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", n.config.WebhookURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk API returned status %d", resp.StatusCode)
	}

	return nil
}

// buildMessage 构造钉钉 Markdown 消息
func (n *DingTalkNotifier) buildMessage(payload models.CallbackPayload) string {
	// 根据事件类型设置标题
	var title, statusIcon string

	switch payload.EventType {
	case models.CallbackEventCanaryStarted:
		title = "## 🚀 金丝雀发布开始"
		statusIcon = "🔄"
	case models.CallbackEventCanaryCompleted:
		title = "## ✅ 金丝雀发布完成"
		statusIcon = "🎉"
	case models.CallbackEventFullCompleted:
		title = "## 🎊 全部发布完成"
		statusIcon = "✨"
	default:
		title = "## 📢 发布通知"
		statusIcon = "📋"
	}

	text := title + "\n\n"

	// 项目和版本信息
	text += fmt.Sprintf("### 📋 发布信息\n\n")
	text += fmt.Sprintf("**项目**: %s\n\n", payload.Project.Name)
	text += fmt.Sprintf("**版本**: %s\n\n", payload.Version.Name)
	text += fmt.Sprintf("**环境**: %s\n\n", payload.Environment)
	text += fmt.Sprintf("**状态**: %s %s\n\n", statusIcon, payload.Task.Status)

	// 部署统计
	text += fmt.Sprintf("### 📊 部署统计\n\n")

	if payload.Deployment.CanaryCount > 0 {
		text += fmt.Sprintf("- 金丝雀数量: **%d**\n", payload.Deployment.CanaryCount)
	}

	text += fmt.Sprintf("- 总数量: **%d**\n", payload.Deployment.TotalCount)
	text += fmt.Sprintf("- 已完成: **%d**\n", payload.Deployment.CompletedCount)

	if payload.Deployment.FailedCount > 0 {
		text += fmt.Sprintf("- ❌ 失败: **%d**\n", payload.Deployment.FailedCount)
	}

	// 添加主机列表（最多显示 5 个）
	if len(payload.Deployment.Hosts) > 0 {
		text += fmt.Sprintf("\n### 🖥️ 主机列表\n\n")
		maxHosts := 5
		if len(payload.Deployment.Hosts) > maxHosts {
			text += fmt.Sprintf("- `%s` ... 等 %d 台主机\n", payload.Deployment.Hosts[0], len(payload.Deployment.Hosts))
		} else {
			for _, host := range payload.Deployment.Hosts {
				text += fmt.Sprintf("- `%s`\n", host)
			}
		}
	}

	// 时间戳
	if !payload.Timestamp.IsZero() {
		text += fmt.Sprintf("\n\n---\n\n_发布时间: %s_", payload.Timestamp.Format("2006-01-02 15:04:05"))
	}

	return text
}

// generateSign 生成签名
func (n *DingTalkNotifier) generateSign(timestamp int64) string {
	key := []byte(n.config.SignSecret)

	// 构造签名串: timestamp + "\n" + secret
	signStr := fmt.Sprintf("%d\n%s", timestamp, n.config.SignSecret)

	// 计算 HMAC-SHA256
	h := hmac.New(sha256.New, key)
	h.Write([]byte(signStr))

	// Base64 编码
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ==================== 企业微信通知器 ====================

// WeChatNotifier 企业微信通知器
type WeChatNotifier struct {
	httpClient *http.Client
	config     *models.WeChatCallbackConfig
	accessToken string
	tokenExpiry time.Time
}

// NewWeChatNotifier 创建企业微信通知器
func NewWeChatNotifier(config *models.WeChatCallbackConfig) *WeChatNotifier {
	return &WeChatNotifier{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// GetType 获取通知器类型
func (n *WeChatNotifier) GetType() models.CallbackType {
	return models.CallbackTypeWeChat
}

// Send 发送企业微信通知
func (n *WeChatNotifier) Send(payload models.CallbackPayload) error {
	// 获取 access_token
	token, err := n.getAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	msg := n.buildMessage(payload)

	// 构造请求体
	body := map[string]interface{}{
		"touser":  n.getToUser(),
		"msgtype": "markdown",
		"agentid": n.config.AgentID,
		"markdown": map[string]interface{}{
			"content": msg,
		},
	}

	// 添加部门或标签（如果配置）
	if n.config.ToParty != "" {
		body["toparty"] = n.config.ToParty
	}
	if n.config.ToTag != "" {
		body["totag"] = n.config.ToTag
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 发送请求
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wechat API returned status %d", resp.StatusCode)
	}

	// 解析响应检查错误码
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("wechat API error (code %d): %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// getAccessToken 获取 access_token
func (n *WeChatNotifier) getAccessToken() (string, error) {
	// 如果 token 未过期，直接返回
	if n.accessToken != "" && time.Now().Before(n.tokenExpiry) {
		return n.accessToken, nil
	}

	// 请求新的 access_token
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		n.config.CorpID, n.config.Secret)

	resp, err := n.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("wechat API error (code %d): %s", result.ErrCode, result.ErrMsg)
	}

	// 缓存 token，提前 5 分钟过期
	n.accessToken = result.AccessToken
	n.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	return n.accessToken, nil
}

// getToUser 获取接收用户
func (n *WeChatNotifier) getToUser() string {
	if n.config.ToUser != "" {
		return n.config.ToUser
	}
	return "@all" // 默认发送给所有人
}

// buildMessage 构造企业微信 Markdown 消息
func (n *WeChatNotifier) buildMessage(payload models.CallbackPayload) string {
	// 根据事件类型设置标题
	var title, statusIcon string

	switch payload.EventType {
	case models.CallbackEventCanaryStarted:
		title = "## 🚀 金丝雀发布开始"
		statusIcon = "🔄"
	case models.CallbackEventCanaryCompleted:
		title = "## ✅ 金丝雀发布完成"
		statusIcon = "🎉"
	case models.CallbackEventFullCompleted:
		title = "## 🎊 全部发布完成"
		statusIcon = "✨"
	default:
		title = "## 📢 发布通知"
		statusIcon = "📋"
	}

	text := title + "\n"

	// 项目和版本信息
	text += fmt.Sprintf("### 📋 发布信息\n\n")
	text += fmt.Sprintf("**项目**: %s\n", payload.Project.Name)
	text += fmt.Sprintf("**版本**: %s\n", payload.Version.Name)
	text += fmt.Sprintf("**环境**: %s\n", payload.Environment)
	text += fmt.Sprintf("**状态**: %s %s\n", statusIcon, payload.Task.Status)

	// 部署统计
	text += fmt.Sprintf("### 📊 部署统计\n\n")

	if payload.Deployment.CanaryCount > 0 {
		text += fmt.Sprintf(">金丝雀数量: <font color=\"info\">%d</font>\n", payload.Deployment.CanaryCount)
	}

	text += fmt.Sprintf(">总数量: <font color=\"info\">%d</font>\n", payload.Deployment.TotalCount)
	text += fmt.Sprintf(">已完成: <font color=\"info\">%d</font>\n", payload.Deployment.CompletedCount)

	if payload.Deployment.FailedCount > 0 {
		text += fmt.Sprintf(">失败: <font color=\"warning\">%d</font>\n", payload.Deployment.FailedCount)
	}

	// 添加主机列表（最多显示 5 个）
	if len(payload.Deployment.Hosts) > 0 {
		text += fmt.Sprintf("\n### 🖥️ 主机列表\n\n")
		maxHosts := 5
		if len(payload.Deployment.Hosts) > maxHosts {
			text += fmt.Sprintf("- `%s` ... 等 %d 台主机\n", payload.Deployment.Hosts[0], len(payload.Deployment.Hosts))
		} else {
			for _, host := range payload.Deployment.Hosts {
				text += fmt.Sprintf("- `%s`\n", host)
			}
		}
	}

	// 时间戳
	if !payload.Timestamp.IsZero() {
		text += fmt.Sprintf("\n_发布时间: %s_\n", payload.Timestamp.Format("2006-01-02 15:04:05"))
	}

	return text
}

// ==================== 自定义 HTTP 回调通知器 ====================

// CustomNotifier 自定义 HTTP 回调通知器
type CustomNotifier struct {
	httpClient     *http.Client
	config         *models.CustomCallbackConfig
	templateEngine *TemplateEngine
}

// NewCustomNotifier 创建自定义回调通知器
func NewCustomNotifier(config *models.CustomCallbackConfig) *CustomNotifier {
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	return &CustomNotifier{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		config:         config,
		templateEngine: NewTemplateEngine(),
	}
}

// GetType 获取通知器类型
func (n *CustomNotifier) GetType() models.CallbackType {
	return models.CallbackTypeCustom
}

// Send 发送自定义 HTTP 回调通知
func (n *CustomNotifier) Send(payload models.CallbackPayload) error {
	// 如果配置了消息模板，使用模板渲染
	body := n.buildBody(payload)

	// 序列化为 JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 确定请求方法
	method := n.config.Method
	if method == "" {
		method = "POST"
	}

	// 创建请求
	req, err := http.NewRequest(method, n.config.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	for key, value := range n.config.Headers {
		req.Header.Set(key, value)
	}

	// 执行请求（带重试）
	var lastErr error
	maxRetries := n.config.RetryCount
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for i := 0; i < maxRetries; i++ {
		resp, err := n.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request (attempt %d/%d): %w", i+1, maxRetries, err)
			// 如果不是最后一次尝试，等待后重试
			if i < maxRetries-1 {
				interval := time.Duration(n.config.RetryInterval) * time.Second
				if interval <= 0 {
					interval = 5 * time.Second
				}
				time.Sleep(interval)
				continue
			}
			break
		}

		// 读取响应体
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 检查状态码
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("custom callback returned status %d: %s", resp.StatusCode, string(respBody))

		// 如果不是最后一次尝试且是可重试的状态码，等待后重试
		if i < maxRetries-1 && n.isRetryableStatus(resp.StatusCode) {
			interval := time.Duration(n.config.RetryInterval) * time.Second
			if interval <= 0 {
				interval = 5 * time.Second
			}
			time.Sleep(interval)
			continue
		}
		break
	}

	return lastErr
}

// buildBody 构造请求体
func (n *CustomNotifier) buildBody(payload models.CallbackPayload) interface{} {
	// 如果配置了自定义模板，使用增强版模板引擎
	if n.config.MsgTemplate != "" {
		rendered, err := n.templateEngine.Render(n.config.MsgTemplate, payload)
		if err != nil {
			// 渲染失败，返回原始 payload
			return payload
		}

		// 尝试解析为 JSON 对象
		var jsonResult map[string]interface{}
		if err := json.Unmarshal([]byte(rendered), &jsonResult); err == nil {
			return jsonResult
		}
		// 如果不是有效 JSON，返回字符串作为消息
		return map[string]interface{}{"message": rendered}
	}

	// 默认返回标准负载
	return payload
}

// isRetryableStatus 判断状态码是否可重试
func (n *CustomNotifier) isRetryableStatus(statusCode int) bool {
	// 可重试的状态码：5xx 服务器错误、429 请求过多
	return statusCode == 429 || statusCode >= 500
}
