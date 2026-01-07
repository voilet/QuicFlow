# QUIC Flow Go 开发规范

> 基于 Go 1.25.4 和项目实际代码风格编写的开发规范文档

## 目录

1. [项目结构](#项目结构)
2. [命名规范](#命名规范)
3. [代码风格](#代码风格)
4. [错误处理](#错误处理)
5. [测试规范](#测试规范)
6. [API 设计规范](#api-设计规范)
7. [配置管理](#配置管理)
8. [并发安全](#并发安全)
9. [日志规范](#日志规范)
10. [文档注释](#文档注释)

---

## 项目结构

### 标准目录布局

```
quic-flow/
├── cmd/                    # 可执行程序入口
│   ├── server/            # 服务器主程序
│   ├── client/            # 客户端主程序
│   ├── cli/               # CLI 工具
│   └── ctl/               # 控制工具
├── pkg/                   # 公共库代码
│   ├── api/              # HTTP API 服务
│   ├── auth/             # 认证授权
│   ├── config/           # 配置管理
│   └── ...
├── web/                   # 前端资源
├── config/                # 配置文件
├── certs/                 # 证书文件
├── scripts/               # 构建和部署脚本
├── tests/                 # 集成测试
└── docs/                  # 项目文档
```

### 包命名规则

```go
// 包名使用小写单词，不使用下划线或驼峰
package config      // 正确
package http_server // 错误
package api         // 正确
```

### 文件组织

```go
// 每个包应该有一个清晰的主题
// 相关功能放在同一目录下

pkg/
├── config/
│   ├── config.go       # 主配置结构和加载逻辑
│   └── defaults.go     # 默认配置（可选）
├── api/
│   ├── http_server.go  # HTTP 服务器
│   ├── handlers.go     # 请求处理器
│   └── middleware.go   # 中间件
└── session/
    ├── manager.go      # 会话管理器
    ├── session.go      # 会话结构
    └── store.go        # 会话存储
```

---

## 命名规范

### 变量命名

```go
// 使用驼峰命名法
var clientCount int          // 局部变量，小写开头
var MaxConnections int        // 导出变量，大写开头
var defaultTimeout = 30       // 常量使用驼峰，非全大写

// 缩写保持一致
var httpServer *http.Server   // 正确
var HTTPServer *http.Server   // 也正确（保持一致）
var httpRequest *http.Request // 正确
```

### 函数命名

```go
// 导出函数：大写开头，PascalCase
func NewSessionManager() *SessionManager {}
func (s *SessionManager) Start() {}
func (s *SessionManager) Stop() {}

// 内部函数：小写开头，camelCase
func validateConfig(cfg *Config) error {}
func processMessage(msg *protocol.DataMessage) {}

// Getter/Setter
func (c *ServerConfig) GetHeartbeatInterval() time.Duration {}
func (h *HTTPServer) SetHardwareStore(store *hardware.Store) {}
```

### 接口命名

```go
// 接口名通常使用 -er 后缀或动词短语
type ServerAPI interface {
    ListClients() []string
    GetClientInfo(clientID string) (*protocol.ClientInfo, error)
}

type Handler interface {
    Handle(ctx context.Context, msg *Message) error
}

// 单方法接口可以命名动名词
type ReleaseAPIRegistrar interface {
    RegisterRoutes(r *gin.RouterGroup)
}
```

### 常量命名

```go
// 使用驼峰命名，分组时使用 iota
const (
    StreamTypeData         StreamType = 0
    StreamTypeSSH          StreamType = 1
    StreamTypeFileTransfer StreamType = 2
    StreamTypePortForward  StreamType = 3
)

// 错误常量
const (
    DefaultTimeout      = 30 * time.Second
    MaxRetries          = 3
    HeartbeatInterval   = 15 * time.Second
)
```

---

## 代码风格

### 导入顺序

```go
import (
	// 标准库
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	// 外部依赖（按字母顺序）
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	// 项目内部包
	"github.com/voilet/quic-flow/pkg/config"
	"github.com/voilet/quic-flow/pkg/session"
)

// 使用分组注释说明导入类别
```

### 结构体定义

```go
// ServerConfig 服务器完整配置
// 注释应该说明结构体的用途
type ServerConfig struct {
	// 服务器基础配置
	Server ServerSettings `mapstructure:"server"`

	// TLS 配置
	TLS TLSSettings `mapstructure:"tls"`

	// 会话管理配置
	Session SessionSettings `mapstructure:"session"`
}

// 字段注释使用 // 格式，位于字段上方
// tag 说明映射关系，如 mapstructure, json
```

### 函数定义

```go
// NewHTTPServer 创建新的 HTTP API 服务器
// addr: 监听地址
// serverAPI: 服务器 API 实现
// commandManager: 命令管理器（可选）
// logger: 日志记录器
func NewHTTPServer(
	addr string,
	serverAPI ServerAPI,
	commandManager *command.CommandManager,
	logger *monitoring.Logger,
) *HTTPServer {
	// 实现代码
}

// 函数注释应说明：
// 1. 功能描述
// 2. 参数说明（复杂参数）
// 3. 返回值说明
```

### 控制结构

```go
// if 语句
if err != nil {
    return err
}

// 条件简单时可以单行
if isOnline { return }

// for 循环
for i, client := range clients {
    // 处理客户端
}

// 遍历 map
for key, value := range m {
    // 处理键值对
}

// 避免：不必要的 else
if condition {
    return true
}
// 继续处理 false 情况
return false
```

---

## 错误处理

### 错误定义

```go
// 包级错误变量
var (
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrSessionNotFound  = errors.New("session not found")
	ErrInvalidClientID  = errors.New("invalid client ID")
)

// 使用 fmt.Errorf 包装错误
if err != nil {
    return fmt.Errorf("failed to read config: %w", err)
}

// 使用自定义错误包（项目使用 pkg/errors）
import pkgerrors "github.com/voilet/quic-flow/pkg/errors"

if session == nil {
    return fmt.Errorf("%w: session is nil", pkgerrors.ErrInvalidConfig)
}
```

### 错误处理模式

```go
// 早期返回
func ProcessRequest(req *Request) (*Response, error) {
    if req == nil {
        return nil, errors.New("request is nil")
    }

    if req.ClientID == "" {
        return nil, pkgerrors.ErrInvalidClientID
    }

    // 处理请求...
    return resp, nil
}

// 不忽略错误
data, err := io.ReadAll(r)
if err != nil {
    return err
}

// 验证错误
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // 处理超时
    }
    return fmt.Errorf("operation failed: %w", err)
}
```

---

## 测试规范

### 测试文件组织

```go
// 测试文件：源文件名 + _test.go
// protocol_test.go
package ssh

import (
    "bytes"
    "testing"
)

// 测试函数命名：Test + 函数名
func TestStreamType_String(t *testing.T) {
    tests := []struct {
        streamType StreamType
        expected   string
    }{
        {StreamTypeData, "Data"},
        {StreamTypeSSH, "SSH"},
        {StreamType(99), "Unknown(99)"},
    }

    for _, tt := range tests {
        t.Run(tt.expected, func(t *testing.T) {
            if got := tt.streamType.String(); got != tt.expected {
                t.Errorf("StreamType.String() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### 表格驱动测试

```go
func TestWriteAndReadHeader(t *testing.T) {
    tests := []struct {
        name       string
        streamType StreamType
    }{
        {"Data stream", StreamTypeData},
        {"SSH stream", StreamTypeSSH},
        {"FileTransfer stream", StreamTypeFileTransfer},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

### 子测试

```go
// 使用 t.Run 创建子测试，便于识别失败用例
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // 测试代码
    })
}
```

---

## API 设计规范

### REST API 设计

```go
// 路由命名规范
// /api/{resource}/{action}
api := h.router.Group("/api")
{
    api.GET("/clients", h.handleListClients)        // 列表
    api.GET("/clients/:id", h.handleGetClient)      // 详情
    api.POST("/send", h.handleSend)                 // 操作
    api.POST("/command", h.handleSendCommand)       // 命令
    api.DELETE("/sessions/:id", h.handleClose)      // 删除
}

// 处理器命名：handle + 动作 + 资源
func (h *HTTPServer) handleListClients(c *gin.Context) {}
func (h *HTTPServer) handleGetClient(c *gin.Context) {}
func (h *HTTPServer) handleSendCommand(c *gin.Context) {}
```

### 请求/响应结构

```go
// 请求结构体：XxxRequest
type SendRequest struct {
    ClientID string `json:"client_id" binding:"required"`
    Type     string `json:"type"`
    Payload  string `json:"payload" binding:"required"`
    WaitAck  bool   `json:"wait_ack"`
}

// 响应结构体：XxxResponse
type SendResponse struct {
    Success bool   `json:"success"`
    MsgID   string `json:"msg_id"`
    Message string `json:"message"`
}

// 列表响应：ListXxxResponse
type ListClientsResponse struct {
    Total       int64          `json:"total"`
    OnlineCount int64          `json:"online_count"`
    Offset      int            `json:"offset,omitempty"`
    Limit       int            `json:"limit,omitempty"`
    Clients     []ClientDetail `json:"clients"`
}
```

### HTTP 状态码使用

```go
// 200 OK - 成功获取/操作
c.JSON(http.StatusOK, response)

// 201 Created - 创建成功
c.JSON(http.StatusCreated, created)

// 400 Bad Request - 请求参数错误
c.JSON(http.StatusBadRequest, gin.H{
    "error": fmt.Sprintf("Invalid request body: %v", err),
})

// 404 Not Found - 资源不存在
c.JSON(http.StatusNotFound, gin.H{
    "error": fmt.Sprintf("Client not found: %v", err),
})

// 500 Internal Server Error - 服务器内部错误
c.JSON(http.StatusInternalServerError, gin.H{
    "error": fmt.Sprintf("Failed to send: %v", err),
})

// 503 Service Unavailable - 服务不可用
c.JSON(http.StatusServiceUnavailable, gin.H{
    "error": "Command manager not initialized",
})
```

---

## 配置管理

### 配置结构定义

```go
// 使用嵌套结构组织配置
type ServerConfig struct {
    Server ServerSettings `mapstructure:"server"`
    TLS    TLSSettings    `mapstructure:"tls"`
    QUIC   QUICSettings   `mapstructure:"quic"`
}

// 每个子配置独立定义
type ServerSettings struct {
    Addr       string `mapstructure:"addr"`
    APIAddr    string `mapstructure:"api_addr"`
    HighPerf   bool   `mapstructure:"high_perf"`
    MaxClients int64  `mapstructure:"max_clients"`
}
```

### 默认值设置

```go
// DefaultConfig 返回默认配置
func DefaultConfig() *ServerConfig {
    return &ServerConfig{
        Server: ServerSettings{
            Addr:       ":8474",
            APIAddr:    ":8475",
            HighPerf:   false,
            MaxClients: 10000,
        },
        // ... 其他配置
    }
}

// HighPerfConfig 返回高性能配置
func HighPerfConfig() *ServerConfig {
    cfg := DefaultConfig()
    cfg.Server.HighPerf = true
    cfg.Server.MaxClients = 150000
    return cfg
}
```

### 配置加载

```go
// Load 加载配置
// configPath: 配置文件路径（可选，为空时使用默认搜索路径）
func Load(configPath string) (*ServerConfig, error) {
    v := viper.New()

    // 设置默认值
    setDefaults(v)

    // 配置文件设置
    v.SetConfigName("server")
    v.SetConfigType("yaml")

    // 搜索路径
    v.AddConfigPath(".")
    v.AddConfigPath("./config")
    v.AddConfigPath("/etc/quic-flow")

    // 读取并解析
    // ...
}
```

---

## 并发安全

### 使用 sync.Map

```go
type SessionManager struct {
    sessions sync.Map // clientID (string) -> *ClientSession
    count    atomic.Int64
}

func (sm *SessionManager) Add(session *ClientSession) error {
    if _, exists := sm.sessions.Load(session.ClientID); exists {
        return fmt.Errorf("session already exists")
    }
    sm.sessions.Store(session.ClientID, session)
    sm.count.Add(1)
    return nil
}
```

### 使用 atomic

```go
type Counter struct {
    value atomic.Int64
}

func (c *Counter) Increment() int64 {
    return c.value.Add(1)
}

func (c *Counter) Get() int64 {
    return c.value.Load()
}
```

### 通道使用

```go
// 使用通道进行 goroutine 通信
type Dispatcher struct {
    taskQueue chan *Task
    stopCh    chan struct{}
    wg        sync.WaitGroup
}

func (d *Dispatcher) Start() {
    d.wg.Add(1)
    go func() {
        defer d.wg.Done()
        for {
            select {
            case task := <-d.taskQueue:
                d.processTask(task)
            case <-d.stopCh:
                return
            }
        }
    }()
}

func (d *Dispatcher) Stop() {
    close(d.stopCh)
    d.wg.Wait()
}
```

### 优雅关闭

```go
func (sm *SessionManager) Stop() {
    close(sm.stopCh)
    if sm.heartbeatTick != nil {
        sm.heartbeatTick.Stop()
    }
    sm.wg.Wait()

    sm.logger.Info("SessionManager stopped")
}
```

---

## 日志规范

### 日志级别使用

```go
// Debug - 调试信息
h.logger.Debug("HTTP request",
    "method", method,
    "path", path,
    "status", statusCode,
)

// Info - 一般信息
sm.logger.Info("Session added",
    "client_id", session.ClientID,
    "remote_addr", session.RemoteAddr,
)

// Warn - 警告信息
sm.logger.Warn("Failed to cancel multi-command task",
    "task_id", taskID,
    "error", err,
)

// Error - 错误信息
h.logger.Error("HTTP server error", "error", err)
```

### 结构化日志

```go
// 使用键值对格式
logger.Info("Command sent via API",
    "command_id", cmd.CommandID,
    "client_id", req.ClientID,
    "command_type", req.CommandType,
    "timeout", timeout,
)

// 避免字符串拼接
logger.Info(fmt.Sprintf("Client %s connected", clientID)) // 不推荐
logger.Info("Client connected", "client_id", clientID)     // 推荐
```

---

## 文档注释

### 包注释

```go
// Package config 提供配置管理功能
//
// 配置加载流程：
// 1. 设置默认值
// 2. 读取配置文件（可选）
// 3. 应用环境变量覆盖
// 4. 验证配置有效性
package config
```

### 函数注释

```go
// NewSessionManager 创建新的会话管理器
//
// 参数：
//   config - 会话管理器配置，包含心跳间隔、超时等设置
//
// 返回：
//   *SessionManager - 初始化完成的会话管理器实例
//
// 使用示例：
//
//	sm := session.NewSessionManager(session.SessionManagerConfig{
//	    HeartbeatInterval: 15 * time.Second,
//	    Logger: logger,
//	})
//	sm.Start()
func NewSessionManager(config SessionManagerConfig) *SessionManager {
    // ...
}
```

### 类型注释

```go
// SessionManager 管理所有客户端会话
//
// 线程安全：所有方法都是并发安全的
// 生命周期：使用 Start() 启动，Stop() 停止
type SessionManager struct {
    sessions sync.Map
    count    atomic.Int64
    // ...
}
```

---

## 项目特定约定

### 命令处理

```go
// 在 pkg/router/handlers 中注册处理器
// 处理器函数签名：
type Handler func(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error)

// 注册处理器
r.Register(command.CmdExecShell, ExecShell)
r.Register(command.CmdGetStatus, GetStatus)
```

### 中间件注册

```go
// HTTP 服务器中间件
h.router.Use(gin.Recovery())
h.router.Use(h.loggerMiddleware())

// 自定义中间件
func (h *HTTPServer) loggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        latency := time.Since(start)
        // 记录日志
    }
}
```

### 路由分组

```go
// 使用路由组组织 API
api := h.router.Group("/api")
{
    // 子路由组
    api.GET("/clients", h.handleListClients)
    api.POST("/command", h.handleSendCommand)

    // 终端路由
    terminal := api.Group("/terminal")
    {
        terminal.GET("/ws/:client_id", tm.HandleWebSocket)
        terminal.GET("/sessions", tm.HandleTerminalSessionsList)
    }
}
```

---

## 依赖管理

### go.mod 使用

```go
module github.com/voilet/quic-flow

go 1.25.4

require (
    github.com/gin-gonic/gin v1.10.1
    github.com/quic-go/quic-go v0.58.0
    // ...
)
```

### 依赖更新

```bash
# 添加新依赖
go get github.com/package/name

# 更新所有依赖
go get -u ./...

# 整理依赖
go mod tidy
```

---

## 构建规范

### Makefile 目标

```makefile
# 标准构建
make build

# 交叉编译
make build-linux-amd64

# 运行测试
make test

# 代码检查
go vet ./...
golangci-lint run
```

---

## 最佳实践总结

1. **代码组织**：按功能模块组织，清晰分离关注点
2. **错误处理**：早期返回，不忽略错误，使用 %w 包装
3. **并发安全**：优先使用 sync.Map 和 atomic，注意 goroutine 生命周期
4. **日志记录**：使用结构化日志，合理使用日志级别
5. **测试覆盖**：表格驱动测试，子测试命名清晰
6. **API 设计**：RESTful 风格，命名一致，错误信息清晰
7. **配置管理**：默认值优先，支持多来源配置
8. **文档注释**：导出的类型和函数必须有注释
