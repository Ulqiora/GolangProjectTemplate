package attribute

type TypeContent int8

const (
	TypeInt TypeContent = iota
	TypeIntSlice

	TypeInt64

	TypeFloat64
	TypeFloat64Slice

	TypeString
	TypeInt64Slice
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
