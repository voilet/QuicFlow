package callback

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"
)

// TemplateEngine 模板引擎
type TemplateEngine struct {
	// 自定义函数
	funcMap map[string]TemplateFunc
}

// TemplateFunc 模板函数类型
type TemplateFunc func(args ...interface{}) string

// TemplateContext 模板上下文
type TemplateContext struct {
	// 基础变量
	EventType   string `json:"event_type"`
	Environment string `json:"environment"`
	Timestamp   string `json:"timestamp"`
	Duration    string `json:"duration"`

	// 项目信息
	ProjectID          string `json:"project_id"`
	ProjectName        string `json:"project_name"`
	ProjectDescription string `json:"project_description"`

	// 版本信息
	VersionID          string `json:"version_id"`
	VersionName        string `json:"version_name"`
	VersionDescription string `json:"version_description"`

	// 任务信息
	TaskID       string `json:"task_id"`
	TaskType     string `json:"task_type"`
	TaskStrategy string `json:"task_strategy"`
	TaskStatus   string `json:"task_status"`

	// 部署统计
	TotalCount     int `json:"total_count"`
	CanaryCount    int `json:"canary_count"`
	CompletedCount int `json:"completed_count"`
	FailedCount    int `json:"failed_count"`
	SuccessRate    int `json:"success_rate"` // 百分比

	// 主机列表
	Hosts       []string `json:"hosts"`
	HostsCount  int      `json:"hosts_count"`
	HostsString string   `json:"hosts_string"` // 逗号分隔的主机列表

	// 状态相关
	IsSuccess bool `json:"is_success"`
	IsFailed  bool `json:"is_failed"`
	HasCanary bool `json:"has_canary"`
}

// NewTemplateEngine 创建模板引擎
func NewTemplateEngine() *TemplateEngine {
	engine := &TemplateEngine{
		funcMap: make(map[string]TemplateFunc),
	}
	engine.registerBuiltinFuncs()
	return engine
}

