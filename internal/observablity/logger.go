package observability

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var log *zap.Logger

func InitLogger(cfg *Config) *zap.Logger {
	//var _ error

	// Настройка уровня логирования
	level := zapcore.InfoLevel
	switch cfg.LogLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	// Настройка вывода
	var output zapcore.WriteSyncer
	if cfg.LogOutputPath == "stdout" {
		output = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.LogOutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		output = zapcore.AddSync(file)
	}

	// Настройка формата
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var encoder zapcore.Encoder
	if cfg.LogFormat == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, output, level)

	log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return log
}

func GetLogger() *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log
}
