# 命令执行结果返回流程分析

## 概述

本文档分析了通过HTTP接口向指定客户端下发指令，以及客户端执行后返回结果的完整流程。

## 当前实现状态

### ✅ 已实现的功能

1. **HTTP接口下发指令** (`pkg/api/http_server.go:handleSendCommand`)
   - 接口：`POST /api/command`
   - 接收命令请求，调用 `commandManager.SendCommand`
   - 立即返回 `CommandID`，不阻塞等待结果

2. **服务端发送指令** (`pkg/command/manager.go:SendCommand`)
   - 创建命令记录，状态为 `pending`
   - 通过 `SendToWithPromise` 发送消息到客户端
   - 启动 `waitForCommandResponse` goroutine 异步等待响应

3. **客户端接收指令** (`pkg/transport/client/receive.go:handleData`)
   - 接收来自服务端的命令消息
   - 分发到 Dispatcher 处理

4. **客户端执行指令** (`pkg/command/handler.go:HandleCommand`)
   - 解析命令载荷
   - 调用 `executor.Execute` 执行命令
   - 返回包含结果的响应消息

5. **客户端返回结果** (`pkg/transport/client/receive.go:handleData`)
   - 从响应中提取执行结果
   - 通过 `sendAck` 发送ACK消息，包含执行结果

6. **服务端接收结果** (`pkg/transport/server/server.go:SendToWithPromise`)
   - 接收客户端返回的ACK消息
   - 完成Promise，将ACK消息传递给等待的goroutine

7. **服务端更新命令状态** (`pkg/command/manager.go:waitForCommandResponse`)
   - 从Promise接收ACK响应
   - 调用 `updateCommandStatus` 更新命令状态和结果
   - 将结果存储在命令记录中

8. **HTTP接口查询命令状态** (`pkg/api/http_server.go:handleGetCommand`)
   - 接口：`GET /api/command/:id`
   - 通过 `commandManager.GetCommand` 查询命令状态
   - 返回命令的完整信息，包括状态和结果

### 🔧 已修复的问题

1. **客户端Shell命令执行** (`cmd/client/main.go:executeShell`)
   - **问题**：之前只返回mock结果，没有真正执行shell命令
   - **修复**：实现了真正的shell命令执行，使用 `exec.CommandContext` 执行命令
   - **特性**：
     - 支持超时控制（默认30秒，最大5分钟）
     - 捕获stdout和stderr输出
     - 限制输出大小（最大10KB）
     - 返回执行结果（success, exit_code, stdout, stderr, message）

## 完整流程

```
1. HTTP请求
   POST /api/command
   {
     "client_id": "client-001",
     "command_type": "exec_shell",
     "payload": {"command": "ls -la"}
   }
   ↓
   返回: {"success": true, "command_id": "xxx", "message": "Command sent successfully"}

2. 服务端处理
   CommandManager.SendCommand()
   - 创建命令记录（status: pending）
   - 发送消息到客户端（WaitAck: true）
   - 启动 waitForCommandResponse goroutine 等待响应
   ↓

3. 客户端接收
   Client.receiveLoop() → handleData()
   - 接收命令消息
   - 分发到 Dispatcher
   ↓

4. 客户端执行
   CommandHandler.HandleCommand()
   - 解析命令载荷
   - executor.Execute("exec_shell", payload)
   - 执行shell命令（exec.CommandContext）
   - 返回执行结果
   ↓

5. 客户端返回结果
   Client.handleData()
   - 从响应中提取结果
   - sendAck(stream, msgID, SUCCESS, result, "")
   - 发送ACK消息（包含执行结果）
   ↓

6. 服务端接收结果
   Server.SendToWithPromise()
   - 读取ACK响应
   - Promise.Complete(ackMsg)
   ↓

7. 服务端更新状态
   CommandManager.waitForCommandResponse()
   - 从Promise接收ACK响应
   - updateCommandStatus(commandID, completed, result, "")
   - 命令状态更新为 completed，结果存储在 Result 字段
   ↓

8. HTTP查询结果
   GET /api/command/:id
   ↓
   返回: {
     "success": true,
     "command": {
       "command_id": "xxx",
       "status": "completed",
       "result": {
         "success": true,
         "exit_code": 0,
         "stdout": "...",
         "stderr": "",
         "message": "命令执行成功"
       }
     }
   }
```