// registerBuiltinFuncs 注册内置函数
func (e *TemplateEngine) registerBuiltinFuncs() {
	// 格式化时间
	e.funcMap["formatTime"] = func(args ...interface{}) string {
		if len(args) == 0 {
			return ""
		}
		if t, ok := args[0].(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
		if s, ok := args[0].(string); ok {
			return s
		}
		return fmt.Sprintf("%v", args[0])
	}

	// 截断字符串
	e.funcMap["truncate"] = func(args ...interface{}) string {
		if len(args) < 2 {
			return ""
		}
		s, ok := args[0].(string)
		if !ok {
			return ""
		}
		maxLen, ok := args[1].(int)
		if !ok {
			return s
		}
		if len(s) <= maxLen {
			return s
		}
		return s[:maxLen] + "..."
	}

	// 大写
	e.funcMap["upper"] = func(args ...interface{}) string {
		if len(args) == 0 {
			return ""
		}
		return strings.ToUpper(fmt.Sprintf("%v", args[0]))
	}

	// 小写
	e.funcMap["lower"] = func(args ...interface{}) string {
		if len(args) == 0 {
			return ""
		}
		return strings.ToLower(fmt.Sprintf("%v", args[0]))
	}

	// 默认值
	e.funcMap["default"] = func(args ...interface{}) string {
		if len(args) < 2 {
			return ""
		}
		val := fmt.Sprintf("%v", args[0])
		if val == "" || val == "0" || val == "false" {
			return fmt.Sprintf("%v", args[1])
		}
		return val
	}

	// 连接字符串
	e.funcMap["join"] = func(args ...interface{}) string {
		if len(args) < 2 {
			return ""
		}
		if arr, ok := args[0].([]string); ok {
			sep := fmt.Sprintf("%v", args[1])
			return strings.Join(arr, sep)
		}
		return ""
	}
}

// RegisterFunc 注册自定义函数
func (e *TemplateEngine) RegisterFunc(name string, fn TemplateFunc) {
	e.funcMap[name] = fn
}

// BuildContext 从 CallbackPayload 构建模板上下文
func (e *TemplateEngine) BuildContext(payload models.CallbackPayload) *TemplateContext {
	ctx := &TemplateContext{
		EventType:   string(payload.EventType),
		Environment: payload.Environment,
		Duration:    payload.Duration,

		ProjectID:          payload.Project.ID,
		ProjectName:        payload.Project.Name,
		ProjectDescription: payload.Project.Description,

		VersionID:          payload.Version.ID,
		VersionName:        payload.Version.Name,
		VersionDescription: payload.Version.Description,

		TaskID:       payload.Task.ID,
		TaskType:     string(payload.Task.Type),
		TaskStrategy: string(payload.Task.Strategy),
		TaskStatus:   payload.Task.Status,

		TotalCount:     payload.Deployment.TotalCount,
		CanaryCount:    payload.Deployment.CanaryCount,
		CompletedCount: payload.Deployment.CompletedCount,
		FailedCount:    payload.Deployment.FailedCount,

		Hosts:      payload.Deployment.Hosts,
		HostsCount: len(payload.Deployment.Hosts),
	}

	// 格式化时间戳
	if !payload.Timestamp.IsZero() {
		ctx.Timestamp = payload.Timestamp.Format("2006-01-02 15:04:05")
	}

	// 计算成功率
	if ctx.TotalCount > 0 {
		ctx.SuccessRate = ctx.CompletedCount * 100 / ctx.TotalCount
	}

	// 主机列表字符串
	if len(ctx.Hosts) > 0 {
		ctx.HostsString = strings.Join(ctx.Hosts, ", ")
	}

	// 状态标志
	ctx.IsSuccess = ctx.FailedCount == 0 && ctx.CompletedCount == ctx.TotalCount
	ctx.IsFailed = ctx.FailedCount > 0
	ctx.HasCanary = ctx.CanaryCount > 0

	return ctx
}

// Render 渲染模板
func (e *TemplateEngine) Render(template string, payload models.CallbackPayload) (string, error) {
	ctx := e.BuildContext(payload)
	return e.RenderWithContext(template, ctx)
}

// RenderWithContext 使用上下文渲染模板
func (e *TemplateEngine) RenderWithContext(template string, ctx *TemplateContext) (string, error) {
	result := template

	// 1. 处理条件块 {{#if condition}}...{{/if}}
	result = e.processConditions(result, ctx)

	// 2. 处理循环块 {{#each array}}...{{/each}}
	result = e.processLoops(result, ctx)

	// 3. 处理简单变量替换 {{variable}}
	result = e.processVariables(result, ctx)

	return result, nil
}

// processConditions 处理条件块
func (e *TemplateEngine) processConditions(template string, ctx *TemplateContext) string {
	// 匹配 {{#if condition}}...{{/if}}
	// 支持 {{#if condition}}...{{else}}...{{/if}}
	ifRegex := regexp.MustCompile(`\{\{#if\s+(\w+)\}\}([\s\S]*?)\{\{/if\}\}`)

	return ifRegex.ReplaceAllStringFunc(template, func(match string) string {
		submatch := ifRegex.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}

		condition := submatch[1]
		content := submatch[2]

		// 检查是否有 else 块
		elseRegex := regexp.MustCompile(`\{\{else\}\}`)
		parts := elseRegex.Split(content, 2)
		truePart := parts[0]
		falsePart := ""
		if len(parts) > 1 {
			falsePart = parts[1]
		}

		// 评估条件
		if e.evaluateCondition(condition, ctx) {
			return truePart
		}
		return falsePart
	})
}

// evaluateCondition 评估条件
func (e *TemplateEngine) evaluateCondition(condition string, ctx *TemplateContext) bool {
	switch condition {
	case "is_success":
		return ctx.IsSuccess
	case "is_failed":
		return ctx.IsFailed
	case "has_canary":
		return ctx.HasCanary
	case "has_hosts":
		return ctx.HostsCount > 0
	case "has_failed":
		return ctx.FailedCount > 0
	case "has_duration":
		return ctx.Duration != ""
	default:
		// 尝试获取布尔值
		val := e.getContextValue(condition, ctx)
		switch v := val.(type) {
		case bool:
			return v
		case int:
			return v != 0
		case string:
			return v != ""
		}
		return false
	}
}

