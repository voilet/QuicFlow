package git

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/voilet/quic-flow/pkg/command"
)

// Client Git 客户端（Server 端直接执行）
type Client struct {
	// 认证配置
	AuthType string // none, ssh, token, basic
	SSHKey   string // SSH 私钥内容
	Token    string // Access Token
	Username string // 用户名
	Password string // 密码
}

// NewClient 创建 Git 客户端
func NewClient() *Client {
	return &Client{}
}

// NewClientWithAuth 创建带认证的 Git 客户端
func NewClientWithAuth(authType, sshKey, token, username, password string) *Client {
	return &Client{
		AuthType: authType,
		SSHKey:   sshKey,
		Token:    token,
		Username: username,
		Password: password,
	}
}

// FetchVersionsRequest 获取版本请求
type FetchVersionsRequest struct {
	RepoURL         string
	MaxTags         int
	MaxCommits      int
	IncludeBranches bool
}

// FetchVersionsResult 获取版本结果
type FetchVersionsResult struct {
	Success       bool              `json:"success"`
	RepoURL       string            `json:"repo_url"`
	DefaultBranch string            `json:"default_branch,omitempty"`
	Tags          []command.GitTag    `json:"tags,omitempty"`
	Branches      []command.GitBranch `json:"branches,omitempty"`
	RecentCommits []command.GitCommit `json:"recent_commits,omitempty"`
	Error         string            `json:"error,omitempty"`
}

// FetchVersions 获取 Git 仓库版本信息（不需要 clone）
func (c *Client) FetchVersions(ctx context.Context, req *FetchVersionsRequest) (*FetchVersionsResult, error) {
	result := &FetchVersionsResult{
		RepoURL: req.RepoURL,
	}

	if req.RepoURL == "" {
		result.Error = "repo_url is required"
		return result, fmt.Errorf("repo_url is required")
	}

	// 设置默认值
	maxTags := req.MaxTags
	if maxTags <= 0 {
		maxTags = 20
	}
	maxCommits := req.MaxCommits
	if maxCommits <= 0 {
		maxCommits = 10
	}

	// 构建认证 URL
	authURL, err := c.buildAuthURL(req.RepoURL)
	if err != nil {
		result.Error = fmt.Sprintf("build auth URL: %v", err)
		return result, err
	}

	// 设置 SSH 环境（如果需要）
	cleanup, err := c.setupSSHEnv()
	if err != nil {
		result.Error = fmt.Sprintf("setup SSH: %v", err)
		return result, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// 获取远程引用
	refs, err := c.lsRemote(ctx, authURL)
	if err != nil {
		result.Error = fmt.Sprintf("ls-remote: %v", err)
		return result, err
	}

	// 解析标签
	tags := c.parseTags(refs)
	if len(tags) > maxTags {
		tags = tags[:maxTags]
	}
	result.Tags = tags

	// 解析分支
	if req.IncludeBranches {
		branches, defaultBranch := c.parseBranches(refs)
		result.Branches = branches
		result.DefaultBranch = defaultBranch
	} else {
		// 只获取默认分支
		_, defaultBranch := c.parseBranches(refs)
		result.DefaultBranch = defaultBranch
	}

	// 获取最近提交（从默认分支）
	if maxCommits > 0 && result.DefaultBranch != "" {
		commits, err := c.fetchRecentCommits(ctx, authURL, result.DefaultBranch, maxCommits)
		if err == nil {
			result.RecentCommits = commits
		}
	}

	result.Success = true
	return result, nil
}

// buildAuthURL 构建带认证的 URL
func (c *Client) buildAuthURL(repoURL string) (string, error) {
	switch c.AuthType {
	case "token":
		// 将 token 嵌入 URL
		if c.Token != "" {
			u, err := url.Parse(repoURL)
			if err != nil {
				return repoURL, nil
			}
			// GitHub/GitLab token 格式：https://token@github.com/...
			u.User = url.User(c.Token)
			return u.String(), nil
		}
	case "basic":
		// Basic 认证
		if c.Username != "" && c.Password != "" {
			u, err := url.Parse(repoURL)
			if err != nil {
				return repoURL, nil
			}
			u.User = url.UserPassword(c.Username, c.Password)
			return u.String(), nil
		}
	}
	return repoURL, nil
}

// setupSSHEnv 设置 SSH 环境
func (c *Client) setupSSHEnv() (func(), error) {
	if c.AuthType != "ssh" || c.SSHKey == "" {
		return nil, nil
	}

	// 创建临时 SSH 密钥文件
	tmpDir, err := os.MkdirTemp("", "git-ssh-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	keyFile := filepath.Join(tmpDir, "id_rsa")
	if err := os.WriteFile(keyFile, []byte(c.SSHKey), 0600); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("write key file: %w", err)
	}

	// 设置 GIT_SSH_COMMAND
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null", keyFile)
	os.Setenv("GIT_SSH_COMMAND", sshCmd)

	cleanup := func() {
		os.Unsetenv("GIT_SSH_COMMAND")
		os.RemoveAll(tmpDir)
	}

	return cleanup, nil
}

// lsRemote 执行 git ls-remote
func (c *Client) lsRemote(ctx context.Context, repoURL string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--refs", repoURL)
	cmd.Env = os.Environ()

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git ls-remote failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	refs := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			commit := parts[0]
			ref := parts[1]
			refs[ref] = commit
		}
	}

	return refs, nil
}

