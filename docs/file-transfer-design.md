# QUIC文件传输功能 - 技术设计文档 (TDD)

**文档版本**: 1.0
**创建日期**: 2026-01-06
**技术负责人**: AI Assistant
**相关文档**: [产品需求文档](./file-transfer-requirements.md)

---

## 1. 架构设计

### 1.1 系统架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              客户端层                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────┐         ┌──────────────────┐                      │
│  │   Web 浏览器      │         │   quic-cli 工具   │                      │
│  │  (Vue 3 前端)     │         │   (Go CLI)        │                      │
│  └────────┬─────────┘         └────────┬─────────┘                      │
│           │                             │                                │
└───────────┼─────────────────────────────┼────────────────────────────────┘
            │ HTTP/HTTPS                  │ QUIC Protocol
            ▼                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              API 网关层                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                    Gin HTTP Server (Port 8475)                    │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │  │
│  │  │ File API    │  │ Auth M/W    │  │ Audit M/W   │              │  │
│  │  │ Handler     │  │             │  │             │              │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘              │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│           │                                                          │   │
└───────────┼──────────────────────────────────────────────────────────┘   │
            │                                                              │
            ▼                                                              │
┌─────────────────────────────────────────────────────────────────────────┐
│                              服务层                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    pkg/filetransfer/                            │   │
│  │  ┌────────────────┐  ┌────────────────┐  ┌─────────────────┐   │   │
│  │  │ Transfer       │  │ Storage        │  │ Progress        │   │   │
│  │  │ Manager        │  │ Manager        │  │ Tracker         │   │   │
│  │  └────────┬───────┘  └────────┬───────┘  └─────────────────┘   │   │
│  └───────────┼──────────────────┼──────────────────────────────────┘   │
│              │                  │                                      │
│  ┌───────────▼──────────────────▼──────────────────────────────────┐   │
│  │                    pkg/protocol/quic                             │   │
│  │              File Transfer Protocol (QUIC Streams)               │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│              │                                                          │
└──────────────┼──────────────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              传输层                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                    pkg/transport/server                           │ │
│  │                    QUIC Server (quic-go)                          │ │
│  │                    Port: 4242 (UDP)                               │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              存储层                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────┐         ┌──────────────────┐                      │
│  │  本地文件系统      │         │   PostgreSQL     │                      │
│  │  /data/quic-files │         │   (元数据)        │                      │
│  └──────────────────┘         └──────────────────┘                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.2 模块划分

```
pkg/filetransfer/
├── manager.go              # 传输管理器，协调整体流程
├── upload.go               # 上传处理逻辑
├── download.go             # 下载处理逻辑
├── storage.go              # 存储抽象层
├── progress.go             # 进度追踪
├── protocol.go             # QUIC 协议适配
├── config.go               # 配置管理
├── metadata.go             # 元数据处理
├── checksum.go             # 校验和计算
├── types.go                # 数据结构定义
├── errors.go               # 错误定义
└── mocks/                  # 测试 Mock
    ├── storage.go
    └── transport.go
```

---

## 2. 核心组件设计

### 2.1 传输管理器 (Transfer Manager)

**职责**: 统一管理所有文件传输任务的生命周期

```go
// pkg/filetransfer/manager.go
package filetransfer

type TransferManager struct {
    config      *Config
    storage     StorageBackend
    tracker     *ProgressTracker
    server      *QUICFileServer
    db          *gorm.DB
    taskQueue   chan *TransferTask
    activeTasks sync.Map // taskID -> *TransferTask
}

type TransferTask struct {
    ID            string
    Type          TransferType // Upload, Download
    FileName      string
    SourcePath    string
    DestPath      string
    FileSize      int64
    Transferred   int64
    Status        TaskStatus
    Speed         int64       // bytes/sec
    CreatedAt     time.Time
    CompletedAt   *time.Time
    Error         error
    Checksum      string      // SHA256
    UserID        string
    ClientIP      string
}

func (tm *TransferManager) Start() error
func (tm *TransferManager) Stop() error
func (tm *TransferManager) SubmitTask(task *TransferTask) (*TransferTask, error)
func (tm *TransferManager) GetTask(id string) (*TransferTask, error)
func (tm *TransferManager) CancelTask(id string) error
func (tm *TransferManager) PauseTask(id string) error
func (tm *TransferManager) ResumeTask(id string) error
```

