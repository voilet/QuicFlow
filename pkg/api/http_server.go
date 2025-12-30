package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/voilet/quic-flow/pkg/command"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/protocol"
	"github.com/voilet/quic-flow/pkg/session"
)

// ServerAPI 定义服务器需要提供的接口
type ServerAPI interface {
	ListClients() []string
	ListClientsWithDetails() []session.ClientInfoBrief
	ListClientsWithDetailsPaginated(offset, limit int) ([]session.ClientInfoBrief, int64)
	GetClientInfo(clientID string) (*protocol.ClientInfo, error)
	SendTo(clientID string, msg *protocol.DataMessage) error
	Broadcast(msg *protocol.DataMessage) (int, []error)
}

// HTTPServer HTTP API 服务器
type HTTPServer struct {
	server         *http.Server
	router         *gin.Engine
	serverAPI      ServerAPI
	commandManager *command.CommandManager
	logger         *monitoring.Logger
	listenAddr     string
}

// NewHTTPServer 创建新的 HTTP API 服务器
func NewHTTPServer(addr string, serverAPI ServerAPI, commandManager *command.CommandManager, logger *monitoring.Logger) *HTTPServer {
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
	api := h.router.Group("/api")
	{
		api.GET("/clients", h.handleListClients)
		api.GET("/clients/:id", h.handleGetClient)
		api.POST("/send", h.handleSend)
		api.POST("/broadcast", h.handleBroadcast)

		// 命令相关接口
		api.POST("/command", h.handleSendCommand)
		api.POST("/command/multi", h.handleSendMultiCommand)              // 多播命令
		api.POST("/command/multi/:id/cancel", h.handleCancelMultiCommand) // 停止多播任务
		api.GET("/command/:id", h.handleGetCommand)
		api.GET("/commands", h.handleListCommands)
	}
	h.router.GET("/health", h.handleHealth)

	h.server = &http.Server{
		Addr:    addr,
		Handler: h.router,
	}

	return h
}

// loggerMiddleware 日志中间件
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

