package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/voilet/quic-flow/pkg/command"
)

// GetHardwareInfo 获取完整硬件信息
// 命令类型: hardware.info
// 用法: r.Register(command.CmdHardwareInfo, handlers.GetHardwareInfo)
func GetHardwareInfo(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	result := command.HardwareInfoResult{}

	// 获取 DMI 信息
	result.DMI = getDMIInfo()

	// 获取主机信息
	result.Host = getHostInfo(ctx)

	// 获取 CPU 信息
	cpuInfo := getCPUInfo(ctx)
	result.ModelName = cpuInfo.modelName
	result.CPUCoreCount = cpuInfo.coreCount
	result.CPUThreadCount = cpuInfo.threadCount
	result.LogicalCPUFrequencyMHz = cpuInfo.logicalFreq
	result.PhysicalCPUFrequencyMHz = cpuInfo.physicalFreq
	result.SiblingsNum = cpuInfo.siblings
	result.NumCPUKernel = cpuInfo.numCPU

	// 获取磁盘信息
	diskInfo := getDiskInfo(ctx)
	result.Disks = diskInfo.disks
	result.TotalDiskCapacityTB = diskInfo.totalTB
	result.TotalDiskCapacityTBDecimal = diskInfo.totalTBDecimal
	result.TotalDiskCapacityBytes = diskInfo.totalBytes

	// 获取内存信息
	result.Memory = getMemoryInfo(ctx)

	// 获取网卡信息
	nicInfo := getNICInfo()
	result.NICInfos = nicInfo.nics
	result.MAC = nicInfo.primaryMAC

	return json.Marshal(result)
}

// getDMIInfo 读取 DMI/SMBIOS 信息
func getDMIInfo() command.DMIInfo {
	dmi := command.DMIInfo{}

	if runtime.GOOS != "linux" {
		return dmi
	}

	dmiPath := "/sys/class/dmi/id"

	readFile := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dmiPath, name))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}

	dmi.BiosDate = readFile("bios_date")
	dmi.BiosVendor = readFile("bios_vendor")
	dmi.BiosVersion = readFile("bios_version")
	dmi.BoardName = readFile("board_name")
	dmi.BoardSerial = readFile("board_serial")
	dmi.BoardVendor = readFile("board_vendor")
	dmi.BoardVersion = readFile("board_version")
	dmi.BoardAssetTag = readFile("board_asset_tag")
	dmi.ChassisType = readFile("chassis_type")
	dmi.ChassisSerial = readFile("chassis_serial")
	dmi.ChassisVendor = readFile("chassis_vendor")
	dmi.ChassisVersion = readFile("chassis_version")
	dmi.ChassisAssetTag = readFile("chassis_asset_tag")
	dmi.ProductName = readFile("product_name")
	dmi.ProductSerial = readFile("product_serial")
	dmi.ProductUUID = readFile("product_uuid")
	dmi.ProductVersion = readFile("product_version")
	dmi.SysVendor = readFile("sys_vendor")
	dmi.Modalias = readFile("modalias")
	dmi.Uevent = readFile("uevent")

	return dmi
}

// getHostInfo 获取主机信息
func getHostInfo(ctx context.Context) command.HostInfo {
	info := command.HostInfo{
		OS: runtime.GOOS,
	}

	hostInfo, err := host.InfoWithContext(ctx)
	if err == nil {
		info.HostID = hostInfo.HostID
		info.Hostname = hostInfo.Hostname
		info.Uptime = hostInfo.Uptime
		info.BootTime = hostInfo.BootTime
		info.Procs = hostInfo.Procs
		info.Platform = hostInfo.Platform
		info.PlatformFamily = hostInfo.PlatformFamily
		info.PlatformVersion = hostInfo.PlatformVersion
		info.KernelVersion = hostInfo.KernelVersion
		info.KernelArch = hostInfo.KernelArch
		info.VirtualizationSystem = hostInfo.VirtualizationSystem
		info.VirtualizationRole = hostInfo.VirtualizationRole
	}

	return info
}

// cpuInfoResult CPU 信息结果
type cpuInfoResult struct {
	modelName    string
	coreCount    int
	threadCount  int
	logicalFreq  float64
	physicalFreq float64
	siblings     string
	numCPU       int
}

