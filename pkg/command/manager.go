package command

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/voilet/quic-flow/pkg/callback"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"github.com/voilet/quic-flow/pkg/protocol"
)

// ServerAPI 服务器接口（用于发送消息）
type ServerAPI interface {
	SendTo(clientID string, msg *protocol.DataMessage) error
	SendToWithPromise(clientID string, msg *protocol.DataMessage, timeout time.Duration) (*callback.Promise, error)
}

// MultiCommandTask 多播任务信息
type MultiCommandTask struct {
	TaskID     string
	ClientIDs  []string
	CommandIDs []string // 已发送的命令ID列表
	CancelFunc context.CancelFunc
	Status     string // running/completed/cancelled
	CreatedAt  time.Time
	mu         sync.RWMutex
}

// CommandManager 命令管理器（服务端）
type CommandManager struct {
	server ServerAPI
	logger *monitoring.Logger

	// 命令存储
	commands map[string]*Command // commandID -> Command
	mu       sync.RWMutex

	// 多播任务跟踪
	multiTasks map[string]*MultiCommandTask // taskID -> MultiCommandTask
	tasksMu    sync.RWMutex

	// 清理配置
	cleanupInterval time.Duration
	maxCommandAge   time.Duration

	// 控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCommandManager 创建命令管理器
func NewCommandManager(server ServerAPI, logger *monitoring.Logger) *CommandManager {
	ctx, cancel := context.WithCancel(context.Background())

	cm := &CommandManager{
		server:          server,
		logger:          logger,
		commands:        make(map[string]*Command),
		multiTasks:      make(map[string]*MultiCommandTask),
		cleanupInterval: 5 * time.Minute,  // 每5分钟清理一次
		maxCommandAge:   30 * time.Minute, // 保留30分钟的命令历史

		ctx:    ctx,
		cancel: cancel,
	}

	// 启动清理任务
	cm.wg.Add(1)
	go cm.cleanupLoop()

	return cm
}

// SendCommand 下发命令到客户端
func (cm *CommandManager) SendCommand(clientID, commandType string, payload json.RawMessage, timeout time.Duration) (*Command, error) {
	if timeout == 0 {
		timeout = 30 * time.Second // 默认30秒超时
	}

	// 创建命令记录
	commandID := uuid.New().String()
	cmd := &Command{
		CommandID:   commandID,
		ClientID:    clientID,
		CommandType: commandType,
		Payload:     payload,
		Status:      CommandStatusPending,
		CreatedAt:   time.Now(),
		Timeout:     timeout,
	}

	// 存储命令
	cm.mu.Lock()
	cm.commands[commandID] = cmd
	cm.mu.Unlock()

	// 构造命令载荷
	cmdPayload := CommandPayload{
		CommandType: commandType,
		Payload:     payload,
	}
	payloadBytes, err := json.Marshal(cmdPayload)
	if err != nil {
		cm.updateCommandStatus(commandID, CommandStatusFailed, nil, fmt.Sprintf("marshal payload failed: %v", err))
		return cmd, fmt.Errorf("marshal command payload: %w", err)
	}

	// 构造消息
	now := time.Now()
	msg := &protocol.DataMessage{
		MsgId:      commandID,
		SenderId:   "server",
		ReceiverId: clientID,
		Type:       protocol.MessageType_MESSAGE_TYPE_COMMAND,
		Payload:    payloadBytes,
		WaitAck:    true,
		Timestamp:  now.UnixMilli(),
	}

	// 发送消息并创建Promise
	promise, err := cm.server.SendToWithPromise(clientID, msg, timeout)
	if err != nil {
		cm.updateCommandStatus(commandID, CommandStatusFailed, nil, fmt.Sprintf("send command failed: %v", err))
		return cmd, fmt.Errorf("send command to client: %w", err)
	}

	// 更新发送时间
	cmd.SentAt = &now

	cm.logger.Info("Command sent to client",
		"command_id", commandID,
		"client_id", clientID,
		"command_type", commandType,
		"timeout", timeout,
	)

	// 启动goroutine等待响应
	cm.wg.Add(1)
	go cm.waitForCommandResponse(cmd, promise)

	return cmd, nil
}

// waitForCommandResponse 等待命令响应
func (cm *CommandManager) waitForCommandResponse(cmd *Command, promise *callback.Promise) {
	defer cm.wg.Done()

	// 等待响应
	select {
	case resp := <-promise.RespChan:
		if resp.Error != nil {
			// 超时或其他错误
			cm.logger.Warn("Command execution failed",
				"command_id", cmd.CommandID,
				"error", resp.Error,
			)
			cm.updateCommandStatus(cmd.CommandID, CommandStatusTimeout, nil, resp.Error.Error())
		} else if resp.AckMessage != nil {
			// 收到Ack响应
			ack := resp.AckMessage
			cm.logger.Info("Command execution completed",
				"command_id", cmd.CommandID,
				"status", ack.Status,
			)

			// 根据Ack状态更新命令状态
			switch ack.Status {
			case protocol.AckStatus_ACK_STATUS_SUCCESS:
				cm.updateCommandStatus(cmd.CommandID, CommandStatusCompleted, ack.Result, "")
			case protocol.AckStatus_ACK_STATUS_FAILURE:
				cm.updateCommandStatus(cmd.CommandID, CommandStatusFailed, ack.Result, ack.Error)
			case protocol.AckStatus_ACK_STATUS_TIMEOUT:
				cm.updateCommandStatus(cmd.CommandID, CommandStatusTimeout, nil, ack.Error)
			default:
				cm.updateCommandStatus(cmd.CommandID, CommandStatusFailed, nil, "unknown ack status")
			}
		}

	case <-cm.ctx.Done():
		return
	}
}

// updateCommandStatus 更新命令状态
func (cm *CommandManager) updateCommandStatus(commandID string, status CommandStatus, result []byte, errMsg string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cmd, exists := cm.commands[commandID]
	if !exists {
		return
	}

	cmd.Status = status
	if result != nil {
		cmd.Result = json.RawMessage(result)
	}
	if errMsg != "" {
		cmd.Error = errMsg
	}

	// 如果是终态，记录完成时间
	if status == CommandStatusCompleted || status == CommandStatusFailed || status == CommandStatusTimeout || status == CommandStatusCancelled {
		now := time.Now()
		cmd.CompletedAt = &now
	}
}

// GetCommand 查询命令状态
func (cm *CommandManager) GetCommand(commandID string) (*Command, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cmd, exists := cm.commands[commandID]
	if !exists {
		return nil, fmt.Errorf("command not found: %s", commandID)
	}

	// 返回副本，避免并发修改
	cmdCopy := *cmd
	return &cmdCopy, nil
}

// ListCommands 列出所有命令（可选过滤条件）
func (cm *CommandManager) ListCommands(clientID string, status CommandStatus) []*Command {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []*Command
	for _, cmd := range cm.commands {
		// 过滤条件
		if clientID != "" && cmd.ClientID != clientID {
			continue
		}
		if status != "" && cmd.Status != status {
			continue
		}

		cmdCopy := *cmd
		result = append(result, &cmdCopy)
	}

	return result
}

// cleanupLoop 定期清理过期命令
func (cm *CommandManager) cleanupLoop() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.cleanup()
		case <-cm.ctx.Done():
			return
		}
	}
}

