package filetransfer

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	quic "github.com/quic-go/quic-go"
)

// 帧类型
const (
	FrameTypeInit       = 0x01
	FrameTypeData       = 0x02
	FrameTypeAck        = 0x03
	FrameTypeComplete   = 0x04
	FrameTypeError      = 0x05
	FrameTypeResume     = 0x06
	FrameTypeProgress   = 0x07
	FrameTypeCancel     = 0x08
)

// Frame 帧接口
type Frame interface {
	Type() uint8
	Encode() ([]byte, error)
}

// BaseFrame 基础帧
type BaseFrame struct {
	frameType uint8
}

func (bf *BaseFrame) Type() uint8 {
	return bf.frameType
}

// InitFrame 初始化帧
type InitFrame struct {
	BaseFrame
	TaskID    string
	FileName  string
	FileSize  int64
	Checksum  string
	ChunkSize uint32
	Options   TransferOptions
}

func NewInitFrame(taskID, fileName string, fileSize int64, checksum string, chunkSize uint32) *InitFrame {
	return &InitFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeInit},
		TaskID:    taskID,
		FileName:  fileName,
		FileSize:  fileSize,
		Checksum:  checksum,
		ChunkSize: chunkSize,
	}
}

func (f *InitFrame) Encode() ([]byte, error) {
	// 格式: [类型(1)] [任务ID长度(2)] [任务ID] [文件名长度(2)] [文件名] [文件大小(8)] [校验和长度(2)] [校验和] [块大小(4)] [选项长度(2)] [选项JSON]
	optsBytes, _ := json.Marshal(f.Options)

	taskIDBytes := []byte(f.TaskID)
	fileNameBytes := []byte(f.FileName)
	checksumBytes := []byte(f.Checksum)

	totalLen := 1 + 2 + len(taskIDBytes) + 2 + len(fileNameBytes) + 8 + 2 + len(checksumBytes) + 4 + 2 + len(optsBytes)
	buf := make([]byte, 0, totalLen)

	buf = append(buf, f.Type())
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(taskIDBytes)))
	buf = append(buf, taskIDBytes...)
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(fileNameBytes)))
	buf = append(buf, fileNameBytes...)
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.FileSize))
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(checksumBytes)))
	buf = append(buf, checksumBytes...)
	binary.BigEndian.PutUint32(buf[len(buf):len(buf)+4], f.ChunkSize)
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(optsBytes)))
	buf = append(buf, optsBytes...)

	return buf, nil
}

// DecodeInitFrame 解码初始化帧
func DecodeInitFrame(data []byte) (*InitFrame, error) {
	if len(data) < 1 || data[0] != FrameTypeInit {
		return nil, fmt.Errorf("invalid init frame")
	}

	offset := 1

	// 任务ID
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid task ID length")
	}
	taskIDLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(taskIDLen) {
		return nil, fmt.Errorf("invalid task ID")
	}
	taskID := string(data[offset : offset+int(taskIDLen)])
	offset += int(taskIDLen)

	// 文件名
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid filename length")
	}
	fileNameLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(fileNameLen) {
		return nil, fmt.Errorf("invalid filename")
	}
	fileName := string(data[offset : offset+int(fileNameLen)])
	offset += int(fileNameLen)

	// 文件大小
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid file size")
	}
	fileSize := int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// 校验和
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid checksum length")
	}
	checksumLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(checksumLen) {
		return nil, fmt.Errorf("invalid checksum")
	}
	checksum := string(data[offset : offset+int(checksumLen)])
	offset += int(checksumLen)

	// 块大小
	if len(data) < offset+4 {
		return nil, fmt.Errorf("invalid chunk size")
	}
	chunkSize := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	// 选项
	var opts TransferOptions
	if len(data) > offset+2 {
		optsLen := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		if len(data) >= offset+int(optsLen) {
			json.Unmarshal(data[offset:offset+int(optsLen)], &opts)
		}
	}

	return &InitFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeInit},
		TaskID:    taskID,
		FileName:  fileName,
		FileSize:  fileSize,
		Checksum:  checksum,
		ChunkSize: chunkSize,
		Options:   opts,
	}, nil
}

// DataFrame 数据帧
type DataFrame struct {
	BaseFrame
	TaskID   string
	Sequence uint64
	Offset   int64
	Data     []byte
}

func NewDataFrame(taskID string, sequence uint64, offset int64, data []byte) *DataFrame {
	return &DataFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeData},
		TaskID:    taskID,
		Sequence:  sequence,
		Offset:    offset,
		Data:      data,
	}
}

