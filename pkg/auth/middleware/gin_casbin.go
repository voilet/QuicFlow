package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// GinCasbinMiddleware Gin Casbin权限中间件
func GinCasbinMiddleware(casbin *CasbinMiddleware, getClaimsFunc func(*gin.Context) (*CustomClaims, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查白名单
		if IsWhiteList(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 获取用户Claims
		claims, err := getClaimsFunc(c)
		if err != nil {
			c.JSON(403, gin.H{"code": 403, "msg": "权限不足"})
			c.Abort()
			return
		}

		// 获取请求路径和方法
		path := c.Request.URL.Path
		method := c.Request.Method

		// 去除路由前缀
		obj := strings.TrimPrefix(path, casbin.config.RouterPrefix)

		// 获取用户角色
		sub := strconv.Itoa(int(claims.AuthorityId))

		// 检查权限
		success, err := casbin.Enforce(sub, obj, method)
		if err != nil {
			c.JSON(500, gin.H{"code": 500, "msg": "权限检查失败"})
			c.Abort()
			return
		}

		if !success {
			c.JSON(403, gin.H{"code": 403, "msg": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}
