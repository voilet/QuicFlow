#!/bin/bash
# QUIC Backbone CLI 功能测试脚本

set -e

echo "========================================="
echo "QUIC Backbone CLI 功能测试"
echo "========================================="
echo ""

# 检查证书
if [ ! -f "certs/server-cert.pem" ] || [ ! -f "certs/server-key.pem" ]; then
    echo "❌ 错误：找不到 TLS 证书"
    echo "请运行: make certs"
    exit 1
fi

echo "✅ TLS 证书存在"
echo ""

# 检查二进制文件
if [ ! -f "bin/quic-server" ] || [ ! -f "bin/quic-client" ] || [ ! -f "bin/quic-ctl" ]; then
    echo "❌ 错误：找不到二进制文件"
    echo "请运行: make build"
    exit 1
fi

echo "✅ 二进制文件存在"
echo ""

# 启动服务器（后台）
echo "🚀 启动服务器..."
./bin/quic-server -addr :8474 -api :8475 > /tmp/quic-cli-server.log 2>&1 &
SERVER_PID=$!
echo "   服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 2

# 检查服务器是否还在运行
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ 服务器启动失败"
    cat /tmp/quic-cli-server.log
    exit 1
fi

echo "✅ 服务器已启动"
echo ""

# 启动客户端 1
echo "🚀 启动客户端 1..."
./bin/quic-client -server localhost:8474 -id client-001 > /tmp/quic-cli-client-1.log 2>&1 &
CLIENT1_PID=$!
echo "   客户端 1 PID: $CLIENT1_PID"

# 等待连接建立
sleep 3

# 启动客户端 2
echo "🚀 启动客户端 2..."
./bin/quic-client -server localhost:8474 -id client-002 > /tmp/quic-cli-client-2.log 2>&1 &
CLIENT2_PID=$!
echo "   客户端 2 PID: $CLIENT2_PID"

# 等待连接建立
sleep 3

# 启动客户端 3
echo "🚀 启动客户端 3..."
./bin/quic-client -server localhost:8474 -id client-003 > /tmp/quic-cli-client-3.log 2>&1 &
CLIENT3_PID=$!
echo "   客户端 3 PID: $CLIENT3_PID"

# 等待连接建立
sleep 3

echo ""
echo "========================================="
echo "测试场景 1: 查询客户端列表"
echo "========================================="

./bin/quic-ctl list

echo ""
echo "========================================="
echo "测试场景 2: 发送消息到指定客户端"
echo "========================================="

./bin/quic-ctl send -client client-001 -type command -payload '{"action":"restart","timeout":30}'

echo ""
echo "========================================="
echo "测试场景 3: 广播消息到所有客户端"
echo "========================================="

./bin/quic-ctl broadcast -type event -payload '{"event":"update_available","version":"1.2.0"}'

echo ""
echo "========================================="
echo "测试场景 4: 发送不同类型的消息"
echo "========================================="

echo "发送 query 消息..."
./bin/quic-ctl send -client client-002 -type query -payload '{"query":"status"}'

echo ""
echo "发送 event 消息..."
./bin/quic-ctl send -client client-003 -type event -payload '{"event":"config_changed"}'

echo ""
echo "========================================="
echo "测试场景 5: 验证消息接收"
echo "========================================="

echo "等待 5 秒让消息处理..."
sleep 5

# 检查客户端日志
echo ""
echo "客户端 1 接收到的消息："
grep "Data message received" /tmp/quic-cli-client-1.log || echo "  (无消息)"

echo ""
echo "客户端 2 接收到的消息："
grep "Data message received" /tmp/quic-cli-client-2.log || echo "  (无消息)"

echo ""
echo "客户端 3 接收到的消息："
grep "Data message received" /tmp/quic-cli-client-3.log || echo "  (无消息)"

echo ""
echo "========================================="
echo "测试场景 6: 清理"
echo "========================================="

# 停止所有客户端
kill $CLIENT1_PID $CLIENT2_PID $CLIENT3_PID 2>/dev/null || true

# 停止服务器
kill $SERVER_PID 2>/dev/null || true

sleep 2

echo "✅ 所有进程已停止"
echo ""

echo "========================================="
echo "日志文件位置："
echo "  服务器: /tmp/quic-cli-server.log"
echo "  客户端 1: /tmp/quic-cli-client-1.log"
echo "  客户端 2: /tmp/quic-cli-client-2.log"
echo "  客户端 3: /tmp/quic-cli-client-3.log"
echo "========================================="
echo ""

echo "🎉 CLI 功能测试完成！"
