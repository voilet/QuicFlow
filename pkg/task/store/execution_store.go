package store

import (
	"context"

	"github.com/voilet/quic-flow/pkg/task/models"
	"gorm.io/gorm"
)

// ExecutionStore 执行记录存储接口
type ExecutionStore interface {
	Create(ctx context.Context, execution *models.Execution) error
	Update(ctx context.Context, execution *models.Execution) error
	GetByID(ctx context.Context, executionID int64) (*models.Execution, error)
	List(ctx context.Context, params *ExecutionListParams) ([]*models.Execution, int64, error)
	GetByTaskID(ctx context.Context, taskID int64, limit int) ([]*models.Execution, error)
	GetByClientID(ctx context.Context, clientID string, limit int) ([]*models.Execution, error)
}

// ExecutionListParams 执行记录列表查询参数
type ExecutionListParams struct {
	Page     int    // 页码
	PageSize int    // 每页数量
	TaskID   *int64 // 任务ID筛选
	ClientID string // 客户端ID筛选
	Status   *int   // 状态筛选
	Keyword  string // 关键词搜索
}

// executionStoreImpl 执行记录存储实现
type executionStoreImpl struct {
	db *gorm.DB
}

// NewExecutionStore 创建执行记录存储
func NewExecutionStore(db *gorm.DB) ExecutionStore {
	return &executionStoreImpl{db: db}
}

// Create 创建执行记录
func (s *executionStoreImpl) Create(ctx context.Context, execution *models.Execution) error {
	return s.db.WithContext(ctx).Create(execution).Error
}

// Update 更新执行记录
func (s *executionStoreImpl) Update(ctx context.Context, execution *models.Execution) error {
	return s.db.WithContext(ctx).Model(execution).Updates(execution).Error
}

// GetByID 根据ID获取执行记录
func (s *executionStoreImpl) GetByID(ctx context.Context, executionID int64) (*models.Execution, error) {
	var execution models.Execution
	err := s.db.WithContext(ctx).Preload("Task").First(&execution, executionID).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// List 列表查询执行记录
func (s *executionStoreImpl) List(ctx context.Context, params *ExecutionListParams) ([]*models.Execution, int64, error) {
	if params == nil {
		params = &ExecutionListParams{
			Page:     1,
			PageSize: 20,
		}
	}

	query := s.db.WithContext(ctx).Model(&models.Execution{})

	// 任务ID筛选
	if params.TaskID != nil {
		query = query.Where("task_id = ?", *params.TaskID)
	}

	// 客户端ID筛选
	if params.ClientID != "" {
		query = query.Where("client_id = ?", params.ClientID)
	}

	// 状态筛选
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 关键词搜索
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("task_name LIKE ? OR output LIKE ? OR error_msg LIKE ?", keyword, keyword, keyword)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var executions []*models.Execution
	offset := (params.Page - 1) * params.PageSize
	err := query.Preload("Task").
		Offset(offset).
		Limit(params.PageSize).
		Order("created_at DESC").
		Find(&executions).Error

	return executions, total, err
}

// GetByTaskID 根据任务ID获取执行记录
func (s *executionStoreImpl) GetByTaskID(ctx context.Context, taskID int64, limit int) ([]*models.Execution, error) {
	var executions []*models.Execution
	query := s.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&executions).Error
	return executions, err
}

// GetByClientID 根据客户端ID获取执行记录
func (s *executionStoreImpl) GetByClientID(ctx context.Context, clientID string, limit int) ([]*models.Execution, error) {
	var executions []*models.Execution
	query := s.db.WithContext(ctx).
		Where("client_id = ?", clientID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&executions).Error
	return executions, err
}
