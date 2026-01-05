package codec

import (
	"encoding/binary"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/voilet/quic-flow/pkg/pool"
	"github.com/voilet/quic-flow/pkg/protocol"
	pkgerrors "github.com/voilet/quic-flow/pkg/errors"
)

// OptimizedCodec 使用对象池优化的编解码器
// 减少 GC 开销，提升性能
type OptimizedCodec struct {
	*ProtobufCodec
}

// NewOptimizedCodec 创建优化的编解码器
func NewOptimizedCodec() *OptimizedCodec {
	return &OptimizedCodec{
		ProtobufCodec: NewProtobufCodec(),
	}
}

// ReadFrameOptimized 从 Reader 读取并解码 Frame（使用对象池）
// 这个函数会复用 Frame 对象，减少内存分配
func (c *OptimizedCodec) ReadFrameOptimized(r io.Reader) (*protocol.Frame, error) {
	// 使用缓冲区池获取 4 字节长度缓冲区
	lengthBuf := pool.GetSmallBuffer()
	defer pool.PutBuffer(lengthBuf)

	if _, err := io.ReadFull(r, lengthBuf[:4]); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("%w: unexpected EOF reading length prefix", pkgerrors.ErrDecodeFailed)
		}
		return nil, fmt.Errorf("%w: failed to read length prefix: %v", pkgerrors.ErrDecodeFailed, err)
	}

	// 解析长度
	length := binary.BigEndian.Uint32(lengthBuf[:4])
	if length == 0 {
		return nil, fmt.Errorf("%w: zero length frame", pkgerrors.ErrDecodeFailed)
	}
	if length > 10*1024*1024 { // 限制最大 10MB
		return nil, fmt.Errorf("%w: frame too large: %d bytes", pkgerrors.ErrDecodeFailed, length)
	}

	// 使用对象池获取数据缓冲区
	data := pool.GetBuffer(int(length))
	defer pool.PutBuffer(data)

	// 读取帧数据
	if _, err := io.ReadFull(r, data[:length]); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("%w: unexpected EOF reading frame data", pkgerrors.ErrDecodeFailed)
		}
		return nil, fmt.Errorf("%w: failed to read frame data: %v", pkgerrors.ErrDecodeFailed, err)
	}

	return c.DecodeFrame(data[:length])
}

// WriteFrameOptimized 将 Frame 写入到 Writer（使用对象池优化的长度缓冲区）
func (c *OptimizedCodec) WriteFrameOptimized(w io.Writer, frame *protocol.Frame) error {
	data, err := c.EncodeFrame(frame)
	if err != nil {
		return err
	}

	// 使用对象池获取长度缓冲区
	lengthBuf := pool.GetSmallBuffer()
	defer pool.PutBuffer(lengthBuf)

	// 写入 4 字节长度前缀（大端序）
	binary.BigEndian.PutUint32(lengthBuf[:4], uint32(len(data)))

	if _, err := w.Write(lengthBuf[:4]); err != nil {
		return fmt.Errorf("%w: failed to write length prefix: %v", pkgerrors.ErrEncodeFailed, err)
	}

	// 写入帧数据
	n, err := w.Write(data)
	if err != nil {
		return fmt.Errorf("%w: %v", pkgerrors.ErrEncodeFailed, err)
	}

	if n != len(data) {
		return fmt.Errorf("%w: partial write", pkgerrors.ErrEncodeFailed)
	}

	return nil
}

// EncodeDataMessageOptimized 编码 DataMessage 到 Frame（使用对象池）
func EncodeDataMessageOptimized(msg *protocol.DataMessage) (*protocol.Frame, error) {
	if msg == nil {
		return nil, fmt.Errorf("%w: message is nil", pkgerrors.ErrInvalidMessage)
	}

	// 使用 protobuf marshal
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrEncodeFailed, err)
	}

	// 从对象池获取 Frame
	frame := pool.GetFrame()
	frame.Type = protocol.FrameType_FRAME_TYPE_DATA
	frame.Payload = payload
	frame.Timestamp = msg.Timestamp

	return frame, nil
}

// DecodeDataMessageOptimized 解码 DataMessage（使用对象池）
func DecodeDataMessageOptimized(frame *protocol.Frame) (*protocol.DataMessage, error) {
	if frame == nil {
		return nil, fmt.Errorf("%w: frame is nil", pkgerrors.ErrInvalidMessage)
	}

	if frame.Type != protocol.FrameType_FRAME_TYPE_DATA {
		return nil, fmt.Errorf("%w: expected DATA frame, got %v", pkgerrors.ErrInvalidFrameType, frame.Type)
	}

	// 从对象池获取 DataMessage
	msg := pool.GetDataMessage()
	if err := proto.Unmarshal(frame.Payload, msg); err != nil {
		pool.PutDataMessage(msg)
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrDecodeFailed, err)
	}

	return msg, nil
}

