package middleware

import (
	"strings"

	"github.com/casbin/casbin/v2"
	casbingormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

// CasbinConfig Casbin配置
type CasbinConfig struct {
	RouterPrefix string // 路由前缀，用于去除路径前缀
	ModelText    string // Casbin模型配置
}

// DefaultCasbinConfig 默认Casbin配置
var DefaultCasbinConfig = CasbinConfig{
	RouterPrefix: "/api",
	ModelText: `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub && keyMatch2(r.obj, p.obj) && r.act == p.act
`,
}

// CasbinMiddleware Casbin权限中间件
type CasbinMiddleware struct {
	enforcer *casbin.SyncedEnforcer
	config   *CasbinConfig
}

// NewCasbinMiddleware 创建Casbin权限中间件
func NewCasbinMiddleware(db *gorm.DB, config ...*CasbinConfig) (*CasbinMiddleware, error) {
	cfg := &DefaultCasbinConfig
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	// 创建GORM适配器
	adapter, err := casbingormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	// 从字符串创建模型
	m, err := model.NewModelFromString(cfg.ModelText)
	if err != nil {
		return nil, err
	}

	// 创建Enforcer
	enforcer, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}

	// 启用自动保存
	enforcer.EnableAutoSave(true)

	// 启用日志（生产环境建议关闭）
	enforcer.EnableLog(false)

	middleware := &CasbinMiddleware{
		enforcer: enforcer,
		config:   cfg,
	}

	// 初始化策略
	if err := middleware.initPolicies(); err != nil {
		return nil, err
	}

	return middleware, nil
}

// initPolicies 初始化默认策略
func (m *CasbinMiddleware) initPolicies() error {
	// 检查是否已有策略
	hasPolicy, _ := m.enforcer.HasPolicy("1", "/api/*", "*")

	if !hasPolicy {
		// 添加超级管理员默认策略（所有权限）
		m.enforcer.AddPolicy("1", "/api/*", "*")
		m.enforcer.AddPolicy("1", "/api/*", "GET")
		m.enforcer.AddPolicy("1", "/api/*", "POST")
		m.enforcer.AddPolicy("1", "/api/*", "PUT")
		m.enforcer.AddPolicy("1", "/api/*", "DELETE")
	}

	return nil
}

// GetEnforcer 获取Enforcer
func (m *CasbinMiddleware) GetEnforcer() *casbin.SyncedEnforcer {
	return m.enforcer
}

// AddPolicy 添加策略
func (m *CasbinMiddleware) AddPolicy(sub, obj, act string) (bool, error) {
	return m.enforcer.AddPolicy(sub, obj, act)
}

// AddPolicies 批量添加策略
func (m *CasbinMiddleware) AddPolicies(rules [][]string) (bool, error) {
	return m.enforcer.AddPolicies(rules)
}

// RemovePolicy 删除策略
func (m *CasbinMiddleware) RemovePolicy(sub, obj, act string) (bool, error) {
	return m.enforcer.RemovePolicy(sub, obj, act)
}

// RemovePolicies 批量删除策略
func (m *CasbinMiddleware) RemovePolicies(rules [][]string) (bool, error) {
	return m.enforcer.RemovePolicies(rules)
}

// GetPolicy 获取所有策略
func (m *CasbinMiddleware) GetPolicy() [][]string {
	policy, _ := m.enforcer.GetPolicy()
	return policy
}

// GetPolicyForUser 获取用户的策略
func (m *CasbinMiddleware) GetPolicyForUser(sub string) [][]string {
	policy, _ := m.enforcer.GetFilteredPolicy(0, sub)
	return policy
}

// ClearPolicy 清空策略
func (m *CasbinMiddleware) ClearPolicy() error {
	m.enforcer.ClearPolicy()
	return nil
}

// UpdatePolicy 更新策略
func (m *CasbinMiddleware) UpdatePolicy(oldPolicy, newPolicy []string) (bool, error) {
	return m.enforcer.UpdatePolicy(oldPolicy, newPolicy)
}

// Enforce 检查权限
func (m *CasbinMiddleware) Enforce(sub, obj, act string) (bool, error) {
	return m.enforcer.Enforce(sub, obj, act)
}

// GetPermissionsForUser 获取用户的所有权限
func (m *CasbinMiddleware) GetPermissionsForUser(sub string) []string {
	permissions, _ := m.enforcer.GetPermissionsForUser(sub)
	// 转换 [][]string 为 []string，取每个权限的第一个元素（路径）
	result := make([]string, len(permissions))
	for i, p := range permissions {
		if len(p) > 1 {
			result[i] = p[1] // 资源路径
		}
	}
	return result
}

// HasPermissionForUser 检查用户是否有特定权限
func (m *CasbinMiddleware) HasPermissionForUser(sub, obj, act string) bool {
	has, _ := m.enforcer.HasPermissionForUser(sub, obj, act)
	return has
}

// GetUsersForRole 获取拥有角色的所有用户
func (m *CasbinMiddleware) GetUsersForRole(sub string) []string {
	users, _ := m.enforcer.GetUsersForRole(sub)
	return users
}

// AddRoleForUser 为用户添加角色
func (m *CasbinMiddleware) AddRoleForUser(user, role string) (bool, error) {
	return m.enforcer.AddRoleForUser(user, role)
}

// DeleteRoleForUser 删除用户角色
func (m *CasbinMiddleware) DeleteRoleForUser(user, role string) (bool, error) {
	return m.enforcer.DeleteRoleForUser(user, role)
}

// DeleteRolesForUser 删除用户的所有角色
func (m *CasbinMiddleware) DeleteRolesForUser(user string) (bool, error) {
	return m.enforcer.DeleteRolesForUser(user)
}

// DeleteUser 删除用户
func (m *CasbinMiddleware) DeleteUser(user string) (bool, error) {
	return m.enforcer.DeleteUser(user)
}

// DeleteRole 删除角色
func (m *CasbinMiddleware) DeleteRole(role string) (bool, error) {
	return m.enforcer.DeleteRole(role)
}

// GetAllRoles 获取所有角色
func (m *CasbinMiddleware) GetAllRoles() []string {
	roles, _ := m.enforcer.GetAllRoles()
	return roles
}

// keyMatch2 路径匹配函数（支持通配符）
// keyMatch2 是 Casbin 内置的，但这里提供一个备用实现
func keyMatch2(key1, key2 string) bool {
	// 完全匹配
	if key1 == key2 {
		return true
	}

	// 通配符匹配
	if strings.HasSuffix(key2, "/*") {
		prefix := strings.TrimSuffix(key2, "/*")
		if key1 == prefix || strings.HasPrefix(key1, prefix+"/") {
			return true
		}
	}

	// 其他通配符模式
	if strings.Contains(key2, "*") {
		// 这里应该使用正则表达式匹配
		// 简化实现，实际应使用regexp
		return strings.HasPrefix(key1, strings.TrimSuffix(key2, "*"))
	}

	return false
}

// WhiteList 白名单路径（不需要权限验证）
var WhiteList = []string{
	"/api/base/login",
	"/api/base/logout",
	"/api/base/captcha",
	"/setup",
	"/health",
}

// IsWhiteList 检查是否在白名单中
func IsWhiteList(path string) bool {
	for _, white := range WhiteList {
		if strings.HasPrefix(path, white) {
			return true
		}
	}
	return false
}
