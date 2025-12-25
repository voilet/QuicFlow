# Next Steps - 下一步计划

本文档提供了项目后续开发的详细路线图和具体行动步骤。

## 🎯 当前状态

**项目完成度**: 约 85%

**核心功能**: ✅ 完成
- QUIC 传输层
- 自动重连机制
- 心跳和会话管理
- 消息路由和分发
- Promise/回调机制
- 完整监控系统
- Prometheus 集成

**待完善**: 示例程序、文档、测试

---

## 🚀 短期目标 (1-2 周)

### Week 1: 示例程序和文档

#### Day 1-2: Echo 示例
```bash
examples/echo/
├── server.go       # Echo 服务器实现
├── client.go       # Echo 客户端实现
└── README.md       # 使用说明
```

**实现要点**:
1. 服务器：
   - 注册 Echo Handler
   - 接收消息并回显
   - 记录处理日志

2. 客户端：
   - 连接服务器
   - 发送测试消息
   - 验证回显内容

3. 运行示例：
   ```bash
   # Terminal 1: 启动服务器
   go run examples/echo/server.go

   # Terminal 2: 启动客户端
   go run examples/echo/client.go
   ```

**预计工时**: 4 小时

#### Day 3-4: 广播示例
```bash
examples/broadcast/
├── server.go       # 广播服务器
├── client.go       # 订阅客户端
└── README.md       # 使用说明
```

**实现要点**:
1. 服务器：
   - 定期广播事件 (如时间戳、状态更新)
   - 支持客户端订阅
   - 统计广播成功率

2. 客户端：
   - 连接并订阅广播
   - 接收并显示广播消息
   - 处理断线重连

3. 场景测试：
   - 启动 3-5 个客户端
   - 验证所有客户端都能收到广播

**预计工时**: 4 小时

#### Day 5: 回调示例
```bash
examples/callback/
├── server.go       # 支持 Ack 的服务器
├── client.go       # 使用 Promise 的客户端
└── README.md       # 使用说明
```

**实现要点**:
1. 演示 AsyncSend + Promise 模式
2. 展示超时处理
3. 对比同步 vs 异步发送

**预计工时**: 4 小时

### Week 2: 文档和配置

#### Day 1-2: API 文档注释
- 为所有公共 API 添加 Go doc 注释
- 确保 `go doc` 输出清晰
- 生成文档：`godoc -http=:6060`

**检查清单**:
```bash
# 需要添加文档的包
pkg/callback/          # Promise, PromiseManager
pkg/dispatcher/        # Dispatcher, MessageHandler
pkg/monitoring/        # Metrics, EventHooks, Logger
pkg/transport/server/  # Server, ServerConfig
pkg/transport/client/  # Client, ClientConfig
pkg/session/           # SessionManager, ClientSession
```

**预计工时**: 6 小时

#### Day 3: 配置验证完善
- 完善 ServerConfig.Validate()
- 完善 ClientConfig.Validate()
- 添加参数范围检查
- 添加错误提示

**示例**:
```go
func (c *ServerConfig) Validate() error {
    if c.TLSCertFile == "" {
        return fmt.Errorf("TLSCertFile is required")
    }
    if c.MaxClients <= 0 {
        return fmt.Errorf("MaxClients must be positive, got %d", c.MaxClients)
    }
    // ... 更多验证
}
```

**预计工时**: 2 小时

#### Day 4: 错误处理标准化
- 审查所有 error 返回
- 确保使用 pkg/errors 中的类型
- 添加错误上下文信息

**检查**:
```bash
# 搜索可能需要改进的错误处理
grep -r "errors.New" pkg/
grep -r "fmt.Errorf" pkg/
```

**预计工时**: 3 小时

#### Day 5: 创建 quickstart 脚本
```bash
scripts/test-quickstart.sh
```

**功能**:
- 自动运行 Echo 示例
- 验证输出是否正确
- 测试重连场景

**预计工时**: 3 小时

---

## 🎓 中期目标 (2-4 周)

### 测试覆盖 (Week 3)

#### 单元测试
**目标**: 70%+ 代码覆盖率

**优先级模块**:
1. `pkg/callback` - Promise 管理
2. `pkg/dispatcher` - 消息路由
3. `pkg/session` - 会话管理
4. `pkg/monitoring` - 指标采集

**示例**:
```go
// pkg/callback/promise_test.go
func TestPromise_Complete(t *testing.T) {
    p := NewPromise("msg-123", 5*time.Second, nil)

    ack := &protocol.AckMessage{
        MsgId: "msg-123",
        Status: protocol.AckStatus_ACK_STATUS_SUCCESS,
    }

    assert.True(t, p.Complete(ack))
    assert.False(t, p.Complete(ack)) // 第二次应该失败
}
```

