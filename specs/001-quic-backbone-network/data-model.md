# 数据模型：QUIC 通信骨干网络

**功能**：001-quic-backbone-network
**创建日期**：2025-12-23
**输入**：功能规范中的关键实体 + 研究阶段的设计决策

## 概述

本文档定义 QUIC 通信骨干网络的核心数据实体及其关系。系统采用纯内存存储，无持久化需求。

## 核心实体

### 1. ClientSession（客户端会话）

**描述**：代表一个已连接的客户端及其元数据。

**字段**：

| 字段名 | 类型 | 必填 | 描述 | 验证规则 |
|--------|------|------|------|---------|
| ClientID | string | 是 | 客户端唯一标识符 | 非空，业务层生成并保证唯一性 |
| Conn | quic.Connection | 是 | QUIC 连接对象 | 非 nil |
| RemoteAddr | string | 是 | 客户端远程地址 | IPv4/IPv6 格式 |
| ConnectedAt | time.Time | 是 | 连接建立时间 | UTC 时间 |
| LastHeartbeat | time.Time | 是 | 最后心跳时间 | UTC 时间，原子更新 |
| TimeoutCount | int32 | 是 | 连续超时次数 | 0-3，达到 3 触发清理 |
| State | ClientState | 是 | 连接状态 | Idle/Connecting/Connected |

**状态转换**：

```
Idle ─────► Connecting ─────► Connected
 ▲                                │
 └────────────────────────────────┘
         (断开/超时)
```

**索引**：
- 主键：ClientID
- 存储结构：`sync.Map`（clientID -> *ClientSession）

**生命周期**：
- 创建：客户端连接成功后创建
- 更新：心跳到达时更新 `LastHeartbeat`
- 删除：心跳超时（连续 3 次 × 15s = 45s）或主动断开

**并发控制**：
- `LastHeartbeat` 使用 `atomic.Value` 无锁更新
- `TimeoutCount` 使用 `atomic.Int32`
- 整个 Session 通过 `sync.Map` 管理，支持并发读写

### 2. Message（消息）

**描述**：代表一条通信消息，支持单播和广播。

**字段**：

| 字段名 | 类型 | 必填 | 描述 | 验证规则 |
|--------|------|------|------|---------|
| MsgID | string | 是 | 消息唯一标识符 | UUID v4 格式 |
| SenderID | string | 是 | 发送方 ID | 非空 |
| ReceiverID | string | 否 | 接收方 ID | 空表示广播 |
| Type | MessageType | 是 | 消息类型 | 枚举值（Command/Event/Query/Response） |
| Payload | []byte | 是 | 业务数据 | 大小 ≤ 1MB |
| WaitAck | bool | 是 | 是否需要确认 | true/false |
| Timestamp | int64 | 是 | 发送时间戳 | Unix 毫秒时间戳 |

**类型枚举**：

```go
type MessageType int32

const (
    MessageType_COMMAND  MessageType = 1  // 指令（需要执行操作）
    MessageType_EVENT    MessageType = 2  // 事件（通知性质）
    MessageType_QUERY    MessageType = 3  // 查询（请求数据）
    MessageType_RESPONSE MessageType = 4  // 响应（返回结果）
)
```

**验证规则**：
- `MsgID`：必须全局唯一，由发送方生成（UUID）
- `Payload`：大小限制 1MB（1,048,576 字节），超过则拒绝
- `ReceiverID`：
  - 单播：必须存在于 SessionManager 中
  - 广播：为空字符串
- `WaitAck`：
  - true：消息发送后等待 `AckMessage`，超时时间可配置（默认 30s）
  - false：仅发送，不等待响应

**生命周期**：
- 创建：业务层调用 `SendMessage()` 或 `Broadcast()` 时创建
- 传输：通过 QUIC 流发送
- 销毁：发送完成后立即释放（无持久化）

**关系**：
- 与 `CallbackRecord` 关联：如果 `WaitAck = true`，创建对应的 CallbackRecord

### 3. CallbackRecord（回调记录）

**描述**：代表一个等待中的消息回调，用于 Promise 模式。

**字段**：

| 字段名 | 类型 | 必填 | 描述 | 验证规则 |
|--------|------|------|------|---------|
| MsgID | string | 是 | 对应的消息 ID | 必须存在的 Message.MsgID |
| RespChan | chan Response | 是 | 响应通道 | buffered channel (cap=1) |
| Timeout | time.Duration | 是 | 超时时间 | 默认 30 秒，可配置 |
| Timer | *time.Timer | 是 | 超时定时器 | 超时后触发清理 |
| CreatedAt | time.Time | 是 | 创建时间 | UTC 时间 |

**Response 结构**：

```go
type Response struct {
    MsgID  string
    Status AckStatus       // SUCCESS / FAILURE / TIMEOUT
    Result []byte          // 执行结果数据（可选）
    Error  error           // 错误信息（如果失败）
}

type AckStatus int32

const (
    AckStatus_SUCCESS AckStatus = 1  // 执行成功
    AckStatus_FAILURE AckStatus = 2  // 执行失败
    AckStatus_TIMEOUT AckStatus = 3  // 等待超时
)
```

