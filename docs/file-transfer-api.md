# QUIC文件传输功能 - API 接口文档

**文档版本**: 1.0
**创建日期**: 2026-01-06
**相关文档**: [产品需求文档](./file-transfer-requirements.md) | [技术设计文档](./file-transfer-design.md)

---

## 1. API 概览

### 1.1 基础信息

| 项目 | 值 |
|------|-----|
| Base URL | `https://api.quic-flow.com:8475` |
| 协议 | HTTPS (生产环境) / HTTP (开发环境) |
| 数据格式 | JSON / 二进制流 |
| 字符编码 | UTF-8 |
| 认证方式 | JWT Bearer Token |

### 1.2 通用响应格式

#### 成功响应
```json
{
  "success": true,
  "data": { ... },
  "message": "Operation completed successfully"
}
```

#### 错误响应
```json
{
  "success": false,
  "error": {
    "code": "FILE_TRANSFER_ERROR",
    "message": "Detailed error message",
    "details": { ... }
  }
}
```

### 1.3 HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 413 | 文件过大 |
| 429 | 请求过于频繁 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 |

---

## 2. 认证

### 2.1 获取 Token

```http
POST /api/auth/login
Content-Type: application/json

Request:
{
  "username": "admin",
  "password": "password123"
}

Response 200:
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 2.2 使用 Token

```http
GET /api/file/transfers
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

## 3. 文件上传 API

### 3.1 初始化上传

**描述**: 创建一个新的文件上传任务，返回任务ID和上传配置。

```http
POST /api/file/upload/init
```

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**:
```json
{
  "filename": "large-file.iso",
  "file_size": 5368709120,
  "checksum": "sha256:a1b2c3d4e5f6...",
  "content_type": "application/octet-stream",
  "path": "/projects/myproject/",
  "metadata": {
    "description": "System image file",
    "tags": ["iso", "system", "image"],
    "project": "myproject",
    "custom_field": "custom_value"
  },
  "options": {
    "overwrite": false,
    "encryption": false,
    "compression": false
  }
}
```

**参数说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| filename | string | 是 | 文件名 |
| file_size | int64 | 是 | 文件大小（字节） |
| checksum | string | 否 | 预计算的 SHA256 哈希值（用于验证） |
| content_type | string | 否 | MIME 类型 |
| path | string | 否 | 目标路径（相对于存储根目录） |
| metadata | object | 否 | 文件元数据 |
| options | object | 否 | 上传选项 |

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "upload_config": {
      "quic_url": "quic://quic-flow.com:4242/upload/550e8400-e29b-41d4-a716-446655440000",
      "chunk_size": 65536,
      "max_retries": 3,
      "timeout": 300
    },
    "status": "pending",
    "created_at": "2026-01-06T14:30:00Z"
  }
}
```

**错误响应**:
```json
// 400 - 参数错误
{
  "success": false,
  "error": {
    "code": "INVALID_PARAMETERS",
    "message": "File size exceeds maximum allowed size"
  }
}

// 507 - 配额不足
{
  "success": false,
  "error": {
    "code": "QUOTA_EXCEEDED",
    "message": "Storage quota exceeded. Available: 500MB, Required: 5GB",
    "details": {
      "available": 524288000,
      "required": 5368709120
    }
  }
}
```

---

### 3.2 上传数据块

**描述**: 上传文件的分块数据。

```http
POST /api/file/upload/chunk
```

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/octet-stream
X-Task-ID: 550e8400-e29b-41d4-a716-446655440000
X-Chunk-Offset: 0
X-Chunk-Sequence: 0
```

**请求体**: 二进制数据块

**查询参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | string | 是 | 任务ID（也可通过请求头传递） |
| offset | int64 | 是 | 块在文件中的偏移量 |
| sequence | int64 | 是 | 块序号 |
| checksum | string | 否 | 块的校验和（可选） |

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "ack": true,
    "received": 65536,
    "total_received": 131072,
    "progress": 0.0024
  }
}
```

**错误响应**:
```json
// 404 - 任务不存在
{
  "success": false,
  "error": {
    "code": "TASK_NOT_FOUND",
    "message": "Upload task not found or expired"
  }
}

