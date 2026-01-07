package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/voilet/quic-flow/pkg/auth/middleware"
	"github.com/voilet/quic-flow/pkg/auth/models"
	"github.com/voilet/quic-flow/pkg/auth/service"
)

// AuthAPI 权限认证API
type AuthAPI struct {
	db              *gorm.DB
	userService     *service.UserService
	authorityService *service.AuthorityService
	menuService     *service.MenuService
	jwtConfig       *middleware.JWTConfig
}

// NewAuthAPI 创建权限认证API
func NewAuthAPI(db *gorm.DB, jwtConfig *middleware.JWTConfig) *AuthAPI {
	return &AuthAPI{
		db:              db,
		userService:     service.NewUserService(db, jwtConfig),
		authorityService: service.NewAuthorityService(db),
		menuService:     service.NewMenuService(db),
		jwtConfig:       jwtConfig,
	}
}

// RegisterRoutes 注册路由
func (a *AuthAPI) RegisterRoutes(r *gin.RouterGroup, jwtMiddleware *middleware.JWTAuthMiddleware) {
	// 公开路由（不需要认证）
	public := r.Group("/base")
	{
		public.POST("/login", a.handleLogin)
		public.POST("/register", a.handleRegister)
	}

	// 需要认证的路由
	auth := r.Group("")
	auth.Use(jwtMiddleware.Handler())
	{
		// 用户相关
		auth.GET("/user/info", a.handleGetUserInfo)
		auth.POST("/user/logout", a.handleLogout)
		auth.PUT("/user/password", a.handleChangePassword)

		// 管理员路由
		admin := auth.Group("")
		admin.Use(a.adminMiddleware())
		{
			// 用户管理
			admin.GET("/user/list", a.handleGetUserList)
			admin.POST("/user/create", a.handleCreateUser)
			admin.PUT("/user/update", a.handleUpdateUser)
			admin.DELETE("/user/delete", a.handleDeleteUser)
			admin.PUT("/user/reset-password", a.handleResetPassword)

			// 角色管理
			admin.GET("/authority/list", a.handleGetAuthorityList)
			admin.POST("/authority/create", a.handleCreateAuthority)
			admin.PUT("/authority/update", a.handleUpdateAuthority)
			admin.DELETE("/authority/delete", a.handleDeleteAuthority)
			admin.POST("/authority/copy", a.handleCopyAuthority)

			// 菜单管理
			admin.GET("/menu/list", a.handleGetMenuList)
			admin.POST("/menu/create", a.handleCreateMenu)
			admin.PUT("/menu/update", a.handleUpdateMenu)
			admin.DELETE("/menu/delete", a.handleDeleteMenu)
			admin.GET("/menu/by-authority", a.handleGetMenuByAuthority)
			admin.POST("/menu/set-authority", a.handleSetMenuAuthority)
		}
	}
}

// handleLogin 用户登录
func (a *AuthAPI) handleLogin(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	resp, err := a.userService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "登录成功",
		"data": resp,
	})
}

// handleRegister 用户注册
func (a *AuthAPI) handleRegister(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if err := a.userService.Register(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "注册成功"})
}

// handleGetUserInfo 获取当前用户信息
func (a *AuthAPI) handleGetUserInfo(c *gin.Context) {
	claims, err := middleware.GetClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录"})
		return
	}

	user, err := a.userService.GetUserByID(claims.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"user": service.UserInfo{
				ID:          user.ID,
				UUID:        user.UUID.String(),
				Username:    user.Username,
				NickName:    user.NickName,
				HeaderImg:   user.HeaderImg,
				AuthorityId: user.AuthorityID,
				Phone:       user.Phone,
				Email:       user.Email,
				Enable:      user.Enable,
			},
			"menus": a.getUserMenus(user.AuthorityID),
		},
	})
}

// handleLogout 用户登出
func (a *AuthAPI) handleLogout(c *gin.Context) {
	token := middleware.GetToken(c)
	if token != "" {
		_ = a.userService.Logout(token)
	}
	middleware.ClearToken(c)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "登出成功"})
}

// handleChangePassword 修改密码
func (a *AuthAPI) handleChangePassword(c *gin.Context) {
	type req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	userID, err := middleware.GetUserId(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录"})
		return
	}

	if err := a.userService.UpdatePassword(userID, r.OldPassword, r.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "密码修改成功"})
}

// handleGetUserList 获取用户列表
func (a *AuthAPI) handleGetUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	username := c.Query("username")
	nickname := c.Query("nickname")

	var enable *uint
	if enableStr := c.Query("enable"); enableStr != "" {
		e, _ := strconv.Atoi(enableStr)
		enableVal := uint(e)
		enable = &enableVal
	}

	users, total, err := a.userService.GetUserList(page, pageSize, username, nickname, enable)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"list":  users,
			"total": total,
			"page":  page,
			"page_size": pageSize,
		},
	})
}