func (f *DataFrame) Encode() ([]byte, error) {
	// 格式: [类型(1)] [任务ID长度(2)] [任务ID] [序列号(8)] [偏移量(8)] [数据长度(4)] [数据]
	taskIDBytes := []byte(f.TaskID)
	dataLen := len(f.Data)

	totalLen := 1 + 2 + len(taskIDBytes) + 8 + 8 + 4 + dataLen
	buf := make([]byte, 0, totalLen)

	buf = append(buf, f.Type())
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(taskIDBytes)))
	buf = append(buf, taskIDBytes...)
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], f.Sequence)
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.Offset))
	binary.BigEndian.PutUint32(buf[len(buf):len(buf)+4], uint32(dataLen))
	buf = append(buf, f.Data...)

	return buf, nil
}

// AckFrame 确认帧
type AckFrame struct {
	BaseFrame
	TaskID    string
	Sequence  uint64
	Offset    int64
	Received  int64
}

func NewAckFrame(taskID string, sequence uint64, offset, received int64) *AckFrame {
	return &AckFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeAck},
		TaskID:    taskID,
		Sequence:  sequence,
		Offset:    offset,
		Received:  received,
	}
}

func (f *AckFrame) Encode() ([]byte, error) {
	taskIDBytes := []byte(f.TaskID)
	totalLen := 1 + 2 + len(taskIDBytes) + 8 + 8 + 8
	buf := make([]byte, 0, totalLen)

	buf = append(buf, f.Type())
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(taskIDBytes)))
	buf = append(buf, taskIDBytes...)
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], f.Sequence)
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.Offset))
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.Received))

	return buf, nil
}

// ProgressFrame 进度帧
type ProgressFrame struct {
	BaseFrame
	TaskID      string
	Progress    float64
	Transferred int64
	Total       int64
	Speed       int64
}

func NewProgressFrame(taskID string, progress float64, transferred, total, speed int64) *ProgressFrame {
	return &ProgressFrame{
		BaseFrame:   BaseFrame{frameType: FrameTypeProgress},
		TaskID:      taskID,
		Progress:    progress,
		Transferred: transferred,
		Total:       total,
		Speed:       speed,
	}
}

func (f *ProgressFrame) Encode() ([]byte, error) {
	taskIDBytes := []byte(f.TaskID)
	totalLen := 1 + 2 + len(taskIDBytes) + 8 + 8 + 8 + 8
	buf := make([]byte, 0, totalLen)

	buf = append(buf, f.Type())
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(taskIDBytes)))
	buf = append(buf, taskIDBytes...)
	// 进度百分比 (0-10000 表示 0.00-100.00)
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.Progress*100))
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.Transferred))
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.Total))
	binary.BigEndian.PutUint64(buf[len(buf):len(buf)+8], uint64(f.Speed))

	return buf, nil
}

// CompleteFrame 完成帧
type CompleteFrame struct {
	BaseFrame
	TaskID   string
	Checksum string
	Status   string
}

func NewCompleteFrame(taskID, checksum, status string) *CompleteFrame {
	return &CompleteFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeComplete},
		TaskID:    taskID,
		Checksum:  checksum,
		Status:    status,
	}
}

func (f *CompleteFrame) Encode() ([]byte, error) {
	taskIDBytes := []byte(f.TaskID)
	checksumBytes := []byte(f.Checksum)
	statusBytes := []byte(f.Status)

	totalLen := 1 + 2 + len(taskIDBytes) + 2 + len(checksumBytes) + 2 + len(statusBytes)
	buf := make([]byte, 0, totalLen)

	buf = append(buf, f.Type())
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(taskIDBytes)))
	buf = append(buf, taskIDBytes...)
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(checksumBytes)))
	buf = append(buf, checksumBytes...)
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(statusBytes)))
	buf = append(buf, statusBytes...)

	return buf, nil
}

// ErrorFrame 错误帧
type ErrorFrame struct {
	BaseFrame
	TaskID string
	Code   int
	Message string
}

func NewErrorFrame(taskID string, code int, message string) *ErrorFrame {
	return &ErrorFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeError},
		TaskID:    taskID,
		Code:      code,
		Message:   message,
	}
}

func (f *ErrorFrame) Encode() ([]byte, error) {
	taskIDBytes := []byte(f.TaskID)
	messageBytes := []byte(f.Message)

	totalLen := 1 + 2 + len(taskIDBytes) + 4 + 2 + len(messageBytes)
	buf := make([]byte, 0, totalLen)

	buf = append(buf, f.Type())
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(taskIDBytes)))
	buf = append(buf, taskIDBytes...)
	binary.BigEndian.PutUint32(buf[len(buf):len(buf)+4], uint32(f.Code))
	binary.BigEndian.PutUint16(buf[len(buf):len(buf)+2], uint16(len(messageBytes)))
	buf = append(buf, messageBytes...)

	return buf, nil
}

// FrameDecoder 帧解码器
type FrameDecoder struct {
	mu sync.Mutex
}

func NewFrameDecoder() *FrameDecoder {
	return &FrameDecoder{}
}