// processLoops 处理循环块
func (e *TemplateEngine) processLoops(template string, ctx *TemplateContext) string {
	// 匹配 {{#each array}}...{{/each}}
	eachRegex := regexp.MustCompile(`\{\{#each\s+(\w+)\}\}([\s\S]*?)\{\{/each\}\}`)

	return eachRegex.ReplaceAllStringFunc(template, func(match string) string {
		submatch := eachRegex.FindStringSubmatch(match)
		if len(submatch) < 3 {
			return match
		}

		arrayName := submatch[1]
		itemTemplate := submatch[2]

		// 获取数组
		var items []string
		switch arrayName {
		case "hosts":
			items = ctx.Hosts
		default:
			return ""
		}

		// 渲染每个元素
		var results []string
		for i, item := range items {
			rendered := itemTemplate
			rendered = strings.ReplaceAll(rendered, "{{this}}", item)
			rendered = strings.ReplaceAll(rendered, "{{@index}}", strconv.Itoa(i))
			rendered = strings.ReplaceAll(rendered, "{{@first}}", strconv.FormatBool(i == 0))
			rendered = strings.ReplaceAll(rendered, "{{@last}}", strconv.FormatBool(i == len(items)-1))
			results = append(results, rendered)
		}

		return strings.Join(results, "")
	})
}

// processVariables 处理变量替换
func (e *TemplateEngine) processVariables(template string, ctx *TemplateContext) string {
	// 匹配 {{variable}} 或 {{variable | filter}}
	varRegex := regexp.MustCompile(`\{\{(\w+)(?:\s*\|\s*(\w+))?\}\}`)

	return varRegex.ReplaceAllStringFunc(template, func(match string) string {
		submatch := varRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}

		varName := submatch[1]
		filter := ""
		if len(submatch) > 2 {
			filter = submatch[2]
		}

		value := e.getContextValue(varName, ctx)
		result := fmt.Sprintf("%v", value)

		// 应用过滤器
		if filter != "" {
			if fn, ok := e.funcMap[filter]; ok {
				result = fn(value)
			}
		}

		return result
	})
}

// getContextValue 获取上下文值
func (e *TemplateEngine) getContextValue(name string, ctx *TemplateContext) interface{} {
	switch name {
	case "event_type":
		return ctx.EventType
	case "environment":
		return ctx.Environment
	case "timestamp":
		return ctx.Timestamp
	case "duration":
		return ctx.Duration
	case "project_id":
		return ctx.ProjectID
	case "project_name":
		return ctx.ProjectName
	case "project_description":
		return ctx.ProjectDescription
	case "version_id":
		return ctx.VersionID
	case "version_name":
		return ctx.VersionName
	case "version_description":
		return ctx.VersionDescription
	case "task_id":
		return ctx.TaskID
	case "task_type":
		return ctx.TaskType
	case "task_strategy":
		return ctx.TaskStrategy
	case "task_status":
		return ctx.TaskStatus
	case "total_count":
		return ctx.TotalCount
	case "canary_count":
		return ctx.CanaryCount
	case "completed_count":
		return ctx.CompletedCount
	case "failed_count":
		return ctx.FailedCount
	case "success_rate":
		return ctx.SuccessRate
	case "hosts_count":
		return ctx.HostsCount
	case "hosts_string":
		return ctx.HostsString
	case "is_success":
		return ctx.IsSuccess
	case "is_failed":
		return ctx.IsFailed
	case "has_canary":
		return ctx.HasCanary
	default:
		return ""
	}
}

