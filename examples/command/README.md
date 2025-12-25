# 命令下发和回调示例

这个示例演示如何使用 QUIC 双向流实现命令下发和状态回调功能。

## 架构说明

```
HTTP API (下发命令)
    ↓
CommandManager (服务端)
    ↓
QUIC Stream (发送COMMAND消息)
    ↓
CommandHandler (客户端)
    ↓
CommandExecutor (业务层执行)
    ↓
QUIC Stream (发送ACK响应)
    ↓
Promise.Complete() (服务端)
    ↓
HTTP API (查询结果)
```

## 运行示例

### 1. 启动服务器

```bash
go run examples/command/server/main.go
```

服务器将启动：
- QUIC 服务器：`localhost:8474`
- HTTP API：`localhost:8080`

### 2. 启动客户端

```bash
go run examples/command/client/main.go -id client-001
```

客户端会：
- 连接到服务器
- 注册命令处理器
- 等待接收命令

### 3. 通过 HTTP API 下发命令

下发重启命令：

```bash
curl -X POST http://localhost:8080/api/command \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "command_type": "restart",
    "payload": {"delay_seconds": 5},
    "timeout": 30
  }'
```

响应示例：
```json
{
  "success": true,
  "command_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Command sent successfully"
}
```

### 4. 查询命令状态

```bash
curl http://localhost:8080/api/command/550e8400-e29b-41d4-a716-446655440000
```

响应示例：
```json
{
  "success": true,
  "command": {
    "command_id": "550e8400-e29b-41d4-a716-446655440000",
    "client_id": "client-001",
    "command_type": "restart",
    "payload": {"delay_seconds": 5},
    "status": "completed",
    "result": {"success": true, "message": "Restarted successfully"},
    "created_at": "2024-01-01T12:00:00Z",
    "sent_at": "2024-01-01T12:00:00Z",
    "completed_at": "2024-01-01T12:00:05Z",
    "timeout": 30000000000
  }
}
```

### 5. 列出所有命令

```bash
# 列出所有命令
curl http://localhost:8080/api/commands

# 按客户端过滤
curl "http://localhost:8080/api/commands?client_id=client-001"

# 按状态过滤
curl "http://localhost:8080/api/commands?status=completed"
```

## 支持的命令类型

示例实现了以下命令类型：

1. **restart** - 重启服务
   - 参数：`delay_seconds` (可选)

2. **update_config** - 更新配置
   - 参数：`config` (JSON对象)

3. **get_status** - 获取状态
   - 无参数

## 命令状态流转

```
pending → executing → completed
                    → failed
                    → timeout
```

- **pending**: 已下发，等待客户端执行
- **executing**: 客户端正在执行（可选状态）
- **completed**: 执行成功
- **failed**: 执行失败
- **timeout**: 执行超时

## 自定义命令执行器

实现 `command.CommandExecutor` 接口：

```go
type MyCommandExecutor struct {
    // 你的业务逻辑
}

func (e *MyCommandExecutor) Execute(commandType string, payload []byte) ([]byte, error) {
    switch commandType {
    case "my_command":
        // 解析参数
        var params MyParams
        json.Unmarshal(payload, &params)

        // 执行业务逻辑
        result := doSomething(params)

        // 返回结果
        return json.Marshal(result)
    default:
        return nil, fmt.Errorf("unknown command type: %s", commandType)
    }
}
```

## 集成到现有项目

### 服务端

```go
// 创建命令管理器
commandManager := command.NewCommandManager(server, logger)

// 创建 HTTP API（传入 commandManager）
httpServer := api.NewHTTPServer(":8080", server, commandManager, logger)
httpServer.Start()
```

### 客户端

```go
// 创建命令执行器
executor := &MyCommandExecutor{}

// 创建命令处理器
handler := command.NewCommandHandler(client, executor, logger)

// 注册到 dispatcher
dispatcher.RegisterHandler(protocol.MessageType_MESSAGE_TYPE_COMMAND, handler)
```

## 注意事项

1. 命令超时时间建议设置为 30-60 秒
2. 命令执行结果会保留 30 分钟后自动清理
3. Promise 容量上限为 50000，超过会拒绝新命令
4. 客户端应该实现幂等的命令处理逻辑
