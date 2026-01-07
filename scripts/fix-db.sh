#!/bin/bash

# 数据库排查和修复脚本
# 用法: ./fix-db.sh [host] [port] [user] [password] [dbname]

set -e

HOST="${1:-192.168.110.104}"
PORT="${2:-15432}"
USER="${3:-postgres}"
PASSWORD="${4:-}"
DBNAME="${5:-quic_release}"

echo "=========================================="
echo "数据库连接排查和修复工具"
echo "=========================================="
echo "主机: $HOST:$PORT"
echo "用户: $USER"
echo "数据库: $DBNAME"
echo ""

# 如果没有提供密码，尝试从环境变量获取
if [ -z "$PASSWORD" ]; then
    PASSWORD="${DB_PASSWORD:-}"
fi

# 如果还是没有密码，提示输入
if [ -z "$PASSWORD" ]; then
    echo -n "请输入数据库密码: "
    read -s PASSWORD
    echo ""
fi

# 设置 PGPASSWORD 环境变量
export PGPASSWORD="$PASSWORD"

echo ""
echo "步骤 1: 测试连接到 PostgreSQL 服务器..."
if psql -h "$HOST" -p "$PORT" -U "$USER" -d postgres -c "\q" 2>/dev/null; then
    echo "✅ 成功连接到 PostgreSQL 服务器"
else
    echo "❌ 无法连接到 PostgreSQL 服务器"
    echo "请检查:"
    echo "  1. 主机地址和端口是否正确"
    echo "  2. 用户名和密码是否正确"
    echo "  3. 网络连接是否正常"
    echo "  4. PostgreSQL 服务是否运行"
    exit 1
fi

echo ""
echo "步骤 2: 列出所有数据库..."
psql -h "$HOST" -p "$PORT" -U "$USER" -d postgres -c "\l" 2>/dev/null | grep -E "^ [a-zA-Z]" | awk '{print "  - " $1}'

echo ""
echo "步骤 3: 检查数据库 '$DBNAME' 是否存在..."
if psql -h "$HOST" -p "$PORT" -U "$USER" -d postgres -lqt 2>/dev/null | cut -d \| -f 1 | grep -qw "$DBNAME"; then
    echo "✅ 数据库 '$DBNAME' 已存在"
    DB_EXISTS=true
else
    echo "❌ 数据库 '$DBNAME' 不存在"
    DB_EXISTS=false
fi

if [ "$DB_EXISTS" = false ]; then
    echo ""
    echo "步骤 4: 创建数据库 '$DBNAME'..."
    if psql -h "$HOST" -p "$PORT" -U "$USER" -d postgres -c "CREATE DATABASE \"$DBNAME\" WITH ENCODING 'UTF8';" 2>/dev/null; then
        echo "✅ 数据库 '$DBNAME' 创建成功"
    else
        echo "❌ 创建数据库失败"
        echo "可能的原因:"
        echo "  1. 用户没有创建数据库的权限"
        echo "  2. 数据库名称已存在但列表查询失败"
        exit 1
    fi
fi

echo ""
echo "步骤 5: 测试连接到数据库 '$DBNAME'..."
if psql -h "$HOST" -p "$PORT" -U "$USER" -d "$DBNAME" -c "\q" 2>/dev/null; then
    echo "✅ 成功连接到数据库 '$DBNAME'"
else
    echo "❌ 无法连接到数据库 '$DBNAME'"
    exit 1
fi

echo ""
echo "=========================================="
echo "✅ 所有检查通过！数据库已准备就绪。"
echo "=========================================="

# 清理环境变量
unset PGPASSWORD

