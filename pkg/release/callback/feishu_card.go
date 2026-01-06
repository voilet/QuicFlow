package callback

import (
	"fmt"
	"strings"

	"github.com/voilet/quic-flow/pkg/release/models"
)

// FeishuCardBuilder 飞书卡片构建器
type FeishuCardBuilder struct {
	payload models.CallbackPayload
}

// NewFeishuCardBuilder 创建飞书卡片构建器
func NewFeishuCardBuilder(payload models.CallbackPayload) *FeishuCardBuilder {
	return &FeishuCardBuilder{payload: payload}
}

// BuildCard 构建飞书卡片
func (b *FeishuCardBuilder) BuildCard() map[string]interface{} {
	header := b.buildHeader()
	elements := b.buildElements()

	return map[string]interface{}{
		"header":   header,
		"elements": elements,
	}
}

// buildHeader 构建卡片头部
func (b *FeishuCardBuilder) buildHeader() map[string]interface{} {
	title, template, icon := b.getTitleAndTemplate()

	return map[string]interface{}{
		"title": map[string]interface{}{
			"tag":     "plain_text",
			"content": fmt.Sprintf("%s %s", icon, title),
		},
		"template": template,
	}
}

// getTitleAndTemplate 根据事件类型获取标题和模板颜色
func (b *FeishuCardBuilder) getTitleAndTemplate() (title, template, icon string) {
	switch b.payload.EventType {
	case models.CallbackEventCanaryStarted:
		return "金丝雀发布开始", "blue", "🚀"
	case models.CallbackEventCanaryCompleted:
		if b.payload.Deployment.FailedCount > 0 {
			return "金丝雀发布完成（有失败）", "orange", "⚠️"
		}
		return "金丝雀发布完成", "green", "✅"
	case models.CallbackEventFullCompleted:
		if b.payload.Deployment.FailedCount > 0 {
			return "全量发布完成（有失败）", "red", "❌"
		}
		return "全量发布完成", "green", "🎉"
	default:
		return "发布通知", "blue", "📢"
	}
}

// buildElements 构建卡片元素
func (b *FeishuCardBuilder) buildElements() []map[string]interface{} {
	elements := []map[string]interface{}{}

	// 1. 基本信息区域
	elements = append(elements, b.buildInfoSection())

	// 2. 分割线
	elements = append(elements, map[string]interface{}{"tag": "hr"})

	// 3. 部署统计区域
	elements = append(elements, b.buildStatsSection())

	// 4. 主机列表（如果有）
	if len(b.payload.Deployment.Hosts) > 0 {
		elements = append(elements, map[string]interface{}{"tag": "hr"})
		elements = append(elements, b.buildHostsSection())
	}

	// 5. 失败信息（如果有）
	if b.payload.Deployment.FailedCount > 0 {
		elements = append(elements, map[string]interface{}{"tag": "hr"})
		elements = append(elements, b.buildFailureSection())
	}

	// 6. 时间戳
	elements = append(elements, b.buildTimestampSection())

	return elements
}

// buildInfoSection 构建基本信息区域
func (b *FeishuCardBuilder) buildInfoSection() map[string]interface{} {
	statusIcon, statusText := b.getStatusIconAndText()

	fields := []map[string]interface{}{
		{
			"is_short": true,
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**📦 项目**\n%s", b.payload.Project.Name),
			},
		},
		{
			"is_short": true,
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**🏷️ 版本**\n%s", b.payload.Version.Name),
			},
		},
		{
			"is_short": true,
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**🌍 环境**\n%s", b.getEnvironmentDisplay()),
			},
		},
		{
			"is_short": true,
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**%s 状态**\n%s", statusIcon, statusText),
			},
		},
	}

	// 添加操作类型
	if b.payload.Task.Type != "" {
		fields = append(fields, map[string]interface{}{
			"is_short": true,
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**⚙️ 操作**\n%s", b.getOperationDisplay()),
			},
		})
	}

	// 添加策略（如果是金丝雀）
	if b.payload.Deployment.CanaryCount > 0 {
		fields = append(fields, map[string]interface{}{
			"is_short": true,
			"text": map[string]interface{}{
				"tag":     "lark_md",
				"content": fmt.Sprintf("**🐦 策略**\n金丝雀发布"),
			},
		})
	}

	return map[string]interface{}{
		"tag":    "div",
		"fields": fields,
	}
}

// getStatusIconAndText 获取状态图标和文本
func (b *FeishuCardBuilder) getStatusIconAndText() (icon, text string) {
	status := b.payload.Task.Status
	switch status {
	case "success", "completed":
		return "✅", "成功"
	case "failed":
		return "❌", "失败"
	case "running":
		return "🔄", "进行中"
	case "pending":
		return "⏳", "等待中"
	case "cancelled":
		return "🚫", "已取消"
	default:
		return "📋", status
	}
}

