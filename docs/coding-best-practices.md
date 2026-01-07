# QUIC Flow 编码最佳实践

> 基于项目实际代码总结的 Go 编码模式和惯例

## 目录

1. [初始化模式](#初始化模式)
2. [生命周期管理](#生命周期管理)
3. [配置转换器](#配置转换器)
4. [接口设计](#接口设计)
5. [错误包装模式](#错误包装模式)
6. [资源清理](#资源清理)
7. [可测试性设计](#可测试性设计)

---

## 初始化模式

### 构造函数模式

```go
// 标准构造函数：New + 类型名
func NewSessionManager(config SessionManagerConfig) *SessionManager {
    // 设置默认值
    if config.HeartbeatCheckInterval == 0 {
        config.HeartbeatCheckInterval = 5 * time.Second
    }
    if config.Logger == nil {
        config.Logger = monitoring.NewDefaultLogger()
    }

    sm := &SessionManager{
        heartbeatInterval: config.HeartbeatCheckInterval,
        logger:            config.Logger,
        stopCh:            make(chan struct{}),
    }

    return sm
}

// 带验证的构造函数
func NewHTTPServer(
    addr string,
    serverAPI ServerAPI,
    commandManager *command.CommandManager,
    logger *monitoring.Logger,
) *HTTPServer {
    // 设置 Gin 模式
    gin.SetMode(gin.ReleaseMode)

    h := &HTTPServer{
        router:         gin.New(),
        serverAPI:      serverAPI,
        commandManager: commandManager,
        logger:         logger,
        listenAddr:     addr,
    }

    // 添加中间件
    h.router.Use(gin.Recovery())
    h.router.Use(h.loggerMiddleware())

    // 注册路由
    h.registerRoutes()

    // 创建 HTTP 服务器
    h.server = &http.Server{
        Addr:    addr,
        Handler: h.router,
    }

    return h
}
```

### 配置结构模式

```go
// 使用专用配置结构而非多参数
type SessionManagerConfig struct {
    HeartbeatCheckInterval time.Duration // 心跳检查间隔
    HeartbeatTimeout       time.Duration // 心跳超时阈值
    MaxTimeoutCount        int32         // 最大超时次数
    Hooks                  *monitoring.EventHooks
    Logger                 *monitoring.Logger
}

// 调用时清晰易懂
sm := session.NewSessionManager(session.SessionManagerConfig{
    HeartbeatInterval: 15 * time.Second,
    Logger:            logger,
})
```

---

## 生命周期管理

### 启动/停止模式

```go
// Start 启动服务（非阻塞）
func (sm *SessionManager) Start() {
    sm.heartbeatTick = time.NewTicker(sm.heartbeatInterval)
    sm.wg.Add(1)

    go sm.heartbeatChecker()

    sm.logger.Info("SessionManager started",
        "heartbeat_interval", sm.heartbeatInterval)
}

// Stop 优雅停止
func (sm *SessionManager) Stop() {
    close(sm.stopCh)
    if sm.heartbeatTick != nil {
        sm.heartbeatTick.Stop()
    }
    sm.wg.Wait()

    sm.logger.Info("SessionManager stopped")
}

// 使用模式
sm := session.NewSessionManager(config)
sm.Start()
defer sm.Stop() // 确保停止
```

### HTTP 服务器启动

```go
// Start 异步启动
func (h *HTTPServer) Start() error {
    h.logger.Info("Starting HTTP API server", "addr", h.listenAddr)

    go func() {
        if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            h.logger.Error("HTTP server error", "error", err)
        }
    }()

    return nil
}

// Stop 优雅关闭
func (h *HTTPServer) Stop(ctx context.Context) error {
    h.logger.Info("Stopping HTTP API server...")
    return h.server.Shutdown(ctx)
}
```

### 后台 goroutine 管理

```go
type Service struct {
    stopCh chan struct{}
    wg     sync.WaitGroup
}

func (s *Service) Start() {
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                s.doWork()
            case <-s.stopCh:
                return
            }
        }
    }()
}

func (s *Service) Stop() {
    close(s.stopCh)
    s.wg.Wait()
}
```

---

## 配置转换器

### 类型转换方法

```go
// 配置结构提供便捷的转换方法
type ServerConfig struct {
    Session SessionSettings `mapstructure:"session"`
    Batch   BatchSettings   `mapstructure:"batch"`
}

// 转换为 time.Duration
func (c *ServerConfig) GetHeartbeatInterval() time.Duration {
    return time.Duration(c.Session.HeartbeatInterval) * time.Second
}

func (c *ServerConfig) GetHeartbeatTimeout() time.Duration {
    return time.Duration(c.Session.HeartbeatTimeout) * time.Second
}

func (c *ServerConfig) GetTaskTimeout() time.Duration {
    return time.Duration(c.Batch.TaskTimeout) * time.Second
}

// 使用
interval := cfg.GetHeartbeatInterval()
timeout := cfg.GetTaskTimeout()
```

---

## 接口设计

### 小接口原则

```go
// 定义最小必要的接口
type ServerAPI interface {
    ListClients() []string
    ListClientsWithDetails() []session.ClientInfoBrief
    GetClientInfo(clientID string) (*protocol.ClientInfo, error)
    SendTo(clientID string, msg *protocol.DataMessage) error
    Broadcast(msg *protocol.DataMessage) (int, []error)
}

// 单方法接口便于 mock 和测试
type ReleaseAPIRegistrar interface {
    RegisterRoutes(r *gin.RouterGroup)
}

// 实现
func (h *HTTPServer) AddReleaseRoutes(releaseAPI ReleaseAPIRegistrar) {
    api := h.router.Group("/api")
    releaseAPI.RegisterRoutes(api)
}
```

### 函数式选项模式

```go
// 复杂配置使用选项模式
type Option func(*Server)

func WithLogger(logger *Logger) Option {
    return func(s *Server) {
        s.logger = logger
    }
}

func WithTimeout(timeout time.Duration) Option {
    return func(s *Server) {
        s.timeout = timeout
    }
}

func NewServer(addr string, opts ...Option) *Server {
    s := &Server{
        addr:    addr,
        timeout: 30 * time.Second,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// 使用
s := NewServer(":8080",
    WithLogger(logger),
    WithTimeout(60*time.Second),
)
```

---

## 错误包装模式

### 错误包装链

```go
// 使用 %w 保留错误链
if err != nil {
    return fmt.Errorf("failed to read config file %s: %w", configPath, err)
}

// 使用项目自定义错误
import pkgerrors "github.com/voilet/quic-flow/pkg/errors"

if session == nil {
    return fmt.Errorf("%w: session is nil", pkgerrors.ErrInvalidConfig)
}

// 检查特定错误
if errors.Is(err, pkgerrors.ErrSessionNotFound) {
    // 处理会话不存在
}
```

### 错误变量定义

```go
// 在包中定义可重用的错误
var (
    ErrInvalidConfig      = errors.New("invalid configuration")
    ErrSessionNotFound    = errors.New("session not found")
    ErrSessionAlreadyExists = errors.New("session already exists")
    ErrInvalidClientID    = errors.New("invalid client ID")
)

// 使用时包装提供上下文
return fmt.Errorf("%w: %s", ErrSessionNotFound, clientID)
```

---

## 资源清理

### defer 使用

```go
func ProcessFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close() // 确保文件关闭

    // 处理文件
    return nil
}

// 多个 defer 按 LIFO 顺序执行
func Process() error {
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        return err
    }
    defer db.Close()

    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback() // 如果未提交则回滚

    // 执行操作
    if err := doWork(tx); err != nil {
        return err
    }

    return tx.Commit() // 提交后 defer 的 Rollback 无效果
}
```

### Context 超时处理

```go
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
    return context.WithTimeout(ctx, timeout)
}

// 使用
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := ch.Send(ctx, msg)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // 处理超时
    }
    return err
}
```

---

## 可测试性设计

### 依赖注入

```go
// 使用接口而非具体类型
type HTTPServer struct {
    serverAPI      ServerAPI      // 接口，便于 mock
    commandManager *command.CommandManager
    logger         *monitoring.Logger
}

// 构造时注入依赖
func NewHTTPServer(
    addr string,
    serverAPI ServerAPI,
    commandManager *command.CommandManager,
    logger *monitoring.Logger,
) *HTTPServer {
    // ...
}
```

### 可配置行为

```go
// 使用配置结构控制行为
type Config struct {
    Logger            *monitoring.Logger
    HardwareStore     *hardware.Store // 可选，允许为 nil
}

func (h *HTTPServer) handleListClients(c *gin.Context) {
    // 检查可选依赖
    if h.hardwareStore != nil {
        // 使用数据库
        devices, total, err := h.hardwareStore.ListDevices(offset, limit)
        // ...
    } else {
        // 降级处理
        clients := h.serverAPI.ListClientsWithDetails()
        // ...
    }
}
```

---

## Gin 框架特定模式

### 中间件模式

```go
// 日志中间件
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

// 使用
h.router.Use(gin.Recovery())
h.router.Use(h.loggerMiddleware())
```

### 路由分组

```go
// 路由分组器模式
func (h *HTTPServer) GetAPIGroup() *gin.RouterGroup {
    return h.router.Group("/api")
}

// 外部注册路由
func (h *HTTPServer) AddTerminalRoutes(tm *TerminalManager) {
    api := h.router.Group("/api/terminal")
    {
        api.GET("/ws/:client_id", tm.HandleWebSocket)
        api.GET("/sessions", tm.HandleTerminalSessionsList)
        api.DELETE("/sessions/:session_id", tm.HandleTerminalSessionClose)
    }
}
```

### 参数解析

```go
// 路径参数
func (h *HTTPServer) handleGetClient(c *gin.Context) {
    clientID := c.Param("id")
    if clientID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID is required"})
        return
    }
    // ...
}

// 查询参数
func (h *HTTPServer) handleListClients(c *gin.Context) {
    offset := 0
    if offsetStr := c.Query("offset"); offsetStr != "" {
        if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
            offset = v
        }
    }
    // ...
}

// JSON 请求体
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

---

## GORM 数据库模式

### 模型定义

```go
type Device struct {
    // 主键
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

    // 业务字段
    ClientID string `gorm:"uniqueIndex;not null" json:"client_id"`
    Hostname string `gorm:"type:varchar(255)" json:"hostname"`
    OS       string `gorm:"type:varchar(50)" json:"os"`

    // 关联
    FullHardwareInfo HardwareInfo `gorm:"embedded;embeddedPrefix:hw_" json:"hardware_info"`
}

// 表名指定
func (Device) TableName() string {
    return "devices"
}
```

### 查询模式

```go
// 带分页的查询
func (s *Store) ListDevices(offset, limit int) ([]Device, int64, error) {
    var devices []Device
    var total int64

    query := s.db.Model(&Device{})

    // 计算总数
    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // 分页查询
    if err := query.Offset(offset).Limit(limit).Find(&devices).Error; err != nil {
        return nil, 0, err
    }

    return devices, total, nil
}

// 条件查询
func (s *Store) ListDevicesByStatus(status string, offset, limit int) ([]Device, int64, error) {
    var devices []Device
    var total int64

    query := s.db.Model(&Device{}).Where("status = ?", status)

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    if err := query.Offset(offset).Limit(limit).Find(&devices).Error; err != nil {
        return nil, 0, err
    }

    return devices, total, nil
}
```

---

## Protobuf 消息处理

### 消息构造

```go
import "github.com/voilet/quic-flow/pkg/protocol"

// 构造消息
msg := &protocol.DataMessage{
    MsgId:      uuid.New().String(),
    SenderId:   "server",
    ReceiverId: req.ClientID,
    Type:       protocol.MessageType_MESSAGE_TYPE_COMMAND,
    Payload:    []byte(req.Payload),
    WaitAck:    req.WaitAck,
    Timestamp:  time.Now().UnixMilli(),
}

// 发送消息
if err := h.serverAPI.SendTo(req.ClientID, msg); err != nil {
    return fmt.Errorf("failed to send message: %w", err)
}
```

### 消息类型转换

```go
// 消息类型转换
msgType := protocol.MessageType_MESSAGE_TYPE_COMMAND
switch req.Type {
case "command":
    msgType = protocol.MessageType_MESSAGE_TYPE_COMMAND
case "event":
    msgType = protocol.MessageType_MESSAGE_TYPE_EVENT
case "query":
    msgType = protocol.MessageType_MESSAGE_TYPE_QUERY
case "":
    // 使用默认值
default:
    return fmt.Errorf("invalid message type: %s", req.Type)
}
```

---

## 代码组织技巧

### 分组注释

```go
// 使用注释分隔代码区域
type ServerConfig struct {
    // === 服务器基础配置 ===
    Addr       string `mapstructure:"addr"`
    APIAddr    string `mapstructure:"api_addr"`
    HighPerf   bool   `mapstructure:"high_perf"`

    // === TLS 配置 ===
    CertFile string `mapstructure:"cert_file"`
    KeyFile  string `mapstructure:"key_file"`

    // === QUIC 配置 ===
    MaxIdleTimeout int `mapstructure:"max_idle_timeout"`
}

// 函数内部分组
func process() {
    // 1. 验证输入
    if err := validate(); err != nil {
        return err
    }

    // 2. 处理数据
    result := doWork()

    // 3. 返回结果
    return result
}
```

### 导出分组

```go
const (
    // === 消息类型 ===
    MsgTypeData     = 0
    MsgTypeSSH      = 1
    MsgTypeFile     = 2

    // === 状态码 ===
    StatusOK       = 200
    StatusError    = 500
)

// 或使用注释分组
const (
    CmdExecShell    = "exec_shell"     // 执行 Shell 命令
    CmdGetStatus    = "get_status"     // 获取状态
    CmdHardwareInfo = "hardware_info"  // 获取硬件信息
)
```

---

## 常用工具函数

### UUID 生成

```go
import "github.com/google/uuid"

// 生成 UUID
msgID := uuid.New().String()
clientID := uuid.New().String()

// 解析 UUID
parsed, err := uuid.Parse(msgID)
```

### 时间处理

```go
import "time"

// 当前时间戳（毫秒）
timestamp := time.Now().UnixMilli()

// 时间间隔
interval := 30 * time.Second
timeout := 5 * time.Minute

// 时间解析
duration, err := time.ParseDuration("30s")

// 格式化时间
formatted := time.Now().Format("2006-01-02 15:04:05")
```

### JSON 处理

```go
// 编码
payloadBytes, err := json.Marshal(params)
if err != nil {
    return fmt.Errorf("failed to marshal params: %w", err)
}

// 解码
var req SendRequest
if err := json.Unmarshal(data, &req); err != nil {
    return fmt.Errorf("failed to unmarshal request: %w", err)
}
```

---

## 调试技巧

### 结构化输出

```go
// 使用 %+v 输出详细结构
fmt.Printf("Config: %+v\n", cfg)

// 使用 #v 输出 Go 语法格式
fmt.Printf("Message: %#v\n", msg)

// 使用 JSON.Marshal 输出可读 JSON
if data, err := json.MarshalIndent(obj, "", "  "); err == nil {
    fmt.Println(string(data))
}
```

### 条件编译

```go
// 调试代码
//go:build debug

func init() {
    log.SetLevel(log.DebugLevel)
}
```

---

## 性能优化提示

### sync.Map vs map + mutex

```go
// 使用 sync.Map 当：
// 1. 键集合稳定（主要是读操作）
// 2. 多 goroutine 并发访问
// 3. 不需要原子操作

// 使用 map + mutex 当：
// 1. 需要复杂的键值操作
// 2. 需要事务性操作
// 3. 性能关键且有锁竞争
```

### 字符串拼接

```go
// 避免
s := ""
for i := 0; i < 100; i++ {
    s += fmt.Sprintf("item %d, ", i)
}

// 推荐
var b strings.Builder
for i := 0; i < 100; i++ {
    b.WriteString(fmt.Sprintf("item %d, ", i))
}
s := b.String()
```

### 内存复用

```go
// 使用 bytes.Buffer 或 sync.Pool
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processData(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    // 使用 buf...
}
```

---

## 安全最佳实践

### 输入验证

```go
// 验证客户端 ID
if clientID == "" {
    return pkgerrors.ErrInvalidClientID
}

// 验证范围
if offset < 0 || limit <= 0 || limit > 1000 {
    return errors.New("invalid pagination parameters")
}

// 验证枚举
switch req.Type {
case "command", "event", "query":
    // 有效
default:
    return fmt.Errorf("invalid message type: %s", req.Type)
}
```

### 敏感信息处理

```go
// 不要在日志中输出敏感信息
logger.Info("User login", "username", username) // 正确
logger.Info("User login", "password", password) // 错误！

// 错误中不包含敏感路径
return fmt.Errorf("failed to read config")      // 正确
return fmt.Errorf("failed to read %s", path)    // 可能泄露路径
```

---

## 文档生成

```bash
# 生成 godoc
go doc github.com/voilet/quic-flow/pkg/config

# 启动本地文档服务器
godoc -http=:6060

# 访问
# http://localhost:6060/pkg/github.com/voilet/quic-flow/
```
