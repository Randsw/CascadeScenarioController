package logger

import (
	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Zaplog *zap.Logger

func InitLogger() {
	var err error
	var cfg zap.Config = zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.EncoderConfig.FunctionKey = "func"
	Zaplog, err = cfg.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer Zaplog.Sync()
}

func Info(message string, fields ...zap.Field) {
	Zaplog.Info(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	Zaplog.Warn(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	Zaplog.Debug(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	Zaplog.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	Zaplog.Fatal(message, fields...)
}