// handleCreateUser 创建用户
func (a *AuthAPI) handleCreateUser(c *gin.Context) {
	var user models.SysUser
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if err := a.userService.CreateUser(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "创建成功"})
}

// handleUpdateUser 更新用户
func (a *AuthAPI) handleUpdateUser(c *gin.Context) {
	var user models.SysUser
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if err := a.userService.UpdateUser(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// handleDeleteUser 删除用户
func (a *AuthAPI) handleDeleteUser(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Query("id"))
	if userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的用户ID"})
		return
	}

	if err := a.userService.DeleteUser(uint(userID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// handleResetPassword 重置密码
func (a *AuthAPI) handleResetPassword(c *gin.Context) {
	type req struct {
		UserID uint   `json:"user_id" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if err := a.userService.ResetPassword(r.UserID, r.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "密码重置成功"})
}

// handleGetAuthorityList 获取角色列表
func (a *AuthAPI) handleGetAuthorityList(c *gin.Context) {
	authorities, err := a.authorityService.GetAuthorityList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": authorities,
	})
}

// handleCreateAuthority 创建角色
func (a *AuthAPI) handleCreateAuthority(c *gin.Context) {
	var req service.CreateAuthorityRequestV2
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	authority, err := a.authorityService.CreateAuthority(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "创建成功",
		"data": authority,
	})
}

// handleUpdateAuthority 更新角色
func (a *AuthAPI) handleUpdateAuthority(c *gin.Context) {
	var req service.UpdateAuthorityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if err := a.authorityService.UpdateAuthority(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// handleDeleteAuthority 删除角色
func (a *AuthAPI) handleDeleteAuthority(c *gin.Context) {
	authorityID, _ := strconv.Atoi(c.Query("id"))
	if authorityID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的角色ID"})
		return
	}

	if err := a.authorityService.DeleteAuthority(uint(authorityID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// handleCopyAuthority 复制角色
func (a *AuthAPI) handleCopyAuthority(c *gin.Context) {
	type req struct {
		OldAuthorityID   uint   `json:"old_authority_id" binding:"required"`
		NewAuthorityName string `json:"new_authority_name" binding:"required"`
	}

	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	authority, err := a.authorityService.CopyAuthority(r.OldAuthorityID, r.NewAuthorityName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "复制成功",
		"data": authority,
	})
}

// handleGetMenuList 获取菜单列表
func (a *AuthAPI) handleGetMenuList(c *gin.Context) {
	menus, err := a.menuService.GetMenuList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": menus,
	})
}

// handleCreateMenu 创建菜单
func (a *AuthAPI) handleCreateMenu(c *gin.Context) {
	var req service.CreateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	menu, err := a.menuService.CreateMenu(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "创建成功",
		"data": menu,
	})
}

// handleUpdateMenu 更新菜单
func (a *AuthAPI) handleUpdateMenu(c *gin.Context) {
	var req service.UpdateMenuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if err := a.menuService.UpdateMenu(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// handleDeleteMenu 删除菜单
func (a *AuthAPI) handleDeleteMenu(c *gin.Context) {
	menuID, _ := strconv.Atoi(c.Query("id"))
	if menuID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的菜单ID"})
		return
	}

	if err := a.menuService.DeleteMenu(uint(menuID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// handleGetMenuByAuthority 获取角色的菜单
func (a *AuthAPI) handleGetMenuByAuthority(c *gin.Context) {
	authorityID, _ := strconv.Atoi(c.Query("authority_id"))
	if authorityID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的角色ID"})
		return
	}

	menus, err := a.menuService.GetMenusByAuthority(uint(authorityID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": menus,
	})
}

// handleSetMenuAuthority 设置角色菜单权限
func (a *AuthAPI) handleSetMenuAuthority(c *gin.Context) {
	type req struct {
		AuthorityID uint `json:"authority_id" binding:"required"`
		MenuIDs     []uint `json:"menu_ids" binding:"required"`
	}

	var r req
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	if err := a.authorityService.SetMenuAuthority(r.AuthorityID, r.MenuIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "设置成功"})
}

// adminMiddleware 管理员中间件
func (a *AuthAPI) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := middleware.GetClaims(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录"})
			c.Abort()
			return
		}

		if claims.AuthorityId != 1 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "需要管理员权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getUserMenus 获取用户菜单
func (a *AuthAPI) getUserMenus(authorityID uint) interface{} {
	if authorityID == 1 {
		// 超级管理员返回所有菜单
		menus, _ := a.menuService.GetMenuList()
		return menus
	}
	menus, _ := a.menuService.GetMenusByAuthority(authorityID)
	return menus
}
