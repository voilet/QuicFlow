# QUIC Backbone CLI 工具使用指南

## 概述

`quic-ctl` 是 QUIC Backbone 的命令行管理工具，用于查询在线客户端列表和向客户端发送任务消息。

## 功能特性

- ✅ 查询所有在线客户端列表
- ✅ 查询单个客户端详细信息
- ✅ 向指定客户端发送消息（支持多种消息类型）
- ✅ 向所有客户端广播消息
- ✅ 基于 HTTP API 的轻量级架构

## 安装

### 构建

```bash
# 构建 CLI 工具
make build

# 或者手动构建
go build -o bin/quic-ctl ./cmd/ctl
```

### 验证安装

```bash
./bin/quic-ctl help
```

## 服务器配置

服务器默认会在 `:8475` 端口启动 HTTP API 服务。可以通过命令行参数自定义：

```bash
./bin/quic-server -addr :8474 -api :8475
```

## 命令参考

### 1. 列出所有在线客户端

查询当前所有连接到服务器的客户端列表。

```bash
# 使用默认 API 地址
./bin/quic-ctl list

# 指定 API 地址
./bin/quic-ctl list -api http://localhost:8475
```

**输出示例：**

```
Connected Clients: 3

CLIENT ID   REMOTE ADDRESS   UPTIME  CONNECTED AT
---------   --------------   ------  ------------
client-001  127.0.0.1:52104  9s      2025-12-24 16:59:40
client-002  127.0.0.1:58931  6s      2025-12-24 16:59:42
client-003  127.0.0.1:58822  3s      2025-12-24 16:59:45
```

### 2. 向指定客户端发送消息

向特定客户端发送任务消息。

```bash
# 基本用法
./bin/quic-ctl send -client <客户端ID> -payload <JSON消息>

# 指定消息类型
./bin/quic-ctl send -client client-001 -type command -payload '{"action":"restart"}'

# 等待客户端确认
./bin/quic-ctl send -client client-001 -type command -payload '{"action":"restart"}' -wait-ack
```

**参数说明：**

- `-client <id>`: **必需**，目标客户端 ID
- `-type <type>`: 消息类型，可选值：
  - `command`: 命令消息（默认）
  - `event`: 事件消息
  - `query`: 查询消息
  - `response`: 响应消息
- `-payload <json>`: **必需**，消息内容（JSON 格式）
- `-wait-ack`: 是否等待客户端确认（默认：false）
- `-api <addr>`: API 服务器地址（默认：http://localhost:8475）

**输出示例：**

```
✅ Message sent successfully
   Client ID: client-001
   Message ID: 87f03fa3-86ef-4c20-8a8f-96831f8443ab
   Type: command
   Payload: {"action":"restart","timeout":30}
```

### 3. 广播消息到所有客户端

向所有在线客户端广播消息。

```bash
# 基本用法
./bin/quic-ctl broadcast -payload <JSON消息>

# 指定消息类型
./bin/quic-ctl broadcast -type event -payload '{"event":"update_available","version":"1.2.0"}'
```

**参数说明：**

- `-type <type>`: 消息类型（默认：event）
- `-payload <json>`: **必需**，消息内容（JSON 格式）
- `-api <addr>`: API 服务器地址（默认：http://localhost:8475）

**输出示例：**

```
✅ Message broadcast completed
   Message ID: a523ace9-a060-4509-b4f7-43393bb2563e
   Type: event
   Payload: {"event":"update_available","version":"1.2.0"}
   Total Clients: 3
   Success: 3
   Failed: 0
```

## 使用示例

### 示例 1：查询在线客户端并发送重启命令

```bash
# 1. 查看当前在线客户端
./bin/quic-ctl list

# 2. 向特定客户端发送重启命令
./bin/quic-ctl send -client client-001 -type command -payload '{"action":"restart","timeout":30}'
```

### 示例 2：发送配置更新事件

```bash
# 向指定客户端发送配置更新事件
./bin/quic-ctl send -client client-002 -type event -payload '{"event":"config_updated","config_path":"/etc/app/config.json"}'
```

### 示例 3：查询客户端状态

```bash
# 发送查询消息
./bin/quic-ctl send -client client-003 -type query -payload '{"query":"status","fields":["cpu","memory","disk"]}'
```

### 示例 4：广播系统维护通知

