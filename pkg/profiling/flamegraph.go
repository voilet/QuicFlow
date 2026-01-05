package profiling

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

// stackKey 调用栈键（用于火焰图树构建）
type stackKey struct {
	function string
	file     string
}

// FlameGraphGenerator 火焰图生成器
type FlameGraphGenerator struct {
	config FlameGraphConfig
}

// NewFlameGraphGenerator 创建火焰图生成器
func NewFlameGraphGenerator(config FlameGraphConfig) *FlameGraphGenerator {
	if config.Width == 0 {
		config = DefaultFlameGraphConfig
	}
	return &FlameGraphGenerator{config: config}
}

// GenerateFromFile 从 pprof 文件生成火焰图
func (g *FlameGraphGenerator) GenerateFromFile(profilePath, outputPath string) error {
	// 读取 pprof 文件
	f, err := os.Open(profilePath)
	if err != nil {
		return fmt.Errorf("failed to open profile: %w", err)
	}
	defer f.Close()

	return g.GenerateFromReader(f, outputPath)
}

// GenerateFromReader 从 reader 生成火焰图
func (g *FlameGraphGenerator) GenerateFromReader(r io.Reader, outputPath string) error {
	// 解析 pprof 数据
	p, err := profile.Parse(r)
	if err != nil {
		return fmt.Errorf("failed to parse profile: %w", err)
	}

	// 生成火焰图 SVG
	svg, err := g.Generate(p)
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(outputPath, []byte(svg), 0644)
}

// Generate 生成火焰图 SVG
func (g *FlameGraphGenerator) Generate(p *profile.Profile) (string, error) {
	// 构建调用图树
	root := g.buildFlameGraphTree(p)

	// 计算每个节点的 x 坐标和宽度
	total := root.Value
	g.calculateLayout(root, 0, total)

	// 生成 SVG
	return g.generateSVG(root, total)
}

// buildFlameGraphTree 构建 flame graph 树
func (g *FlameGraphGenerator) buildFlameGraphTree(p *profile.Profile) *FlameGraphNode {
	// 按函数聚合
	children := make(map[stackKey]*FlameGraphNode)
	totalSamples := int64(0)

	for _, sample := range p.Sample {
		value := int64(0)
		for _, v := range sample.Value {
			value += v
		}
		if value == 0 {
			value = 1
		}
		totalSamples += value

		// 构建调用栈（从根到叶子）
		stack := make([]stackKey, 0, len(sample.Location))
		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			for _, line := range loc.Line {
				fn := line.Function
				if fn != nil {
					name := fn.Name
					if name == "" {
						name = fn.SystemName
					}
					if name == "" {
						name = "(unknown)"
					}
					stack = append(stack, stackKey{
						function: shortenName(name),
						file:     fn.Filename,
					})
				}
			}
		}

		// 添加到树中
		if len(stack) > 0 {
			addToTree(children, stack, 0, value)
		}
	}

	// 创建根节点
	rootChildren := make([]*FlameGraphNode, 0, len(children))
	for _, child := range children {
		rootChildren = append(rootChildren, child)
	}

	return &FlameGraphNode{
		Name:     "root",
		Value:    totalSamples,
		Children: rootChildren,
	}
}

// addToTree 递归添加节点到树
func addToTree(children map[stackKey]*FlameGraphNode, stack []stackKey, depth int, value int64) {
	if depth >= len(stack) {
		return
	}

	key := stack[depth]
	node, exists := children[key]
	if !exists {
		node = &FlameGraphNode{
			Name:  key.function,
			Value: 0,
		}
		children[key] = node
	}
	node.Value += value

	// 递归处理子节点
	if node.Children == nil {
		node.Children = make([]*FlameGraphNode, 0)
	}

	// 为子节点创建 map
	childMap := make(map[stackKey]*FlameGraphNode)
	for _, child := range node.Children {
		childKey := stackKey{function: child.Name}
		childMap[childKey] = child
	}

	// 如果还有下一层，递归
	if depth+1 < len(stack) {
		addToTree(childMap, stack, depth+1, value)
		// 更新 children
		node.Children = make([]*FlameGraphNode, 0, len(childMap))
		for _, child := range childMap {
			node.Children = append(node.Children, child)
		}
	}
}

