#!/bin/bash

# SSH over QUIC 测试脚本

set -e

echo "=== SSH over QUIC 测试 ==="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

API_ADDR="http://localhost:8080"
CLIENT_ID=""

# 等待服务就绪
wait_for_server() {
    echo -n "等待服务器启动..."
    for i in {1..30}; do
        if curl -s "${API_ADDR}/api/clients" > /dev/null 2>&1; then
            echo -e " ${GREEN}就绪${NC}"
            return 0
        fi
        sleep 1
        echo -n "."
    done
    echo -e " ${RED}超时${NC}"
    return 1
}

# 获取第一个连接的客户端
get_client_id() {
    CLIENT_ID=$(curl -s "${API_ADDR}/api/clients" | jq -r '.clients[0].client_id // empty')
    if [ -z "$CLIENT_ID" ]; then
        echo -e "${YELLOW}没有已连接的客户端${NC}"
        return 1
    fi
    echo -e "客户端 ID: ${GREEN}${CLIENT_ID}${NC}"
    return 0
}

# 测试一次性 SSH 命令执行
test_ssh_oneshot() {
    echo ""
    echo "=== 测试一次性 SSH 命令执行 ==="

    RESPONSE=$(curl -s -X POST "${API_ADDR}/api/ssh/exec-oneshot" \
        -H "Content-Type: application/json" \
        -d "{
            \"client_id\": \"${CLIENT_ID}\",
            \"user\": \"admin\",
            \"password\": \"admin123\",
            \"command\": \"uname -a\"
        }")

    SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        OUTPUT=$(echo "$RESPONSE" | jq -r '.output')
        echo -e "${GREEN}成功${NC}"
        echo "命令输出: $OUTPUT"
        return 0
    else
        ERROR=$(echo "$RESPONSE" | jq -r '.error')
        echo -e "${RED}失败: $ERROR${NC}"
        return 1
    fi
}

# 测试建立持久 SSH 连接
test_ssh_connect() {
    echo ""
    echo "=== 测试建立持久 SSH 连接 ==="

    RESPONSE=$(curl -s -X POST "${API_ADDR}/api/ssh/connect" \
        -H "Content-Type: application/json" \
        -d "{
            \"client_id\": \"${CLIENT_ID}\",
            \"user\": \"admin\",
            \"password\": \"admin123\"
        }")

    SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        echo -e "${GREEN}连接成功${NC}"
        return 0
    else
        ERROR=$(echo "$RESPONSE" | jq -r '.error')
        echo -e "${RED}失败: $ERROR${NC}"
        return 1
    fi
}

# 测试在持久连接上执行命令
test_ssh_exec() {
    echo ""
    echo "=== 测试在持久连接上执行命令 ==="

    for cmd in "whoami" "hostname" "pwd" "ls -la /tmp"; do
        echo -n "执行: $cmd ... "

        RESPONSE=$(curl -s -X POST "${API_ADDR}/api/ssh/exec/${CLIENT_ID}" \
            -H "Content-Type: application/json" \
            -d "{\"command\": \"${cmd}\"}")

        SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
        if [ "$SUCCESS" = "true" ]; then
            OUTPUT=$(echo "$RESPONSE" | jq -r '.output')
            echo -e "${GREEN}成功${NC}"
            echo "  输出: $(echo "$OUTPUT" | head -1)"
        else
            ERROR=$(echo "$RESPONSE" | jq -r '.error')
            echo -e "${RED}失败: $ERROR${NC}"
        fi
    done
}

# 测试列出 SSH 连接
test_ssh_list() {
    echo ""
    echo "=== 测试列出 SSH 连接 ==="

    RESPONSE=$(curl -s "${API_ADDR}/api/ssh/connections")

    SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        COUNT=$(echo "$RESPONSE" | jq -r '.count')
        echo -e "${GREEN}成功${NC}, 当前连接数: $COUNT"
        echo "$RESPONSE" | jq -r '.connections[]'
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 测试检查 SSH 连接状态
test_ssh_status() {
    echo ""
    echo "=== 测试检查 SSH 连接状态 ==="

    RESPONSE=$(curl -s "${API_ADDR}/api/ssh/status/${CLIENT_ID}")

    SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
    CONNECTED=$(echo "$RESPONSE" | jq -r '.connected')

    if [ "$SUCCESS" = "true" ]; then
        echo -e "${GREEN}成功${NC}, 连接状态: $CONNECTED"
        return 0
    else
        echo -e "${RED}失败${NC}"
        return 1
    fi
}

# 测试断开 SSH 连接
test_ssh_disconnect() {
    echo ""
    echo "=== 测试断开 SSH 连接 ==="

    RESPONSE=$(curl -s -X POST "${API_ADDR}/api/ssh/disconnect/${CLIENT_ID}")

    SUCCESS=$(echo "$RESPONSE" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        echo -e "${GREEN}断开成功${NC}"
        return 0
    else
        ERROR=$(echo "$RESPONSE" | jq -r '.error')
        echo -e "${RED}失败: $ERROR${NC}"
        return 1
    fi
}

# 主测试流程
main() {
    echo "API 地址: ${API_ADDR}"
    echo ""

    # 等待服务就绪
    if ! wait_for_server; then
        echo -e "${RED}服务器未就绪，请先启动服务器和客户端${NC}"
        echo ""
        echo "启动方式："
        echo "  1. 启动服务器: ./bin/quic-server"
        echo "  2. 启动客户端: ./bin/quic-client --ssh"
        exit 1
    fi

    # 获取客户端 ID
    if ! get_client_id; then
        echo -e "${RED}请先启动客户端连接${NC}"
        echo ""
        echo "启动客户端: ./bin/quic-client --ssh"
        exit 1
    fi

    # 运行测试
    PASSED=0
    FAILED=0

    echo ""
    echo "开始测试..."

    # 一次性执行测试
    if test_ssh_oneshot; then
        ((PASSED++))
    else
        ((FAILED++))
    fi

    # 持久连接测试
    if test_ssh_connect; then
        ((PASSED++))

        # 在持久连接上执行命令
        test_ssh_exec

        # 列出连接
        if test_ssh_list; then
            ((PASSED++))
        else
            ((FAILED++))
        fi

        # 检查状态
        if test_ssh_status; then
            ((PASSED++))
        else
            ((FAILED++))
        fi

        # 断开连接
        if test_ssh_disconnect; then
            ((PASSED++))
        else
            ((FAILED++))
        fi
    else
        ((FAILED++))
    fi

    # 测试结果
    echo ""
    echo "=== 测试结果 ==="
    echo -e "通过: ${GREEN}${PASSED}${NC}"
    echo -e "失败: ${RED}${FAILED}${NC}"

    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}所有测试通过！${NC}"
        exit 0
    else
        echo -e "${RED}部分测试失败${NC}"
        exit 1
    fi
}

main
