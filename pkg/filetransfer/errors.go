package filetransfer

import (
	"fmt"
)

// 错误码常量
const (
	ErrCodeTaskNotFound        = 40001
	ErrCodeInvalidChecksum     = 40002
	ErrCodeStorageQuotaExceeded = 40003
	ErrCodeFileTooLarge        = 40004
	ErrCodeInvalidOffset       = 40005
	ErrCodeFileNotFound        = 40006
	ErrCodeFileAlreadyExists   = 40007
	ErrCodeTransferFailed      = 50001
	ErrCodeStorageError        = 50002
	ErrCodeInvalidParameters   = 40008
	ErrCodeUnauthorized        = 40009
	ErrCodeForbidden           = 40010
)

// TransferError 传输错误
type TransferError struct {
	Code    int
	Message string
	Cause   error
}

// Error 实现 error 接口
func (e *TransferError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回底层错误
func (e *TransferError) Unwrap() error {
	return e.Cause
}

// NewTransferError 创建新的传输错误
func NewTransferError(code int, message string, cause error) *TransferError {
	return &TransferError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// 预定义错误
var (
	ErrTaskNotFound         = &TransferError{Code: ErrCodeTaskNotFound, Message: "Task not found"}
	ErrInvalidChecksum      = &TransferError{Code: ErrCodeInvalidChecksum, Message: "Checksum verification failed"}
	ErrQuotaExceeded        = &TransferError{Code: ErrCodeStorageQuotaExceeded, Message: "Storage quota exceeded"}
	ErrFileTooLarge         = &TransferError{Code: ErrCodeFileTooLarge, Message: "File size exceeds maximum allowed size"}
	ErrInvalidOffset        = &TransferError{Code: ErrCodeInvalidOffset, Message: "Invalid chunk offset"}
	ErrFileNotFound         = &TransferError{Code: ErrCodeFileNotFound, Message: "File not found"}
	ErrFileAlreadyExists    = &TransferError{Code: ErrCodeFileAlreadyExists, Message: "File already exists"}
	ErrTransferFailed       = &TransferError{Code: ErrCodeTransferFailed, Message: "Transfer failed"}
	ErrStorageError         = &TransferError{Code: ErrCodeStorageError, Message: "Storage operation failed"}
	ErrInvalidParameters    = &TransferError{Code: ErrCodeInvalidParameters, Message: "Invalid parameters"}
	ErrUnauthorized         = &TransferError{Code: ErrCodeUnauthorized, Message: "Unauthorized access"}
	ErrForbidden            = &TransferError{Code: ErrCodeForbidden, Message: "Forbidden operation"}
)

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	if te, ok := err.(*TransferError); ok {
		// 网络相关错误通常可重试
		return te.Code == ErrCodeTransferFailed || te.Code == ErrCodeStorageError
	}
	return false
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) int {
	if te, ok := err.(*TransferError); ok {
		return te.Code
	}
	return 50000 // 未知错误
}
