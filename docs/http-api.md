# HTTP API 实现说明

## 框架选择

本项目使用 [Gin](https://github.com/gin-gonic/gin) 作为 HTTP API 框架。

## Gin 框架优势

1. **高性能**：基于 httprouter，性能优异
2. **易用性**：简洁的 API 设计，易于上手
3. **中间件支持**：灵活的中间件机制
4. **参数验证**：内置参数绑定和验证功能
5. **错误处理**：统一的错误处理和恢复机制

## 主要特性

### 1. 路由分组

使用 Gin 的路由分组功能组织 API：

```go
api := h.router.Group("/api")
{
    api.GET("/clients", h.handleListClients)
    api.GET("/clients/:id", h.handleGetClient)
    api.POST("/send", h.handleSend)
    api.POST("/broadcast", h.handleBroadcast)
}
h.router.GET("/health", h.handleHealth)
```

### 2. 参数验证

使用 Gin 的参数绑定和验证：

```go
type SendRequest struct {
    ClientID string `json:"client_id" binding:"required"`
    Type     string `json:"type"`
    Payload  string `json:"payload" binding:"required"`
    WaitAck  bool   `json:"wait_ack"`
}

func (h *HTTPServer) handleSend(c *gin.Context) {
    var req SendRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": fmt.Sprintf("Invalid request body: %v", err),
        })
        return
    }
    // ...
}
```

### 3. 中间件

实现了两个中间件：

- **Recovery 中间件**：自动恢复 panic，防止服务器崩溃
- **Logger 中间件**：记录所有 HTTP 请求的日志

```go
// 添加中间件
h.router.Use(gin.Recovery())
h.router.Use(h.loggerMiddleware())
```

### 4. 日志记录

自定义日志中间件，集成项目的监控日志系统：

```go
func (h *HTTPServer) loggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        method := c.Request.Method

        c.Next()

        latency := time.Since(start)
        statusCode := c.Writer.Status()

        h.logger.Debug("HTTP request",
            "method", method,
            "path", path,
            "status", statusCode,
            "latency", latency,
            "client_ip", c.ClientIP(),
        )
    }
}
```

### 5. JSON 响应

使用 Gin 的 JSON 响应方法：

```go
c.JSON(http.StatusOK, gin.H{
    "status": "ok",
    "time":   time.Now().Unix(),
})
```

## API 端点

### GET /api/clients
获取所有在线客户端列表

**响应示例：**
```json
{
  "total": 3,
  "clients": [
    {
      "client_id": "client-001",
      "remote_addr": "127.0.0.1:52104",
      "connected_at": 1703404800000,
      "uptime": "9s"
    }
  ]
}
```

### GET /api/clients/:id
获取指定客户端详细信息

**响应示例：**
```json
{
  "client_id": "client-001",
  "remote_addr": "127.0.0.1:52104",
  "connected_at": 1703404800000,
  "last_heartbeat": 1703404809000,
  "state": "CLIENT_STATE_CONNECTED"
}
```

### POST /api/send
向指定客户端发送消息

**请求体：**
```json
{
  "client_id": "client-001",
  "type": "command",
  "payload": "{\"action\":\"restart\"}",
  "wait_ack": false
}
```

**响应示例：**
```json
{
  "success": true,
  "msg_id": "491e0615-ca0e-48cd-a24b-80f760cc5f69",
  "message": "Message sent successfully"
}
```

**参数验证：**
- `client_id`: 必需
- `payload`: 必需
- `type`: 可选，默认为 "command"
- `wait_ack`: 可选，默认为 false

### POST /api/broadcast
向所有客户端广播消息

**请求体：**
```json
{
  "type": "event",
  "payload": "{\"event\":\"update_available\"}"
}
```

**响应示例：**
```json
{
  "success": true,
  "msg_id": "c6eeb4eb-35bb-475c-b661-f74fca7c941b",
  "total": 3,
  "success_count": 3,
  "failed_count": 0
}
```

### GET /health
健康检查端点

**响应示例：**
```json
{
  "status": "ok",
  "time": 1703404800
}
```

## 配置

### Gin 模式

生产环境使用 Release 模式，减少日志输出：

```go
gin.SetMode(gin.ReleaseMode)
```

开发环境可以设置为 Debug 模式：

```go
gin.SetMode(gin.DebugMode)
```

### 服务器配置

在 `cmd/server/main.go` 中配置 API 监听地址：

```go
apiAddr := flag.String("api", ":8475", "HTTP API 监听地址")
```

## 依赖

添加到 `go.mod` 的依赖：

```
github.com/gin-gonic/gin v1.11.0
```

安装依赖：

```bash
go get -u github.com/gin-gonic/gin
```

## 测试

使用测试脚本验证功能：

```bash
./scripts/test-cli.sh
```

或者手动测试：

```bash
# 启动服务器
./bin/quic-server -api :8475

# 测试健康检查
curl http://localhost:8475/health

# 测试参数验证
curl -X POST http://localhost:8475/api/send \
  -H "Content-Type: application/json" \
  -d '{"type":"command"}'
```

## 性能优化

1. **使用路由分组**：减少路由匹配时间
2. **Release 模式**：生产环境使用 Release 模式，减少不必要的日志输出
3. **参数验证**：在请求处理早期验证参数，快速失败
4. **优雅关闭**：支持优雅关闭，确保正在处理的请求完成

## 错误处理

Gin 提供了统一的错误处理机制：

- **参数验证错误**：返回 400 Bad Request
- **资源不存在**：返回 404 Not Found
- **服务器错误**：返回 500 Internal Server Error
- **Panic 恢复**：自动恢复并返回 500

## 扩展性

框架设计易于扩展：

1. **添加新端点**：在路由分组中添加新的处理函数
2. **添加中间件**：使用 `router.Use()` 添加全局中间件
3. **自定义验证**：实现自定义验证器
4. **添加认证**：实现认证中间件

---

最后更新：2025-12-24
