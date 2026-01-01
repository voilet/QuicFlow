package ssh

import (
	"encoding/binary"
	"fmt"
	"io"
)

// StreamType 流类型标识
type StreamType uint8

const (
	// StreamTypeData 普通业务数据流
	StreamTypeData StreamType = 0x00

	// StreamTypeSSH SSH 反向隧道流
	StreamTypeSSH StreamType = 0x01

	// StreamTypeFileTransfer 文件传输流
	StreamTypeFileTransfer StreamType = 0x02

	// StreamTypePortForward 端口转发流
	StreamTypePortForward StreamType = 0x03
)

// 魔数和版本
const (
	// MagicNumber 协议魔数，用于识别 SSH-over-QUIC 流
	// 选择一个不会与普通数据冲突的值
	MagicNumber uint32 = 0x51534853 // "QSSH" in ASCII (QUIC SSH)

	// ProtocolVersion 协议版本
	ProtocolVersion uint8 = 1

	// HeaderSize 握手头部大小：魔数(4) + 版本(1) + 类型(1) = 6 字节
	HeaderSize = 6
)

// StreamHeader 流握手头部
// 在 AcceptStream 后发送，用于识别流类型
type StreamHeader struct {
	Magic   uint32     // 魔数
	Version uint8      // 协议版本
	Type    StreamType // 流类型
}

// String 返回流类型的字符串表示
func (t StreamType) String() string {
	switch t {
	case StreamTypeData:
		return "Data"
	case StreamTypeSSH:
		return "SSH"
	case StreamTypeFileTransfer:
		return "FileTransfer"
	case StreamTypePortForward:
		return "PortForward"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

// WriteHeader 写入流握手头部
func WriteHeader(w io.Writer, streamType StreamType) error {
	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint32(header[0:4], MagicNumber)
	header[4] = ProtocolVersion
	header[5] = byte(streamType)

	_, err := w.Write(header)
	return err
}

// ReadHeader 读取流握手头部
// 返回头部和错误，如果魔数不匹配则返回 ErrInvalidMagic
func ReadHeader(r io.Reader) (*StreamHeader, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	magic := binary.BigEndian.Uint32(header[0:4])
	if magic != MagicNumber {
		return nil, ErrInvalidMagic
	}

	version := header[4]
	if version != ProtocolVersion {
		return nil, fmt.Errorf("%w: got %d, want %d", ErrUnsupportedVersion, version, ProtocolVersion)
	}

	return &StreamHeader{
		Magic:   magic,
		Version: version,
		Type:    StreamType(header[5]),
	}, nil
}

// TryReadHeader 尝试读取流握手头部
// 如果前几个字节不是魔数，则返回已读取的字节和 nil 头部
// 这用于区分普通数据流和 SSH 流
func TryReadHeader(r io.Reader) (*StreamHeader, []byte, error) {
	peek := make([]byte, HeaderSize)
	n, err := io.ReadFull(r, peek)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			// 读取的数据不足，返回已读取的部分
			return nil, peek[:n], nil
		}
		return nil, nil, err
	}

	magic := binary.BigEndian.Uint32(peek[0:4])
	if magic != MagicNumber {
		// 不是 SSH 流，返回已读取的数据
		return nil, peek, nil
	}

	version := peek[4]
	if version != ProtocolVersion {
		return nil, nil, fmt.Errorf("%w: got %d, want %d", ErrUnsupportedVersion, version, ProtocolVersion)
	}

	return &StreamHeader{
		Magic:   magic,
		Version: version,
		Type:    StreamType(peek[5]),
	}, nil, nil
}

// PortForwardRequest 端口转发请求
type PortForwardRequest struct {
	Host string
	Port uint16
}

// WritePortForwardRequest 写入端口转发请求
func WritePortForwardRequest(w io.Writer, req *PortForwardRequest) error {
	hostBytes := []byte(req.Host)
	if len(hostBytes) > 255 {
		return fmt.Errorf("host name too long: %d > 255", len(hostBytes))
	}

	// 格式：hostLen(1) + host(n) + port(2)
	buf := make([]byte, 1+len(hostBytes)+2)
	buf[0] = byte(len(hostBytes))
	copy(buf[1:], hostBytes)
	binary.BigEndian.PutUint16(buf[1+len(hostBytes):], req.Port)

	_, err := w.Write(buf)
	return err
}

// ReadPortForwardRequest 读取端口转发请求
func ReadPortForwardRequest(r io.Reader) (*PortForwardRequest, error) {
	// 读取主机名长度
	lenBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, fmt.Errorf("failed to read host length: %w", err)
	}

	hostLen := int(lenBuf[0])
	if hostLen == 0 {
		return nil, fmt.Errorf("host name cannot be empty")
	}

	// 读取主机名和端口
	buf := make([]byte, hostLen+2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("failed to read host and port: %w", err)
	}

	return &PortForwardRequest{
		Host: string(buf[:hostLen]),
		Port: binary.BigEndian.Uint16(buf[hostLen:]),
	}, nil
}
