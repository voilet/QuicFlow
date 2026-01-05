package profiling

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/gin-gonic/gin"
)

// StandardHandler 标准的 pprof HTTP 处理器
// 兼容 go tool pprof 和 net/http/pprof
type StandardHandler struct {
	// 可以在这里添加额外的配置
}

// NewStandardHandler 创建标准 pprof 处理器
func NewStandardHandler() *StandardHandler {
	return &StandardHandler{}
}

// RegisterRoutes 注册标准的 pprof 路由
// 这些端点与 go tool pprof 兼容
func (h *StandardHandler) RegisterRoutes(r gin.IRouter) {
	pprofGroup := r.Group("/debug/pprof")
	{
		// 标准 pprof 端点
		pprofGroup.GET("/", h.Index)           // pprof 索引
		pprofGroup.GET("/cmdline", h.Cmdline)  // 命令行参数
		pprofGroup.GET("/profile", h.Profile)  // CPU profile (支持 seconds 参数)
		pprofGroup.GET("/heap", h.Heap)        // 堆内存 profile
		pprofGroup.GET("/goroutine", h.Goroutine) // Goroutine profile
		pprofGroup.GET("/block", h.Block)       // 阻塞 profile
		pprofGroup.GET("/mutex", h.Mutex)       // 互斥锁 profile
		pprofGroup.GET("/allocs", h.Allocs)     // 内存分配 profile
		pprofGroup.GET("/threadcreate", h.ThreadCreate) // 线程创建 profile

		// 符号表
		pprofGroup.GET("/symbol", h.Symbol)     // 符号查找
		pprofGroup.POST("/symbol", h.Symbol)

		// 运行时统计
		pprofGroup.GET("/heap/bytes", h.HeapBytes)
		pprofGroup.GET("/heap/objects", h.HeapObjects)
		pprofGroup.GET("/allocs/bytes", h.AllocsBytes)
		pprofGroup.GET("/allocs/objects", h.AllocsObjects)
		pprofGroup.GET("/goroutine/count", h.GoroutineCount)
		pprofGroup.GET("/threadcreate/count", h.ThreadCreateCount)
		pprofGroup.GET("/block/count", h.BlockCount)
		pprofGroup.GET("/mutex/count", h.MutexCount)

		// 可视化（SVG）
		pprofGroup.GET("/svg", h.SVG)
	}
}

// Index pprof 索引页面
func (h *StandardHandler) Index(c *gin.Context) {
	var profiles []string
	for _, p := range pprof.Profiles() {
		profiles = append(profiles, p.Name())
	}

	html := `<!DOCTYPE html>
<html>
<head>
	<title>/debug/pprof/</title>
</head>
<body>
	<h1>Profiles</h1>
	<table>`
	for _, p := range profiles {
		html += fmt.Sprintf("<tr><td><a href='%s'>%s</a></td></tr>", p, p)
	}
	html += `
</table>
	<h1>Runtime Statistics</h1>
	<table>
		<tr><td><a href="/debug/pprof/heap">heap</a></td></tr>
		<tr><td><a href="/debug/pprof/goroutine">goroutine</a></td></tr>
		<tr><td><a href="/debug/pprof/threadcreate">threadcreate</a></td></tr>
		<tr><td><a href="/debug/pprof/block">block</a></td></tr>
		<tr><td><a href="/debug/pprof/mutex">mutex</a></td></tr>
		<tr><td><a href="/debug/pprof/allocs">allocs</a></td></tr>
	</table>
</body>
</html>`

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// Cmdline 返回命令行参数
func (h *StandardHandler) Cmdline(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%s\n", "quic-server") // 可以从 os.Args 获取
}

// Profile CPU profile
// 支持 seconds 参数：?seconds=30
func (h *StandardHandler) Profile(c *gin.Context) {
	// 解析参数
	seconds := c.DefaultQuery("seconds", "30")
	sec, err := parseInt(seconds)
	if err != nil || sec < 1 {
		sec = 30
	} else if sec > 3600 {
		sec = 3600 // 最多1小时
	}

	// 设置 HTTP 头
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="profile"`)

	// 获取响应写入器
	w := c.Writer

	// CPU 采集
	if err := pprof.StartCPUProfile(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 定期刷新缓冲区
	flush(w, time.Second/2)

	// 等待指定时间
	sleep(sec)

	pprof.StopCPUProfile()
}

// Heap 堆内存 profile
// GET /debug/pprof/heap?debug=1
func (h *StandardHandler) Heap(c *gin.Context) {
	debug := c.Query("debug")
	if debug == "1" {
		h.writeHeapProfileDebug(c.Writer)
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="heap"`)

	w := c.Writer
	if err := pprof.WriteHeapProfile(w); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
	}
}

