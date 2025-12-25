# 研究报告：QUIC 通信骨干网络技术决策

**功能**：001-quic-backbone-network
**创建日期**：2025-12-23
**目的**：解决实现计划中的 NEEDS CLARIFICATION 项，为 Phase 1 设计提供技术基础

## 1. 弱网测试方案

### 决策

采用 **Toxiproxy** 作为主要弱网测试工具，辅以 Go 原生的 `net` 包模拟功能进行单元测试。

### 理由

**候选方案对比**：

| 工具 | 优势 | 劣势 | 适用场景 |
|------|------|------|---------|
| **tc + netem** | 系统级控制，真实模拟 | 需要 root 权限，配置复杂，仅 Linux | 生产环境测试 |
| **Toxiproxy** | 代理模式，跨平台，易配置 API | 增加网络跳，轻微性能开销 | CI/CD 集成测试 |
| **go-replayers** | Go 原生，轻量级 | 功能有限，社区支持少 | 简单场景 |
| **自定义 Mock** | 完全控制，无外部依赖 | 开发成本高，模拟不真实 | 单元测试 |

**选择 Toxiproxy 的原因**：

1. **跨平台支持**：开发团队可能使用 macOS/Linux/Windows，Toxiproxy 在所有平台都能运行
2. **CI/CD 友好**：Docker 镜像可用，易于集成到自动化测试流程
3. **丰富的毒素类型**：
   - `latency`：添加延迟
   - `down`：完全断开连接
   - `bandwidth`：限制带宽
   - `slow_close`：慢速关闭连接
   - `timeout`：超时
   - `slicer`：分片（模拟乱序）
   - `limit_data`：限制数据量
4. **API 友好**：HTTP API 控制，易于在测试代码中动态调整网络条件
5. **生产案例**：Shopify 开源项目，被广泛使用并维护

**测试策略**：

```go
// 示例：使用 Toxiproxy 模拟 20% 丢包
func TestWeakNetwork(t *testing.T) {
    // 启动 Toxiproxy 代理服务器
    proxy := setupToxiproxy("localhost:8474") // QUIC 服务器实际端口

    // 添加 20% 丢包毒素（Toxiproxy 模拟丢包通过 slow_close + timeout 组合）
    proxy.AddToxic("latency", "latency", "downstream", 1.0, toxiproxy.Attributes{
        "latency": 50,  // 50ms 延迟
        "jitter":  20,  // ±20ms 抖动
    })

    // 连接到代理端口（而非直接连接服务器）
    client := NewClient("localhost:8080") // Toxiproxy 监听端口

    // 发送 100 条消息，验证全部送达
    for i := 0; i < 100; i++ {
        err := client.SendMessage(payload)
        assert.NoError(t, err)
    }

    // 验证最终送达率
    assert.Equal(t, 100, receivedCount)
}
```

**辅助方案**：

对于单元测试，使用 Go 的 `net` 包自定义 `net.Conn` 包装器，模拟延迟和错误：

```go
type FlakyConn struct {
    net.Conn
    dropRate float64 // 丢包率
}

func (fc *FlakyConn) Write(b []byte) (int, error) {
    if rand.Float64() < fc.dropRate {
        return 0, errors.New("simulated packet loss")
    }
    return fc.Conn.Write(b)
}
```

### 备选方案

如果 Toxiproxy 在特定环境中无法使用（如受限的生产环境），可降级到 **tc + netem**（Linux）或 **Network Link Conditioner**（macOS）。

## 2. 回调管理策略

### 决策

采用 **超时驱动的自动清理 + 容量上限保护** 策略，具体方案：

1. **超时清理**：每个 Promise 设置独立的超时定时器（默认 30 秒），超时后自动清理并通知业务层
2. **容量上限**：Promise Map 容量上限设为 **50,000 条**，达到上限时拒绝新的 WaitAck 消息（返回错误）
3. **优雅降级**：达到 80% 容量（40,000 条）时触发警告日志和监控告警

### 理由

**问题分析**：

Promise Map 的风险在于：
- 客户端崩溃/离线导致回调永远不会返回
- 慢客户端处理时间过长
- 恶意客户端攻击（发送大量需要回调的消息但不响应）

**方案对比**：

