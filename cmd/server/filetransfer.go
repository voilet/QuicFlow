package main

import (
	"github.com/voilet/quic-flow/pkg/api"
	"github.com/voilet/quic-flow/pkg/filetransfer"
	"github.com/voilet/quic-flow/pkg/monitoring"
	"gorm.io/gorm"
)

// SetupFileTransfer 设置文件传输模块
func SetupFileTransfer(releaseDB *gorm.DB, logger *monitoring.Logger) (*filetransfer.Manager, *filetransfer.UploadManager, *filetransfer.DownloadManager, *api.FileTransferAPI) {
	if releaseDB == nil {
		logger.Warn("File transfer disabled (database not configured)")
		return nil, nil, nil, nil
	}

	// 加载文件传输配置
	ftConfig, err := filetransfer.LoadFromFile("config/server.yaml")
	if err != nil {
		logger.Warn("Failed to load filetransfer config, using defaults", "error", err)
		ftConfig = &filetransfer.FileTransferConfig{
			Enabled:     true,
			StorageRoot: "/data/quic-files",
			TempDir:     "/tmp/quic-upload",
		}
		// SetDefaults is called internally by LoadFromFile, but we need to call it manually for the fallback config
		// For now, just set the required fields directly
		if ftConfig.PathTemplate == "" {
			ftConfig.PathTemplate = "{date}/{user}"
		}
		if ftConfig.MaxFileSize == 0 {
			ftConfig.MaxFileSize = 10 * 1024 * 1024 * 1024 // 10GB
		}
		if ftConfig.UserQuota == 0 {
			ftConfig.UserQuota = 100 * 1024 * 1024 * 1024 // 100GB
		}
		if ftConfig.MaxConcurrentTransfers == 0 {
			ftConfig.MaxConcurrentTransfers = 10
		}
	}

	// 确保必要的目录存在
	if err := ftConfig.EnsureDirectories(); err != nil {
		logger.Error("Failed to create filetransfer directories", "error", err)
		return nil, nil, nil, nil
	}

	// 创建存储后端
	storage, err := filetransfer.NewLocalStorage(
		ftConfig.StorageRoot,
		ftConfig.PathTemplate,
		ftConfig.UserQuota,
		releaseDB,
	)
	if err != nil {
		logger.Error("Failed to create storage backend", "error", err)
		return nil, nil, nil, nil
	}

	// 自动迁移文件传输表（必须在 manager.Start() 之前执行）
	if err := filetransfer.AutoMigrateFileTransfer(releaseDB); err != nil {
		logger.Error("Failed to migrate filetransfer tables", "error", err)
		return nil, nil, nil, nil
	} else {
		// 创建索引
		filetransfer.CreateIndexes(releaseDB)
		logger.Info("Filetransfer tables migrated successfully")
	}

	// 创建传输管理器
	manager, err := filetransfer.NewManager(ftConfig.ToConfig(), storage, releaseDB)
	if err != nil {
		logger.Error("Failed to create transfer manager", "error", err)
		return nil, nil, nil, nil
	}

	// 启动管理器
	if err := manager.Start(); err != nil {
		logger.Error("Failed to start transfer manager", "error", err)
		return nil, nil, nil, nil
	}

	// 创建上传管理器
	uploadManager, err := filetransfer.NewUploadManager(manager, ftConfig.TempDir)
	if err != nil {
		logger.Error("Failed to create upload manager", "error", err)
		return nil, nil, nil, nil
	}

	// 创建下载管理器
	downloadManager := filetransfer.NewDownloadManager(manager)

	// 创建文件传输 API
	fileAPI := api.NewFileTransferAPI(manager, uploadManager, downloadManager, releaseDB)

	logger.Info("File transfer module initialized",
		"storage_root", ftConfig.StorageRoot,
		"max_file_size", ftConfig.MaxFileSize,
		"user_quota", ftConfig.UserQuota)

	return manager, uploadManager, downloadManager, fileAPI
}

// AddFileTransferRoutes 添加文件传输路由
func AddFileTransferRoutes(httpServer *api.HTTPServer, fileAPI *api.FileTransferAPI) {
	router := httpServer.GetRouter()
	fileAPI.RegisterRoutes(router.Group("/api"))
}
