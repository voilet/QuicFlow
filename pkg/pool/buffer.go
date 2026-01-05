// Package pool 提供高性能内存池，减少 GC 开销
// 使用 sync.Pool 复用常用对象，降低内存分配频率
package pool

import (
	"bytes"
	"sync"
)

// 各种缓冲区大小的定义
const (
	// SmallBufferSize 小缓冲区大小 (1KB) - 用于小型帧
	SmallBufferSize = 1 * 1024

	// MediumBufferSize 中等缓冲区大小 (8KB) - 用于中型消息
	MediumBufferSize = 8 * 1024

	// LargeBufferSize 大缓冲区大小 (32KB) - 用于大型消息
	LargeBufferSize = 32 * 1024

	// MaxBufferSize 最大缓冲区大小 (256KB) - 用于非常大的消息
	MaxBufferSize = 256 * 1024
)

// BufferPool 缓冲区池
type BufferPool struct {
	small  sync.Pool
	medium sync.Pool
	large  sync.Pool
	max    sync.Pool
}

// 全局缓冲区池实例
var globalBufferPool = &BufferPool{
	small: sync.Pool{
		New: func() interface{} {
			b := make([]byte, SmallBufferSize)
			return &b
		},
	},
	medium: sync.Pool{
		New: func() interface{} {
			b := make([]byte, MediumBufferSize)
			return &b
		},
	},
	large: sync.Pool{
		New: func() interface{} {
			b := make([]byte, LargeBufferSize)
			return &b
		},
	},
	max: sync.Pool{
		New: func() interface{} {
			b := make([]byte, MaxBufferSize)
			return &b
		},
	},
}

// GetBuffer 获取指定大小的缓冲区
// size: 期望的缓冲区大小
// 返回的缓冲区可能大于请求的大小
func GetBuffer(size int) []byte {
	var bPtr *[]byte

	switch {
	case size <= SmallBufferSize:
		bPtr = globalBufferPool.small.Get().(*[]byte)
	case size <= MediumBufferSize:
		bPtr = globalBufferPool.medium.Get().(*[]byte)
	case size <= LargeBufferSize:
		bPtr = globalBufferPool.large.Get().(*[]byte)
	default:
		bPtr = globalBufferPool.max.Get().(*[]byte)
	}

	b := *bPtr
	// 重置缓冲区长度为 0，但保留容量
	b = b[:0]
	return b
}

// PutBuffer 归还缓冲区到池中
func PutBuffer(b []byte) {
	// 检查缓冲区容量，确定归还到哪个池
	capacity := cap(b)

	switch {
	case capacity == SmallBufferSize:
		globalBufferPool.small.Put(&b)
	case capacity == MediumBufferSize:
		globalBufferPool.medium.Put(&b)
	case capacity == LargeBufferSize:
		globalBufferPool.large.Put(&b)
	case capacity == MaxBufferSize:
		globalBufferPool.max.Put(&b)
	// 对于非标准大小的缓冲区，不回收（让 GC 处理）
	}
}

// GetSmallBuffer 获取小缓冲区 (1KB)
func GetSmallBuffer() []byte {
	bPtr := globalBufferPool.small.Get().(*[]byte)
	b := *bPtr
	return b[:0]
}

// GetMediumBuffer 获取中等缓冲区 (8KB)
func GetMediumBuffer() []byte {
	bPtr := globalBufferPool.medium.Get().(*[]byte)
	b := *bPtr
	return b[:0]
}

// GetLargeBuffer 获取大缓冲区 (32KB)
func GetLargeBuffer() []byte {
	bPtr := globalBufferPool.large.Get().(*[]byte)
	b := *bPtr
	return b[:0]
}

// BytesBufferPool bytes.Buffer 对象池
var BytesBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// GetBytesBuffer 获取 bytes.Buffer 对象
func GetBytesBuffer() *bytes.Buffer {
	buf := BytesBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBytesBuffer 归还 bytes.Buffer 对象
func PutBytesBuffer(buf *bytes.Buffer) {
	// 如果缓冲区太大（超过 1MB），不回收，让 GC 处理
	if buf.Cap() > 1024*1024 {
		return
	}
	BytesBufferPool.Put(buf)
}

// Stats 池统计信息（用于监控）
type Stats struct {
	SmallAllocated int64
	MediumAllocated int64
	LargeAllocated  int64
	MaxAllocated    int64
}

// GetStats 获取池统计信息（估算值）
func GetStats() *Stats {
	return &Stats{
		// sync.Pool 不直接暴露统计信息，这里返回配置值
		// 实际监控需要在使用时手动计数
	}
}