// cleanup 清理过期命令
func (cm *CommandManager) cleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	var toDelete []string

	for commandID, cmd := range cm.commands {
		// 只清理已完成的命令
		if cmd.Status != CommandStatusCompleted && cmd.Status != CommandStatusFailed && cmd.Status != CommandStatusTimeout && cmd.Status != CommandStatusCancelled {
			continue
		}

		// 检查是否过期
		if cmd.CompletedAt != nil && now.Sub(*cmd.CompletedAt) > cm.maxCommandAge {
			toDelete = append(toDelete, commandID)
		}
	}

	// 删除过期命令
	for _, commandID := range toDelete {
		delete(cm.commands, commandID)
	}

	if len(toDelete) > 0 {
		cm.logger.Debug("Cleaned up expired commands", "count", len(toDelete))
	}
}

// Stop 停止命令管理器
func (cm *CommandManager) Stop() {
	cm.logger.Info("Stopping command manager...")
	cm.cancel()
	cm.wg.Wait()
	cm.logger.Info("Command manager stopped")
}

// SendCommandToMultiple 同时下发命令到多个客户端（多播）
// 并行发送命令并等待所有响应，支持取消
func (cm *CommandManager) SendCommandToMultiple(clientIDs []string, commandType string, payload json.RawMessage, timeout time.Duration) *MultiCommandResponse {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// 创建任务上下文和取消函数
	taskID := uuid.New().String()
	taskCtx, taskCancel := context.WithCancel(cm.ctx)

	// 创建任务跟踪
	task := &MultiCommandTask{
		TaskID:     taskID,
		ClientIDs:  clientIDs,
		CommandIDs: make([]string, 0, len(clientIDs)),
		CancelFunc: taskCancel,
		Status:     "running",
		CreatedAt:  time.Now(),
	}

	// 注册任务
	cm.tasksMu.Lock()
	cm.multiTasks[taskID] = task
	cm.tasksMu.Unlock()

	// 任务完成后清理
	defer func() {
		cm.tasksMu.Lock()
		if t, ok := cm.multiTasks[taskID]; ok {
			t.mu.Lock()
			if t.Status == "running" {
				t.Status = "completed"
			}
			t.mu.Unlock()
		}
		// 延迟删除任务（保留一段时间供查询）
		go func() {
			time.Sleep(5 * time.Minute)
			cm.tasksMu.Lock()
			delete(cm.multiTasks, taskID)
			cm.tasksMu.Unlock()
		}()
		cm.tasksMu.Unlock()
	}()

	total := len(clientIDs)
	results := make([]*ClientCommandResult, total)
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	cancelledCount := 0

	cm.logger.Info("Sending command to multiple clients",
		"task_id", taskID,
		"client_count", total,
		"command_type", commandType,
		"timeout", timeout,
	)

	// 并行发送命令到所有客户端
	for i, clientID := range clientIDs {
		// 检查是否已取消
		select {
		case <-taskCtx.Done():
			cm.logger.Info("Multi-command task cancelled",
				"task_id", taskID,
				"sent_count", i,
				"total", total,
			)
			// 标记未发送的命令为取消状态
			for j := i; j < total; j++ {
				results[j] = &ClientCommandResult{
					ClientID: clientIDs[j],
					Status:   CommandStatusCancelled,
					Error:    "Task cancelled before sending",
				}
			}
			mu.Lock()
			cancelledCount = total - i
			mu.Unlock()
			goto done
		default:
		}

		wg.Add(1)
		go func(index int, cid string) {
			defer wg.Done()

			result := &ClientCommandResult{
				ClientID: cid,
				Status:   CommandStatusPending,
			}

			// 再次检查是否已取消
			select {
			case <-taskCtx.Done():
				result.Status = CommandStatusCancelled
				result.Error = "Task cancelled"
				mu.Lock()
				results[index] = result
				cancelledCount++
				mu.Unlock()
				return
			default:
			}

			// 发送命令
			cmd, err := cm.SendCommand(cid, commandType, payload, timeout)
			if err != nil {
				result.Status = CommandStatusFailed
				result.Error = err.Error()
				cm.logger.Error("Failed to send command to client",
					"task_id", taskID,
					"client_id", cid,
					"error", err,
				)
			} else {
				result.CommandID = cmd.CommandID

				// 记录命令ID
				task.mu.Lock()
				task.CommandIDs = append(task.CommandIDs, cmd.CommandID)
				task.mu.Unlock()

				// 等待命令完成（带取消检查）
				finalCmd := cm.waitForCommandCompletionWithContext(taskCtx, cmd.CommandID, timeout+5*time.Second)
				if finalCmd != nil {
					// 检查是否被取消
					select {
					case <-taskCtx.Done():
						result.Status = CommandStatusCancelled
						result.Error = "Task cancelled"
						// 更新命令状态为取消
						cm.updateCommandStatus(cmd.CommandID, CommandStatusCancelled, nil, "Task cancelled")
					default:
						result.Status = finalCmd.Status
						result.Result = finalCmd.Result
						result.Error = finalCmd.Error

						if finalCmd.Status == CommandStatusCompleted {
							mu.Lock()
							successCount++
							mu.Unlock()
						}
					}
				} else {
					// 检查是否因为取消而返回nil
					select {
					case <-taskCtx.Done():
						result.Status = CommandStatusCancelled
						result.Error = "Task cancelled"
					default:
						result.Status = CommandStatusTimeout
						result.Error = "failed to get command result"
					}
				}
			}

			mu.Lock()
			results[index] = result
			mu.Unlock()
		}(i, clientID)
	}

	// 等待所有命令完成或取消
	wg.Wait()

done:
	failedCount := total - successCount - cancelledCount
	response := &MultiCommandResponse{
		TaskID:         taskID,
		Success:        failedCount == 0 && cancelledCount == 0,
		Total:          total,
		SuccessCount:   successCount,
		FailedCount:    failedCount,
		CancelledCount: cancelledCount,
		Results:        results,
		Status:         task.Status,
	}

	if cancelledCount > 0 {
		response.Message = fmt.Sprintf("Command sent to %d clients: %d succeeded, %d failed, %d cancelled", total, successCount, failedCount, cancelledCount)
		task.mu.Lock()
		task.Status = "cancelled"
		task.mu.Unlock()
		response.Status = "cancelled"
	} else {
		response.Message = fmt.Sprintf("Command sent to %d clients: %d succeeded, %d failed", total, successCount, failedCount)
	}

	cm.logger.Info("Multi-command completed",
		"task_id", taskID,
		"total", total,
		"success", successCount,
		"failed", failedCount,
		"cancelled", cancelledCount,
	)

	return response
}