## 数据流

### 命令下发
```
HTTP Request → CommandManager → Server.SendToWithPromise → Client
```

### 结果返回
```
Client.Execute → CommandHandler → Client.sendAck → Server.ReceiveACK → 
CommandManager.waitForCommandResponse → CommandManager.updateCommandStatus
```

### 结果查询
```
HTTP GET /api/command/:id → CommandManager.GetCommand → HTTP Response
```

## 关键数据结构

### CommandRequest (HTTP请求)
```go
type CommandRequest struct {
    ClientID    string          `json:"client_id"`
    CommandType string          `json:"command_type"`
    Payload     json.RawMessage `json:"payload"`
    Timeout     int             `json:"timeout,omitempty"`
}
```

### Command (命令记录)
```go
type Command struct {
    CommandID   string          `json:"command_id"`
    ClientID    string          `json:"client_id"`
    CommandType string          `json:"command_type"`
    Payload     json.RawMessage `json:"payload"`
    Status      CommandStatus   `json:"status"`
    Result      json.RawMessage `json:"result,omitempty"`  // 执行结果
    Error       string          `json:"error,omitempty"`
    CreatedAt   time.Time       `json:"created_at"`
    SentAt      *time.Time      `json:"sent_at,omitempty"`
    CompletedAt *time.Time      `json:"completed_at,omitempty"`
    Timeout     time.Duration   `json:"timeout"`
}
```

### ExecShellResult (Shell命令执行结果)
```go
{
    "success": true,
    "exit_code": 0,
    "stdout": "命令输出",
    "stderr": "错误输出",
    "message": "命令执行成功"
}
```

## 使用示例

### 1. 下发命令
```bash
curl -X POST http://localhost:8080/api/command \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "command_type": "exec_shell",
    "payload": {"command": "ls -la /tmp"},
    "timeout": 30
  }'
```

响应：
```json
{
  "success": true,
  "command_id": "abc-123-def-456",
  "message": "Command sent successfully"
}
```

### 2. 查询命令状态和结果
```bash
curl http://localhost:8080/api/command/abc-123-def-456
```

响应（执行中）：
```json
{
  "success": true,
  "command": {
    "command_id": "abc-123-def-456",
    "client_id": "client-001",
    "command_type": "exec_shell",
    "status": "pending",
    "created_at": "2024-01-01T10:00:00Z"
  }
}
```

响应（执行完成）：
```json
{
  "success": true,
  "command": {
    "command_id": "abc-123-def-456",
    "client_id": "client-001",
    "command_type": "exec_shell",
    "status": "completed",
    "result": {
      "success": true,
      "exit_code": 0,
      "stdout": "total 0\ndrwxrwxrwt ...",
      "stderr": "",
      "message": "命令执行成功"
    },
    "completed_at": "2024-01-01T10:00:05Z"
  }
}
```

## 注意事项

1. **异步执行**：HTTP接口下发命令后立即返回，不等待执行结果。需要通过查询接口获取结果。

2. **超时控制**：
   - 命令级别超时：在 `CommandRequest` 中设置 `timeout`（秒）
   - Shell命令级别超时：在 `exec_shell` 的 `payload` 中设置 `timeout`（秒）
   - 默认超时：30秒

3. **输出限制**：Shell命令的输出（stdout/stderr）限制为10KB，超出部分会被截断。

4. **命令状态**：
   - `pending`: 已下发，等待客户端执行
   - `executing`: 客户端正在执行（暂未使用）
   - `completed`: 执行完成（成功）
   - `failed`: 执行失败
   - `timeout`: 执行超时

5. **结果存储**：命令结果存储在 `Command.Result` 字段中，格式为JSON。

6. **清理机制**：已完成的命令会在30分钟后自动清理。

## 总结

✅ **返回结果功能已完整实现**：
- 客户端执行命令后，将结果通过ACK消息返回给服务端
- 服务端接收结果并更新命令状态
- HTTP接口可以查询命令状态和结果

🔧 **已修复**：
- 客户端现在真正执行shell命令，而不是返回mock结果

📝 **使用方式**：
1. 通过 `POST /api/command` 下发命令
2. 通过 `GET /api/command/:id` 查询命令状态和结果
3. 可以轮询查询接口直到命令完成

