#!/bin/bash
# LVM 和磁盘挂载点调试脚本

echo "=========================================="
echo "1. 系统分区信息 (df -h)"
echo "=========================================="
df -h | grep -v tmpfs | grep -v overlay

echo ""
echo "=========================================="
echo "2. 块设备信息 (lsblk)"
echo "=========================================="
lsblk -o NAME,SIZE,TYPE,MOUNTPOINT,PKNAME

echo ""
echo "=========================================="
echo "3. /proc/mounts 中的 LVM 挂载"
echo "=========================================="
grep -E "^/dev/(mapper|dm-)" /proc/mounts 2>/dev/null || echo "无 LVM 挂载"

echo ""
echo "=========================================="
echo "4. /dev/mapper/ 设备列表"
echo "=========================================="
ls -la /dev/mapper/ 2>/dev/null || echo "/dev/mapper 不存在"

echo ""
echo "=========================================="
echo "5. dm-* 设备及其 name 和 slaves"
echo "=========================================="
for dm in /sys/block/dm-*; do
    if [ -d "$dm" ]; then
        dm_name=$(basename "$dm")
        name_file="$dm/dm/name"
        slaves_dir="$dm/slaves"

        echo "--- $dm_name ---"
        if [ -f "$name_file" ]; then
            echo "  name: $(cat "$name_file")"
        fi
        if [ -d "$slaves_dir" ]; then
            echo "  slaves: $(ls "$slaves_dir" 2>/dev/null | tr '\n' ' ')"
        fi
        echo ""
    fi
done

echo ""
echo "=========================================="
echo "6. 物理磁盘 (/sys/block/)"
echo "=========================================="
for disk in /sys/block/*; do
    name=$(basename "$disk")
    # 跳过虚拟设备
    if [[ "$name" == loop* ]] || [[ "$name" == ram* ]] || [[ "$name" == dm-* ]]; then
        continue
    fi
    size_sectors=$(cat "$disk/size" 2>/dev/null)
    if [ -n "$size_sectors" ] && [ "$size_sectors" -gt 0 ]; then
        size_bytes=$((size_sectors * 512))
        size_gb=$((size_bytes / 1024 / 1024 / 1024))
        echo "$name: ${size_gb}GB"
    fi
done

echo ""
echo "=========================================="
echo "7. 测试设备名解析"
echo "=========================================="
# 模拟 getPhysicalDisks 逻辑
test_device() {
    local devName="$1"
    echo "测试设备: $devName"

    # dm-* 设备
    if [[ "$devName" == dm-* ]]; then
        echo "  -> 类型: dm 设备"
        slaves_dir="/sys/block/$devName/slaves"
        if [ -d "$slaves_dir" ]; then
            echo "  -> slaves: $(ls "$slaves_dir" 2>/dev/null | tr '\n' ' ')"
        fi
        return
    fi

    # NVMe 设备
    if [[ "$devName" == nvme* ]]; then
        echo "  -> 类型: NVMe"
        # nvme0n1p1 -> nvme0n1
        disk=$(echo "$devName" | sed 's/p[0-9]*$//')
        echo "  -> 磁盘: $disk"
        return
    fi

    # 普通分区
    if [[ "$devName" =~ ^[a-z]+[0-9]+$ ]]; then
        disk=$(echo "$devName" | sed 's/[0-9]*$//')
        echo "  -> 类型: 普通分区"
        echo "  -> 磁盘: $disk"
        return
    fi

    # 可能是 mapper 名称
    mapper_path="/dev/mapper/$devName"
    if [ -L "$mapper_path" ]; then
        target=$(readlink "$mapper_path")
        dm_name=$(basename "$target")
        echo "  -> 类型: mapper 符号链接"
        echo "  -> 目标: $dm_name"
        if [ -d "/sys/block/$dm_name/slaves" ]; then
            echo "  -> slaves: $(ls "/sys/block/$dm_name/slaves" 2>/dev/null | tr '\n' ' ')"
        fi
        return
    fi

    # 遍历 dm-* 匹配 name
    for dm in /sys/block/dm-*; do
        if [ -f "$dm/dm/name" ]; then
            name=$(cat "$dm/dm/name")
            if [ "$name" == "$devName" ]; then
                dm_name=$(basename "$dm")
                echo "  -> 类型: 通过 dm/name 匹配到 $dm_name"
                if [ -d "$dm/slaves" ]; then
                    echo "  -> slaves: $(ls "$dm/slaves" 2>/dev/null | tr '\n' ' ')"
                fi
                return
            fi
        fi
    done

    echo "  -> 未识别"
}

# 从 /proc/mounts 获取设备名进行测试
echo "从 /proc/mounts 中提取设备进行测试:"
awk '{print $1}' /proc/mounts | grep -E "^/dev/" | while read device; do
    devName=$(basename "$device")
    test_device "$devName"
    echo ""
done

echo ""
echo "=========================================="
echo "8. Go gopsutil 分区信息模拟"
echo "=========================================="
echo "以下是 findmnt 输出（类似 gopsutil 获取的信息）:"
findmnt -l -o SOURCE,TARGET,FSTYPE | grep -v tmpfs | grep -v overlay | head -20