// 409 - 偏移量错误
{
  "success": false,
  "error": {
    "code": "INVALID_OFFSET",
    "message": "Chunk offset mismatch. Expected: 65536, Got: 131072",
    "details": {
      "expected_offset": 65536,
      "received_offset": 131072
    }
  }
}
```

---

### 3.3 完成上传

**描述**: 标记上传完成，进行最终校验。

```http
POST /api/file/upload/complete
```

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "checksum": "sha256:a1b2c3d4e5f6...",
  "metadata": {
    "completed_at": "2026-01-06T14:35:00Z"
  }
}
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "file_info": {
      "file_id": "file_660e8400-e29b-41d4-a716-446655440000",
      "file_name": "large-file.iso",
      "file_path": "/data/quic-files/2026-01-06/admin/myproject/large-file.iso",
      "file_size": 5368709120,
      "checksum": "sha256:a1b2c3d4e5f6...",
      "download_url": "/api/file/download/file_660e8400-e29b-41d4-a716-446655440000"
    },
    "transfer_stats": {
      "duration_ms": 312456,
      "average_speed": "17.18 MB/s",
      "total_bytes": 5368709120
    },
    "completed_at": "2026-01-06T14:35:16Z"
  }
}
```

**错误响应**:
```json
// 400 - 校验失败
{
  "success": false,
  "error": {
    "code": "CHECKSUM_MISMATCH",
    "message": "File checksum verification failed",
    "details": {
      "expected": "sha256:a1b2c3d4e5f6...",
      "actual": "sha256:x9y8z7w6v5u4..."
    }
  }
}
```

---

### 3.4 取消上传

**描述**: 取消正在进行的上传任务。

```http
DELETE /api/file/upload/{task_id}
```

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| task_id | string | 任务ID |

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "cancelled",
    "transferred": 2147483648,
    "cleanup": true,
    "message": "Upload cancelled and temporary files cleaned up"
  }
}
```

---

## 4. 文件下载 API

### 4.1 请求下载

**描述**: 创建一个新的下载任务。

```http
POST /api/file/download/request
```

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**:
```json
{
  "file_id": "file_660e8400-e29b-41d4-a716-446655440000",
  "file_path": "/remote/path/to/file",
  "local_path": "/local/destination/path",
  "offset": 0,
  "options": {
    "verify_checksum": true,
    "resume": true,
    "threads": 4,
    "compression": false
  }
}
```

**参数说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_id | string | 条件必填 | 文件ID（与file_path二选一） |
| file_path | string | 条件必填 | 文件路径（与file_id二选一） |
| local_path | string | 否 | 本地保存路径（用于记录） |
| offset | int64 | 否 | 起始偏移量（断点续传） |
| options | object | 否 | 下载选项 |

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "770e8400-e29b-41d4-a716-446655440000",
    "download_config": {
      "quic_url": "quic://quic-flow.com:4242/download/770e8400-e29b-41d4-a716-446655440000",
      "file_info": {
        "name": "archive.zip",
        "size": 2147483648,
        "checksum": "sha256:def456...",
        "content_type": "application/zip"
      },
      "chunk_size": 1048576,
      "timeout": 600
    },
    "status": "ready",
    "created_at": "2026-01-06T14:40:00Z"
  }
}
```

**错误响应**:
```json
// 404 - 文件不存在
{
  "success": false,
  "error": {
    "code": "FILE_NOT_FOUND",
    "message": "Requested file does not exist or has been deleted"
  }
}
```

---

### 4.2 获取下载流

**描述**: 通过 QUIC 协议获取文件数据流。

**注意**: 此接口通过 QUIC 协议实现，非 HTTP。

```
QUIC Stream: /download/{task_id}
```

**流程**:
1. 客户端建立 QUIC 连接到服务器
2. 打开指定路径的流
3. 服务端发送文件元数据（InitFrame）
4. 服务端发送文件数据块（DataFrame）
5. 客户端发送确认（AckFrame）
6. 传输完成或出错

**InitFrame 格式**:
```json
{
  "type": "init",
  "task_id": "770e8400-e29b-41d4-a716-446655440000",
  "file_name": "archive.zip",
  "file_size": 2147483648,
  "checksum": "sha256:def456...",
  "chunk_size": 1048576
}
```

---

### 4.3 断点续传

**描述**: 从断点位置恢复下载。

```http
POST /api/file/download/resume
```

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**:
```json
{
  "task_id": "770e8400-e29b-41d4-a716-446655440000",
  "offset": 1073741824
}
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "770e8400-e29b-41d4-a716-446655440000",
    "status": "resuming",
    "offset": 1073741824,
    "remaining": 1073741824
  }
}
```

---

### 4.4 取消下载

