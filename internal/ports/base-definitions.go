package ports

import (
	"context"
)

type ScanFunc func(dest ...any) error

type BaseModel interface {
	Params() map[string]interface{}
	Fields() []string
	PrimaryKey() any
	Scan(fields []string, scan ScanFunc) error
}

const (
	TxField = "transaction"
)

type BaseRepository[M BaseModel] interface {
	SelectOne(ctx context.Context, sql string, args ...any) (M, error)
	Select(ctx context.Context, sql string, args ...any) ([]M, error)
	Create(ctx context.Context, m M) (M, error)
}
