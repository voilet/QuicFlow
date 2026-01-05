# 登录验证权限系统需求文档

## 1. 项目概述

### 1.1 背景
当前 quic-flow 项目缺少完整的用户认证和权限管理系统，所有 API 接口均无身份验证，存在严重的安全风险。本需求参考 gin-vue-admin 的权限设计，为项目添加完善的 RBAC（基于角色的访问控制）权限系统。

### 1.2 参考设计
本项目权限系统参考 [gin-vue-admin](https://github.com/flipped-aurora/gin-vue-admin) 的成熟架构：
- JWT 认证机制
- Casbin RBAC 权限控制
- 前后端分离的权限验证
- 细粒度的按钮级权限控制

## 2. 功能需求

### 2.1 用户认证
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 用户登录 | 用户名/密码登录，返回 JWT Token | P0 |
| Token 刷新 | 支持自动刷新过期 Token | P0 |
| 多点登录控制 | 支持单点/多点登录模式切换 | P1 |
| 登出 | 主动登出并将 Token 加入黑名单 | P0 |
| 密码加密 | 使用 bcrypt 加密存储密码 | P0 |

### 2.2 用户管理
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 用户列表 | 分页查询用户列表 | P0 |
| 创建用户 | 创建新用户并分配角色 | P0 |
| 编辑用户 | 修改用户信息和角色 | P0 |
| 删除用户 | 软删除用户 | P0 |
| 冻结用户 | 启用/禁用用户账户 | P1 |
| 修改密码 | 管理员重置用户密码 | P0 |

### 2.3 角色管理
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 角色列表 | 查询所有角色 | P0 |
| 创建角色 | 创建新角色并分配权限 | P0 |
| 编辑角色 | 修改角色和权限 | P0 |
| 删除角色 | 删除角色（需检查用户关联） | P0 |
| 角色层级 | 支持父角色继承（可选） | P2 |

### 2.4 API 权限管理
| 功能 | 描述 | 优先级 |
|------|------|--------|
| API 列表 | 查询所有已注册 API | P0 |
| 分配 API 权限 | 为角色分配 API 访问权限 | P0 |
| Casbin 策略管理 | 基于 Casbin 的策略存储和执行 | P0 |
| 路径通配符 | 支持 keyMatch2 通配符匹配 | P1 |

### 2.5 菜单权限管理
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 菜单列表 | 树形结构菜单 | P0 |
| 创建菜单 | 创建菜单并分配给角色 | P0 |
| 编辑菜单 | 修改菜单属性 | P0 |
| 删除菜单 | 删除菜单（检查子菜单） | P0 |
| 动态路由 | 前端根据权限动态加载路由 | P0 |

### 2.6 按钮级权限
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 按钮权限定义 | 定义页面按钮权限点 | P1 |
| 角色按钮分配 | 为角色分配按钮权限 | P1 |
| 前端指令 | v-auth 指令控制按钮显示 | P1 |

### 2.7 审计日志
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 登录日志 | 记录用户登录行为 | P1 |
| 操作日志 | 记录 API 调用和操作 | P1 |
| 日志查询 | 按用户/时间查询日志 | P1 |

## 3. 技术架构

### 3.1 技术栈
| 组件 | 技术 |
|------|------|
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | PostgreSQL |
| 认证 | JWT (golang-jwt/jwt) |
| 权限 | Casbin |
| 密码加密 | bcrypt |

### 3.2 数据库设计

#### 3.2.1 sys_users (用户表)
```sql
CREATE TABLE sys_users (
    id              BIGSERIAL PRIMARY KEY,
    created_at      TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP,
    uuid            UUID             NOT NULL UNIQUE,
    username        VARCHAR(50)      NOT NULL UNIQUE,
    password        VARCHAR(255)     NOT NULL,
    nick_name       VARCHAR(50)      DEFAULT '系统用户',
    header_img      VARCHAR(255)     DEFAULT '',
    authority_id    BIGINT           DEFAULT 888,
    phone           VARCHAR(20),
    email           VARCHAR(100),
    enable          INT              DEFAULT 1
);
```

#### 3.2.2 sys_authorities (角色表)
```sql
CREATE TABLE sys_authorities (
    created_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    deleted_at       TIMESTAMP,
    authority_id     BIGINT       PRIMARY KEY,
    authority_name   VARCHAR(100) NOT NULL,
    parent_id        BIGINT,
    default_router   VARCHAR(100) DEFAULT 'dashboard'
);
```

#### 3.2.3 sys_apis (API 表)
```sql
CREATE TABLE sys_apis (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP,
    path        VARCHAR(255) NOT NULL,
    description VARCHAR(255),
    api_group   VARCHAR(100),
    method      VARCHAR(10)  DEFAULT 'POST'
);
```

#### 3.2.4 sys_base_menus (菜单表)
```sql
CREATE TABLE sys_base_menus (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMP,
    parent_id   BIGINT       DEFAULT 0,
    path        VARCHAR(255) NOT NULL,
    name        VARCHAR(255),
    hidden      BOOLEAN      DEFAULT FALSE,
    component   VARCHAR(255),
    sort        INT          DEFAULT 0,
    meta_title  VARCHAR(50),
    meta_icon   VARCHAR(50),
    meta_keep_alive BOOLEAN DEFAULT FALSE
);
```

#### 3.2.5 sys_authority_menus (角色菜单关联表)
```sql
CREATE TABLE sys_authority_menus (
    sys_authority_authority_id BIGINT NOT NULL,
    sys_base_menu_id            BIGINT NOT NULL,
    PRIMARY KEY (sys_authority_authority_id, sys_base_menu_id)
);
```

#### 3.2.6 sys_user_authority (用户角色关联表)
```sql
CREATE TABLE sys_user_authority (
    sys_user_id        BIGINT NOT NULL,
    sys_authority_authority_id BIGINT NOT NULL,
    PRIMARY KEY (sys_user_id, sys_authority_authority_id)
);
```

#### 3.2.7 casbin_rule (Casbin 策略表)
```sql
CREATE TABLE casbin_rule (
    id    BIGSERIAL PRIMARY KEY,
    ptype VARCHAR(100),
    v0    VARCHAR(100),
    v1    VARCHAR(100),
    v2    VARCHAR(100),
    v3    VARCHAR(100),
    v4    VARCHAR(100),
    v5    VARCHAR(100)
);
```

#### 3.2.8 sys_jwt_blacklists (Token 黑名单)
```sql
CREATE TABLE sys_jwt_blacklists (
    id         BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    jwt        VARCHAR(1000) NOT NULL
);
```

#### 3.2.9 sys_operation_records (操作记录)
```sql
CREATE TABLE sys_operation_records (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
    ip          VARCHAR(50),
    method      VARCHAR(10),
    path        VARCHAR(255),
    status      INT,
    latency     BIGINT,
    user_id     BIGINT,
    user_name   VARCHAR(50),
    error_msg   TEXT
);
```

### 3.3 中间件设计

#### 3.3.1 JWT 认证中间件
```
请求 → 获取 Token → 验证 Token → 检查黑名单 → 刷新 Token → 设置上下文
```

#### 3.3.2 Casbin 权限中间件
```
请求 → 获取用户角色 → 提取路径和方法 → Casbin 策略检查 → 允许/拒绝
```

### 3.4 权限模型 (Casbin)
```
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub && keyMatch2(r.obj, p.obj) && r.act == p.act
```

### 3.5 默认角色和权限

#### 超级管理员 (authority_id = 1)
- 所有 API 访问权限
- 所有菜单访问权限
- 用户管理权限
- 角色管理权限
- 系统配置权限

#### 普通用户 (authority_id = 888)
- 基础 API 访问权限
- 默认菜单访问权限
- 仅查看权限

## 4. API 设计

### 4.1 认证相关 API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/base/login | 用户登录 |
| POST | /api/base/logout | 用户登出 |
| GET  | /api/base/captcha | 获取验证码 |

### 4.2 用户管理 API
| 方法 | 路径 | 描述 | 权限 |
|------|------|------|------|
| GET    | /api/user/list       | 用户列表 | user:list |
| POST   | /api/user/create     | 创建用户 | user:create |
| PUT    | /api/user/update     | 更新用户 | user:update |
| DELETE | /api/user/delete     | 删除用户 | user:delete |
| PUT    | /api/user/password   | 修改密码 | user:password |
| GET    | /api/user/info       | 当前用户信息 | - |

### 4.3 角色管理 API
| 方法 | 路径 | 描述 | 权限 |
|------|------|------|------|
| GET    | /api/authority/list      | 角色列表 | authority:list |
| POST   | /api/authority/create    | 创建角色 | authority:create |
| PUT    | /api/authority/update    | 更新角色 | authority:update |
| DELETE | /api/authority/delete    | 删除角色 | authority:delete |
| POST   | /api/authority/copyRole  | 复制角色 | authority:copy |

### 4.4 API 权限管理
| 方法 | 路径 | 描述 | 权限 |
|------|------|------|------|
| GET  | /api/api/list          | API 列表 | api:list |
| POST | /api/api/create        | 创建 API | api:create |
| PUT  | /api/api/update        | 更新 API | api:update |
| DELETE | /api/api/delete      | 删除 API | api:delete |
| GET  | /api/api/getByAuthority | 获取角色 API 权限 | api:getAuth |
| POST | /api/api/modifyAuthorityAuth | 修改角色 API 权限 | api:setAuth |

### 4.5 菜单管理 API
| 方法 | 路径 | 描述 | 权限 |
|------|------|------|------|
| GET  | /api/menu/list       | 菜单列表 | menu:list |
| POST | /api/menu/create     | 创建菜单 | menu:create |
| PUT  | /api/menu/update     | 更新菜单 | menu:update |
| DELETE | /api/menu/delete   | 删除菜单 | menu:delete |
| GET  | /api/menu/getByAuthority | 获取角色菜单 | menu:getAuth |
| POST | /api/menu/modifyAuthorityAuth | 修改角色菜单 | menu:setAuth |

## 5. 前端实现

### 5.1 登录页面
- 用户名/密码输入
- 验证码（可选）
- 记住我功能
- 错误提示

### 5.2 权限指令
```javascript
// v-auth 指令：控制按钮显示
<el-button v-auth="'user:create'">创建用户</el-button>

// v-auths 指令：多权限控制（满足其一）
<el-button v-auths="['user:update', 'user:delete']">操作</el-button>

// v-auths-all 指令：多权限控制（全部满足）
<el-button v-auths-all="['admin', 'super_admin']">超级操作</el-button>
```

### 5.3 路由守卫
```javascript
router.beforeEach(async (to, from, next) => {
    // 检查 Token
    // 获取用户权限
    // 动态加载路由
    // 验证页面权限
})
```

### 5.4 Axios 拦截器
```javascript
// 请求拦截：添加 Token
// 响应拦截：处理 401/403
```

## 6. 实施计划

### 阶段一：基础认证 (P0)
1. 数据库表创建
2. 用户和角色模型
3. JWT 认证中间件
4. 登录/登出 API
5. 前端登录页面

### 阶段二：权限控制 (P0)
1. Casbin 集成
2. API 权限中间件
3. 角色/权限管理 API
4. 前端路由守卫

### 阶段三：管理功能 (P1)
1. 用户管理界面
2. 角色管理界面
3. API 权限管理界面
4. 菜单管理界面
5. 操作日志

### 阶段四：增强功能 (P2)
1. 按钮级权限
2. 多点登录控制
3. 角色层级继承
4. 数据权限

## 7. 安全考虑

1. **密码安全**：bcrypt 加密，加盐存储
2. **Token 安全**：
   - 合理的过期时间
   - Token 刷新机制
   - 黑名单机制
3. **HTTPS**：生产环境强制 HTTPS
4. **CORS**：配置合法的跨域来源
5. **SQL 注入**：使用 GORM 参数化查询
6. **XSS 防护**：前端输入过滤
7. **审计日志**：记录敏感操作

## 8. 默认账户

```
用户名: admin
密码: admin123
角色: 超级管理员
```
