package store

import (
	"context"

	"github.com/voilet/quic-flow/pkg/task/models"
	"gorm.io/gorm"
)

// GroupStore 分组存储接口
type GroupStore interface {
	Create(ctx context.Context, group *models.TaskGroup) error
	Update(ctx context.Context, group *models.TaskGroup) error
	Delete(ctx context.Context, groupID int64) error
	GetByID(ctx context.Context, groupID int64) (*models.TaskGroup, error)
	List(ctx context.Context) ([]*models.TaskGroup, error)
	GetClients(ctx context.Context, groupID int64) ([]*models.Client, error)
	AddClients(ctx context.Context, groupID int64, clientIDs []string) error
	RemoveClient(ctx context.Context, groupID int64, clientID string) error
}

// groupStoreImpl 分组存储实现
type groupStoreImpl struct {
	db *gorm.DB
}

// NewGroupStore 创建分组存储
func NewGroupStore(db *gorm.DB) GroupStore {
	return &groupStoreImpl{db: db}
}

// Create 创建分组
func (s *groupStoreImpl) Create(ctx context.Context, group *models.TaskGroup) error {
	return s.db.WithContext(ctx).Create(group).Error
}

// Update 更新分组
func (s *groupStoreImpl) Update(ctx context.Context, group *models.TaskGroup) error {
	return s.db.WithContext(ctx).Model(group).Updates(group).Error
}

// Delete 删除分组（软删除）
func (s *groupStoreImpl) Delete(ctx context.Context, groupID int64) error {
	return s.db.WithContext(ctx).Delete(&models.TaskGroup{}, groupID).Error
}

// GetByID 根据ID获取分组
func (s *groupStoreImpl) GetByID(ctx context.Context, groupID int64) (*models.TaskGroup, error) {
	var group models.TaskGroup
	err := s.db.WithContext(ctx).Preload("Tasks").First(&group, groupID).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// List 列表查询分组
func (s *groupStoreImpl) List(ctx context.Context) ([]*models.TaskGroup, error) {
	var groups []*models.TaskGroup
	err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&groups).Error
	return groups, err
}

// GetClients 获取分组下的客户端列表
func (s *groupStoreImpl) GetClients(ctx context.Context, groupID int64) ([]*models.Client, error) {
	var clients []*models.Client
	err := s.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Find(&clients).Error
	return clients, err
}

// AddClients 添加客户端到分组
func (s *groupStoreImpl) AddClients(ctx context.Context, groupID int64, clientIDs []string) error {
	if len(clientIDs) == 0 {
		return nil
	}
	// 批量更新客户端的 group_id
	return s.db.WithContext(ctx).
		Model(&models.Client{}).
		Where("client_id IN ?", clientIDs).
		Update("group_id", groupID).Error
}

// RemoveClient 从分组移除客户端
func (s *groupStoreImpl) RemoveClient(ctx context.Context, groupID int64, clientID string) error {
	// 将客户端的 group_id 设置为 NULL
	return s.db.WithContext(ctx).
		Model(&models.Client{}).
		Where("client_id = ? AND group_id = ?", clientID, groupID).
		Update("group_id", nil).Error
}
