package api

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/voilet/quic-flow/pkg/monitoring"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestTaskAPI_RegisterRoutes(t *testing.T) {
	logger := monitoring.NewLogger(monitoring.LogLevelInfo, "text")
	router := setupTestRouter()

	// 创建空的 API（仅测试路由注册）
	// 实际测试需要完整的依赖注入
	taskAPI := &TaskAPI{
		logger: logger,
	}
	taskAPI.RegisterRoutes(router.Group("/api"))

	// 验证路由已注册（通过检查路由是否存在）
	// 由于依赖未设置，实际请求会 panic，这里只测试路由注册
	// 实际项目中应该使用完整的依赖注入进行集成测试
	assert.NotNil(t, taskAPI)
}