**描述**: 取消正在进行的下载任务。

```http
DELETE /api/file/download/{task_id}
```

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "770e8400-e29b-41d4-a716-446655440000",
    "status": "cancelled",
    "downloaded": 1073741824,
    "message": "Download cancelled"
  }
}
```

---

## 5. 传输状态查询 API

### 5.1 查询任务进度

**描述**: 获取指定传输任务的实时进度。

```http
GET /api/file/transfer/{task_id}/progress
```

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| task_id | string | 任务ID |

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "transferring",
    "progress": 45.7,
    "transferred": 2453285565,
    "total": 5368709120,
    "speed": {
      "bytes_per_sec": 31457280,
      "formatted": "30.00 MB/s",
      "average": "25.50 MB/s"
    },
    "eta": {
      "seconds": 92,
      "formatted": "00:01:32"
    },
    "timeline": {
      "started_at": "2026-01-06T14:30:00Z",
      "updated_at": "2026-01-06T14:31:08Z",
      "estimated_completion": "2026-01-06T14:31:40Z"
    },
    "retries": 0
  }
}
```

---

### 5.2 批量查询任务状态

**描述**: 批量获取多个任务的状态。

```http
POST /api/file/transfer/batch-status
```

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**:
```json
{
  "task_ids": [
    "550e8400-e29b-41d4-a716-446655440000",
    "660e8400-e29b-41d4-a716-446655440000",
    "770e8400-e29b-41d4-a716-446655440000"
  ]
}
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "tasks": [
      {
        "task_id": "550e8400-e29b-41d4-a716-446655440000",
        "status": "completed",
        "progress": 100
      },
      {
        "task_id": "660e8400-e29b-41d4-a716-446655440000",
        "status": "transferring",
        "progress": 67.5
      },
      {
        "task_id": "770e8400-e29b-41d4-a716-446655440000",
        "status": "pending",
        "progress": 0
      }
    ]
  }
}
```

---

### 5.3 列出传输历史

**描述**: 获取用户的传输历史记录。

```http
GET /api/file/transfers
```

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| type | string | 否 | all | 传输类型: upload/download/all |
| status | string | 否 | all | 状态: pending/transferring/completed/failed/cancelled/all |
| limit | int | 否 | 20 | 每页数量 |
| offset | int | 否 | 0 | 偏移量 |
| start_date | string | 否 | - | 开始日期 (ISO 8601) |
| end_date | string | 否 | - | 结束日期 (ISO 8601) |
| sort_by | string | 否 | created_at | 排序字段: created_at/file_size/duration |
| sort_order | string | 否 | desc | 排序方向: asc/desc |

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "total": 156,
    "page": 1,
    "page_size": 20,
    "items": [
      {
        "task_id": "550e8400-e29b-41d4-a716-446655440000",
        "file_name": "backup.zip",
        "transfer_type": "upload",
        "status": "completed",
        "file_size": 2147483648,
        "progress": 100,
        "speed": "25.3 MB/s",
        "duration_ms": 85234,
        "created_at": "2026-01-06T14:30:00Z",
        "completed_at": "2026-01-06T14:31:25Z"
      },
      {
        "task_id": "660e8400-e29b-41d4-a716-446655440000",
        "file_name": "data.tar.gz",
        "transfer_type": "download",
        "status": "transferring",
        "file_size": 5368709120,
        "progress": 45.7,
        "speed": "18.2 MB/s",
        "created_at": "2026-01-06T14:25:00Z"
      }
    ]
  }
}
```

---

### 5.4 获取任务详情

**描述**: 获取指定任务的详细信息。

```http
GET /api/file/transfer/{task_id}
```

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "transfer_type": "upload",
    "status": "completed",
    "file_info": {
      "file_id": "file_660e8400-e29b-41d4-a716-446655440000",
      "file_name": "backup.zip",
      "file_path": "/remote/path/to/backup.zip",
      "file_size": 2147483648,
      "content_type": "application/zip",
      "checksum": "sha256:a1b2c3d4..."
    },
    "progress": {
      "current": 2147483648,
      "total": 2147483648,
      "percentage": 100
    },
    "statistics": {
      "started_at": "2026-01-06T14:30:00Z",
      "completed_at": "2026-01-06T14:31:25Z",
      "duration_ms": 85234,
      "average_speed": "25.3 MB/s",
      "peak_speed": "32.1 MB/s",
      "retries": 0,
      "bytes_transferred": 2147483648
    },
    "user_info": {
      "user_id": "user-123",
      "username": "admin",
      "client_ip": "192.168.1.100"
    },
    "metadata": {
      "description": "Nightly backup",
      "tags": ["backup", "nightly"],
      "project": "main"
    }
  }
}
```