| 方案 | 优势 | 劣势 | 适用场景 |
|------|------|------|---------|
| **超时清理** | 内存可控，符合业务语义 | 需要为每个 Promise 维护定时器 | 所有场景 |
| **LRU 淘汰** | 简单，自动淘汰最旧条目 | 可能淘汰仍在等待的有效 Promise | 不适合本场景 |
| **固定 TTL** | 实现简单 | 无法区分快慢场景，灵活性差 | 简单场景 |
| **无限制** | 实现简单 | 内存泄漏风险高，不可接受 | ❌ 不可行 |

**选择超时清理 + 容量上限的原因**：

1. **符合业务语义**：FR-018 明确要求"超时后触发超时处理逻辑"，超时清理直接对应业务需求
2. **内存安全**：容量上限（50,000 条）提供最后防线，假设每个 Promise 占用 200 字节，最大内存占用约 10MB
3. **可观测**：80% 阈值告警让运维团队提前介入，而非等到问题发生
4. **灵活性**：超时时间可配置（默认 30s），不同业务场景可调整

**实现设计**：

```go
type PromiseManager struct {
    promises  sync.Map                      // MsgID -> *Promise
    count     atomic.Int64                  // 当前 Promise 数量
    maxCount  int64                         // 容量上限（50000）
    warnCount int64                         // 警告阈值（40000）
    metrics   *Metrics                      // 监控指标
}

type Promise struct {
    MsgID       string
    RespChan    chan Response
    Timeout     time.Duration
    Timer       *time.Timer    // 超时定时器
    CreatedAt   time.Time
}

func (pm *PromiseManager) Create(msgID string, timeout time.Duration) (*Promise, error) {
    // 检查容量
    current := pm.count.Load()
    if current >= pm.maxCount {
        pm.metrics.PromiseRejections.Inc()
        return nil, ErrPromiseCapacityFull
    }

    if current >= pm.warnCount {
        log.Warn("Promise Map达到警告阈值", "current", current, "max", pm.maxCount)
        pm.metrics.PromiseWarnTriggered.Inc()
    }

    // 创建 Promise
    promise := &Promise{
        MsgID:     msgID,
        RespChan:  make(chan Response, 1),
        Timeout:   timeout,
        CreatedAt: time.Now(),
    }

    // 设置超时定时器
    promise.Timer = time.AfterFunc(timeout, func() {
        pm.Cleanup(msgID, ErrTimeout)
    })

    pm.promises.Store(msgID, promise)
    pm.count.Add(1)

    return promise, nil
}

func (pm *PromiseManager) Cleanup(msgID string, reason error) {
    if val, ok := pm.promises.LoadAndDelete(msgID); ok {
        promise := val.(*Promise)
        promise.Timer.Stop()

        // 通知业务层（超时或错误）
        select {
        case promise.RespChan <- Response{Error: reason}:
        default:
        }
        close(promise.RespChan)

        pm.count.Add(-1)
        pm.metrics.PromiseCleanups.Inc()

        if reason == ErrTimeout {
            pm.metrics.PromiseTimeouts.Inc()
        }
    }
}
```

**容量计算**：

- 单个 Promise 内存占用：约 200 字节（结构体 + channel + 定时器）
- 上限 50,000 条 × 200 字节 = 10MB
- 性能考虑：`sync.Map` 在高并发读多写少场景下性能优于带锁的普通 map
- 规范要求支持 10,000 并发客户端，假设 20% 客户端同时有 2 个待回调消息：10,000 × 0.2 × 2 = 4,000 条（远低于上限）

### 备选方案

如果超时定时器的 goroutine 开销成为瓶颈（不太可能），可改用 **时间轮（Timing Wheel）** 算法，将多个 Promise 的超时检查合并到一个定时器。

## 3. 监控方案

### 决策

采用 **内置简单计数器 + Prometheus 指标导出接口** 的混合方案，具体：

1. **核心指标**：内置 `atomic` 计数器实时维护关键指标（连接数、吞吐量、延迟）
2. **查询 API**：提供 `GetMetrics()` 方法返回当前指标快照
3. **Prometheus 导出**：可选的 HTTP endpoint (`/metrics`) 导出 Prometheus 格式指标
4. **结构化日志**：使用 `slog` 记录所有关键事件

