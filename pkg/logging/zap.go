package logging

import (
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	once   sync.Once
)

func InitLogger(logPath string) {
	once.Do(func() {
		if err := os.MkdirAll(filepath.Dir(logPath), 0744); err != nil {
			panic(err)
		}

		logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}

		encoderConfig := zapcore.EncoderConfig{
			TimeKey:      "timestamp",
			MessageKey:   "msg",
			LineEnding:   zapcore.DefaultLineEnding,
			EncodeLevel:  zapcore.LowercaseLevelEncoder,
			EncodeTime:   zapcore.ISO8601TimeEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		}

		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		)

		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(logFile),
			zapcore.DebugLevel,
		)

		core := zapcore.NewTee(consoleCore, fileCore)

		logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	})
}

func GetLogger() *zap.Logger {
	if logger == nil {
		InitLogger("logs/temp.log")
	}
	return logger
}

func Infof(format string, args ...interface{}) {
	GetLogger().Sugar().Infof(format, args...)
}

func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Errorf(format string, args ...interface{}) {
	GetLogger().Sugar().Errorf(format, args...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Debugf(format string, args ...interface{}) {
	GetLogger().Sugar().Debugf(format, args...)
}

func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Warnf(format string, args ...interface{}) {
	GetLogger().Sugar().Warnf(format, args...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

func Fatalf(format string, args ...interface{}) {
	GetLogger().Sugar().Fatalf(format, args...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}