// calculateLayout 计算布局（x 坐标和宽度）
func (g *FlameGraphGenerator) calculateLayout(node *FlameGraphNode, offset, total int64) {
	// 这里我们不需要实际存储坐标，因为我们在生成 SVG 时动态计算
	// 但我们需要确保子节点的值总和正确
	if len(node.Children) > 0 {
		sum := int64(0)
		for _, child := range node.Children {
			sum += child.Value
		}
		// 如果有舍入误差，调整
		if sum != node.Value {
			// 找到最大的子节点调整
			maxChild := node.Children[0]
			for _, child := range node.Children {
				if child.Value > maxChild.Value {
					maxChild = child
				}
			}
			maxChild.Value += (node.Value - sum)
		}
	}
}

// SVG 节点信息
type svgNode struct {
	name      string
	x         float64
	width     float64
	depth     int
	percentage float64
}

// generateSVG 生成 SVG
func (g *FlameGraphGenerator) generateSVG(root *FlameGraphNode, total int64) (string, error) {
	var buf bytes.Buffer

	// 收集所有节点并按深度分组
	depths := make(map[int][]*svgNode)
	maxDepth := g.collectNodes(root, 0, total, 0, depths)

	// SVG 头部
	width := g.config.Width
	height := (maxDepth + 1) * (g.config.Height + 1)

	buf.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">
<style>
	.func:hover { opacity: 0.8; }
	.func-text { font-family: monospace; font-size: %dpx; fill: #000; pointer-events: none; }
</style>
`, width, height, width, height, g.config.FontHeight))

	// 生成每个节点
	for depth := 0; depth <= maxDepth; depth++ {
		nodes := depths[depth]
		for _, node := range nodes {
			if node.width < g.config.MinWidth {
				continue
			}
			color := g.getColor(node.name, node.percentage)
			y := depth * (g.config.Height + 1)

			buf.WriteString(fmt.Sprintf(`<g class="func" transform="translate(%.1f,%d)">
  <title>%s (%.1f%%)</title>
  <rect x="0" y="0" width="%.1f" height="%d" fill="%s" rx="1" />
  <text x="2" y="%d" class="func-text">%s</text>
</g>
`, node.x, y, node.name, node.percentage, node.width, g.config.Height, color, g.config.FontHeight+2, truncateText(node.name, int(node.width/6))))

		}
	}

	buf.WriteString("</svg>")
	return buf.String(), nil
}

// collectNodes 收集所有节点
func (g *FlameGraphGenerator) collectNodes(node *FlameGraphNode, depth int, total int64, offset float64, depths map[int][]*svgNode) int {
	if total == 0 {
		total = 1
	}

	percentage := float64(node.Value) / float64(total) * 100
	width := float64(node.Value) / float64(total) * float64(g.config.Width)

	// 不添加 root 节点
	if node.Name != "root" {
		depths[depth] = append(depths[depth], &svgNode{
			name:       node.Name,
			x:          offset,
			width:      width,
			depth:      depth,
			percentage: percentage,
		})
	}

	maxDepth := depth
	if len(node.Children) > 0 {
		// 按值排序子节点
		children := make([]*FlameGraphNode, len(node.Children))
		copy(children, node.Children)
		sort.Slice(children, func(i, j int) bool {
			return children[i].Value > children[j].Value
		})

		childOffset := offset
		for _, child := range children {
			childWidth := float64(child.Value) / float64(total) * float64(g.config.Width)
			d := g.collectNodes(child, depth+1, total, childOffset, depths)
			if d > maxDepth {
				maxDepth = d
			}
			childOffset += childWidth
		}
	}

	return maxDepth
}

// getColor 获取颜色
func (g *FlameGraphGenerator) getColor(name string, percentage float64) string {
	switch g.config.ColorScheme {
	case "cool":
		return g.getCoolColor(name, percentage)
	case "rainbow":
		return g.getRainbowColor(name)
	default:
		return g.getWarmColor(name, percentage)
	}
}

// getWarmColor 暖色系（红色、橙色、黄色）
func (g *FlameGraphGenerator) getWarmColor(name string, percentage float64) string {
	// 基于 name 的哈希值生成色相
	hash := hashString(name)
	hue := 0 + (hash % 60) // 0-60: 红色到黄色

	// 饱和度基于百分比（热点更饱和）
	saturation := 70 + (hash % 30)
	if percentage > 20 {
		saturation = 90
	}

	// 亮度
	lightness := 50 + (hash % 20)

	return fmt.Sprintf("hsl(%d, %d%%, %d%%)", hue, saturation, lightness)
}

// getCoolColor 冷色系（蓝色、青色、紫色）
func (g *FlameGraphGenerator) getCoolColor(name string, percentage float64) string {
	hash := hashString(name)
	hue := 180 + (hash % 100) // 180-280: 青色到紫色

	saturation := 60 + (hash % 30)
	if percentage > 20 {
		saturation = 85
	}

	lightness := 45 + (hash % 25)

	return fmt.Sprintf("hsl(%d, %d%%, %d%%)", hue, saturation, lightness)
}

// getRainbowColor 彩虹色
func (g *FlameGraphGenerator) getRainbowColor(name string) string {
	hash := hashString(name)
	hue := hash % 360
	return fmt.Sprintf("hsl(%d, 70%%, 55%%)", hue)
}

// hashString 计算字符串哈希
func hashString(s string) int {
	hash := 0
	for i, c := range s {
		hash = hash*31 + int(c) + i
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// shortenName 缩短函数名
func shortenName(name string) string {
	// 移除常见的包前缀
	name = strings.TrimPrefix(name, "github.com/")
	name = strings.TrimPrefix(name, "golang.org/")
	name = strings.TrimPrefix(name, "internal/")
	name = strings.TrimPrefix(name, "vendor/")

	// 如果名字太长，截断
	if len(name) > 80 {
		// 尝试保留最后几个部分
		parts := strings.Split(name, "/")
		if len(parts) > 3 {
			name = strings.Join(parts[len(parts)-2:], "/")
		} else {
			name = "..." + name[len(name)-77:]
		}
	}

	return name
}

// truncateText 截断文本以适应宽度
func truncateText(text string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	return string(runes[:maxChars]) + ".."
}

// GenerateFlameGraph 为指定的 profile 生成火焰图
func (p *Profiler) GenerateFlameGraph(profileID string) (string, error) {
	profile, err := p.GetProfile(profileID)
	if err != nil {
		return "", fmt.Errorf("profile not found: %w", err)
	}

	if profile.Status != StatusCompleted {
		return "", fmt.Errorf("profile is not completed")
	}

	// 生成火焰图文件名
	flameName := fmt.Sprintf("flame-%s.svg", profileID[:8])
	flamePath := filepath.Join(p.storeDir, flameName)

	// 检查是否已存在
	if _, err := os.Stat(flamePath); err == nil {
		// 已存在，更新记录
		if profile.FlamePath != flamePath {
			p.SetFlameGraphPath(profileID, flamePath)
		}
		return flamePath, nil
	}

	// 生成火焰图
	generator := NewFlameGraphGenerator(DefaultFlameGraphConfig)
	if err := generator.GenerateFromFile(profile.FilePath, flamePath); err != nil {
		return "", fmt.Errorf("failed to generate flame graph: %w", err)
	}

	// 更新记录
	if err := p.SetFlameGraphPath(profileID, flamePath); err != nil {
		os.Remove(flamePath)
		return "", fmt.Errorf("failed to update profile: %w", err)
	}

	return flamePath, nil
}
