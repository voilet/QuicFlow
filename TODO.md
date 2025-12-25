# TODO List

本文档列出了项目的待办事项和优先级。

## ✅ 已完成 (Completed)

### Phase 1-2: 设置和基础 (100%)
- [x] T001-T007: 项目设置、依赖安装、证书生成
- [x] T008-T012: Protobuf 定义和生成
- [x] T013: 基础错误类型
- [x] T014-T015: Protobuf Codec 实现
- [x] T016-T019: 监控模块 (Metrics, Hooks, Logger, Histogram)
- [x] T020-T021: 会话管理 (Session, SessionManager)

### Phase 3: User Story 1 - MVP (100%)
- [x] T022: TLS 安全模块
- [x] T023-T026: 服务器核心实现
- [x] T027-T029: 客户端核心实现
- [x] T030: 客户端心跳循环
- [x] T031-T032: 服务器心跳检测
- [x] T033-T034: 查询方法 (ListClients, GetClientInfo)

### Phase 4: User Story 2 - 消息传输 (100%)
- [x] T035-T037: Dispatcher 和 MessageHandler
- [x] T038-T039: 单播和广播
- [x] T040-T043: Promise 机制和 Ack 处理
- [x] T044-T045: 客户端消息发送和接收
- [x] T046: QUIC 可靠传输文档

### Phase 5: User Story 3 - 高级监控 (100%)
- [x] T047: 增强的 GetMetrics 方法 (6 → 27 字段)
- [x] T048: Prometheus 指标导出器
- [x] T049: 监控示例程序

### Phase 6: 完善和交付 (部分完成 ~30%)
- [x] T050: 完整监控集成 - 所有事件钩子
- [x] T051: 完整指标采集
- [x] T052: 延迟统计实现
- [x] T058: 项目 README.md

## 🔄 进行中 (In Progress)

无

## 📋 待办事项 (Todo)

### 高优先级 (High Priority)

#### 示例程序 (Examples)
- [ ] **T053**: 创建完整示例服务器 `cmd/server/main.go`
  - 当前状态：基础版本已存在
  - 需要：添加消息处理、Dispatcher 集成示例
  - 预计：2-3 小时

- [ ] **T054**: 创建完整示例客户端 `cmd/client/main.go`
  - 当前状态：基础版本已存在
  - 需要：添加消息发送、WaitAck 示例
  - 预计：2-3 小时

- [ ] **T055**: 创建 Echo 示例 `examples/echo/`
  - 描述：服务器回显客户端消息
  - 文件：server.go, client.go, README.md
  - 预计：3-4 小时
  - 优先级：⭐⭐⭐

- [ ] **T056**: 创建广播示例 `examples/broadcast/`
  - 描述：服务器向所有客户端广播事件
  - 文件：server.go, client.go, README.md
  - 预计：3-4 小时
  - 优先级：⭐⭐⭐

- [ ] **T057**: 创建回调示例 `examples/callback/`
  - 描述：演示 AsyncSend 和 Promise 机制
  - 文件：server.go, client.go, README.md
  - 预计：3-4 小时
  - 优先级：⭐⭐⭐

#### 文档和配置
- [ ] **T059**: 创建 Go 文档注释
  - 范围：pkg/ 下所有导出的函数和类型
  - 工具：使用 `go doc` 验证
  - 预计：4-6 小时
  - 优先级：⭐⭐

- [ ] **T060**: 实现配置验证
  - 文件：`pkg/transport/server/config.go`, `pkg/transport/client/config.go`
  - 内容：完善 Validate() 方法，检查必填字段和范围
  - 当前状态：基础验证已存在
  - 预计：2 小时
  - 优先级：⭐⭐

- [ ] **T061**: 实现日志级别配置
  - 功能：支持环境变量 `LOG_LEVEL` 或配置参数
  - 级别：DEBUG/INFO/WARN/ERROR
  - 预计：1-2 小时
  - 优先级：⭐

#### 错误处理和优雅关闭
- [ ] **T062**: 错误处理标准化
  - 任务：确保所有错误使用 pkg/errors 中定义的类型
  - 检查：所有 error 返回值都有清晰的上下文
  - 预计：2-3 小时
  - 优先级：⭐⭐