**存储结构**：
- `sync.Map`（MsgID -> *CallbackRecord）
- 计数器：`atomic.Int64` 跟踪当前 Promise 数量
- 容量限制：50,000 条（达到上限拒绝新请求）

**生命周期**：
- 创建：`WaitAck = true` 的消息发送时创建
- 清理：以下情况之一触发
  1. 收到 `AckMessage`（正常响应）
  2. 超时定时器触发（默认 30s）
  3. 连接断开（批量清理该客户端的所有 Promise）

**清理机制**：
- 超时清理：`time.AfterFunc()` 定时器，超时后自动调用 `Cleanup(msgID, ErrTimeout)`
- 容量保护：达到 40,000 条（80% 阈值）时触发警告日志

**并发安全**：
- `sync.Map` 保证并发读写安全
- `RespChan` 使用 buffered channel 避免阻塞
- 清理时使用 `LoadAndDelete()` 原子操作

### 4. HeartbeatRecord（心跳记录）

**描述**：跟踪客户端心跳状态（实际上与 ClientSession 合并，此处作为逻辑实体说明）。

**字段**（嵌入在 ClientSession 中）：

| 字段名 | 类型 | 描述 |
|--------|------|------|
| LastHeartbeat | time.Time | 最后收到心跳的时间 |
| TimeoutCount | int32 | 连续超时次数（0-3） |

**心跳周期**：
- 客户端发送间隔：15 秒
- 服务器检查间隔：5 秒（通过定时器检查所有会话）
- 超时阈值：45 秒（3 × 15 秒）

**心跳流程**：

```
Client                    Server
  │                         │
  ├─────────────────────────┤
  │ Ping (clientID)         │
  │                         │
  │         Pong            │
  │◄────────────────────────┤
  │                         │
  │ (更新 LastHeartbeat,     │
  │  重置 TimeoutCount)      │
  │                         │
  │ ... 15 秒后 ...         │
  │                         │
  ├─────────────────────────┤
  │ Ping                    │
```

如果服务器在 45 秒内未收到 Ping：
1. TimeoutCount 增加到 3
2. 触发 `OnHeartbeatTimeout` 钩子
3. 关闭 QUIC 连接
4. 从 SessionManager 中删除

### 5. Frame（帧）

**描述**：QUIC 流上的传输单元，封装不同类型的消息。

**字段**：

| 字段名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| Type | FrameType | 是 | 帧类型 |
| Payload | []byte | 是 | 具体消息内容 |
| Timestamp | int64 | 是 | 时间戳 |

**帧类型枚举**：

```go
type FrameType int32

const (
    FrameType_PING FrameType = 1  // 心跳请求（包含 PingFrame）
    FrameType_PONG FrameType = 2  // 心跳响应（包含 PongFrame）
    FrameType_DATA FrameType = 3  // 数据消息（包含 DataMessage）
    FrameType_ACK  FrameType = 4  // 确认消息（包含 AckMessage）
)
```

**编解码**：
- 序列化：Protocol Buffers
- 传输：每个 Frame 对应一个 QUIC Stream
- 帧边界：通过 Stream 的自然边界（每个 Frame 使用独立 Stream）

## 实体关系图

```
┌──────────────────┐
│  ClientSession   │
│  ──────────────  │
│  ClientID (PK)   │
│  Conn            │
│  RemoteAddr      │
│  ConnectedAt     │
│  LastHeartbeat   │────┐
│  TimeoutCount    │    │ 心跳检查
└────────┬─────────┘    │
         │              │
         │ 1:N          ▼
         │         ┌────────────────┐
         │         │ HeartbeatTimer │ (定时检查器)
         │         └────────────────┘
         │
         │ 发送/接收
         │
         ▼
┌────────────────┐         1:1          ┌──────────────────┐
│    Message     │ ───────(WaitAck)───► │ CallbackRecord   │
│  ────────────  │                      │  ──────────────  │
│  MsgID (PK)    │                      │  MsgID (FK)      │
│  SenderID      │                      │  RespChan        │
│  ReceiverID    │                      │  Timeout         │
│  Type          │                      │  Timer           │
│  Payload       │                      │  CreatedAt       │
│  WaitAck       │                      └──────────────────┘
│  Timestamp     │
└────────┬───────┘
         │
         │ 封装
         │
         ▼
┌────────────────┐
│     Frame      │
│  ────────────  │
│  Type          │
│  Payload       │ (包含序列化的 Message/Ping/Pong/Ack)
│  Timestamp     │
└────────────────┘
         │
         │ QUIC Stream
         │
         ▼
     [ 网络传输 ]
```

## 数据流

### 1. 消息发送流（单播 + WaitAck）

