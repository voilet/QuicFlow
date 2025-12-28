#!/bin/bash

# QUIC Flow 高并发系统调优脚本
# 目标：支持 10W 客户端连接，5W 并发任务
#
# 使用方法：
#   sudo ./scripts/tune-system.sh          # 应用调优设置
#   sudo ./scripts/tune-system.sh check    # 检查当前设置
#   sudo ./scripts/tune-system.sh persist  # 持久化设置

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 目标配置值
TARGET_FILE_MAX=1000000          # 系统最大文件描述符
TARGET_NR_OPEN=1000000           # 进程最大文件描述符
TARGET_SOMAXCONN=65535           # socket 监听队列
TARGET_NETDEV_MAX_BACKLOG=65536  # 网络设备接收队列
TARGET_UDP_RMEM_MAX=67108864     # UDP 接收缓冲区 (64MB)
TARGET_UDP_WMEM_MAX=67108864     # UDP 发送缓冲区 (64MB)
TARGET_RMEM_MAX=134217728        # 最大接收缓冲区 (128MB)
TARGET_WMEM_MAX=134217728        # 最大发送缓冲区 (128MB)
TARGET_RMEM_DEFAULT=67108864     # 默认接收缓冲区 (64MB)
TARGET_WMEM_DEFAULT=67108864     # 默认发送缓冲区 (64MB)
TARGET_NETDEV_BUDGET=600         # 网络设备处理预算
TARGET_BUSY_POLL=50              # 忙轮询超时 (μs)
TARGET_BUSY_READ=50              # 忙读超时 (μs)

# 检查是否是 root 用户
check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}请使用 root 权限运行此脚本${NC}"
        exit 1
    fi
}

# 获取当前值
get_current() {
    local param=$1
    local file=$2
    if [ -f "$file" ]; then
        cat "$file" 2>/dev/null || echo "N/A"
    else
        sysctl -n "$param" 2>/dev/null || echo "N/A"
    fi
}

# 设置内核参数
set_sysctl() {
    local param=$1
    local value=$2
    local current=$(sysctl -n "$param" 2>/dev/null || echo "0")

    if [ "$current" = "$value" ]; then
        echo -e "  ${GREEN}✓${NC} $param = $value (已设置)"
    else
        sysctl -w "$param=$value" > /dev/null 2>&1
        echo -e "  ${YELLOW}→${NC} $param: $current → $value"
    fi
}

# 检查当前配置
check_config() {
    echo "=== QUIC Flow 系统配置检查 ==="
    echo ""

    echo "📁 文件描述符限制:"
    echo "  系统最大 (file-max): $(get_current 'fs.file-max' '')"
    echo "  进程最大 (nr_open):  $(get_current 'fs.nr_open' '')"
    echo "  当前 ulimit -n:      $(ulimit -n)"
    echo ""

    echo "🔌 网络连接队列:"
    echo "  somaxconn:           $(get_current 'net.core.somaxconn' '')"
    echo "  netdev_max_backlog:  $(get_current 'net.core.netdev_max_backlog' '')"
    echo ""

    echo "📦 UDP 缓冲区 (quic-go 建议 7MB+):"
    echo "  rmem_max:            $(get_current 'net.core.rmem_max' '') bytes"
    echo "  wmem_max:            $(get_current 'net.core.wmem_max' '') bytes"
    echo "  rmem_default:        $(get_current 'net.core.rmem_default' '') bytes"
    echo "  wmem_default:        $(get_current 'net.core.wmem_default' '') bytes"
    echo ""

    echo "⚡ 网络性能:"
    echo "  netdev_budget:       $(get_current 'net.core.netdev_budget' '')"
    echo "  busy_poll:           $(get_current 'net.core.busy_poll' '')"
    echo "  busy_read:           $(get_current 'net.core.busy_read' '')"
    echo ""

    echo "🧮 内存配置:"
    echo "  vm.max_map_count:    $(get_current 'vm.max_map_count' '')"
    echo ""

    # 检查 quic-go 缓冲区警告
    local rmem=$(get_current 'net.core.rmem_max' '')
    if [ "$rmem" != "N/A" ] && [ "$rmem" -lt 7340032 ]; then
        echo -e "${RED}⚠️  警告: rmem_max < 7MB, quic-go 会显示缓冲区警告${NC}"
    fi
}

