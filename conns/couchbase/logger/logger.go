package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/couchbase/gocb/v2"
	"go.uber.org/zap"
)

const (
	pairNumber = 2

	loggerKey = "couchbase"
)

type (
	ZapLogger struct {
		logger *zap.Logger
	}

	DefaultLogger struct {
		logger *log.Logger
	}
)

func NewZapLogger(l *zap.Logger) *ZapLogger {
	return &ZapLogger{logger: l.With(zap.String("module", loggerKey))}
}

func (zl *ZapLogger) Log(level gocb.LogLevel, _ int, format string, v ...interface{}) error {
	switch level {
	case gocb.LogError:
		zl.logger.Error(fmt.Sprintf(format, v...))
	case gocb.LogWarn:
		zl.logger.Warn(fmt.Sprintf(format, v...))
	default:
		zl.logger.Info(fmt.Sprintf(format, v...))
	}

	return nil
}

func (zl *ZapLogger) Info(msg string, keysAndValues ...any) {
	zl.logger.Info(msg, zl.getFields(keysAndValues)...)
}

func (zl *ZapLogger) getFields(keysAndValues []any) []zap.Field {
	if len(keysAndValues)%pairNumber != 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(keysAndValues)/pairNumber)
	for i := 0; i < len(keysAndValues); i += pairNumber {
		name, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, zap.Any(name, keysAndValues[i+1]))
	}

	return fields
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{logger: log.New(os.Stderr, loggerKey, log.Lmicroseconds|log.Lshortfile)}
}

func (dl *DefaultLogger) Info(msg string, keysAndValues ...any) {
	dl.logger.Print(append([]any{msg}, keysAndValues...)...)
}

func (dl *DefaultLogger) Log(_ gocb.LogLevel, _ int, format string, v ...interface{}) error {
	dl.logger.Printf(format, v...)

	return nil
}