// Goroutine Goroutine profile
// GET /debug/pprof/goroutine?debug=1
func (h *StandardHandler) Goroutine(c *gin.Context) {
	debug := c.Query("debug")
	if debug == "1" {
		h.writeGoroutineProfileDebug(c.Writer)
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="goroutine"`)

	p := pprof.Lookup("goroutine")
	if p == nil {
		http.Error(c.Writer, "goroutine profile not found", http.StatusNotFound)
		return
	}
	p.WriteTo(c.Writer, 0)
}

// Block 阻塞 profile
// GET /debug/pprof/block?debug=1
func (h *StandardHandler) Block(c *gin.Context) {
	debug := c.Query("debug")
	if debug == "1" {
		h.writeBlockProfileDebug(c.Writer)
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="block"`)

	p := pprof.Lookup("block")
	if p == nil {
		http.Error(c.Writer, "block profile not enabled (set runtime.SetBlockProfileRate)", http.StatusNotFound)
		return
	}
	p.WriteTo(c.Writer, 0)
}

// Mutex 互斥锁 profile
// GET /debug/pprof/mutex?debug=1
func (h *StandardHandler) Mutex(c *gin.Context) {
	debug := c.Query("debug")
	if debug == "1" {
		h.writeMutexProfileDebug(c.Writer)
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="mutex"`)

	p := pprof.Lookup("mutex")
	if p == nil {
		http.Error(c.Writer, "mutex profile not enabled (set runtime.SetMutexProfileFraction)", http.StatusNotFound)
		return
	}
	p.WriteTo(c.Writer, 0)
}

// Allocs 内存分配 profile
func (h *StandardHandler) Allocs(c *gin.Context) {
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="allocs"`)

	p := pprof.Lookup("allocs")
	if p == nil {
		http.Error(c.Writer, "allocs profile not found", http.StatusNotFound)
		return
	}
	p.WriteTo(c.Writer, 0)
}

