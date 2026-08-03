package logging

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(cfg config.LoggingConfig) (*zap.Logger, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.File), 0o750); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	rotating := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.File,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	})
	cores := []zapcore.Core{zapcore.NewCore(encoder, rotating, level)}
	if cfg.AlsoConsole {
		cores = append(cores, zapcore.NewCore(encoder.Clone(), zapcore.AddSync(os.Stdout), level))
	}
	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}