// parseTags 解析标签
func (c *Client) parseTags(refs map[string]string) []command.GitTag {
	var tags []command.GitTag

	for ref, commit := range refs {
		if strings.HasPrefix(ref, "refs/tags/") {
			name := strings.TrimPrefix(ref, "refs/tags/")
			// 跳过 ^{} 后缀的引用
			if strings.HasSuffix(name, "^{}") {
				continue
			}
			tags = append(tags, command.GitTag{
				Name:   name,
				Commit: commit[:8],
			})
		}
	}

	// 按版本号排序（使用语义化版本排序）
	sort.Slice(tags, func(i, j int) bool {
		return compareVersions(tags[i].Name, tags[j].Name) > 0
	})

	return tags
}

// parseBranches 解析分支
func (c *Client) parseBranches(refs map[string]string) ([]command.GitBranch, string) {
	var branches []command.GitBranch
	defaultBranch := ""

	// 检查 HEAD 指向
	if headRef, ok := refs["HEAD"]; ok {
		for ref := range refs {
			if refs[ref] == headRef && strings.HasPrefix(ref, "refs/heads/") {
				defaultBranch = strings.TrimPrefix(ref, "refs/heads/")
				break
			}
		}
	}

	// 如果没找到，默认为 main 或 master
	if defaultBranch == "" {
		if _, ok := refs["refs/heads/main"]; ok {
			defaultBranch = "main"
		} else if _, ok := refs["refs/heads/master"]; ok {
			defaultBranch = "master"
		}
	}

	for ref, commit := range refs {
		if strings.HasPrefix(ref, "refs/heads/") {
			name := strings.TrimPrefix(ref, "refs/heads/")
			branches = append(branches, command.GitBranch{
				Name:      name,
				Commit:    commit[:8],
				IsDefault: name == defaultBranch,
				IsRemote:  true,
			})
		}
	}

	// 将默认分支放在第一位
	sort.Slice(branches, func(i, j int) bool {
		if branches[i].IsDefault {
			return true
		}
		if branches[j].IsDefault {
			return false
		}
		return branches[i].Name < branches[j].Name
	})

	return branches, defaultBranch
}

// fetchRecentCommits 获取最近提交
func (c *Client) fetchRecentCommits(ctx context.Context, repoURL, branch string, limit int) ([]command.GitCommit, error) {
	// 创建临时目录进行浅克隆
	tmpDir, err := os.MkdirTemp("", "git-fetch-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	// 浅克隆仓库
	cloneCmd := exec.CommandContext(ctx, "git", "clone",
		"--depth", fmt.Sprintf("%d", limit+5),
		"--single-branch",
		"--branch", branch,
		repoURL,
		tmpDir,
	)
	cloneCmd.Env = os.Environ()
	if err := cloneCmd.Run(); err != nil {
		return nil, fmt.Errorf("clone: %w", err)
	}

	// 获取提交日志
	logCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "log",
		"--format=%H|%h|%an|%ae|%s|%ci",
		fmt.Sprintf("-n%d", limit),
	)
	output, err := logCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("log: %w", err)
	}

	var commits []command.GitCommit
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) >= 6 {
			commits = append(commits, command.GitCommit{
				FullHash:  parts[0],
				Hash:      parts[1],
				Author:    parts[2],
				Email:     parts[3],
				Message:   parts[4],
				CreatedAt: parts[5],
			})
		}
	}

	return commits, nil
}

// compareVersions 比较版本号（语义化版本）
func compareVersions(a, b string) int {
	// 去除 v 前缀
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	// 提取版本号部分
	re := regexp.MustCompile(`^(\d+)(?:\.(\d+))?(?:\.(\d+))?`)

	matchA := re.FindStringSubmatch(a)
	matchB := re.FindStringSubmatch(b)

	if matchA == nil && matchB == nil {
		return strings.Compare(a, b)
	}
	if matchA == nil {
		return -1
	}
	if matchB == nil {
		return 1
	}

	// 比较主版本号
	for i := 1; i <= 3; i++ {
		var numA, numB int
		if i < len(matchA) && matchA[i] != "" {
			fmt.Sscanf(matchA[i], "%d", &numA)
		}
		if i < len(matchB) && matchB[i] != "" {
			fmt.Sscanf(matchB[i], "%d", &numB)
		}
		if numA != numB {
			return numA - numB
		}
	}

	return 0
}

// ValidateRepo 验证仓库是否可访问
func (c *Client) ValidateRepo(ctx context.Context, repoURL string) error {
	authURL, err := c.buildAuthURL(repoURL)
	if err != nil {
		return err
	}

	cleanup, err := c.setupSSHEnv()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", authURL)
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("repository not accessible: %s", string(exitErr.Stderr))
		}
		return fmt.Errorf("repository not accessible: %w", err)
	}

	return nil
}
