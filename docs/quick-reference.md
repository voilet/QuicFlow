# QUIC Backbone 快速参考

快速查阅常用命令和参数。完整文档请参考 [配置指南](configuration-guide.md)。

## 服务器 (quic-server)

### 基本用法

```bash
# 默认配置启动
./bin/quic-server

# 自定义端口
./bin/quic-server -addr :9000 -api :9001

# 自定义证书
./bin/quic-server -cert /path/to/cert.pem -key /path/to/key.pem
```

### 参数速查

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-addr` | `:8474` | QUIC 监听地址 |
| `-api` | `:8475` | HTTP API 地址 |
| `-cert` | `certs/server-cert.pem` | TLS 证书 |
| `-key` | `certs/server-key.pem` | TLS 私钥 |

---

## 客户端 (quic-client)

### 基本用法

```bash
# 连接本地服务器
./bin/quic-client

# 连接远程服务器
./bin/quic-client -server 192.168.1.100:8474 -id my-client

# 生产环境（启用证书验证）
./bin/quic-client -server prod.example.com:8474 -id prod-001 -insecure=false
```

### 参数速查

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-server` | `localhost:8474` | 服务器地址 |
| `-id` | `client-001` | 客户端 ID |
| `-insecure` | `true` | 跳过证书验证 |

⚠️ **生产环境必须设置 `-insecure=false`**

---

## CLI 工具 (quic-ctl)

### 列出客户端

```bash
# 查看所有在线客户端
./bin/quic-ctl list

# 指定 API 地址
./bin/quic-ctl list -api http://server:8475
```

### 发送消息

```bash
# 发送命令
./bin/quic-ctl send -client client-001 -type command -payload '{"action":"restart"}'

# 发送事件
./bin/quic-ctl send -client client-001 -type event -payload '{"event":"update"}'

# 发送查询
./bin/quic-ctl send -client client-001 -type query -payload '{"query":"status"}'
```

### 广播消息

```bash
# 广播事件
./bin/quic-ctl broadcast -type event -payload '{"event":"update","version":"1.2.0"}'

# 广播命令（谨慎使用）
./bin/quic-ctl broadcast -type command -payload '{"action":"refresh"}'
```

### 参数速查

#### list 命令
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-api` | `http://localhost:8475` | API 地址 |

#### send 命令
| 参数 | 默认值 | 必需 | 说明 |
|------|--------|------|------|
| `-api` | `http://localhost:8475` | 否 | API 地址 |
| `-client` | - | **是** | 客户端 ID |
| `-type` | `command` | 否 | 消息类型 |
| `-payload` | - | **是** | JSON 内容 |
| `-wait-ack` | `false` | 否 | 等待确认 |

#### broadcast 命令
| 参数 | 默认值 | 必需 | 说明 |
|------|--------|------|------|
| `-api` | `http://localhost:8475` | 否 | API 地址 |
| `-type` | `event` | 否 | 消息类型 |
| `-payload` | - | **是** | JSON 内容 |

---

## 消息类型

| 类型 | 用途 | 示例 |
|------|------|------|
| `command` | 执行操作 | `{"action":"restart"}` |
| `event` | 通知事件 | `{"event":"update"}` |
| `query` | 请求信息 | `{"query":"status"}` |
| `response` | 回复请求 | `{"status":"ok"}` |

---

## 常用场景

### 开发环境

```bash
# 1. 启动服务器
./bin/quic-server

# 2. 启动客户端（跳过证书验证）
./bin/quic-client -id dev-client

# 3. 查看客户端
./bin/quic-ctl list

# 4. 发送测试消息
./bin/quic-ctl send -client dev-client -payload '{"test":"hello"}'
```

### 生产环境

```bash
# 1. 启动服务器（使用生产证书）
./bin/quic-server \
  -cert /etc/ssl/quic/cert.pem \
  -key /etc/ssl/quic/key.pem \
  -addr 0.0.0.0:8474 \
  -api 127.0.0.1:8475

# 2. 启动客户端（启用证书验证）
./bin/quic-client \
  -server prod.example.com:8474 \
  -id prod-client-001 \
  -insecure=false

# 3. 管理客户端
./bin/quic-ctl list
./bin/quic-ctl send -client prod-client-001 -type command -payload '{"action":"backup"}'
```

