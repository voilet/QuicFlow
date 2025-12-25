# 命令下发和回调系统

## 概述

命令下发和回调系统基于 QUIC 双向流实现，允许服务端通过 HTTP API 向客户端下发命令，并等待客户端执行后的回调结果。

## 架构设计

### 组件架构

```
┌─────────────────────────────────────────────────────────────────┐
│                          HTTP API Layer                         │
│  - POST /api/command      (下发命令)                              │
│  - GET  /api/command/:id  (查询命令状态)                          │
│  - GET  /api/commands     (列出命令)                              │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      CommandManager (服务端)                     │
│  - 命令生命周期管理                                                │
│  - Promise创建和追踪                                              │
│  - 超时控制                                                       │
│  - 命令历史存储                                                    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                        QUIC Stream
                   (MESSAGE_TYPE_COMMAND)
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                     CommandHandler (客户端)                      │
│  - 接收命令消息                                                    │
│  - 调用CommandExecutor                                            │
│  - 构造响应                                                        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                  CommandExecutor (业务层实现)                     │
│  - 具体命令的执行逻辑                                               │
│  - 返回执行结果                                                    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
                        QUIC Stream
                   (MESSAGE_TYPE_RESPONSE)
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      Promise.Complete()                          │
│  - 接收Ack响应                                                    │
│  - 更新命令状态                                                    │
│  - 触发回调                                                        │
└─────────────────────────────────────────────────────────────────┘
```

### 消息流转

```
1. HTTP POST /api/command
   ↓
2. CommandManager.SendCommand()
   - 创建Command记录
   - 创建Promise
   - 发送DataMessage (TYPE=COMMAND)
   ↓
3. Client收到COMMAND消息
   ↓
4. CommandHandler.HandleCommand()
   - 解析CommandPayload
   - 调用CommandExecutor.Execute()
   ↓
5. 返回DataMessage (TYPE=RESPONSE)
   - 包含AckMessage
   ↓
6. Server收到RESPONSE消息
   ↓
7. Promise.Complete()
   - 更新Command状态
   - 完成Promise
   ↓
8. HTTP GET /api/command/:id
   - 返回命令执行结果
```

## 核心组件

### 1. CommandManager (服务端)

**职责**：
- 管理命令的生命周期
- 创建和追踪 Promise
- 维护命令历史
- 定期清理过期命令

**关键方法**：
```go
// 下发命令
func (cm *CommandManager) SendCommand(
    clientID, commandType string,
    payload json.RawMessage,
    timeout time.Duration,
) (*Command, error)

// 查询命令状态
func (cm *CommandManager) GetCommand(commandID string) (*Command, error)

// 列出命令
func (cm *CommandManager) ListCommands(clientID string, status CommandStatus) []*Command
```

### 2. CommandHandler (客户端)

**职责**：
- 接收和解析命令消息
- 调用业务层执行器
- 构造响应消息

**关键方法**：
```go
// 处理命令消息
func (h *CommandHandler) HandleCommand(
    ctx context.Context,
    msg *protocol.DataMessage,
) (*protocol.DataMessage, error)
```

### 3. CommandExecutor (业务层接口)

**职责**：
- 定义命令执行接口
- 由业务层实现具体逻辑

**接口定义**：
```go
type CommandExecutor interface {
    Execute(commandType string, payload []byte) (result []byte, err error)
}
```

## 数据结构

### Command 命令对象

```go
type Command struct {
    CommandID   string          `json:"command_id"`   // 命令唯一ID
    ClientID    string          `json:"client_id"`    // 目标客户端
    CommandType string          `json:"command_type"` // 命令类型
    Payload     json.RawMessage `json:"payload"`      // 命令参数

    Status      CommandStatus   `json:"status"`       // 当前状态
    Result      json.RawMessage `json:"result,omitempty"` // 执行结果
    Error       string          `json:"error,omitempty"`  // 错误信息

    CreatedAt   time.Time       `json:"created_at"`   // 创建时间
    SentAt      *time.Time      `json:"sent_at,omitempty"` // 发送时间
    CompletedAt *time.Time      `json:"completed_at,omitempty"` // 完成时间
    Timeout     time.Duration   `json:"timeout"`      // 超时时长
}
```

### CommandStatus 命令状态

```go
type CommandStatus string

const (
    CommandStatusPending   CommandStatus = "pending"   // 已下发，等待执行
    CommandStatusExecuting CommandStatus = "executing" // 正在执行
    CommandStatusCompleted CommandStatus = "completed" // 执行完成
    CommandStatusFailed    CommandStatus = "failed"    // 执行失败
    CommandStatusTimeout   CommandStatus = "timeout"   // 执行超时
)
```