### 理由

**方案对比**：

| 方案 | 优势 | 劣势 | 适用场景 |
|------|------|------|---------|
| **仅内置计数器** | 零依赖，简单高效 | 无时序数据，无可视化 | 最小化部署 |
| **强制 Prometheus** | 强大的监控生态 | 增加依赖和复杂度，不灵活 | 大型生产环境 |
| **混合方案** | 灵活，满足不同场景 | 实现复杂度稍高 | **最佳选择** |
| **仅日志** | 简单 | 不适合实时监控，查询困难 | ❌ 不满足需求 |

**选择混合方案的原因**：

1. **满足规范**：SC-010 要求"实时提供指标"，内置计数器满足此需求
2. **灵活性**：小型部署可以只用内置 API，大型生产环境可启用 Prometheus
3. **零强制依赖**：Prometheus 导出是可选功能，不依赖 Prometheus 库也能运行
4. **低开销**：`atomic` 计数器几乎零开销，不影响性能

**实现设计**：

```go
// 内置指标结构
type Metrics struct {
    // 连接相关
    ConnectedClients    atomic.Int64
    TotalConnections    atomic.Int64
    TotalDisconnects    atomic.Int64

    // 消息相关
    MessagesSent        atomic.Int64
    MessagesReceived    atomic.Int64
    MessagesFailed      atomic.Int64

    // 延迟（滑动窗口平均值，P99 通过 histogram 计算）
    latencyHistogram    *Histogram  // 自定义轻量级 histogram

    // 心跳相关
    HeartbeatTimeouts   atomic.Int64

    // 回调相关
    PromiseCreated      atomic.Int64
    PromiseCompleted    atomic.Int64
    PromiseTimeouts     atomic.Int64

    // 时间窗口（用于计算吞吐量）
    lastResetTime       atomic.Value // time.Time
}

// 获取指标快照（满足 SC-010）
func (m *Metrics) GetSnapshot() MetricsSnapshot {
    now := time.Now()
    lastReset := m.lastResetTime.Load().(time.Time)
    duration := now.Sub(lastReset).Seconds()

    return MetricsSnapshot{
        ConnectedClients:  m.ConnectedClients.Load(),
        TotalConnections:  m.TotalConnections.Load(),
        MessageThroughput: int64(float64(m.MessagesSent.Load()) / duration), // 消息/秒
        AverageLatency:    m.latencyHistogram.Mean(),
        P99Latency:        m.latencyHistogram.Percentile(0.99),
        Timestamp:         now,
    }
}

// Prometheus 导出（可选）
func (m *Metrics) PrometheusHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        snapshot := m.GetSnapshot()

        // 手动生成 Prometheus 文本格式（避免依赖 prometheus/client_golang）
        fmt.Fprintf(w, "# HELP quic_connected_clients 当前连接的客户端数量\n")
        fmt.Fprintf(w, "# TYPE quic_connected_clients gauge\n")
        fmt.Fprintf(w, "quic_connected_clients %d\n", snapshot.ConnectedClients)

        fmt.Fprintf(w, "# HELP quic_message_throughput 消息吞吐量（消息/秒）\n")
        fmt.Fprintf(w, "# TYPE quic_message_throughput gauge\n")
        fmt.Fprintf(w, "quic_message_throughput %d\n", snapshot.MessageThroughput)

        // ... 其他指标
    })
}
```

**轻量级 Histogram 实现**：

使用固定桶（buckets）的简化 histogram，避免依赖外部库：

```go
type Histogram struct {
    buckets []atomic.Int64  // 预定义延迟桶：[0-10ms, 10-50ms, 50-100ms, 100-200ms, 200ms+]
    count   atomic.Int64
    sum     atomic.Int64    // 总延迟（微秒）
}

func (h *Histogram) Observe(latencyMs int64) {
    h.count.Add(1)
    h.sum.Add(latencyMs * 1000) // 转为微秒存储

    // 更新对应的桶
    if latencyMs < 10 {
        h.buckets[0].Add(1)
    } else if latencyMs < 50 {
        h.buckets[1].Add(1)
    } // ...
}

func (h *Histogram) Percentile(p float64) int64 {
    // 简化的百分位计算（基于桶边界估算）
    // 实际实现可以使用更精确的算法（如 t-digest）
}
```