### 批量操作

```bash
# 向所有客户端广播更新通知
./bin/quic-ctl broadcast -type event -payload '{"event":"update_available","version":"1.2.0"}'

# 向特定客户端发送重启命令
for client in client-001 client-002 client-003; do
    ./bin/quic-ctl send -client $client -type command -payload '{"action":"restart"}'
    sleep 1
done
```

---

## HTTP API 快速参考

### 获取客户端列表

```bash
curl http://localhost:8475/api/clients
```

### 获取单个客户端信息

```bash
curl http://localhost:8475/api/clients/client-001
```

### 发送消息

```bash
curl -X POST http://localhost:8475/api/send \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "type": "command",
    "payload": "{\"action\":\"restart\"}"
  }'
```

### 广播消息

```bash
curl -X POST http://localhost:8475/api/broadcast \
  -H "Content-Type: application/json" \
  -d '{
    "type": "event",
    "payload": "{\"event\":\"update_available\"}"
  }'
```

### 健康检查

```bash
curl http://localhost:8475/health
```

---

## 故障排查

### 服务器无法启动

```bash
# 检查端口是否被占用
lsof -i :8474
netstat -tuln | grep 8474

# 检查证书文件
ls -l certs/
openssl x509 -in certs/server-cert.pem -text -noout
```

### 客户端连接失败

```bash
# 测试网络连通性
ping server-hostname
telnet server-hostname 8474

# 检查服务器是否运行
./bin/quic-ctl list -api http://server-hostname:8475
```

### API 无法访问

```bash
# 检查 API 端口
curl http://localhost:8475/health

# 检查服务器日志
tail -f /tmp/quic-server.log
```

---

## 配置文件示例

### systemd 服务配置

**服务器** (`/etc/systemd/system/quic-server.service`):

```ini
[Unit]
Description=QUIC Backbone Server
After=network.target

[Service]
Type=simple
User=quic
WorkingDirectory=/opt/quic-backbone
ExecStart=/opt/quic-backbone/bin/quic-server \
  -cert /etc/quic/server-cert.pem \
  -key /etc/quic/server-key.pem \
  -addr :8474 \
  -api 127.0.0.1:8475
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**客户端** (`/etc/systemd/system/quic-client.service`):

```ini
[Unit]
Description=QUIC Backbone Client
After=network.target

[Service]
Type=simple
User=quic
ExecStart=/opt/quic-backbone/bin/quic-client \
  -server server.example.com:8474 \
  -id %H \
  -insecure=false
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### 启动服务

```bash
# 启用并启动服务器
sudo systemctl enable quic-server
sudo systemctl start quic-server
sudo systemctl status quic-server

# 启用并启动客户端
sudo systemctl enable quic-client
sudo systemctl start quic-client
sudo systemctl status quic-client

# 查看日志
sudo journalctl -u quic-server -f
sudo journalctl -u quic-client -f
```

---

## 性能调优速查

### 高并发场景

通过代码配置：

```go
config.MaxClients = 50000
config.MaxIncomingStreams = 5000
config.MaxPromises = 100000
```

### 低延迟场景

```go
config.HeartbeatInterval = 5 * time.Second
config.DefaultMessageTimeout = 10 * time.Second
```

### 不稳定网络

```go
config.ReconnectEnabled = true
config.InitialBackoff = 3 * time.Second
config.MaxBackoff = 180 * time.Second
```

---

## 相关文档

- 📖 [完整配置指南](configuration-guide.md)
- 🔧 [CLI 使用指南](cli-guide.md)
- 🌐 [HTTP API 文档](http-api.md)
- 🚀 [快速开始](../quickstart.md)

---

**提示**: 使用 `./bin/quic-ctl help` 查看完整帮助信息
