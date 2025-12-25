package monitoring

import (
	"context"
	"log/slog"
	"os"
)

// Logger 封装结构化日志功能
type Logger struct {
	logger *slog.Logger
	level  slog.Level
}

// LogLevel 定义日志级别类型
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// NewLogger 创建新的 Logger 实例
// level: 日志级别（DEBUG/INFO/WARN/ERROR）
// format: 日志格式（"json" 或 "text"）
func NewLogger(level LogLevel, format string) *Logger {
	var slogLevel slog.Level
	switch level {
	case LogLevelDebug:
		slogLevel = slog.LevelDebug
	case LogLevelInfo:
		slogLevel = slog.LevelInfo
	case LogLevelWarn:
		slogLevel = slog.LevelWarn
	case LogLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		logger: slog.New(handler),
		level:  slogLevel,
	}
}

// NewDefaultLogger 创建默认配置的 Logger（INFO 级别，文本格式）
func NewDefaultLogger() *Logger {
	return NewLogger(LogLevelInfo, "text")
}

// Debug 记录 DEBUG 级别日志
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info 记录 INFO 级别日志
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn 记录 WARN 级别日志
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error 记录 ERROR 级别日志
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// DebugContext 使用 context 记录 DEBUG 级别日志
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

// InfoContext 使用 context 记录 INFO 级别日志
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

// WarnContext 使用 context 记录 WARN 级别日志
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

// ErrorContext 使用 context 记录 ERROR 级别日志
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}

// With 创建带有预设字段的子 Logger
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
		level:  l.level,
	}
}

// WithGroup 创建带有分组的子 Logger
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		logger: l.logger.WithGroup(name),
		level:  l.level,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	var slogLevel slog.Level
	switch level {
	case LogLevelDebug:
		slogLevel = slog.LevelDebug
	case LogLevelInfo:
		slogLevel = slog.LevelInfo
	case LogLevelWarn:
		slogLevel = slog.LevelWarn
	case LogLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	l.level = slogLevel
}

// Enabled 检查指定级别的日志是否会被记录
func (l *Logger) Enabled(level LogLevel) bool {
	var slogLevel slog.Level
	switch level {
	case LogLevelDebug:
		slogLevel = slog.LevelDebug
	case LogLevelInfo:
		slogLevel = slog.LevelInfo
	case LogLevelWarn:
		slogLevel = slog.LevelWarn
	case LogLevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	return l.logger.Enabled(context.Background(), slogLevel)
}

// 全局默认 Logger 实例
var defaultLogger = NewDefaultLogger()

// Debug 使用默认 Logger 记录 DEBUG 级别日志
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info 使用默认 Logger 记录 INFO 级别日志
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn 使用默认 Logger 记录 WARN 级别日志
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error 使用默认 Logger 记录 ERROR 级别日志
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// SetDefaultLogger 设置默认 Logger
func SetDefaultLogger(logger *Logger) {
	if logger != nil {
		defaultLogger = logger
	}
}

// GetDefaultLogger 获取默认 Logger
func GetDefaultLogger() *Logger {
	return defaultLogger
}