### 2.2 存储抽象层 (Storage Backend)

**职责**: 提供统一的存储接口，支持多种存储后端

```go
// pkg/filetransfer/storage.go
type StorageBackend interface {
    // 存储文件
    Store(ctx context.Context, path string, reader io.Reader, metadata FileMetadata) error

    // 检索文件
    Retrieve(ctx context.Context, path string) (io.ReadCloser, FileMetadata, error)

    // 删除文件
    Delete(ctx context.Context, path string) error

    // 检查文件是否存在
    Exists(ctx context.Context, path string) (bool, error)

    // 获取文件信息
    Stat(ctx context.Context, path string) (FileMetadata, error)

    // 列出文件
    List(ctx context.Context, prefix string, limit int) ([]FileMetadata, error)
}

type FileMetadata struct {
    Name         string
    Path         string
    Size         int64
    ModTime      time.Time
    ContentType  string
    Checksum     string
    UserID       string
    Tags         []string
}

// 本地文件系统实现
type LocalStorage struct {
    rootPath    string
    pathTemplate string
    quota       int64
}

func (ls *LocalStorage) resolvePath(user, project, filename string) string {
    // 根据 path_template 解析实际路径
    // 示例: "{date}/{user}/{project}/{filename}"
}

func (ls *LocalStorage) checkQuota(size int64) error {
    // 检查配额
}
```

### 2.3 QUIC 文件传输协议

**职责**: 在 QUIC 流上实现高效的文件传输协议

```go
// pkg/filetransfer/protocol.go
// Frame Types
const (
    FrameTypeInit       = 0x01  // 初始化传输
    FrameTypeData       = 0x02  // 数据块
    FrameTypeAck        = 0x03  // 确认
    FrameTypeComplete   = 0x04  // 传输完成
    FrameTypeError      = 0x05  // 错误
    FrameTypeResume     = 0x06  // 断点续传
)

// FileTransferFrame 通用帧结构
type FileTransferFrame struct {
    Type    uint8
    Length  uint32
    Payload []byte
}

// InitFrame 初始化帧
type InitFrame struct {
    TaskID     string
    FileName   string
    FileSize   int64
    Checksum   string
    ChunkSize  uint32
    Options    TransferOptions
}

// DataFrame 数据帧
type DataFrame struct {
    TaskID   string
    Sequence uint64
    Offset   int64
    Data     []byte
}

// AckFrame 确认帧
type AckFrame struct {
    TaskID   string
    Sequence uint64
    Offset   int64
    Received int64
}

// QUICFileServer 服务端
type QUICFileServer struct {
    quicServer *transport.Server
    manager    *TransferManager
    streams    sync.Map // taskID -> QUIC stream
}

func (qfs *QUICFileServer) HandleUpload(stream quic.Stream) error
func (qfs *QUICFileServer) HandleDownload(stream quic.Stream) error
func (qfs *QUICFileServer) SendFile(taskID string, path string) error
```

### 2.4 进度追踪器 (Progress Tracker)

```go
// pkg/filetransfer/progress.go
type ProgressTracker struct {
    sync.RWMutex
    progress map[string]*TransferProgress
    subscribers map[string][]chan ProgressUpdate
}

type TransferProgress struct {
    TaskID      string
    Status      TaskStatus
    Transferred int64
    Total       int64
    Speed       int64           // bytes/sec
    ETA         time.Duration
    StartedAt   time.Time
    UpdatedAt   time.Time
}

type ProgressUpdate struct {
    TaskID      string
    Progress    float64         // 0-100
    Transferred int64
    Speed       string          // "25 MB/s"
    ETA         string          // "00:02:30"
}

func (pt *ProgressTracker) Update(taskID string, transferred int64)
func (pt *ProgressTracker) Get(taskID string) (*TransferProgress, error)
func (pt *ProgressTracker) Subscribe(taskID string) <-chan ProgressUpdate
func (pt *ProgressTracker) Unsubscribe(taskID string, ch chan ProgressUpdate)
```

---

## 3. 数据库设计

### 3.1 文件传输记录表

