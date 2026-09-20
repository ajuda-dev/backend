package logger

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	log muLogger

	LOG_OUTPUT       = "LOG_OUTPUT"
	LOG_LEVEL        = "LOG_LEVEL"
	LOG_MAX_SIZE_MB  = "LOG_MAX_SIZE_MB"
	LOG_MAX_AGE_DAYS = "LOG_MAX_AGE_DAYS"
	LOG_MAX_BACKUPS  = "LOG_MAX_BACKUPS"
)

type muLogger struct {
	mu      sync.RWMutex
	z       *zap.Logger
	fileOut *lumberjack.Logger
}

func init() {
	Configure()
}

func Configure() {
	encoderCfg := zapcore.EncoderConfig{
		LevelKey:     "level",
		TimeKey:      "time",
		MessageKey:   "message",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	encoder := zapcore.NewJSONEncoder(encoderCfg)
	level := getLevelLogs()
	cores := []zapcore.Core{
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level),
	}

	log.mu.Lock()
	defer log.mu.Unlock()
	if log.z != nil {
		_ = log.z.Sync()
	}
	if log.fileOut != nil {
		_ = log.fileOut.Close()
		log.fileOut = nil
	}

	if path := fileLogPath(); path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
			fileOut := &lumberjack.Logger{
				Filename:   path,
				MaxSize:    getEnvInt(LOG_MAX_SIZE_MB, 50),
				MaxAge:     getEnvInt(LOG_MAX_AGE_DAYS, 14),
				MaxBackups: getEnvInt(LOG_MAX_BACKUPS, 7),
				Compress:   true,
			}
			log.fileOut = fileOut
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(fileOut), level))
		}
	}
	log.z = zap.New(zapcore.NewTee(cores...))
}

func Info(message string, tags ...zap.Field) {
	l := logger()
	l.Info(message, tags...)
	_ = l.Sync()
}

func Warn(message string, tags ...zap.Field) {
	l := logger()
	l.Warn(message, tags...)
	_ = l.Sync()
}

func Error(message string, err error, tags ...zap.Field) {
	tags = append(tags, zap.NamedError("error", err))
	l := logger()
	l.Error(message, tags...)
	_ = l.Sync()
}

func logger() *zap.Logger {
	log.mu.RLock()
	defer log.mu.RUnlock()
	return log.z
}

func fileLogPath() string {
	output := strings.TrimSpace(os.Getenv(LOG_OUTPUT))
	if output == "" || strings.EqualFold(output, "stdout") {
		return ""
	}
	return output
}

func getLevelLogs() zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(LOG_LEVEL))) {
	case "info":
		return zapcore.InfoLevel
	case "error":
		return zapcore.ErrorLevel
	case "debug":
		return zapcore.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}

func getEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