// getEnvironmentDisplay 获取环境显示名称
func (b *FeishuCardBuilder) getEnvironmentDisplay() string {
	env := b.payload.Environment
	switch env {
	case "production", "prod":
		return "🔴 生产环境"
	case "staging", "stage":
		return "🟡 预发环境"
	case "test", "testing":
		return "🟢 测试环境"
	case "development", "dev":
		return "🔵 开发环境"
	default:
		if env == "" {
			return "未指定"
		}
		return env
	}
}

// getOperationDisplay 获取操作类型显示
func (b *FeishuCardBuilder) getOperationDisplay() string {
	switch b.payload.Task.Type {
	case models.OperationTypeDeploy:
		return "部署"
	case models.OperationTypeInstall:
		return "安装"
	case models.OperationTypeUpdate:
		return "更新"
	case models.OperationTypeRollback:
		return "回滚"
	case models.OperationTypeUninstall:
		return "卸载"
	default:
		return string(b.payload.Task.Type)
	}
}

// buildStatsSection 构建部署统计区域
func (b *FeishuCardBuilder) buildStatsSection() map[string]interface{} {
	d := b.payload.Deployment

	// 计算成功率
	successRate := 0
	if d.TotalCount > 0 {
		successRate = (d.CompletedCount - d.FailedCount) * 100 / d.TotalCount
	}

	// 构建进度条样式的统计
	var statsContent strings.Builder
	statsContent.WriteString("**📊 部署统计**\n\n")

	// 总数和完成数
	statsContent.WriteString(fmt.Sprintf("• 总目标数: **%d**\n", d.TotalCount))

	if d.CanaryCount > 0 {
		statsContent.WriteString(fmt.Sprintf("• 金丝雀数: **%d** (%.0f%%)\n", d.CanaryCount, float64(d.CanaryCount)*100/float64(d.TotalCount)))
	}

	statsContent.WriteString(fmt.Sprintf("• 已完成: **%d**\n", d.CompletedCount))

	if d.FailedCount > 0 {
		statsContent.WriteString(fmt.Sprintf("• <font color='red'>失败: **%d**</font>\n", d.FailedCount))
	}

	// 成功率显示
	if d.TotalCount > 0 {
		rateColor := "green"
		if successRate < 100 && successRate >= 80 {
			rateColor = "orange"
		} else if successRate < 80 {
			rateColor = "red"
		}
		statsContent.WriteString(fmt.Sprintf("\n**成功率**: <font color='%s'>%d%%</font>", rateColor, successRate))
	}

	// 耗时
	if b.payload.Duration != "" {
		statsContent.WriteString(fmt.Sprintf("\n**耗时**: %s", b.payload.Duration))
	}

	return map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"tag":     "lark_md",
			"content": statsContent.String(),
		},
	}
}

// buildHostsSection 构建主机列表区域
func (b *FeishuCardBuilder) buildHostsSection() map[string]interface{} {
	hosts := b.payload.Deployment.Hosts
	maxDisplay := 10

	var content strings.Builder
	content.WriteString(fmt.Sprintf("**🖥️ 目标主机** (%d 台)\n\n", len(hosts)))

	// 显示主机列表
	displayCount := len(hosts)
	if displayCount > maxDisplay {
		displayCount = maxDisplay
	}

	for i := 0; i < displayCount; i++ {
		content.WriteString(fmt.Sprintf("`%s`", hosts[i]))
		if i < displayCount-1 {
			content.WriteString(" ")
		}
	}

	if len(hosts) > maxDisplay {
		content.WriteString(fmt.Sprintf("\n... 等 %d 台主机", len(hosts)-maxDisplay))
	}

	return map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"tag":     "lark_md",
			"content": content.String(),
		},
	}
}

// buildFailureSection 构建失败信息区域
func (b *FeishuCardBuilder) buildFailureSection() map[string]interface{} {
	return map[string]interface{}{
		"tag": "div",
		"text": map[string]interface{}{
			"tag":     "lark_md",
			"content": fmt.Sprintf("**⚠️ 失败告警**\n\n共有 <font color='red'>**%d**</font> 台主机部署失败，请及时处理！", b.payload.Deployment.FailedCount),
		},
	}
}

// buildTimestampSection 构建时间戳区域
func (b *FeishuCardBuilder) buildTimestampSection() map[string]interface{} {
	timeStr := ""
	if !b.payload.Timestamp.IsZero() {
		timeStr = b.payload.Timestamp.Format("2006-01-02 15:04:05")
	}

	return map[string]interface{}{
		"tag": "note",
		"elements": []map[string]interface{}{
			{
				"tag":     "plain_text",
				"content": fmt.Sprintf("🕐 %s | 任务ID: %s", timeStr, b.payload.Task.ID),
			},
		},
	}
}