// GetAvailableVariables 获取所有可用的模板变量
func (e *TemplateEngine) GetAvailableVariables() []TemplateVariable {
	return []TemplateVariable{
		// 基础信息
		{Name: "event_type", Description: "事件类型 (canary_started, canary_completed, full_completed)", Example: "canary_completed"},
		{Name: "environment", Description: "环境名称", Example: "production"},
		{Name: "timestamp", Description: "事件时间", Example: "2024-01-15 14:30:00"},
		{Name: "duration", Description: "执行耗时", Example: "5m30s"},

		// 项目信息
		{Name: "project_id", Description: "项目 ID", Example: "proj-001"},
		{Name: "project_name", Description: "项目名称", Example: "My Project"},
		{Name: "project_description", Description: "项目描述", Example: "A sample project"},

		// 版本信息
		{Name: "version_id", Description: "版本 ID", Example: "ver-001"},
		{Name: "version_name", Description: "版本号", Example: "v1.2.3"},
		{Name: "version_description", Description: "版本描述", Example: "Bug fixes"},

		// 任务信息
		{Name: "task_id", Description: "任务 ID", Example: "task-001"},
		{Name: "task_type", Description: "任务类型", Example: "deploy"},
		{Name: "task_strategy", Description: "发布策略", Example: "canary"},
		{Name: "task_status", Description: "任务状态", Example: "success"},

		// 部署统计
		{Name: "total_count", Description: "总主机数", Example: "10"},
		{Name: "canary_count", Description: "金丝雀数量", Example: "2"},
		{Name: "completed_count", Description: "已完成数量", Example: "10"},
		{Name: "failed_count", Description: "失败数量", Example: "0"},
		{Name: "success_rate", Description: "成功率（百分比）", Example: "100"},

		// 主机相关
		{Name: "hosts_count", Description: "主机总数", Example: "10"},
		{Name: "hosts_string", Description: "主机列表（逗号分隔）", Example: "host1, host2, host3"},

		// 条件变量
		{Name: "is_success", Description: "是否全部成功", Example: "true"},
		{Name: "is_failed", Description: "是否有失败", Example: "false"},
		{Name: "has_canary", Description: "是否有金丝雀", Example: "true"},
	}
}

// TemplateVariable 模板变量定义
type TemplateVariable struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
}

// GetConditionExamples 获取条件语法示例
func (e *TemplateEngine) GetConditionExamples() []TemplateExample {
	return []TemplateExample{
		{
			Name:        "条件渲染 - 成功",
			Description: "仅在成功时显示内容",
			Template:    "{{#if is_success}}部署成功！{{/if}}",
			Result:      "部署成功！",
		},
		{
			Name:        "条件渲染 - 带 else",
			Description: "成功或失败显示不同内容",
			Template:    "{{#if is_success}}✅ 成功{{else}}❌ 失败{{/if}}",
			Result:      "✅ 成功",
		},
		{
			Name:        "条件渲染 - 失败数量",
			Description: "有失败时显示失败数量",
			Template:    "{{#if has_failed}}失败: {{failed_count}}{{/if}}",
			Result:      "失败: 2",
		},
		{
			Name:        "循环渲染 - 主机列表",
			Description: "循环显示每个主机",
			Template:    "{{#each hosts}}- {{this}}\n{{/each}}",
			Result:      "- host1\n- host2\n",
		},
	}
}

// TemplateExample 模板示例
type TemplateExample struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Template    string `json:"template"`
	Result      string `json:"result"`
}

// Preview 预览模板渲染结果
func (e *TemplateEngine) Preview(template string) (string, error) {
	// 使用示例数据进行预览
	samplePayload := models.CallbackPayload{
		EventType:   models.CallbackEventFullCompleted,
		Environment: "production",
		Timestamp:   time.Now(),
		Duration:    "2m30s",
		Project: models.CallbackProject{
			ID:          "sample-project-id",
			Name:        "示例项目",
			Description: "这是一个示例项目",
		},
		Version: models.CallbackVersion{
			ID:          "sample-version-id",
			Name:        "v1.0.0",
			Description: "示例版本",
		},
		Task: models.CallbackTask{
			ID:       "sample-task-id",
			Type:     models.OperationTypeDeploy,
			Strategy: models.StrategyTypeCanary,
			Status:   "success",
		},
		Deployment: models.CallbackDeployment{
			TotalCount:     10,
			CanaryCount:    2,
			CompletedCount: 10,
			FailedCount:    0,
			Hosts:          []string{"host-1", "host-2", "host-3"},
		},
	}

	return e.Render(template, samplePayload)
}

