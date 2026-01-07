package models

import (
	"time"

	"gorm.io/gorm"
)

// ==================== 凭证管理模型 ====================

// CredentialType 凭证类型
type CredentialType string

const (
	CredentialTypeDockerRegistry CredentialType = "docker_registry" // Docker 镜像仓库
	CredentialTypeGitSSH         CredentialType = "git_ssh"          // Git SSH 私钥
	CredentialTypeGitToken       CredentialType = "git_token"        // Git 访问令牌
	CredentialTypeUsernamePassword CredentialType = "username_password" // 用户名密码
)

// CredentialScope 凭证范围
type CredentialScope string

const (
	CredentialScopeGlobal  CredentialScope = "global"  // 全局凭证
	CredentialScopeProject CredentialScope = "project" // 项目专属凭证
)

// Credential 凭证（敏感信息加密存储）
type Credential struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string          `gorm:"size:100;not null" json:"name"`
	Type      CredentialType  `gorm:"size:50;not null;index" json:"type"`
	Scope     CredentialScope `gorm:"size:20;not null;default:'global';index" json:"scope"`
	ProjectID *string         `gorm:"type:uuid;index" json:"project_id,omitempty"` // 项目凭证时关联
	Description string         `gorm:"size:500" json:"description"`

	// 加密存储的凭证数据
	// 根据类型存储不同内容:
	// - docker_registry: {"server_url": "...", "username": "...", "password": "..."}
	// - git_ssh: {"ssh_key": "...", "passphrase": "..."}
	// - git_token: {"server_url": "...", "token": "..."}
	// - username_password: {"server_url": "...", "username": "...", "password": "..."}
	EncryptedData string `gorm:"type:text;not null" json:"-"` // 加密后的 JSON

	// 统计信息（用于列表展示，不敏感）
	UseCount   int        `gorm:"default:0" json:"use_count"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	CreatedBy string     `gorm:"size:100;not null" json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Project *Project `gorm:"foreignKey:ProjectID" json:"-"`
}

func (Credential) TableName() string {
	return "credentials"
}

// CredentialData 解密后的凭证数据
type CredentialData struct {
	// Docker Registry / 用户名密码
	ServerURL string `json:"server_url,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`

	// Git SSH
	SSHKey      string `json:"ssh_key,omitempty"`
	SSHPassphrase string `json:"ssh_passphrase,omitempty"`

	// Git Token (复用 Password 字段存储 Token)
	Token string `json:"token,omitempty"`
}

// ProjectCredential 项目与凭证的关联关系
type ProjectCredential struct {
	ProjectID   string `gorm:"type:uuid;primaryKey" json:"project_id"`
	CredentialID string `gorm:"type:uuid;primaryKey" json:"credential_id"`
	Alias       string `gorm:"size:100" json:"alias"` // 在项目中的显示名称

	CreatedAt time.Time `json:"created_at"`

	// 关联
	Credential *Credential `gorm:"foreignKey:CredentialID" json:"credential,omitempty"`
}

func (ProjectCredential) TableName() string {
	return "project_credentials"
}

// ==================== Webhook 触发器模型 ====================

// WebhookSource Webhook 来源平台
type WebhookSource string