// getCPUInfo 获取 CPU 信息
func getCPUInfo(ctx context.Context) cpuInfoResult {
	result := cpuInfoResult{
		numCPU: runtime.NumCPU(),
	}

	cpuInfos, err := cpu.InfoWithContext(ctx)
	if err == nil && len(cpuInfos) > 0 {
		result.modelName = cpuInfos[0].ModelName
		result.physicalFreq = cpuInfos[0].Mhz

		// 统计物理核心和逻辑处理器
		physicalIDs := make(map[string]bool)
		for _, info := range cpuInfos {
			physicalIDs[info.PhysicalID] = true
		}

		// 计算核心数
		if len(physicalIDs) > 0 {
			result.coreCount = int(cpuInfos[0].Cores) * len(physicalIDs)
		} else {
			result.coreCount = int(cpuInfos[0].Cores)
		}

		result.threadCount = len(cpuInfos)
		result.siblings = strconv.Itoa(int(cpuInfos[0].Cores))
	}

	// 尝试获取逻辑 CPU 频率
	percentages, err := cpu.PercentWithContext(ctx, 0, false)
	if err == nil && len(percentages) > 0 {
		// 如果有运行时频率信息，可以在这里获取
		result.logicalFreq = result.physicalFreq
	}

	return result
}

// diskInfoResult 磁盘信息结果
type diskInfoResult struct {
	disks          []command.DiskInfo
	totalTB        float64 // 二进制计算 (1TiB = 1024^4 bytes)
	totalTBDecimal float64 // 十进制计算 (1TB = 1000^4 bytes，厂商标注)
	totalBytes     uint64
}

const (
	// 二进制单位 (1 TiB = 1024^4 bytes)
	bytesPerTiB = 1024 * 1024 * 1024 * 1024
	// 十进制单位 (1 TB = 1000^4 bytes，硬盘厂商使用)
	bytesPerTBDecimal = 1000 * 1000 * 1000 * 1000
)

// getPhysicalDisksFromPath 从设备完整路径获取物理磁盘名列表
// 处理 /dev/sda1, /dev/mapper/xxx, /dev/vgname/lvname 等格式
func getPhysicalDisksFromPath(devicePath string) []string {
	// 处理 /dev/vgname/lvname 格式 (如 /dev/bydata/btvdp_xxx)
	// 转换为 mapper 名称 vgname-lvname
	if strings.HasPrefix(devicePath, "/dev/") && !strings.HasPrefix(devicePath, "/dev/mapper/") {
		parts := strings.Split(devicePath, "/")
		if len(parts) == 4 {
			// /dev/vgname/lvname -> vgname-lvname
			vgName := parts[2]
			lvName := parts[3]
			// 检查是否是 VG/LV 格式（VG 不是标准设备名如 sda, nvme 等）
			if !isStandardDevicePrefix(vgName) {
				mapperName := vgName + "-" + lvName
				return getPhysicalDisks(mapperName)
			}
		}
	}

	// 使用设备名处理
	devName := filepath.Base(devicePath)
	return getPhysicalDisks(devName)
}