**日志方案**：

使用 Go 1.21+ 的 `log/slog` 包（标准库）：

```go
slog.Info("客户端连接",
    "client_id", clientID,
    "remote_addr", conn.RemoteAddr().String(),
    "timestamp", time.Now())

slog.Warn("心跳超时",
    "client_id", clientID,
    "timeout_count", timeoutCount,
    "will_disconnect", timeoutCount >= 3)
```

### 备选方案

如果项目已有 Prometheus 监控基础设施，可以直接使用 `github.com/prometheus/client_golang` 库，简化 Prometheus 集成。

## 4. QUIC 实现最佳实践

### 决策

基于 `quic-go` 库的生产最佳实践，采用以下配置和模式：

### 关键配置

**1. 连接配置**

```go
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    NextProtos:   []string{"quic-backbone-v1"},  // ALPN 标识
    MinVersion:   tls.VersionTLS13,               // 强制 TLS 1.3
}

quicConfig := &quic.Config{
    MaxIdleTimeout:                 60 * time.Second,  // 空闲超时（匹配心跳周期）
    MaxIncomingStreams:             1000,              // 每连接最大并发流数
    MaxIncomingUniStreams:          100,               // 单向流数量
    KeepAlivePeriod:                0,                 // 禁用 QUIC 层 keep-alive（使用应用层心跳）
    InitialStreamReceiveWindow:     512 * 1024,        // 512KB 接收窗口
    MaxStreamReceiveWindow:         6 * 1024 * 1024,   // 6MB 最大窗口
    InitialConnectionReceiveWindow: 1024 * 1024,       // 1MB 连接窗口
    MaxConnectionReceiveWindow:     15 * 1024 * 1024,  // 15MB 最大窗口
}
```

**配置理由**：

- `MaxIdleTimeout = 60s`：略大于心跳周期（45s），避免心跳检测前 QUIC 层就断开
- `KeepAlivePeriod = 0`：避免 QUIC 层和应用层心跳冲突
- 接收窗口：根据消息大小（< 1MB）和吞吐量要求调整，平衡内存和性能

**2. 流管理**

```go
// 每个消息使用独立的 QUIC 流（Stream）
func (c *Client) SendMessage(msg *Message) error {
    stream, err := c.conn.OpenStreamSync(context.Background())
    if err != nil {
        return err
    }
    defer stream.Close()

    // 写入消息
    data, _ := proto.Marshal(msg)
    _, err = stream.Write(data)
    if err != nil {
        return err
    }

    // 如果需要响应，等待读取
    if msg.WaitAck {
        resp, err := io.ReadAll(stream)
        // 处理响应...
    }

    return nil
}
```

**模式选择**：
- **每消息一流**：简单、隔离性好、避免消息边界问题
- 备选：**连接级长流 + 帧分割**（更高效但实现复杂，本期不采用）

**3. 并发连接管理**

```go
// 服务器端：使用 goroutine per connection 模式
func (s *Server) handleConnection(conn quic.Connection) {
    defer conn.CloseWithError(0, "connection closed")

    for {
        stream, err := conn.AcceptStream(context.Background())
        if err != nil {
            return // 连接关闭
        }

        // 每个流用独立 goroutine 处理
        go s.handleStream(conn, stream)
    }
}

func (s *Server) handleStream(conn quic.Connection, stream quic.Stream) {
    defer stream.Close()

    data, err := io.ReadAll(stream)
    // 解码消息、分发到 Dispatcher...
}
```

**理由**：
- `goroutine per connection` 是 Go 的自然并发模型
- 支持 10,000 连接 × 每连接 1 个 goroutine = 10,000 goroutines（Go 可轻松处理）
- 每个流独立 goroutine：避免单连接内的流阻塞（head-of-line blocking 在应用层）

**4. 错误处理和重试**

