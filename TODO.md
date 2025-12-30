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
- [ ] **T067**: 在 QUIC 流上运行 SSH 协议层
  - 描述：使用 `golang.org/x/crypto/ssh` 包，将 QUIC 流包装成 `net.Conn`，在其上建立 SSH 握手
  - 架构设计：
    - **传输层**：UDP
    - **隧道层**：QUIC (多路复用、加密、拥塞控制)
    - **应用层**：SSH (权限控制、Shell 交互、文件传输)
  - 实现步骤：
    1. **实现 StreamConn 适配器**
       - 将 `quic.Stream` 转换为 `net.Conn`
       - 实现 `LocalAddr()`, `RemoteAddr()`, `SetDeadline()` 等方法
       - 文件：`pkg/ssh/adapter.go`
    2. **客户端（内网侧）：运行 SSH Server**
       - 监听来自 QUIC 的流
       - 在 Stream 上启动 SSH 服务
       - 配置 SSH Server（主机密钥、密码验证）
       - 处理 SSH 请求（Shell、端口转发等）
       - 文件：`pkg/ssh/server.go`
    3. **服务端（公网侧）：作为 SSH Client**
       - 主动打开 QUIC Stream
       - 在 Stream 上运行 SSH Client 逻辑
       - 配置 SSH 客户端认证
       - 支持多会话复用（每个会话一个独立的 QUIC Stream）
       - 文件：`pkg/ssh/client.go`
    4. **Stream 类型识别**
       - 在 `AcceptStream` 后发送握手信号（魔数）
       - 区分业务数据、文件传输、反向 SSH 等 Stream 类型
       - 文件：`pkg/ssh/protocol.go`
    5. **配置和文档**
       - 添加 SSH 相关配置选项
       - 编写使用文档和示例
       - 文件：`pkg/ssh/config.go`, `docs/ssh-over-quic.md`
  - 优势：
    - ✅ 协议分层清晰，职责明确
    - ✅ 双重加密：QUIC 层 + SSH 层
    - ✅ 内网穿透：反向 SSH 通道永久可用
    - ✅ 多路复用：同一 QUIC 连接可开启多个 SSH 会话
    - ✅ 复用现有 QUIC 长连接，无需额外连接
  - 技术要点：
    - 使用 `golang.org/x/crypto/ssh` 标准库
    - `quic.Stream` 实现 `io.Reader` 和 `io.Writer`
    - 适配器模式：`StreamConn` 实现 `net.Conn` 接口
    - `ssh.NewClientConn` / `ssh.NewServerConn` 需要标准 `net.Conn`
  - 安全考虑：
    - QUIC 层加密 + SSH 层加密双重保护
    - 即使 QUIC 认证被攻破，仍需 SSH 密钥/密码
    - 生产环境建议使用密钥认证而非密码
  - 参考：
    - `golang.org/x/crypto/ssh` 官方文档
    - QUIC Stream 接口：`github.com/quic-go/quic-go`
  - 预计：15-20 小时
  - 优先级：⭐⭐⭐
  - 状态：待开始

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

最后更新：2025-12-31