// Start 启动 HTTP 服务器
func (h *HTTPServer) Start() error {
	h.logger.Info("Starting HTTP API server", "addr", h.listenAddr)

	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop 停止 HTTP 服务器
func (h *HTTPServer) Stop(ctx context.Context) error {
	h.logger.Info("Stopping HTTP API server...")
	return h.server.Shutdown(ctx)
}

// ClientDetail 客户端详情
type ClientDetail struct {
	ClientID    string `json:"client_id"`
	RemoteAddr  string `json:"remote_addr"`
	ConnectedAt int64  `json:"connected_at"`
	Uptime      string `json:"uptime"`
}

// ListClientsResponse 客户端列表响应
type ListClientsResponse struct {
	Total   int64          `json:"total"`
	Offset  int            `json:"offset,omitempty"`
	Limit   int            `json:"limit,omitempty"`
	Clients []ClientDetail `json:"clients"`
}

// handleListClients 处理获取客户端列表请求
// 支持分页: ?offset=0&limit=100
func (h *HTTPServer) handleListClients(c *gin.Context) {
	// 解析分页参数
	offset := 0
	limit := 0 // 0 表示全部

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	// 使用优化的一次遍历方法获取客户端详情
	var clients []session.ClientInfoBrief
	var total int64

	if limit > 0 {
		// 分页查询
		clients, total = h.serverAPI.ListClientsWithDetailsPaginated(offset, limit)
	} else {
		// 全量查询
		clients = h.serverAPI.ListClientsWithDetails()
		total = int64(len(clients))
	}

	// 转换为响应格式（添加 uptime 计算）
	now := time.Now()
	details := make([]ClientDetail, len(clients))
	for i, client := range clients {
		uptime := now.Sub(time.UnixMilli(client.ConnectedAt))
		details[i] = ClientDetail{
			ClientID:    client.ClientID,
			RemoteAddr:  client.RemoteAddr,
			ConnectedAt: client.ConnectedAt,
			Uptime:      uptime.Round(time.Second).String(),
		}
	}

	response := ListClientsResponse{
		Total:   total,
		Clients: details,
	}

	// 只在分页时返回分页信息
	if limit > 0 {
		response.Offset = offset
		response.Limit = limit
	}

	c.JSON(http.StatusOK, response)
}

// handleGetClient 处理获取单个客户端信息请求
func (h *HTTPServer) handleGetClient(c *gin.Context) {
	clientID := c.Param("id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Client ID is required",
		})
		return
	}

	info, err := h.serverAPI.GetClientInfo(clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Client not found: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

// SendRequest 发送消息请求结构
type SendRequest struct {
	ClientID string `json:"client_id" binding:"required"`
	Type     string `json:"type"`
	Payload  string `json:"payload" binding:"required"`
	WaitAck  bool   `json:"wait_ack"`
}

// SendResponse 发送消息响应
type SendResponse struct {
	Success bool   `json:"success"`
	MsgID   string `json:"msg_id"`
	Message string `json:"message"`
}

// handleSend 处理发送消息请求
func (h *HTTPServer) handleSend(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// 解析消息类型
	msgType := protocol.MessageType_MESSAGE_TYPE_COMMAND
	switch req.Type {
	case "command":
		msgType = protocol.MessageType_MESSAGE_TYPE_COMMAND
	case "event":
		msgType = protocol.MessageType_MESSAGE_TYPE_EVENT
	case "query":
		msgType = protocol.MessageType_MESSAGE_TYPE_QUERY
	case "response":
		msgType = protocol.MessageType_MESSAGE_TYPE_RESPONSE
	case "":
		// 使用默认值 command
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid message type: %s", req.Type),
		})
		return
	}

	// 构造消息
	msg := &protocol.DataMessage{
		MsgId:      uuid.New().String(),
		SenderId:   "server",
		ReceiverId: req.ClientID,
		Type:       msgType,
		Payload:    []byte(req.Payload),
		WaitAck:    req.WaitAck,
		Timestamp:  time.Now().UnixMilli(),
	}

	// 发送消息
	if err := h.serverAPI.SendTo(req.ClientID, msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to send message: %v", err),
		})
		return
	}

	h.logger.Info("Message sent via API", "client_id", req.ClientID, "msg_id", msg.MsgId, "type", req.Type)

	c.JSON(http.StatusOK, SendResponse{
		Success: true,
		MsgID:   msg.MsgId,
		Message: "Message sent successfully",
	})
}

// BroadcastRequest 广播消息请求结构
type BroadcastRequest struct {
	Type    string `json:"type"`
	Payload string `json:"payload" binding:"required"`
}

// BroadcastResponse 广播消息响应
type BroadcastResponse struct {
	Success      bool     `json:"success"`
	MsgID        string   `json:"msg_id"`
	Total        int      `json:"total"`
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	Errors       []string `json:"errors,omitempty"`
}

// handleBroadcast 处理广播消息请求
func (h *HTTPServer) handleBroadcast(c *gin.Context) {
	var req BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// 解析消息类型
	msgType := protocol.MessageType_MESSAGE_TYPE_EVENT
	switch req.Type {
	case "command":
		msgType = protocol.MessageType_MESSAGE_TYPE_COMMAND
	case "event":
		msgType = protocol.MessageType_MESSAGE_TYPE_EVENT
	case "query":
		msgType = protocol.MessageType_MESSAGE_TYPE_QUERY
	case "response":
		msgType = protocol.MessageType_MESSAGE_TYPE_RESPONSE
	case "":
		// 使用默认值 event
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid message type: %s", req.Type),
		})
		return
	}

	// 构造消息
	msg := &protocol.DataMessage{
		MsgId:     uuid.New().String(),
		SenderId:  "server",
		Type:      msgType,
		Payload:   []byte(req.Payload),
		WaitAck:   false,
		Timestamp: time.Now().UnixMilli(),
	}

	// 广播消息
	successCount, errors := h.serverAPI.Broadcast(msg)

	h.logger.Info("Message broadcast via API", "msg_id", msg.MsgId, "type", req.Type, "success", successCount, "failed", len(errors))

	response := BroadcastResponse{
		Success:      true,
		MsgID:        msg.MsgId,
		Total:        successCount + len(errors),
		SuccessCount: successCount,
		FailedCount:  len(errors),
	}

	if len(errors) > 0 {
		errMsgs := make([]string, len(errors))
		for i, err := range errors {
			errMsgs[i] = err.Error()
		}
		response.Errors = errMsgs
	}

	c.JSON(http.StatusOK, response)
}

