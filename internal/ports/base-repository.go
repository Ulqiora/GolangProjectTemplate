package ports

import (
	"context"
	"fmt"

	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/jackc/pgx/v5"
)

const (
	TxField = "transaction"
)

type ScanFunc func(dest ...any) error

type BaseModel interface {
	Params() map[string]interface{}
	Fields() []string
	PrimaryKey() any
	Scan(fields []string, scan ScanFunc) error
}

func findExecutor(ctx context.Context, pool postgres.IPostgres) (postgres.SqlExecutor, func(), error) {
	var connection postgres.SqlExecutor
	var fn func()
	if tx, ok := ctx.Value(TxField).(pgx.Tx); ok {
		connection = tx
	} else {
		impl, err := pool.Connection(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("[BaseRepositoryImpl]: %w", err)
		}
		fn = impl.Release
		connection = impl
	}
	return connection, fn, nil
}

type BaseRepository[M BaseModel] interface {
	SelectOne(ctx context.Context, sql string, args ...any) (*M, error)
	Select(ctx context.Context, sql string, args ...any) ([]*M, error)
	Create(ctx context.Context, m *M) error
}

type BaseRepositoryImpl[M BaseModel] struct {
	pool         postgres.IPostgres
	generateFunc func() *M
	tableName    string
}

func NewBaseRepository[M BaseModel](pool postgres.IPostgres, tablename string, generateFunc func() *M) BaseRepository[M] {
	return &BaseRepositoryImpl[M]{
		pool:         pool,
		generateFunc: generateFunc,
		tableName:    tablename,
	}
}

func (repo *BaseRepositoryImpl[M]) SelectOne(ctx context.Context, sql string, args ...any) (*M, error) {
	connection, fn, err := findExecutor(ctx, repo.pool)
	defer func() {
		if fn != nil {
			fn()
		}
	}()
	if err != nil {
		return nil, err
	}
	row := connection.QueryRow(ctx, sql, args...)

	object := repo.generateFunc()
	err = row.Scan(object)
	if err != nil {
		return nil, err
	}
	return object, nil
}

func (repo *BaseRepositoryImpl[M]) Select(ctx context.Context, sql string, args ...any) ([]*M, error) {
	connection, fn, err := findExecutor(ctx, repo.pool)
	defer func() {
		if fn != nil {
			fn()
		}
	}()
	if err != nil {
		return nil, err
	}
	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fields := repo.fields(rows)

	var objects []*M

	for rows.Next() {
		object := repo.generateFunc()
		err := (*object).Scan(fields, rows.Scan)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func (repo *BaseRepositoryImpl[M]) Create(ctx context.Context, m *M) error {
	//connection, err := findExecutor(ctx, repo.pool)
	//if err != nil {
	//	return err
	//}
	//values := m.Params()
	//squirell
	//
	//sql, args, err2 := squirrel.
	//	StatementBuilder.
	//	PlaceholderFormat(squirrel.Dollar).
	//	Insert(Self.table).
	//	SetMap(values).
	//	ToSql()
	//if err2 != nil {
	//	return err2
	//}
	//
	//conn, err3 := Self.db.Conn(ctx)
	//if err3 != nil {
	//	return err3
	//}
	//
	//if _, err = conn.Exec(ctx, sql, args...); err != nil {
	//	return err
	//}

	return nil
}

func (repo *BaseRepositoryImpl[M]) fields(rows pgx.Rows) []string {
	columns := rows.FieldDescriptions()
	fields := make([]string, 0, len(columns))

	for _, d := range columns {
		fields = append(fields, d.Name)
	}
	return fields
}
