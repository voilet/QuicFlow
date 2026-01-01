# WebSSH 增强功能 TODO

## 概述

在现有 SSH-over-QUIC 终端基础上，增加以下企业级功能：
1. **trzsz 文件传输** - 支持通过终端上传下载文件
2. **命令审计** - 记录所有执行的命令用于安全审计
3. **会话录像回放** - 完整录制终端会话，支持动态回放

---

## 功能一：trzsz 文件传输

### 背景
[trzsz](https://github.com/trzsz/trzsz-go) 是 rz/sz 的现代替代品，支持通过终端传输文件，无需额外端口。

### 实现方案

#### 1.1 服务端集成
```go
// pkg/ssh/trzsz.go
type TrzszHandler struct {
    enabled bool
    maxSize int64  // 最大传输文件大小
    allowedPaths []string  // 允许的路径白名单
}
```

#### 1.2 客户端要求
- Client 机器需安装 `trzsz` 命令行工具
- 或使用内置的 trzsz 协议处理

#### 1.3 Web 前端集成
```javascript
// web/src/utils/trzsz.js
import { TrzszFilter } from 'trzsz'

// 在 xterm.js 中集成 trzsz 过滤器
const trzszFilter = new TrzszFilter({
    writeToTerminal: (data) => term.write(data),
    sendToServer: (data) => ws.send(data),
    chooseSendFiles: async () => { /* 文件选择对话框 */ },
    chooseSaveDirectory: async () => { /* 保存目录选择 */ },
})
```

### 任务清单
- [ ] T-TRZSZ-01: 研究 trzsz-go 库集成方式
- [ ] T-TRZSZ-02: 在 pkg/ssh/server.go 中添加 trzsz 支持
- [ ] T-TRZSZ-03: Web 前端安装 trzsz.js 依赖
- [ ] T-TRZSZ-04: Terminal.vue 集成 TrzszFilter
- [ ] T-TRZSZ-05: 添加文件传输进度条 UI
- [ ] T-TRZSZ-06: 添加传输大小限制和路径白名单配置
- [ ] T-TRZSZ-07: 测试上传下载功能

---

## 功能二：命令审计

### 需求
记录用户执行的每条命令，包含：
- 执行时间
- 执行用户
- 客户端 ID
- 命令内容
- 执行结果状态（成功/失败）

### 数据结构

```go
// pkg/audit/command_log.go
type CommandLog struct {
    ID          string    `json:"id"`
    SessionID   string    `json:"session_id"`
    ClientID    string    `json:"client_id"`
    Username    string    `json:"username"`
    Command     string    `json:"command"`
    ExecutedAt  time.Time `json:"executed_at"`
    ExitCode    int       `json:"exit_code"`
    Duration    int64     `json:"duration_ms"`
    RemoteIP    string    `json:"remote_ip"`
}
```

### 实现方案

#### 2.1 命令检测
通过 PTY 输出流检测命令：
- 检测 `\r` 或 `\n` 后的输入作为命令
- 解析 PS1 提示符来分隔命令
- 使用 shell 的 PROMPT_COMMAND 或 precmd hook

#### 2.2 存储后端
```go
// pkg/audit/store.go
type AuditStore interface {
    SaveCommand(log *CommandLog) error
    QueryCommands(filter *CommandFilter) ([]*CommandLog, error)
    GetCommandsBySession(sessionID string) ([]*CommandLog, error)
}

// 实现：
// - FileStore: 写入本地 JSON/CSV 文件
// - SQLiteStore: 本地 SQLite 数据库
// - (可选) ElasticSearch: 用于大规模部署
```

#### 2.3 API 端点
```
GET  /api/audit/commands              # 查询命令列表
GET  /api/audit/commands/:session_id  # 按会话查询
GET  /api/audit/export                # 导出审计日志
```

### 任务清单
- [ ] T-AUDIT-01: 设计命令审计数据结构
- [ ] T-AUDIT-02: 实现命令检测逻辑（PTY 流解析）
- [ ] T-AUDIT-03: 实现 FileStore 存储后端
- [ ] T-AUDIT-04: 实现 SQLiteStore 存储后端
- [ ] T-AUDIT-05: 添加审计 API 端点
- [ ] T-AUDIT-06: Web 端审计日志查看页面
- [ ] T-AUDIT-07: 添加导出功能（CSV/JSON）
- [ ] T-AUDIT-08: 添加审计日志保留策略（自动清理）

---

## 功能三：会话录像回放

### 需求
完整录制终端会话，支持：
- 实时录制所有终端输出
- 记录精确的时间戳
- 支持动态回放（可暂停、快进、慢放）
- 类似 asciinema 的体验

### 数据格式

采用 [asciicast v2](https://github.com/asciinema/asciinema/blob/develop/doc/asciicast-v2.md) 格式：

```json
// 文件头
{"version": 2, "width": 80, "height": 24, "timestamp": 1234567890, "env": {"TERM": "xterm-256color"}}
// 事件流（每行一个事件）
[0.0, "o", "$ "]
[0.5, "i", "l"]
[0.6, "i", "s"]
[0.7, "i", "\r"]
[0.8, "o", "file1.txt  file2.txt\r\n$ "]
```

事件类型：
- `o`: 输出（服务器 → 客户端）
- `i`: 输入（客户端 → 服务器）

### 实现方案

#### 3.1 录制器
```go
// pkg/recording/recorder.go
type SessionRecorder struct {
    sessionID   string
    startTime   time.Time
    width       int
    height      int
    events      []Event
    file        *os.File
    mu          sync.Mutex
}

type Event struct {
    Time   float64 `json:"time"`   // 相对开始时间（秒）
    Type   string  `json:"type"`   // "o" 或 "i"
    Data   string  `json:"data"`   // 内容
}

func (r *SessionRecorder) RecordOutput(data []byte)
func (r *SessionRecorder) RecordInput(data []byte)
func (r *SessionRecorder) RecordResize(cols, rows int)
func (r *SessionRecorder) Close() error
```

#### 3.2 播放器 API
```
GET  /api/recordings                    # 录像列表
GET  /api/recordings/:id                # 获取录像元数据
GET  /api/recordings/:id/stream         # SSE 流式播放
GET  /api/recordings/:id/download       # 下载 asciicast 文件
DELETE /api/recordings/:id              # 删除录像
```

#### 3.3 Web 播放器
```javascript
// web/src/components/AsciiPlayer.vue
// 使用 asciinema-player 或自定义播放器

import 'asciinema-player'

// 播放器控件：
// - 播放/暂停
// - 进度条（可拖动）
// - 播放速度（0.5x, 1x, 2x, 4x）
// - 全屏
```

#### 3.4 存储结构
```
data/recordings/
├── 2026/
│   └── 01/
│       └── 01/
│           ├── session-abc123.cast    # 录像文件
│           └── session-abc123.meta    # 元数据
```

### 任务清单
- [ ] T-REC-01: 设计录像数据结构（asciicast v2 兼容）
- [ ] T-REC-02: 实现 SessionRecorder
- [ ] T-REC-03: 在 PTY 会话中集成录制器
- [ ] T-REC-04: 实现录像文件存储管理
- [ ] T-REC-05: 添加录像 API 端点
- [ ] T-REC-06: Web 端录像列表页面
- [ ] T-REC-07: 集成 asciinema-player 或实现自定义播放器
- [ ] T-REC-08: 添加录像搜索和过滤功能
- [ ] T-REC-09: 添加录像保留策略和自动清理
- [ ] T-REC-10: 支持录像导出和分享

---

## 配置项

```yaml
# config/server.yaml
webssh:
  # trzsz 文件传输
  trzsz:
    enabled: true
    max_file_size: "100MB"
    allowed_paths:
      - "/tmp"
      - "/home"

  # 命令审计
  audit:
    enabled: true
    store_type: "sqlite"  # file, sqlite
    store_path: "data/audit.db"
    retention_days: 90

  # 会话录像
  recording:
    enabled: true
    store_path: "data/recordings"
    retention_days: 30
    max_file_size: "50MB"
```

---

## 文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `pkg/ssh/trzsz.go` | 新建 | trzsz 协议处理 |
| `pkg/audit/command_log.go` | 新建 | 命令审计数据结构 |
| `pkg/audit/store.go` | 新建 | 审计存储接口 |
| `pkg/audit/file_store.go` | 新建 | 文件存储实现 |
| `pkg/audit/sqlite_store.go` | 新建 | SQLite 存储实现 |
| `pkg/recording/recorder.go` | 新建 | 会话录制器 |
| `pkg/recording/player.go` | 新建 | 录像播放器 |
| `pkg/recording/store.go` | 新建 | 录像存储管理 |
| `pkg/api/audit_api.go` | 新建 | 审计 API |
| `pkg/api/recording_api.go` | 新建 | 录像 API |
| `cmd/server/ssh.go` | 修改 | 集成录制和审计 |
| `web/src/views/AuditLog.vue` | 新建 | 审计日志页面 |
| `web/src/views/Recordings.vue` | 新建 | 录像列表页面 |
| `web/src/components/AsciiPlayer.vue` | 新建 | 录像播放器组件 |
| `web/src/views/Terminal.vue` | 修改 | 集成 trzsz |

---

## 实现优先级

1. **Phase 1**: 命令审计（基础安全需求）
2. **Phase 2**: 会话录像回放（合规需求）
3. **Phase 3**: trzsz 文件传输（便利性功能）

---

## 依赖

### Go
```bash
go get github.com/trzsz/trzsz-go
go get github.com/mattn/go-sqlite3
```

### Web
```bash
cd web
npm install trzsz
npm install asciinema-player
```