```sql
CREATE TABLE file_transfers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL UNIQUE,
    file_name       VARCHAR(512) NOT NULL,
    file_path       TEXT NOT NULL,
    file_size       BIGINT NOT NULL,
    file_hash       CHAR(64),
    transfer_type   VARCHAR(10) NOT NULL CHECK (transfer_type IN ('upload', 'download')),
    status          VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'transferring', 'paused', 'completed', 'failed', 'cancelled')),
    progress        INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    speed           BIGINT DEFAULT 0,
    bytes_transferred BIGINT DEFAULT 0,
    user_id         UUID NOT NULL REFERENCES users(id),
    client_ip       INET,
    error_message   TEXT,
    metadata        JSONB,
    started_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at    TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_file_transfers_task_id ON file_transfers(task_id);
CREATE INDEX idx_file_transfers_user_id ON file_transfers(user_id);
CREATE INDEX idx_file_transfers_status ON file_transfers(status);
CREATE INDEX idx_file_transfers_created_at ON file_transfers(created_at DESC);
CREATE INDEX idx_file_transfers_user_status ON file_transfers(user_id, status);

-- 复合索引用于查询活跃传输
CREATE INDEX idx_file_transfers_active ON file_transfers(user_id, status)
    WHERE status IN ('pending', 'transferring', 'paused');

-- 触发器：自动更新 updated_at
CREATE TRIGGER update_file_transfers_updated_at
    BEFORE UPDATE ON file_transfers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 3.2 文件元数据表

```sql
CREATE TABLE file_metadata (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_path       TEXT NOT NULL UNIQUE,
    file_name       VARCHAR(512) NOT NULL,
    file_size       BIGINT NOT NULL,
    file_hash       CHAR(64) NOT NULL,
    content_type    VARCHAR(100),
    storage_path    TEXT NOT NULL,
    user_id         UUID NOT NULL REFERENCES users(id),
    upload_count    INTEGER DEFAULT 1,
    download_count  INTEGER DEFAULT 0,
    tags            TEXT[],
    description     TEXT,
    is_deleted      BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_file_metadata_hash ON file_metadata(file_hash);
CREATE INDEX idx_file_metadata_user_id ON file_metadata(user_id);
CREATE INDEX idx_file_metadata_tags ON file_metadata USING GIN(tags);
CREATE INDEX idx_file_metadata_created_at ON file_metadata(created_at DESC);
```

---

## 4. API 接口详细设计

### 4.1 HTTP API 设计

#### 4.1.1 初始化上传

```http
POST /api/file/upload/init
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "filename": "large-file.iso",
  "file_size": 5368709120,
  "checksum": "sha256:abc123...",
  "path": "/projects/myproject/",
  "metadata": {
    "description": "System image",
    "tags": ["iso", "system"]
  }
}

Response 200:
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "upload_url": "quic://server:4242/upload/550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "chunk_size": 65536
}
```

#### 4.1.2 上传数据块

```http
POST /api/file/upload/chunk
Authorization: Bearer <token>
Content-Type: application/octet-stream

Request Body: Binary chunk data
Query Parameters:
  - task_id: UUID
  - offset: int64 (字节偏移量)
  - sequence: int64 (块序号)

Response 200:
{
  "ack": true,
  "received": 65536,
  "total_received": 131072
}
```

#### 4.1.3 完成上传

```http
POST /api/file/upload/complete
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "checksum": "sha256:abc123..."
}

Response 200:
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "file_path": "/data/quic-files/2026-01-06/user1/project/large-file.iso",
  "file_size": 5368709120,
  "duration_ms": 15234
}
```

#### 4.1.4 请求下载

```http
POST /api/file/download/request
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "file_path": "/remote/path/to/file",
  "offset": 0,  // 用于断点续传
  "options": {
    "verify_checksum": true,
    "threads": 4
  }
}

Response 200:
{
  "task_id": "660e8400-e29b-41d4-a716-446655440000",
  "download_url": "quic://server:4242/download/660e8400-e29b-41d4-a716-446655440000",
  "file_info": {
    "name": "archive.zip",
    "size": 2147483648,
    "checksum": "sha256:def456..."
  },
  "status": "ready"
}
```

#### 4.1.5 查询传输进度

```http
GET /api/file/transfer/{task_id}/progress
Authorization: Bearer <token>