// waitForCommandCompletion 等待命令完成
func (cm *CommandManager) waitForCommandCompletion(commandID string, timeout time.Duration) *Command {
	return cm.waitForCommandCompletionWithContext(cm.ctx, commandID, timeout)
}

// waitForCommandCompletionWithContext 等待命令完成（支持context取消）
func (cm *CommandManager) waitForCommandCompletionWithContext(ctx context.Context, commandID string, timeout time.Duration) *Command {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cmd, err := cm.GetCommand(commandID)
			if err != nil {
				return nil
			}

			// 检查是否是终态
			switch cmd.Status {
			case CommandStatusCompleted, CommandStatusFailed, CommandStatusTimeout, CommandStatusCancelled:
				return cmd
			}

			// 检查是否超时
			if time.Now().After(deadline) {
				return cmd
			}

		case <-ctx.Done():
			return nil
		}
	}
}

// CancelMultiCommand 取消正在执行的多播任务
func (cm *CommandManager) CancelMultiCommand(taskID string) error {
	cm.tasksMu.RLock()
	task, ok := cm.multiTasks[taskID]
	cm.tasksMu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.mu.Lock()
	if task.Status != "running" {
		task.mu.Unlock()
		return fmt.Errorf("task is not running: %s", task.Status)
	}
	task.Status = "cancelled"
	task.mu.Unlock()

	// 调用取消函数
	task.CancelFunc()

	// 更新所有相关命令的状态为取消
	task.mu.RLock()
	commandIDs := make([]string, len(task.CommandIDs))
	copy(commandIDs, task.CommandIDs)
	task.mu.RUnlock()

	for _, cmdID := range commandIDs {
		cm.mu.RLock()
		cmd, exists := cm.commands[cmdID]
		cm.mu.RUnlock()

		if exists {
			// 只取消未完成状态的命令
			switch cmd.Status {
			case CommandStatusPending, CommandStatusExecuting:
				cm.updateCommandStatus(cmdID, CommandStatusCancelled, nil, "Task cancelled")
			}
		}
	}

	cm.logger.Info("Multi-command task cancelled",
		"task_id", taskID,
		"command_count", len(commandIDs),
	)

	return nil
}

// GetMultiCommandTask 获取多播任务信息
func (cm *CommandManager) GetMultiCommandTask(taskID string) (*MultiCommandTask, error) {
	cm.tasksMu.RLock()
	defer cm.tasksMu.RUnlock()

	task, ok := cm.multiTasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// 返回副本（手动复制字段，避免复制锁）
	task.mu.RLock()
	taskCopy := &MultiCommandTask{
		TaskID:     task.TaskID,
		ClientIDs:  make([]string, len(task.ClientIDs)),
		CommandIDs: make([]string, len(task.CommandIDs)),
		Status:     task.Status,
		CreatedAt:  task.CreatedAt,
	}
	copy(taskCopy.ClientIDs, task.ClientIDs)
	copy(taskCopy.CommandIDs, task.CommandIDs)
	task.mu.RUnlock()

	return taskCopy, nil
}