**工具**:
```bash
# 运行测试
go test ./pkg/... -v

# 生成覆盖率报告
go test ./pkg/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**预计工时**: 12 小时

#### 集成测试
**场景**:
1. 连接和重连
2. 消息传输（单播、广播）
3. 心跳超时和清理
4. Promise 超时处理

**预计工时**: 8 小时

### 性能优化 (Week 4)

#### 性能分析
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Memory profiling
go test -memprofile=mem.prof -bench=.

# 分析结果
go tool pprof cpu.prof
```

**检查点**:
- [ ] 内存分配热点
- [ ] CPU 使用热点
- [ ] Goroutine 泄漏
- [ ] 锁竞争

**预计工时**: 6 小时

#### 优化实施
**可能的优化点**:
1. 对象池 (sync.Pool) 用于频繁分配的对象
2. 减少不必要的内存拷贝
3. 优化锁粒度
4. 批量处理消息

**预计工时**: 8 小时

---

## 🏢 长期目标 (1-3 个月)

### CI/CD 集成

**GitHub Actions 工作流**:
```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: go test ./...
      - run: golangci-lint run
```

**预计工时**: 4 小时

### 完整文档

#### API 文档
- 所有公共 API 的详细说明
- 参数说明和返回值
- 使用示例

#### 架构文档
- 系统架构图
- 模块交互图
- 数据流图
- 设计决策记录 (ADR)

#### 部署文档
- 生产环境配置建议
- 容器化部署 (Docker)
- Kubernetes 部署示例

**预计工时**: 20 小时

### 功能增强

#### WebSocket 支持
- 为 Web 浏览器客户端提供支持
- WebSocket to QUIC 网关

**预计工时**: 12 小时

#### 消息压缩
- 可选的消息压缩 (gzip/zstd)
- 配置阈值（如 >1KB 才压缩）

**预计工时**: 6 小时

#### 认证和授权
- 客户端认证（Token-based）
- 权限管理（读/写/广播权限）

**预计工时**: 10 小时

---

## 📋 立即可执行的任务

如果你现在就想开始，这里是按优先级排序的任务清单：

### 🔥 今天就可以做 (2-4 小时)

1. **创建 Echo 示例** (最高优先级)
   ```bash
   mkdir -p examples/echo
   touch examples/echo/{server.go,client.go,README.md}
   ```

2. **完善配置验证**
   - 编辑 `pkg/transport/server/config.go`
   - 编辑 `pkg/transport/client/config.go`
   - 添加完整的 Validate() 逻辑

3. **添加基础单元测试**
   - 从 `pkg/callback/promise_test.go` 开始
   - 测试 Promise 的基本功能

### 📅 本周可完成 (10-15 小时)

4. **创建广播示例**
5. **创建回调示例**
6. **添加 API 文档注释** (优先处理常用 API)
7. **创建 quickstart 测试脚本**

### 📆 下周可完成 (15-20 小时)

8. **完整的单元测试覆盖**
9. **集成测试场景**
10. **性能分析和优化**

---

## 🎯 推荐的开发顺序

如果你是第一次继续开发，建议按以下顺序：

```
1️⃣ Echo 示例 (理解系统如何工作)
   ↓
2️⃣ 配置验证 (提升健壮性)
   ↓
3️⃣ 单元测试 (确保质量)
   ↓
4️⃣ 广播和回调示例 (展示高级功能)
   ↓
5️⃣ API 文档注释 (方便他人使用)
   ↓
6️⃣ 性能分析和优化 (生产就绪)
```

---

## 💡 开发建议

### 代码规范
- 遵循 Go 官方代码风格
- 使用 `gofmt` 格式化代码
- 运行 `golangci-lint` 检查

### 提交规范
```
feat: 添加 Echo 示例
fix: 修复 Promise 超时处理
docs: 更新 API 文档
test: 添加 Session 单元测试
perf: 优化消息编码性能
refactor: 重构 Dispatcher 逻辑
```

### 测试策略
- 单元测试：测试单个函数/方法
- 集成测试：测试模块间交互
- 端到端测试：测试完整场景

### 性能基准
- 目标：10,000+ 并发连接
- 延迟：P99 < 50ms (局域网)
- 内存：< 50KB 每连接
- 吞吐量：10,000+ msg/s

---

## 📞 需要帮助？

如果在开发过程中遇到问题：

1. **查看文档**: docs/ 目录
2. **查看示例**: examples/ 目录
3. **运行测试**: `./scripts/test-mvp.sh`
4. **检查日志**: /tmp/quic-*.log

---

## 🎉 最终目标

**3 个月后的理想状态**:

- ✅ 完整的示例程序（Echo, Broadcast, Callback）
- ✅ 70%+ 测试覆盖率
- ✅ 完整的 API 文档
- ✅ 性能优化完成
- ✅ CI/CD 集成
- ✅ Docker 镜像
- ✅ 生产部署文档
- ✅ 社区贡献指南

**结果**: 一个可以在生产环境中使用的、文档完善的、社区友好的 QUIC 通信框架。

---

最后更新：2025-12-24