Response 200:
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "transferring",
  "progress": 45.7,
  "transferred": 2453285565,
  "total": 5368709120,
  "speed": {
    "bytes_per_sec": 31457280,
    "formatted": "30.00 MB/s"
  },
  "eta": {
    "seconds": 92,
    "formatted": "00:01:32"
  },
  "started_at": "2026-01-06T14:30:00Z",
  "updated_at": "2026-01-06T14:31:08Z"
}
```

#### 4.1.6 取消传输

```http
DELETE /api/file/transfer/{task_id}
Authorization: Bearer <token>

Response 200:
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "cancelled",
  "transferred": 2453285565,
  "cleanup": true
}
```

#### 4.1.7 列出传输历史

```http
GET /api/file/transfers?status=completed&limit=20&offset=0
Authorization: Bearer <token>

Response 200:
{
  "total": 156,
  "items": [
    {
      "task_id": "550e8400-e29b-41d4-a716-446655440000",
      "file_name": "backup.zip",
      "transfer_type": "upload",
      "status": "completed",
      "file_size": 2147483648,
      "duration_ms": 45230,
      "created_at": "2026-01-06T14:30:00Z"
    }
  ]
}
```

---

### 4.2 WebSocket 实时推送

```javascript
// 客户端订阅传输进度
const ws = new WebSocket('wss://server:8475/ws/file/transfer/progress');

ws.send(JSON.stringify({
  action: 'subscribe',
  task_id: '550e8400-e29b-41d4-a716-446655440000',
  token: 'jwt_token'
}));

// 服务端推送
{
  "event": "progress_update",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "progress": 67.5,
    "speed": "25.3 MB/s",
    "eta": "00:00:45"
  }
}
```

---

## 5. quic-cli 工具设计

### 5.1 命令结构

```bash
quic-cli [command] [flags]

Commands:
  download   从服务器下载文件
  upload     上传文件到服务器
  status     查询传输状态
  cancel     取消传输任务
  verify     验证文件完整性

Global Flags:
  --server string      QUIC 服务器地址 (default "localhost:4242")
  --api-server string  API 服务器地址 (default "https://localhost:8475")
  --token string       认证令牌
  --config string      配置文件路径 (default "~/.quic-cli.yaml")
  --verbose            详细输出
  --timeout duration   超时时间 (default 5m)
```

### 5.2 下载命令实现

```go
// cmd/cli/download.go
type DownloadCommand struct {
    Source      string
    Dest        string
    Resume      bool
    Verify      bool
    Threads     int
    Offset      int64
    ShowProgress bool
}

func (dc *DownloadCommand) Execute(ctx context.Context) error {
    // 1. 调用 HTTP API 初始化下载
    taskID, fileInfo, err := dc.initDownload()
    if err != nil {
        return err
    }

    // 2. 建立 QUIC 连接
    conn, err := dc.establishQUICConnection()
    if err != nil {
        return err
    }
    defer conn.Close()

    // 3. 创建下载流
    stream, err := dc.openDownloadStream(conn, taskID)
    if err != nil {
        return err
    }

    // 4. 执行下载
    progress := dc.startProgressDisplay(fileInfo.Size)
    transferred, err := dc.downloadStream(stream, fileInfo, progress)

    // 5. 验证
    if dc.Verify {
        dc.verifyChecksum(dc.Dest, fileInfo.Checksum)
    }

    return nil
}
```

### 5.3 进度条显示

```go
// 使用 cli/sprogress 库
type ProgressBar struct {
    bar     *mpb.Bar
    proxy   mpb.BarProxy
    current int64
    total   int64
}

func (pb *ProgressBar) Write(p []byte) (int, error) {
    n := len(p)
    pb.current += int64(n)
    pb.proxy.SetTotal(pb.current, false)
    return n, nil
}

