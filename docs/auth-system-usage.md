# 登录验证权限系统使用文档

## 1. 项目结构

```
pkg/auth/
├── models/              # 数据模型
│   ├── sys_user.go
│   ├── sys_authority.go
│   ├── sys_base_menu.go
│   ├── sys_api.go
│   ├── sys_jwt_blacklist.go
│   ├── sys_operation_record.go
│   └── models.go
├── middleware/          # 中间件
│   ├── jwt.go          # JWT 认证中间件
│   ├── casbin.go      # Casbin 权限中间件
│   └── gin_casbin.go  # Gin Casbin 包装器
├── service/            # 服务层
│   ├── user_service.go
│   ├── authority_service.go
│   └── menu_service.go
├── api/               # API 层
│   └── auth_api.go
├── init/              # 初始化
│   └── seed.go
└── integration.go     # 集成入口
```

## 2. 后端集成

### 2.1 初始化权限系统

```go
import (
    "github.com/voilet/quic-flow/pkg/auth"
)

// 创建权限管理器
authManager, err := auth.NewManager(db, &auth.Config{
    JWTSigningKey: "your-secret-key",
    JWTExpires:    "7d",
    BufferTime:    "1h",
    RouterPrefix:  "/api",
})
if err != nil {
    log.Fatal("Failed to create auth manager:", err)
}

// 初始化数据库表和种子数据
if err := authManager.Initialize(); err != nil {
    log.Fatal("Failed to initialize auth:", err)
}
```

### 2.2 注册路由

```go
// 在 HTTP 服务器中注册权限路由
authGroup := router.Group("/api")

// 注册公开路由（登录等不需要认证）
authManager.RegisterPublicRoutes(authGroup)

// 注册需要认证的路由
authManager.RegisterRoutes(authGroup)
```

### 2.3 保护现有路由

```go
// 获取 JWT 中间件
jwtMiddleware := authManager.GetJWTMiddleware()

// 应用到现有路由组
apiGroup := router.Group("/api")
apiGroup.Use(jwtMiddleware.Handler())
{
    apiGroup.GET("/clients", handleListClients)
    apiGroup.POST("/command", handleSendCommand)
    // ... 其他路由
}
```

## 3. 数据库表

系统会自动创建以下表：

| 表名 | 说明 |
|------|------|
| sys_users | 用户表 |
| sys_authorities | 角色表 |
| sys_user_authorities | 用户角色关联表 |
| sys_base_menus | 菜单表 |
| sys_base_menu_parameters | 菜单参数表 |
| sys_authority_menus | 角色菜单关联表 |
| sys_apis | API表 |
| sys_jwt_blacklists | Token黑名单 |
| sys_operation_records | 操作记录 |
| casbin_rule | Casbin策略表 |

## 4. 默认账户

```
用户名: admin
密码: admin123
角色: 超级管理员
```

**首次使用后请立即修改密码！**

## 5. 前端使用

### 5.1 登录

```javascript
import { api, setToken } from '@/api'

// 登录
const res = await api.login({
  username: 'admin',
  password: 'admin123'
})

if (res.code === 0) {
  setToken(res.data.token)
  localStorage.setItem('user', JSON.stringify(res.data.user))
}
```

### 5.2 权限指令

```vue
<template>
  <!-- 单个权限检查 -->
  <el-button v-auth="'user:create'">创建用户</el-button>

  <!-- 多个权限检查（满足其一） -->
  <el-button v-auths="['user:update', 'user:delete']">操作</el-button>

  <!-- 多个权限检查（全部满足） -->
  <el-button v-auths-all="['admin', 'super_admin']">超级操作</el-button>
</template>
```

### 5.3 路由守卫

路由守卫已自动配置，未登录用户会自动跳转到登录页面。

## 6. API 接口

### 6.1 认证相关

| 接口 | 方法 | 说明 |
|------|------|------|
| /api/base/login | POST | 用户登录 |
| /api/user/logout | POST | 用户登出 |
| /api/user/info | GET | 获取当前用户信息 |
| /api/user/password | PUT | 修改密码 |

### 6.2 用户管理（管理员）

| 接口 | 方法 | 说明 |
|------|------|------|
| /api/user/list | GET | 用户列表 |
| /api/user/create | POST | 创建用户 |
| /api/user/update | PUT | 更新用户 |
| /api/user/delete | DELETE | 删除用户 |
| /api/user/reset-password | PUT | 重置密码 |

### 6.3 角色管理（管理员）

| 接口 | 方法 | 说明 |
|------|------|------|
| /api/authority/list | GET | 角色列表 |
| /api/authority/create | POST | 创建角色 |
| /api/authority/update | PUT | 更新角色 |
| /api/authority/delete | DELETE | 删除角色 |
| /api/authority/copy | POST | 复制角色 |

### 6.4 菜单管理（管理员）

| 接口 | 方法 | 说明 |
|------|------|------|
| /api/menu/list | GET | 菜单列表 |
| /api/menu/create | POST | 创建菜单 |
| /api/menu/update | PUT | 更新菜单 |
| /api/menu/delete | DELETE | 删除菜单 |
| /api/menu/by-authority | GET | 获取角色菜单 |
| /api/menu/set-authority | POST | 设置角色菜单 |

## 7. 安全建议

1. **修改默认密码**：首次部署后立即修改 admin 密码
2. **更换 JWT 密钥**：在生产环境使用随机生成的密钥
3. **启用 HTTPS**：生产环境必须使用 HTTPS
4. **定期备份**：定期备份数据库
5. **审计日志**：定期检查操作记录

## 8. 扩展开发

### 8.1 添加新的 API 权限

```go
// 在 Casbin 中添加策略
casbinMiddleware.AddPolicy("1", "/api/new-api", "GET")
casbinMiddleware.AddPolicy("1", "/api/new-api", "POST")
```

### 8.2 添加新的菜单

```go
// 通过 API 或直接创建
menu := &models.SysBaseMenu{
    ParentId:  0,
    Path:      "/new-page",
    Name:      "NewPage",
    Title:     "新页面",
    Icon:      "Star",
    Component: "views/NewPage.vue",
    Sort:      10,
}
db.Create(menu)
```

## 9. 故障排查

### 问题：登录后返回 401
- 检查 Token 是否正确设置在请求头 `x-token` 中
- 检查 JWT 配置是否正确

### 问题：API 返回 403
- 检查用户角色是否有该 API 的权限
- 检查 Casbin 策略是否正确配置

### 问题：菜单不显示
- 检查角色是否分配了菜单权限
- 检查 sys_authority_menus 表中是否有对应记录
