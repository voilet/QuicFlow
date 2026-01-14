package store

import (
	"context"
	"fmt"

	"github.com/voilet/quic-flow/pkg/task/models"
	"gorm.io/gorm"
)

// TaskStore 任务存储接口
type TaskStore interface {
	Create(ctx context.Context, task *models.Task) error
	Update(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, taskID int64) error
	GetByID(ctx context.Context, taskID int64) (*models.Task, error)
	List(ctx context.Context, params *ListParams) ([]*models.Task, int64, error)
	ListEnabled(ctx context.Context) ([]*models.Task, error)
	BindGroup(ctx context.Context, taskID int64, groupID int64) error
	UnbindGroup(ctx context.Context, taskID int64, groupID int64) error
	GetGroupIDs(ctx context.Context, taskID int64) ([]int64, error)
}

// ListParams 列表查询参数
type ListParams struct {
	Page     int    // 页码（从1开始）
	PageSize int    // 每页数量
	Status   *int   // 状态筛选（可选）
	Keyword  string // 关键词搜索（名称、描述）
}

// taskStoreImpl 任务存储实现
type taskStoreImpl struct {
	db *gorm.DB
}

// NewTaskStore 创建任务存储
func NewTaskStore(db *gorm.DB) TaskStore {
	return &taskStoreImpl{db: db}
}

// Create 创建任务
func (s *taskStoreImpl) Create(ctx context.Context, task *models.Task) error {
	return s.db.WithContext(ctx).Create(task).Error
}

// Update 更新任务
func (s *taskStoreImpl) Update(ctx context.Context, task *models.Task) error {
	return s.db.WithContext(ctx).Model(task).Updates(task).Error
}

// Delete 删除任务（软删除）
func (s *taskStoreImpl) Delete(ctx context.Context, taskID int64) error {
	return s.db.WithContext(ctx).Delete(&models.Task{}, taskID).Error
}

// GetByID 根据ID获取任务
func (s *taskStoreImpl) GetByID(ctx context.Context, taskID int64) (*models.Task, error) {
	var task models.Task
	err := s.db.WithContext(ctx).Preload("Groups").First(&task, taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List 列表查询任务
func (s *taskStoreImpl) List(ctx context.Context, params *ListParams) ([]*models.Task, int64, error) {
	if params == nil {
		params = &ListParams{
			Page:     1,
			PageSize: 20,
		}
	}

	query := s.db.WithContext(ctx).Model(&models.Task{})

	// 状态筛选
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 关键词搜索
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var tasks []*models.Task
	offset := (params.Page - 1) * params.PageSize
	err := query.Preload("Groups").
		Offset(offset).
		Limit(params.PageSize).
		Order("created_at DESC").
		Find(&tasks).Error

	return tasks, total, err
}

// ListEnabled 获取所有启用的任务
func (s *taskStoreImpl) ListEnabled(ctx context.Context) ([]*models.Task, error) {
	var tasks []*models.Task
	err := s.db.WithContext(ctx).
		Where("status = ?", int(models.TaskStatusEnabled)).
		Preload("Groups").
		Find(&tasks).Error
	return tasks, err
}

// BindGroup 绑定任务到分组
func (s *taskStoreImpl) BindGroup(ctx context.Context, taskID int64, groupID int64) error {
	var task models.Task
	if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	var group models.TaskGroup
	if err := s.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return fmt.Errorf("group not found: %w", err)
	}

	return s.db.WithContext(ctx).Model(&task).Association("Groups").Append(&group)
}

// UnbindGroup 解绑任务和分组
func (s *taskStoreImpl) UnbindGroup(ctx context.Context, taskID int64, groupID int64) error {
	var task models.Task
	if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	var group models.TaskGroup
	if err := s.db.WithContext(ctx).First(&group, groupID).Error; err != nil {
		return fmt.Errorf("group not found: %w", err)
	}

	return s.db.WithContext(ctx).Model(&task).Association("Groups").Delete(&group)
}

// GetGroupIDs 获取任务关联的分组ID列表
func (s *taskStoreImpl) GetGroupIDs(ctx context.Context, taskID int64) ([]int64, error) {
	var task models.Task
	if err := s.db.WithContext(ctx).Preload("Groups").First(&task, taskID).Error; err != nil {
		return nil, err
	}

	groupIDs := make([]int64, len(task.Groups))
	for i, group := range task.Groups {
		groupIDs[i] = group.ID
	}

	return groupIDs, nil
}
