#!/bin/bash

# 命令下发功能测试脚本

set -e

echo "==================================="
echo "QUIC 命令下发和回调功能测试"
echo "==================================="
echo ""

# 配置
SERVER_HOST="localhost:8080"
CLIENT_ID="client-001"

echo "1. 下发重启命令"
echo "-----------------------------------"
RESTART_RESPONSE=$(curl -s -X POST "http://${SERVER_HOST}/api/command" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"${CLIENT_ID}\",
    \"command_type\": \"restart\",
    \"payload\": {\"delay_seconds\": 2},
    \"timeout\": 30
  }")

echo "响应: ${RESTART_RESPONSE}"
COMMAND_ID=$(echo ${RESTART_RESPONSE} | jq -r '.command_id')
echo "命令ID: ${COMMAND_ID}"
echo ""

echo "2. 等待命令执行 (3秒)..."
sleep 3
echo ""

echo "3. 查询命令状态"
echo "-----------------------------------"
STATUS_RESPONSE=$(curl -s "http://${SERVER_HOST}/api/command/${COMMAND_ID}")
echo "响应: ${STATUS_RESPONSE}" | jq '.'
COMMAND_STATUS=$(echo ${STATUS_RESPONSE} | jq -r '.command.status')
echo "命令状态: ${COMMAND_STATUS}"
echo ""

echo "4. 下发配置更新命令"
echo "-----------------------------------"
UPDATE_RESPONSE=$(curl -s -X POST "http://${SERVER_HOST}/api/command" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"${CLIENT_ID}\",
    \"command_type\": \"update_config\",
    \"payload\": {
      \"config\": {
        \"log_level\": \"debug\",
        \"timeout\": 60
      }
    },
    \"timeout\": 30
  }")

echo "响应: ${UPDATE_RESPONSE}"
UPDATE_COMMAND_ID=$(echo ${UPDATE_RESPONSE} | jq -r '.command_id')
echo "命令ID: ${UPDATE_COMMAND_ID}"
echo ""

echo "5. 下发获取状态命令"
echo "-----------------------------------"
STATUS_CMD_RESPONSE=$(curl -s -X POST "http://${SERVER_HOST}/api/command" \
  -H "Content-Type: application/json" \
  -d "{
    \"client_id\": \"${CLIENT_ID}\",
    \"command_type\": \"get_status\",
    \"payload\": {},
    \"timeout\": 30
  }")

echo "响应: ${STATUS_CMD_RESPONSE}"
STATUS_CMD_ID=$(echo ${STATUS_CMD_RESPONSE} | jq -r '.command_id')
echo "命令ID: ${STATUS_CMD_ID}"
echo ""

echo "6. 等待命令执行 (2秒)..."
sleep 2
echo ""

echo "7. 查询所有命令"
echo "-----------------------------------"
ALL_COMMANDS=$(curl -s "http://${SERVER_HOST}/api/commands?client_id=${CLIENT_ID}")
echo "响应: ${ALL_COMMANDS}" | jq '.'
echo ""

echo "8. 查询已完成的命令"
echo "-----------------------------------"
COMPLETED_COMMANDS=$(curl -s "http://${SERVER_HOST}/api/commands?client_id=${CLIENT_ID}&status=completed")
echo "响应: ${COMPLETED_COMMANDS}" | jq '.'
echo ""

echo "==================================="
echo "测试完成！"
echo "==================================="
echo ""
echo "总结:"
echo "  - 共下发 3 个命令"
echo "  - restart 命令: ${COMMAND_ID} (${COMMAND_STATUS})"
echo "  - update_config 命令: ${UPDATE_COMMAND_ID}"
echo "  - get_status 命令: ${STATUS_CMD_ID}"
echo ""
echo "可以通过以下命令查看详细状态:"
echo "  curl http://${SERVER_HOST}/api/command/${COMMAND_ID} | jq '.'"
echo ""