// EncodePingFrameOptimized 编码 PingFrame 到 Frame（使用对象池）
func EncodePingFrameOptimized(ping *protocol.PingFrame, timestamp int64) (*protocol.Frame, error) {
	if ping == nil {
		return nil, fmt.Errorf("%w: ping is nil", pkgerrors.ErrInvalidMessage)
	}

	payload, err := proto.Marshal(ping)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrEncodeFailed, err)
	}

	frame := pool.GetFrame()
	frame.Type = protocol.FrameType_FRAME_TYPE_PING
	frame.Payload = payload
	frame.Timestamp = timestamp

	return frame, nil
}

// DecodePingFrameOptimized 解码 PingFrame（使用对象池）
func DecodePingFrameOptimized(frame *protocol.Frame) (*protocol.PingFrame, error) {
	if frame == nil {
		return nil, fmt.Errorf("%w: frame is nil", pkgerrors.ErrInvalidMessage)
	}

	if frame.Type != protocol.FrameType_FRAME_TYPE_PING {
		return nil, fmt.Errorf("%w: expected PING frame, got %v", pkgerrors.ErrInvalidFrameType, frame.Type)
	}

	ping := pool.GetPingFrame()
	if err := proto.Unmarshal(frame.Payload, ping); err != nil {
		pool.PutPingFrame(ping)
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrDecodeFailed, err)
	}

	return ping, nil
}

// EncodePongFrameOptimized 编码 PongFrame 到 Frame（使用对象池）
func EncodePongFrameOptimized(pong *protocol.PongFrame, timestamp int64) (*protocol.Frame, error) {
	if pong == nil {
		return nil, fmt.Errorf("%w: pong is nil", pkgerrors.ErrInvalidMessage)
	}

	payload, err := proto.Marshal(pong)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrEncodeFailed, err)
	}

	frame := pool.GetFrame()
	frame.Type = protocol.FrameType_FRAME_TYPE_PONG
	frame.Payload = payload
	frame.Timestamp = timestamp

	return frame, nil
}

// DecodePongFrameOptimized 解码 PongFrame（使用对象池）
func DecodePongFrameOptimized(frame *protocol.Frame) (*protocol.PongFrame, error) {
	if frame == nil {
		return nil, fmt.Errorf("%w: frame is nil", pkgerrors.ErrInvalidMessage)
	}

	if frame.Type != protocol.FrameType_FRAME_TYPE_PONG {
		return nil, fmt.Errorf("%w: expected PONG frame, got %v", pkgerrors.ErrInvalidFrameType, frame.Type)
	}

	pong := pool.GetPongFrame()
	if err := proto.Unmarshal(frame.Payload, pong); err != nil {
		pool.PutPongFrame(pong)
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrDecodeFailed, err)
	}

	return pong, nil
}

// EncodeAckMessageOptimized 编码 AckMessage 到 Frame（使用对象池）
func EncodeAckMessageOptimized(ack *protocol.AckMessage, timestamp int64) (*protocol.Frame, error) {
	if ack == nil {
		return nil, fmt.Errorf("%w: ack is nil", pkgerrors.ErrInvalidMessage)
	}

	payload, err := proto.Marshal(ack)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrEncodeFailed, err)
	}

	frame := pool.GetFrame()
	frame.Type = protocol.FrameType_FRAME_TYPE_ACK
	frame.Payload = payload
	frame.Timestamp = timestamp

	return frame, nil
}

// DecodeAckMessageOptimized 解码 AckMessage（使用对象池）
func DecodeAckMessageOptimized(frame *protocol.Frame) (*protocol.AckMessage, error) {
	if frame == nil {
		return nil, fmt.Errorf("%w: frame is nil", pkgerrors.ErrInvalidMessage)
	}

	if frame.Type != protocol.FrameType_FRAME_TYPE_ACK {
		return nil, fmt.Errorf("%w: expected ACK frame, got %v", pkgerrors.ErrInvalidFrameType, frame.Type)
	}

	ack := pool.GetAckMessage()
	if err := proto.Unmarshal(frame.Payload, ack); err != nil {
		pool.PutAckMessage(ack)
		return nil, fmt.Errorf("%w: %v", pkgerrors.ErrDecodeFailed, err)
	}

	return ack, nil
}

// PutFrame 归还 Frame 到对象池
// 用于使用完 Frame 后进行回收
func PutFrame(frame *protocol.Frame) {
	pool.PutFrame(frame)
}

// PutDataMessage 归还 DataMessage 到对象池
func PutDataMessage(msg *protocol.DataMessage) {
	pool.PutDataMessage(msg)
}

// PutPingFrame 归还 PingFrame 到对象池
func PutPingFrame(ping *protocol.PingFrame) {
	pool.PutPingFrame(ping)
}

// PutPongFrame 归还 PongFrame 到对象池
func PutPongFrame(pong *protocol.PongFrame) {
	pool.PutPongFrame(pong)
}

// PutAckMessage 归还 AckMessage 到对象池
func PutAckMessage(ack *protocol.AckMessage) {
	pool.PutAckMessage(ack)
}
