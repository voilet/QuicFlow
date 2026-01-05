package middleware

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// TokenExpired Token过期
	TokenExpired = errors.New("token is expired")
	// TokenInvalid Token无效
	TokenInvalid = errors.New("token is invalid")
)

// JWTConfig JWT配置
type JWTConfig struct {
	SigningKey    string        // 签名密钥
	ExpiresTime   time.Duration // 过期时间
	BufferTime    time.Duration // 缓冲时间（自动刷新）
	Issuer        string        // 签发者
}

// DefaultJWTConfig 默认JWT配置
var DefaultJWTConfig = JWTConfig{
	SigningKey:  "quic-flow-secret-key-change-in-production",
	ExpiresTime: 7 * 24 * time.Hour, // 7天
	BufferTime:  1 * time.Hour,
	Issuer:      "quic-flow",
}

// CustomClaims 自定义Claims
type CustomClaims struct {
	UUID        uuid.UUID `json:"uuid"`
	ID          uint      `json:"id"`
	Username    string    `json:"username"`
	NickName    string    `json:"nickName"`
	AuthorityId uint      `json:"authorityId"`
	BufferTime  int64     `json:"bufferTime"`
	jwt.RegisteredClaims
}

// JWT JWT工具
type JWT struct {
	SigningKey []byte
	Config     *JWTConfig
}

// NewJWT 创建JWT
func NewJWT(config ...*JWTConfig) *JWT {
	cfg := &DefaultJWTConfig
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}
	return &JWT{
		SigningKey: []byte(cfg.SigningKey),
		Config:     cfg,
	}
}

// CreateToken 创建Token
func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	if claims.BufferTime == 0 {
		claims.BufferTime = int64(j.Config.BufferTime)
	}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(j.Config.ExpiresTime))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	claims.Issuer = j.Config.Issuer
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}

// CreateTokenByOldToken 使用旧Token创建新Token（用于刷新）
func (j *JWT) CreateTokenByOldToken(oldToken string, claims CustomClaims) (string, error) {
	// 验证旧Token但不检查过期
	_, err := jwt.ParseWithClaims(oldToken, &claims, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})
	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return "", err
	}
	return j.CreateToken(claims)
}

// ParseToken 解析Token
func (j *JWT) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, TokenExpired
		}
		return nil, TokenInvalid
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, TokenInvalid
}

// GetToken 从请求中获取Token
func GetToken(c *gin.Context) string {
	token := c.GetHeader("x-token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}
	return token
}

// SetToken 设置Token到响应头
func SetToken(c *gin.Context, token string, maxAge int) {
	c.Header("x-token", token)
	c.SetCookie("x-token", token, maxAge, "/", "", false, true)
}

// ClearToken 清除Token
func ClearToken(c *gin.Context) {
	c.Header("x-token", "")
	c.SetCookie("x-token", "", -1, "/", "", false, true)
}

// GetClaims 从上下文获取Claims
func GetClaims(c *gin.Context) (*CustomClaims, error) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, errors.New("claims not found")
	}
	return claims.(*CustomClaims), nil
}

// GetUserId 从上下文获取用户ID
func GetUserId(c *gin.Context) (uint, error) {
	claims, err := GetClaims(c)
	if err != nil {
		return 0, err
	}
	return claims.ID, nil
}

// GetAuthorityId 从上下文获取角色ID
func GetAuthorityId(c *gin.Context) (uint, error) {
	claims, err := GetClaims(c)
	if err != nil {
		return 0, err
	}
	return claims.AuthorityId, nil
}

// GetUsername 从上下文获取用户名
func GetUsername(c *gin.Context) (string, error) {
	claims, err := GetClaims(c)
	if err != nil {
		return "", err
	}
	return claims.Username, nil
}

// JWTAuthMiddleware JWT认证中间件
type JWTAuthMiddleware struct {
	jwt     *JWT
	db      *gorm.DB
	config  *JWTConfig
}

// NewJWTAuthMiddleware 创建JWT认证中间件
func NewJWTAuthMiddleware(db *gorm.DB, config ...*JWTConfig) *JWTAuthMiddleware {
	cfg := &DefaultJWTConfig
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}
	return &JWTAuthMiddleware{
		jwt:    NewJWT(cfg),
		db:     db,
		config: cfg,
	}
}

// Handler 返回Gin中间件函数
func (m *JWTAuthMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := GetToken(c)
		if token == "" {
			c.JSON(401, gin.H{"code": 401, "msg": "未登录或非法访问"})
			c.Abort()
			return
		}

		// 检查黑名单
		if isInBlacklist(m.db, token) {
			ClearToken(c)
			c.JSON(401, gin.H{"code": 401, "msg": "您的帐户异地登陆或令牌失效"})
			c.Abort()
			return
		}

		// 解析Token
		claims, err := m.jwt.ParseToken(token)
		if err != nil {
			ClearToken(c)
			if errors.Is(err, TokenExpired) {
				c.JSON(401, gin.H{"code": 401, "msg": "登录已过期，请重新登录"})
			} else {
				c.JSON(401, gin.H{"code": 401, "msg": err.Error()})
			}
			c.Abort()
			return
		}

		// 设置Claims到上下文
		c.Set("claims", claims)

		// Token自动刷新
		if claims.ExpiresAt.Unix()-time.Now().Unix() < claims.BufferTime {
			dr, _ := time.ParseDuration(m.config.ExpiresTime.String())
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(dr))
			newToken, _ := m.jwt.CreateTokenByOldToken(token, *claims)
			newClaims, _ := m.jwt.ParseToken(newToken)
			c.Header("new-token", newToken)
			c.Header("new-expires-at", strconv.FormatInt(newClaims.ExpiresAt.Unix(), 10))
			SetToken(c, newToken, int(dr.Seconds()/60))
		}

		c.Next()
	}
}

// isInBlacklist 检查JWT是否在黑名单中
func isInBlacklist(db *gorm.DB, jwt string) bool {
	var count int64
	db.Table("sys_jwt_blacklists").Where("jwt = ? AND deleted_at is null", jwt).Count(&count)
	return count > 0
}
