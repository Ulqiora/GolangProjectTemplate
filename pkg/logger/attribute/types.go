package attribute

import (
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func Int(key string, content int) Field {
	return &implField{
		Content: content,
		Type:    TypeInt,
		Key:     key,
	}
}

func IntSlice(key string, content []int) Field {
	return &implField{
		Content: content,
		Type:    TypeIntSlice,
		Key:     key,
	}
}

func Int64(key string, content int64) Field {
	return &implField{
		Content: content,
		Type:    TypeInt64,
		Key:     key,
	}
}

func Float64(key string, content float64) Field {
	return &implField{
		Content: content,
		Type:    TypeFloat64,
		Key:     key,
	}
}

func Float64Slice(key string, content []float64) Field {
	return &implField{
		Content: content,
		Type:    TypeFloat64Slice,
		Key:     key,
	}
}

func String(key string, content string) Field {
	return &implField{
		Content: content,
		Type:    TypeString,
		Key:     key,
	}
}

func Int64Slice(key string, content []int64) Field {
	return &implField{
		Content: content,
		Type:    TypeInt64Slice,
		Key:     key,
	}
}

func ToZapField(f Field) zap.Field {
	switch f.GetType() {
	case TypeInt:
		return zap.Int(f.GetKey(), f.GetContent().(int))
	case TypeIntSlice:
		return zap.Ints(f.GetKey(), f.GetContent().([]int))
	case TypeInt64:
		return zap.Int64(f.GetKey(), f.GetContent().(int64))

	case TypeFloat64:
		return zap.Float64(f.GetKey(), f.GetContent().(float64))
	case TypeFloat64Slice:
		return zap.Float64s(f.GetKey(), f.GetContent().([]float64))

	case TypeString:
		return zap.String(f.GetKey(), f.GetContent().(string))
	case TypeInt64Slice:
		return zap.Int64s(f.GetKey(), f.GetContent().([]int64))
	}
	return zap.Field{}
}

func ToOtelAttribute(f Field) attribute.KeyValue {
	switch f.GetType() {
	case TypeInt:
		return attribute.Int(f.GetKey(), f.GetContent().(int))
	case TypeIntSlice:
		return attribute.IntSlice(f.GetKey(), f.GetContent().([]int))
	case TypeInt64:
		return attribute.Int64(f.GetKey(), f.GetContent().(int64))

	case TypeFloat64:
		return attribute.Float64(f.GetKey(), f.GetContent().(float64))
	case TypeFloat64Slice:
		return attribute.Float64Slice(f.GetKey(), f.GetContent().([]float64))

	case TypeString:
		return attribute.String(f.GetKey(), f.GetContent().(string))
	case TypeInt64Slice:
		return attribute.Int64Slice(f.GetKey(), f.GetContent().([]int64))
	}
	return attribute.KeyValue{}
}

func ToOtelAttributes(fiels []Field) []attribute.KeyValue {
	result := make([]attribute.KeyValue, 0, len(fiels))
	for _, field := range fiels {
		result = append(result, ToOtelAttribute(field))
	}
	return result
}
