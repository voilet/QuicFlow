package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/voilet/quic-flow/pkg/auth/middleware"
	"github.com/voilet/quic-flow/pkg/auth/models"
)

// UserService 用户服务
type UserService struct {
	db     *gorm.DB
	jwt    *middleware.JWT
	config *middleware.JWTConfig
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, jwtConfig *middleware.JWTConfig) *UserService {
	return &UserService{
		db:     db,
		jwt:    middleware.NewJWT(jwtConfig),
		config: jwtConfig,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Captcha   string `json:"captcha"`
	CaptchaID string `json:"captcha_id"`
}

// SetCaptchaVerifyFunc 设置验证码验证函数
var captchaVerifyFunc func(id, code string) bool

// SetCaptchaVerify 设置验证码验证函数（供外部设置）
func SetCaptchaVerify(fn func(id, code string) bool) {
	captchaVerifyFunc = fn
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	NickName string `json:"nick_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User      *UserInfo `json:"user"`
	Token     string    `json:"token"`
	ExpiresAt int64     `json:"expires_at"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID         uint   `json:"id"`
	UUID       string `json:"uuid"`
	Username   string `json:"username"`
	NickName   string `json:"nick_name"`
	HeaderImg  string `json:"header_img"`
	AuthorityId uint  `json:"authority_id"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Enable     uint   `json:"enable"`
}

// Login 用户登录
func (s *UserService) Login(req *LoginRequest) (*LoginResponse, error) {
	// 验证验证码
	if req.CaptchaID != "" && captchaVerifyFunc != nil {
		if !captchaVerifyFunc(req.CaptchaID, req.Captcha) {
			return nil, errors.New("验证码错误")
		}
	}

	// 查找用户
	var user models.SysUser
	err := s.db.Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, fmt.Errorf("数据库错误: %w", err)
	}

	// 检查用户状态
	if user.Enable != 1 {
		return nil, errors.New("用户已被禁用")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 生成Token
	now := time.Now()
	claims := middleware.CustomClaims{
		UUID:        user.UUID,
		ID:          user.ID,
		Username:    user.Username,
		NickName:    user.NickName,
		AuthorityId: user.AuthorityID,
		BufferTime:  int64(s.config.BufferTime),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.ExpiresTime)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token, err := s.jwt.CreateToken(claims)
	if err != nil {
		return nil, fmt.Errorf("生成Token失败: %w", err)
	}

	return &LoginResponse{
		User: &UserInfo{
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
		Token:     token,
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}

// Register 用户注册
func (s *UserService) Register(req *RegisterRequest) error {
	// 检查用户名是否已存在
	var count int64
	s.db.Model(&models.SysUser{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		return errors.New("用户名已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户
	user := models.SysUser{
		UUID:        uuid.New(),
		Username:    req.Username,
		Password:    string(hashedPassword),
		NickName:    req.NickName,
		Email:       req.Email,
		Phone:       req.Phone,
		AuthorityID: 888, // 默认普通用户角色
		Enable:      1,
		HeaderImg:   "",
	}

	if req.NickName == "" {
		user.NickName = req.Username
	}

	return s.db.Create(&user).Error
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(id uint) (*models.SysUser, error) {
	var user models.SysUser
	err := s.db.Preload("Authority").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(username string) (*models.SysUser, error) {
	var user models.SysUser
	err := s.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserList 获取用户列表
func (s *UserService) GetUserList(page, pageSize int, username, nickname string, enable *uint) ([]*models.SysUser, int64, error) {
	var users []*models.SysUser
	var total int64

	query := s.db.Model(&models.SysUser{})

	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if nickname != "" {
		query = query.Where("nick_name LIKE ?", "%"+nickname+"%")
	}
	if enable != nil {
		query = query.Where("enable = ?", *enable)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	err := query.Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&users).Error

	return users, total, err
}

// CreateUser 创建用户
func (s *UserService) CreateUser(user *models.SysUser) error {
	// 检查用户名是否已存在
	var count int64
	s.db.Model(&models.SysUser{}).Where("username = ?", user.Username).Count(&count)
	if count > 0 {
		return errors.New("用户名已存在")
	}

	// 加密密码
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("密码加密失败: %w", err)
		}
		user.Password = string(hashedPassword)
	}

	// 设置默认值
	if user.UUID == (uuid.UUID{}) {
		user.UUID = uuid.New()
	}
	if user.Enable == 0 {
		user.Enable = 1
	}
	if user.NickName == "" {
		user.NickName = user.Username
	}

	return s.db.Create(user).Error
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(user *models.SysUser) error {
	return s.db.Model(user).Updates(map[string]interface{}{
		"nick_name":    user.NickName,
		"header_img":   user.HeaderImg,
		"phone":        user.Phone,
		"email":        user.Email,
		"authority_id": user.AuthorityID,
		"enable":       user.Enable,
	}).Error
}

// UpdatePassword 更新密码
func (s *UserService) UpdatePassword(userID uint, oldPassword, newPassword string) error {
	var user models.SysUser
	err := s.db.First(&user, userID).Error
	if err != nil {
		return err
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	return s.db.Model(&user).Update("password", string(hashedPassword)).Error
}

// ResetPassword 重置密码（管理员操作）
func (s *UserService) ResetPassword(userID uint, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	return s.db.Model(&models.SysUser{}).Where("id = ?", userID).Update("password", string(hashedPassword)).Error
}

// DeleteUser 删除用户（软删除）
func (s *UserService) DeleteUser(userID uint) error {
	return s.db.Delete(&models.SysUser{}, userID).Error
}

// SetUserAuthority 设置用户角色
func (s *UserService) SetUserAuthority(userID, authorityID uint) error {
	return s.db.Model(&models.SysUser{}).Where("id = ?", userID).Update("authority_id", authorityID).Error
}

// Logout 用户登出
func (s *UserService) Logout(token string) error {
	// 将Token加入黑名单
	return models.JoinBlacklist(s.db, token)
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(userID uint, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	return s.db.Model(&models.SysUser{}).Where("id = ?", userID).Update("password", string(hashedPassword)).Error
}
