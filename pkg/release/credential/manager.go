package credential

import (
	"errors"
	"fmt"
	"time"

	"github.com/voilet/quic-flow/pkg/release/models"
	"gorm.io/gorm"
)

var (
	// ErrCredentialNotFound 凭证不存在
	ErrCredentialNotFound = errors.New("credential not found")
	// ErrCredentialInUse 凭证正在使用中
	ErrCredentialInUse = errors.New("credential is in use")
)

// Manager 凭证管理器
type Manager struct {
	db     *gorm.DB
	cipher Cipher
}

// NewManager 创建凭证管理器
func NewManager(db *gorm.DB, secretKey string) (*Manager, error) {
	cipher, err := NewCipher(secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return &Manager{
		db:     db,
		cipher: cipher,
	}, nil
}

// Create 创建凭证
func (m *Manager) Create(cred *models.Credential, data *models.CredentialData) error {
	// 加密数据
	encryptedData, err := m.cipher.EncryptData(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential data: %w", err)
	}
	cred.EncryptedData = encryptedData

	// 验证项目凭证必须有 ProjectID
	if cred.Scope == models.CredentialScopeProject && cred.ProjectID == nil {
		return errors.New("project credential must have a project_id")
	}

	// 全局凭证清空 ProjectID
	if cred.Scope == models.CredentialScopeGlobal {
		cred.ProjectID = nil
	}

	return m.db.Create(cred).Error
}

// Get 根据 ID 获取凭证（不包含敏感数据）
func (m *Manager) Get(id string) (*models.Credential, error) {
	var cred models.Credential
	err := m.db.Where("id = ?", id).First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCredentialNotFound
		}
		return nil, err
	}
	return &cred, nil
}

// GetWithData 根据 ID 获取凭证（包含解密后的敏感数据）
func (m *Manager) GetWithData(id string) (*models.Credential, *models.CredentialData, error) {
	cred, err := m.Get(id)
	if err != nil {
		return nil, nil, err
	}

	data, err := m.cipher.DecryptData(cred.EncryptedData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	return cred, data, nil
}

// DecryptData 解密凭证数据（公开方法供 API 使用）
func (m *Manager) DecryptData(id string) (*models.CredentialData, error) {
	cred, err := m.Get(id)
	if err != nil {
		return nil, err
	}

	return m.cipher.DecryptData(cred.EncryptedData)
}

// List 列出凭证
func (m *Manager) List(projectID *string, scope *models.CredentialScope) ([]*models.Credential, error) {
	var creds []*models.Credential
	query := m.db.Model(&models.Credential{})

	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	} else {
		// 获取全局凭证 + 项目凭证
		query = query.Where("project_id IS NULL OR project_id = ?", "")
	}

	if scope != nil {
		query = query.Where("scope = ?", *scope)
	}

	err := query.Order("created_at DESC").Find(&creds).Error
	return creds, err
}

// ListByProject 列出项目可用的凭证（全局凭证 + 项目专属凭证）
func (m *Manager) ListByProject(projectID string) ([]*models.Credential, error) {
	var creds []*models.Credential
	err := m.db.Where("project_id IS NULL OR project_id = ?", projectID).
		Order("scope ASC, created_at DESC").
		Find(&creds).Error
	return creds, err
}

// Update 更新凭证（只能更新名称、描述，不能更新敏感数据）
func (m *Manager) Update(cred *models.Credential) error {
	return m.db.Model(&models.Credential{}).
		Where("id = ?", cred.ID).
		Updates(map[string]interface{}{
			"name":        cred.Name,
			"description": cred.Description,
		}).Error
}

// UpdateData 更新凭证数据（需要提供完整数据）
func (m *Manager) UpdateData(id string, data *models.CredentialData) error {
	encryptedData, err := m.cipher.EncryptData(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential data: %w", err)
	}

	return m.db.Model(&models.Credential{}).
		Where("id = ?", id).
		Update("encrypted_data", encryptedData).Error
}

// Delete 删除凭证
func (m *Manager) Delete(id string) error {
	// 检查是否被使用
	var count int64
	if err := m.db.Model(&models.ProjectCredential{}).
		Where("credential_id = ?", id).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrCredentialInUse
	}

	return m.db.Delete(&models.Credential{}, "id = ?", id).Error
}

// RecordUsage 记录凭证使用
func (m *Manager) RecordUsage(id string) error {
	now := time.Now()
	return m.db.Model(&models.Credential{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"use_count":    gorm.Expr("use_count + 1"),
			"last_used_at": now,
		}).Error
}

// AddToProject 将凭证添加到项目
func (m *Manager) AddToProject(projectID, credentialID, alias string) error {
	// 检查凭证是否存在
	var cred models.Credential
	if err := m.db.Where("id = ?", credentialID).First(&cred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCredentialNotFound
		}
		return err
	}

	// 创建关联
	pc := &models.ProjectCredential{
		ProjectID:    projectID,
		CredentialID: credentialID,
		Alias:        alias,
	}

	return m.db.Create(pc).Error
}

// RemoveFromProject 从项目移除凭证
func (m *Manager) RemoveFromProject(projectID, credentialID string) error {
	return m.db.Where("project_id = ? AND credential_id = ?", projectID, credentialID).
		Delete(&models.ProjectCredential{}).Error
}

// GetProjectCredentials 获取项目关联的凭证
func (m *Manager) GetProjectCredentials(projectID string) ([]*models.ProjectCredential, error) {
	var pcs []*models.ProjectCredential
	err := m.db.Preload("Credential").
		Where("project_id = ?", projectID).
		Find(&pcs).Error
	return pcs, err
}