```
1. 业务层调用 SendMessage(clientID, payload, waitAck=true)
   ↓
2. 创建 Message 对象（生成 MsgID）
   ↓
3. 创建 CallbackRecord（MsgID -> chan Response）
   ↓
4. 将 Message 封装为 Frame (FrameType_DATA)
   ↓
5. 通过 QUIC Stream 发送到目标客户端
   ↓
6. 等待 Response（阻塞在 channel 或超时）
   ↓
7. 收到 AckMessage → 写入 Response channel
   或超时 → 触发 Timer cleanup
   ↓
8. 业务层获得 Response，CallbackRecord 被清理
```

### 2. 广播流（无 WaitAck）

```
1. 业务层调用 Broadcast(payload)
   ↓
2. 创建 Message 对象（ReceiverID = ""）
   ↓
3. 遍历 SessionManager.sessions (sync.Map.Range)
   ↓
4. 对每个 ClientSession：
   ├─ 将 Message 封装为 Frame
   └─ 通过 QUIC Stream 发送
   ↓
5. 不等待响应，直接返回
```

### 3. 心跳流

```
Client 端（每 15 秒）：
1. 创建 PingFrame (clientID)
   ↓
2. 封装为 Frame (FrameType_PING)
   ↓
3. 通过独立的 UniStream 发送
   ↓
4. 等待 PongFrame

Server 端（接收 Ping）：
1. 解码 Frame，识别 FrameType_PING
   ↓
2. 更新 ClientSession.LastHeartbeat
   ↓
3. 重置 ClientSession.TimeoutCount = 0
   ↓
4. 返回 PongFrame (serverTime)

Server 端（定时检查，每 5 秒）：
1. 遍历 sessions，检查 LastHeartbeat
   ↓
2. 如果 now - LastHeartbeat > 15s：
   ├─ TimeoutCount++
   └─ 如果 TimeoutCount >= 3：
       ├─ 触发 OnHeartbeatTimeout 钩子
       ├─ 关闭连接
       └─ 删除 ClientSession
```

## 容量规划

基于规范要求（10,000 并发客户端）：

| 实体 | 数量估算 | 单个大小 | 总内存占用 |
|------|----------|---------|-----------|
| ClientSession | 10,000 | ~500 B | ~5 MB |
| CallbackRecord | 20,000 (20% 客户端 × 2 消息) | ~200 B | ~4 MB |
| Message (瞬时) | ~1,000 (缓冲中) | ~1 KB | ~1 MB |
| Frame (瞬时) | ~1,000 (编码中) | ~1 KB | ~1 MB |
| **总计** | | | **~11 MB** |

加上 goroutine 和其他开销，总内存约 **100 MB**，远低于 8GB 限制。

## 持久化策略

**无持久化需求**：

- 所有数据仅存储在内存中
- 服务器重启后，客户端需要重新连接
- 历史消息不保留（规范明确"超出范围"）

如果未来需要持久化（如消息队列），可扩展以下实体：
- `MessageLog`：存储历史消息
- `SessionState`：持久化会话状态（支持服务器重启后恢复）

（本期不实现）

## 验证规则汇总

| 实体 | 字段 | 规则 | 错误处理 |
|------|------|------|---------|
| Message | Payload | ≤ 1MB | 拒绝发送，返回 `ErrPayloadTooLarge` |
| Message | ReceiverID | 存在于 SessionManager | 返回 `ErrClientNotConnected` |
| CallbackRecord | 总数量 | ≤ 50,000 | 返回 `ErrPromiseCapacityFull` |
| ClientSession | ClientID | 唯一性 | 由业务层保证 |
| Frame | Type | 有效的 FrameType | 忽略无效帧，记录日志 |

## 状态一致性

### 并发一致性保证

1. **SessionManager**：
   - 使用 `sync.Map` 保证并发安全
   - `LastHeartbeat` 使用 `atomic.Value` 无锁更新
   - 删除操作使用 `LoadAndDelete` 原子操作

2. **PromiseManager**：
   - 使用 `sync.Map` 保证并发安全
   - 计数器使用 `atomic.Int64` 原子操作
   - Timer cleanup 在 goroutine 中执行，通过 channel 通信

3. **Dispatcher**：
   - Worker pool 通过 buffered channel 协调
   - 无共享状态，handler 之间隔离

### 异常情况处理

| 异常 | 影响实体 | 处理策略 |
|------|---------|---------|
| 客户端崩溃 | ClientSession, CallbackRecord | 心跳超时清理（45s），批量清理 Promise |
| 网络中断 | Message（传输中） | QUIC 层重传，应用层重连 |
| 服务器重启 | 所有内存数据 | 客户端检测断开，自动重连 |
| Promise 泄漏 | CallbackRecord | 超时定时器自动清理 + 容量上限保护 |
| 消息积压 | Dispatcher.msgChan | Channel 满时拒绝新消息，返回错误 |

## 下一步

- 参考 `contracts/protobuf/` 目录查看 Protobuf 定义
- 参考 `contracts/api/` 目录查看 Go API 接口定义
- 参考 `quickstart.md` 了解如何使用这些数据模型
