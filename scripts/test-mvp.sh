#!/bin/bash
# QUIC Backbone MVP 测试脚本

set -e

echo "========================================="
echo "QUIC Backbone MVP 功能测试"
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
./bin/quic-server -addr :8474 > /tmp/quic-server.log 2>&1 &
SERVER_PID=$!
echo "   服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 2

# 检查服务器是否还在运行
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ 服务器启动失败"
    cat /tmp/quic-server.log
    exit 1
fi

echo "✅ 服务器已启动"
echo ""

# 启动客户端 1
echo "🚀 启动客户端 1..."
./bin/quic-client -server localhost:8474 -id client-001 > /tmp/quic-client-1.log 2>&1 &
CLIENT1_PID=$!
echo "   客户端 1 PID: $CLIENT1_PID"

# 等待连接建立
sleep 3

# 启动客户端 2
echo "🚀 启动客户端 2..."
./bin/quic-client -server localhost:8474 -id client-002 > /tmp/quic-client-2.log 2>&1 &
CLIENT2_PID=$!
echo "   客户端 2 PID: $CLIENT2_PID"

# 等待连接建立
sleep 3

echo ""
echo "========================================="
echo "测试场景 1: 连接建立"
echo "========================================="

# 检查服务器日志
if grep -q "Client connected" /tmp/quic-server.log; then
    echo "✅ 服务器成功接受客户端连接"
else
    echo "❌ 服务器未接受客户端连接"
fi

# 检查客户端日志
if grep -q "Connected to server" /tmp/quic-client-1.log; then
    echo "✅ 客户端 1 成功连接到服务器"
else
    echo "❌ 客户端 1 连接失败"
fi

if grep -q "Connected to server" /tmp/quic-client-2.log; then
    echo "✅ 客户端 2 成功连接到服务器"
else
    echo "❌ 客户端 2 连接失败"
fi

echo ""
echo "========================================="
echo "测试场景 2: 心跳机制"
echo "========================================="
echo "等待 20 秒以观察心跳..."
sleep 20

# 检查心跳日志
if grep -q "Pong" /tmp/quic-server.log; then
    echo "✅ 服务器成功处理心跳"
else
    echo "⚠️  服务器心跳日志未找到（可能日志级别为 INFO，心跳在 DEBUG 级别）"
fi

if grep -q "Pong received" /tmp/quic-client-1.log; then
    echo "✅ 客户端 1 成功接收心跳响应"
else
    echo "⚠️  客户端 1 心跳日志未找到（可能日志级别为 INFO）"
fi

echo ""
echo "========================================="
echo "测试场景 3: 客户端断开"
echo "========================================="
echo "断开客户端 1..."
kill $CLIENT1_PID 2>/dev/null || true
sleep 2

if grep -q "Client disconnected.*client-001" /tmp/quic-server.log; then
    echo "✅ 服务器检测到客户端 1 断开"
else
    echo "⚠️  服务器未检测到客户端断开（可能需要等待心跳超时）"
fi

echo ""
echo "========================================="
echo "测试场景 4: 清理"
echo "========================================="

# 停止客户端 2
kill $CLIENT2_PID 2>/dev/null || true

# 停止服务器
kill $SERVER_PID 2>/dev/null || true

sleep 2

echo "✅ 所有进程已停止"
echo ""

echo "========================================="
echo "日志文件位置："
echo "  服务器: /tmp/quic-server.log"
echo "  客户端 1: /tmp/quic-client-1.log"
echo "  客户端 2: /tmp/quic-client-2.log"
echo "========================================="
echo ""
echo "🎉 MVP 功能测试完成！"
