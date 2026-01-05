package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/voilet/quic-flow/pkg/auth/models"
)

// MenuService 菜单服务
type MenuService struct {
	db *gorm.DB
}

// NewMenuService 创建菜单服务
func NewMenuService(db *gorm.DB) *MenuService {
	return &MenuService{db: db}
}

// CreateMenuRequest 创建菜单请求
type CreateMenuRequest struct {
	ParentId  uint   `json:"parent_id"`
	Path      string `json:"path" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Component string `json:"component"`
	Hidden    bool   `json:"hidden"`
	Sort      int    `json:"sort"`
	Title     string `json:"title" binding:"required"`
	Icon      string `json:"icon"`
	KeepAlive bool   `json:"keep_alive"`
}

// UpdateMenuRequest 更新菜单请求
type UpdateMenuRequest struct {
	ID        uint   `json:"id" binding:"required"`
	ParentId  uint   `json:"parent_id"`
	Path      string `json:"path"`
	Name      string `json:"name"`
	Component string `json:"component"`
	Hidden    *bool  `json:"hidden"`
	Sort      *int   `json:"sort"`
	Title     string `json:"title"`
	Icon      string `json:"icon"`
	KeepAlive *bool  `json:"keep_alive"`
}

// GetMenuList 获取菜单列表（树形结构）
func (s *MenuService) GetMenuList() ([]*models.SysBaseMenu, error) {
	var menus []*models.SysBaseMenu
	err := s.db.Order("sort ASC").Find(&menus).Error
	if err != nil {
		return nil, err
	}
	return s.buildMenuTree(menus, 0), nil
}

// buildMenuTree 构建菜单树
func (s *MenuService) buildMenuTree(menus []*models.SysBaseMenu, parentID uint) []*models.SysBaseMenu {
	var tree []*models.SysBaseMenu
	for _, menu := range menus {
		if menu.ParentId == parentID {
			children := s.buildMenuTree(menus, menu.ID)
			// 将指针切片转换为值切片
			menu.Children = make([]models.SysBaseMenu, len(children))
			for i, child := range children {
				menu.Children[i] = *child
			}
			tree = append(tree, menu)
		}
	}
	return tree
}

// GetMenuByID 根据ID获取菜单
func (s *MenuService) GetMenuByID(id uint) (*models.SysBaseMenu, error) {
	var menu models.SysBaseMenu
	err := s.db.Preload("Parameters").First(&menu, id).Error
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

// CreateMenu 创建菜单
func (s *MenuService) CreateMenu(req *CreateMenuRequest) (*models.SysBaseMenu, error) {
	menu := &models.SysBaseMenu{
		ParentId:    req.ParentId,
		Path:        req.Path,
		Name:        req.Name,
		Component:   req.Component,
		Hidden:      req.Hidden,
		Sort:        req.Sort,
		Title:       req.Title,
		Icon:        req.Icon,
		KeepAlive:   req.KeepAlive,
		DefaultMenu: false,
		CloseTab:    false,
	}

	if err := s.db.Create(menu).Error; err != nil {
		return nil, err
	}
	return menu, nil
}

// UpdateMenu 更新菜单
func (s *MenuService) UpdateMenu(req *UpdateMenuRequest) error {
	// 检查菜单是否存在
	var count int64
	s.db.Model(&models.SysBaseMenu{}).Where("id = ?", req.ID).Count(&count)
	if count == 0 {
		return errors.New("菜单不存在")
	}

	updates := make(map[string]interface{})
	if req.Path != "" {
		updates["path"] = req.Path
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Component != "" {
		updates["component"] = req.Component
	}
	if req.Hidden != nil {
		updates["hidden"] = *req.Hidden
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.KeepAlive != nil {
		updates["keep_alive"] = *req.KeepAlive
	}

	return s.db.Model(&models.SysBaseMenu{}).
		Where("id = ?", req.ID).
		Updates(updates).Error
}

// DeleteMenu 删除菜单
func (s *MenuService) DeleteMenu(id uint) error {
	// 检查是否有子菜单
	var childCount int64
	s.db.Model(&models.SysBaseMenu{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return errors.New("该菜单下还有子菜单，无法删除")
	}

	return s.db.Delete(&models.SysBaseMenu{}, id).Error
}

// GetMenusByAuthority 根据角色获取菜单
func (s *MenuService) GetMenusByAuthority(authorityID uint) ([]*models.SysBaseMenu, error) {
	var menus []*models.SysBaseMenu
	err := s.db.
		Joins("JOIN sys_authority_menus ON sys_authority_menus.sys_base_menu_id = sys_base_menus.id").
		Where("sys_authority_menus.sys_authority_authority_id = ?", authorityID).
		Order("sort ASC").
		Find(&menus).Error

	if err != nil {
		return nil, err
	}
	return s.buildMenuTree(menus, 0), nil
}
