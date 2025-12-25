# 命令执行流程调试分析

## 问题描述

服务端发送命令后，客户端收到流但没有处理命令。

## 问题分析

### 日志分析

**服务端日志**：
```
time=2025-12-25T20:08:45.155+08:00 level=INFO msg="Opening stream to client (with promise)"
time=2025-12-25T20:08:45.155+08:00 level=INFO msg="Stream opened, writing frame"
time=2025-12-25T20:08:45.155+08:00 level=INFO msg="Frame written, waiting for ACK response"
time=2025-12-25T20:08:45.155+08:00 level=INFO msg="✅ Message sent, waiting for ACK in background"
```

**客户端日志**：
```
time=2025-12-25T20:08:45.155+08:00 level=INFO msg="📨 Stream accepted from server, starting handler"
```

### 根本原因

1. **服务端发送消息后没有关闭流的写端**
   - 服务端在 `SendToWithPromise` 中发送消息后，没有关闭流的写端
   - 客户端使用 `io.ReadAll` 读取帧，会一直等待直到EOF
   - 由于服务端没有关闭写端，客户端一直阻塞等待

2. **QUIC流的半关闭机制**
   - QUIC双向流支持半关闭（half-close）
   - 服务端发送完数据后，应该关闭流的写端，通知客户端数据已发送完毕
   - 客户端读取完数据后，可以在同一个流上写入ACK
   - 客户端写入ACK后，关闭流的写端
   - 服务端读取完ACK后，关闭流

## 修复方案

### 1. 服务端修复

在 `pkg/transport/server/server.go` 的 `SendToWithPromise` 方法中：

```go
// 发送消息后，关闭流的写端
stream.CancelWrite(0) // 取消写端，错误码0表示正常关闭写端
```

### 2. 客户端修复

在 `pkg/transport/codec/protobuf.go` 的 `ReadFrame` 方法中：

```go
// 读取所有数据，即使遇到错误，如果已读取到数据也使用
data, err := io.ReadAll(r)
if err != nil && len(data) == 0 {
    // 如果没有读取到任何数据就出错，返回错误
    return nil, fmt.Errorf("%w: %v", pkgerrors.ErrDecodeFailed, err)
}
// 如果读取到数据（即使有错误），尝试解码
```

### 3. 添加详细日志

在客户端添加更详细的日志，帮助调试：
- 读取帧的开始和结束
- 帧类型和大小
- 错误处理

## 修复后的流程

1. **服务端发送命令**
   - 打开流
   - 写入命令数据
   - **关闭流的写端**（`CancelWrite(0)`）
   - 在goroutine中等待ACK

2. **客户端接收命令**
   - 接受流
   - 读取帧（会收到EOF，因为服务端关闭了写端）
   - 解码数据消息
   - 分发到Dispatcher
   - 执行命令
   - 在同一个流上写入ACK
   - 关闭流的写端

3. **服务端接收ACK**
   - 读取ACK帧
   - 完成Promise
   - 更新命令状态

## 测试验证

修复后，应该看到以下日志：

**客户端**：
```
level=INFO msg="📨 Stream accepted from server, starting handler"
level=INFO msg="开始读取流中的帧"
level=INFO msg="成功读取帧" frame_type=FRAME_TYPE_DATA
level=INFO msg="处理DATA帧"
level=INFO msg="✅ Data message received" msg_id=xxx type=MESSAGE_TYPE_COMMAND
level=INFO msg="执行Shell命令" command=xxx
level=INFO msg="Shell命令执行完成" success=true exit_code=0
level=INFO msg="✅ ACK sent successfully with result"
```

**服务端**：
```
level=INFO msg="✅ ACK received from client" status=ACK_STATUS_SUCCESS has_result=true
level=INFO msg="Command execution completed" status=completed
```

## 注意事项

1. **错误码0**：`CancelWrite(0)` 使用错误码0表示正常关闭写端，不是错误
2. **部分读取**：即使 `io.ReadAll` 返回错误，如果已读取到数据，也应该尝试解码
3. **流的状态**：关闭写端后，流仍然可以读取和写入（双向流的特性）

