package infra

import (
	"runtime"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type CallerInfo struct {
	File     string
	Function string
	Line     int
}
type LogLevel string

const (
	INFO  LogLevel = "INFO"
	ERROR LogLevel = "ERROR"
	DEBUG LogLevel = "DEBUG"
)

func GetCallerInfo(skip int) CallerInfo {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return CallerInfo{}
	}
	fn := runtime.FuncForPC(pc)
	return CallerInfo{
		File:     file,
		Function: fn.Name(),
		Line:     line,
	}
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

func NewLogger(env *Env) *zap.Logger {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	config.EncoderConfig.EncodeTime = customTimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.DisableStacktrace = true

	if env.AppEnv == "DEV" {
		config.Development = true
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	return logger
}

func Log(log *zap.Logger, level LogLevel, msg string, err error, fields ...zap.Field) {
	caller := GetCallerInfo(3)
	logFields := []zap.Field{
		zap.Error(err),
		zap.String("caller_file", caller.File),
		zap.String("caller_func", caller.Function),
		zap.Int("caller_line", caller.Line),
	}
	logFields = append(logFields, fields...)
	switch level {
	case "INFO":
		log.Info(msg, logFields...)
	case "ERROR":
		log.Error(msg, logFields...)
	case "DEBUG":
		log.Debug(msg, logFields...)
	default:
		log.Info(msg, logFields...)
	}
}

func LogError(log *zap.Logger, msg string, err error, fields ...zap.Field) {
	Log(log, ERROR, msg, err, fields...)
}

func LogInfo(log *zap.Logger, msg string, fields ...zap.Field) {
	Log(log, INFO, msg, nil, fields...)
}

func LogDebug(log *zap.Logger, msg string, fields ...zap.Field) {
	Log(log, DEBUG, msg, nil, fields...)
}
