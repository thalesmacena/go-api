package log

import (
	"encoding/json"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	Logger *zap.SugaredLogger
)

// UtcTime returns the UTC time format string "2006-01-02T15:04:05.000Z"
func UtcTime() string {
	return "2006-01-02T15:04:05.000Z"
}

// utcTimeEncoder encodes time in UTC with Z suffix and milliseconds
func utcTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format(UtcTime()))
}

// getLevelEnablerFromEnv reads LOG_LEVEL and maps it to a zap level; defaults to info
func getLevelEnablerFromEnv() zapcore.Level {
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	switch levelStr {
	case "debug":
		return zap.DebugLevel
	case "info", "":
		return zap.InfoLevel
	case "warn", "warning":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "dpanic":
		return zap.DPanicLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		return zap.InfoLevel
	}
}

// formatArgsForJSON converts struct arguments to JSON format for better logging with Infof
func formatArgsForJSON(args []interface{}) []interface{} {
	formattedArgs := make([]interface{}, len(args))
	for i, arg := range args {
		if arg != nil {
			v := reflect.ValueOf(arg)
			// Check if it's a struct (and not a basic type)
			if v.Kind() == reflect.Struct {
				if jsonBytes, err := json.Marshal(arg); err == nil {
					formattedArgs[i] = string(jsonBytes)
				} else {
					formattedArgs[i] = arg
				}
			} else {
				formattedArgs[i] = arg
			}
		} else {
			formattedArgs[i] = arg
		}
	}
	return formattedArgs
}

func init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.MessageKey = "message"      // Altera o nome do campo "msg" para "message"
	encoderConfig.EncodeTime = utcTimeEncoder // Formato UTC com Z e milliseconds
	encoderConfig.TimeKey = "@timestamp"      // Name do campo timestamp
	encoderConfig.CallerKey = "logger_name"   // Name do campo do nome do log

	level := getLevelEnablerFromEnv()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(zapcore.AddSync(os.Stdout)),
		level,
	)

	opts := []zap.Option{
		zap.AddCallerSkip(1),
	}

	appName := strings.TrimSpace(os.Getenv("APP_NAME"))
	if appName != "" {
		opts = append(opts, zap.Fields(zap.String("logName", appName)))
	}

	logIndex := strings.TrimSpace(os.Getenv("LOG_INDEX"))
	if logIndex != "" {
		opts = append(opts, zap.Fields(zap.String("logIndex", logIndex)))
	}

	logger = zap.New(core, opts...)

	Logger = logger.Sugar()
}

// GetProductionLogger returns the production logger, do not use this logger for loggin, only for setup
func GetProductionLogger() *zap.Logger {
	return logger
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
	formattedArgs := formatArgsForJSON(args)
	Logger.Infof(message, formattedArgs...)
}

// Debug logs a message at DebugLevel. The message includes any fields passed at the log site, as well as any fields accumulated on the logger.
func Debug(message string, fields ...zap.Field) {
	logger.Debug(message, fields...)
}

// Debugw logs a message with some additional context. The variadic key-value pairs are treated as they are in With.
func Debugw(message string, keysAndValues ...interface{}) {
	Logger.Debugw(message, keysAndValues...)
}

// Debugf formats the message according to the format specifier and logs it at DebugLevel.
func Debugf(message string, args ...interface{}) {
	formattedArgs := formatArgsForJSON(args)
	Logger.Debugf(message, formattedArgs...)
}

// Error logs a message at ErrorLevel. The message includes any fields passed at the log site, as well as any fields accumulated on the logger.
func Error(message string, fields ...zap.Field) {
	logger.Error(message, fields...)
}

// ErrorWithStack logs a message at ErrorLevel with stack trace
func ErrorWithStack(message string, fields ...zap.Field) {
	stackTrace := string(debug.Stack())
	allFields := append(fields, zap.String("stack_trace", stackTrace))
	logger.Error(message, allFields...)
}

// Errorw logs a message with some additional context. The variadic key-value pairs are treated as they are in With.
func Errorw(message string, keysAndValues ...interface{}) {
	Logger.Errorw(message, keysAndValues...)
}

// ErrorwWithStack logs a message with additional context and stack trace
func ErrorwWithStack(message string, keysAndValues ...interface{}) {
	stackTrace := string(debug.Stack())
	allKeysAndValues := append(keysAndValues, "stack_trace", stackTrace)
	Logger.Errorw(message, allKeysAndValues...)
}

// Errorf formats the message according to the format specifier and logs it at ErrorLevel.
func Errorf(message string, args ...interface{}) {
	formattedArgs := formatArgsForJSON(args)
	Logger.Errorf(message, formattedArgs...)
}

// ErrorfWithStack formats the message and logs it at ErrorLevel with stack trace
func ErrorfWithStack(message string, args ...interface{}) {
	formattedArgs := formatArgsForJSON(args)
	stackTrace := string(debug.Stack())
	Logger.Errorw(message, append(formattedArgs, "stack_trace", stackTrace)...)
}

// Fatal logs a message at FatalLevel. The message includes any fields passed at the log site, as well as any fields accumulated on the logger.
func Fatal(message string, fields ...zap.Field) {
	logger.Fatal(message, fields...)
}

// FatalWithStack logs a message at FatalLevel with stack trace, then calls os.Exit
func FatalWithStack(message string, fields ...zap.Field) {
	stackTrace := string(debug.Stack())
	allFields := append(fields, zap.String("stack_trace", stackTrace))
	logger.Fatal(message, allFields...)
}

// Fatalw logs a message with some additional context, then calls os.Exit. The variadic key-value pairs are treated as they are in With.
func Fatalw(message string, keysAndValues ...interface{}) {
	Logger.Fatalw(message, keysAndValues...)
}

// FatalwWithStack logs a message with additional context and stack trace, then calls os.Exit
func FatalwWithStack(message string, keysAndValues ...interface{}) {
	stackTrace := string(debug.Stack())
	allKeysAndValues := append(keysAndValues, "stack_trace", stackTrace)
	Logger.Fatalw(message, allKeysAndValues...)
}

// Fatalf formats the message according to the format specifier and calls os.Exit.
func Fatalf(message string, args ...interface{}) {
	formattedArgs := formatArgsForJSON(args)
	Logger.Fatalf(message, formattedArgs...)
}

// FatalfWithStack formats the message and logs it at FatalLevel with stack trace, then calls os.Exit
func FatalfWithStack(message string, args ...interface{}) {
	formattedArgs := formatArgsForJSON(args)
	stackTrace := string(debug.Stack())
	Logger.Fatalw(message, append(formattedArgs, "stack_trace", stackTrace)...)
}
