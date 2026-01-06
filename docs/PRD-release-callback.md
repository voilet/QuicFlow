# 发布回调功能产品需求文档 (PRD)

## 一、需求概述

为发布系统增加多渠道回调通知功能，支持在发布关键节点自动发送通知到飞书、钉钉、企业微信或自定义接口，提升发布流程的可观测性和团队协作效率。

## 二、需求背景

当前发布系统缺乏主动通知机制，相关人员无法及时获知发布进展。需要增加自动化回调能力，在发布的关键节点（金丝雀发布开始/完成、全量发布完成）主动推送通知。

## 三、功能需求

### 3.1 支持的回调渠道

| 渠道 | 说明 | 优先级 |
|------|------|--------|
| 飞书 | 通过飞书机器人 Webhook 发送消息 | P0 |
| 钉钉 | 通过钉钉机器人 Webhook 发送消息 | P0 |
| 企业微信 | 通过企业微信应用推送消息 | P0 |
| 自定义接口 | 支持自定义 HTTP 回调接口 | P1 |

### 3.2 回调触发时机

| 触发时机 | 说明 | 触发条件 |
|----------|------|----------|
| 金丝雀开始发布 | 金丝雀批次开始部署时 | status: pending → running (金丝雀阶段) |
| 金丝雀发布完成 | 金丝雀批次验证完成，等待推广 | 金丝雀任务全部完成 |
| 全部发布完成 | 所有批次部署完成 | status: running → success |

### 3.3 回调消息内容

标准回调消息包含以下字段：

```json
{
  "event_type": "canary_started | canary_completed | full_completed",
  "project": {
    "id": "项目ID",
    "name": "项目名称",
    "description": "项目描述"
  },
  "version": {
    "id": "版本ID",
    "name": "版本名称/标签",
    "description": "版本描述"
  },
  "task": {
    "id": "任务ID",
    "type": "deploy | rollback | stop",
    "strategy": "canary | rolling | blue_green",
    "status": "当前状态"
  },
  "deployment": {
    "total_count": 100,          // 总发布数量
    "canary_count": 10,          // 金丝雀数量（仅金丝雀事件）
    "completed_count": 10,       // 已完成数量
    "failed_count": 0,           // 失败数量
    "hosts": ["host1", "host2"]  // 发布的主机列表
  },
  "timestamp": "2026-01-06T20:00:00Z",
  "duration": "30s",             // 发布耗时
  "environment": "production | staging | development"
}
```

## 四、技术方案

### 4.1 数据模型设计

```go
// 回调配置
type CallbackConfig struct {
    ID          string                 `json:"id"`
    ProjectID   string                 `json:"project_id"`   // 项目级配置
    Enabled     bool                   `json:"enabled"`
    Channels    []CallbackChannel      `json:"channels"`
    Events      []string               `json:"events"`       // 订阅的事件类型
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// 回调渠道
type CallbackChannel struct {
    Type     string                 `json:"type"`      // feishu, dingtalk, wechat, custom
    Enabled  bool                   `json:"enabled"`
    Config   map[string]interface{} `json:"config"`    // 渠道特定配置
}

// 飞书配置
type FeishuConfig struct {
    WebhookURL string `json:"webhook_url"`
    SignSecret string `json:"sign_secret,omitempty"` // 可选，签名验证
}

// 钉钉配置
type DingTalkConfig struct {
    WebhookURL string `json:"webhook_url"`
    SignSecret string `json:"sign_secret,omitempty"`
}

// 企业微信配置
type WeChatConfig struct {
    CorpID     string `json:"corp_id"`
    AgentID    int64  `json:"agent_id"`
    Secret     string `json:"secret"`
    ToUser     string `json:"to_user,omitempty"`     // 接收用户，默认全部
}

// 自定义接口配置
type CustomCallbackConfig struct {
    URL            string            `json:"url"`
    Method         string            `json:"method"`          // POST/PUT
    Headers        map[string]string `json:"headers"`
    Timeout        int               `json:"timeout"`         // 秒
    RetryCount     int               `json:"retry_count"`
    RetryInterval  int               `json:"retry_interval"`  // 秒
}

// 回调历史记录
type CallbackHistory struct {
    ID          string                 `json:"id"`
    TaskID      string                 `json:"task_id"`
    EventType   string                 `json:"event_type"`
    ChannelType string                 `json:"channel_type"`
    Status      string                 `json:"status"`         // success, failed, pending
    Request     map[string]interface{} `json:"request"`
    Response    string                 `json:"response"`
    Error       string                 `json:"error,omitempty"`
    CreatedAt   time.Time              `json:"created_at"`
}
```

