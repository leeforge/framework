package logging

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CusTimeEncoder creates a custom time encoder that adds the prefix and formats the time.
func CusTimeEncoder(config Config) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(config.Prefix + t.Format(config.TimeFormat))
	}
}

// GetEncoder returns a zapcore.Encoder based on the config format.
func GetEncoder(config Config) zapcore.Encoder {
	encoderConfig := getEncoderConfig(config)
	if config.Format == "json" {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getEncoderConfig creates a zapcore.EncoderConfig from the Config.
func getEncoderConfig(config Config) zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:     config.MessageKey,
		LevelKey:       config.LevelKey,
		TimeKey:        config.TimeKey,
		NameKey:        config.NameKey,
		CallerKey:      config.CallerKey,
		StacktraceKey:  config.StacktraceKey,
		LineEnding:     config.LineEnding,
		EncodeLevel:    config.ZapEncodeLevel(),
		EncodeTime:     CusTimeEncoder(config),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
}

// getEncoderCore creates a zapcore.Core for a specific level.
func getEncoderCore(config Config, level zapcore.Level, levelFunc zapcore.LevelEnabler) zapcore.Core {
	writer := getWriteSyncerWithRegistry(config, level.String())
	return zapcore.NewCore(GetEncoder(config), writer, levelFunc)
}

// getLevelPriority returns a LevelEnabler that only enables the exact level.
func getLevelPriority(level zapcore.Level) zap.LevelEnablerFunc {
	return func(l zapcore.Level) bool {
		return l == level
	}
}

// getZapCores creates zapcore.Core instances for all levels >= config.Level.
func getZapCores(config Config) []zapcore.Core {
	cores := make([]zapcore.Core, 0, 7)
	for level := config.TransportLevel(); level <= zapcore.FatalLevel; level++ {
		cores = append(cores, getEncoderCore(config, level, getLevelPriority(level)))
	}
	return cores
}
