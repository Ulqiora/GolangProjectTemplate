package attribute

type TypeContent int8

const (
	TypeInt TypeContent = iota
	TypeIntSlice

	TypeInt8
	TypeInt16
	TypeInt32
	TypeInt64

	TypeUint8
	TypeUint16
	TypeUint32
	TypeUint64

	TypeFloat32
	TypeFloat32Slice

	TypeFloat64
	TypeFloat64Slice

	TypeString
	TypeBytes

	TypeInt64Slice
	TypeUint64Slice
)

type Field interface {
	GetType() TypeContent
	GetContent() interface{}
	GetKey() string
}

type implField struct {
	Key     string
	Content interface{}
	Type    TypeContent
}

func (f *implField) GetKey() string {
	return f.Key
}

func (f *implField) GetType() TypeContent {
	return f.Type
}

func (f *implField) GetContent() interface{} {
	return f.Content
}
