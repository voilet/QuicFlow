# 命令系统快速开始指南

这是一个 5 分钟快速开始指南，帮助你快速理解和使用 QUIC 命令下发和回调系统。

## 核心概念

```
┌──────────┐         ┌──────────┐         ┌──────────┐
│ HTTP API │ ─命令─> │  Server  │ ─QUIC─> │  Client  │
│          │ <─结果─ │ (Promise)│ <─回调─ │(Executor)│
└──────────┘         └──────────┘         └──────────┘
```

## 5 步快速集成

### 步骤 1：服务端 - 创建 CommandManager

```go
import "github.com/voilet/QuicFlow/pkg/command"

// 在服务器启动时创建
commandManager := command.NewCommandManager(server, logger)
```

### 步骤 2：服务端 - 集成到 HTTP API

```go
import "github.com/voilet/QuicFlow/pkg/api"

// 传入 commandManager
httpServer := api.NewHTTPServer(
    ":8080",
    server,
    commandManager, // ← 添加这个参数
    logger,
)
httpServer.Start()
```

### 步骤 3：客户端 - 实现命令执行器

```go
import "github.com/voilet/QuicFlow/pkg/command"

type MyExecutor struct{}

func (e *MyExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
    switch commandType {
    case "restart":
        // 执行重启
        return json.Marshal(map[string]bool{"success": true})
    default:
        return nil, fmt.Errorf("unknown command: %s", commandType)
    }
}
```

### 步骤 4：客户端 - 注册命令处理器

```go
// 创建处理器
executor := &MyExecutor{}
handler := command.NewCommandHandler(client, executor, logger)

// 注册到 dispatcher
dispatcher.RegisterHandler(
    protocol.MessageType_MESSAGE_TYPE_COMMAND,
    handler,
)
```

### 步骤 5：下发命令

```bash
# 下发命令
curl -X POST http://localhost:8080/api/command \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "command_type": "restart",
    "payload": {},
    "timeout": 30
  }'

# 响应示例
{
  "success": true,
  "command_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Command sent successfully"
}
```

## 查询命令状态

```bash
# 查询单个命令
curl http://localhost:8080/api/command/550e8400-e29b-41d4-a716-446655440000

# 响应示例
{
  "success": true,
  "command": {
    "command_id": "550e8400-e29b-41d4-a716-446655440000",
    "client_id": "client-001",
    "command_type": "restart",
    "status": "completed",
    "result": {"success": true},
    "created_at": "2024-01-01T12:00:00Z",
    "completed_at": "2024-01-01T12:00:05Z"
  }
}
```

## 完整示例

### 服务端完整代码

```go
package main

import (
    "log"

    "github.com/voilet/QuicFlow/pkg/api"
    "github.com/voilet/QuicFlow/pkg/command"
    "github.com/voilet/QuicFlow/pkg/monitoring"
    "github.com/voilet/QuicFlow/pkg/transport/server"
)

func main() {
    // 1. 创建 logger
    logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")

    // 2. 创建服务器配置
    config := &server.ServerConfig{
        TLSCertFile: "certs/server.crt",
        TLSKeyFile:  "certs/server.key",
        Logger:      logger,
        // ... 其他配置
    }

    // 3. 创建服务器
    srv, err := server.NewServer(config)
    if err != nil {
        log.Fatal(err)
    }

    // 4. 创建命令管理器
    commandManager := command.NewCommandManager(srv, logger)

    // 5. 创建 HTTP API
    httpServer := api.NewHTTPServer(":8080", srv, commandManager, logger)
    httpServer.Start()

    // 6. 启动 QUIC 服务器
    if err := srv.Start(":8474"); err != nil {
        log.Fatal(err)
    }

    logger.Info("Server started with command support")

    // 等待信号...
}
```

### 客户端完整代码

```go
package main

import (
    "encoding/json"
    "log"

    "github.com/voilet/QuicFlow/pkg/command"
    "github.com/voilet/QuicFlow/pkg/dispatcher"
    "github.com/voilet/QuicFlow/pkg/monitoring"
    "github.com/voilet/QuicFlow/pkg/protocol"
    "github.com/voilet/QuicFlow/pkg/transport/client"
)

// 实现命令执行器
type MyExecutor struct{}

func (e *MyExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
    switch commandType {
    case "restart":
        // 执行重启逻辑
        log.Println("Executing restart command")
        return json.Marshal(map[string]interface{}{
            "success": true,
            "message": "Restarted successfully",
        })
    case "update_config":
        // 执行配置更新
        log.Println("Executing update_config command")
        var config map[string]interface{}
        json.Unmarshal(payload, &config)
        return json.Marshal(map[string]interface{}{
            "success":        true,
            "updated_fields": len(config),
        })
    default:
        return nil, fmt.Errorf("unknown command: %s", commandType)
    }
}

func main() {
    // 1. 创建 logger
    logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")

    // 2. 创建客户端配置
    config := &client.ClientConfig{
        ClientID:           "client-001",
        InsecureSkipVerify: true,
        Logger:             logger,
        // ... 其他配置
    }

    // 3. 创建客户端
    c, err := client.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }

    // 4. 创建命令执行器
    executor := &MyExecutor{}

    // 5. 创建命令处理器
    commandHandler := command.NewCommandHandler(c, executor, logger)

    // 6. 创建并配置 dispatcher
    dispatcherConfig := &dispatcher.DispatcherConfig{
        WorkerCount: 10,
        Logger:      logger,
    }
    disp := dispatcher.NewDispatcher(dispatcherConfig)

    // 7. 注册命令处理器
    disp.RegisterHandler(
        protocol.MessageType_MESSAGE_TYPE_COMMAND,
        commandHandler,
    )

    // 8. 启动 dispatcher
    disp.Start()

    // 9. 连接到服务器
    if err := c.Connect("localhost:8474"); err != nil {
        log.Fatal(err)
    }

    logger.Info("Client connected and ready to receive commands")

    // 等待信号...
}
```

