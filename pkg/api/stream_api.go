package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/voilet/quic-flow/pkg/command"
)

// StreamCommandRequest 流式命令请求
type StreamCommandRequest struct {
	ClientIDs   []string        `json:"client_ids" binding:"required"`
	CommandType string          `json:"command_type" binding:"required"`
	Payload     json.RawMessage `json:"payload"`
	Timeout     int             `json:"timeout"` // 秒
}

// StreamCommandEvent SSE 事件
type StreamCommandEvent struct {
	Type     string                      `json:"type"` // "start", "progress", "result", "complete"
	TaskID   string                      `json:"task_id,omitempty"` // 任务ID（用于取消）
	ClientID string                      `json:"client_id,omitempty"`
	Result   *command.ClientCommandResult `json:"result,omitempty"`
	Summary  *StreamSummary              `json:"summary,omitempty"`
}

// StreamSummary 流式命令汇总
type StreamSummary struct {
	Total        int `json:"total"`
	SuccessCount int `json:"success_count"`
	FailedCount  int `json:"failed_count"`
	Duration     int `json:"duration_ms"`
}

// handleStreamMultiCommand 处理流式多播命令请求 (SSE)
func (h *HTTPServer) handleStreamMultiCommand(c *gin.Context) {
	if h.commandManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Command manager not initialized",
		})
		return
	}

	var req StreamCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if len(req.ClientIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "client_ids cannot be empty",
		})
		return
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 获取底层 ResponseWriter 并启用 Flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Streaming not supported",
		})
		return
	}

	h.logger.Info("Stream multi-command request received",
		"client_count", len(req.ClientIDs),
		"command_type", req.CommandType,
		"timeout", timeout,
	)

	startTime := time.Now()
	total := len(req.ClientIDs)

	// 使用 SendCommandToMultiple 来获取 task_id 并支持取消
	// 在 goroutine 中执行，通过 channel 接收结果
	resultChan := make(chan *command.ClientCommandResult, total)
	taskIDChan := make(chan string, 1)
	
	// 启动后台任务执行
	go func() {
		response := h.commandManager.SendCommandToMultiple(req.ClientIDs, req.CommandType, req.Payload, timeout)
		
		// 发送 task_id
		taskIDChan <- response.TaskID
		
		// 将结果发送到通道
		for _, result := range response.Results {
			resultChan <- result
	}
		close(resultChan)
	}()

	// 等待并发送开始事件（包含 task_id）
	taskID := <-taskIDChan
	startEvent := StreamCommandEvent{
		Type:   "start",
		TaskID: taskID,
	}
	data, _ := json.Marshal(startEvent)
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	flusher.Flush()

	// 流式输出结果
	var successCount, failedCount int
	for result := range resultChan {
		// 更新统计
		if result.Status == command.CommandStatusCompleted {
			successCount++
		} else {
			failedCount++
		}

		event := StreamCommandEvent{
			Type:     "result",
			TaskID:   taskID, // 每个结果也包含 task_id
			ClientID: result.ClientID,
			Result:   result,
		}

		data, _ = json.Marshal(event)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
	}

	// 发送完成事件
	duration := time.Since(startTime).Milliseconds()
	completeEvent := StreamCommandEvent{
		Type: "complete",
		Summary: &StreamSummary{
			Total:        total,
			SuccessCount: successCount,
			FailedCount:  failedCount,
			Duration:     int(duration),
		},
	}

	data, _ = json.Marshal(completeEvent)
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	flusher.Flush()

	h.logger.Info("Stream multi-command completed",
		"total", total,
		"success", successCount,
		"failed", failedCount,
		"duration_ms", duration,
	)
}

// waitForCommandResult 等待单个命令完成
func (h *HTTPServer) waitForCommandResult(commandID string, timeout time.Duration) *command.Command {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cmd, err := h.commandManager.GetCommand(commandID)
			if err != nil {
				return nil
			}

			switch cmd.Status {
			case command.CommandStatusCompleted, command.CommandStatusFailed, command.CommandStatusTimeout:
				return cmd
			}

			if time.Now().After(deadline) {
				return cmd
			}
		}
	}
}

// AddStreamRoutes 添加流式 API 路由
func (h *HTTPServer) AddStreamRoutes() {
	api := h.router.Group("/api")
	{
		api.POST("/command/stream", h.handleStreamMultiCommand)
	}
	h.logger.Info("Stream API routes registered")
}
