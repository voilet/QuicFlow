package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/voilet/quic-flow/pkg/dispatcher"
	"github.com/voilet/quic-flow/pkg/protocol"
	"github.com/voilet/quic-flow/pkg/session"
	"github.com/voilet/quic-flow/pkg/task/models"
	"github.com/voilet/quic-flow/pkg/task/store"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"google.golang.org/protobuf/proto"
)

// TaskDispatcher 任务分发器
type TaskDispatcher struct {
	dispatcher *dispatcher.Dispatcher
	sessionMgr *session.SessionManager
	taskStore  store.TaskStore
	logger     *monitoring.Logger
}

// NewTaskDispatcher 创建任务分发器
func NewTaskDispatcher(
	disp *dispatcher.Dispatcher,
	sessionMgr *session.SessionManager,
	taskStore store.TaskStore,
	logger *monitoring.Logger,
) *TaskDispatcher {
	return &TaskDispatcher{
		dispatcher: disp,
		sessionMgr: sessionMgr,
		taskStore:  taskStore,
		logger:     logger,
	}
}

// Dispatch 分发任务到目标客户端
func (d *TaskDispatcher) Dispatch(ctx context.Context, task *models.Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	// 获取任务关联的分组
	groupIDs, err := d.taskStore.GetGroupIDs(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get task groups: %w", err)
	}

	// 如果没有关联分组，记录警告但不返回错误
	if len(groupIDs) == 0 {
		d.logger.Warn("Task has no associated groups", "task_id", task.ID, "task_name", task.Name)
		return nil
	}

	// 获取分组下的所有在线客户端
	var targetClients []string
	for _, groupID := range groupIDs {
		clients, err := d.getOnlineClientsByGroup(ctx, groupID)
		if err != nil {
			d.logger.Warn("Failed to get clients for group", "group_id", groupID, "error", err)
			continue
		}
		targetClients = append(targetClients, clients...)
	}

	if len(targetClients) == 0 {
		d.logger.Warn("No online clients found for task", "task_id", task.ID, "task_name", task.Name)
		return nil
	}

	// 分发到每个客户端
	var lastErr error
	for _, clientID := range targetClients {
		if err := d.dispatchToClient(ctx, clientID, task); err != nil {
			d.logger.Error("Failed to dispatch task to client",
				"task_id", task.ID,
				"client_id", clientID,
				"error", err)
			lastErr = err
			// 继续分发到其他客户端，不因单个失败而中断
		}
	}

	return lastErr
}

// dispatchToClient 分发任务到单个客户端
func (d *TaskDispatcher) dispatchToClient(ctx context.Context, clientID string, task *models.Task) error {
	// 生成执行ID（使用时间戳+任务ID）
	executionID := fmt.Sprintf("%d-%d", task.ID, time.Now().UnixMilli())

	// 构造任务执行消息
	execMsg := &protocol.TaskExecution{
		ExecutionId:    executionID,
		TaskId:         fmt.Sprintf("%d", task.ID),
		TaskName:       task.Name,
		ExecutorType:   protocol.ExecutorType(task.ExecutorType),
		ExecutorConfig: task.ExecutorConfig,
		Timeout:        int32(task.Timeout),
		RetryCount:     int32(task.RetryCount),
		RetryInterval:  int32(task.RetryInterval),
		ExecutionType:  protocol.ExecutionType_EXECUTION_TYPE_SCHEDULED,
		Timestamp:      time.Now().UnixMilli(),
	}

	// 序列化消息
	payload, err := proto.Marshal(execMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal task execution: %w", err)
	}

	// 构造数据消息
	msg := &protocol.DataMessage{
		MsgId:      executionID,
		SenderId:   "server",
		ReceiverId: clientID,
		Type:       protocol.MessageType_MESSAGE_TYPE_COMMAND,
		Payload:    payload,
		WaitAck:    true,
		Timestamp:  time.Now().UnixMilli(),
	}

	// 通过 dispatcher 发送消息
	if err := d.dispatcher.Dispatch(ctx, msg, nil); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	d.logger.Info("Task dispatched to client",
		"task_id", task.ID,
		"task_name", task.Name,
		"client_id", clientID,
		"execution_id", executionID)

	return nil
}

// getOnlineClientsByGroup 获取分组下的在线客户端ID列表
func (d *TaskDispatcher) getOnlineClientsByGroup(ctx context.Context, groupID int64) ([]string, error) {
	// 获取所有在线客户端
	allOnlineClients := d.sessionMgr.ListClientIDs()

	// 注意：这里简化处理，实际应该查询数据库中的客户端分组关系
	// 由于 session manager 中没有存储分组信息，我们需要通过其他方式获取
	// 这里暂时返回所有在线客户端，实际实现中应该：
	// 1. 查询数据库获取分组下的所有客户端ID（包括离线）
	// 2. 过滤出在线客户端

	// TODO: 实现从数据库查询客户端分组关系的逻辑
	// 目前先返回所有在线客户端作为占位实现
	// 在实际生产环境中，应该查询 models.Client 表，根据 group_id 过滤
	return allOnlineClients, nil
}