const (
	WebhookSourceGitHub  WebhookSource = "github"
	WebhookSourceGitLab  WebhookSource = "gitlab"
	WebhookSourceGitee   WebhookSource = "gitee"
)

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ProjectID string          `gorm:"type:uuid;index;not null" json:"project_id"`
	Name      string          `gorm:"size:100;not null" json:"name"`
	Enabled   bool            `gorm:"default:true" json:"enabled"`

	// 触发条件
	Source       WebhookSource `gorm:"size:20;not null;index" json:"source"`
	BranchFilter StringSlice   `gorm:"type:jsonb" json:"branch_filter"` // ["main", "release/*"]
	EventTypes   StringSlice   `gorm:"type:jsonb" json:"event_types"`   // ["push", "tag_create"]

	// 触发动作
	Action    string `gorm:"size:20;not null" json:"action"` // deploy, build
	TargetEnv string `gorm:"size:50" json:"target_env"`       // 目标环境名称
	AutoDeploy bool   `gorm:"default:false" json:"auto_deploy"` // 是否自动部署

	// Webhook 信息
	Secret string `gorm:"type:text;not null" json:"-"` // HMAC 密钥（加密存储）
	URL    string `gorm:"size:500;not null" json:"url"`  // 公开 URL

	// 元信息
	LastTriggerAt *time.Time `json:"last_trigger_at,omitempty"`
	TriggerCount  int       `gorm:"default:0" json:"trigger_count"`

	CreatedBy string     `gorm:"size:100;not null" json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Project *Project `gorm:"foreignKey:ProjectID" json:"-"`
}

func (WebhookConfig) TableName() string {
	return "webhook_configs"
}

// TriggerRecord Webhook 触发记录
type TriggerRecord struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	WebhookID string          `gorm:"type:uuid;index;not null" json:"webhook_id"`
	Source    WebhookSource   `gorm:"size:20;not null;index" json:"source"`

	// Git 推送信息
	Branch    string `gorm:"size:200;index" json:"branch"`
	Tag       string `gorm:"size:200" json:"tag,omitempty"`
	Commit    string `gorm:"size:100" json:"commit"`
	ShortSHA  string `gorm:"size:50" json:"short_sha"`
	Committer string `gorm:"size:100" json:"committer"`
	Message   string `gorm:"type:text" json:"message"`

	// 触发结果
	TaskID  *string `gorm:"type:uuid" json:"task_id,omitempty"`
	Status  string  `gorm:"size:20;not null;index" json:"status"` // success, failed, skipped
	Error   string  `gorm:"type:text" json:"error,omitempty"`

	// 原始负载（用于调试）
	Payload string `gorm:"type:text" json:"-"`

	TriggeredAt time.Time `gorm:"index;not null" json:"triggered_at"`

	// 关联
	Webhook *WebhookConfig `gorm:"foreignKey:WebhookID" json:"-"`
}

func (TriggerRecord) TableName() string {
	return "trigger_records"
}

// TriggerStatus 触发状态
type TriggerStatus string

const (
	TriggerStatusSuccess TriggerStatus = "success"
	TriggerStatusFailed  TriggerStatus = "failed"
	TriggerStatusSkipped TriggerStatus = "skipped"
)

// ==================== 成员权限模型 ====================

// ProjectMemberRole 项目成员角色
type ProjectMemberRole string

const (
	ProjectRoleOwner      ProjectMemberRole = "owner"      // 所有者（完全控制）
	ProjectRoleMaintainer ProjectMemberRole = "maintainer" // 维护者（可部署、可配置）
	ProjectRoleDeveloper  ProjectMemberRole = "developer"  // 开发者（开发环境部署）
	ProjectRoleViewer     ProjectMemberRole = "viewer"     // 访客（只读）
)

// ProjectMember 项目成员
type ProjectMember struct {
	ID        string             `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ProjectID string             `gorm:"type:uuid;index:idx_project_member;not null" json:"project_id"`
	UserID    string             `gorm:"size:100;index:idx_project_member;not null" json:"user_id"`
	Role      ProjectMemberRole  `gorm:"size:20;not null" json:"role"`

	AddedBy string     `gorm:"size:100;not null" json:"added_by"`
	AddedAt time.Time  `gorm:"not null" json:"added_at"`

	// 关联
	Project *Project `gorm:"foreignKey:ProjectID" json:"-"`
}

func (ProjectMember) TableName() string {
	return "project_members"
}

// User 用户
type User struct {
	ID          string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Username    string     `gorm:"uniqueIndex;size:50;not null" json:"username"`
	DisplayName string     `gorm:"size:100" json:"display_name"`
	Email       string     `gorm:"size:200;index" json:"email"`
	Avatar      string     `gorm:"size:500" json:"avatar,omitempty"`
	IsAdmin     bool       `gorm:"default:false" json:"is_admin"` // 超级管理员
	Status      string     `gorm:"size:20;default:'active'" json:"status"` // active, disabled

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)