# 应用调优设置
apply_tuning() {
    echo "=== 应用 QUIC Flow 系统调优 ==="
    echo ""

    echo "📁 调整文件描述符限制..."
    set_sysctl "fs.file-max" "$TARGET_FILE_MAX"
    set_sysctl "fs.nr_open" "$TARGET_NR_OPEN"
    echo ""

    echo "🔌 调整网络连接队列..."
    set_sysctl "net.core.somaxconn" "$TARGET_SOMAXCONN"
    set_sysctl "net.core.netdev_max_backlog" "$TARGET_NETDEV_MAX_BACKLOG"
    echo ""

    echo "📦 调整 UDP/网络缓冲区..."
    set_sysctl "net.core.rmem_max" "$TARGET_RMEM_MAX"
    set_sysctl "net.core.wmem_max" "$TARGET_WMEM_MAX"
    set_sysctl "net.core.rmem_default" "$TARGET_RMEM_DEFAULT"
    set_sysctl "net.core.wmem_default" "$TARGET_WMEM_DEFAULT"
    echo ""

    echo "⚡ 调整网络性能参数..."
    set_sysctl "net.core.netdev_budget" "$TARGET_NETDEV_BUDGET"
    set_sysctl "net.core.busy_poll" "$TARGET_BUSY_POLL"
    set_sysctl "net.core.busy_read" "$TARGET_BUSY_READ"
    echo ""

    echo "🧮 调整内存参数..."
    set_sysctl "vm.max_map_count" "262144"
    echo ""

    echo -e "${GREEN}✅ 调优设置已应用${NC}"
    echo ""
    echo "⚠️  注意: 这些设置在重启后会失效，使用 'persist' 命令持久化"
}

# 持久化设置
persist_settings() {
    local SYSCTL_CONF="/etc/sysctl.d/99-quic-flow.conf"

    echo "=== 持久化 QUIC Flow 系统调优设置 ==="
    echo ""

    cat > "$SYSCTL_CONF" << EOF
# QUIC Flow 高并发优化配置
# 目标：10W 客户端连接，5W 并发任务
# 生成时间: $(date)

# 文件描述符
fs.file-max = $TARGET_FILE_MAX
fs.nr_open = $TARGET_NR_OPEN

# 网络连接队列
net.core.somaxconn = $TARGET_SOMAXCONN
net.core.netdev_max_backlog = $TARGET_NETDEV_MAX_BACKLOG

# UDP/网络缓冲区 (quic-go 需要至少 7MB)
net.core.rmem_max = $TARGET_RMEM_MAX
net.core.wmem_max = $TARGET_WMEM_MAX
net.core.rmem_default = $TARGET_RMEM_DEFAULT
net.core.wmem_default = $TARGET_WMEM_DEFAULT

# 网络性能
net.core.netdev_budget = $TARGET_NETDEV_BUDGET
net.core.busy_poll = $TARGET_BUSY_POLL
net.core.busy_read = $TARGET_BUSY_READ

# 内存
vm.max_map_count = 262144
EOF

    echo "  配置文件已写入: $SYSCTL_CONF"

    # 应用配置
    sysctl -p "$SYSCTL_CONF" > /dev/null 2>&1

    echo ""
    echo -e "${GREEN}✅ 配置已持久化，重启后仍然有效${NC}"

    # 提示配置 ulimit
    echo ""
    echo "📋 还需要配置进程级别的文件描述符限制:"
    echo ""
    echo "在 /etc/security/limits.conf 中添加:"
    echo "  * soft nofile 1000000"
    echo "  * hard nofile 1000000"
    echo ""
    echo "或在 systemd service 文件中添加:"
    echo "  [Service]"
    echo "  LimitNOFILE=1000000"
}

# 创建 systemd service 文件
create_service() {
    local SERVICE_FILE="/etc/systemd/system/quic-server.service"
    local BINARY_PATH="${1:-/usr/local/bin/quic-server}"
    local CERT_PATH="${2:-/etc/quic-flow/certs}"

    echo "=== 创建 systemd service 文件 ==="

    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=QUIC Flow Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/quic-flow

# 命令行
ExecStart=$BINARY_PATH \\
    --addr :8474 \\
    --cert $CERT_PATH/server-cert.pem \\
    --key $CERT_PATH/server-key.pem \\
    --api :8475

# 资源限制 (高并发优化)
LimitNOFILE=1000000
LimitNPROC=65535
LimitCORE=infinity

# 内存限制 (可选)
# MemoryLimit=32G

# 重启策略
Restart=always
RestartSec=5

# 环境变量
Environment=GOMAXPROCS=0
Environment=GOGC=100

[Install]
WantedBy=multi-user.target
EOF

    echo "  Service 文件已创建: $SERVICE_FILE"
    echo ""
    echo "使用方法:"
    echo "  systemctl daemon-reload"
    echo "  systemctl enable quic-server"
    echo "  systemctl start quic-server"
}

# 主程序
main() {
    local cmd=${1:-apply}

    case "$cmd" in
        check)
            check_config
            ;;
        apply)
            check_root
            apply_tuning
            ;;
        persist)
            check_root
            apply_tuning
            persist_settings
            ;;
        service)
            check_root
            create_service "${2:-}" "${3:-}"
            ;;
        *)
            echo "QUIC Flow 系统调优脚本"
            echo ""
            echo "用法: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  check    检查当前系统配置"
            echo "  apply    应用调优设置 (默认)"
            echo "  persist  持久化调优设置"
            echo "  service  创建 systemd service 文件"
            echo ""
            ;;
    esac
}

main "$@"
