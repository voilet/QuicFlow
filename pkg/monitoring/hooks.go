package monitoring

// EventHooks 定义系统事件回调钩子
// 所有钩子函数都是可选的，可以为 nil
type EventHooks struct {
	// OnConnect 在客户端连接成功时调用
	// clientID: 客户端唯一标识
	OnConnect func(clientID string)

	// OnDisconnect 在客户端断开连接时调用
	// clientID: 客户端唯一标识
	// reason: 断开原因（错误或 nil 表示正常断开）
	OnDisconnect func(clientID string, reason error)

	// OnHeartbeatTimeout 在客户端心跳超时时调用（即将被清理）
	// clientID: 客户端唯一标识
	OnHeartbeatTimeout func(clientID string)

	// OnReconnect 在客户端重连成功时调用
	// clientID: 客户端唯一标识
	// attemptCount: 重连尝试次数（从 1 开始计数）
	OnReconnect func(clientID string, attemptCount int)

	// OnMessageSent 在消息发送后调用（成功或失败都会调用）
	// msgID: 消息唯一标识
	// clientID: 目标客户端 ID（广播时为空）
	// err: 发送错误（nil 表示成功）
	OnMessageSent func(msgID string, clientID string, err error)

	// OnMessageReceived 在接收到消息时调用
	// msgID: 消息唯一标识
	// clientID: 发送方客户端 ID
	OnMessageReceived func(msgID string, clientID string)

	// OnBroadcast 在广播消息时调用
	// msgID: 消息唯一标识
	// targetCount: 目标客户端数量
	// successCount: 成功发送的客户端数量
	OnBroadcast func(msgID string, targetCount int, successCount int)

	// OnPromiseTimeout 在 Promise 超时时调用
	// msgID: 消息唯一标识
	OnPromiseTimeout func(msgID string)

	// OnError 在发生错误时调用（通用错误处理）
	// err: 错误对象
	// context: 错误上下文描述
	OnError func(err error, context string)
}

// SafeOnConnect 安全地调用 OnConnect 钩子（防止 panic）
func (h *EventHooks) SafeOnConnect(clientID string) {
	if h == nil || h.OnConnect == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
			// 这里可以记录日志
		}
	}()

	h.OnConnect(clientID)
}

// SafeOnDisconnect 安全地调用 OnDisconnect 钩子
func (h *EventHooks) SafeOnDisconnect(clientID string, reason error) {
	if h == nil || h.OnDisconnect == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnDisconnect(clientID, reason)
}

// SafeOnHeartbeatTimeout 安全地调用 OnHeartbeatTimeout 钩子
func (h *EventHooks) SafeOnHeartbeatTimeout(clientID string) {
	if h == nil || h.OnHeartbeatTimeout == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnHeartbeatTimeout(clientID)
}

// SafeOnReconnect 安全地调用 OnReconnect 钩子
func (h *EventHooks) SafeOnReconnect(clientID string, attemptCount int) {
	if h == nil || h.OnReconnect == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnReconnect(clientID, attemptCount)
}

// SafeOnMessageSent 安全地调用 OnMessageSent 钩子
func (h *EventHooks) SafeOnMessageSent(msgID string, clientID string, err error) {
	if h == nil || h.OnMessageSent == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnMessageSent(msgID, clientID, err)
}

// SafeOnMessageReceived 安全地调用 OnMessageReceived 钩子
func (h *EventHooks) SafeOnMessageReceived(msgID string, clientID string) {
	if h == nil || h.OnMessageReceived == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnMessageReceived(msgID, clientID)
}

// SafeOnBroadcast 安全地调用 OnBroadcast 钩子
func (h *EventHooks) SafeOnBroadcast(msgID string, targetCount int, successCount int) {
	if h == nil || h.OnBroadcast == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnBroadcast(msgID, targetCount, successCount)
}

// SafeOnPromiseTimeout 安全地调用 OnPromiseTimeout 钩子
func (h *EventHooks) SafeOnPromiseTimeout(msgID string) {
	if h == nil || h.OnPromiseTimeout == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnPromiseTimeout(msgID)
}

// SafeOnError 安全地调用 OnError 钩子
func (h *EventHooks) SafeOnError(err error, context string) {
	if h == nil || h.OnError == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			// 钩子函数 panic 不应影响主流程
		}
	}()

	h.OnError(err, context)
}