---

## 6. 文件管理 API

### 6.1 列出文件

**描述**: 列出用户可访问的文件。

```http
GET /api/file/list
```

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| path | string | 否 | / | 目录路径 |
| recursive | bool | 否 | false | 是否递归列出子目录 |
| limit | int | 否 | 100 | 返回数量限制 |
| offset | int | 否 | 0 | 偏移量 |
| sort_by | string | 否 | name | 排序字段: name/size/modified |
| sort_order | string | 否 | asc | 排序方向 |

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "path": "/projects/myproject",
    "files": [
      {
        "file_id": "file-001",
        "name": "config.yaml",
        "type": "file",
        "size": 2048,
        "modified_at": "2026-01-06T10:30:00Z",
        "content_type": "text/yaml",
        "checksum": "sha256:abc123..."
      },
      {
        "file_id": "file-002",
        "name": "data",
        "type": "directory",
        "size": 0,
        "item_count": 15,
        "modified_at": "2026-01-06T09:15:00Z"
      }
    ],
    "total": 2
  }
}
```

---

### 6.2 获取文件信息

**描述**: 获取指定文件的详细信息。

```http
GET /api/file/info/{file_id}
```

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "file_id": "file-001",
    "file_name": "config.yaml",
    "file_path": "/projects/myproject/config.yaml",
    "file_size": 2048,
    "content_type": "text/yaml",
    "checksum": "sha256:abc123...",
    "created_at": "2026-01-05T14:30:00Z",
    "modified_at": "2026-01-06T10:30:00Z",
    "owner": {
      "user_id": "user-123",
      "username": "admin"
    },
    "permissions": {
      "read": true,
      "write": true,
      "delete": true,
      "share": true
    },
    "statistics": {
      "upload_count": 1,
      "download_count": 15,
      "last_accessed": "2026-01-06T12:00:00Z"
    },
    "metadata": {
      "description": "Application configuration",
      "tags": ["config", "yaml"],
      "project": "myproject"
    }
  }
}
```

---

### 6.3 删除文件

**描述**: 删除指定的文件。

```http
DELETE /api/file/{file_id}
```

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "file_id": "file-001",
    "status": "deleted",
    "deleted_at": "2026-01-06T15:00:00Z"
  }
}
```

---

### 6.4 更新文件元数据

**描述**: 更新文件的元数据信息。

```http
PATCH /api/file/{file_id}/metadata
```

**请求头**:
```
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**:
```json
{
  "description": "Updated description",
  "tags": ["config", "yaml", "updated"],
  "custom_field": "updated_value"
}
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "file_id": "file-001",
    "updated_fields": ["description", "tags"],
    "updated_at": "2026-01-06T15:05:00Z"
  }
}
```

---

## 7. 配置管理 API

### 7.1 获取存储配额

**描述**: 获取用户的存储配额信息。