// isStandardDevicePrefix 检查是否是标准设备前缀
func isStandardDevicePrefix(name string) bool {
	standardPrefixes := []string{
		"sd", "hd", "vd", "xvd", // SCSI, IDE, virtio, Xen
		"nvme", "mmcblk", // NVMe, eMMC
		"loop", "ram", "dm-", // 虚拟设备
	}
	for _, prefix := range standardPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// getPhysicalDisks 获取设备对应的物理磁盘名列表
// 处理 LVM、普通分区、NVMe、bcache 等情况
func getPhysicalDisks(devName string) []string {
	var result []string

	// 处理 bcache 设备
	if strings.HasPrefix(devName, "bcache") {
		slaves := getBcacheBackingDevices(devName)
		if len(slaves) > 0 {
			return slaves
		}
		return result
	}

	// 处理 dm-* 设备（LVM、device-mapper）
	if strings.HasPrefix(devName, "dm-") {
		slaves := getLVMSlaves(devName)
		if len(slaves) > 0 {
			return slaves
		}
		return result
	}

	// 处理 NVMe 设备 (nvme0n1p1 -> nvme0n1)
	if strings.HasPrefix(devName, "nvme") {
		if idx := strings.LastIndex(devName, "p"); idx > 0 {
			suffix := devName[idx+1:]
			if _, err := strconv.Atoi(suffix); err == nil {
				result = append(result, devName[:idx])
				return result
			}
		}
		result = append(result, devName)
		return result
	}

	// 处理普通磁盘分区 (sda1 -> sda, vda1 -> vda)
	// 仅匹配标准设备名格式：2-3个字母 + 数字
	if isStandardPartition(devName) {
		diskName := strings.TrimRight(devName, "0123456789")
		if diskName != "" && diskName != devName {
			result = append(result, diskName)
			return result
		}
	}

	// 可能是 LVM mapper 名称（如 bydata-xxx）
	// 尝试通过 /dev/mapper/ 符号链接找到 dm-* 设备
	mapperPath := filepath.Join("/dev/mapper", devName)
	if target, err := os.Readlink(mapperPath); err == nil {
		dmName := filepath.Base(target)
		if strings.HasPrefix(dmName, "dm-") {
			slaves := getLVMSlaves(dmName)
			if len(slaves) > 0 {
				return slaves
			}
		}
	}

	// 尝试通过遍历 /sys/block/dm-*/dm/name 匹配 mapper 名称
	dmDevices, _ := filepath.Glob("/sys/block/dm-*")
	for _, dmPath := range dmDevices {
		nameFile := filepath.Join(dmPath, "dm", "name")
		if nameData, err := os.ReadFile(nameFile); err == nil {
			if strings.TrimSpace(string(nameData)) == devName {
				dmName := filepath.Base(dmPath)
				slaves := getLVMSlaves(dmName)
				if len(slaves) > 0 {
					return slaves
				}
			}
		}
	}

	return result
}

// isStandardPartition 检查是否为标准分区格式（如 sda1, vda2, xvda1）
func isStandardPartition(name string) bool {
	if len(name) < 2 {
		return false
	}

	// 标准分区格式：字母前缀 + 数字后缀
	// 例如：sda1, sdb2, vda1, xvda1, hda1
	prefixes := []string{"sd", "vd", "xvd", "hd"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			// 检查前缀后是否是字母+数字的组合
			rest := name[len(prefix):]
			if len(rest) >= 2 {
				// 第一个字符应该是字母 (a-z)
				if rest[0] >= 'a' && rest[0] <= 'z' {
					// 剩余应该全是数字
					for _, c := range rest[1:] {
						if c < '0' || c > '9' {
							return false
						}
					}
					return len(rest) > 1 // 至少有一个数字
				}
			}
		}
	}
	return false
}

// getLVMSlaves 获取 LVM/dm 设备的底层物理磁盘
func getLVMSlaves(dmName string) []string {
	var result []string
	seen := make(map[string]bool)

	// 递归查找底层设备
	var findSlaves func(name string)
	findSlaves = func(name string) {
		slavesPath := filepath.Join("/sys/block", name, "slaves")
		entries, err := os.ReadDir(slavesPath)
		if err != nil || len(entries) == 0 {
			return
		}

		for _, entry := range entries {
			slaveName := entry.Name()

			// 如果是 dm-* 设备，继续递归查找
			if strings.HasPrefix(slaveName, "dm-") {
				findSlaves(slaveName)
				continue
			}

			// 找到物理设备，提取磁盘名
			physicalDisks := getPhysicalDiskFromPartition(slaveName)
			for _, disk := range physicalDisks {
				if !seen[disk] {
					seen[disk] = true
					result = append(result, disk)
				}
			}
		}
	}

	findSlaves(dmName)
	return result
}