// ThreadCreate 线程创建 profile
func (h *StandardHandler) ThreadCreate(c *gin.Context) {
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="threadcreate"`)

	p := pprof.Lookup("threadcreate")
	if p == nil {
		http.Error(c.Writer, "threadcreate profile not found", http.StatusNotFound)
		return
	}
	p.WriteTo(c.Writer, 0)
}

// Symbol 符号查找
func (h *StandardHandler) Symbol(c *gin.Context) {
	// 简化实现 - 返回空符号表
	// 完整实现需要解析二进制文件的符号表
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "num_symbols: 0\n")
}

// SVG 生成火焰图 SVG
func (h *StandardHandler) SVG(c *gin.Context) {
	// 需要解析当前的 profile 并生成 SVG
	// 这里先返回一个简单的响应
	c.String(http.StatusOK, "SVG generation requires pprof tool. Use: go tool pprof -http=:8080 http://localhost:8475/debug/pprof/profile")
}

// GoroutineCount goroutine 数量
func (h *StandardHandler) GoroutineCount(c *gin.Context) {
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%d\n", runtime.NumGoroutine())
}

// ThreadCreateCount 线程创建数量
func (h *StandardHandler) ThreadCreateCount(c *gin.Context) {
	// 简化实现
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "0\n")
}

// BlockCount 阻塞数量
func (h *StandardHandler) BlockCount(c *gin.Context) {
	// 需要从 block profile 获取
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "0\n")
}

// MutexCount 互斥锁数量
func (h *StandardHandler) MutexCount(c *gin.Context) {
	// 需要从 mutex profile 获取
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "0\n")
}

// HeapBytes 堆内存字节数
func (h *StandardHandler) HeapBytes(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%d\n", m.HeapAlloc)
}

// HeapObjects 堆对象数量
func (h *StandardHandler) HeapObjects(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%d\n", m.HeapObjects)
}

// AllocsBytes 累计分配字节数
func (h *StandardHandler) AllocsBytes(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%d\n", m.TotalAlloc)
}

// AllocsObjects 累计分配对象数量
func (h *StandardHandler) AllocsObjects(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, "%d\n", m.Mallocs-m.Frees)
}

// 辅助函数

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func sleep(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
}

func flush(w io.Writer, d time.Duration) {
	if f, ok := w.(http.Flusher); ok {
		go func() {
			for {
				time.Sleep(d)
				f.Flush()
			}
		}()
	}
}

// Debug 格式输出

func (h *StandardHandler) writeHeapProfileDebug(w io.Writer) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Fprintf(w, "heap profile: %d: %d [%d: %d] @ heap/%d\n",
		1, m.HeapAlloc, 1, m.HeapAlloc, 1)
	fmt.Fprintf(w, "# runtime.MemStats\n")
	fmt.Fprintf(w, "# Alloc = %d\n", m.Alloc)
	fmt.Fprintf(w, "# TotalAlloc = %d\n", m.TotalAlloc)
	fmt.Fprintf(w, "# Sys = %d\n", m.Sys)
	fmt.Fprintf(w, "# Lookups = %d\n", m.Lookups)
	fmt.Fprintf(w, "# Mallocs = %d\n", m.Mallocs)
	fmt.Fprintf(w, "# Frees = %d\n", m.Frees)
	fmt.Fprintf(w, "# HeapAlloc = %d\n", m.HeapAlloc)
	fmt.Fprintf(w, "# HeapSys = %d\n", m.HeapSys)
	fmt.Fprintf(w, "# HeapIdle = %d\n", m.HeapIdle)
	fmt.Fprintf(w, "# HeapInuse = %d\n", m.HeapInuse)
	fmt.Fprintf(w, "# HeapReleased = %d\n", m.HeapReleased)
	fmt.Fprintf(w, "# HeapObjects = %d\n", m.HeapObjects)
	fmt.Fprintf(w, "# Stack = %d / %d\n", m.StackInuse, m.StackSys)
	fmt.Fprintf(w, "# MSpan = %d / %d\n", m.MSpanInuse, m.MSpanSys)
	fmt.Fprintf(w, "# MCache = %d / %d\n", m.MCacheInuse, m.MCacheSys)
	fmt.Fprintf(w, "# BuckHashSys = %d\n", m.BuckHashSys)
	fmt.Fprintf(w, "# GCSys = %d\n", m.GCSys)
	fmt.Fprintf(w, "# NextGC = %d\n", m.NextGC)
	fmt.Fprintf(w, "# LastGC = %d\n", m.LastGC)
	fmt.Fprintf(w, "# PauseNs = %d\n", m.PauseTotalNs)
	fmt.Fprintf(w, "# NumGC = %d\n", m.NumGC)
	fmt.Fprintf(w, "# NumForcedGC = %d\n", m.NumForcedGC)
	fmt.Fprintf(w, "# GCCPUFraction = %d\n", m.GCCPUFraction)
	fmt.Fprintf(w, "# EnableGC = %v\n", m.EnableGC)
	fmt.Fprintf(w, "# DebugGC = %v\n", m.DebugGC)
}

func (h *StandardHandler) writeGoroutineProfileDebug(w io.Writer) {
	p := pprof.Lookup("goroutine")
	if p == nil {
		fmt.Fprintf(w, "goroutine profile: not found\n")
		return
	}

	// 写入 debug 格式
	p.WriteTo(w, 1)
}

func (h *StandardHandler) writeBlockProfileDebug(w io.Writer) {
	p := pprof.Lookup("block")
	if p == nil {
		fmt.Fprintf(w, "block profile: not enabled (use runtime.SetBlockProfileRate)\n")
		return
	}

	p.WriteTo(w, 1)
}

func (h *StandardHandler) writeMutexProfileDebug(w io.Writer) {
	p := pprof.Lookup("mutex")
	if p == nil {
		fmt.Fprintf(w, "mutex profile: not enabled (use runtime.SetMutexProfileFraction)\n")
		return
	}

	p.WriteTo(w, 1)
}
