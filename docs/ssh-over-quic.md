# SSH-over-QUIC 实现文档

## 概述

T067 功能在 QUIC 流上实现 SSH 协议层，支持反向 SSH 隧道，实现内网穿透。

## 架构

```
┌──────────────────────────────────────────────────────────────┐
│                        应用层                                 │
│    ┌────────────────────────────────────────────────────┐   │
│    │              SSH 协议层                             │   │
│    │  - 权限控制 (密码/公钥认证)                         │   │
│    │  - Shell 交互                                       │   │
│    │  - 端口转发                                         │   │
│    │  - 文件传输 (SCP/SFTP)                             │   │
│    └────────────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────────────────┤
│                        隧道层                                 │
│    ┌────────────────────────────────────────────────────┐   │
│    │              QUIC 协议层                            │   │
│    │  - 多路复用 (一个连接支持多个 SSH 会话)             │   │
│    │  - TLS 1.3 加密                                    │   │
│    │  - 拥塞控制                                         │   │
│    │  - 0-RTT 连接恢复                                   │   │
│    └────────────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────────────────┤
│                        传输层                                 │
│    ┌────────────────────────────────────────────────────┐   │
│    │                 UDP                                 │   │
│    └────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

## 工作原理

### 反向 SSH 隧道

传统 SSH 需要服务端在公网监听端口，客户端主动连接。但在内网穿透场景中：

1. **内网机器**（QUIC 客户端）无法直接从公网访问
2. **公网服务器**（QUIC 服务端）可以被访问

SSH-over-QUIC 反转了这个关系：

1. **内网机器**主动连接到公网服务器，建立 QUIC 长连接
2. **内网机器**在 QUIC 连接上运行 SSH Server
3. **公网服务器**通过已建立的 QUIC 连接，作为 SSH Client 连接到内网的 SSH Server

### 流程图

```
内网机器 (QUIC Client + SSH Server)          公网服务器 (QUIC Server + SSH Client)
    │                                                    │
    │  ──────────── 1. QUIC 连接 ──────────────────>    │
    │               (内网主动发起)                       │
    │                                                    │
    │  <───────────── 2. SSH 流 ────────────────────    │
    │               (服务器请求 SSH 连接)                │
    │                                                    │
    │  ──────────── 3. SSH 握手 ──────────────────>     │
    │               (双向)                               │
    │                                                    │
    │  <═══════════ 4. SSH 会话 ═══════════════════>    │
    │               (Shell/Exec/端口转发)                │
```

## 文件结构

```
pkg/ssh/
├── adapter.go     # StreamConn 适配器：将 quic.Stream 转换为 net.Conn
├── protocol.go    # 流类型识别协议（魔数 + 版本 + 类型）
├── errors.go      # 错误定义
├── config.go      # SSH 服务器和客户端配置
├── server.go      # SSH 服务器（在内网机器运行）
├── client.go      # SSH 客户端（在公网服务器运行）
└── manager.go     # SSH 连接管理器
```

## 使用方法

### 在内网机器（QUIC 客户端）配置 SSH 服务器

```go
import (
    "github.com/voilet/quic-flow/pkg/ssh"
    gossh "golang.org/x/crypto/ssh"
)

// 创建 SSH 服务器配置
serverConfig := ssh.DefaultServerConfig()
serverConfig.PasswordAuth = true
serverConfig.PasswordCallback = func(conn gossh.ConnMetadata, password []byte) (*gossh.Permissions, error) {
    if conn.User() == "admin" && string(password) == "secret" {
        return nil, nil
    }
    return nil, fmt.Errorf("authentication failed")
}

// 创建并启动 SSH 管理器
manager := ssh.NewManager()
manager.InitServer(serverConfig)
manager.StartServer()

// 当收到 SSH 类型的 QUIC 流时：
// manager.HandleSSHStream(stream, conn)
```

### 在公网服务器（QUIC 服务端）配置 SSH 客户端

```go
import (
    "github.com/voilet/quic-flow/pkg/ssh"
)

// 创建 SSH 客户端配置
clientConfig := &ssh.ClientConfig{
    User:     "admin",
    Password: "secret",
}

// 打开 QUIC 流并建立 SSH 连接
client, err := ssh.NewClient(clientConfig)
client.Connect(stream, conn)

// 执行命令
output, err := client.RunCommand("ls -la")

// 或启动交互式 Shell
session, err := client.StartShell(nil)
```

## 流类型识别

为了区分普通业务数据和 SSH 流，使用 6 字节的握手头部：

```
+----------------+----------+----------+
|  Magic (4B)    | Ver (1B) | Type(1B) |
+----------------+----------+----------+
| 0x51534853     |    1     |   0-3    |
| "QSSH"         |          |          |
+----------------+----------+----------+
```

流类型：
- 0x00: 普通数据流
- 0x01: SSH 流
- 0x02: 文件传输流
- 0x03: 端口转发流

## 安全考虑

1. **双重加密**：QUIC 层使用 TLS 1.3 加密，SSH 层再次加密
2. **双重认证**：QUIC 层可以使用客户端证书认证，SSH 层使用密码/公钥认证
3. **主机密钥**：生产环境应验证 SSH 主机密钥，防止中间人攻击
4. **密钥管理**：建议使用公钥认证而非密码认证

## 性能优势

1. **多路复用**：同一 QUIC 连接可以支持多个 SSH 会话
2. **0-RTT 恢复**：QUIC 支持快速连接恢复
3. **头部压缩**：QUIC 使用更高效的头部压缩
4. **连接迁移**：网络切换时保持连接

## 配置选项

### SSH 服务器配置

| 选项                 | 类型     | 默认值       | 说明                     |
|---------------------|----------|-------------|--------------------------|
| HostKeyPath         | string   | ""          | 主机密钥文件路径          |
| PasswordAuth        | bool     | true        | 启用密码认证              |
| NoClientAuth        | bool     | false       | 禁用认证（不安全）        |
| Shell               | string   | "/bin/sh"   | 默认 Shell               |
| IdleTimeout         | Duration | 30m         | 空闲超时                  |
| MaxAuthTries        | int      | 3           | 最大认证尝试次数          |
| AllowTcpForwarding  | bool     | true        | 允许 TCP 端口转发         |
| AllowPty            | bool     | true        | 允许分配伪终端            |

### SSH 客户端配置

| 选项            | 类型     | 默认值  | 说明                    |
|----------------|----------|--------|-------------------------|
| User           | string   | "root" | 用户名                   |
| Password       | string   | ""     | 密码（密码认证）          |
| PrivateKeyPath | string   | ""     | 私钥文件路径（公钥认证）   |
| Timeout        | Duration | 30s    | 连接超时                  |
| KeepAlive      | Duration | 30s    | 保活间隔                  |
