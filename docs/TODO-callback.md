# 发布回调功能开发任务清单

## 已完成 ✅

- [x] **Phase 1.1-1.6**: 数据模型设计、回调配置 API、飞书/钉钉/企业微信回调实现、发布引擎集成
- [x] **Phase 1.7**: 回调历史记录存储和查询 API
- [x] **Phase 2.1**: 自定义 HTTP 回调接口支持
  - 实现 `CustomNotifier` 通知器
  - 支持自定义 URL、HTTP 方法、请求头、超时、重试配置
  - 支持消息模板变量替换
- [x] **Phase 1.8**: Web UI 回调配置管理页面
  - 创建 `CallbackConfig.vue` 页面
  - 支持项目选择、配置 CRUD 操作
  - 支持四种渠道配置（飞书、钉钉、企业微信、自定义）
  - 支持测试回调功能
- [x] **Phase 2.2**: 回调失败重试机制和队列
  - 创建 `RetryQueue` 重试队列
  - 指数退避重试策略
  - 异步工作线程处理
  - 集成到 Manager

## 待开发 🚧

（所有计划任务已完成）

### Phase 2.3: 回调模板自定义功能 ✅
- [x] 扩展消息模板引擎，支持更丰富的变量
- [x] 支持条件渲染和循环
- [x] 添加模板预览功能

### Phase 2.4: Web UI 回调历史查看页面 ✅
- [x] 创建 `CallbackHistory.vue` 页面
- [x] 支持按项目、任务、状态筛选
- [x] 展示请求/响应详情
- [x] 支持重试失败记录

### Phase 3.1: 消息模板管理功能 ✅
- [x] 模板 CRUD API
- [x] 模板预览功能
- [x] 默认模板管理

### Phase 3.2: Webhook 签名验证机制 ✅
- [x] 飞书签名验证
- [x] 钉钉签名验证
- [x] 企业微信签名验证（通过 access_token 认证）

### Phase 3.3: 回调统计与监控指标 ✅
- [x] 成功率统计
- [x] 延迟统计
- [x] 队列状态监控
- [x] 统计 API 端点

---

## 文件清单

### 后端文件
- `pkg/release/models/callback.go` - 数据模型
- `pkg/release/callback/notifier.go` - 通知器实现
- `pkg/release/callback/manager.go` - 回调管理器
- `pkg/release/callback/retry_queue.go` - 重试队列
- `pkg/release/api/callback_api.go` - API 处理器

### 前端文件
- `web/src/views/CallbackConfig.vue` - 回调配置页面
- `web/src/api/index.js` - API 接口定义
- `web/src/router/index.js` - 路由配置
- `web/src/App.vue` - 导航菜单

### 数据库表
- `callback_configs` - 回调配置表
- `callback_history` - 回调历史记录表