// handleHealth 健康检查
func (h *HTTPServer) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// handleSendCommand 处理下发命令请求
func (h *HTTPServer) handleSendCommand(c *gin.Context) {
	if h.commandManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Command manager not initialized",
		})
		return
	}

	var req command.CommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// 设置默认超时
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// 下发命令
	cmd, err := h.commandManager.SendCommand(req.ClientID, req.CommandType, req.Payload, timeout)
	if err != nil {
		h.logger.Error("Failed to send command",
			"client_id", req.ClientID,
			"command_type", req.CommandType,
			"error", err,
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to send command: %v", err),
		})
		return
	}

	h.logger.Info("Command sent via API",
		"command_id", cmd.CommandID,
		"client_id", req.ClientID,
		"command_type", req.CommandType,
		"timeout", timeout,
	)

	c.JSON(http.StatusOK, command.CommandResponse{
		Success:   true,
		CommandID: cmd.CommandID,
		Message:   "Command sent successfully",
	})
}

// handleGetCommand 处理查询命令状态请求
func (h *HTTPServer) handleGetCommand(c *gin.Context) {
	if h.commandManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Command manager not initialized",
		})
		return
	}

	commandID := c.Param("id")
	if commandID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Command ID is required",
		})
		return
	}

	cmd, err := h.commandManager.GetCommand(commandID)
	if err != nil {
		c.JSON(http.StatusNotFound, command.CommandStatusResponse{
			Success: false,
			Error:   fmt.Sprintf("Command not found: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, command.CommandStatusResponse{
		Success: true,
		Command: cmd,
	})
}

// ListCommandsRequest 查询命令列表请求
type ListCommandsRequest struct {
	ClientID string                `form:"client_id"` // 可选：按客户端ID过滤
	Status   command.CommandStatus `form:"status"`    // 可选：按状态过滤
}

// ListCommandsResponse 查询命令列表响应
type ListCommandsResponse struct {
	Success  bool               `json:"success"`
	Total    int                `json:"total"`
	Commands []*command.Command `json:"commands"`
}

// handleListCommands 处理查询命令列表请求
func (h *HTTPServer) handleListCommands(c *gin.Context) {
	if h.commandManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Command manager not initialized",
		})
		return
	}

	var req ListCommandsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid query parameters: %v", err),
		})
		return
	}

	commands := h.commandManager.ListCommands(req.ClientID, req.Status)

	c.JSON(http.StatusOK, ListCommandsResponse{
		Success:  true,
		Total:    len(commands),
		Commands: commands,
	})
}

// handleSendMultiCommand 处理多播命令请求（同时下发到多个客户端）
func (h *HTTPServer) handleSendMultiCommand(c *gin.Context) {
	if h.commandManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Command manager not initialized",
		})
		return
	}

	var req command.MultiCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// 验证客户端列表
	if len(req.ClientIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "client_ids cannot be empty",
		})
		return
	}

	// 设置默认超时
	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	h.logger.Info("Multi-command request received",
		"client_count", len(req.ClientIDs),
		"command_type", req.CommandType,
		"timeout", timeout,
	)

	// 下发多播命令
	response := h.commandManager.SendCommandToMultiple(req.ClientIDs, req.CommandType, req.Payload, timeout)

	h.logger.Info("Multi-command completed",
		"total", response.Total,
		"success", response.SuccessCount,
		"failed", response.FailedCount,
	)

	c.JSON(http.StatusOK, response)
}

// handleCancelMultiCommand 处理停止多播任务请求
func (h *HTTPServer) handleCancelMultiCommand(c *gin.Context) {
	if h.commandManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Command manager not initialized",
		})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}

	// 取消任务
	err := h.commandManager.CancelMultiCommand(taskID)
	if err != nil {
		h.logger.Warn("Failed to cancel multi-command task",
			"task_id", taskID,
			"error", err,
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Failed to cancel task: %v", err),
		})
		return
	}

	h.logger.Info("Multi-command task cancelled via API",
		"task_id", taskID,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"task_id": taskID,
		"message": "Task cancelled successfully",
	})
}
