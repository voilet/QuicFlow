package variable

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"

	"gorm.io/gorm"
)

// Manager 变量管理器
type Manager struct {
	db *gorm.DB
	mu sync.RWMutex

	// 系统变量缓存
	systemVars map[string]string
}

// NewManager 创建变量管理器
func NewManager(db *gorm.DB) *Manager {
	m := &Manager{
		db:         db,
		systemVars: make(map[string]string),
	}
	m.initSystemVars()
	return m
}

// initSystemVars 初始化系统变量
func (m *Manager) initSystemVars() {
	hostname, _ := os.Hostname()
	m.systemVars["HOSTNAME"] = hostname
}

// Context 变量解析上下文
type Context struct {
	// 发布相关
	ReleaseID        string
	ReleaseVersion   string
	ReleaseEnv       string
	ReleaseUser      string
	ReleaseTime      time.Time

	// 目标相关
	TargetID       string
	TargetName     string
	TargetHost     string
	TargetIP       string
	TargetClientID string
	TargetLabels   map[string]string

	// Git相关
	GitRepo        string
	GitBranch      string
	GitCommit      string
	GitCommitShort string
	GitTag         string
	GitMessage     string

	// 容器相关
	ImageRegistry  string
	ImageName      string
	ImageTag       string
	ContainerName  string

	// K8s相关
	K8sCluster    string
	K8sNamespace  string
	K8sDeployment string
	K8sReplicas   int
	K8sContext    string

	// 路径相关
	AppDir    string
	BackupDir string
	LogDir    string
	TempDir   string

	// 操作相关
	CurrentVersion  string
	RollbackVersion string
	ArtifactURL     string
	KeepData        bool

	// 自定义变量
	Custom map[string]string
}

// Resolve 解析变量
func (m *Manager) Resolve(ctx context.Context, text string, varCtx *Context) (string, error) {
	if varCtx == nil {
		varCtx = &Context{}
	}

	// 构建变量映射
	vars := m.buildVarMap(varCtx)

	// 替换变量
	result := m.replaceVars(text, vars)

	return result, nil
}

// ResolveMap 解析变量映射
func (m *Manager) ResolveMap(ctx context.Context, data map[string]string, varCtx *Context) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range data {
		resolved, err := m.Resolve(ctx, v, varCtx)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", k, err)
		}
		result[k] = resolved
	}
	return result, nil
}

// buildVarMap 构建变量映射
func (m *Manager) buildVarMap(ctx *Context) map[string]string {
	vars := make(map[string]string)

	// 系统变量
	for k, v := range m.systemVars {
		vars[k] = v
	}

	// 发布相关
	vars["RELEASE_ID"] = ctx.ReleaseID
	vars["RELEASE_VERSION"] = ctx.ReleaseVersion
	vars["RELEASE_ENV"] = ctx.ReleaseEnv
	vars["RELEASE_USER"] = ctx.ReleaseUser
	if !ctx.ReleaseTime.IsZero() {
		vars["RELEASE_TIME"] = ctx.ReleaseTime.Format(time.RFC3339)
		vars["RELEASE_TIMESTAMP"] = fmt.Sprintf("%d", ctx.ReleaseTime.Unix())
	}

	// 目标相关
	vars["TARGET_ID"] = ctx.TargetID
	vars["TARGET_NAME"] = ctx.TargetName
	vars["TARGET_HOST"] = ctx.TargetHost
	vars["TARGET_IP"] = ctx.TargetIP
	vars["TARGET_CLIENT_ID"] = ctx.TargetClientID

	// Git相关
	vars["GIT_REPO"] = ctx.GitRepo
	vars["GIT_BRANCH"] = ctx.GitBranch
	vars["GIT_COMMIT"] = ctx.GitCommit
	vars["GIT_COMMIT_SHORT"] = ctx.GitCommitShort
	vars["GIT_TAG"] = ctx.GitTag
	vars["GIT_MESSAGE"] = ctx.GitMessage

	// 容器相关
	vars["IMAGE_REGISTRY"] = ctx.ImageRegistry
	vars["IMAGE_NAME"] = ctx.ImageName
	vars["IMAGE_TAG"] = ctx.ImageTag
	if ctx.ImageRegistry != "" && ctx.ImageName != "" && ctx.ImageTag != "" {
		vars["IMAGE_FULL"] = fmt.Sprintf("%s/%s:%s", ctx.ImageRegistry, ctx.ImageName, ctx.ImageTag)
	}
	vars["CONTAINER_NAME"] = ctx.ContainerName

	// K8s相关
	vars["K8S_CLUSTER"] = ctx.K8sCluster
	vars["K8S_NAMESPACE"] = ctx.K8sNamespace
	vars["K8S_DEPLOYMENT"] = ctx.K8sDeployment
	vars["K8S_REPLICAS"] = fmt.Sprintf("%d", ctx.K8sReplicas)
	vars["K8S_CONTEXT"] = ctx.K8sContext

	// 路径相关
	vars["APP_DIR"] = ctx.AppDir
	vars["BACKUP_DIR"] = ctx.BackupDir
	vars["LOG_DIR"] = ctx.LogDir
	vars["TEMP_DIR"] = ctx.TempDir
	vars["WORK_DIR"] = ctx.AppDir // 别名

	// 操作相关
	vars["CURRENT_VERSION"] = ctx.CurrentVersion
	vars["ROLLBACK_VERSION"] = ctx.RollbackVersion
	vars["ARTIFACT_URL"] = ctx.ArtifactURL
	if ctx.KeepData {
		vars["KEEP_DATA"] = "true"
	} else {
		vars["KEEP_DATA"] = "false"
	}

	// 自定义变量
	for k, v := range ctx.Custom {
		vars[k] = v
	}

	return vars
}