func (fd *FrameDecoder) Decode(data []byte) (Frame, error) {
	fd.mu.Lock()
	defer fd.mu.Unlock()

	if len(data) < 1 {
		return nil, fmt.Errorf("empty frame")
	}

	frameType := data[0]

	switch frameType {
	case FrameTypeInit:
		return DecodeInitFrame(data)
	case FrameTypeData:
		return fd.decodeDataFrame(data)
	case FrameTypeAck:
		return fd.decodeAckFrame(data)
	case FrameTypeComplete:
		return fd.decodeCompleteFrame(data)
	case FrameTypeError:
		return fd.decodeErrorFrame(data)
	case FrameTypeProgress:
		return fd.decodeProgressFrame(data)
	default:
		return nil, fmt.Errorf("unknown frame type: %d", frameType)
	}
}

func (fd *FrameDecoder) decodeDataFrame(data []byte) (*DataFrame, error) {
	offset := 1

	// 任务ID
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid data frame")
	}
	taskIDLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(taskIDLen) {
		return nil, fmt.Errorf("invalid task ID in data frame")
	}
	taskID := string(data[offset : offset+int(taskIDLen)])
	offset += int(taskIDLen)

	// 序列号
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid sequence in data frame")
	}
	sequence := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	// 偏移量
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid offset in data frame")
	}
	offsetVal := int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// 数据长度
	if len(data) < offset+4 {
		return nil, fmt.Errorf("invalid data length in data frame")
	}
	dataLen := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	// 数据
	if len(data) < offset+int(dataLen) {
		return nil, fmt.Errorf("invalid data in data frame")
	}
	frameData := make([]byte, dataLen)
	copy(frameData, data[offset:offset+int(dataLen)])

	return &DataFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeData},
		TaskID:    taskID,
		Sequence:  sequence,
		Offset:    offsetVal,
		Data:      frameData,
	}, nil
}

func (fd *FrameDecoder) decodeAckFrame(data []byte) (*AckFrame, error) {
	offset := 1

	// 任务ID
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid ack frame")
	}
	taskIDLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(taskIDLen) {
		return nil, fmt.Errorf("invalid task ID in ack frame")
	}
	taskID := string(data[offset : offset+int(taskIDLen)])
	offset += int(taskIDLen)

	// 序列号
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid sequence in ack frame")
	}
	sequence := binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	// 偏移量
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid offset in ack frame")
	}
	offsetVal := int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// 接收字节数
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid received in ack frame")
	}
	received := int64(binary.BigEndian.Uint64(data[offset : offset+8]))

	return &AckFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeAck},
		TaskID:    taskID,
		Sequence:  sequence,
		Offset:    offsetVal,
		Received:  received,
	}, nil
}

func (fd *FrameDecoder) decodeCompleteFrame(data []byte) (*CompleteFrame, error) {
	offset := 1

	// 任务ID
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid complete frame")
	}
	taskIDLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(taskIDLen) {
		return nil, fmt.Errorf("invalid task ID in complete frame")
	}
	taskID := string(data[offset : offset+int(taskIDLen)])
	offset += int(taskIDLen)

	// 校验和
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid checksum length in complete frame")
	}
	checksumLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(checksumLen) {
		return nil, fmt.Errorf("invalid checksum in complete frame")
	}
	checksum := string(data[offset : offset+int(checksumLen)])
	offset += int(checksumLen)

	// 状态
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid status length in complete frame")
	}
	statusLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(statusLen) {
		return nil, fmt.Errorf("invalid status in complete frame")
	}
	status := string(data[offset : offset+int(statusLen)])

	return &CompleteFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeComplete},
		TaskID:    taskID,
		Checksum:  checksum,
		Status:    status,
	}, nil
}

func (fd *FrameDecoder) decodeErrorFrame(data []byte) (*ErrorFrame, error) {
	offset := 1

	// 任务ID
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid error frame")
	}
	taskIDLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(taskIDLen) {
		return nil, fmt.Errorf("invalid task ID in error frame")
	}
	taskID := string(data[offset : offset+int(taskIDLen)])
	offset += int(taskIDLen)

	// 错误码
	if len(data) < offset+4 {
		return nil, fmt.Errorf("invalid code in error frame")
	}
	code := int(binary.BigEndian.Uint32(data[offset : offset+4]))
	offset += 4

	// 错误消息
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid message length in error frame")
	}
	msgLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(msgLen) {
		return nil, fmt.Errorf("invalid message in error frame")
	}
	message := string(data[offset : offset+int(msgLen)])

	return &ErrorFrame{
		BaseFrame: BaseFrame{frameType: FrameTypeError},
		TaskID:    taskID,
		Code:      code,
		Message:   message,
	}, nil
}