```http
GET /api/file/quota
```

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "user_id": "user-123",
    "quota": {
      "total": 107374182400,
      "used": 53687091200,
      "available": 53687091200,
      "usage_percentage": 50.0,
      "formatted": {
        "total": "100 GB",
        "used": "50 GB",
        "available": "50 GB"
      }
    },
    "file_count": 1250,
    "project_breakdown": [
      {
        "project": "myproject",
        "size": 32212254720,
        "file_count": 850
      },
      {
        "project": "test",
        "size": 21474836480,
        "file_count": 400
      }
    ]
  }
}
```

---

### 7.2 获取系统配置

**描述**: 获取文件传输系统的配置信息。

```http
GET /api/file/config
```

**请求头**:
```
Authorization: Bearer <token>
```

**成功响应 200**:
```json
{
  "success": true,
  "data": {
    "upload": {
      "max_file_size": 10737418240,
      "max_concurrent_uploads": 5,
      "chunk_size": 65536,
      "supported_formats": ["*"],
      "checksum_required": false
    },
    "download": {
      "max_concurrent_downloads": 10,
      "chunk_size": 1048576,
      "resume_support": true,
      "multi_thread_support": true,
      "max_threads": 8
    },
    "storage": {
      "retention_days": 30,
      "auto_cleanup": true,
      "compression_available": false
    },
    "quotas": {
      "user_quota": 107374182400,
      "project_quota": 53687091200
    }
  }
}
```

---

## 8. WebSocket 实时通知

### 8.1 连接端点

```
wss://api.quic-flow.com:8475/ws/file/transfer
```

### 8.2 订阅进度更新

**客户端发送**:
```json
{
  "action": "subscribe",
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "token": "jwt_token"
}
```

**服务端响应**:
```json
{
  "event": "subscribed",
  "task_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 8.3 进度更新推送

**服务端推送**:
```json
{
  "event": "progress_update",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "progress": 67.5,
    "transferred": 3623878656,
    "total": 5368709120,
    "speed": "25.3 MB/s",
    "eta": "00:00:45",
    "updated_at": "2026-01-06T14:31:15Z"
  }
}
```

### 8.4 传输完成通知

**服务端推送**:
```json
{
  "event": "transfer_complete",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "duration_ms": 15234,
    "file_info": {
      "file_id": "file-001",
      "file_path": "/remote/path/to/file"
    }
  }
}
```

### 8.5 错误通知

**服务端推送**:
```json
{
  "event": "transfer_error",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "error": {
      "code": "CONNECTION_LOST",
      "message": "Connection to server lost",
      "retryable": true,
      "retry_after": 5
    }
  }
}
```

### 8.6 取消订阅

**客户端发送**:
```json
{
  "action": "unsubscribe",
  "task_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

## 9. 错误码参考

| 错误码 | HTTP状态 | 说明 |
|--------|----------|------|
| INVALID_PARAMETERS | 400 | 请求参数无效 |
| UNAUTHORIZED | 401 | 未授权访问 |
| FORBIDDEN | 403 | 禁止访问 |
| FILE_NOT_FOUND | 404 | 文件不存在 |
| TASK_NOT_FOUND | 404 | 任务不存在 |
| FILE_ALREADY_EXISTS | 409 | 文件已存在 |
| INVALID_OFFSET | 409 | 偏移量无效 |
| CHECKSUM_MISMATCH | 400 | 校验和不匹配 |
| QUOTA_EXCEEDED | 507 | 配额超限 |
| FILE_TOO_LARGE | 413 | 文件过大 |
| TRANSFER_FAILED | 500 | 传输失败 |
| STORAGE_ERROR | 500 | 存储错误 |
| CONFLICT | 409 | 操作冲突 |
| RATE_LIMIT_EXCEEDED | 429 | 超出速率限制 |

---

## 10. SDK 使用示例

### 10.1 Go SDK

```go
package main

import (
    "context"
    "fmt"
    "github.com/quic-flow/go-sdk/filetransfer"
)

func main() {
    // 创建客户端
    client := filetransfer.NewClient(&filetransfer.Config{
        BaseURL: "https://api.quic-flow.com:8475",
        Token:   "your-jwt-token",
    })

    ctx := context.Background()

    // 初始化上传
    task, err := client.InitUpload(ctx, &filetransfer.InitUploadRequest{
        Filename: "large-file.iso",
        FileSize: 5368709120,
        Path:     "/projects/myproject/",
    })
    if err != nil {
        panic(err)
    }

    // 上传文件
    err = client.UploadFile(ctx, task.TaskID, "/local/path/to/large-file.iso")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Upload completed: %s\n", task.TaskID)
}
```

### 10.2 JavaScript SDK

```javascript
import { FileTransferClient } from '@quic-flow/js-sdk';

const client = new FileTransferClient({
  baseURL: 'https://api.quic-flow.com:8475',
  token: 'your-jwt-token'
});

// 上传文件
async function uploadFile(file) {
  const task = await client.initUpload({
    filename: file.name,
    fileSize: file.size,
    path: '/uploads/'
  });

  // 订阅进度
  client.subscribeProgress(task.taskId, (progress) => {
    console.log(`Progress: ${progress.progress}%`);
  });

  await client.uploadFile(task.taskId, file);
  console.log('Upload completed!');
}

// 下载文件
async function downloadFile(fileId) {
  const task = await client.requestDownload({ fileId });

  client.subscribeProgress(task.taskId, (progress) => {
    console.log(`Progress: ${progress.progress}%`);
  });

  await client.downloadFile(task.taskId, './downloaded-file.bin');
}
```

---

**文档变更历史**

| 版本 | 日期 | 作者 | 变更说明 |
|------|------|------|----------|
| 1.0 | 2026-01-06 | AI Assistant | 初始版本 |