## HTTP API 速查

### 1. 下发命令

```bash
POST /api/command
Content-Type: application/json

{
  "client_id": "client-001",
  "command_type": "restart",
  "payload": {"delay_seconds": 5},
  "timeout": 30
}
```

### 2. 查询命令

```bash
GET /api/command/{command_id}
```

### 3. 列出命令

```bash
# 所有命令
GET /api/commands

# 按客户端过滤
GET /api/commands?client_id=client-001

# 按状态过滤
GET /api/commands?status=completed

# 组合过滤
GET /api/commands?client_id=client-001&status=pending
```

## 命令状态说明

| 状态 | 说明 |
|------|------|
| `pending` | 已下发，等待客户端执行 |
| `executing` | 客户端正在执行 |
| `completed` | 执行成功 |
| `failed` | 执行失败 |
| `timeout` | 执行超时 |

## 常见命令类型示例

### 重启服务

```json
{
  "command_type": "restart",
  "payload": {
    "delay_seconds": 5,
    "graceful": true
  }
}
```

### 更新配置

```json
{
  "command_type": "update_config",
  "payload": {
    "config": {
      "log_level": "debug",
      "timeout": 60
    }
  }
}
```

### 获取状态

```json
{
  "command_type": "get_status",
  "payload": {}
}
```

## 错误处理

### 客户端不在线

```json
{
  "error": "client not connected: client-001"
}
```

### 命令执行失败

```json
{
  "command": {
    "status": "failed",
    "error": "invalid parameters: delay_seconds must be positive"
  }
}
```

### 命令超时

```json
{
  "command": {
    "status": "timeout",
    "error": "promise timeout"
  }
}
```

## 最佳实践

### 1. 超时设置

```go
// 短命令（查询类）
timeout: 10 * time.Second

// 常规命令（配置更新）
timeout: 30 * time.Second

// 长命令（重启、升级）
timeout: 60 * time.Second
```

### 2. 参数验证

```go
func (e *MyExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
    // 解析参数
    var params RestartParams
    if err := json.Unmarshal(payload, &params); err != nil {
        return nil, fmt.Errorf("invalid payload: %w", err)
    }

    // 验证参数
    if params.DelaySeconds < 0 {
        return nil, fmt.Errorf("delay_seconds must be non-negative")
    }

    // 执行命令
    // ...
}
```

### 3. 结果格式

```go
// 成功结果
result := map[string]interface{}{
    "success": true,
    "message": "Operation completed",
    "data":    someData,
}

// 失败结果
return nil, fmt.Errorf("operation failed: %v", reason)
```

## 调试技巧

### 1. 查看日志

```bash
# 服务端日志
grep "Command" server.log

# 客户端日志
grep "Command" client.log
```

### 2. 实时监控命令

```bash
# 持续查询命令状态
watch -n 1 "curl -s http://localhost:8080/api/commands | jq '.'"
```

### 3. 查看详细状态

```bash
# 查询单个命令的完整信息
curl http://localhost:8080/api/command/{command_id} | jq '.'
```

## 下一步

- 📖 阅读 [完整技术文档](docs/command-system.md)
- 💻 查看 [示例代码](examples/command/)
- 🧪 运行 [测试脚本](examples/command/test-command.sh)

## 常见问题

**Q: 命令执行失败，但状态显示 pending？**
A: 可能客户端未连接或未注册命令处理器。检查客户端日志。

**Q: 如何实现异步命令（不等待结果）？**
A: 将 timeout 设置为 0，客户端仍会执行但服务端不等待结果。

**Q: 命令历史会永久保存吗？**
A: 不会，默认保留 30 分钟后自动清理。可实现持久化存储。

**Q: 如何实现命令权限控制？**
A: 在 HTTP API 中添加认证中间件，在 CommandExecutor 中验证权限。

---

**需要帮助？** 查看 [完整文档](COMMAND_SYSTEM.md) 或查看 [示例代码](examples/command/)
