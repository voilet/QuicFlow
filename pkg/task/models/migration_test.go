package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 执行迁移
	err = Migrate(db)
	assert.NoError(t, err)

	// 验证表是否存在
	assert.True(t, db.Migrator().HasTable(&Task{}))
	assert.True(t, db.Migrator().HasTable(&TaskGroup{}))
	assert.True(t, db.Migrator().HasTable("tb_task_group_relation")) // many2many 关联表
	assert.True(t, db.Migrator().HasTable(&Execution{}))
	assert.True(t, db.Migrator().HasTable(&Client{}))
}

func TestAllModels(t *testing.T) {
	// 验证所有模型都已注册
	assert.NotEmpty(t, AllModels)
	assert.Len(t, AllModels, 4) // TaskGroupRelation 由 many2many 自动管理

	// 验证模型类型
	assert.Contains(t, AllModels, &Task{})
	assert.Contains(t, AllModels, &TaskGroup{})
	assert.Contains(t, AllModels, &Execution{})
	assert.Contains(t, AllModels, &Client{})
}
