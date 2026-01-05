// Package pool 提供高性能内存池，减少 GC 开销
package pool

import (
	"sync"

	"github.com/voilet/quic-flow/pkg/protocol"
)

// FramePool 协议帧对象池
// 复用 Frame 对象，减少 GC 压力
var FramePool = sync.Pool{
	New: func() interface{} {
		return &protocol.Frame{
			// 预分配 Payload 切片，避免小消息时的额外分配
			Payload: make([]byte, 0, 512),
		}
	},
}

// GetFrame 获取 Frame 对象
func GetFrame() *protocol.Frame {
	frame := FramePool.Get().(*protocol.Frame)
	// 重置字段
	frame.Type = protocol.FrameType_FRAME_TYPE_UNSPECIFIED
	frame.Payload = frame.Payload[:0]
	frame.Timestamp = 0
	return frame
}

// PutFrame 归还 Frame 对象到池中
func PutFrame(frame *protocol.Frame) {
	if frame == nil {
		return
	}
	// 重置 Payload 切片以避免内存泄漏
	if cap(frame.Payload) > 64*1024 {
		// 如果 Payload 太大（超过 64KB），不回收
		frame.Payload = nil
	} else {
		frame.Payload = frame.Payload[:0]
	}
	FramePool.Put(frame)
}

// DataMessagePool DataMessage 对象池
var DataMessagePool = sync.Pool{
	New: func() interface{} {
		return &protocol.DataMessage{
			Payload: make([]byte, 0, 512),
		}
	},
}

// GetDataMessage 获取 DataMessage 对象
func GetDataMessage() *protocol.DataMessage {
	msg := DataMessagePool.Get().(*protocol.DataMessage)
	// 重置字段
	msg.MsgId = ""
	msg.SenderId = ""
	msg.ReceiverId = ""
	msg.Payload = msg.Payload[:0]
	msg.Timestamp = 0
	msg.WaitAck = false
	return msg
}

// PutDataMessage 归还 DataMessage 对象到池中
func PutDataMessage(msg *protocol.DataMessage) {
	if msg == nil {
		return
	}
	// 重置 Payload
	if cap(msg.Payload) > 64*1024 {
		msg.Payload = nil
	} else {
		msg.Payload = msg.Payload[:0]
	}
	DataMessagePool.Put(msg)
}

// PingFramePool PingFrame 对象池
var PingFramePool = sync.Pool{
	New: func() interface{} {
		return &protocol.PingFrame{}
	},
}

// GetPingFrame 获取 PingFrame 对象
func GetPingFrame() *protocol.PingFrame {
	return PingFramePool.Get().(*protocol.PingFrame)
}

// PutPingFrame 归还 PingFrame 对象到池中
func PutPingFrame(frame *protocol.PingFrame) {
	if frame != nil {
		frame.ClientId = ""
		PingFramePool.Put(frame)
	}
}

// PongFramePool PongFrame 对象池
var PongFramePool = sync.Pool{
	New: func() interface{} {
		return &protocol.PongFrame{}
	},
}

// GetPongFrame 获取 PongFrame 对象
func GetPongFrame() *protocol.PongFrame {
	return PongFramePool.Get().(*protocol.PongFrame)
}

// PutPongFrame 归还 PongFrame 对象到池中
func PutPongFrame(frame *protocol.PongFrame) {
	if frame != nil {
		frame.ServerTime = 0
		PongFramePool.Put(frame)
	}
}

// AckMessagePool AckMessage 对象池
var AckMessagePool = sync.Pool{
	New: func() interface{} {
		return &protocol.AckMessage{
			Result: make([]byte, 0, 256),
		}
	},
}

// GetAckMessage 获取 AckMessage 对象
func GetAckMessage() *protocol.AckMessage {
	msg := AckMessagePool.Get().(*protocol.AckMessage)
	msg.MsgId = ""
	msg.Status = protocol.AckStatus_ACK_STATUS_UNSPECIFIED
	msg.Error = ""
	msg.Result = msg.Result[:0]
	return msg
}

// PutAckMessage 归还 AckMessage 对象到池中
func PutAckMessage(msg *protocol.AckMessage) {
	if msg != nil {
		if cap(msg.Result) > 4096 {
			msg.Result = nil
		} else {
			msg.Result = msg.Result[:0]
		}
		AckMessagePool.Put(msg)
	}
}
