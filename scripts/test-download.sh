#!/bin/bash
# 文件下载功能测试脚本
# 用于验证 Web 端和 CLI 下载功能

set -e

API_BASE="http://localhost:8475/api/file"
TEST_FILE="/tmp/test-download-$(date +%s).txt"
DOWNLOAD_FILE="/tmp/downloaded-$(date +%s).txt"

echo "========================================="
echo "文件下载功能测试"
echo "========================================="
echo ""

# 检查服务器是否运行
if ! curl -s "$API_BASE/config" > /dev/null 2>&1; then
    echo "❌ 错误：服务器未运行或无法访问"
    echo "请确保服务器在 http://localhost:8475 运行"
    exit 1
fi

echo "✅ 服务器运行正常"
echo ""

# 创建测试文件
echo "========================================="
echo "步骤 1: 创建测试文件"
echo "========================================="
echo "这是一个测试文件，用于验证下载功能。" > "$TEST_FILE"
echo "创建时间: $(date)" >> "$TEST_FILE"
echo "文件大小: $(wc -c < "$TEST_FILE") 字节" >> "$TEST_FILE"
TEST_FILE_SIZE=$(wc -c < "$TEST_FILE")
echo "测试文件: $TEST_FILE"
echo "文件大小: $TEST_FILE_SIZE 字节"
echo ""

# 步骤 1: 初始化上传
echo "========================================="
echo "步骤 2: 初始化上传"
echo "========================================="
INIT_RESPONSE=$(curl -s -X POST "$API_BASE/upload/init" \
  -H "Content-Type: application/json" \
  -d "{
    \"filename\": \"$(basename "$TEST_FILE")\",
    \"file_size\": $TEST_FILE_SIZE,
    \"path\": \"/test/\"
  }")

echo "响应: $INIT_RESPONSE"

# 提取 task_id
TASK_ID=$(echo "$INIT_RESPONSE" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TASK_ID" ]; then
    echo "❌ 无法获取 task_id"
    echo "$INIT_RESPONSE"
    exit 1
fi

echo "✅ 任务ID: $TASK_ID"
echo ""

# 步骤 2: 上传文件（单次上传整个文件）
echo "========================================="
echo "步骤 3: 上传文件内容"
echo "========================================="
UPLOAD_RESPONSE=$(curl -s -X POST "$API_BASE/upload/chunk?task_id=$TASK_ID&offset=0&sequence=0" \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@$TEST_FILE")

echo "响应: $UPLOAD_RESPONSE"
echo ""

# 步骤 3: 完成上传
echo "========================================="
echo "步骤 4: 完成上传"
echo "========================================="
COMPLETE_RESPONSE=$(curl -s -X POST "$API_BASE/upload/complete" \
  -H "Content-Type: application/json" \
  -d "{
    \"task_id\": \"$TASK_ID\"
  }")

echo "响应: $COMPLETE_RESPONSE"
echo ""

# 提取 file_id (如果存在)
FILE_ID=$(echo "$COMPLETE_RESPONSE" | grep -o '"file_id":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -n "$FILE_ID" ]; then
    echo "✅ 文件ID: $FILE_ID"
fi
echo ""

# 步骤 4: 请求下载
echo "========================================="
echo "步骤 5: 请求下载"
echo "========================================="
DOWNLOAD_REQUEST_RESPONSE=$(curl -s -X POST "$API_BASE/download/request" \
  -H "Content-Type: application/json" \
  -d "{
    \"file_path\": \"/test/$(basename "$TEST_FILE")\"
  }")

echo "响应: $DOWNLOAD_REQUEST_RESPONSE"

# 提取下载任务ID
DOWNLOAD_TASK_ID=$(echo "$DOWNLOAD_REQUEST_RESPONSE" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$DOWNLOAD_TASK_ID" ]; then
    echo "❌ 无法获取下载任务ID"
    echo "$DOWNLOAD_REQUEST_RESPONSE"
    exit 1
fi

echo "✅ 下载任务ID: $DOWNLOAD_TASK_ID"
echo ""

# 步骤 5: 下载文件
echo "========================================="
echo "步骤 6: 下载文件"
echo "========================================="
HTTP_CODE=$(curl -s -X GET "$API_BASE/download/$DOWNLOAD_TASK_ID" \
  -o "$DOWNLOAD_FILE" \
  -w "%{http_code}")

if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ 下载失败，HTTP状态码: $HTTP_CODE"
    exit 1
fi

DOWNLOADED_SIZE=$(wc -c < "$DOWNLOAD_FILE")
echo "✅ 下载完成"
echo "下载文件: $DOWNLOAD_FILE"
echo "下载大小: $DOWNLOADED_SIZE 字节"
echo ""

# 步骤 6: 验证文件
echo "========================================="
echo "步骤 7: 验证文件内容"
echo "========================================="

if [ "$TEST_FILE_SIZE" != "$DOWNLOADED_SIZE" ]; then
    echo "❌ 文件大小不匹配"
    echo "原始大小: $TEST_FILE_SIZE"
    echo "下载大小: $DOWNLOADED_SIZE"
    exit 1
fi

# 比较文件内容
if ! cmp -s "$TEST_FILE" "$DOWNLOAD_FILE"; then
    echo "❌ 文件内容不匹配"
    echo "原始文件:"
    cat "$TEST_FILE"
    echo ""
    echo "下载文件:"
    cat "$DOWNLOAD_FILE"
    exit 1
fi

echo "✅ 文件验证成功"
echo ""

# 显示文件内容
echo "文件内容:"
echo "----------------------------------------"
cat "$DOWNLOAD_FILE"
echo "----------------------------------------"
echo ""

# 获取传输历史
echo "========================================="
echo "步骤 8: 查询传输历史"
echo "========================================="
TRANSFERS=$(curl -s -X GET "$API_BASE/transfers?limit=5")
echo "最近的传输记录:"
echo "$TRANSFERS" | grep -o '"file_name":"[^"]*"' | head -5
echo ""

# 清理
echo "========================================="
echo "清理"
echo "========================================="
rm -f "$TEST_FILE" "$DOWNLOAD_FILE"
echo "✅ 临时文件已清理"
echo ""

echo "🎉 下载功能测试完成！"
echo ""
echo "========================================="
echo "测试总结"
echo "========================================="
echo "✅ 文件上传: 成功"
echo "✅ 下载请求: 成功"
echo "✅ 文件下载: 成功"
echo "✅ 内容验证: 通过"
echo "========================================="
