#!/bin/bash
# QUIC Backbone 重连功能测试脚本

set -e

echo "========================================="
echo "QUIC Backbone 重连功能测试"
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
if [ ! -f "bin/quic-server" ] || [ ! -f "bin/quic-client" ]; then
    echo "❌ 错误：找不到二进制文件"
    echo "请运行: make build"
    exit 1
fi

echo "✅ 二进制文件存在"
echo ""

# 启动服务器（后台）
echo "🚀 启动服务器..."
./bin/quic-server -addr :8474 > /tmp/quic-reconnect-server.log 2>&1 &
SERVER_PID=$!
echo "   服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 2

# 检查服务器是否还在运行
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ 服务器启动失败"
    cat /tmp/quic-reconnect-server.log
    exit 1
fi

echo "✅ 服务器已启动"
echo ""

# 启动客户端（后台）
echo "🚀 启动客户端..."
./bin/quic-client -server localhost:8474 -id client-reconnect-test > /tmp/quic-reconnect-client.log 2>&1 &
CLIENT_PID=$!
echo "   客户端 PID: $CLIENT_PID"

# 等待连接建立
sleep 3

echo ""
echo "========================================="
echo "测试场景 1: 初始连接"
echo "========================================="

# 检查客户端是否成功连接
if grep -q "Connected to server" /tmp/quic-reconnect-client.log; then
    echo "✅ 客户端成功连接到服务器"
else
    echo "❌ 客户端连接失败"
    echo "客户端日志："
    cat /tmp/quic-reconnect-client.log
    kill $CLIENT_PID $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo ""
echo "========================================="
echo "测试场景 2: 服务器断开，测试重连"
echo "========================================="

echo "🔪 杀死服务器进程..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true

echo "   等待 5 秒，让客户端检测到断开..."
sleep 5

# 检查客户端是否检测到断开
if grep -q "Connection lost\|Disconnected from server\|Heartbeat failed" /tmp/quic-reconnect-client.log; then
    echo "✅ 客户端检测到连接断开"
else
    echo "⚠️  客户端可能未检测到断开（继续测试）"
fi

echo ""
echo "🚀 重新启动服务器..."
./bin/quic-server -addr :8474 >> /tmp/quic-reconnect-server.log 2>&1 &
SERVER_PID=$!
echo "   新服务器 PID: $SERVER_PID"

# 等待服务器启动和客户端重连
echo "   等待 15 秒，让客户端重连..."
sleep 15

echo ""
echo "========================================="
echo "测试场景 3: 验证重连成功"
echo "========================================="

# 检查客户端日志中的重连记录
if grep -q "Reconnected successfully\|🔄 Reconnected to server" /tmp/quic-reconnect-client.log; then
    echo "✅ 客户端成功重连到服务器"
    RECONNECT_SUCCESS=true
else
    echo "❌ 客户端未能重连到服务器"
    RECONNECT_SUCCESS=false
fi

# 检查重连尝试次数
RECONNECT_ATTEMPTS=$(grep -c "Reconnect failed\|attempting to reconnect" /tmp/quic-reconnect-client.log || echo "0")
echo "   重连尝试次数: $RECONNECT_ATTEMPTS"

# 检查服务器是否接受了重连
if grep -q "Client connected.*client-reconnect-test" /tmp/quic-reconnect-server.log | tail -1; then
    echo "✅ 服务器接受了客户端重连"
else
    echo "⚠️  服务器日志中未找到重连记录"
fi

echo ""
echo "========================================="
echo "测试场景 4: 清理"
echo "========================================="

# 停止客户端
kill $CLIENT_PID 2>/dev/null || true

# 停止服务器
kill $SERVER_PID 2>/dev/null || true

sleep 2

echo "✅ 所有进程已停止"
echo ""

echo "========================================="
echo "日志文件位置："
echo "  服务器: /tmp/quic-reconnect-server.log"
echo "  客户端: /tmp/quic-reconnect-client.log"
echo ""
echo "查看完整日志："
echo "  tail -f /tmp/quic-reconnect-client.log"
echo "========================================="
echo ""

if [ "$RECONNECT_SUCCESS" = true ]; then
    echo "🎉 重连功能测试通过！"
    exit 0
else
    echo "❌ 重连功能测试失败"
    echo ""
    echo "客户端日志最后 50 行："
    tail -50 /tmp/quic-reconnect-client.log
    exit 1
fi
