# 实现计划：QUIC 通信骨干网络

**分支**：`001-quic-backbone-network` | **日期**：2025-12-23 | **规范**：[spec.md](./spec.md)
**输入**：来自 `/specs/001-quic-backbone-network/spec.md` 的功能规范

**注意**：此模板由 `/speckit.plan` 命令填充。请参阅 `.specify/templates/commands/plan.md` 了解执行工作流。

## 摘要

本功能旨在构建一个工业级、高性能的 QUIC 通信骨干网络，支持客户端-服务器模式下的可靠消息传输。核心需求包括：

1. **可靠连接管理**：基于 QUIC 协议的安全连接，支持自动重连和指数退避策略
2. **消息可靠传输**：单播、广播消息分发，在弱网环境下保证送达
3. **异步回调机制**：Promise 模式的请求-响应链路，支持超时处理
4. **健康监控**：心跳检测（15s 间隔，3 次超时清理）和状态钩子

技术方法：采用 lucas-clemente/quic-go 作为 QUIC 实现，使用 Protobuf 序列化，TLS 1.3 安全通道，支持 10,000+ 并发连接，消息延迟 P99 < 50ms（局域网）。

## 技术上下文

**语言/版本**：Go 1.21+（lucas-clemente/quic-go 要求）
**主要依赖**：
  - `github.com/lucas-clemente/quic-go`（QUIC 协议实现）
  - `google.golang.org/protobuf`（消息序列化）
  - `crypto/tls`（TLS 1.3 安全通道）
**存储**：内存存储（SyncMap 存储会话和回调映射），无持久化需求
**测试**：Go testing 框架 + testify（断言库），需要弱网模拟工具（NEEDS CLARIFICATION: 具体弱网测试工具选型）
**目标平台**：Linux/macOS 服务器（生产环境），支持跨平台客户端
**项目类型**：单一项目（库 + 示例服务器/客户端）
**性能目标**：
  - 连接建立 < 100ms
  - 消息延迟 P99 < 50ms（局域网）/ < 200ms（广域网）
  - 吞吐量 > 10,000 条/秒（单连接）
  - 并发连接 > 10,000（单实例）
**约束**：
  - 必须使用 QUIC 协议（不降级到 TCP）
  - 必须使用 TLS 1.3
  - 单实例部署（不考虑集群）
  - 消息不持久化（仅内存传输）
**规模/范围**：
  - 支持 10,000+ 并发客户端
  - 单条消息 < 1MB
  - 回调映射表需要限制大小防止内存溢出（NEEDS CLARIFICATION: 回调映射表清理策略和容量限制）

## 宪章检查

*门控：在阶段 0 研究之前必须通过。在阶段 1 设计之后重新检查。*

根据项目宪章（版本 2.0.0）的核心原则进行检查：

### I. 可靠优先（强制）

**检查项**：
- ✅ **QUIC 层重传机制**：使用 quic-go 内置的可靠流传输
- ✅ **应用层重连机制**：客户端实现 Idle → Connecting → Connected 状态机，指数退避重试
- ✅ **指令回调确认**：消息支持 WaitAck 标记，服务器维护 Promise Map 追踪送达状态
- ✅ **弱网降级策略**：超时重试、指数退避（1s, 2s, 4s... 最大 60s）
- ✅ **禁止假设网络可靠**：所有消息发送都考虑失败和重试场景
- ✅ **禁止无确认发送**：广播消息虽无 Ack，但在规范中明确标记为"尽力而为"模式

**潜在风险**：
- ⚠️ 广播消息无法保证每个客户端都收到（部分客户端可能离线或网络中断）
- **缓解措施**：在 quickstart.md 中明确说明广播的"尽力而为"语义，业务层需要根据场景决定是否使用单播替代

**评估**：✅ 通过

### II. 低耦合消息处理（强制）

**检查项**：
- ✅ **传输层与业务分离**：消息传输（QUIC + Protobuf）独立于业务处理逻辑
- ✅ **插件化处理机制**：消息路由通过 Dispatcher 模块，业务层注册 Handler 处理不同消息类型
- ✅ **标准化接口**：定义 MessageHandler 接口（OnMessage(msg) -> response）
- ✅ **消息路由机制**：根据消息类型（Protobuf 字段）分发到对应 Handler
- ⚠️ **处理器版本管理**：未在规范中明确要求（推荐实现）
- ✅ **禁止传输层硬编码业务逻辑**：传输层仅负责连接、编解码、路由
- ✅ **禁止处理器直接依赖**：Handler 之间通过消息总线通信（发送新消息）