### CommandPayload 命令载荷

```go
type CommandPayload struct {
    CommandType string          `json:"command_type"` // 命令类型
    Payload     json.RawMessage `json:"payload"`      // 命令参数
}
```

## 集成指南

### 服务端集成

#### 1. 创建 CommandManager

```go
import (
    "github.com/voilet/QuicFlow/pkg/command"
    "github.com/voilet/QuicFlow/pkg/transport/server"
)

// 创建服务器
srv, err := server.NewServer(serverConfig)
if err != nil {
    log.Fatal(err)
}

// 创建命令管理器
commandManager := command.NewCommandManager(srv, logger)
```

#### 2. 集成到 HTTP API

```go
import "github.com/voilet/QuicFlow/pkg/api"

// 创建 HTTP 服务器（传入 commandManager）
httpServer := api.NewHTTPServer(
    ":8080",
    srv,           // ServerAPI
    commandManager, // CommandManager
    logger,
)

// 启动 HTTP 服务器
if err := httpServer.Start(); err != nil {
    log.Fatal(err)
}
```

#### 3. 启动服务

```go
// 启动 QUIC 服务器
if err := srv.Start(":8474"); err != nil {
    log.Fatal(err)
}

// 服务器现在支持命令下发功能
```

### 客户端集成

#### 1. 实现 CommandExecutor

```go
import "github.com/voilet/QuicFlow/pkg/command"

type MyCommandExecutor struct {
    // 业务相关的字段
}

func (e *MyCommandExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
    switch commandType {
    case "restart":
        return e.handleRestart(payload)
    case "update_config":
        return e.handleUpdateConfig(payload)
    default:
        return nil, fmt.Errorf("unknown command: %s", commandType)
    }
}

func (e *MyCommandExecutor) handleRestart(payload []byte) ([]byte, error) {
    // 解析参数
    var params RestartParams
    if err := json.Unmarshal(payload, &params); err != nil {
        return nil, err
    }

    // 执行重启逻辑
    // ...

    // 返回结果
    result := RestartResult{Success: true}
    return json.Marshal(result)
}
```

#### 2. 创建 CommandHandler

```go
// 创建命令执行器
executor := &MyCommandExecutor{}

// 创建命令处理器
commandHandler := command.NewCommandHandler(client, executor, logger)
```

#### 3. 注册到 Dispatcher

```go
import (
    "github.com/voilet/QuicFlow/pkg/dispatcher"
    "github.com/voilet/QuicFlow/pkg/protocol"
)

// 创建 dispatcher
disp := dispatcher.NewDispatcher(dispatcherConfig)

// 注册命令处理器
disp.RegisterHandler(
    protocol.MessageType_MESSAGE_TYPE_COMMAND,
    commandHandler,
)

// 启动 dispatcher
disp.Start()
```

## HTTP API 使用

### 1. 下发命令

**请求**：
```bash
POST /api/command
Content-Type: application/json

{
  "client_id": "client-001",
  "command_type": "restart",
  "payload": {
    "delay_seconds": 5
  },
  "timeout": 30
}
```

**响应**：
```json
{
  "success": true,
  "command_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Command sent successfully"
}
```

### 2. 查询命令状态

**请求**：
```bash
GET /api/command/550e8400-e29b-41d4-a716-446655440000
```

**响应**：
```json
{
  "success": true,
  "command": {
    "command_id": "550e8400-e29b-41d4-a716-446655440000",
    "client_id": "client-001",
    "command_type": "restart",
    "payload": {"delay_seconds": 5},
    "status": "completed",
    "result": {"success": true, "message": "Restarted"},
    "created_at": "2024-01-01T12:00:00Z",
    "sent_at": "2024-01-01T12:00:00Z",
    "completed_at": "2024-01-01T12:00:05Z",
    "timeout": 30000000000
  }
}
```

### 3. 列出命令

**请求**：
```bash
# 列出所有命令
GET /api/commands

# 按客户端过滤
GET /api/commands?client_id=client-001

# 按状态过滤
GET /api/commands?status=completed

# 组合过滤
GET /api/commands?client_id=client-001&status=pending
```

**响应**：
```json
{
  "success": true,
  "total": 2,
  "commands": [
    {
      "command_id": "...",
      "client_id": "client-001",
      "command_type": "restart",
      "status": "completed",
      ...
    },
    {
      "command_id": "...",
      "client_id": "client-001",
      "command_type": "update_config",
      "status": "pending",
      ...
    }
  ]
}
```

