# 数据库连接排查和修复工具

## 问题描述

如果遇到以下错误：
```
database "quic_release" does not exist (SQLSTATE 3D000)
```

说明数据库 `quic_release` 不存在，需要创建。

## 解决方案

### 方法 1: 使用 Shell 脚本（推荐）

```bash
# 交互式运行（会提示输入密码）
./scripts/fix-db.sh

# 通过环境变量提供密码
DB_PASSWORD=yourpassword ./scripts/fix-db.sh

# 通过参数提供所有信息
./scripts/fix-db.sh 192.168.110.104 15432 postgres yourpassword quic_release
```

### 方法 2: 使用 Go 脚本

```bash
# 交互式运行
go run scripts/check-db.go

# 通过参数提供信息
go run scripts/check-db.go 192.168.110.104 15432 postgres yourpassword quic_release

# 通过环境变量提供密码
DB_PASSWORD=yourpassword go run scripts/check-db.go 192.168.110.104 15432 postgres
```

### 方法 3: 手动使用 psql

```bash
# 连接到 PostgreSQL 服务器
psql -h 192.168.110.104 -p 15432 -U postgres -d postgres

# 在 psql 中执行以下命令：
CREATE DATABASE quic_release WITH ENCODING 'UTF8';

# 退出
\q
```

## 脚本功能

两个脚本都会执行以下步骤：

1. ✅ 测试连接到 PostgreSQL 服务器
2. ✅ 列出所有数据库
3. ✅ 检查目标数据库是否存在
4. ✅ 如果不存在，自动创建数据库
5. ✅ 测试连接到目标数据库

## 注意事项

- 确保 PostgreSQL 服务器正在运行
- 确保用户有创建数据库的权限
- 如果使用密码，可以通过环境变量 `DB_PASSWORD` 提供，避免在命令行中暴露密码