**架构设计**：
```
传输层（Transport Layer）
  ├── QUIC 连接管理
  ├── Protobuf 序列化/反序列化
  └── 消息路由（Dispatcher）

业务层（Business Layer）
  ├── MessageHandler 接口
  ├── 具体 Handler 实现（用户注册）
  └── 消息发送 API
```

**评估**：✅ 通过

### III. 透明化监控（强制）

**检查项**：
- ✅ **连接状态实时监控**：提供 ListClients() API 查询所有在线客户端
- ✅ **心跳透明运行**：心跳机制独立于业务消息，15s 间隔自动发送 Ping/Pong
- ✅ **状态钩子接口**：提供事件钩子（OnConnect, OnDisconnect, OnHeartbeatTimeout, OnReconnect）
- ✅ **性能指标采集**：记录连接数、消息吞吐量、延迟（NEEDS CLARIFICATION: 指标采集方式，是否需要集成 Prometheus）
- ✅ **结构化日志**：记录关键事件（连接、断开、消息发送/接收、心跳超时）
- ⚠️ **可视化监控面板**：未在规范中要求（推荐实现，可后续扩展）
- ✅ **禁止心跳与业务混淆**：心跳帧独立处理，不占用业务消息通道
- ✅ **禁止隐藏状态变化**：所有关键状态变化通过钩子通知业务层

**监控接口设计**：
```go
type EventHooks struct {
    OnConnect           func(clientID string)
    OnDisconnect        func(clientID string, reason error)
    OnHeartbeatTimeout  func(clientID string)
    OnReconnect         func(clientID string, attemptCount int)
    OnMessageSent       func(msgID string, clientID string)
    OnMessageReceived   func(msgID string, clientID string)
}

type Metrics struct {
    ConnectedClients    int
    MessageThroughput   int64  // 消息/秒
    AverageLatency      int64  // 毫秒
    P99Latency          int64  // 毫秒
}
```

**评估**：✅ 通过

### 宪章合规性总结

| 原则 | 状态 | 备注 |
|------|------|------|
| I. 可靠优先 | ✅ 通过 | 广播消息的"尽力而为"语义需在文档中明确说明 |
| II. 低耦合消息处理 | ✅ 通过 | 架构设计清晰分离传输层和业务层 |
| III. 透明化监控 | ✅ 通过 | 需在 Phase 0 研究指标采集方式（内置 vs Prometheus） |

**整体评估**：✅ 通过 - 可以进入 Phase 0 研究阶段

## 项目结构

### 文档（此功能）

```
specs/001-quic-backbone-network/
├── plan.md              # 此文件（/speckit.plan 命令输出）
├── research.md          # 阶段 0 输出（/speckit.plan 命令）
├── data-model.md        # 阶段 1 输出（/speckit.plan 命令）
├── quickstart.md        # 阶段 1 输出（/speckit.plan 命令）
├── contracts/           # 阶段 1 输出（/speckit.plan 命令）
│   ├── protobuf/        # Protobuf 消息定义
│   └── api/             # Go API 接口定义
└── tasks.md             # 阶段 2 输出（/speckit.tasks 命令 - 不由 /speckit.plan 创建）
```

### 源代码（仓库根目录）

选择**选项 1：单一项目**（Go 库项目）

```
quic_project/
├── pkg/                    # 公共库代码
│   ├── protocol/           # Protobuf 生成代码和消息定义
│   ├── transport/          # QUIC 传输层实现
│   │   ├── server/         # 服务器端连接管理
│   │   ├── client/         # 客户端连接管理
│   │   └── codec/          # 编解码器
│   ├── session/            # 会话管理
│   │   ├── manager.go      # SessionManager（管理所有客户端会话）
│   │   └── heartbeat.go    # 心跳检测
│   ├── dispatcher/         # 消息路由和分发
│   │   ├── router.go       # 消息路由器
│   │   └── handler.go      # MessageHandler 接口
│   ├── callback/           # 异步回调机制
│   │   ├── promise.go      # Promise Map 管理
│   │   └── timeout.go      # 超时处理
│   └── monitoring/         # 监控和指标
│       ├── metrics.go      # 指标采集
│       ├── hooks.go        # 事件钩子
│       └── logger.go       # 结构化日志
├── cmd/                    # 命令行工具和示例
│   ├── server/             # 示例服务器
│   └── client/             # 示例客户端
├── examples/               # 使用示例
├── tests/                  # 测试代码
│   ├── unit/               # 单元测试
│   ├── integration/        # 集成测试
│   └── benchmark/          # 性能测试和弱网测试
├── docs/                   # 额外文档
├── go.mod                  # Go 模块定义
└── README.md               # 项目说明
```