## 配置建议

### 超时设置

```go
// 短时命令（如查询状态）
timeout := 10 * time.Second

// 常规命令（如更新配置）
timeout := 30 * time.Second

// 长时命令（如重启、更新软件）
timeout := 60 * time.Second
```

### 容量配置

```go
// 服务端 Promise 容量
MaxPromises: 50000

// 命令历史保留时间
maxCommandAge: 30 * time.Minute

// 清理间隔
cleanupInterval: 5 * time.Minute
```

## 错误处理

### 常见错误

1. **客户端不在线**
   ```json
   {
     "error": "client not connected: client-001"
   }
   ```

2. **命令超时**
   ```json
   {
     "command": {
       "status": "timeout",
       "error": "promise timeout"
     }
   }
   ```

3. **执行失败**
   ```json
   {
     "command": {
       "status": "failed",
       "error": "invalid parameters: ..."
     }
   }
   ```

### 错误处理最佳实践

```go
func (e *MyCommandExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
    // 1. 参数验证
    if err := validatePayload(payload); err != nil {
        return nil, fmt.Errorf("invalid payload: %w", err)
    }

    // 2. 执行前检查
    if !canExecute() {
        return nil, fmt.Errorf("cannot execute: resource busy")
    }

    // 3. 执行业务逻辑
    result, err := doExecute(payload)
    if err != nil {
        return nil, fmt.Errorf("execution failed: %w", err)
    }

    // 4. 返回结果
    return result, nil
}
```

## 监控和日志

### 关键指标

- `command_sent_total`: 发送的命令总数
- `command_completed_total`: 完成的命令总数
- `command_failed_total`: 失败的命令总数
- `command_timeout_total`: 超时的命令总数
- `command_duration_seconds`: 命令执行时长

### 日志示例

```
[INFO] Command sent via API command_id=xxx client_id=client-001 command_type=restart
[INFO] Command execution completed command_id=xxx status=completed
[ERROR] Command execution failed command_id=xxx error="invalid parameters"
[WARN] Command timeout command_id=xxx timeout=30s
```

## 安全考虑

1. **认证授权**
   - HTTP API 应添加认证中间件
   - 验证调用者权限

2. **参数验证**
   - 严格验证命令参数
   - 防止注入攻击

3. **命令白名单**
   - 限制可执行的命令类型
   - 实施命令审批流程

4. **速率限制**
   - 限制单个客户端的命令频率
   - 防止命令泛洪攻击

## 性能优化

1. **批量命令**
   - 对多个客户端下发相同命令时使用并发

2. **命令优先级**
   - 为紧急命令分配更高优先级

3. **结果缓存**
   - 对查询类命令缓存结果

4. **异步执行**
   - 长时命令使用异步执行模式

## 故障恢复

### 服务端重启

- 命令状态会丢失（内存存储）
- 建议实现持久化存储（可选）

### 客户端断线

- 未完成的命令会超时
- 重连后可重新下发

### 网络故障

- Promise 超时机制自动处理
- 客户端重连后恢复正常

## 扩展功能

### 1. 命令历史持久化

```go
// 实现持久化接口
type CommandStore interface {
    Save(cmd *Command) error
    Load(commandID string) (*Command, error)
    List(filters ...Filter) ([]*Command, error)
}
```

### 2. 命令优先级

```go
type Command struct {
    // ...
    Priority int `json:"priority"` // 1-10, 10最高
}
```

### 3. 命令依赖

```go
type Command struct {
    // ...
    DependsOn []string `json:"depends_on"` // 依赖的命令ID列表
}
```

### 4. 命令批处理

```go
type BatchCommandRequest struct {
    ClientIDs   []string        `json:"client_ids"`
    CommandType string          `json:"command_type"`
    Payload     json.RawMessage `json:"payload"`
}
```

## 总结

命令下发和回调系统提供了一个强大且灵活的方式来实现服务端对客户端的远程控制。通过 QUIC 双向流和 Promise 机制，实现了高效可靠的命令执行和状态反馈。

关键优势：
- ✅ 基于 QUIC 的高性能双向通信
- ✅ Promise 机制保证可靠回调
- ✅ 完整的命令生命周期管理
- ✅ 灵活的业务层扩展接口
- ✅ 完善的超时和错误处理
- ✅ 简洁的 HTTP API 接口
