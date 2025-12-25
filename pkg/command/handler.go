package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/voilet/QuicFlow/pkg/monitoring"
	"github.com/voilet/QuicFlow/pkg/protocol"
)

// ClientAPI 客户端接口（用于发送消息）
type ClientAPI interface {
	SendMessage(ctx context.Context, msg *protocol.DataMessage, waitAck bool, timeout time.Duration) (*protocol.AckMessage, error)
}

// CommandHandler 命令处理器（客户端）
type CommandHandler struct {
	client   ClientAPI
	logger   *monitoring.Logger
	executor CommandExecutor // 业务层实现的命令执行器
}

// NewCommandHandler 创建命令处理器
func NewCommandHandler(client ClientAPI, executor CommandExecutor, logger *monitoring.Logger) *CommandHandler {
	return &CommandHandler{
		client:   client,
		logger:   logger,
		executor: executor,
	}
}

// HandleCommand 处理收到的命令消息
// 这个方法应该被注册为 MESSAGE_TYPE_COMMAND 的处理器
func (h *CommandHandler) HandleCommand(ctx context.Context, msg *protocol.DataMessage) (*protocol.DataMessage, error) {
	h.logger.Info("Received command from server",
		"command_id", msg.MsgId,
		"sender", msg.SenderId,
	)

	// 解析命令载荷
	var cmdPayload CommandPayload
	if err := json.Unmarshal(msg.Payload, &cmdPayload); err != nil {
		h.logger.Error("Failed to unmarshal command payload",
			"command_id", msg.MsgId,
			"error", err,
		)
		return h.buildErrorResponse(msg.MsgId, fmt.Sprintf("invalid command payload: %v", err)), nil
	}

	h.logger.Info("Executing command",
		"command_id", msg.MsgId,
		"command_type", cmdPayload.CommandType,
	)

	// 执行命令
	result, err := h.executor.Execute(cmdPayload.CommandType, cmdPayload.Payload)
	if err != nil {
		h.logger.Error("Command execution failed",
			"command_id", msg.MsgId,
			"command_type", cmdPayload.CommandType,
			"error", err,
		)
		return h.buildErrorResponse(msg.MsgId, fmt.Sprintf("command execution failed: %v", err)), nil
	}

	h.logger.Info("Command execution succeeded",
		"command_id", msg.MsgId,
		"command_type", cmdPayload.CommandType,
	)

	// 构造成功响应
	return h.buildSuccessResponse(msg.MsgId, result), nil
}

// buildSuccessResponse 构造成功响应
func (h *CommandHandler) buildSuccessResponse(commandID string, result []byte) *protocol.DataMessage {
	ack := &protocol.AckMessage{
		MsgId:  commandID,
		Status: protocol.AckStatus_ACK_STATUS_SUCCESS,
		Result: result,
	}

	ackBytes, _ := json.Marshal(ack)

	return &protocol.DataMessage{
		MsgId:     commandID,
		Type:      protocol.MessageType_MESSAGE_TYPE_RESPONSE,
		Payload:   ackBytes,
		Timestamp: time.Now().UnixMilli(),
	}
}

// buildErrorResponse 构造错误响应
func (h *CommandHandler) buildErrorResponse(commandID string, errMsg string) *protocol.DataMessage {
	ack := &protocol.AckMessage{
		MsgId:  commandID,
		Status: protocol.AckStatus_ACK_STATUS_FAILURE,
		Error:  errMsg,
	}

	ackBytes, _ := json.Marshal(ack)

	return &protocol.DataMessage{
		MsgId:     commandID,
		Type:      protocol.MessageType_MESSAGE_TYPE_RESPONSE,
		Payload:   ackBytes,
		Timestamp: time.Now().UnixMilli(),
	}
}

// SetExecutor 设置命令执行器（允许动态更换）
func (h *CommandHandler) SetExecutor(executor CommandExecutor) {
	h.executor = executor
}