### 4.2 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                        发布执行引擎                           │
│                  (Release Engine)                           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │  事件发布器      │
                    │  Event Publisher │
                    └─────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
     ┌────────────┐  ┌────────────┐  ┌────────────┐
     │ 飞书发送器  │  │ 钉钉发送器  │  │ 微信发送器  │
     │ Feishu     │  │ DingTalk   │  │ WeChat     │
     └────────────┘  └────────────┘  └────────────┘
              │               │               │
              └───────────────┼───────────────┘
                              ▼
                    ┌─────────────────┐
                    │  回调历史记录    │
                    │  History Store  │
                    └─────────────────┘
```

### 4.3 API 设计

```bash
# 回调配置管理
GET    /api/v1/release/callbacks                    # 列出所有配置
GET    /api/v1/release/projects/:id/callbacks       # 获取项目回调配置
POST   /api/v1/release/projects/:id/callbacks       # 创建回调配置
PUT    /api/v1/release/callbacks/:id                # 更新回调配置
DELETE /api/v1/release/callbacks/:id                # 删除回调配置
POST   /api/v1/release/callbacks/:id/test           # 测试回调

# 回调历史
GET    /api/v1/release/callbacks/history            # 回调历史列表
GET    /api/v1/release/tasks/:id/callbacks          # 任务回调历史
GET    /api/v1/release/callbacks/:id/history        # 配置的回调历史
```

### 4.4 消息格式

#### 飞书消息格式
```json
{
  "msg_type": "interactive",
  "card": {
    "header": {
      "title": {
        "tag": "plain_text",
        "content": "发布通知：金丝雀发布开始"
      },
      "template": "blue"
    },
    "elements": [
      {
        "tag": "div",
        "text": {
          "tag": "lark_md",
          "content": "**项目**: my-app\n**版本**: v1.2.3\n**状态**: 发布中..."
        }
      }
    ]
  }
}
```

#### 钉钉消息格式
```json
{
  "msgtype": "markdown",
  "markdown": {
    "title": "发布通知",
    "text": "## 发布通知\n\n**项目**: my-app\n**版本**: v1.2.3\n..."
  }
}
```

#### 企业微信消息格式
```json
{
  "msgtype": "markdown",
  "markdown": {
    "content": "## 发布通知\n\n>项目: my-app\n>版本: v1.2.3"
  }
}
```

#### 自定义接口格式
直接发送标准 JSON 消息体

## 五、开发任务

### Phase 1: 核心功能 (P0)
- [ ] 数据模型设计与数据库迁移
- [ ] 回调配置管理 API
- [ ] 飞书回调实现
- [ ] 钉钉回调实现
- [ ] 企业微信回调实现
- [ ] 发布引擎集成回调触发点
- [ ] 回调历史记录
- [ ] Web UI: 回调配置页面

### Phase 2: 增强功能 (P1)
- [ ] 自定义回调接口支持
- [ ] 回调失败重试机制
- [ ] 回调模板自定义
- [ ] Web UI: 回调历史查看

### Phase 3: 优化功能 (P2)
- [ ] 回调消息模板管理
- [ ] Webhook 签名验证
- [ ] 回调统计与监控
- [ ] 批量回调支持

## 六、验收标准

1. 支持飞书、钉钉、企业微信三种主流协作平台
2. 在金丝雀发布开始、完成和全部发布完成时准确触发回调
3. 回调消息包含完整的项目、版本、主机、状态信息
4. 回调失败时有重试机制和日志记录
5. Web UI 可视化配置和测试回调功能

## 七、风险与依赖

| 风险点 | 影响 | 缓解措施 |
|--------|------|----------|
| 第三方 API 限流 | 回调发送失败 | 实现重试队列和降级策略 |
| 敏感信息泄露 | Webhook URL 泄露 | 配置加密存储，权限控制 |
| 回调阻塞发布 | 回调超时影响发布 | 异步发送，超时控制 |
| 消息格式变更 | 第三方平台更新 | 抽象消息格式，版本兼容 |

## 八、后续优化方向

1. 支持更多渠道（Slack、Discord、邮件等）
2. 条件触发回调（如失败时、长时间运行时）
3. 回调审批流集成
4. 回调数据分析与报表
