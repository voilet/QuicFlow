package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/voilet/quic-flow/pkg/command"
)

// GetNetworkInterfaces 获取网络接口列表
// 命令类型: network.interfaces
// 用法: r.Register(command.CmdNetworkInterfaces, handlers.GetNetworkInterfaces)
func GetNetworkInterfaces(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.NetworkInterfacesParams
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var result command.NetworkInterfacesResult
	for _, iface := range interfaces {
		ni := command.NetworkInterface{
			Name:         iface.Name,
			Index:        iface.Index,
			HardwareAddr: iface.HardwareAddr.String(),
			MTU:          iface.MTU,
			Flags:        parseFlags(iface.Flags),
			IsUp:         iface.Flags&net.FlagUp != 0,
		}

		// 获取 IP 地址
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				ni.Addresses = append(ni.Addresses, addr.String())
			}
		}

		// 在 Linux 上获取额外信息
		if runtime.GOOS == "linux" {
			ni.IsPhysical = isPhysicalInterface(iface.Name)
			ni.Driver = getInterfaceDriver(iface.Name)
			ni.Speed = getInterfaceSpeed(iface.Name)
			ni.Duplex = getInterfaceDuplex(iface.Name)
			ni.LinkDetected = getInterfaceLinkDetected(iface.Name)
		} else {
			// 非 Linux 系统，简单判断物理网卡
			ni.IsPhysical = isPhysicalInterfaceGeneric(iface)
			ni.Speed = -1 // 未知
			ni.Duplex = "unknown"
		}

		// 如果只要物理网卡，跳过虚拟接口
		if params.PhysicalOnly && !ni.IsPhysical {
			continue
		}

		result.Interfaces = append(result.Interfaces, ni)
	}

	result.Count = len(result.Interfaces)
	return json.Marshal(result)
}

// GetNetworkSpeed 获取网卡协商速率
// 命令类型: network.speed
// 用法: r.Register(command.CmdNetworkSpeed, handlers.GetNetworkSpeed)
func GetNetworkSpeed(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var params command.NetworkSpeedParams
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}

	var result command.NetworkSpeedResult

	if params.InterfaceName != "" {
		// 获取指定接口的速率
		info, err := getSpeedInfo(params.InterfaceName)
		if err != nil {
			return nil, err
		}
		result.Interfaces = append(result.Interfaces, *info)
	} else {
		// 获取所有物理接口的速率
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil, fmt.Errorf("failed to get interfaces: %w", err)
		}

		for _, iface := range interfaces {
			// 只处理物理网卡
			if runtime.GOOS == "linux" && !isPhysicalInterface(iface.Name) {
				continue
			}
			if runtime.GOOS != "linux" && !isPhysicalInterfaceGeneric(iface) {
				continue
			}

			info, err := getSpeedInfo(iface.Name)
			if err != nil {
				continue // 跳过无法获取信息的接口
			}
			result.Interfaces = append(result.Interfaces, *info)
		}
	}

	result.Count = len(result.Interfaces)
	return json.Marshal(result)
}

// getSpeedInfo 获取单个接口的速率信息
func getSpeedInfo(name string) (*command.NetworkSpeedInfo, error) {
	info := &command.NetworkSpeedInfo{
		Name:   name,
		Speed:  -1,
		Duplex: "unknown",
	}

	if runtime.GOOS == "linux" {
		info.Speed = getInterfaceSpeed(name)
		info.Duplex = getInterfaceDuplex(name)
		info.LinkDetected = getInterfaceLinkDetected(name)
		info.AutoNeg = getInterfaceAutoNeg(name)
		info.Driver = getInterfaceDriver(name)
		info.BusInfo = getInterfaceBusInfo(name)
	} else {
		// 非 Linux 系统，尝试基本信息
		iface, err := net.InterfaceByName(name)
		if err != nil {
			return nil, fmt.Errorf("interface %s not found: %w", name, err)
		}
		info.LinkDetected = iface.Flags&net.FlagUp != 0
	}

	return info, nil
}