// getBcacheBackingDevices 获取 bcache 设备的底层物理磁盘
// bcache 是 Linux 块设备缓存层，将 SSD 作为 HDD 的缓存
func getBcacheBackingDevices(bcacheName string) []string {
	var result []string
	seen := make(map[string]bool)

	// 方法1: 通过 /sys/block/bcacheN/slaves/ 获取底层设备
	slavesPath := filepath.Join("/sys/block", bcacheName, "slaves")
	entries, err := os.ReadDir(slavesPath)
	if err == nil && len(entries) > 0 {
		for _, entry := range entries {
			slaveName := entry.Name()
			// 获取物理磁盘名（去掉分区号）
			physicalDisks := getPhysicalDiskFromPartition(slaveName)
			for _, disk := range physicalDisks {
				if !seen[disk] {
					seen[disk] = true
					result = append(result, disk)
				}
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	// 方法2: 通过 /sys/block/bcacheN/bcache/backing_dev_name 获取
	backingDevPath := filepath.Join("/sys/block", bcacheName, "bcache", "backing_dev_name")
	if data, err := os.ReadFile(backingDevPath); err == nil {
		devName := strings.TrimSpace(string(data))
		if devName != "" {
			physicalDisks := getPhysicalDiskFromPartition(devName)
			for _, disk := range physicalDisks {
				if !seen[disk] {
					seen[disk] = true
					result = append(result, disk)
				}
			}
		}
	}

	return result
}

// getPhysicalDiskFromPartition 从分区名获取物理磁盘名
func getPhysicalDiskFromPartition(partName string) []string {
	var result []string

	// NVMe: nvme0n1p1 -> nvme0n1
	if strings.HasPrefix(partName, "nvme") {
		if idx := strings.LastIndex(partName, "p"); idx > 0 {
			suffix := partName[idx+1:]
			if _, err := strconv.Atoi(suffix); err == nil {
				result = append(result, partName[:idx])
				return result
			}
		}
		result = append(result, partName)
		return result
	}

	// 普通磁盘: sda1 -> sda
	diskName := strings.TrimRight(partName, "0123456789")
	if diskName != "" {
		result = append(result, diskName)
	}

	return result
}

// getDiskInfo 获取磁盘信息
func getDiskInfo(ctx context.Context) diskInfoResult {
	result := diskInfoResult{}

	if runtime.GOOS != "linux" {
		return getGenericDiskInfo(ctx)
	}

	// 获取分区使用情况
	partitions, _ := disk.PartitionsWithContext(ctx, false)
	mountUsageMap := make(map[string][]command.DiskMountUsage)

	for _, p := range partitions {
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue
		}

		// 获取物理磁盘名（处理 LVM 和普通分区）
		diskNames := getPhysicalDisksFromPath(p.Device)

		for _, diskName := range diskNames {
			mountUsageMap[diskName] = append(mountUsageMap[diskName], command.DiskMountUsage{
				MountPoint:  p.Mountpoint,
				UsedPercent: usage.UsedPercent,
			})
		}
	}

	// 读取 /sys/block 获取磁盘信息
	blockDevices, err := os.ReadDir("/sys/block")
	if err != nil {
		return getGenericDiskInfo(ctx)
	}

	for _, dev := range blockDevices {
		name := dev.Name()

		// 跳过虚拟设备
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "dm-") ||
			strings.HasPrefix(name, "bcache") || // bcache 缓存设备
			strings.HasPrefix(name, "md") || // RAID 设备
			strings.HasPrefix(name, "drbd") || // DRBD 设备
			strings.HasPrefix(name, "rbd") || // Ceph RBD 设备
			strings.HasPrefix(name, "nbd") { // 网络块设备
			continue
		}

		diskInfo := command.DiskInfo{
			Device: name,
			Type:   "disk",
		}

		basePath := filepath.Join("/sys/block", name)

		// 读取大小（扇区数，每扇区 512 字节）
		if sizeData, err := os.ReadFile(filepath.Join(basePath, "size")); err == nil {
			sectors, _ := strconv.ParseUint(strings.TrimSpace(string(sizeData)), 10, 64)
			diskInfo.SizeRoundedBytes = sectors * 512
			diskInfo.SizeRoundedTB = float64(diskInfo.SizeRoundedBytes) / bytesPerTiB       // 二进制计算
			diskInfo.SizeTBDecimal = float64(diskInfo.SizeRoundedBytes) / bytesPerTBDecimal // 十进制计算（厂商标注）
			result.totalBytes += diskInfo.SizeRoundedBytes
		}

		// 读取型号
		if modelData, err := os.ReadFile(filepath.Join(basePath, "device/model")); err == nil {
			diskInfo.Model = strings.TrimSpace(string(modelData))
		}

		// 判断磁盘类型（SSD/HDD）
		if rotationalData, err := os.ReadFile(filepath.Join(basePath, "queue/rotational")); err == nil {
			if strings.TrimSpace(string(rotationalData)) == "0" {
				diskInfo.Kind = "SSD"
			} else {
				diskInfo.Kind = "HDD"
			}
		}

		// 检查是否为 NVMe
		if strings.HasPrefix(name, "nvme") {
			diskInfo.Kind = "NVMe"
		}

		// 添加挂载使用情况（过滤 boot 分区）
		if usages, ok := mountUsageMap[name]; ok {
			var filteredUsages []command.DiskMountUsage
			for _, u := range usages {
				// 过滤掉 boot 相关挂载点
				if u.MountPoint == "/boot" || u.MountPoint == "/boot/efi" {
					continue
				}
				filteredUsages = append(filteredUsages, u)
				// 检查是否包含根分区
				if u.MountPoint == "/" {
					diskInfo.IsSystemDisk = true
				}
			}
			diskInfo.MountUsages = filteredUsages
		}

		result.disks = append(result.disks, diskInfo)
	}

	result.totalTB = float64(result.totalBytes) / bytesPerTiB
	result.totalTBDecimal = float64(result.totalBytes) / bytesPerTBDecimal

	return result
}

// getGenericDiskInfo 通用磁盘信息获取（非 Linux 系统）
func getGenericDiskInfo(ctx context.Context) diskInfoResult {
	result := diskInfoResult{}

	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return result
	}

	for _, p := range partitions {
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue
		}

		diskInfo := command.DiskInfo{
			Device:           p.Device,
			Type:             p.Fstype,
			Kind:             "Unknown",
			SizeRoundedBytes: usage.Total,
			SizeRoundedTB:    float64(usage.Total) / bytesPerTiB,
			SizeTBDecimal:    float64(usage.Total) / bytesPerTBDecimal,
			MountUsages: []command.DiskMountUsage{
				{
					MountPoint:  p.Mountpoint,
					UsedPercent: usage.UsedPercent,
				},
			},
		}

		if p.Mountpoint == "/" || p.Mountpoint == "C:\\" {
			diskInfo.IsSystemDisk = true
		}

		result.disks = append(result.disks, diskInfo)
		result.totalBytes += usage.Total
	}

	result.totalTB = float64(result.totalBytes) / bytesPerTiB
	result.totalTBDecimal = float64(result.totalBytes) / bytesPerTBDecimal

	return result
}

