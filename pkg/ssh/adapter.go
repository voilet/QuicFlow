// Package ssh 在 QUIC 流上实现 SSH 协议层
// 架构：UDP -> QUIC (多路复用、加密、拥塞控制) -> SSH (权限控制、Shell 交互)
package ssh

import (
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// StreamConn 将 quic.Stream 适配为 net.Conn 接口
// 这允许在 QUIC 流上运行标准的 SSH 协议
type StreamConn struct {
	stream *quic.Stream
	conn   *quic.Conn
}

// NewStreamConn 创建一个新的 StreamConn 适配器
func NewStreamConn(stream *quic.Stream, conn *quic.Conn) *StreamConn {
	return &StreamConn{
		stream: stream,
		conn:   conn,
	}
}

// Read 从流中读取数据
func (sc *StreamConn) Read(b []byte) (int, error) {
	return sc.stream.Read(b)
}

// Write 向流中写入数据
func (sc *StreamConn) Write(b []byte) (int, error) {
	return sc.stream.Write(b)
}

// Close 关闭流
func (sc *StreamConn) Close() error {
	return sc.stream.Close()
}

// LocalAddr 返回本地地址
// 使用底层 QUIC 连接的本地地址
func (sc *StreamConn) LocalAddr() net.Addr {
	return sc.conn.LocalAddr()
}

// RemoteAddr 返回远程地址
// 使用底层 QUIC 连接的远程地址
func (sc *StreamConn) RemoteAddr() net.Addr {
	return sc.conn.RemoteAddr()
}

// SetDeadline 设置读写超时
func (sc *StreamConn) SetDeadline(t time.Time) error {
	return sc.stream.SetDeadline(t)
}

// SetReadDeadline 设置读取超时
func (sc *StreamConn) SetReadDeadline(t time.Time) error {
	return sc.stream.SetReadDeadline(t)
}

// SetWriteDeadline 设置写入超时
func (sc *StreamConn) SetWriteDeadline(t time.Time) error {
	return sc.stream.SetWriteDeadline(t)
}

// Stream 返回底层的 QUIC 流
func (sc *StreamConn) Stream() *quic.Stream {
	return sc.stream
}

// Connection 返回底层的 QUIC 连接
func (sc *StreamConn) Connection() *quic.Conn {
	return sc.conn
}

// StreamID 返回流 ID
func (sc *StreamConn) StreamID() quic.StreamID {
	return sc.stream.StreamID()
}

// 确保 StreamConn 实现了 net.Conn 接口
var _ net.Conn = (*StreamConn)(nil)