// ==================== 默认飞书模板 ====================

// GetDefaultFeishuTemplates 获取默认飞书卡片模板（JSON 格式）
func GetDefaultFeishuTemplates() map[string]string {
	return map[string]string{
		// 简洁模板
		"feishu_simple": `{
  "header": {
    "title": {
      "tag": "plain_text",
      "content": "{{#if is_success}}✅{{else}}❌{{/if}} {{project_name}} 发布通知"
    },
    "template": "{{#if is_success}}green{{else}}red{{/if}}"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**版本**: {{version_name}}\n**环境**: {{environment}}\n**状态**: {{task_status}}\n**完成/总数**: {{completed_count}}/{{total_count}}"
      }
    },
    {
      "tag": "note",
      "elements": [
        {
          "tag": "plain_text",
          "content": "{{timestamp}}"
        }
      ]
    }
  ]
}`,

		// 详细模板
		"feishu_detailed": `{
  "header": {
    "title": {
      "tag": "plain_text",
      "content": "{{#if is_success}}🎉 发布成功{{else}}❌ 发布失败{{/if}} - {{project_name}}"
    },
    "template": "{{#if is_success}}green{{else}}red{{/if}}"
  },
  "elements": [
    {
      "tag": "div",
      "fields": [
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**📦 项目**\n{{project_name}}"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**🏷️ 版本**\n{{version_name}}"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**🌍 环境**\n{{environment}}"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**{{#if is_success}}✅{{else}}❌{{/if}} 状态**\n{{task_status}}"
          }
        }
      ]
    },
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**📊 部署统计**\n• 总数: **{{total_count}}**\n• 完成: **{{completed_count}}**{{#if has_failed}}\n• <font color='red'>失败: **{{failed_count}}**</font>{{/if}}{{#if has_canary}}\n• 金丝雀: **{{canary_count}}**{{/if}}\n\n**成功率**: {{success_rate}}%{{#if has_duration}}\n**耗时**: {{duration}}{{/if}}"
      }
    },
    {{#if has_hosts}}
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**🖥️ 主机列表** ({{hosts_count}} 台)\n{{hosts_string}}"
      }
    },
    {{/if}}
    {
      "tag": "note",
      "elements": [
        {
          "tag": "plain_text",
          "content": "🕐 {{timestamp}} | 任务ID: {{task_id}}"
        }
      ]
    }
  ]
}`,

		// 金丝雀专用模板
		"feishu_canary": `{
  "header": {
    "title": {
      "tag": "plain_text",
      "content": "🐦 金丝雀发布 - {{project_name}}"
    },
    "template": "blue"
  },
  "elements": [
    {
      "tag": "div",
      "fields": [
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**版本**\n{{version_name}}"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "**环境**\n{{environment}}"
          }
        }
      ]
    },
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**🐦 金丝雀进度**\n\n金丝雀数量: **{{canary_count}}** / 总数: **{{total_count}}**\n\n当前状态: {{#if is_success}}✅ 金丝雀验证通过{{else}}🔄 金丝雀验证中{{/if}}"
      }
    },
    {
      "tag": "note",
      "elements": [
        {
          "tag": "plain_text",
          "content": "{{timestamp}}"
        }
      ]
    }
  ]
}`,

		// 告警模板（失败时使用）
		"feishu_alert": `{
  "header": {
    "title": {
      "tag": "plain_text",
      "content": "🚨 发布告警 - {{project_name}}"
    },
    "template": "red"
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**⚠️ 部署过程中发生错误**\n\n项目: {{project_name}}\n版本: {{version_name}}\n环境: {{environment}}"
      }
    },
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "**❌ 失败统计**\n\n• 失败数量: <font color='red'>**{{failed_count}}**</font>\n• 成功数量: {{completed_count}}\n• 总数量: {{total_count}}\n• 成功率: <font color='red'>{{success_rate}}%</font>"
      }
    },
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "请及时检查并处理失败的部署任务！"
      }
    },
    {
      "tag": "note",
      "elements": [
        {
          "tag": "plain_text",
          "content": "🕐 {{timestamp}} | 任务ID: {{task_id}}"
        }
      ]
    }
  ]
}`,
	}
}

// GetFeishuTemplateByEvent 根据事件类型获取推荐模板
func GetFeishuTemplateByEvent(eventType models.CallbackEventType, hasFailed bool) string {
	templates := GetDefaultFeishuTemplates()

	// 失败时使用告警模板
	if hasFailed {
		return templates["feishu_alert"]
	}

	// 根据事件类型选择模板
	switch eventType {
	case models.CallbackEventCanaryStarted, models.CallbackEventCanaryCompleted:
		return templates["feishu_canary"]
	default:
		return templates["feishu_detailed"]
	}
}
