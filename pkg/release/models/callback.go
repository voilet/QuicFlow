package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ==================== 回调配置模型 ====================

// CallbackConfig 回调配置（项目级）
type CallbackConfig struct {
	ID        string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ProjectID string           `gorm:"type:uuid;index;not null" json:"project_id"`
	Name      string           `gorm:"size:100;not null" json:"name"`
	Enabled   bool             `gorm:"default:true" json:"enabled"`
	Channels  CallbackChannels `gorm:"type:jsonb;not null" json:"channels"`
	Events    CallbackEvents   `gorm:"type:jsonb;not null" json:"events"`

	// 关联
	Project Project `gorm:"foreignKey:ProjectID" json:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CallbackConfig) TableName() string {
	return "callback_configs"
}

// CallbackChannels 回调渠道列表
type CallbackChannels []CallbackChannel

func (c CallbackChannels) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *CallbackChannels) Scan(value interface{}) error {
	if value == nil {
		*c = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// CallbackChannel 回调渠道配置
type CallbackChannel struct {
	Type    CallbackType `json:"type"`              // feishu, dingtalk, wechat, custom
	Enabled bool         `json:"enabled"`           // 是否启用
	Config  interface{}  `json:"config,omitempty"`  // 渠道特定配置
}

// CallbackChannelConfig 通用渠道配置（用于数据库存储和解析）
type CallbackChannelConfig struct {
	// 飞书配置
	Feishu *FeishuCallbackConfig `json:"feishu,omitempty"`
	// 钉钉配置
	DingTalk *DingTalkCallbackConfig `json:"dingtalk,omitempty"`
	// 企业微信配置
	WeChat *WeChatCallbackConfig `json:"wechat,omitempty"`
	// 自定义接口配置
	Custom *CustomCallbackConfig `json:"custom,omitempty"`
}

func (c CallbackChannelConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *CallbackChannelConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// CallbackType 回调渠道类型
type CallbackType string

const (
	CallbackTypeFeishu   CallbackType = "feishu"
	CallbackTypeDingTalk CallbackType = "dingtalk"
	CallbackTypeWeChat   CallbackType = "wechat"
	CallbackTypeCustom   CallbackType = "custom"
)

// FeishuCallbackConfig 飞书回调配置
type FeishuCallbackConfig struct {
	WebhookURL  string `json:"webhook_url"`                     // Webhook 地址
	SignSecret  string `json:"sign_secret,omitempty"`           // 签名密钥（可选）
	MsgTemplate string `json:"msg_template,omitempty"`          // 消息模板（可选）
}

func (f FeishuCallbackConfig) Value() (driver.Value, error) {
	return json.Marshal(f)
}

func (f *FeishuCallbackConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, f)
}

// DingTalkCallbackConfig 钉钉回调配置
type DingTalkCallbackConfig struct {
	WebhookURL  string `json:"webhook_url"`                     // Webhook 地址
	SignSecret  string `json:"sign_secret,omitempty"`           // 签名密钥（可选）
	MsgTemplate string `json:"msg_template,omitempty"`          // 消息模板（可选）
}

func (d DingTalkCallbackConfig) Value() (driver.Value, error) {
	return json.Marshal(d)
}

func (d *DingTalkCallbackConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, d)
}

// WeChatCallbackConfig 企业微信回调配置
type WeChatCallbackConfig struct {
	CorpID     string `json:"corp_id"`                          // 企业 ID
	AgentID    int64  `json:"agent_id"`                         // 应用 ID
	Secret     string `json:"secret"`                           // 应用密钥
	ToUser     string `json:"to_user,omitempty"`                // 接收用户（默认 @all）
	ToParty    string `json:"to_party,omitempty"`               // 接收部门
	ToTag      string `json:"to_tag,omitempty"`                 // 接收标签
	MsgTemplate string `json:"msg_template,omitempty"`          // 消息模板（可选）
}

func (w WeChatCallbackConfig) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func (w *WeChatCallbackConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, w)
}

// CustomCallbackConfig 自定义回调配置
type CustomCallbackConfig struct {
	URL           string            `json:"url"`                              // 回调 URL
	Method        string            `json:"method,omitempty"`                 // HTTP 方法（默认 POST）
	Headers       map[string]string `json:"headers,omitempty"`                // 请求头
	Timeout       int               `json:"timeout,omitempty"`                // 超时时间（秒，默认 30）
	RetryCount    int               `json:"retry_count,omitempty"`            // 重试次数（默认 3）
	RetryInterval int               `json:"retry_interval,omitempty"`         // 重试间隔（秒，默认 5）
	MsgTemplate   string            `json:"msg_template,omitempty"`           // 消息模板（可选）
}

func (c CustomCallbackConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *CustomCallbackConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// CallbackEvents 回调事件类型列表
type CallbackEvents []string

func (e CallbackEvents) Value() (driver.Value, error) {
	return json.Marshal(e)
}

func (e *CallbackEvents) Scan(value interface{}) error {
	if value == nil {
		*e = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, e)
}

// CallbackEventType 回调事件类型
type CallbackEventType string

const (
	CallbackEventCanaryStarted   CallbackEventType = "canary_started"    // 金丝雀开始
	CallbackEventCanaryCompleted CallbackEventType = "canary_completed"  // 金丝雀完成
	CallbackEventFullCompleted   CallbackEventType = "full_completed"    // 全量完成
)

// AllCallbackEvents 所有回调事件类型
var AllCallbackEvents = []CallbackEventType{
	CallbackEventCanaryStarted,
	CallbackEventCanaryCompleted,
	CallbackEventFullCompleted,
}

// ==================== 回调历史记录模型 ====================

// CallbackHistory 回调历史记录
type CallbackHistory struct {
	ID         string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TaskID     string           `gorm:"type:uuid;index;not null" json:"task_id"`
	ConfigID   string           `gorm:"type:uuid;index" json:"config_id"`
	EventType  CallbackEventType `gorm:"size:50;index;not null" json:"event_type"`
	Channel    CallbackType     `gorm:"size:20;not null;index" json:"channel"`
	Status     CallbackStatus   `gorm:"size:20;not null;index" json:"status"`
	Request    CallbackPayload  `gorm:"type:jsonb" json:"request"`
	Response   string           `gorm:"type:text" json:"response,omitempty"`
	Error      string           `gorm:"type:text" json:"error,omitempty"`
	RetryCount int              `gorm:"default:0" json:"retry_count"`
	Duration   int              `gorm:"default:0" json:"duration"` // 毫秒

	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (CallbackHistory) TableName() string {
	return "callback_history"
}

// CallbackStatus 回调状态
type CallbackStatus string

const (
	CallbackStatusPending CallbackStatus = "pending" // 等待重试
	CallbackStatusSuccess CallbackStatus = "success" // 成功
	CallbackStatusFailed  CallbackStatus = "failed"  // 失败
	CallbackStatusRetrying CallbackStatus = "retrying" // 重试中
)

// CallbackPayload 回调消息负载
type CallbackPayload struct {
	EventType   CallbackEventType `json:"event_type"`
	Project     CallbackProject   `json:"project"`
	Version     CallbackVersion   `json:"version"`
	Task        CallbackTask      `json:"task"`
	Deployment  CallbackDeployment `json:"deployment"`
	Timestamp   time.Time         `json:"timestamp"`
	Duration    string            `json:"duration,omitempty"`
	Environment string            `json:"environment"`
}

func (c CallbackPayload) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func (c *CallbackPayload) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// CallbackProject 项目信息
type CallbackProject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CallbackVersion 版本信息
type CallbackVersion struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CallbackTask 任务信息
type CallbackTask struct {
	ID      string       `json:"id"`
	Type    OperationType `json:"type"`
	Strategy StrategyType `json:"strategy,omitempty"`
	Status  string       `json:"status"`
}

// CallbackDeployment 部署信息
type CallbackDeployment struct {
	TotalCount     int      `json:"total_count"`
	CanaryCount    int      `json:"canary_count,omitempty"`
	CompletedCount int      `json:"completed_count"`
	FailedCount    int      `json:"failed_count"`
	Hosts          []string `json:"hosts"`
}

// ==================== 消息模板模型 ====================

// MessageTemplate 消息模板
type MessageTemplate struct {
	ID          string             `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ProjectID   *string            `gorm:"type:uuid;index" json:"project_id,omitempty"` // nil 表示全局模板
	Name        string             `gorm:"size:100;not null" json:"name"`
	Description string             `gorm:"size:500" json:"description,omitempty"`
	Content     string             `gorm:"type:text;not null" json:"content"`
	Type        MessageTemplateType `gorm:"size:20;not null;default:'custom'" json:"type"`
	Channel     CallbackType       `gorm:"size:20" json:"channel,omitempty"` // 适用的渠道，空表示通用
	IsDefault   bool               `gorm:"default:false" json:"is_default"`  // 是否默认模板
	Variables   TemplateVariables  `gorm:"type:jsonb" json:"variables,omitempty"` // 模板使用的变量

	// 关联
	Project *Project `gorm:"foreignKey:ProjectID" json:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MessageTemplate) TableName() string {
	return "message_templates"
}

// MessageTemplateType 模板类型
type MessageTemplateType string

const (
	MessageTemplateTypeSystem MessageTemplateType = "system"  // 系统内置模板
	MessageTemplateTypeCustom MessageTemplateType = "custom"  // 用户自定义模板
)

// TemplateVariables 模板变量列表
type TemplateVariables []string

func (v TemplateVariables) Value() (driver.Value, error) {
	return json.Marshal(v)
}

func (v *TemplateVariables) Scan(value interface{}) error {
	if value == nil {
		*v = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, v)
}

// CreateMessageTemplateRequest 创建模板请求
type CreateMessageTemplateRequest struct {
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	Content     string       `json:"content" binding:"required"`
	Channel     CallbackType `json:"channel"`
	IsDefault   bool         `json:"is_default"`
}

// UpdateMessageTemplateRequest 更新模板请求
type UpdateMessageTemplateRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Content     string       `json:"content"`
	Channel     CallbackType `json:"channel"`
	IsDefault   bool         `json:"is_default"`
}