// getMemoryInfo 获取内存信息
func getMemoryInfo(ctx context.Context) command.MemoryInfo {
	result := command.MemoryInfo{}

	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		result.TotalBytes = vmem.Total
		result.TotalGB = float64(vmem.Total) / (1024 * 1024 * 1024)
		result.TotalGBRounded = int(math.Round(result.TotalGB))
	}

	// 尝试读取 DMI 内存信息（仅 Linux）
	if runtime.GOOS == "linux" {
		result.Modules = getDMIMemoryModules()
		result.Count = len(result.Modules)
	}

	return result
}

// getDMIMemoryModules 从 dmidecode 读取内存模块信息
func getDMIMemoryModules() []command.MemoryModule {
	var modules []command.MemoryModule

	// 尝试从 /sys/firmware/dmi/tables 读取
	// 这通常需要 root 权限，作为备选方案使用模拟数据
	dmiPath := "/sys/firmware/dmi/entries/17-*"

	entries, err := filepath.Glob(dmiPath)
	if err != nil || len(entries) == 0 {
		return modules
	}

	for i, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(entry, "raw"))
		if err != nil {
			continue
		}

		module := command.MemoryModule{
			Locator: fmt.Sprintf("DIMM %d", i),
		}

		// 解析 DMI Type 17 结构
		if len(raw) >= 0x15 {
			size := uint16(raw[0x0C]) | uint16(raw[0x0D])<<8
			if size > 0 && size != 0xFFFF {
				if size&0x8000 != 0 {
					module.Size = fmt.Sprintf("%d KB", size&0x7FFF)
				} else {
					module.Size = fmt.Sprintf("%d MB", size)
				}
			}
		}

		if module.Size != "" {
			module.Type = "RAM"
			modules = append(modules, module)
		}
	}

	return modules
}