```go
// 客户端重连状态机
type ClientState int

const (
    StateIdle ClientState = iota
    StateConnecting
    StateConnected
)

func (c *Client) reconnectLoop() {
    backoff := 1 * time.Second
    maxBackoff := 60 * time.Second

    for {
        c.state = StateConnecting
        conn, err := quic.DialAddr(c.serverAddr, c.tlsConfig, c.quicConfig)

        if err != nil {
            log.Error("连接失败，重试中", "backoff", backoff, "error", err)
            time.Sleep(backoff)
            backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
            continue
        }

        c.conn = conn
        c.state = StateConnected
        backoff = 1 * time.Second // 重置退避时间

        // 启动心跳和消息处理
        go c.heartbeatLoop()
        go c.receiveLoop()

        // 阻塞直到连接断开
        <-c.disconnectChan
        c.state = StateIdle
    }
}
```

### 性能优化

1. **复用 TLS 配置**：避免每次连接都加载证书
2. **零拷贝**：使用 `io.Copy` 而非多次 `Read`/`Write`
3. **批量发送**（可选）：对于广播场景，可考虑合并小消息（本期不实现）
4. **连接池**（不适用）：由于是长连接场景，不需要连接池

### 已知陷阱

1. ❌ **不要在同一流上混合心跳和数据消息**：会导致解析混乱
   - ✅ 解决：心跳使用独立的单向流（`UniStream`）
2. ❌ **不要忘记关闭流**：QUIC 流不会自动关闭
   - ✅ 解决：使用 `defer stream.Close()`
3. ❌ **不要阻塞 `AcceptStream`**：会阻止新连接接入
   - ✅ 解决：每个流用独立 goroutine

## 5. Protobuf 消息设计

### 决策

采用分层协议设计，包含**帧（Frame）**和**消息（Message）**两层：

- **帧层**：定义通信原语（Ping/Pong/Data）
- **消息层**：定义业务消息结构

### Protobuf 定义

**文件结构**：
```
contracts/protobuf/
├── frame.proto      # 帧定义
├── message.proto    # 消息定义
└── types.proto      # 公共类型
```

**frame.proto**：

```protobuf
syntax = "proto3";

package quic.backbone.v1;

option go_package = "github.com/voilet/QuicFlow/pkg/protocol;protocol";

// 帧类型
enum FrameType {
  FRAME_TYPE_UNSPECIFIED = 0;
  FRAME_TYPE_PING        = 1;  // 心跳请求
  FRAME_TYPE_PONG        = 2;  // 心跳响应
  FRAME_TYPE_DATA        = 3;  // 数据消息
  FRAME_TYPE_ACK         = 4;  // 确认帧
}

// 顶层帧结构（流的第一个消息）
message Frame {
  FrameType type        = 1;
  bytes     payload     = 2;  // 根据 type 包含不同的消息
  int64     timestamp   = 3;  // Unix 毫秒时间戳
}

// 心跳请求
message PingFrame {
  string client_id = 1;
}

// 心跳响应
message PongFrame {
  int64 server_time = 1;  // 服务器时间戳（用于时钟同步检查）
}
```

**message.proto**：

```protobuf
syntax = "proto3";

package quic.backbone.v1;

option go_package = "github.com/voilet/QuicFlow/pkg/protocol;protocol";

// 数据消息
message DataMessage {
  string msg_id       = 1;  // 消息唯一 ID（UUID）
  string sender_id    = 2;  // 发送方 ID
  string receiver_id  = 3;  // 接收方 ID（空表示广播）
  MessageType type    = 4;  // 消息类型
  bytes payload       = 5;  // 业务数据（JSON 或 Protobuf）
  bool wait_ack       = 6;  // 是否需要确认
  int64 timestamp     = 7;  // 发送时间戳
}

// 消息类型（业务层自定义）
enum MessageType {
  MESSAGE_TYPE_UNSPECIFIED = 0;
  MESSAGE_TYPE_COMMAND     = 1;  // 指令
  MESSAGE_TYPE_EVENT       = 2;  // 事件
  MESSAGE_TYPE_QUERY       = 3;  // 查询
  MESSAGE_TYPE_RESPONSE    = 4;  // 响应
}

// 确认消息
message AckMessage {
  string msg_id    = 1;  // 对应的消息 ID
  AckStatus status = 2;  // 执行状态
  bytes result     = 3;  // 执行结果（可选）
  string error     = 4;  // 错误信息（如果失败）
}

// 确认状态
enum AckStatus {
  ACK_STATUS_UNSPECIFIED = 0;
  ACK_STATUS_SUCCESS     = 1;  // 成功
  ACK_STATUS_FAILURE     = 2;  // 失败
  ACK_STATUS_TIMEOUT     = 3;  // 超时
}
```