func (fd *FrameDecoder) decodeProgressFrame(data []byte) (*ProgressFrame, error) {
	offset := 1

	// 任务ID
	if len(data) < offset+2 {
		return nil, fmt.Errorf("invalid progress frame")
	}
	taskIDLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2
	if len(data) < offset+int(taskIDLen) {
		return nil, fmt.Errorf("invalid task ID in progress frame")
	}
	taskID := string(data[offset : offset+int(taskIDLen)])
	offset += int(taskIDLen)

	// 进度
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid progress value")
	}
	progressVal := binary.BigEndian.Uint64(data[offset : offset+8])
	progress := float64(progressVal) / 100
	offset += 8

	// 已传输
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid transferred in progress frame")
	}
	transferred := int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// 总大小
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid total in progress frame")
	}
	total := int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// 速度
	if len(data) < offset+8 {
		return nil, fmt.Errorf("invalid speed in progress frame")
	}
	speed := int64(binary.BigEndian.Uint64(data[offset : offset+8]))

	return &ProgressFrame{
		BaseFrame:   BaseFrame{frameType: FrameTypeProgress},
		TaskID:      taskID,
		Progress:    progress,
		Transferred: transferred,
		Total:       total,
		Speed:       speed,
	}, nil
}

// FrameWriter 帧写入器
type FrameWriter struct {
	writer io.Writer
	mu     sync.Mutex
}

func NewFrameWriter(w io.Writer) *FrameWriter {
	return &FrameWriter{writer: w}
}

func (fw *FrameWriter) WriteFrame(frame Frame) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	data, err := frame.Encode()
	if err != nil {
		return err
	}

	// 先写入长度 (4字节)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

	if _, err := fw.writer.Write(lenBuf); err != nil {
		return err
	}

	_, err = fw.writer.Write(data)
	return err
}

// FrameReader 帧读取器
type FrameReader struct {
	reader io.Reader
	mu     sync.Mutex
}

func NewFrameReader(r io.Reader) *FrameReader {
	return &FrameReader{reader: r}
}

func (fr *FrameReader) ReadFrame() (Frame, error) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	// 读取长度
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(fr.reader, lenBuf); err != nil {
		return nil, err
	}

	frameLen := binary.BigEndian.Uint32(lenBuf)

	// 读取帧数据
	data := make([]byte, frameLen)
	if _, err := io.ReadFull(fr.reader, data); err != nil {
		return nil, err
	}

	decoder := NewFrameDecoder()
	return decoder.Decode(data)
}

// QUICTransferSession QUIC 传输会话
type QUICTransferSession struct {
	conn   *quic.Conn
	stream *quic.Stream
	taskID string
	reader *FrameReader
	writer *FrameWriter
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

func NewQUICTransferSession(conn *quic.Conn, stream *quic.Stream, taskID string) *QUICTransferSession {
	ctx, cancel := context.WithCancel(context.Background())

	return &QUICTransferSession{
		conn:   conn,
		stream: stream,
		taskID: taskID,
		reader: NewFrameReader(stream),
		writer: NewFrameWriter(stream),
		ctx:    ctx,
		cancel: cancel,
	}
}

// SendInit 发送初始化帧
func (s *QUICTransferSession) SendInit(fileName string, fileSize int64, checksum string, chunkSize uint32) error {
	frame := NewInitFrame(s.taskID, fileName, fileSize, checksum, chunkSize)
	return s.writer.WriteFrame(frame)
}

// SendData 发送数据帧
func (s *QUICTransferSession) SendData(sequence uint64, offset int64, data []byte) error {
	frame := NewDataFrame(s.taskID, sequence, offset, data)
	return s.writer.WriteFrame(frame)
}

// SendAck 发送确认帧
func (s *QUICTransferSession) SendAck(sequence uint64, offset, received int64) error {
	frame := NewAckFrame(s.taskID, sequence, offset, received)
	return s.writer.WriteFrame(frame)
}

// SendComplete 发送完成帧
func (s *QUICTransferSession) SendComplete(checksum, status string) error {
	frame := NewCompleteFrame(s.taskID, checksum, status)
	return s.writer.WriteFrame(frame)
}

// SendError 发送错误帧
func (s *QUICTransferSession) SendError(code int, message string) error {
	frame := NewErrorFrame(s.taskID, code, message)
	return s.writer.WriteFrame(frame)
}

// SendProgress 发送进度帧
func (s *QUICTransferSession) SendProgress(progress float64, transferred, total, speed int64) error {
	frame := NewProgressFrame(s.taskID, progress, transferred, total, speed)
	return s.writer.WriteFrame(frame)
}

// ReceiveFrame 接收帧
func (s *QUICTransferSession) ReceiveFrame() (Frame, error) {
	return s.reader.ReadFrame()
}

// Close 关闭会话
func (s *QUICTransferSession) Close() error {
	s.cancel()
	if s.stream != nil {
		return s.stream.Close()
	}
	return nil
}
