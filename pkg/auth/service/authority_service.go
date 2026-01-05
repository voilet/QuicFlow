package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/voilet/quic-flow/pkg/auth/models"
)

// AuthorityService 角色服务
type AuthorityService struct {
	db *gorm.DB
}

// NewAuthorityService 创建角色服务
func NewAuthorityService(db *gorm.DB) *AuthorityService {
	return &AuthorityService{db: db}
}

// CreateAuthorityRequestV2 创建角色请求
type CreateAuthorityRequestV2 struct {
	AuthorityName string `json:"authority_name" binding:"required"`
	ParentId      *uint  `json:"parent_id"`
	DefaultRouter string `json:"default_router"`
}

// CreateAuthorityRequest 创建角色请求（别名）
type CreateAuthorityRequest = CreateAuthorityRequestV2

// UpdateAuthorityRequest 更新角色请求
type UpdateAuthorityRequest struct {
	AuthorityId   uint   `json:"authority_id" binding:"required"`
	AuthorityName string `json:"authority_name"`
	ParentId      *uint  `json:"parent_id"`
	DefaultRouter string `json:"default_router"`
}

// GetAuthorityList 获取角色列表
func (s *AuthorityService) GetAuthorityList() ([]*models.SysAuthority, error) {
	var authorities []*models.SysAuthority
	err := s.db.Order("authority_id").Find(&authorities).Error
	return authorities, err
}

// GetAuthorityByID 根据ID获取角色
func (s *AuthorityService) GetAuthorityByID(id uint) (*models.SysAuthority, error) {
	var authority models.SysAuthority
	err := s.db.Preload("SysBaseMenus").First(&authority, id).Error
	if err != nil {
		return nil, err
	}
	return &authority, nil
}

// CreateAuthority 创建角色
func (s *AuthorityService) CreateAuthority(req *CreateAuthorityRequest) (*models.SysAuthority, error) {
	// 检查角色名是否已存在
	var count int64
	s.db.Model(&models.SysAuthority{}).Where("authority_name = ?", req.AuthorityName).Count(&count)
	if count > 0 {
		return nil, errors.New("角色名已存在")
	}

	// 获取最大ID
	var maxID uint
	s.db.Model(&models.SysAuthority{}).Select("COALESCE(MAX(authority_id), 0)").Scan(&maxID)

	authority := &models.SysAuthority{
		AuthorityId:   maxID + 1,
		AuthorityName: req.AuthorityName,
		ParentId:      req.ParentId,
		DefaultRouter: req.DefaultRouter,
	}
	if authority.DefaultRouter == "" {
		authority.DefaultRouter = "dashboard"
	}

	if err := s.db.Create(authority).Error; err != nil {
		return nil, err
	}
	return authority, nil
}

// UpdateAuthority 更新角色
func (s *AuthorityService) UpdateAuthority(req *UpdateAuthorityRequest) error {
	// 检查角色是否存在
	var count int64
	s.db.Model(&models.SysAuthority{}).Where("authority_id = ?", req.AuthorityId).Count(&count)
	if count == 0 {
		return errors.New("角色不存在")
	}

	// 检查角色名是否已被其他角色使用
	if req.AuthorityName != "" {
		var nameCount int64
		s.db.Model(&models.SysAuthority{}).
			Where("authority_name = ? AND authority_id != ?", req.AuthorityName, req.AuthorityId).
			Count(&nameCount)
		if nameCount > 0 {
			return errors.New("角色名已存在")
		}
	}

	updates := make(map[string]interface{})
	if req.AuthorityName != "" {
		updates["authority_name"] = req.AuthorityName
	}
	if req.DefaultRouter != "" {
		updates["default_router"] = req.DefaultRouter
	}
	if req.ParentId != nil {
		updates["parent_id"] = req.ParentId
	}

	return s.db.Model(&models.SysAuthority{}).
		Where("authority_id = ?", req.AuthorityId).
		Updates(updates).Error
}

// DeleteAuthority 删除角色
func (s *AuthorityService) DeleteAuthority(id uint) error {
	// 检查是否有用户关联
	var userCount int64
	s.db.Model(&models.SysUser{}).Where("authority_id = ?", id).Count(&userCount)
	if userCount > 0 {
		return errors.New("该角色下还有用户，无法删除")
	}

	return s.db.Delete(&models.SysAuthority{}, "authority_id = ?", id).Error
}

// CopyAuthority 复制角色
func (s *AuthorityService) CopyAuthority(oldAuthorityID uint, newAuthorityName string) (*models.SysAuthority, error) {
	// 获取原角色
	var oldAuthority models.SysAuthority
	err := s.db.Preload("SysBaseMenus").First(&oldAuthority, oldAuthorityID).Error
	if err != nil {
		return nil, err
	}

	// 检查新角色名
	var count int64
	s.db.Model(&models.SysAuthority{}).Where("authority_name = ?", newAuthorityName).Count(&count)
	if count > 0 {
		return nil, errors.New("角色名已存在")
	}

	// 获取最大ID
	var maxID uint
	s.db.Model(&models.SysAuthority{}).Select("COALESCE(MAX(authority_id), 0)").Scan(&maxID)

	// 创建新角色
	newAuthority := &models.SysAuthority{
		AuthorityId:   maxID + 1,
		AuthorityName: newAuthorityName,
		DefaultRouter: oldAuthority.DefaultRouter,
	}

	tx := s.db.Begin()
	if err := tx.Create(newAuthority).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 复制菜单关联
	if len(oldAuthority.SysBaseMenus) > 0 {
		for _, menu := range oldAuthority.SysBaseMenus {
			if err := tx.Exec(
				"INSERT INTO sys_authority_menus (sys_authority_authority_id, sys_base_menu_id) VALUES (?, ?)",
				newAuthority.AuthorityId, menu.ID,
			).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	tx.Commit()
	return newAuthority, nil
}

// SetMenuAuthority 设置角色菜单权限
func (s *AuthorityService) SetMenuAuthority(authorityID uint, menuIDs []uint) error {
	// 删除旧的菜单关联
	s.db.Exec("DELETE FROM sys_authority_menus WHERE sys_authority_authority_id = ?", authorityID)

	// 添加新的菜单关联
	if len(menuIDs) > 0 {
		for _, menuID := range menuIDs {
			s.db.Exec(
				"INSERT INTO sys_authority_menus (sys_authority_authority_id, sys_base_menu_id) VALUES (?, ?)",
				authorityID, menuID,
			)
		}
	}

	return nil
}

// GetMenuAuthority 获取角色的菜单权限
func (s *AuthorityService) GetMenuAuthority(authorityID uint) ([]uint, error) {
	var menuIDs []uint
	err := s.db.Model(&models.SysAuthorityMenu{}).
		Where("sys_authority_authority_id = ?", authorityID).
		Pluck("sys_base_menu_id", &menuIDs).Error
	return menuIDs, err
}
