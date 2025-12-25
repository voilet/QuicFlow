# 快速开始：QUIC 通信骨干网络

**功能**：001-quic-backbone-network
**创建日期**：2025-12-23
**目标读者**：开发者（使用本库的业务应用开发者）

## 概述

QUIC 通信骨干网络是一个基于 QUIC 协议的高性能、可靠的客户端-服务器通信库。本指南将帮助您快速上手，在 5 分钟内运行您的第一个示例。

## 前置要求

- Go 1.21 或更高版本
- TLS 证书和密钥文件（用于安全通信）

### 生成测试证书（开发环境）

```bash
# 生成自签名证书（仅用于开发测试）
openssl req -x509 -newkey rsa:4096 -keyout server-key.pem -out server-cert.pem -days 365 -nodes -subj "/CN=localhost"
```

**⚠️ 警告**：生产环境请使用由受信任 CA 签发的证书。

## 安装

```bash
go get github.com/voilet/QuicFlow
```

## 5 分钟示例

### 步骤 1：启动服务器

创建 `server.go`：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    backbone "github.com/voilet/QuicFlow"
)

// 定义消息处理器
type EchoHandler struct{}

func (h *EchoHandler) OnMessage(ctx context.Context, msg *backbone.IncomingMessage) (*backbone.Response, error) {
    log.Printf("收到来自 %s 的消息: %s", msg.SenderID, string(msg.Payload))

    // Echo 回消息内容
    return &backbone.Response{
        Status: backbone.AckStatusSuccess,
        Result: []byte(fmt.Sprintf("Echo: %s", msg.Payload)),
    }, nil
}