// parseFlags 解析网络接口标志
func parseFlags(flags net.Flags) []string {
	var result []string
	if flags&net.FlagUp != 0 {
		result = append(result, "up")
	}
	if flags&net.FlagBroadcast != 0 {
		result = append(result, "broadcast")
	}
	if flags&net.FlagLoopback != 0 {
		result = append(result, "loopback")
	}
	if flags&net.FlagPointToPoint != 0 {
		result = append(result, "pointtopoint")
	}
	if flags&net.FlagMulticast != 0 {
		result = append(result, "multicast")
	}
	if flags&net.FlagRunning != 0 {
		result = append(result, "running")
	}
	return result
}

// isPhysicalInterface 判断是否为物理网卡 (Linux)
func isPhysicalInterface(name string) bool {
	// 排除常见的虚拟接口
	virtualPrefixes := []string{
		"lo",     // loopback
		"veth",   // virtual ethernet (Docker/containers)
		"docker", // Docker bridge
		"br-",    // bridge
		"virbr",  // libvirt bridge
		"vnet",   // KVM/libvirt
		"tap",    // TAP device
		"tun",    // TUN device
		"dummy",  // dummy interface
		"bond",   // bonding (虽然是物理的聚合，但特殊处理)
		"team",   // team interface
		"vlan",   // VLAN interface
	}

	nameLower := strings.ToLower(name)
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(nameLower, prefix) {
			return false
		}
	}

	// 检查是否有对应的物理设备
	devicePath := filepath.Join("/sys/class/net", name, "device")
	if _, err := os.Stat(devicePath); err == nil {
		return true
	}

	return false
}

// isPhysicalInterfaceGeneric 通用判断是否为物理网卡
func isPhysicalInterfaceGeneric(iface net.Interface) bool {
	// 排除 loopback
	if iface.Flags&net.FlagLoopback != 0 {
		return false
	}

	// 有 MAC 地址的通常是物理网卡
	if len(iface.HardwareAddr) > 0 {
		// 排除常见虚拟接口名称
		name := strings.ToLower(iface.Name)
		virtualNames := []string{"lo", "veth", "docker", "br-", "virbr", "vnet", "tap", "tun"}
		for _, v := range virtualNames {
			if strings.HasPrefix(name, v) {
				return false
			}
		}
		return true
	}

	return false
}

// getInterfaceSpeed 获取接口速率 (Mbps)
func getInterfaceSpeed(name string) int {
	speedPath := filepath.Join("/sys/class/net", name, "speed")
	data, err := os.ReadFile(speedPath)
	if err != nil {
		return -1
	}

	speed, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1
	}

	return speed
}

// getInterfaceDuplex 获取双工模式
func getInterfaceDuplex(name string) string {
	duplexPath := filepath.Join("/sys/class/net", name, "duplex")
	data, err := os.ReadFile(duplexPath)
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(data))
}

// getInterfaceLinkDetected 检测是否有链路
func getInterfaceLinkDetected(name string) bool {
	carrierPath := filepath.Join("/sys/class/net", name, "carrier")
	data, err := os.ReadFile(carrierPath)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(data)) == "1"
}

// getInterfaceAutoNeg 获取是否自动协商
func getInterfaceAutoNeg(name string) bool {
	// ethtool 的自动协商信息在 /sys 中没有直接暴露
	// 需要通过 ioctl 或 netlink 获取，这里简化处理
	// 大多数现代网卡默认开启自动协商
	return true
}

// getInterfaceDriver 获取驱动名称
func getInterfaceDriver(name string) string {
	driverPath := filepath.Join("/sys/class/net", name, "device", "driver")
	target, err := os.Readlink(driverPath)
	if err != nil {
		return ""
	}

	// 返回驱动名称（链接目标的最后一部分）
	return filepath.Base(target)
}

// getInterfaceBusInfo 获取总线信息
func getInterfaceBusInfo(name string) string {
	devicePath := filepath.Join("/sys/class/net", name, "device")
	target, err := os.Readlink(devicePath)
	if err != nil {
		return ""
	}

	// 提取 PCI 地址等总线信息
	// 例如: ../../../0000:00:03.0 -> 0000:00:03.0
	return filepath.Base(target)
}