```bash
# 向所有客户端广播维护通知
./bin/quic-ctl broadcast -type event -payload '{"event":"maintenance","start_time":"2025-12-25T00:00:00Z","duration":"2h"}'
```

## 消息类型说明

### Command (命令)

用于向客户端发送执行指令。

**示例 Payload：**

```json
{
  "action": "restart",
  "timeout": 30,
  "parameters": {
    "graceful": true
  }
}
```

### Event (事件)

用于通知客户端发生的事件。

**示例 Payload：**

```json
{
  "event": "config_changed",
  "timestamp": 1703404800000,
  "data": {
    "config_version": "1.2.0"
  }
}
```

### Query (查询)

用于向客户端查询信息。

**示例 Payload：**

```json
{
  "query": "system_status",
  "fields": ["cpu", "memory", "network"]
}
```

### Response (响应)

用于响应客户端的请求。

**示例 Payload：**

```json
{
  "request_id": "req-123",
  "status": "success",
  "data": {
    "cpu_usage": 45.2,
    "memory_usage": 62.8
  }
}
```

## HTTP API 接口

CLI 工具通过 HTTP API 与服务器通信。你也可以直接使用 API：

### GET /api/clients

获取所有在线客户端列表。

```bash
curl http://localhost:8475/api/clients
```

### GET /api/clients/:id

获取指定客户端的详细信息。

```bash
curl http://localhost:8475/api/clients/client-001
```

### POST /api/send

向指定客户端发送消息。

```bash
curl -X POST http://localhost:8475/api/send \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-001",
    "type": "command",
    "payload": "{\"action\":\"restart\"}",
    "wait_ack": false
  }'
```

### POST /api/broadcast

广播消息到所有客户端。

```bash
curl -X POST http://localhost:8475/api/broadcast \
  -H "Content-Type: application/json" \
  -d '{
    "type": "event",
    "payload": "{\"event\":\"update_available\"}"
  }'
```

### GET /health

健康检查端点。

```bash
curl http://localhost:8475/health
```

## 故障排查

### 无法连接到 API 服务器

```
Error: failed to connect to API server: dial tcp [::1]:8475: connect: connection refused
```

**解决方法：**

1. 确认服务器正在运行并监听 API 端口
2. 检查防火墙设置
3. 使用 `-api` 参数指定正确的 API 地址

### 客户端不存在

```
Error: API error (status 404): Client not found: client not found
```

**解决方法：**

1. 使用 `list` 命令查看可用的客户端 ID
2. 确认客户端 ID 拼写正确

### 无效的 Payload 格式

```
Error: API error (status 400): Invalid request body: invalid character 'a' looking for beginning of value
```

**解决方法：**

1. 确保 payload 是有效的 JSON 格式
2. 使用单引号包裹 JSON 字符串（在 bash 中）
3. 使用 JSON 验证工具检查格式

## 最佳实践

1. **使用有意义的消息 ID**：虽然系统会自动生成 UUID，但在日志中追踪会更容易

2. **合理使用消息类型**：
   - 使用 `command` 发送需要执行的操作
   - 使用 `event` 通知状态变化
   - 使用 `query` 查询信息
   - 使用 `response` 回复查询

3. **Payload 结构化**：保持 payload 的 JSON 结构清晰，便于客户端解析

4. **错误处理**：检查命令的退出码和输出，处理可能的错误情况

5. **脚本化管理**：将常用操作封装为脚本，方便批量管理

## 示例脚本

### 批量发送命令

```bash
#!/bin/bash
# 向所有客户端发送重启命令

CLIENTS=$(./bin/quic-ctl list | tail -n +4 | awk '{print $1}')

for client in $CLIENTS; do
    echo "Sending restart command to $client..."
    ./bin/quic-ctl send -client "$client" -type command -payload '{"action":"restart"}'
    sleep 1
done
```

### 健康检查脚本

```bash
#!/bin/bash
# 定期检查客户端在线状态

while true; do
    echo "=== $(date) ==="
    ./bin/quic-ctl list
    echo ""
    sleep 60
done
```

## 相关文档

- [快速开始指南](../quickstart.md)
- [服务器配置](../docs/server-config.md)
- [客户端配置](../docs/client-config.md)
- [消息协议](../docs/protocol.md)

---

最后更新：2025-12-24