// 显示样式:
// [████████████████████░░░░░░░░] 75% | 25.3 MB/s | 00:00:15 / 00:01:00
```

---

## 6. 前端设计

### 6.1 文件上传组件

```vue
<!-- web/src/components/filetransfer/FileUpload.vue -->
<template>
  <div class="file-upload-container">
    <!-- 拖拽区域 -->
    <div
      class="upload-zone"
      :class="{ 'drag-over': isDragOver }"
      @drop.prevent="handleDrop"
      @dragover.prevent="isDragOver = true"
      @dragleave="isDragOver = false"
    >
      <el-icon :size="48"><UploadFilled /></el-icon>
      <p>拖拽文件到此处，或</p>
      <el-button type="primary" @click="selectFiles">选择文件</el-button>
      <input
        ref="fileInput"
        type="file"
        multiple
        @change="handleFileSelect"
        style="display: none"
      />
    </div>

    <!-- 文件列表 -->
    <div class="file-list" v-if="files.length > 0">
      <div v-for="file in files" :key="file.id" class="file-item">
        <div class="file-info">
          <el-icon><Document /></el-icon>
          <span class="file-name">{{ file.name }}</span>
          <span class="file-size">{{ formatSize(file.size) }}</span>
        </div>

        <!-- 进度条 -->
        <el-progress
          :percentage="file.progress"
          :status="file.status"
          :stroke-width="8"
        >
          <template #default="{ percentage }">
            <span class="progress-text">
              {{ percentage }}% | {{ file.speed }} | {{ file.eta }}
            </span>
          </template>
        </el-progress>

        <!-- 操作按钮 -->
        <div class="file-actions">
          <el-button
            v-if="file.status === 'uploading'"
            size="small"
            @click="pauseUpload(file.id)"
          >
            暂停
          </el-button>
          <el-button
            v-if="file.status === 'paused'"
            size="small"
            @click="resumeUpload(file.id)"
          >
            继续
          </el-button>
          <el-button
            size="small"
            type="danger"
            @click="cancelUpload(file.id)"
          >
            取消
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { uploadFile, initUpload, completeUpload } from '@/api/file'

interface UploadFile {
  id: string
  name: string
  size: number
  progress: number
  status: 'pending' | 'uploading' | 'paused' | 'completed' | 'error'
  speed: string
  eta: string
  taskID?: string
}

const files = ref<UploadFile[]>([])
const isDragOver = ref(false)

// 分块上传逻辑
async function uploadFileChunk(file: UploadFile, chunk: Blob, offset: number) {
  const formData = new FormData()
  formData.append('chunk', chunk)

  await uploadFile({
    task_id: file.taskID,
    offset,
    sequence: Math.floor(offset / CHUNK_SIZE),
    data: chunk
  })

  // 更新进度
  file.progress = Math.floor((offset / file.size) * 100)
}
</script>
```

### 6.2 API 客户端

```typescript
// web/src/api/file.ts
import axios from 'axios'
import { useWebSocket } from '@/hooks/useWebSocket'

const api = axios.create({
  baseURL: '/api/file'
})

// 初始化上传
export async function initUpload(params: {
  filename: string
  file_size: number
  path?: string
  metadata?: Record<string, any>
}) {
  const { data } = await api.post('/upload/init', params)
  return data
}

