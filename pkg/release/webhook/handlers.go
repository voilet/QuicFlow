package webhook

import (
	"encoding/json"
	"fmt"
)

// PayloadEvent Webhook 事件类型
type PayloadEvent string

const (
	// EventPush 代码推送事件
	EventPush PayloadEvent = "push"
	// EventTagCreate 标签创建事件
	EventTagCreate PayloadEvent = "tag_create"
	// EventTagDelete 标签删除事件
	EventTagDelete PayloadEvent = "tag_delete"
	// EventMergeRequest 合并请求事件 (GitLab)
	EventMergeRequest PayloadEvent = "merge_request"
	// EventPullRequest 拉取请求事件 (GitHub)
	EventPullRequest PayloadEvent = "pull_request"
	// EventRelease 发布事件
	EventRelease PayloadEvent = "release"
)

// PushInfo 推送信息
type PushInfo struct {
	Branch     string // 分支名
	Tag        string // 标签名 (如果有)
	Commit     string // 提交 SHA
	ShortSHA   string // 短 SHA (前 8 位)
	Committer  string // 提交者
	Message    string // 提交消息
	Timestamp  string // 时间戳
	CompareURL string // 对比 URL
}

// Handler Webhook 处理器接口
type Handler interface {
	// ParsePayload 解析 webhook payload
	ParsePayload(payload []byte, eventType string) (*PushInfo, error)
	// ExtractSignature 提取签名
	ExtractSignature(signatureHeader string) string
	// VerifySignature 验证签名
	VerifySignature(payload []byte, signature string) error
}

// GitHubHandler GitHub Webhook 处理器
type GitHubHandler struct {
	verifier *Verifier
}

// NewGitHubHandler 创建 GitHub 处理器
func NewGitHubHandler(secret string) *GitHubHandler {
	return &GitHubHandler{
		verifier: NewVerifier(secret),
	}
}

// GitHubPushPayload GitHub Push 事件 payload
type GitHubPushPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		FullName    string `json:"full_name"`
		CloneURL    string `json:"clone_url"`
		HTMLURL     string `json:"html_url"`
		Private     bool   `json:"private"`
	} `json:"repository"`
	HeadCommit struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
		URL string `json:"url"`
	} `json:"head_commit"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"commmits"`
	Compare  string `json:"compare,omitempty"`
	Sender   struct {
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	} `json:"sender"`
}

// GitHubReleasePayload GitHub Release 事件 payload
type GitHubReleasePayload struct {
	Action  string `json:"action"`
	Release struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Draft       bool   `json:"draft"`
		Prerelease  bool   `json:"prerelease"`
		CreatedAt   string `json:"created_at"`
		PublishedAt string `json:"published_at"`
	} `json:"release"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// ParsePayload 解析 GitHub webhook payload
func (h *GitHubHandler) ParsePayload(payload []byte, eventType string) (*PushInfo, error) {
	info := &PushInfo{}

	switch eventType {
	case "push":
		var data GitHubPushPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse push payload: %w", err)
		}

		// 解析分支或标签
		if len(data.Ref) > 0 {
			info.Branch = data.Ref
		}

		if len(data.HeadCommit.ID) > 0 {
			info.Commit = data.HeadCommit.ID
			info.ShortSHA = ShortSHA(data.HeadCommit.ID)
		}

		info.Committer = data.HeadCommit.Author.Name
		info.Message = data.HeadCommit.Message
		info.Timestamp = data.HeadCommit.Timestamp
		info.CompareURL = data.Compare

	case "release", "published":
		var data GitHubReleasePayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse release payload: %w", err)
		}

		info.Tag = data.Release.TagName
		info.Commit = data.Release.TagName
		info.Message = fmt.Sprintf("Release: %s", data.Release.Name)
		info.Committer = data.Sender.Login
		info.Timestamp = data.Release.PublishedAt

	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}

	return info, nil
}

// ExtractSignature 提取 GitHub 签名
func (h *GitHubHandler) ExtractSignature(signatureHeader string) string {
	sig, _ := ExtractSignature(signatureHeader)
	return sig
}

// VerifySignature 验证 GitHub 签名
func (h *GitHubHandler) VerifySignature(payload []byte, signature string) error {
	return h.verifier.VerifyGitHub(payload, signature)
}

// GitLabHandler GitLab Webhook 处理器
type GitLabHandler struct {
	verifier *Verifier
}

// NewGitLabHandler 创建 GitLab 处理器
func NewGitLabHandler(secret string) *GitLabHandler {
	return &GitLabHandler{
		verifier: NewVerifier(secret),
	}
}

// GitLabPushPayload GitLab Push 事件 payload
type GitLabPushPayload struct {
	ObjectKind  string `json:"object_kind"`
	Ref         string `json:"ref"`
	CheckoutSha string `json:"checkout_sha"`
	Project     struct {
		Name        string `json:"name"`
		FullName    string `json:"path_with_namespace"`
		HTTPURL     string `json:"http_url"`
		SSHURL      string `json:"ssh_url"`
		HomePage    string `json:"homepage"`
		Private     bool   `json:"private"`
	} `json:"project"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commmits"`
	TotalCommitsCount int `json:"total_commits_count"`
	UserUsername      string `json:"user_username"`
	UserEmail         string `json:"user_email"`
	UserName          string `json:"user_name"`
	Repository        struct {
		Name        string `json:"name"`
		HomePage    string `json:"homepage"`
		URL         string `json:"url"`
		Description string `json:"description"`
	} `json:"repository"`
}

