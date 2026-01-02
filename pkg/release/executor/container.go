package executor

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"
)

// ContainerNameGenerator 容器名称生成器
type ContainerNameGenerator struct {
	config *models.ContainerNamingConfig
}

// NewContainerNameGenerator 创建容器名称生成器
func NewContainerNameGenerator(config *models.ContainerNamingConfig) *ContainerNameGenerator {
	return &ContainerNameGenerator{config: config}
}

// GenerateContext 名称生成上下文
type GenerateContext struct {
	Prefix    string
	Env       string
	Version   string
	Timestamp time.Time
	Index     int
}

// Generate 生成容器名称
func (g *ContainerNameGenerator) Generate(ctx GenerateContext) string {
	if g.config == nil {
		// 使用默认命名规则
		return g.generateDefault(ctx)
	}

	// 使用模板生成
	if g.config.Template != "" {
		return g.generateFromTemplate(ctx)
	}

	// 使用配置生成
	return g.generateFromConfig(ctx)
}

// generateDefault 默认命名规则
func (g *ContainerNameGenerator) generateDefault(ctx GenerateContext) string {
	parts := []string{}

	if ctx.Prefix != "" {
		parts = append(parts, ctx.Prefix)
	}

	if ctx.Env != "" {
		parts = append(parts, ctx.Env)
	}

	if ctx.Index > 0 {
		parts = append(parts, fmt.Sprintf("%d", ctx.Index))
	}

	name := strings.Join(parts, "-")
	return g.sanitizeName(name, 63)
}

// generateFromConfig 根据配置生成
func (g *ContainerNameGenerator) generateFromConfig(ctx GenerateContext) string {
	parts := []string{}

	// 使用配置的前缀，或上下文的前缀
	prefix := g.config.Prefix
	if prefix == "" {
		prefix = ctx.Prefix
	}
	if prefix != "" {
		parts = append(parts, prefix)
	}

	// 包含环境名
	if g.config.IncludeEnv && ctx.Env != "" {
		parts = append(parts, ctx.Env)
	}

	// 包含版本号
	if g.config.IncludeVer && ctx.Version != "" {
		// 清理版本号（去除不允许的字符）
		version := g.cleanVersion(ctx.Version)
		parts = append(parts, version)
	}

	// 添加索引
	if ctx.Index > 0 {
		parts = append(parts, fmt.Sprintf("%d", ctx.Index))
	}

	// 使用分隔符连接
	separator := g.config.Separator
	if separator == "" {
		separator = "-"
	}

	name := strings.Join(parts, separator)

	// 限制长度
	maxLength := g.config.MaxLength
	if maxLength <= 0 {
		maxLength = 63 // Docker 容器名称最大长度
	}

	return g.sanitizeName(name, maxLength)
}

// generateFromTemplate 根据模板生成
func (g *ContainerNameGenerator) generateFromTemplate(ctx GenerateContext) string {
	name := g.config.Template

	// 替换变量
	replacements := map[string]string{
		"${PREFIX}":    ctx.Prefix,
		"${ENV}":       ctx.Env,
		"${VERSION}":   g.cleanVersion(ctx.Version),
		"${TIMESTAMP}": ctx.Timestamp.Format("20060102150405"),
		"${INDEX}":     fmt.Sprintf("%d", ctx.Index),
		"${DATE}":      ctx.Timestamp.Format("20060102"),
		"${TIME}":      ctx.Timestamp.Format("150405"),
	}

	for k, v := range replacements {
		name = strings.ReplaceAll(name, k, v)
	}

	// 限制长度
	maxLength := g.config.MaxLength
	if maxLength <= 0 {
		maxLength = 63
	}

	return g.sanitizeName(name, maxLength)
}

// cleanVersion 清理版本号
func (g *ContainerNameGenerator) cleanVersion(version string) string {
	// 去除 v 前缀
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")

	// 替换不允许的字符
	version = strings.ReplaceAll(version, "+", "-")
	version = strings.ReplaceAll(version, "_", "-")

	return version
}

// sanitizeName 清理名称，确保符合 Docker 命名规则
func (g *ContainerNameGenerator) sanitizeName(name string, maxLength int) string {
	// Docker 容器名称规则：
	// - 只能包含字母、数字、下划线、点和连字符
	// - 必须以字母或数字开头
	// - 不能以连字符或下划线结尾

	// 替换不允许的字符
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	name = re.ReplaceAllString(name, "-")

	// 合并连续的连字符
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")

	// 确保以字母或数字开头
	name = strings.TrimLeft(name, "-_.")

	// 确保不以连字符或下划线结尾
	name = strings.TrimRight(name, "-_")

	// 限制长度
	if len(name) > maxLength {
		name = name[:maxLength]
		// 再次清理末尾
		name = strings.TrimRight(name, "-_")
	}

	// 如果名称为空，生成默认名称
	if name == "" {
		name = fmt.Sprintf("container-%d", time.Now().UnixNano())
	}

	return name
}

// ValidateName 验证容器名称
func (g *ContainerNameGenerator) ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("container name cannot be empty")
	}

	if len(name) > 63 {
		return fmt.Errorf("container name too long (max 63 characters)")
	}

	// 检查是否以字母或数字开头
	if !regexp.MustCompile(`^[a-zA-Z0-9]`).MatchString(name) {
		return fmt.Errorf("container name must start with a letter or number")
	}

	// 检查是否只包含允许的字符
	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`).MatchString(name) {
		return fmt.Errorf("container name contains invalid characters")
	}

	// 检查是否以连字符或下划线结尾
	if regexp.MustCompile(`[-_]$`).MatchString(name) {
		return fmt.Errorf("container name cannot end with hyphen or underscore")
	}

	return nil
}

// GenerateUniqueName 生成唯一名称（检查是否已存在）
func (g *ContainerNameGenerator) GenerateUniqueName(ctx GenerateContext, existingNames []string) string {
	baseName := g.Generate(ctx)

	// 检查是否已存在
	nameSet := make(map[string]bool)
	for _, n := range existingNames {
		nameSet[n] = true
	}

	if !nameSet[baseName] {
		return baseName
	}

	// 添加后缀
	for i := 1; i <= 100; i++ {
		name := fmt.Sprintf("%s-%d", baseName, i)
		if len(name) > 63 {
			// 缩短基础名称
			maxBase := 63 - len(fmt.Sprintf("-%d", i))
			name = fmt.Sprintf("%s-%d", baseName[:maxBase], i)
		}
		if !nameSet[name] {
			return name
		}
	}

	// 使用时间戳
	return fmt.Sprintf("%s-%d", baseName[:min(50, len(baseName))], time.Now().UnixNano())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DefaultNamingConfig 默认命名配置
func DefaultNamingConfig() *models.ContainerNamingConfig {
	return &models.ContainerNamingConfig{
		Separator:  "-",
		IncludeEnv: true,
		IncludeVer: false,
		MaxLength:  63,
		Template:   "${PREFIX}-${ENV}-${INDEX}",
	}
}

// K8sNamingConfig K8s 命名配置
func K8sNamingConfig() *models.ContainerNamingConfig {
	return &models.ContainerNamingConfig{
		Separator:  "-",
		IncludeEnv: true,
		IncludeVer: false,
		MaxLength:  63,
		Template:   "${PREFIX}-${ENV}",
	}
}
