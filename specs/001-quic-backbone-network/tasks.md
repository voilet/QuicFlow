# 任务：QUIC 通信骨干网络

**输入**：来自 `/specs/001-quic-backbone-network/` 的设计文档
**先决条件**：plan.md（必需）、spec.md（用户故事必需）、research.md、data-model.md、contracts/

**版本**：v2.0（优化版）
**优化内容**：
- ✅ 重组用户故事：心跳机制整合到 US1（对齐用户 Phase 2 路线图）
- ✅ 统一示例程序：移除分散更新，统一到阶段 6 创建
- ✅ 统一监控集成：避免重复，阶段 6 一次性完整集成
- ✅ 增加基础设施：错误类型定义提前到阶段 2
- ✅ 合并细粒度任务：Dispatcher、TLS、心跳等任务合并

**测试**：规范未明确要求测试，本任务清单不包含测试任务。如需 TDD 方法，请明确指定。

**组织**：任务按用户故事分组，以实现每个故事的独立实现和测试。

## 格式：`[ID] [P?] [Story] 描述`
- **[P]**：可以并行运行（不同文件、无依赖关系）
- **[Story]**：此任务属于哪个用户故事（例如 US1、US2、US3）
- 在描述中包含确切的文件路径

## 路径约定
- **单一项目**：仓库根目录的 `pkg/`、`cmd/`、`tests/`
- 下面显示的路径基于 plan.md 中定义的单一 Go 项目结构

---

## 阶段 1：设置（共享基础设施）

**目的**：项目初始化和基本结构

- [X] T001 初始化 Go 模块：go.mod（module github.com/voilet/QuicFlow）
- [X] T002 创建项目目录结构：pkg/, cmd/, tests/, examples/, docs/, scripts/
- [X] T003 [P] 安装核心依赖：github.com/lucas-clemente/quic-go, google.golang.org/protobuf
- [X] T004 [P] 创建 .gitignore 文件（忽略生成代码、二进制文件、证书）
- [X] T005 [P] 配置 golangci-lint：.golangci.yml（启用 gofmt、govet、staticcheck 等）
- [X] T006 [P] 创建 Makefile：定义 build、test、proto-gen、lint、clean 命令
- [X] T007 [P] 生成开发用 TLS 证书和密钥：scripts/gen-certs.sh（使用 openssl）

---

## 阶段 2：基础（阻塞先决条件）

**目的**：在任何用户故事实现之前必须完成的核心基础设施

**⚠️ 关键**：在此阶段完成之前，不能开始用户故事工作

### Protobuf 协议定义