func main() {
    // 创建服务器配置
    config := backbone.ServerConfig{
        TLSCertFile:       "server-cert.pem",
        TLSKeyFile:        "server-key.pem",
        MaxClients:        10000,
        HeartbeatInterval: 15 * time.Second,
    }

    // 创建服务器
    server, err := backbone.NewServer(config)
    if err != nil {
        log.Fatal("创建服务器失败:", err)
    }

    // 注册消息处理器
    server.RegisterHandler(backbone.MessageTypeCommand, &EchoHandler{})

    // 设置事件钩子
    server.SetEventHooks(backbone.EventHooks{
        OnConnect: func(clientID string) {
            log.Printf("客户端连接: %s", clientID)
        },
        OnDisconnect: func(clientID string, reason error) {
            log.Printf("客户端断开: %s, 原因: %v", clientID, reason)
        },
    })

    // 启动服务器
    log.Println("启动服务器，监听 :8474...")
    go func() {
        if err := server.Start(":8474"); err != nil {
            log.Fatal("服务器启动失败:", err)
        }
    }()

    // 等待中断信号
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    // 优雅关闭
    log.Println("正在关闭服务器...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    server.Stop(ctx)
}
```

运行服务器：

```bash
go run server.go
```

### 步骤 2：启动客户端

创建 `client.go`：

```go
package main

import (
    "log"
    "time"

    backbone "github.com/voilet/QuicFlow"
)

func main() {
    // 创建客户端配置
    config := backbone.ClientConfig{
        ClientID:           "client-001",
        InsecureSkipVerify: true, // 仅用于开发（跳过证书验证）
        ReconnectEnabled:   true,
    }

    // 创建客户端
    client, err := backbone.NewClient(config)
    if err != nil {
        log.Fatal("创建客户端失败:", err)
    }

    // 设置事件钩子
    client.SetEventHooks(backbone.EventHooks{
        OnConnect: func(clientID string) {
            log.Println("已连接到服务器")
        },
        OnReconnect: func(clientID string, attemptCount int) {
            log.Printf("重连成功（第 %d 次尝试）", attemptCount)
        },
    })

    // 连接到服务器
    log.Println("连接到服务器 localhost:8474...")
    if err := client.Connect("localhost:8474"); err != nil {
        log.Fatal("连接失败:", err)
    }

    // 发送消息
    msg := &backbone.Message{
        Type:    backbone.MessageTypeCommand,
        Payload: []byte("Hello, QUIC Backbone!"),
        WaitAck: true, // 等待服务器响应
        Timeout: 5 * time.Second,
    }

    log.Println("发送消息...")
    resp, err := client.SendMessage(msg)
    if err != nil {
        log.Fatal("发送消息失败:", err)
    }

    log.Printf("收到响应: %s (状态: %v)", string(resp.Result), resp.Status)

    // 保持连接
    time.Sleep(30 * time.Second)

    // 断开连接
    client.Disconnect()
}
```

运行客户端：

```bash
go run client.go
```

### 预期输出

**服务器端**：
```
2025-12-23 10:00:00 启动服务器，监听 :8474...
2025-12-23 10:00:05 客户端连接: client-001
2025-12-23 10:00:05 收到来自 client-001 的消息: Hello, QUIC Backbone!
```

**客户端端**：
```
2025-12-23 10:00:05 连接到服务器 localhost:8474...
2025-12-23 10:00:05 已连接到服务器
2025-12-23 10:00:05 发送消息...
2025-12-23 10:00:05 收到响应: Echo: Hello, QUIC Backbone! (状态: SUCCESS)
```

## 核心功能详解

### 1. 消息类型

系统支持 4 种消息类型，根据业务场景选择：

```go
backbone.MessageTypeCommand   // 指令（需要执行操作）
backbone.MessageTypeEvent     // 事件（通知性质）
backbone.MessageTypeQuery     // 查询（请求数据）
backbone.MessageTypeResponse  // 响应（返回结果）
```

### 2. 广播消息

服务器向所有在线客户端广播消息：

```go
msg := &backbone.Message{
    Type:    backbone.MessageTypeEvent,
    Payload: []byte(`{"event": "server_maintenance"}`),
    WaitAck: false, // 广播不等待响应
}

count, err := server.Broadcast(msg)
log.Printf("广播到 %d 个客户端", count)
```

**⚠️ 重要**：广播是"尽力而为"模式，不保证所有客户端都能收到（部分客户端可能离线或网络中断）。如果需要可靠送达，请使用单播（`SendTo`）。

### 3. 异步回调（WaitAck）

发送需要确认的消息并等待响应：

```go
msg := &backbone.Message{
    Type:    backbone.MessageTypeQuery,
    Payload: []byte(`{"query": "status"}`),
    WaitAck: true,          // 等待响应
    Timeout: 10 * time.Second, // 10 秒超时
}

resp, err := server.SendTo("client-001", msg)
if err != nil {
    log.Printf("发送失败: %v", err)
    return
}

switch resp.Status {
case backbone.AckStatusSuccess:
    log.Printf("执行成功: %s", string(resp.Result))
case backbone.AckStatusFailure:
    log.Printf("执行失败: %v", resp.Error)
case backbone.AckStatusTimeout:
    log.Printf("超时")
}
```

**客户端处理需要 Ack 的消息**：

```go
type QueryHandler struct{}

func (h *QueryHandler) OnMessage(ctx context.Context, msg *backbone.IncomingMessage) (*backbone.Response, error) {
    // 处理查询
    result := processQuery(msg.Payload)

    // 返回响应
    return &backbone.Response{
        Status: backbone.AckStatusSuccess,
        Result: result,
    }, nil
}

client.RegisterHandler(backbone.MessageTypeQuery, &QueryHandler{})
```

### 4. 监控和指标

实时获取服务器指标：

```go
metrics := server.GetMetrics()
log.Printf("连接数: %d", metrics.ConnectedClients)
log.Printf("吞吐量: %d 消息/秒", metrics.MessageThroughput)
log.Printf("平均延迟: %d ms", metrics.AverageLatency)
log.Printf("P99 延迟: %d ms", metrics.P99Latency)
```

### 5. 事件钩子

监听连接生命周期事件：

```go
server.SetEventHooks(backbone.EventHooks{
    OnConnect: func(clientID string) {
        log.Printf("[事件] 客户端 %s 已连接", clientID)
        // 可以在这里执行自定义逻辑，如记录到数据库
    },

    OnDisconnect: func(clientID string, reason error) {
        log.Printf("[事件] 客户端 %s 断开: %v", clientID, reason)
    },

    OnHeartbeatTimeout: func(clientID string) {
        log.Printf("[告警] 客户端 %s 心跳超时", clientID)
        // 可以触发告警通知
    },

    OnMessageSent: func(msgID string, clientID string) {
        // 细粒度的消息追踪
    },
})
```

### 6. 客户端自动重连

客户端内置自动重连机制，无需手动处理：

```go
config := backbone.ClientConfig{
    ReconnectEnabled: true,
    InitialBackoff:   1 * time.Second,  // 首次重试延迟
    MaxBackoff:       60 * time.Second, // 最大重试延迟
}

// 连接断开后，客户端会自动重连：
// - 第 1 次重试：1 秒后
// - 第 2 次重试：2 秒后
// - 第 3 次重试：4 秒后
// - 第 4 次重试：8 秒后
// - ...
// - 最大延迟：60 秒
```

### 7. 查询在线客户端

列出所有当前在线的客户端：

```go
clients := server.ListClients()
for _, clientID := range clients {
    info, err := server.GetClientInfo(clientID)
    if err != nil {
        continue
    }

    log.Printf("客户端: %s", info.ClientID)
    log.Printf("  地址: %s", info.RemoteAddr)
    log.Printf("  连接时间: %s", info.ConnectedAt)
    log.Printf("  最后心跳: %s", info.LastHeartbeat)
}
```

## 高级配置

### 服务器配置

```go
config := backbone.ServerConfig{
    TLSCertFile:           "server-cert.pem",
    TLSKeyFile:            "server-key.pem",
    MaxClients:            10000,              // 最大并发客户端数
    HeartbeatInterval:     15 * time.Second,  // 心跳间隔
    HeartbeatTimeout:      45 * time.Second,  // 心跳超时（3 × 间隔）
    MaxPromises:           50000,             // 最大待回调消息数
    PromiseWarnThreshold:  40000,             // 警告阈值（80%）
    DefaultMessageTimeout: 30 * time.Second,  // 默认消息超时
}
```

### 客户端配置

```go
config := backbone.ClientConfig{
    ClientID:           "my-client-id",
    TLSCertFile:        "client-cert.pem",  // 双向 TLS（可选）
    TLSKeyFile:         "client-key.pem",
    InsecureSkipVerify: false,              // 生产环境必须为 false
    ReconnectEnabled:   true,
    InitialBackoff:     1 * time.Second,
    MaxBackoff:         60 * time.Second,
}
```

## 性能优化建议

### 1. 消息大小

- 推荐：< 64KB
- 最大：1MB
- 超过 1MB 的数据建议分片传输或使用其他方式（如对象存储）

### 2. 并发处理

服务器默认使用 10 个 worker 处理消息。如果业务逻辑复杂（如数据库查询），可以增加 worker 数量（通过自定义 Dispatcher）。

### 3. 批量发送

如果需要向同一客户端发送多条消息，考虑合并为单条消息（减少往返次数）。

### 4. 心跳调优

根据网络环境调整心跳间隔：
- 稳定网络：15 秒（默认）
- 不稳定网络：10 秒
- 广域网/跨国：20 秒

## 常见问题

### Q1: 如何处理消息丢失？

**A**: 系统通过 QUIC 层的可靠流传输保证消息不丢失。如果网络中断：
- 服务器 → 客户端：QUIC 会自动重传，客户端重连后继续接收
- 客户端 → 服务器：客户端重连后需要重新发送（应用层需要重试逻辑）

### Q2: 广播消息是否可靠？

**A**: 广播是"尽力而为"模式，不保证所有客户端都收到。如果需要确保送达，请使用单播（`SendTo`）并检查返回值。

### Q3: 如何处理慢客户端？

**A**: 系统内置保护机制：
- Dispatcher 的 channel 缓冲（1000 条），满时拒绝新消息
- 消息处理超时（默认 30 秒），超时后取消 handler 执行

### Q4: 如何集成 Prometheus 监控？

**A**:

```go
import "net/http"

// 启动 Prometheus 指标导出
http.Handle("/metrics", server.GetMetricsHandler()) // 假设 API 提供此方法
go http.ListenAndServe(":9090", nil)
```

然后在 Prometheus 配置中添加：

```yaml
scrape_configs:
  - job_name: 'quic-backbone'
    static_configs:
      - targets: ['localhost:9090']
```

### Q5: 客户端 ID 如何生成？

**A**: 客户端 ID 由业务层生成并保证唯一性。推荐方案：
- UUID
- 设备 MAC 地址
- 业务系统分配的唯一标识（如用户 ID + 设备 ID）

## 下一步

- 阅读 [API 文档](./contracts/api/interfaces.go)
- 查看 [数据模型](./data-model.md)
- 了解 [Protobuf 协议](./contracts/protobuf/)
- 参考 [实现计划](./plan.md)

## 支持

- GitHub Issues: https://github.com/voilet/QuicFlow/issues
- 文档: https://yourorg.github.io/quic-backbone/
