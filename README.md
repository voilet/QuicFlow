# QUIC Backbone Network

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A high-performance, industrial-grade QUIC-based communication backbone network for reliable message transmission between clients and servers.

## Features

### Core Capabilities

- **Reliable QUIC Transport**: Built on [quic-go](https://github.com/quic-go/quic-go) with TLS 1.3 encryption
- **Auto-Reconnection**: Exponential backoff strategy (1s → 60s) with configurable retry limits
- **Heartbeat Mechanism**: Automatic health monitoring (15s interval, 45s timeout, 3-strike cleanup)
- **Session Management**: Concurrent session tracking with atomic operations
- **Message Routing**: Worker pool-based dispatcher with configurable concurrency
- **Unicast & Broadcast**: Send messages to specific clients or all connected clients
- **Promise/Callback**: Async request-response pattern with timeout handling
- **Event Hooks**: Real-time notifications for connections, disconnections, messages, and timeouts

### Advanced Features

- **Comprehensive Metrics**: 27+ metrics covering connections, messages, latency, errors, and system stats
- **Prometheus Export**: HTTP endpoint for Prometheus scraping (text format 0.0.4)
- **Latency Tracking**: P50, P95, P99 percentiles with histogram-based distribution
- **Error Handling**: Standardized error types with context-aware logging
- **Graceful Shutdown**: Proper cleanup of goroutines, connections, and resources
- **Weak Network Support**: QUIC's built-in congestion control and fast recovery

## Quick Start

### Prerequisites

- Go 1.21 or higher
- OpenSSL (for generating TLS certificates)

### Installation

```bash
# Clone the repository
git clone https://github.com/voilet/QuicFlow.git
cd quic-backbone

# Install dependencies
go mod download

# Generate TLS certificates
./scripts/gen-certs.sh

# Build binaries
make build
```

### Run the Server

```bash
# 使用默认配置启动（自动搜索 config/server.yaml）
./bin/quic-server

# 使用指定配置文件启动
./bin/quic-server -c config/server.yaml

# 使用高性能配置启动（支持 10W 连接 + 5W 并发任务）
./bin/quic-server -c config/server-highperf.yaml

# 生成配置文件
./bin/quic-server genconfig -o my-config.yaml
./bin/quic-server genconfig --high-perf -o highperf.yaml

# 查看版本信息
./bin/quic-server version
```

### Run the Client

```bash
# Basic client
./bin/quic-client -server localhost:8474 -id client-001

# Client with auto-reconnect
./bin/quic-client -server localhost:8474 -id client-002 -insecure
```

## Load Testing Tool

QUIC Flow 提供了一个专用的负载测试工具 `quic-loadtest`，用于批量启动客户端进行大规模连接测试。

### 构建负载测试工具

```bash
# 仅构建 loadtest 工具
make build-loadtest

# 或构建所有工具
make build
```

### 使用方法

```bash
# 启动 1 万个客户端连接
./bin/quic-loadtest -s 127.0.0.1:8474 -n 10000 -c 200

# 参数说明:
#   -s, --server      服务器地址 (默认: 127.0.0.1:8474)
#   -n, --count       客户端数量 (默认: 10000)
#   -p, --prefix      客户端 ID 前缀 (默认: load-client)
#   -c, --concurrency 并发连接数 (默认: 100)
#   -k, --insecure    跳过 TLS 验证 (默认: true)
#   --keep-alive      保持连接 (默认: true)
#   --report-interval 状态报告间隔秒数 (默认: 5)
#   --log-level       日志级别 debug/info/warn/error (默认: warn)
```

### 生成客户端 ID 列表

```bash
# 生成 1 万个客户端 ID 到文件
./bin/quic-loadtest generate -n 10000 -o /tmp/clients.txt

# 生成自定义前缀的 ID
./bin/quic-loadtest generate -n 5000 -p my-client -o clients.txt

# 输出到标准输出
./bin/quic-loadtest generate -n 100 -p test
```

### 负载测试示例

```bash
# 终端 1: 启动服务器 (高性能模式)
./bin/quic-server -c config/server-highperf.yaml

# 终端 2: 启动 1 万个客户端
./bin/quic-loadtest -s 127.0.0.1:8474 -n 10000 -c 200

# 终端 3: 通过 Web 管理界面向所有客户端下发命令
cd web && npm run dev
# 访问 http://localhost:3000
```

## Web Management Interface

QUIC Flow 包含一个基于 Vue 3 + Element Plus 的 Web 管理界面，用于客户端管理和批量命令下发。

### 启动 Web 界面

```bash
cd web

# 安装依赖
npm install

# 开发模式启动
npm run dev

# 生产构建
npm run build
```

访问 `http://localhost:3000` 打开管理界面。

### 功能特性

- **客户端列表**: 实时显示所有连接的客户端
- **多选批量下发**: 支持选择多个客户端批量执行命令
- **实时结果展示**: 命令执行结果实时返回并展示
- **流式执行 (SSE)**: 支持 SSE 流式返回，先完成的结果先显示
- **执行统计**: 显示已发送、已返回、未执行、不在线的客户端统计

### 批量命令下发

1. 在「客户端列表」页面勾选目标客户端
2. 点击「批量下发」跳转到命令下发页面
3. 选择命令类型（Shell 命令、获取状态等）
4. 点击「批量执行」或「流式执行 (SSE)」
5. 查看执行结果和统计信息

## Release Management System

QUIC Flow 提供完整的发布管理系统，支持多种部署类型和三层配置架构。

### 支持的部署类型

| 部署类型 | 说明 | 适用场景 |
|----------|------|----------|
| **脚本部署** | 传统脚本执行 | 通用场景 |
| **容器部署** | Docker 容器管理 | 容器化应用 |
| **Git 拉取** | Git 仓库代码同步 | 代码更新部署 |
| **Kubernetes** | K8s 资源管理 | 云原生应用 |

### 三层配置架构

发布系统采用三层配置架构，实现灵活的配置管理：

```
┌─────────────────────────────────────────────────────────────┐
│                    任务级别 (Task)                            │
│    临时覆盖：镜像、副本数、资源限制、环境变量追加                  │
└─────────────────────────────────────────────────────────────┘
                            ▲ 覆盖
┌─────────────────────────────────────────────────────────────┐
│                    版本级别 (Version)                         │
│    发布配置：镜像tag、环境变量、资源限制、部署脚本                 │
└─────────────────────────────────────────────────────────────┘
                            ▲ 覆盖
┌─────────────────────────────────────────────────────────────┐
│                    项目级别 (Project)                         │
│    基础设施：仓库地址、容器名、端口、卷、网络、安全、健康检查       │
└─────────────────────────────────────────────────────────────┘
```

### 配置优先级

| 配置类型 | 项目级别 | 版本级别 | 任务级别 |
|----------|----------|----------|----------|
| 端口/卷/网络 | ✅ 固定 | | |
| 镜像 tag | 默认值 | ✅ 必填 | ⚙️ 覆盖 |
| 环境变量 | 默认值 | ✅ 增量 | ⚙️ 追加 |
| 资源限制 | 默认值 | ⚙️ 覆盖 | ⚙️ 覆盖 |
| 副本数 (K8s) | 默认值 | ⚙️ 覆盖 | ⚙️ 覆盖 |
| 部署脚本 | 默认值 | ⚙️ 条件覆盖 | |

### 部署脚本优先级

部署脚本（`pre_script` 和 `post_script`）采用**条件覆盖**策略：

- **项目脚本**：作为所有版本的默认脚本
- **版本脚本**：仅当非空时覆盖项目脚本

```
如果 版本脚本 != "" {
    使用版本脚本
} 否则 {
    使用项目脚本（默认值）
}
```

| 场景 | 项目脚本 | 版本脚本 | 最终执行 |
|------|----------|----------|----------|
| 仅使用通用脚本 | `restart.sh` | 空 | `restart.sh` |
| 版本特定脚本 | `restart.sh` | `migrate.sh` | `migrate.sh` |
| 无需脚本 | 空 | 空 | 无 |

### 功能特性

- **版本管理**：创建、编辑、删除版本
- **部署任务**：立即执行或定时执行
- **金丝雀发布**：支持百分比金丝雀和自动/手动全量
- **配置预览**：部署前预览三层合并后的最终配置
- **实时日志**：SSE 流式返回部署执行日志
- **容器日志**：查看容器运行时日志

详细设计文档：[配置分层设计方案](docs/config-layer-design.md)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       Application Layer                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Business   │  │   Business   │  │   Business   │      │
│  │  Handler 1   │  │  Handler 2   │  │  Handler N   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Dispatcher Layer                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Message Router (Worker Pool Pattern)                │   │
│  │  - 10 workers (configurable)                         │   │
│  │  - Task queue (1000 capacity)                        │   │
│  │  - Timeout control (30s default)                     │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Transport Layer                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Session    │  │   Promise    │  │   Codec      │      │
│  │   Manager    │  │   Manager    │  │  (Protobuf)  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                         QUIC Layer                           │
│  ┌────────────────────────────────────────────────────┐     │
│  │  quic-go (RFC 9000)                                │     │
│  │  - TLS 1.3 encryption                              │     │
│  │  - Multiplexing without head-of-line blocking      │     │
│  │  - Built-in congestion control                     │     │
│  └────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

QUIC Flow 使用 YAML 配置文件管理服务器参数，基于 [Viper](https://github.com/spf13/viper) 实现。

### 配置文件

项目提供两个预设配置文件：

| 配置文件                      | 模式       | 适用场景                   |
| ----------------------------- | ---------- | -------------------------- |
| `config/server.yaml`          | 标准模式   | 10K 连接，开发和小规模部署 |
| `config/server-highperf.yaml` | 高性能模式 | 100K+ 连接，50K 并发任务   |

### 启动服务器

```bash
# 使用标准配置启动
./bin/quic-server -c config/server.yaml

# 使用高性能配置启动
./bin/quic-server -c config/server-highperf.yaml

# 生成默认配置文件
./bin/quic-server genconfig -o my-config.yaml

# 生成高性能配置文件
./bin/quic-server genconfig --high-perf -o highperf-config.yaml
```

### 配置参数说明

#### 标准模式 vs 高性能模式

| 参数                        | 标准模式 | 高性能模式 | 说明                   |
| --------------------------- | -------- | ---------- | ---------------------- |
| `server.max_clients`        | 10,000   | 150,000    | 最大客户端连接数       |
| `message.worker_count`      | 20       | 200        | Dispatcher Worker 数量 |
| `message.task_queue_size`   | 2,000    | 100,000    | 任务队列大小           |
| `message.max_promises`      | 50,000   | 150,000    | 最大 Promise 数量      |
| `quic.max_incoming_streams` | 1,000    | 10,000     | 每连接最大并发流       |
| `batch.enabled`             | false    | true       | 批量执行功能           |
| `batch.max_concurrency`     | 5,000    | 5,000      | 批量执行并发数         |

#### 完整配置示例

```yaml
# 服务器基础配置
server:
  addr: ":8474" # QUIC 监听地址
  api_addr: ":8475" # HTTP API 地址
  high_perf: false # 高性能模式标记
  max_clients: 10000 # 最大客户端数

# TLS 配置
tls:
  cert_file: "certs/server-cert.pem"
  key_file: "certs/server-key.pem"

# QUIC 协议配置
quic:
  max_idle_timeout: 60 # 空闲超时（秒）
  max_incoming_streams: 1000 # 每连接最大并发流
  max_incoming_uni_streams: 100 # 单向流数量
  initial_stream_receive_window: 524288 # 初始流接收窗口（512KB）
  max_stream_receive_window: 6291456 # 最大流接收窗口（6MB）
  initial_connection_receive_window: 1048576 # 初始连接接收窗口（1MB）
  max_connection_receive_window: 15728640 # 最大连接接收窗口（15MB）

# 会话管理配置
session:
  heartbeat_interval: 15 # 心跳间隔（秒）
  heartbeat_timeout: 45 # 心跳超时（秒）
  heartbeat_check_interval: 5 # 心跳检查间隔（秒）
  max_timeout_count: 3 # 最大超时次数

# 消息处理配置
message:
  worker_count: 20 # Dispatcher Worker 数量
  task_queue_size: 2000 # 任务队列大小
  handler_timeout: 30 # 处理超时（秒）
  max_promises: 50000 # 最大 Promise 数量
  promise_warn_threshold: 40000 # Promise 警告阈值
  default_message_timeout: 30 # 默认消息超时（秒）

# 批量执行配置
batch:
  enabled: false # 是否启用
  max_concurrency: 5000 # 最大并发数
  task_timeout: 60 # 单任务超时（秒）
  job_timeout: 600 # 整体任务超时（秒）
  max_retries: 2 # 最大重试次数
  retry_interval: 1 # 重试间隔（秒）

# 日志配置
log:
  level: "info" # debug, info, warn, error
  format: "text" # text, json
  file: "" # 日志文件路径（空=stdout）
```

### 环境变量

支持通过环境变量覆盖配置，前缀为 `QUIC_`：

```bash
# 示例
export QUIC_SERVER_ADDR=":9090"
export QUIC_SERVER_MAX_CLIENTS=50000
export QUIC_MESSAGE_WORKER_COUNT=100
export QUIC_LOG_LEVEL="debug"

./bin/quic-server -c config/server.yaml
```

### 高性能模式系统调优

使用高性能模式前，需要调优 Linux 系统参数：

```bash
# 运行系统调优脚本
sudo ./scripts/tune-system.sh persist

# 主要调整参数：
# - 文件描述符限制: 1,000,000
# - UDP 缓冲区: 256MB
# - 端口范围: 10000-65535
# - TCP/UDP 内存: 自动优化
```

### Client Configuration

```go
config := &client.ClientConfig{
    ClientID: "client-001",

    // TLS settings
    InsecureSkipVerify: false, // Set to true for testing

    // Reconnection settings
    ReconnectEnabled: true,
    InitialBackoff:   1 * time.Second,
    MaxBackoff:       60 * time.Second,

    // Heartbeat settings
    HeartbeatInterval: 15 * time.Second,
    HeartbeatTimeout:  45 * time.Second,
}

c, err := client.NewClient(config)
```

## HTTP API

服务器提供 HTTP API 用于客户端管理和命令下发，默认监听 `:8475`。

### 客户端管理

```bash
# 获取所有客户端列表
curl http://localhost:8475/api/clients

# 获取单个客户端信息
curl http://localhost:8475/api/clients/{client_id}

# 健康检查
curl http://localhost:8475/health
```

### 命令下发

```bash
# 向单个客户端发送命令
curl -X POST http://localhost:8475/api/command \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "command_type": "exec_shell",
    "payload": {"command": "ls -la"},
    "timeout": 30
  }'

# 批量命令下发 (等待所有完成后返回)
curl -X POST http://localhost:8475/api/command/multi \
  -H "Content-Type: application/json" \
  -d '{
    "client_ids": ["client-001", "client-002", "client-003"],
    "command_type": "exec_shell",
    "payload": {"command": "hostname"},
    "timeout": 30
  }'

# 查询命令状态
curl http://localhost:8475/api/command/{command_id}

# 命令历史列表
curl http://localhost:8475/api/commands
```

### SSE 流式命令 (实时返回)

流式命令 API 使用 Server-Sent Events (SSE) 技术，实现命令结果的实时返回。先完成的客户端结果会先推送到前端，无需等待所有客户端完成。

```bash
# 流式批量命令 (SSE)
curl -N -X POST http://localhost:8475/api/command/stream \
  -H "Content-Type: application/json" \
  -d '{
    "client_ids": ["client-001", "client-002", "client-003"],
    "command_type": "exec_shell",
    "payload": {"command": "sleep 1 && hostname"},
    "timeout": 30
  }'
```

**SSE 事件格式:**

```
data: {"type":"result","client_id":"client-001","result":{...}}

data: {"type":"result","client_id":"client-002","result":{...}}

data: {"type":"complete","summary":{"total":3,"success_count":3,"failed_count":0,"duration_ms":1234}}
```

**JavaScript 调用示例:**

```javascript
// 使用 Fetch API 消费 SSE 流
const response = await fetch("/api/command/stream", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    client_ids: ["client-001", "client-002"],
    command_type: "exec_shell",
    payload: { command: "hostname" },
    timeout: 30,
  }),
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;

  const text = decoder.decode(value);
  // 解析 "data: {...}\n\n" 格式
  const lines = text.split("\n\n");
  for (const line of lines) {
    if (line.startsWith("data: ")) {
      const event = JSON.parse(line.slice(6));
      if (event.type === "result") {
        console.log("收到结果:", event.result);
      } else if (event.type === "complete") {
        console.log("全部完成:", event.summary);
      }
    }
  }
}
```

### 批量执行 vs 流式执行对比

| 特性     | 批量执行 (`/command/multi`) | 流式执行 (`/command/stream`) |
| -------- | --------------------------- | ---------------------------- |
| 返回方式 | 等待所有客户端完成后返回    | 实时返回每个结果             |
| 用户体验 | 需要等待最慢的客户端        | 先完成的先显示               |
| 技术实现 | 标准 HTTP JSON 响应         | Server-Sent Events (SSE)     |
| 适用场景 | 少量客户端、需要统一处理    | 大量客户端、需要实时反馈     |

## Batch Execution

高性能模式支持批量向多个客户端发送命令：

### HTTP API

```bash
# 发起批量执行
curl -X POST http://localhost:8475/api/batch/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "system.collect_info",
    "payload": {"type": "hardware"},
    "target_clients": ["client-001", "client-002", "client-003"],
    "wait_for_result": true
  }'

# 查询任务状态
curl http://localhost:8475/api/batch/jobs/{job_id}

# 列出所有任务
curl http://localhost:8475/api/batch/jobs

# 取消任务
curl -X POST http://localhost:8475/api/batch/jobs/{job_id}/cancel
```

### 批量执行特性

- **并发控制**: 最大 5000 并发发送
- **进度追踪**: 实时查看成功/失败/待处理数量
- **超时处理**: 单任务 60s，整体任务 30min
- **自动重试**: 失败任务自动重试 2 次
- **任务取消**: 支持中途取消任务

## Examples

### Basic Echo Server

```go
// examples/echo/server.go
package main

import (
    "context"
    "log"

    "github.com/voilet/QuicFlow/pkg/dispatcher"
    "github.com/voilet/QuicFlow/pkg/protocol"
    "github.com/voilet/QuicFlow/pkg/transport/server"
)

// Echo handler echoes messages back to the sender
type EchoHandler struct{}

func (h *EchoHandler) OnMessage(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
    log.Printf("Received message: %s", string(msg.Payload))

    // Echo back to sender
    return &protocol.DataMessage{
        Type:    protocol.MessageType_MESSAGE_TYPE_RESPONSE,
        Payload: msg.Payload,
    }, nil
}

func main() {
    // Create server
    srv, _ := server.NewServer(config)

    // Register echo handler
    dispatcher := dispatcher.NewDispatcher(nil)
    dispatcher.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_COMMAND, &EchoHandler{})

    // Start server
    srv.Start(":8474")
}
```

### Send Message with Callback

```go
// examples/callback/client.go
package main

import (
    "context"
    "log"
    "time"

    "github.com/voilet/QuicFlow/pkg/protocol"
    "github.com/voilet/QuicFlow/pkg/transport/client"
)

func main() {
    c, _ := client.NewClient(config)
    c.Connect("localhost:8474")

    // Send message and wait for Ack
    msg := &protocol.DataMessage{
        Type:    protocol.MessageType_MESSAGE_TYPE_COMMAND,
        Payload: []byte("Hello, Server!"),
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    ack, err := c.SendMessage(ctx, msg, true, 0)
    if err != nil {
        log.Fatalf("Send failed: %v", err)
    }

    log.Printf("Received Ack: status=%v", ack.Status)
}
```

## Monitoring

### Prometheus Integration

Access metrics at `http://localhost:9090/metrics`:

```promql
# Connected clients
quic_backbone_connected_clients

# Message throughput (per minute)
rate(quic_backbone_messages_sent_total[1m])

# P99 latency
quic_backbone_latency_p99_milliseconds

# Error rate
rate(quic_backbone_encoding_errors_total[1m])
```

### Event Hooks

```go
hooks := &monitoring.EventHooks{
    OnConnect: func(clientID string) {
        log.Printf("Client connected: %s", clientID)
    },
    OnMessageSent: func(msgID, clientID string, err error) {
        if err != nil {
            log.Printf("Send failed: %v", err)
        }
    },
}
```

## Performance

### 性能规格

| 指标       | 标准模式      | 高性能模式     |
| ---------- | ------------- | -------------- |
| 最大连接数 | 10,000        | 100,000+       |
| 并发任务数 | 2,000         | 50,000         |
| 消息吞吐量 | 10,000+ msg/s | 100,000+ msg/s |
| P50 延迟   | < 5ms         | < 10ms         |
| P99 延迟   | < 50ms        | < 100ms        |
| 每连接内存 | ~50KB         | ~50KB          |

### Benchmarks

```bash
# 运行并发连接测试
go test -v ./tests -run TestHighConcurrencyConnections

# 运行命令发送基准测试
go test -bench=BenchmarkCommandSend ./tests

# 运行完整负载测试（需要先启动服务器）
go test -v ./tests -run TestConcurrentCommands
```

### Optimization Tips

1. **系统调优**: 高性能模式前运行 `sudo ./scripts/tune-system.sh persist`
2. **Worker 数量**: 根据 CPU 核心数调整 `message.worker_count`
3. **队列大小**: `task_queue_size` 应 >= 预期并发任务数 × 2
4. **心跳间隔**: 高并发时增加间隔以减少开销
5. **Promise 容量**: 监控活跃 Promise 数量，及时调整限制

### 数据库性能调优

在高并发场景下，数据库操作可能成为性能瓶颈。以下是推荐的配置：

#### GORM 日志级别

生产环境必须禁用 GORM SQL 日志以避免性能损耗：

```yaml
database:
  log_level: silent  # silent | error | warn | info
```

| 级别   | 性能影响 | 说明                           |
| ------ | -------- | ------------------------------ |
| silent | 无       | 禁用所有日志（生产环境推荐）   |
| error  | 极低     | 仅记录错误                     |
| warn   | 低       | 警告和错误                     |
| info   | 高       | 完整 SQL 日志（仅开发调试）    |

#### 连接池配置

根据并发负载调整连接池参数：

```yaml
database:
  max_idle_conns: 10      # 最大空闲连接数
  max_open_conns: 100     # 最大打开连接数
  conn_max_lifetime: 3600 # 连接最大存活时间（秒）
```

**推荐配置**（基于并发连接数）：

| 并发连接数 | max_idle_conns | max_open_conns |
| ---------- | -------------- | -------------- |
| < 1,000    | 10             | 50             |
| 1,000-5,000 | 15            | 100            |
| 5,000-20,000 | 20           | 200            |
| > 20,000   | 50             | 500+           |

#### 性能对比（16C 1万客户端并发）

| 指标       | 优化前（info日志） | 优化后（silent日志） | 提升 |
| ---------- | ----------------- | -------------------- | ---- |
| CPU 使用率 | 85%               | 45%                  | 47%  |
| 内存占用   | 2.1GB             | 1.8GB                | 14%  |
| 响应时间   | 180ms             | 85ms                 | 53%  |

## Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run MVP test
./scripts/test-mvp.sh

# Generate coverage report
make coverage
```

## Project Structure

```
.
├── cmd/                    # Command-line programs
│   ├── server/             # Server binary
│   ├── client/             # Client binary
│   ├── ctl/                # CLI management tool
│   └── loadtest/           # Load testing tool (批量客户端连接)
├── config/                 # Configuration files
│   ├── server.yaml         # Standard mode config (10K connections)
│   └── server-highperf.yaml # High-perf mode config (100K+ connections)
├── web/                    # Web management interface (Vue 3 + Element Plus)
│   ├── src/
│   │   ├── api/            # API client (axios + SSE)
│   │   ├── views/          # Page components
│   │   │   ├── ClientList.vue    # 客户端列表 (多选批量下发)
│   │   │   └── CommandSend.vue   # 命令下发 (支持 SSE 流式执行)
│   │   └── router/         # Vue Router
│   └── package.json
├── pkg/                    # Library code
│   ├── api/                # HTTP API handlers
│   │   ├── http_server.go  # REST API
│   │   ├── stream_api.go   # SSE streaming API
│   │   └── batch_api.go    # Batch execution API
│   ├── batch/              # Batch execution engine
│   ├── callback/           # Promise/callback mechanism
│   ├── command/            # Command types and handlers
│   ├── config/             # Viper configuration management
│   ├── dispatcher/         # Message routing
│   ├── errors/             # Error types
│   ├── monitoring/         # Metrics and logging
│   ├── protocol/           # Protobuf definitions
│   ├── router/             # Command router
│   │   └── handlers/       # Built-in command handlers
│   ├── session/            # Session management
│   └── transport/          # QUIC transport layer
├── examples/               # Example programs
├── scripts/                # Build and test scripts
│   ├── gen-certs.sh        # Generate TLS certificates
│   └── tune-system.sh      # System tuning for high-perf
├── tests/                  # Integration and load tests
├── docs/                   # Documentation
└── certs/                  # TLS certificates

```

## Documentation

### User Guides

- 📖 [配置指南](docs/configuration-guide.md) - 完整的参数配置说明（服务器、客户端、CLI）
- 🚀 [快速参考](docs/quick-reference.md) - 常用命令和参数速查
- 🔧 [CLI 使用指南](docs/cli-guide.md) - CLI 工具详细使用说明
- 🌐 [HTTP API 文档](docs/http-api.md) - HTTP API 接口说明
- 📦 [配置分层设计](docs/config-layer-design.md) - 发布系统三层配置架构与优先级说明

### Technical Documentation

- [API 文档](docs/API.md) - API 详细说明
- [网络可靠性设计](docs/network-reliability.md) - 网络可靠性架构

### Quick Links

```bash
# 查看服务器参数
./bin/quic-server -h

# 查看客户端参数
./bin/quic-client -h

# 查看 CLI 工具帮助
./bin/quic-ctl help
```

## API Documentation

See [docs/API.md](docs/API.md) for detailed API documentation.

For network reliability information, see [docs/network-reliability.md](docs/network-reliability.md).

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [quic-go](https://github.com/quic-go/quic-go)
- Protocol buffers from [Google Protocol Buffers](https://protobuf.dev/)
- Inspired by modern messaging systems

## Support

- GitHub Issues: [https://github.com/voilet/QuicFlow/issues](https://github.com/voilet/QuicFlow/issues)
- Documentation: [https://github.com/voilet/QuicFlow/wiki](https://github.com/voilet/QuicFlow/wiki)

---

**Note**: This is an industrial-grade implementation suitable for production use. For educational purposes or simple use cases, consider the examples in the `examples/` directory.
