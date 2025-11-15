package logger

import (
	"errors"

	"GolangTemplateProject/pkg/logger/attribute"
	"go.uber.org/zap"
)

var (
	defaultLogger Logger
)

func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}

func DefaultLogger() Logger {
	return defaultLogger
}

type Logger interface {
	Debug(msg string, fields ...attribute.Field)
	Info(msg string, fields ...attribute.Field)
	Warn(msg string, fields ...attribute.Field)
	Error(msg string, fields ...attribute.Field)
	DPanic(msg string, fields ...attribute.Field)
	Panic(msg string, fields ...attribute.Field)
	Fatal(msg string, fields ...attribute.Field)
	Sync() error
	With(fields ...attribute.Field) Logger
}

type LoggerImpl struct {
	loggerBase *zap.Logger
}

func (t LoggerImpl) Debug(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	t.loggerBase.Debug(msg, zapFields...)
}

func (t LoggerImpl) Info(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	t.loggerBase.Info(msg, zapFields...)
}

func (t LoggerImpl) Warn(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	t.loggerBase.Warn(msg, zapFields...)
}

func (t LoggerImpl) Error(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	t.loggerBase.Error(msg, zapFields...)
}

func (t LoggerImpl) DPanic(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	t.loggerBase.DPanic(msg, zapFields...)
}

func (t LoggerImpl) Panic(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	t.loggerBase.Panic(msg, zapFields...)
}

func (t LoggerImpl) Fatal(msg string, fields ...attribute.Field) {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	t.loggerBase.Fatal(msg, zapFields...)
}

func (t LoggerImpl) Sync() error {
	return t.loggerBase.Sync()
}

func (t LoggerImpl) With(fields ...attribute.Field) Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, toZapField(f))
	}
	logger := t.loggerBase.With(zapFields...)
	return LoggerImpl{
		loggerBase: logger,
	}
}

func NewLogger() (Logger, error) {
	logger, err := zap.NewDevelopment()
	return LoggerImpl{
		loggerBase: logger,
	}, err
}

func toZapField(f attribute.Field) zap.Field {
	switch f.GetType() {
	case attribute.TypeInt:
		return zap.Int(f.GetKey(), f.GetContent().(int))
	case attribute.TypeIntSlice:
		return zap.Ints(f.GetKey(), f.GetContent().([]int))
	case attribute.TypeInt8:
		return zap.Int8(f.GetKey(), f.GetContent().(int8))
	case attribute.TypeInt16:
		return zap.Int16(f.GetKey(), f.GetContent().(int16))
	case attribute.TypeInt32:
		return zap.Int32(f.GetKey(), f.GetContent().(int32))
	case attribute.TypeInt64:
		return zap.Int64(f.GetKey(), f.GetContent().(int64))

	case attribute.TypeUint8:
		return zap.Uint8(f.GetKey(), f.GetContent().(uint8))
	case attribute.TypeUint16:
		return zap.Uint16(f.GetKey(), f.GetContent().(uint16))
	case attribute.TypeUint32:
		return zap.Uint32(f.GetKey(), f.GetContent().(uint32))
	case attribute.TypeUint64:
		return zap.Uint64(f.GetKey(), f.GetContent().(uint64))

	case attribute.TypeFloat32:
		return zap.Float32(f.GetKey(), f.GetContent().(float32))
	case attribute.TypeFloat32Slice:
		return zap.Float32s(f.GetKey(), f.GetContent().([]float32))

	case attribute.TypeFloat64:
		return zap.Float64(f.GetKey(), f.GetContent().(float64))
	case attribute.TypeFloat64Slice:
		return zap.Float64s(f.GetKey(), f.GetContent().([]float64))

	case attribute.TypeString:
		return zap.String(f.GetKey(), f.GetContent().(string))
	case attribute.TypeBytes:
		return zap.ByteString(f.GetKey(), f.GetContent().([]byte))

	case attribute.TypeInt64Slice:
		return zap.Int64s(f.GetKey(), f.GetContent().([]int64))
	case attribute.TypeUint64Slice:
		return zap.Uint64s(f.GetKey(), f.GetContent().([]uint64))
	}
	return zap.Error(errors.New("unknown field type"))
}