// GitLabTagPayload GitLab Tag 事件 payload
type GitLabTagPayload struct {
	ObjectKind string `json:"object_kind"`
	Ref        string `json:"ref"`
	TagName    string `json:"ref"`
	Project    struct {
		Name     string `json:"name"`
		FullName string `json:"path_with_namespace"`
		HTTPURL  string `json:"http_url"`
	} `json:"project"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// ParsePayload 解析 GitLab webhook payload
func (h *GitLabHandler) ParsePayload(payload []byte, eventType string) (*PushInfo, error) {
	info := &PushInfo{}

	objectKind := eventType
	if objectKind == "" {
		// 尝试从 payload 中获取 object_kind
		var base struct {
			ObjectKind string `json:"object_kind"`
		}
		if err := json.Unmarshal(payload, &base); err == nil {
			objectKind = base.ObjectKind
		}
	}

	switch objectKind {
	case "push":
		var data GitLabPushPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse push payload: %w", err)
		}

		info.Branch = data.Ref
		info.Committer = data.UserName
		info.Timestamp = "" // GitLab push 不在顶层提供时间戳

		if len(data.Commits) > 0 {
			info.Commit = data.Commits[0].ID
			info.ShortSHA = ShortSHA(data.Commits[0].ID)
			info.Message = data.Commits[0].Message
		}

	case "tag_push":
		var data GitLabTagPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse tag_push payload: %w", err)
		}

		info.Tag = data.TagName
		info.Branch = data.TagName
		info.Committer = data.UserName

	case "release":
		// GitLab release events
		var data struct {
			TagName string `json:"tag"`
			Name    string `json:"name"`
			Action  string `json:"action"`
			Project struct {
				Name string `json:"name"`
			} `json:"project"`
			User struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"user"`
		}
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, fmt.Errorf("failed to parse release payload: %w", err)
		}

		info.Tag = data.TagName
		info.Branch = data.TagName
		info.Message = fmt.Sprintf("Release: %s", data.Name)
		info.Committer = data.User.Name

	default:
		return nil, fmt.Errorf("unsupported object_kind: %s", objectKind)
	}

	return info, nil
}

// ExtractSignature 提取 GitLab 签名
func (h *GitLabHandler) ExtractSignature(signatureHeader string) string {
	sig, _ := ExtractSignature(signatureHeader)
	return sig
}

// VerifySignature 验证 GitLab 签名
func (h *GitLabHandler) VerifySignature(payload []byte, signature string) error {
	return h.verifier.VerifyGitLab(payload, signature)
}

// ShortSHA 生成短 SHA (前 8 位)
func ShortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// GetHandler 根据来源平台获取对应的处理器
func GetHandler(source WebhookSource, secret string) Handler {
	switch source {
	case WebhookSourceGitHub:
		return NewGitHubHandler(secret)
	case WebhookSourceGitLab:
		return NewGitLabHandler(secret)
	case WebhookSourceGitee:
		// Gitee 与 GitHub 格式类似，复用处理器
		return NewGitHubHandler(secret)
	default:
		return nil
	}
}

// WebhookSource 从 models 迁移
type WebhookSource string

const (
	WebhookSourceGitHub WebhookSource = "github"
	WebhookSourceGitLab WebhookSource = "gitlab"
	WebhookSourceGitee  WebhookSource = "gitee"
)