// nicInfoResult 网卡信息结果
type nicInfoResult struct {
	nics       []command.NICInfo
	primaryMAC string
}

// getNICInfo 获取网卡信息（仅物理网卡）
func getNICInfo() nicInfoResult {
	result := nicInfoResult{}

	interfaces, err := net.Interfaces()
	if err != nil {
		return result
	}

	for _, iface := range interfaces {
		// 跳过 loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 只获取物理网卡
		if !isPhysicalNIC(iface.Name) {
			continue
		}

		nic := command.NICInfo{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			IsPhysical: true,
		}

		// 判断状态
		if iface.Flags&net.FlagUp != 0 {
			nic.Status = "up"
		} else {
			nic.Status = "down"
		}

		// 获取 IP 地址
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}

				if ipNet.IP.To4() != nil {
					nic.IPAddress = ipNet.IP.String()
				} else if ipNet.IP.To16() != nil && nic.IPv6 == "" {
					nic.IPv6 = ipNet.IP.String()
				}
			}
		}

		// 获取网卡速率（仅 Linux）
		nic.Speed = getNICSpeed(iface.Name)

		result.nics = append(result.nics, nic)

		// 记录第一个物理网卡的 MAC 作为主 MAC
		if result.primaryMAC == "" && len(iface.HardwareAddr) > 0 {
			// 转换为无分隔符的大写格式
			result.primaryMAC = strings.ToUpper(strings.ReplaceAll(iface.HardwareAddr.String(), ":", ""))
		}
	}

	return result
}

// isPhysicalNIC 判断是否为物理网卡
// 排除已知的虚拟网卡，保留物理网卡和容器主网卡
func isPhysicalNIC(name string) bool {
	// 排除已知的虚拟网卡前缀
	virtualPrefixes := []string{
		"lo",      // loopback
		"docker",  // docker bridge
		"br-",     // docker/bridge networks
		"veth",    // virtual ethernet (docker container peers)
		"virbr",   // libvirt bridge
		"vnet",    // libvirt virtual network
		"tun",     // tunnel devices
		"tap",     // tap devices
		"dummy",   // dummy interfaces
		"bond",    // bonding (除非是实际使用的)
		"team",    // team interfaces
		"macvlan", // macvlan interfaces
		"ipvlan",  // ipvlan interfaces
		"vxlan",   // vxlan interfaces
		"flannel", // flannel CNI
		"cni",     // CNI interfaces
		"cali",    // calico interfaces
		"tunl",    // tunnel interfaces
		"wg",      // wireguard
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}

	if runtime.GOOS != "linux" {
		return true
	}

	// Linux: 优先检查 /sys/class/net/<name>/device 是否存在（真实物理网卡）
	devicePath := filepath.Join("/sys/class/net", name, "device")
	if _, err := os.Stat(devicePath); err == nil {
		return true
	}

	// 容器环境中的 eth*/ens*/enp*/eno* 等常见网卡命名也应该被识别
	physicalPatterns := []string{
		"eth", // 传统命名 eth0, eth1
		"ens", // systemd 命名 ens33, ens192
		"enp", // systemd 命名 enp0s3
		"eno", // 板载网卡 eno1
		"em",  // Dell/HP 服务器命名
		"p",   // 某些服务器命名 p1p1
	}

	for _, pattern := range physicalPatterns {
		if strings.HasPrefix(name, pattern) {
			return true
		}
	}

	return false
}

// getNICSpeed 获取网卡速率
func getNICSpeed(name string) string {
	if runtime.GOOS != "linux" {
		return "Unknown!"
	}

	speedPath := filepath.Join("/sys/class/net", name, "speed")
	data, err := os.ReadFile(speedPath)
	if err != nil {
		return "Unknown!"
	}

	speed := strings.TrimSpace(string(data))
	if speed == "-1" || speed == "" {
		return "Unknown!"
	}

	return speed + " Mbps"
}

// GetNetworkStats 获取网络统计信息（可选扩展）
func GetNetworkStats(ctx context.Context) ([]psnet.IOCountersStat, error) {
	return psnet.IOCountersWithContext(ctx, true)
}