// 上传数据块
export async function uploadChunk(params: {
  task_id: string
  offset: number
  sequence: number
  data: Blob
}) {
  const formData = new FormData()
  formData.append('data', params.data)

  const { data } = await api.post(`/upload/chunk?task_id=${params.task_id}&offset=${params.offset}&sequence=${params.sequence}`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return data
}

// 完成上传
export async function completeUpload(params: { task_id: string; checksum: string }) {
  const { data } = await api.post('/upload/complete', params)
  return data
}

// 订阅进度更新
export function subscribeProgress(taskId: string, callback: (update: ProgressUpdate) => void) {
  return useWebSocket(`/ws/file/transfer/progress`, {
    onMessage: (event) => {
      const message = JSON.parse(event.data)
      if (message.data.task_id === taskId) {
        callback(message.data)
      }
    }
  })
}
```

---

## 7. 性能优化

### 7.1 流式处理

```go
// 避免将整个文件加载到内存
func (tm *TransferManager) StreamUpload(stream io.Reader, destPath string) error {
    file, err := os.Create(destPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // 使用缓冲区
    buffer := make([]byte, tm.config.BufferSize)

    for {
        n, err := stream.Read(buffer)
        if n > 0 {
            if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
                return writeErr
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }

    return nil
}
```

### 7.2 并发传输

```go
// 多线程下载
func (dc *DownloadCommand) MultiThreadDownload(fileInfo FileInfo, threads int) error {
    chunkSize := fileInfo.Size / int64(threads)
    var wg sync.WaitGroup
    errChan := make(chan error, threads)

    for i := 0; i < threads; i++ {
        wg.Add(1)
        go func(threadNum int) {
            defer wg.Done()

            offset := int64(threadNum) * chunkSize
            endOffset := offset + chunkSize
            if threadNum == threads-1 {
                endOffset = fileInfo.Size
            }

            err := dc.downloadChunk(offset, endOffset)
            if err != nil {
                errChan <- err
            }
        }(i)
    }

    wg.Wait()
    close(errChan)

    for err := range errChan {
        if err != nil {
            return err
        }
    }

    return nil
}
```

### 7.3 零拷贝优化

```go
// 使用 sendfile 系统调用
import "syscall"

func zeroCopySendFile(outFD int, inFD int, offset int64, count int) (int, error) {
    return syscall.Sendfile(inFD, outFD, &offset, count)
}
```

---

## 8. 错误处理

### 8.1 错误码定义

```go
// pkg/filetransfer/errors.go
const (
    ErrCodeTaskNotFound     = 40001
    ErrCodeInvalidChecksum  = 40002
    ErrCodeStorageQuotaExceeded = 40003
    ErrCodeFileTooLarge     = 40004
    ErrCodeInvalidOffset    = 40005
    ErrCodeTransferFailed   = 50001
    ErrCodeStorageError     = 50002
)

type TransferError struct {
    Code    int
    Message string
    Cause   error
}

func (e *TransferError) Error() string {
    return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
}

// 预定义错误
var (
    ErrTaskNotFound     = &TransferError{Code: ErrCodeTaskNotFound, Message: "Task not found"}
    ErrInvalidChecksum  = &TransferError{Code: ErrCodeInvalidChecksum, Message: "Checksum verification failed"}
    ErrQuotaExceeded    = &TransferError{Code: ErrCodeStorageQuotaExceeded, Message: "Storage quota exceeded"}
)
```

### 8.2 重试策略

```go
type RetryConfig struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func (rc *RetryConfig) Execute(fn func() error) error {
    delay := rc.InitialDelay

    for attempt := 0; attempt < rc.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }

        // 判断是否可重试
        if !isRetryable(err) {
            return err
        }

        time.Sleep(delay)
        delay = time.Duration(float64(delay) * rc.Multiplier)
        if delay > rc.MaxDelay {
            delay = rc.MaxDelay
        }
    }

    return fmt.Errorf("max retry attempts exceeded")
}
```

---

## 9. 测试策略

### 9.1 单元测试

```go
// pkg/filetransfer/storage_test.go
func TestLocalStorage_Store(t *testing.T) {
    storage := NewLocalStorage("/tmp/test", "{date}/{user}", 1024*1024*100)

    tests := []struct {
        name    string
        path    string
        data    []byte
        wantErr bool
    }{
        {
            name:    "successful store",
            path:    "/test/file.txt",
            data:    []byte("hello world"),
            wantErr: false,
        },
        {
            name:    "quota exceeded",
            path:    "/test/large.bin",
            data:    make([]byte, 200*1024*1024), // 200MB
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := storage.Store(context.Background(), tt.path, bytes.NewReader(tt.data), FileMetadata{})
            if (err != nil) != tt.wantErr {
                t.Errorf("Store() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 9.2 集成测试

```go
// tests/integration/file_transfer_test.go
func TestFileUploadDownloadFlow(t *testing.T) {
    // 启动测试服务器
    server := startTestServer(t)
    defer server.Close()

    // 创建测试文件
    testData := []byte("test data for transfer")
    tempFile := createTempFile(t, testData)
    defer os.Remove(tempFile.Name())

    // 1. 初始化上传
    initResp, err := client.InitUpload(InitUploadRequest{
        Filename: "test.txt",
        FileSize: int64(len(testData)),
    })
    require.NoError(t, err)

    // 2. 上传数据
    err = client.UploadChunk(initResp.TaskID, 0, 0, testData)
    require.NoError(t, err)

    // 3. 完成上传
    completeResp, err := client.CompleteUpload(initResp.TaskID, checksum(testData))
    require.NoError(t, err)

    // 4. 请求下载
    downloadResp, err := client.RequestDownload(completeResp.FilePath)
    require.NoError(t, err)

    // 5. 下载数据
    downloadedData, err := client.DownloadFile(downloadResp.TaskID)
    require.NoError(t, err)

    // 6. 验证
    assert.Equal(t, testData, downloadedData)
}
```

### 9.3 性能测试

```go
// tests/benchmark/file_transfer_benchmark_test.go
func BenchmarkFileTransfer(b *testing.B) {
    server := startTestServer(b)
    defer server.Close()

    fileSizes := []int64{
        10 * 1024 * 1024,      // 10MB
        100 * 1024 * 1024,     // 100MB
        1024 * 1024 * 1024,    // 1GB
    }

    for _, size := range fileSizes {
        b.Run(fmt.Sprintf("%dMB", size/1024/1024), func(b *testing.B) {
            data := make([]byte, size)
            rand.Read(data)

            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                taskID, _ := client.InitUpload("benchmark.bin", size)
                client.UploadData(taskID, data)
                client.CompleteUpload(taskID, "")
            }
        })
    }
}
```

---

## 10. 部署配置

### 10.1 配置示例

```yaml
# config/server.yaml
file_transfer:
  # 基础配置
  enabled: true
  storage_root: "/data/quic-files"
  path_template: "{date}/{user}/{project}"

  # 限制配置
  max_file_size: 10737418240      # 10GB
  max_concurrent_transfers: 100
  storage_quota: 1099511627776    # 1TB
  user_quota: 107374182400        # 100GB per user

  # 性能配置
  buffer_size: 65536              # 64KB
  chunk_size: 1048576             # 1MB
  upload_threads: 4
  download_threads: 4

  # 功能开关
  compression: false
  checksum_verify: true
  resume_support: true

  # 保留策略
  retention_days: 30
  auto_cleanup: true
```

### 10.2 环境变量

```bash
# 覆盖配置文件的环境变量
export QUIC_FILE_STORAGE_ROOT=/mnt/nfs/quic-files
export QUIC_FILE_MAX_SIZE=10737418240
export QUIC_FILE_QUOTA=1099511627776
export QUIC_FILE_RETENTION_DAYS=30
```

---

## 11. 监控指标

### 11.1 Prometheus 指标

```go
// pkg/filetransfer/metrics.go
var (
    // 传输计数
    transferTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "quic_file_transfer_total",
            Help: "Total number of file transfers",
        },
        []string{"type", "status"},
    )

    // 传输大小
    transferSize = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "quic_file_transfer_size_bytes",
            Help:    "File transfer size in bytes",
            Buckets: prometheus.ExponentialBuckets(1024, 2, 20), // 1KB to 1GB+
        },
        []string{"type"},
    )

    // 传输时长
    transferDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "quic_file_transfer_duration_seconds",
            Help:    "File transfer duration",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
        },
        []string{"type"},
    )

    // 传输速度
    transferSpeed = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "quic_file_transfer_speed_bytes_per_sec",
            Help: "Current transfer speed",
        },
        []string{"task_id", "type"},
    )

    // 活跃传输数
    activeTransfers = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "quic_file_active_transfers",
            Help: "Number of active transfers",
        },
    )
)
```

---

## 12. 安全考虑

### 12.1 文件类型验证

```go
type FileValidator struct {
    allowedTypes map[string]bool
    maxFileSize  int64
}

func (fv *FileValidator) Validate(fileHeader *multipart.FileHeader) error {
    // 1. 检查文件扩展名
    ext := filepath.Ext(fileHeader.Filename)
    if !fv.allowedTypes[ext] {
        return fmt.Errorf("file type %s not allowed", ext)
    }

    // 2. 检查文件大小
    if fileHeader.Size > fv.maxFileSize {
        return fmt.Errorf("file too large: %d bytes", fileHeader.Size)
    }

    // 3. 验证魔数（Magic Number）
    file, err := fileHeader.Open()
    if err != nil {
        return err
    }
    defer file.Close()

    header := make([]byte, 512)
    file.Read(header)
    mimeType := http.DetectContentType(header)

    if !fv.isValidMimeType(mimeType, ext) {
        return fmt.Errorf("mismatched file type: %s", mimeType)
    }

    return nil
}
```

### 12.2 路径遍历防护

```go
func sanitizePath(path string) string {
    // 移除路径遍历字符
    path = strings.ReplaceAll(path, "..", "")
    path = strings.ReplaceAll(path, "\\", "")

    // 确保路径在存储根目录内
    path = filepath.Clean(path)
    if filepath.IsAbs(path) {
        path = filepath.Base(path)
    }

    return path
}
```

---

**文档变更历史**

| 版本 | 日期 | 作者 | 变更说明 |
|------|------|------|----------|
| 1.0 | 2026-01-06 | AI Assistant | 初始版本 |