// replaceVars 替换变量
func (m *Manager) replaceVars(text string, vars map[string]string) string {
	// 匹配 ${VAR} 或 $VAR
	re := regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

	return re.ReplaceAllStringFunc(text, func(match string) string {
		var varName string
		if strings.HasPrefix(match, "${") {
			varName = match[2 : len(match)-1]
		} else {
			varName = match[1:]
		}

		if val, ok := vars[varName]; ok {
			return val
		}
		// 未找到变量，保持原样
		return match
	})
}

// LoadProjectVariables 加载项目变量
func (m *Manager) LoadProjectVariables(ctx context.Context, projectID string) (map[string]string, error) {
	var vars []models.Variable
	if err := m.db.WithContext(ctx).
		Where("project_id = ? AND environment_id IS NULL", projectID).
		Find(&vars).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, v := range vars {
		if v.Type == models.VariableTypeSecret {
			// TODO: 解密密钥变量
			result[v.Name] = v.Value
		} else {
			result[v.Name] = v.Value
		}
	}

	return result, nil
}

// LoadEnvironmentVariables 加载环境变量
func (m *Manager) LoadEnvironmentVariables(ctx context.Context, envID string, envName string) (map[string]string, error) {
	var vars []models.Variable
	if err := m.db.WithContext(ctx).
		Where("environment_id = ?", envID).
		Find(&vars).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, v := range vars {
		switch v.Type {
		case models.VariableTypeSecret:
			// TODO: 解密
			result[v.Name] = v.Value
		case models.VariableTypeEnvSpecific:
			if val, ok := v.EnvValues[envName]; ok {
				result[v.Name] = val
			}
		default:
			result[v.Name] = v.Value
		}
	}

	return result, nil
}

// MergeVariables 合并变量（按优先级）
// 优先级: 系统变量 < 项目变量 < 环境变量 < 发布变量 < 运行时变量
func (m *Manager) MergeVariables(
	systemVars map[string]string,
	projectVars map[string]string,
	envVars map[string]string,
	releaseVars map[string]string,
	runtimeVars map[string]string,
) map[string]string {
	result := make(map[string]string)

	// 按优先级从低到高合并
	for k, v := range systemVars {
		result[k] = v
	}
	for k, v := range projectVars {
		result[k] = v
	}
	for k, v := range envVars {
		result[k] = v
	}
	for k, v := range releaseVars {
		result[k] = v
	}
	for k, v := range runtimeVars {
		result[k] = v
	}

	return result
}

// CreateVariable 创建变量
func (m *Manager) CreateVariable(ctx context.Context, v *models.Variable) error {
	return m.db.WithContext(ctx).Create(v).Error
}

// UpdateVariable 更新变量
func (m *Manager) UpdateVariable(ctx context.Context, v *models.Variable) error {
	return m.db.WithContext(ctx).Save(v).Error
}

// DeleteVariable 删除变量
func (m *Manager) DeleteVariable(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).Delete(&models.Variable{}, "id = ?", id).Error
}

// GetVariable 获取变量
func (m *Manager) GetVariable(ctx context.Context, id string) (*models.Variable, error) {
	var v models.Variable
	if err := m.db.WithContext(ctx).First(&v, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}