- [ ] **T063**: 实现优雅关闭
  - 当前状态：基础实现已存在
  - 改进：添加超时控制、资源清理日志
  - 预计：2 小时
  - 优先级：⭐⭐

### 中优先级 (Medium Priority)

#### 测试和验证
- [ ] **T064**: 创建 quickstart 验证脚本
  - 文件：`scripts/test-quickstart.sh`
  - 功能：自动运行文档中的示例，验证输出
  - 预计：2-3 小时
  - 优先级：⭐

- [ ] **单元测试**: 为核心模块添加单元测试
  - 范围：pkg/callback, pkg/dispatcher, pkg/session
  - 目标覆盖率：70%+
  - 预计：8-10 小时
  - 优先级：⭐⭐

- [ ] **集成测试**: 端到端测试场景
  - 场景：连接、消息传输、重连、广播
  - 工具：使用 Go testing 框架
  - 预计：6-8 小时
  - 优先级：⭐

#### 性能优化
- [ ] **T065**: 性能分析
  - 工具：pprof (CPU, Memory)
  - 测试：10,000 并发连接
  - 输出：性能报告和瓶颈识别
  - 预计：4-6 小时
  - 优先级：⭐

- [ ] **T066**: 性能优化
  - 目标：减少内存分配、优化锁竞争
  - 依赖：T065 完成后
  - 预计：6-8 小时
  - 优先级：⭐

### 低优先级 (Low Priority)

#### 文档扩展
- [ ] 创建 API 文档 `docs/API.md`
  - 内容：详细的 API 参考
  - 格式：Markdown
  - 预计：6-8 小时

- [ ] 创建架构文档 `docs/ARCHITECTURE.md`
  - 内容：系统架构、设计决策、模块交互
  - 图表：使用 Mermaid 或 ASCII
  - 预计：4-6 小时

- [ ] 创建贡献指南 `CONTRIBUTING.md`
  - 内容：代码规范、提交流程、测试要求
  - 预计：2-3 小时

#### 工具和自动化
- [ ] 添加 Makefile 完善
  - 目标：test, coverage, lint, format, clean
  - 预计：2 小时

- [ ] 配置 CI/CD
  - 平台：GitHub Actions
  - 流程：test, build, lint
  - 预计：3-4 小时

- [ ] 添加代码质量检查
  - 工具：golangci-lint (已配置)
  - 集成：pre-commit hook
  - 预计：2 小时

#### 功能增强
- [ ] WebSocket 支持 (可选)
  - 描述：为 Web 客户端提供 WebSocket 接口
  - 预计：8-10 小时

- [ ] 消息压缩 (可选)
  - 算法：gzip/zstd
  - 配置：可选启用
  - 预计：4-6 小时

- [ ] 消息加密 (可选)
  - 层级：应用层加密 (TLS 之上)
  - 算法：AES-256-GCM
  - 预计：6-8 小时

## 📊 优先级说明

- ⭐⭐⭐ **Critical**: 必须完成才能达到生产就绪
- ⭐⭐ **High**: 强烈建议完成，提升项目质量
- ⭐ **Medium**: 可选，但有价值
- (无星) **Low**: 未来改进

## 🎯 里程碑

### Milestone 1: 基础功能 ✅
- Phase 1-3 完成
- 状态：已完成

### Milestone 2: 完整功能 ✅
- Phase 4-5 完成
- 状态：已完成

### Milestone 3: 生产就绪 (80% 完成)
- Phase 6 核心任务完成
- 剩余：示例程序、文档注释
- 预计完成：再需 20-30 小时

### Milestone 4: 企业级 (未开始)
- 完整测试覆盖
- 性能优化
- CI/CD 集成
- 预计完成：再需 40-50 小时

## 📝 注释

- 所有时间预估基于单人开发
- 优先级可能根据实际需求调整
- 标记 ✅ 的任务已完成
- 标记 🔄 的任务正在进行
- 标记 ⏸️ 的任务已暂停

---

最后更新：2025-12-24