- [X] T008 定义帧协议：pkg/protocol/frame.proto（Frame, FrameType, PingFrame, PongFrame）
- [X] T009 定义消息协议：pkg/protocol/message.proto（DataMessage, MessageType, AckMessage, AckStatus）
- [X] T010 定义公共类型：pkg/protocol/types.proto（ClientInfo, ClientState, MetricsSnapshot）
- [X] T011 生成 Protobuf Go 代码：运行 protoc --go_out=. pkg/protocol/*.proto

### 基础模块

- [X] T012 [P] 定义基础错误类型：pkg/errors/errors.go（ErrConnectionClosed, ErrClientNotConnected, ErrPayloadTooLarge, ErrPromiseCapacityFull, ErrTimeout）
- [X] T013 [P] 实现 Protobuf 编解码器：pkg/transport/codec/protobuf.go（Encode/Decode Frame）
- [X] T014 [P] 实现基础监控模块：pkg/monitoring/metrics.go（Metrics 结构体，atomic 计数器）
- [X] T015 [P] 实现事件钩子模块：pkg/monitoring/hooks.go（EventHooks 结构体，6 种事件回调）
- [X] T016 [P] 实现结构化日志封装：pkg/monitoring/logger.go（基于 log/slog，支持不同级别）
- [X] T017 [P] 实现轻量级 Histogram：pkg/monitoring/histogram.go（延迟百分位统计）
- [X] T018 实现 SessionManager 结构体：pkg/session/manager.go（使用 sync.Map，支持 Add/Remove/GetByID/Range，保证并发安全）
- [X] T019 实现 ClientSession 数据结构：pkg/session/session.go（ClientID, Conn, LastHeartbeat, TimeoutCount 等字段）

**检查点**：基础就绪 - 用户故事实现现在可以并行开始

---

## 阶段 3：用户故事 1 - 建立可靠连接 + 心跳机制 (优先级: P1) 🎯 MVP

**目标**：实现基于 QUIC 的安全连接建立，支持客户端自动重连、指数退避策略，以及心跳健康检测

**独立测试**：启动服务器和客户端，观察连接建立过程，断开网络后验证自动重连，停止客户端心跳验证服务器自动清理

**🎯 对齐用户路线图**：
- ✅ Phase 1: QUIC 握手 + TLS + Protobuf（阶段 2 已完成）
- ✅ Phase 2: SessionManager + 重连状态机 + 心跳检测定时器（本阶段）

### 用户故事 1 的实现

#### TLS 安全通道

- [X] T020 [P] [US1] 实现 TLS 安全模块：pkg/transport/tls/tls.go（加载证书、配置 TLS 1.3、ALPN 设置）

#### 服务器端连接管理

- [X] T021 [P] [US1] 实现服务器 QUIC 配置：pkg/transport/server/config.go（TLS 配置、QUIC 参数优化）
- [X] T022 [US1] 实现服务器主结构体：pkg/transport/server/server.go（Server 结构体，Start/Stop 方法）
- [X] T023 [US1] 实现连接接受循环：pkg/transport/server/server.go 的 acceptLoop（接受新连接，创建 ClientSession）
- [X] T024 [US1] 实现单连接处理：pkg/transport/server/server.go 的 handleConnection（每连接一个 goroutine，流处理）
- [X] T025 [US1] 实现流处理：pkg/transport/server/server.go 的 handleStream（解码 Frame，路由到处理器）

#### 客户端连接管理

- [X] T026 [P] [US1] 实现客户端 QUIC 配置：pkg/transport/client/config.go（TLS 配置、QUIC 参数优化）
- [X] T027 [US1] 实现客户端主结构体：pkg/transport/client/client.go（Client 结构体，状态机字段）
- [X] T028 [US1] 实现客户端 DialLoop 函数：pkg/transport/client/client.go 的 reconnectLoop（通过 for-select 监听连接异常并触发重连，指数退避 1s→60s）
- [X] T029 [US1] 实现连接状态管理：pkg/transport/client/client.go（Idle → Connecting → Connected 状态转换）

#### 心跳机制（整合自原 US3）

- [X] T030 [P] [US1] 实现客户端心跳循环：pkg/transport/client/heartbeat.go（每 15 秒发送 PingFrame，接收 PongFrame）
- [X] T031 [US1] 实现服务器心跳处理：pkg/transport/server/server.go 的 handlePing（接收 Ping 返回 Pong，更新 LastHeartbeat）
- [X] T032 [US1] 实现服务器心跳检查器：pkg/session/manager.go 的 heartbeatChecker（定时器每 5 秒检查所有会话，连续 3 次超时则清理）

#### 连接状态查询

- [X] T033 [P] [US1] 实现 ListClients 方法：pkg/transport/server/server.go（返回所有在线客户端 ID 列表）
- [X] T034 [P] [US1] 实现 GetClientInfo 方法：pkg/transport/server/server.go（返回指定客户端详细信息）

**检查点**：此时，用户故事 1 应该完全功能并可独立测试（连接建立、自动重连、心跳检测、会话清理）

---

## 阶段 4：用户故事 2 - 消息可靠传输 + 异步回调 (优先级: P1)

**目标**：实现单播、广播消息分发，以及异步回调（Promise 模式）

**独立测试**：发送单播和广播消息，在正常网络和模拟弱网环境下验证消息送达率，测试带回调的消息

**🎯 对齐用户路线图**：
- ✅ Phase 3: Dispatcher + 请求-响应链路 + 广播 API（本阶段）

### 用户故事 2 的实现

#### 消息路由与分发（合并优化）

- [ ] T035 [P] [US2] 实现 Dispatcher 核心：pkg/dispatcher/dispatcher.go（Dispatcher 结构体、worker pool、消息分发循环）
- [ ] T036 [US2] 实现 Handler 注册和路由：pkg/dispatcher/dispatcher.go（RegisterHandler 方法、按 MessageType 路由、调用 OnMessage）
- [ ] T037 [P] [US2] 定义 MessageHandler 接口：pkg/dispatcher/handler.go（OnMessage 方法签名，context 超时控制）

#### 单播和广播

- [ ] T038 [P] [US2] 实现服务器单播方法：pkg/transport/server/server.go 的 SendTo（发送到指定客户端，验证客户端存在）
- [ ] T039 [P] [US2] 实现服务器广播方法：pkg/transport/server/server.go 的 Broadcast（遍历 SessionManager，统计成功数）

#### 异步回调（Promise 模式）

- [ ] T040 [P] [US2] 实现 Promise 数据结构：pkg/callback/promise.go（Promise 结构体，RespChan, Timer, 超时机制）
- [ ] T041 [US2] 实现 PromiseManager：pkg/callback/manager.go（Create/Cleanup 方法，sync.Map 存储，容量保护 50k 上限）
- [ ] T042 [US2] 封装 AsyncSend 方法：pkg/transport/server/server.go（利用 UUID 追踪消息，创建 Promise，返回 Response channel）
- [ ] T043 [US2] 实现 Ack 消息处理：pkg/transport/server/server.go 和 client.go（接收 AckMessage，写入 Promise.RespChan）

#### 客户端消息发送

- [ ] T044 [US2] 实现客户端 SendMessage 方法：pkg/transport/client/client.go（支持 WaitAck 参数，超时控制）
- [ ] T045 [US2] 实现客户端消息接收循环：pkg/transport/client/client.go 的 receiveLoop（接收服务器消息，分发到 Dispatcher）

#### 弱网支持验证

- [ ] T046 [US2] 验证 QUIC 可靠传输：确保 quic-go 配置正确（接收窗口、流控参数），文档记录在 docs/network-reliability.md

**检查点**：此时，用户故事 1 和 2 都应该独立工作（连接 + 心跳 + 消息传输 + 回调）

---

## 阶段 5：用户故事 3 - 高级监控特性 (优先级: P2)

**目的**：提供高级监控能力，支持 Prometheus 集成和可视化（可选特性）

**独立测试**：启动服务器，访问 Prometheus 指标端点，验证指标数据正确

### 用户故事 3 的实现

- [ ] T047 [P] [US3] 实现 GetMetrics 方法：pkg/transport/server/server.go（返回 MetricsSnapshot，包含当前连接数、吞吐量、延迟等）
- [ ] T048 [P] [US3] 实现 Prometheus 指标导出器：pkg/monitoring/prometheus.go（PrometheusHandler，手动生成文本格式指标）
- [ ] T049 [US3] 创建监控示例程序：examples/monitoring/main.go（展示如何使用 EventHooks、GetMetrics、Prometheus 导出）

**检查点**：高级监控特性完成，可选择性启用

---

## 阶段 6：完善和跨领域关注点

**目的**：影响多个用户故事的改进和最终交付准备

### 统一监控集成（优化：避免分散）

- [ ] T050 [P] 完整监控集成：在所有关键点触发事件钩子（OnConnect, OnDisconnect, OnHeartbeatTimeout, OnReconnect, OnMessageSent, OnMessageReceived）
- [ ] T051 [P] 完整指标采集：确保所有 Metrics 字段正确更新（ConnectedClients, MessageThroughput, AverageLatency, P99Latency 等）
- [ ] T052 [P] 实现延迟统计：在消息发送和接收时记录时间戳，使用 Histogram 计算延迟分布

### 统一示例程序（优化：避免分散更新）

- [ ] T053 创建完整示例服务器：cmd/server/main.go（包含连接管理、消息处理、心跳、监控的完整示例）
- [ ] T054 创建完整示例客户端：cmd/client/main.go（包含连接、重连、发送消息、WaitAck 的完整示例）
- [ ] T055 [P] 创建 Echo 示例：examples/echo/（服务器 echo 回客户端消息，基于 quickstart.md）
- [ ] T056 [P] 创建广播示例：examples/broadcast/（服务器向所有客户端广播事件）
- [ ] T057 [P] 创建回调示例：examples/callback/（演示 AsyncSend 和 Promise 机制）

### 文档和配置

- [ ] T058 [P] 创建 README.md：项目说明、特性列表、安装指南、快速开始链接
- [ ] T059 [P] 创建 Go 文档注释：为所有公共 API 添加文档注释（pkg/ 下的所有导出函数和类型）
- [ ] T060 [P] 实现配置验证：ServerConfig 和 ClientConfig 的 Validate() 方法（检查必填字段、范围）
- [ ] T061 [P] 实现日志级别配置：支持通过环境变量 LOG_LEVEL 或配置调整日志级别（DEBUG/INFO/WARN/ERROR）

### 错误处理和优雅关闭

- [ ] T062 [P] 错误处理标准化：确保所有错误使用 pkg/errors 中定义的错误类型，提供清晰错误信息
- [ ] T063 [P] 实现优雅关闭：Server.Stop 和 Client.Disconnect 的完整实现（等待 goroutine 完成，超时控制）

### 验证和优化

- [ ] T064 创建 quickstart 验证脚本：scripts/test-quickstart.sh（自动运行 quickstart.md 中的示例，验证输出）
- [ ] T065 [P] 性能分析：使用 pprof 分析内存和 CPU 使用，识别热点路径
- [ ] T066 [P] 性能优化：优化识别出的热点（如减少内存分配、优化锁竞争）

---

## 依赖关系和执行顺序

### 阶段依赖关系

```
阶段 1（设置）
     ↓
阶段 2（基础）← 阻塞所有用户故事
     │
     ├─────────────┬─────────────┐
     │             │             │
     ▼             ▼             ▼
   US1 (P1)     US2 (P1)     US3 (P2)
   [MVP核心]    [依赖US1]    [可选特性]
     │             │             │
     └─────────────┴─────────────┘
                   ↓
            阶段 6（完善）
```

**说明**：
- **阶段 1-2**：必须顺序完成，是所有工作的基础
- **US1（阶段 3）**：MVP 核心，包含连接 + 心跳，必须首先完成
- **US2（阶段 4）**：依赖 US1（需要连接已建立），实现消息传输和回调
- **US3（阶段 5）**：可选高级特性，可以与 US2 并行开发（但建议在 US2 后）
- **阶段 6**：所有用户故事完成后的完善工作

### 关键路径（MVP）

```
T001-T007（设置）→ T008-T019（基础）→ T020-T034（US1）→ T053-T054（示例）

最小 MVP = 34 个任务
```

### 用户故事依赖关系（优化后）

**US1 (P1)** - 15 个任务：
- 无前置依赖（仅依赖阶段 2 基础）
- **包含心跳机制**（对齐用户 Phase 2 路线图）
- 是 US2 的前置依赖

**US2 (P1)** - 12 个任务：
- 依赖 US1（需要连接已建立）
- 部分任务（T035-T037 Dispatcher、T040-T041 Promise）可在 US1 期间并行开发

**US3 (P2)** - 3 个任务：
- 可选特性，不阻塞 MVP
- 可以在 US2 完成后或与 US2 并行

### 并行机会（优化后）

#### 阶段 1（设置）- 全部可并行：

```bash
并行组：T003, T004, T005, T006, T007
顺序：T001 → T002 → 并行组
```

#### 阶段 2（基础）：

```bash
并行组 1（Protobuf 定义）：T008, T009, T010
顺序：T011（生成代码）必须在并行组 1 之后

并行组 2（基础模块）：T012, T013, T014, T015, T016, T017
顺序：T018-T019（SessionManager）必须在 T011 之后
```

#### 阶段 3（US1）：

```bash
并行组 1（配置）：T020, T021, T026
并行组 2（心跳）：T030, T031（需要连接建立后）
并行组 3（查询接口）：T033, T034

顺序（服务器）：T022 → T023 → T024 → T025
顺序（客户端）：T027 → T028 → T029
顺序（心跳）：T030-T031 → T032
```

#### 阶段 4（US2）：

```bash
可提前并行（在 US1 期间）：
- T035 (Dispatcher 核心)
- T037 (MessageHandler 接口)
- T040 (Promise 结构)

必须在 US1 后：
- T036, T038-T039, T041-T046

并行组（US1 完成后）：T038, T039, T040, T044, T045
```

#### 阶段 5（US3）：

```bash
全部可并行：T047, T048, T049
```

#### 阶段 6（完善）：

```bash
并行组 1（监控）：T050, T051, T052
并行组 2（示例）：T055, T056, T057（需 T053-T054 后）
并行组 3（文档配置）：T058, T059, T060, T061, T062, T063
并行组 4（优化）：T065, T066

顺序：T053-T054（完整示例）→ T055-T057（专项示例）
```

---

## 实现策略

### MVP 优先（US1 - 连接 + 心跳）

**任务范围**：T001-T034（34 个任务）

**交付物**：
- ✅ 基于 QUIC 的安全连接
- ✅ TLS 1.3 加密
- ✅ 客户端自动重连（指数退避）
- ✅ 心跳检测（15s 间隔，3 次超时清理）
- ✅ 基本监控钩子

**验证**：
1. 启动服务器和客户端，验证连接建立
2. 断开网络，验证自动重连
3. 停止客户端心跳，验证服务器在 45s 后清理

**时间估算**：
- 单人：2-3 周
- 双人并行：1.5-2 周

### 增量交付

**第 1 次交付（MVP）**：
- 阶段 1 + 2 + 3（T001-T034）
- 演示：连接、重连、心跳

**第 2 次交付（核心功能）**：
- + 阶段 4（T035-T046）
- 演示：单播、广播、异步回调

**第 3 次交付（生产就绪）**：
- + 阶段 5 + 6（T047-T066）
- 演示：完整监控、文档、示例

### 并行团队策略

**2 人团队**：
1. **阶段 1-2**：一起完成（2-3 天）
2. **阶段 3（US1）**：
   - 开发者 A：服务器端（T021-T025, T031-T032）
   - 开发者 B：客户端（T026-T030）+ TLS（T020）
3. **阶段 4（US2）**：
   - 开发者 A：Dispatcher + Promise（T035-T041）
   - 开发者 B：单播广播 + 客户端发送（T038-T039, T044-T046）
4. **阶段 5-6**：分工完成（可选特性 + 文档示例）

**3 人团队**：
1. **阶段 1-2**：一起完成
2. **阶段 3（US1）**：
   - 开发者 A：服务器端
   - 开发者 B：客户端
   - 开发者 C：心跳机制 + 监控准备
3. **阶段 4（US2）**：
   - 开发者 A：Dispatcher
   - 开发者 B：Promise + 回调
   - 开发者 C：消息发送 + 弱网验证
4. **阶段 5-6**：并行完成高级特性和完善工作

---

## 备注

- **[P] 任务**：不同文件、无依赖关系，可以并行执行
- **[Story] 标签**：将任务映射到特定用户故事以便追溯
- **每个用户故事应该可独立完成和测试**
- **在每个任务或逻辑组后提交代码**
- **在任何检查点停止以独立验证故事**

## 优化亮点 ✨

### 1. 与用户路线图完美对齐 ✅

**用户提供的 3 阶段路线图**：
- **Phase 1: 基础设施**（QUIC 握手 + TLS + Protobuf）→ 阶段 2 ✅
- **Phase 2: 连接管理**（SessionManager + 重连 + 心跳）→ 阶段 3 (US1) ✅
- **Phase 3: 消息路由与回调**（Dispatcher + 请求响应 + 广播）→ 阶段 4 (US2) ✅

### 2. 任务粒度优化 ✅

- ✅ 合并了 8 个过细任务（TLS、Dispatcher、心跳）
- ✅ 移除了 12 个冗余任务（分散的示例更新、监控集成）
- ✅ 新增了 1 个关键任务（基础错误定义）

### 3. 总任务数优化 ✅

| 项目 | 原版本 | 优化后 | 改进 |
|------|--------|--------|------|
| 总任务数 | 78 | **66** | -12（减少 15%） |
| 阶段 1 | 5 | 7 | +2（更细致） |
| 阶段 2 | 11 | 12 | +1（新增错误定义） |
| 阶段 3 (US1) | 16 | **15** | -1（合并 TLS） |
| 阶段 4 (US2) | 22 | **12** | -10（合并 Dispatcher，移除示例更新） |
| 阶段 5 (US3) | 13 | **3** | -10（简化为高级监控） |
| 阶段 6 | 11 | **17** | +6（统一示例和监控集成） |

### 4. MVP 路径更清晰 ✅

**MVP = 阶段 1 + 2 + 3 = 34 个任务**（原版 32 个）

包含：
- ✅ 连接建立
- ✅ 自动重连
- ✅ **心跳机制**（新增，对齐用户需求）
- ✅ 会话管理
- ✅ 基本监控

### 5. 并行机会更多 ✅

- 原版：33 个可并行任务
- 优化后：**35 个可并行任务**
- 改进：合并任务后，更多任务可以独立并行

---

## 用户提供的任务集成

以下是用户提供的 4 个任务，已集成到优化后的任务列表中：

| 用户任务 | 集成位置 | 任务 ID | 说明 |
|---------|---------|---------|------|
| 定义 protocol.proto | 阶段 2 基础 | T008-T010 | 拆分为 frame.proto, message.proto, types.proto |
| 编写 SessionManager | 阶段 2 基础 | T018-T019 | manager.go + session.go |
| 实现客户端 DialLoop | 阶段 3 US1 | T028 | pkg/transport/client/reconnect.go |
| 封装 AsyncSend 方法 | 阶段 4 US2 | T042 | pkg/transport/server/server.go |

---

## 总任务数统计（优化后）

- **阶段 1（设置）**：7 个任务
- **阶段 2（基础）**：12 个任务
- **阶段 3（US1）**：15 个任务（包含心跳机制）
- **阶段 4（US2）**：12 个任务
- **阶段 5（US3）**：3 个任务
- **阶段 6（完善）**：17 个任务

**总计**：**66 个任务**（相比原 78 个，减少 12 个，优化 15%）

**MVP 任务数**：34 个（阶段 1-3）
**核心功能任务数**：46 个（阶段 1-4）
**完整功能任务数**：66 个（全部阶段）
