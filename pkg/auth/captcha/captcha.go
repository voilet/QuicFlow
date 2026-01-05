package captcha

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"github.com/voilet/quic-flow/pkg/monitoring"
)

// Captcha 验证码
type Captcha struct {
	logger *monitoring.Logger
	store  base64Captcha.Store
}

// NewCaptcha 创建验证码实例
func NewCaptcha(logger *monitoring.Logger) *Captcha {
	return &Captcha{
		logger: logger,
		store:  getDefaultStore(),
	}
}

// Generate 生成验证码
func (c *Captcha) Generate() (id, b64PNG, code string) {
	// 使用与 gin-vue-admin 相同的参数
	// height=80, width=240, length=6, maxSkew=0.7, dotCount=20
	driver := base64Captcha.NewDriverDigit(80, 240, 6, 0.7, 20)
	cp := base64Captcha.NewCaptcha(driver, c.store)

	id, b64PNG, answer, err := cp.Generate()
	if err != nil {
		// 如果生成失败，返回空
		return "", "", ""
	}

	return id, b64PNG, answer
}

// Verify 验证验证码
func (c *Captcha) Verify(id, code string) bool {
	return c.store.Verify(id, code, true)
}

// VerifyCode 验证码存储（兼容旧接口）
type VerifyCode struct {
	ID        string
	Code      string
	ExpiresAt time.Time
}

// CodeStore 验证码存储（兼容旧接口，直接使用 base64Captcha 的 store）
type CodeStore struct {
	store base64Captcha.Store
}

// NewCodeStore 创建验证码存储
func NewCodeStore() *CodeStore {
	return &CodeStore{
		store: getDefaultStore(),
	}
}

// Set 设置验证码
func (s *CodeStore) Set(id, code string, duration time.Duration) {
	// base64Captcha 的 store 已经自动管理过期，这里只需要设置即可
	_ = s.store.Set(id, code)
}

// Verify 验证验证码
func (s *CodeStore) Verify(id, code string) bool {
	return s.store.Verify(id, code, true)
}

// Delete 删除验证码
func (s *CodeStore) Delete(id string) {
	// base64Captcha 的 store 在验证后自动删除
}

// cleanup 定期清理过期验证码（base64Captcha 内部已处理，这里保留接口兼容性）
func (s *CodeStore) cleanup() {
	// base64Captcha 的 Store 内部已经处理了过期清理
}

// 全局验证码存储
var (
	storeOnce  sync.Once
	globalStore base64Captcha.Store
)

// getDefaultStore 获取默认 store（5分钟过期）
func getDefaultStore() base64Captcha.Store {
	storeOnce.Do(func() {
		// 创建内存存储，collectNum=10240, expiration=5分钟
		globalStore = base64Captcha.NewMemoryStore(10240, 5*time.Minute)
	})
	return globalStore
}

// GetCodeStore 获取验证码存储
func GetCodeStore() *CodeStore {
	return &CodeStore{
		store: getDefaultStore(),
	}
}

// RegisterRoutes 注册验证码路由
func (c *Captcha) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/captcha", func(ctx *gin.Context) {
		id, b64PNG, _ := c.Generate()

		// base64Captcha 已经自动存储验证码

		ctx.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
			"data": gin.H{
				"id":      id,
				"img":     b64PNG,
				"expires": 300,
			},
		})
	})
}
