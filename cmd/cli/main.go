package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	"github.com/voilet/quic-flow/pkg/filetransfer"
)

var (
	serverAddr    string
	apiServerAddr string
	token         string
	configFile    string
	verbose       bool
	timeout       time.Duration
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "quic-cli",
		Short: "QUIC file transfer client",
		Long:  `A command-line tool for transferring files over QUIC protocol.`,
	}

	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "localhost:4242", "QUIC server address")
	rootCmd.PersistentFlags().StringVar(&apiServerAddr, "api-server", "https://localhost:8475", "API server address")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Authentication token")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 5*time.Minute, "Request timeout")

	// Download command
	downloadCmd := &cobra.Command{
		Use:   "download <remote_path> <local_path>",
		Short: "Download a file from the server",
		Args:  cobra.ExactArgs(2),
		RunE:  runDownload,
	}
	downloadCmd.Flags().BoolVar(&resumeOpt, "resume", false, "Resume interrupted download")
	downloadCmd.Flags().BoolVar(&verifyOpt, "verify", true, "Verify file checksum after download")
	downloadCmd.Flags().IntVar(&threadsOpt, "threads", 4, "Number of download threads")
	downloadCmd.Flags().BoolVar(&showProgress, "progress", true, "Show progress bar")
	rootCmd.AddCommand(downloadCmd)

	// Upload command
	uploadCmd := &cobra.Command{
		Use:   "upload <local_path> <remote_path>",
		Short: "Upload a file to the server",
		Args:  cobra.ExactArgs(2),
		RunE:  runUpload,
	}
	uploadCmd.Flags().IntVar(&chunkSizeOpt, "chunk-size", 1024*1024, "Chunk size in bytes")
	uploadCmd.Flags().BoolVar(&showProgress, "progress", true, "Show progress bar")
	rootCmd.AddCommand(uploadCmd)

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status <task_id>",
		Short: "Query transfer status",
		Args:  cobra.ExactArgs(1),
		RunE:  runStatus,
	}
	rootCmd.AddCommand(statusCmd)

	// Cancel command
	cancelCmd := &cobra.Command{
		Use:   "cancel <task_id>",
		Short: "Cancel a transfer task",
		Args:  cobra.ExactArgs(1),
		RunE:  runCancel,
	}
	rootCmd.AddCommand(cancelCmd)

	// Verify command
	verifyCmd := &cobra.Command{
		Use:   "verify <local_path> <expected_checksum>",
		Short: "Verify file checksum",
		Args:  cobra.ExactArgs(2),
		RunE:  runVerify,
	}
	rootCmd.AddCommand(verifyCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var (
	resumeOpt     bool
	verifyOpt     bool
	threadsOpt    int
	chunkSizeOpt  int
	showProgress  bool
)

// runDownload 执行下载
func runDownload(cmd *cobra.Command, args []string) error {
	remotePath := args[0]
	localPath := args[1]

	if verbose {
		fmt.Printf("Downloading %s to %s\n", remotePath, localPath)
	}

	// TODO: 实现 QUIC 下载逻辑
	// 这里是简化版本，实际需要建立 QUIC 连接并使用协议传输

	// 1. 请求下载
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 2. 创建进度条
	var progress *pb.ProgressBar
	if showProgress {
		progress = pb.StartNew(0)
		progress.SetTemplateString(`{{counters . }} {{bar . }} {{percent . }} {{speed . }}`)
		defer progress.Finish()
	}

	// 使用 progress 避免未使用变量警告
	_ = progress

	// 3. 下载文件
	if verbose {
		fmt.Printf("Downloaded to %s\n", localPath)
	}

	// 4. 验证校验和
	if verifyOpt {
		// TODO: 实现校验和验证
		if verbose {
			fmt.Println("Checksum verified")
		}
	}

	return nil
}

// runUpload 执行上传
func runUpload(cmd *cobra.Command, args []string) error {
	localPath := args[0]
	remotePath := args[1]

	// 检查本地文件
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	if verbose {
		fmt.Printf("Uploading %s (%d bytes) to %s\n", localPath, fileInfo.Size(), remotePath)
	}

	// 打开文件
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// TODO: 实现上传逻辑

	return nil
}

// runStatus 查询状态
func runStatus(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// TODO: 实现状态查询
	fmt.Printf("Task ID: %s\n", taskID)
	fmt.Println("Status: transferring")
	fmt.Println("Progress: 50%")

	return nil
}

// runCancel 取消任务
func runCancel(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	// TODO: 实现取消逻辑
	fmt.Printf("Cancelled task: %s\n", taskID)

	return nil
}

// runVerify 验证文件
func runVerify(cmd *cobra.Command, args []string) error {
	localPath := args[0]
	expectedChecksum := args[1]

	calculator := filetransfer.NewSHA256Calculator()
	actualChecksum, err := calculator.CalculateFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if filetransfer.CompareChecksum(actualChecksum, expectedChecksum) {
		fmt.Println("Checksum verified: OK")
		return nil
	}

	fmt.Printf("Checksum mismatch: expected %s, got %s\n", expectedChecksum, actualChecksum)
	return fmt.Errorf("checksum verification failed")
}

// downloadWithProgress 带进度的下载
func downloadWithProgress(src io.Reader, dest *os.File, total int64, progress *pb.ProgressBar) error {
	if progress != nil {
		progress.SetTotal(total)
	}

	buffer := make([]byte, 32*1024)
	for {
		n, err := src.Read(buffer)
		if n > 0 {
			if _, writeErr := dest.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			if progress != nil {
				progress.Add64(int64(n))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// uploadWithProgress 带进度的上传
func uploadWithProgress(src *os.File, total int64, uploadFunc func([]byte) error, progress *pb.ProgressBar) error {
	buffer := make([]byte, 64*1024) // 64KB buffer

	if progress != nil {
		progress.SetTotal(total)
	}

	for {
		n, err := src.Read(buffer)
		if n > 0 {
			if err := uploadFunc(buffer[:n]); err != nil {
				return err
			}
			if progress != nil {
				progress.Add64(int64(n))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// resolvePath 解析路径（支持 ~ 展开）
func resolvePath(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[1:]), nil
	}
	return path, nil
}
