package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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
	Type     string                      `json:"type"` // "progress", "result", "complete"
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
	var successCount, failedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 结果通道
	resultChan := make(chan *command.ClientCommandResult, total)

	// 并行发送命令
	for _, clientID := range req.ClientIDs {
		wg.Add(1)
		go func(cid string) {
			defer wg.Done()

			result := &command.ClientCommandResult{
				ClientID: cid,
				Status:   command.CommandStatusPending,
			}

			cmd, err := h.commandManager.SendCommand(cid, req.CommandType, req.Payload, timeout)
			if err != nil {
				result.Status = command.CommandStatusFailed
				result.Error = err.Error()
			} else {
				result.CommandID = cmd.CommandID
				// 等待命令完成
				finalCmd := h.waitForCommandResult(cmd.CommandID, timeout+5*time.Second)
				if finalCmd != nil {
					result.Status = finalCmd.Status
					result.Result = finalCmd.Result
					result.Error = finalCmd.Error
				} else {
					result.Status = command.CommandStatusTimeout
					result.Error = "timeout waiting for result"
				}
			}

			// 更新统计
			mu.Lock()
			if result.Status == command.CommandStatusCompleted {
				successCount++
			} else {
				failedCount++
			}
			mu.Unlock()

			// 发送到结果通道
			resultChan <- result
		}(clientID)
	}

	// 启动关闭通道的 goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 流式输出结果
	for result := range resultChan {
		event := StreamCommandEvent{
			Type:     "result",
			ClientID: result.ClientID,
			Result:   result,
		}

		data, _ := json.Marshal(event)
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

	data, _ := json.Marshal(completeEvent)
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