**结构决策**：

选择单一项目结构的理由：
1. **项目性质**：这是一个 Go 库（library）项目，不是独立的应用或服务
2. **代码复用**：服务器端和客户端共享大量代码（协议、编解码、监控等）
3. **Go 惯例**：Go 项目通常使用 `pkg/` 存放公共库代码，`cmd/` 存放可执行程序
4. **简化依赖**：单一 go.mod 文件管理所有依赖，避免多模块复杂性

包组织原则：
- `pkg/protocol/`：Protobuf 定义和生成代码（与业务无关的协议层）
- `pkg/transport/`：QUIC 底层传输实现（连接、流、编解码）
- `pkg/session/`：会话管理和心跳（服务器端核心逻辑）
- `pkg/dispatcher/`：消息路由（解耦传输层和业务层的关键）
- `pkg/callback/`：异步回调（Promise 模式实现）
- `pkg/monitoring/`：监控、日志、钩子（可观测性）

## 复杂性跟踪

本项目符合宪章的简洁性原则，无需证明的违规。所有复杂性都由核心需求驱动：

| 组件 | 复杂性来源 | 正当性 |
|------|----------|--------|
| QUIC 传输层 | 协议本身复杂 | 需求明确要求 QUIC，无法简化 |
| Promise Map | 内存管理复杂 | 异步回调是核心需求，需要映射表追踪 |
| 心跳机制 | 并发定时器 | 健康监控需求，标准实现方式 |
| 状态机（客户端） | 3 状态转换 | 自动重连需求，状态机是最清晰的表达方式 |
| 消息路由 | Dispatcher 模式 | 低耦合原则要求，避免传输层硬编码业务逻辑 |

**无不合理的复杂性**。所有设计都是为满足规范和宪章要求的最小必要复杂度。

## Phase 0: 研究任务

以下是需要在 Phase 0 完成的研究任务，以解决 NEEDS CLARIFICATION 项：

1. **弱网测试工具选型**
   - 目标：确定用于模拟丢包、延迟、乱序的测试工具
   - 候选：tc（Linux Traffic Control）、netem、Toxiproxy、go-replayers
   - 输出：research.md 中的"弱网测试方案"章节

2. **回调映射表清理策略**
   - 目标：设计 Promise Map 的容量限制和清理机制
   - 考虑：超时清理、LRU 淘汰、容量上限（如 10,000 条）
   - 输出：research.md 中的"回调管理策略"章节

3. **指标采集方式**
   - 目标：确定是内置简单计数器还是集成 Prometheus
   - 考虑：规范要求"实时提供指标"，但未明确是否需要外部监控系统
   - 权衡：简单性 vs 可扩展性
   - 输出：research.md 中的"监控方案"章节

4. **QUIC 库最佳实践**
   - 目标：研究 lucas-clemente/quic-go 的连接池、流管理、配置优化
   - 参考：官方文档、性能测试报告、生产案例
   - 输出：research.md 中的"QUIC 实现最佳实践"章节

5. **Protobuf 消息设计**
   - 目标：设计消息协议的 .proto 文件结构
   - 考虑：消息类型、帧格式（Ping/Pong/Data）、版本兼容性
   - 输出：research.md 中的"消息协议设计"章节 + contracts/protobuf/ 初稿

6. **并发模式研究**
   - 目标：确定 SessionManager 和 Dispatcher 的并发模型
   - 考虑：Goroutine per connection vs worker pool、channel vs sync.Map
   - 输出：research.md 中的"并发架构"章节

**Phase 0 输出物**：`research.md`（包含以上 6 个研究主题的决策和理由）