**types.proto**：

```protobuf
syntax = "proto3";

package quic.backbone.v1;

option go_package = "github.com/voilet/QuicFlow/pkg/protocol;protocol";

// 客户端信息
message ClientInfo {
  string client_id      = 1;
  string remote_addr    = 2;
  int64  connected_at   = 3;
  int64  last_heartbeat = 4;
}
```

### 版本兼容性策略

1. **向后兼容**：新增字段使用更大的字段编号，旧客户端忽略未知字段
2. **版本协商**：在 TLS ALPN 中指定协议版本（如 `quic-backbone-v1`）
3. **废弃字段**：标记为 `deprecated` 而非删除

### 使用示例

```go
// 发送数据消息
msg := &protocol.DataMessage{
    MsgId:      uuid.New().String(),
    SenderId:   "server-1",
    ReceiverId: clientID,
    Type:       protocol.MessageType_MESSAGE_TYPE_COMMAND,
    Payload:    []byte(`{"action": "reboot"}`),
    WaitAck:    true,
    Timestamp:  time.Now().UnixMilli(),
}

frame := &protocol.Frame{
    Type:      protocol.FrameType_FRAME_TYPE_DATA,
    Payload:   mustMarshal(msg),
    Timestamp: time.Now().UnixMilli(),
}

data, _ := proto.Marshal(frame)
stream.Write(data)
```

## 6. 并发架构

### 决策

采用 **Goroutine-per-connection + Channel-based message passing** 模式：

1. **连接并发**：每个 QUIC 连接一个 goroutine（`handleConnection`）
2. **流并发**：每个 QUIC 流一个 goroutine（`handleStream`）
3. **会话管理**：使用 `sync.Map` 存储 ClientSession（读多写少场景）
4. **消息分发**：Dispatcher 通过 channel 接收消息，worker pool 处理

### 架构图

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ QUIC Connection
┌──────▼──────────────────────────────────────┐
│  Server                                     │
│  ├─ Listener goroutine                      │
│  │   └─ Accept connections                  │
│  │                                           │
│  ├─ handleConnection (goroutine per conn)   │
│  │   ├─ AcceptStream loop                   │
│  │   └─ handleStream (goroutine per stream) │
│  │       ├─ Decode Frame                    │
│  │       └─ Route to Dispatcher              │
│  │                                           │
│  ├─ SessionManager                           │
│  │   ├─ sync.Map (clientID -> Session)      │
│  │   └─ Heartbeat checker (1 goroutine)     │
│  │                                           │
│  └─ Dispatcher                               │
│      ├─ Message channel (buffered 1000)     │
│      └─ Worker pool (10 goroutines)         │
│          └─ Call MessageHandler              │
└─────────────────────────────────────────────┘
```

### SessionManager 实现

```go
type SessionManager struct {
    sessions       sync.Map                    // clientID -> *ClientSession
    count          atomic.Int64
    heartbeatTick  *time.Ticker                // 心跳检查定时器（每 5 秒检查一次）
    hooks          EventHooks                  // 状态钩子
}

type ClientSession struct {
    ClientID      string
    Conn          quic.Connection
    RemoteAddr    string
    ConnectedAt   time.Time
    LastHeartbeat atomic.Value                 // time.Time（原子更新）
    TimeoutCount  atomic.Int32                 // 连续超时次数
    mu            sync.RWMutex
}

