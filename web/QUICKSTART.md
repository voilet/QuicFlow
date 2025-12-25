# Web 前端快速开始

## 3 步启动前端

### 步骤 1: 安装依赖

```bash
cd web
npm install
```

预计安装时间：1-2 分钟

### 步骤 2: 启动开发服务器

```bash
npm run dev
```

启动成功后，控制台会显示：

```
  VITE v5.0.11  ready in 500 ms

  ➜  Local:   http://localhost:3000/
  ➜  Network: use --host to expose
```

### 步骤 3: 打开浏览器

访问：http://localhost:3000

## 页面预览

### 客户端管理页面

![客户端管理](./screenshots/client-list.png)

**功能**：
- 📊 实时统计面板
- 📋 客户端列表
- ⚡ 快速操作（下发命令、查看历史）

---

### 命令下发页面

![命令下发](./screenshots/command-send.png)

**功能**：
- 📝 命令表单编辑
- 📚 预设命令模板
- 🕒 最近下发命令
- ✅ 实时执行结果

---

### 命令历史页面

![命令历史](./screenshots/command-history.png)

**功能**：
- 🔍 多维度筛选
- 📊 分页显示
- 🔄 失败重试
- 📱 详情展开

## 测试数据

如果没有客户端连接，可以先启动后端服务和一个测试客户端：

```bash
# 终端1 - 启动服务器
cd ..
go run cmd/server/main.go

# 终端2 - 启动客户端
go run examples/command/client_example.go -id test-client-001

# 终端3 - 启动前端
cd web
npm run dev
```

## 常用操作

### 下发重启命令

1. 进入"命令下发"页面
2. 选择客户端：`test-client-001`
3. 命令类型选择：`重启服务`
4. 使用默认参数或点击"使用此模板"
5. 点击"下发命令"
6. 查看执行结果

### 查看命令历史

1. 进入"命令历史"页面
2. 选择客户端ID筛选
3. 点击"查询"
4. 展开查看详细信息

### 重试失败命令

1. 在"命令历史"页面找到失败的命令
2. 点击"重试"按钮
3. 确认后命令将重新下发

## 开发技巧

### 热更新

保存代码后，页面会自动刷新。无需手动刷新浏览器。

### 调试

打开浏览器开发者工具（F12），查看：
- Network 标签：查看 API 请求
- Console 标签：查看日志输出

### 修改 API 地址

如果后端运行在不同端口，修改 `vite.config.js`：

```javascript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:YOUR_PORT',  // 修改这里
      changeOrigin: true
    }
  }
}
```

## 生产构建

```bash
npm run build
```

构建完成后，`dist/` 目录包含所有静态文件。

### 预览构建结果

```bash
npm run preview
```

## 问题排查

### 端口被占用

错误信息：`Port 3000 is already in use`

解决方法：
```bash
# 修改 vite.config.js 中的端口
server: {
  port: 3001  # 改为其他端口
}
```

### 无法连接后端

错误信息：`Network Error` 或 `404`

检查：
1. 后端是否启动（`localhost:8475`）
2. 代理配置是否正确
3. 后端是否启用了 CORS

### 页面空白

解决方法：
1. 清除浏览器缓存
2. 检查浏览器控制台错误
3. 重新安装依赖：`rm -rf node_modules && npm install`

## 下一步

- 📖 阅读 [完整文档](README.md)
- 🎨 自定义主题和样式
- 🔌 集成更多功能

## 需要帮助？

- [前端 README](README.md)
- [后端文档](../QUICKSTART_COMMAND.md)
- [完整系统文档](../COMMAND_SYSTEM.md)