// PreviewWithPayload 使用自定义 payload 预览模板
func (e *TemplateEngine) PreviewWithPayload(template string, payload models.CallbackPayload) (string, error) {
	return e.Render(template, payload)
}

// ValidateTemplate 验证模板语法
func (e *TemplateEngine) ValidateTemplate(template string) *TemplateValidationResult {
	result := &TemplateValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 检查未闭合的条件块
	ifCount := strings.Count(template, "{{#if")
	endifCount := strings.Count(template, "{{/if}}")
	if ifCount != endifCount {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("条件块未正确闭合: %d 个 {{#if}} 但有 %d 个 {{/if}}", ifCount, endifCount))
	}

	// 检查未闭合的循环块
	eachCount := strings.Count(template, "{{#each")
	endEachCount := strings.Count(template, "{{/each}}")
	if eachCount != endEachCount {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("循环块未正确闭合: %d 个 {{#each}} 但有 %d 个 {{/each}}", eachCount, endEachCount))
	}

	// 检查无效的变量名
	varRegex := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := varRegex.FindAllStringSubmatch(template, -1)
	validVars := e.getValidVariableNames()

	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			// 跳过特殊变量
			if varName == "this" || varName == "else" {
				continue
			}
			if !validVars[varName] {
				result.Warnings = append(result.Warnings, fmt.Sprintf("未知变量: %s", varName))
			}
		}
	}

	// 尝试渲染预览
	if result.Valid {
		_, err := e.Preview(template)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("渲染失败: %s", err.Error()))
		}
	}

	return result
}

// getValidVariableNames 获取有效变量名集合
func (e *TemplateEngine) getValidVariableNames() map[string]bool {
	vars := e.GetAvailableVariables()
	result := make(map[string]bool)
	for _, v := range vars {
		result[v.Name] = true
	}
	// 添加循环内置变量
	result["@index"] = true
	result["@first"] = true
	result["@last"] = true
	return result
}

// TemplateValidationResult 模板验证结果
type TemplateValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// GetDefaultTemplates 获取默认模板
func (e *TemplateEngine) GetDefaultTemplates() map[string]string {
	return map[string]string{
		"simple": `项目: {{project_name}}
版本: {{version_name}}
状态: {{task_status}}
{{#if is_success}}✅ 部署成功！{{else}}❌ 部署失败{{/if}}`,

		"detailed": `## 发布通知

**项目**: {{project_name}}
**版本**: {{version_name}}
**环境**: {{environment}}
**状态**: {{task_status}}

### 部署统计
- 总数: {{total_count}}
- 完成: {{completed_count}}
{{#if has_failed}}- 失败: {{failed_count}}{{/if}}
{{#if has_canary}}- 金丝雀: {{canary_count}}{{/if}}

{{#if has_hosts}}### 主机列表
{{#each hosts}}- {{this}}
{{/each}}{{/if}}

---
时间: {{timestamp}}`,

		"json": `{
  "event": "{{event_type}}",
  "project": "{{project_name}}",
  "version": "{{version_name}}",
  "status": "{{task_status}}",
  "stats": {
    "total": {{total_count}},
    "completed": {{completed_count}},
    "failed": {{failed_count}}
  },
  "timestamp": "{{timestamp}}"
}`,
	}
}

// RenderToJSON 渲染模板并解析为 JSON
func (e *TemplateEngine) RenderToJSON(template string, payload models.CallbackPayload) (map[string]interface{}, error) {
	rendered, err := e.Render(template, payload)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(rendered), &result); err != nil {
		// 如果不是有效 JSON，包装成消息对象
		return map[string]interface{}{"message": rendered}, nil
	}

	return result, nil
}