// 心跳检查（单独的 goroutine）
func (sm *SessionManager) heartbeatChecker() {
    for range sm.heartbeatTick.C {
        now := time.Now()

        sm.sessions.Range(func(key, value interface{}) bool {
            session := value.(*ClientSession)
            lastHB := session.LastHeartbeat.Load().(time.Time)

            // 检查是否超过 15 秒未收到心跳
            if now.Sub(lastHB) > 15*time.Second {
                count := session.TimeoutCount.Add(1)

                if count >= 3 {
                    // 连续 3 次超时，清理会话
                    sm.Remove(session.ClientID)
                    session.Conn.CloseWithError(0, "heartbeat timeout")

                    if sm.hooks.OnHeartbeatTimeout != nil {
                        sm.hooks.OnHeartbeatTimeout(session.ClientID)
                    }
                }
            } else {
                // 重置超时计数
                session.TimeoutCount.Store(0)
            }

            return true // 继续迭代
        })
    }
}
```

### Dispatcher 实现

```go
type Dispatcher struct {
    handlers     sync.Map                      // MessageType -> MessageHandler
    msgChan      chan *IncomingMessage         // 缓冲 1000 条消息
    workerCount  int                           // worker 数量（默认 10）
    wg           sync.WaitGroup
}

type IncomingMessage struct {
    ClientID string
    Message  *protocol.DataMessage
    Stream   quic.Stream  // 用于发送响应
}

type MessageHandler interface {
    OnMessage(ctx context.Context, msg *IncomingMessage) (*protocol.AckMessage, error)
}

// 启动 worker pool
func (d *Dispatcher) Start() {
    for i := 0; i < d.workerCount; i++ {
        d.wg.Add(1)
        go d.worker(i)
    }
}

func (d *Dispatcher) worker(id int) {
    defer d.wg.Done()

    for msg := range d.msgChan {
        // 根据消息类型查找 handler
        handler, ok := d.handlers.Load(msg.Message.Type)
        if !ok {
            log.Warn("未找到 handler", "type", msg.Message.Type)
            continue
        }

        // 调用 handler
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        ack, err := handler.(MessageHandler).OnMessage(ctx, msg)
        cancel()

        // 如果需要 Ack，发送响应
        if msg.Message.WaitAck {
            d.sendAck(msg.Stream, msg.Message.MsgId, ack, err)
        }
    }
}

// 分发消息（非阻塞）
func (d *Dispatcher) Dispatch(msg *IncomingMessage) error {
    select {
    case d.msgChan <- msg:
        return nil
    default:
        return ErrDispatcherFull // channel 满了，拒绝新消息
    }
}
```

### 并发模型选择理由

1. **Goroutine-per-connection**：
   - Go 的 goroutine 非常轻量（2KB 栈），10,000 个连接仅占用 20MB
   - 代码简单清晰，符合 Go 惯例
   - 自然地处理连接生命周期

2. **sync.Map vs mutex map**：
   - SessionManager 是典型的读多写少场景（连接建立后很少修改）
   - `sync.Map` 在此场景下性能优于带锁的 `map`
   - 支持无锁的 `Range` 遍历（心跳检查）

3. **Worker pool vs goroutine-per-message**：
   - 消息处理可能包含 I/O 操作（如数据库查询），不适合无限制创建 goroutine
   - Worker pool 限制并发处理数量，避免资源耗尽
   - Buffered channel (1000) 提供短暂的消息缓冲，平滑流量峰值

4. **Channel vs callback**：
   - Channel 是 Go 的惯用通信方式，类型安全
   - 解耦 Dispatcher 和 Handler，便于测试

### 性能考虑

- **10,000 连接** × 2KB goroutine = 20MB
- **1,000 消息缓冲** × 1KB/消息 = 1MB
- **50,000 Promise** × 200B = 10MB
- **总内存占用**：约 100MB（不包括消息 payload）

符合规范要求的 < 8GB 内存限制。

## 总结

所有 NEEDS CLARIFICATION 项已解决：

| 项目 | 决策 | 输出位置 |
|------|------|---------|
| 弱网测试工具 | Toxiproxy + 自定义 Mock | 第 1 节 |
| 回调管理策略 | 超时清理 + 容量上限（50,000） | 第 2 节 |
| 指标采集方式 | 内置计数器 + 可选 Prometheus | 第 3 节 |
| QUIC 最佳实践 | quic-go 配置优化、流管理模式 | 第 4 节 |
| Protobuf 消息设计 | 分层协议（Frame + Message） | 第 5 节 + contracts/ |
| 并发架构 | Goroutine-per-connection + Worker pool | 第 6 节 |

**下一步**：进入 **Phase 1** 设计阶段，生成 `data-model.md`、`contracts/`、`quickstart.md`。
