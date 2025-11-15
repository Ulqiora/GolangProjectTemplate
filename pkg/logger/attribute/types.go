package attribute

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

func Int32(key string, content int32) Field {
	return &implField{
		Content: content,
		Type:    TypeInt32,
		Key:     key,
	}
}

func Int16(key string, content int16) Field {
	return &implField{
		Content: content,
		Type:    TypeInt16,
		Key:     key,
	}
}

func Int8(key string, content int8) Field {
	return &implField{
		Content: content,
		Type:    TypeInt8,
		Key:     key,
	}
}

func Uint64(key string, content uint64) Field {
	return &implField{
		Content: content,
		Type:    TypeUint64,
		Key:     key,
	}
}

func Uint32(key string, content uint32) Field {
	return &implField{
		Content: content,
		Type:    TypeUint32,
		Key:     key,
	}
}

func Uint16(key string, content uint16) Field {
	return &implField{
		Content: content,
		Type:    TypeUint16,
		Key:     key,
	}
}

func Uint8(key string, content uint8) Field {
	return &implField{
		Content: content,
		Type:    TypeUint8,
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

func Float32(key string, content float32) Field {
	return &implField{
		Content: content,
		Type:    TypeFloat32,
		Key:     key,
	}
}

func Float32Slice(key string, content []float32) Field {
	return &implField{
		Content: content,
		Type:    TypeFloat32Slice,
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

func Bytes(key string, content []byte) Field {
	return &implField{
		Content: content,
		Type:    TypeBytes,
		Key:     key,
	}
}

func Uint64Slice(key string, content []uint64) Field {
	return &implField{
		Content: content,
		Type:    TypeUint64Slice,
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
