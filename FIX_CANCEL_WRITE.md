# 修复 CancelWrite 导致的读取问题

## 问题描述

1. **客户端读取帧失败**：`stream 1 canceled by remote with error code 0`
2. **服务端读取ACK失败**：`failed to decode message: no data read`

## 根本原因

当服务端或客户端调用 `CancelWrite(0)` 关闭流的写端时：
- `io.ReadAll` 可能会在读取数据之前就返回错误
- 即使数据已经发送，读取操作也可能因为流的写端被关闭而失败
- 需要改进读取逻辑，确保在遇到 `CancelWrite` 错误时也能获取已发送的数据

## 修复方案

### 1. 改进 `ReadFrame` 实现 (`pkg/transport/codec/protobuf.go`)

**之前**：使用 `io.ReadAll`，遇到错误就返回
```go
data, err := io.ReadAll(r)
if err != nil {
    return nil, err
}
```

**现在**：使用缓冲区逐步读取，即使遇到错误也能获取已读取的数据
```go
var data []byte
buf := make([]byte, 4096)

for {
    n, err := r.Read(buf)
    if n > 0 {
        data = append(data, buf[:n]...)
    }
    if err != nil {
        // 如果是CancelWrite错误，但已经读取到数据，继续处理
        if len(data) > 0 {
            break
        }
        return nil, err
    }
}
```

### 2. 客户端发送ACK后关闭写端 (`pkg/transport/client/receive.go`)

**添加**：客户端发送ACK后，关闭流的写端
```go
// 关闭流的写端，通知服务端ACK已发送完毕
stream.CancelWrite(0) // 错误码0表示正常关闭写端
```

### 3. 改进错误处理 (`pkg/transport/client/receive.go`)

**添加**：处理 `CancelWrite` 错误的特定错误消息
```go
errStr == "stream 1 canceled by remote with error code 0"
```

## 完整流程

1. **服务端发送命令**
   - 打开流
   - 写入命令数据
   - **关闭流的写端** (`CancelWrite(0)`)
   - 在goroutine中等待ACK

2. **客户端接收命令**
   - 接受流
   - **逐步读取帧**（即使遇到 `CancelWrite` 错误，也能获取已读取的数据）
   - 解码数据消息
   - 分发到Dispatcher
   - 执行命令
   - 在同一个流上写入ACK
   - **关闭流的写端** (`CancelWrite(0)`)

3. **服务端接收ACK**
   - **逐步读取ACK帧**（即使遇到 `CancelWrite` 错误，也能获取已读取的数据）
   - 解码ACK消息
   - 完成Promise
   - 更新命令状态

## 关键改进

1. **逐步读取**：不再使用 `io.ReadAll`，而是使用缓冲区逐步读取
2. **错误容忍**：即使遇到 `CancelWrite` 错误，如果已读取到数据，继续处理
3. **双向关闭**：服务端和客户端都关闭流的写端，确保对方能正确读取

## 测试验证

修复后，应该看到：

**客户端日志**：
```
level=INFO msg="开始读取流中的帧"
level=INFO msg="成功读取帧" frame_type=FRAME_TYPE_DATA
level=INFO msg="✅ Data message received"
level=INFO msg="执行Shell命令"
level=INFO msg="ACK帧已写入，关闭流的写端"
level=INFO msg="✅ Ack sent"
```

**服务端日志**：
```
level=INFO msg="Write side closed, waiting for ACK"
level=INFO msg="✅ ACK received from client" status=ACK_STATUS_SUCCESS has_result=true
level=INFO msg="Command execution completed" status=completed
```

## 注意事项

1. **错误码0**：`CancelWrite(0)` 使用错误码0表示正常关闭写端，不是错误
2. **部分读取**：逐步读取确保即使遇到错误，也能获取已读取的数据
3. **流的状态**：关闭写端后，流仍然可以读取（双向流的特性）
4. **错误处理**：需要检查特定的错误消息，如 `"stream 1 canceled by remote with error code 0"`

