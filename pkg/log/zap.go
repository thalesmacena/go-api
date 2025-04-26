package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var (
	logger *zap.Logger
	Logger *zap.SugaredLogger
)

func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = "msg"                      // Altera o nome do campo "msg" para "msg"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // Formato ISO8601 para timestamp
	encoderConfig.TimeKey = "@timestamp"                  // Nome do campo timestamp
	encoderConfig.CallerKey = "logger_name"               // Nome do campo do nome do log

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(zapcore.AddSync(os.Stdout)),
		zap.InfoLevel,
	)

	logger = zap.New(core,
		zap.Fields(zap.String("logName", os.Getenv("APPLICATION_NAME"))),
		zap.AddCallerSkip(1))

	Logger = logger.Sugar()
}

// Info logs a message at InfoLevel. The message includes any fields passed at the log site, as well as any fields accumulated on the logger.
func Info(message string, fields ...zap.Field) {
	logger.Info(message, fields...)
}

// Infow logs a message with some additional context. The variadic key-value pairs are treated as they are in With.
func Infow(message string, keysAndValues ...interface{}) {
	Logger.Infow(message, keysAndValues...)
}

// Infof formats the message according to the format specifier and logs it at InfoLevel.
func Infof(message string, args ...interface{}) {
	Logger.Infof(message, args...)
}

// Debug logs a message at DebugLevel. The message includes any fields passed at the log site, as well as any fields accumulated on the logger.
func Debug(message string, fields ...zap.Field) {
	logger.Debug(message, fields...)
}

// Debugw logs a message with some additional context. The variadic key-value pairs are treated as they are in With.
func Debugw(message string, keysAndValues ...interface{}) {
	Logger.Debugw(message, keysAndValues...)
}

// Debugf formats the message according to the format specifier and logs it at
func Debugf(message string, args ...interface{}) {
	Logger.Debugf(message, args...)
}

// Error logs a message at ErrorLevel. The message includes any fields passed at the log site, as well as any fields accumulated on the logger.
func Error(message string, fields ...zap.Field) {
	logger.Error(message, fields...)
}

// Errorw logs a message with some additional context. The variadic key-value pairs are treated as they are in With.
func Errorw(message string, keysAndValues ...interface{}) {
	Logger.Errorw(message, keysAndValues...)
}

// Errorf formats the message according to the format specifier and logs it at ErrorLevel.
func Errorf(message string, args ...interface{}) {
	Logger.Errorf(message, args...)
}

// Fatal logs a message at FatalLevel. The message includes any fields passed at the log site, as well as any fields accumulated on the logger.
func Fatal(message string, fields ...zap.Field) {
	logger.Fatal(message, fields...)
}

// Fatalw logs a message with some additional context, then calls os.Exit. The variadic key-value pairs are treated as they are in With.
func Fatalw(message string, keysAndValues ...interface{}) {
	Logger.Fatalw(message, keysAndValues...)
}

// Fatalf formats the message according to the format specifier and calls os.Exit.
func Fatalf(message string, args ...interface{}) {
	Logger.Fatalf(message, args...)
}
