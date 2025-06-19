package ports

import (
	"context"
	"fmt"

	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

const (
	TxField = "transaction"
)

type BaseModel interface {
	Params() map[string]interface{}
	Fields() []string
	PrimaryKey() any
}

func findExecutor(ctx context.Context, pool postgres.IPostgres) (postgres.SqlExecutor, error) {
	var connection postgres.SqlExecutor
	if tx, ok := ctx.Value(TxField).(pgx.Tx); ok {
		connection = tx
	} else {
		impl, err := pool.Connection(ctx)
		if err != nil {
			return nil, fmt.Errorf("[BaseRepositoryImpl]: %w", err)
		}
		connection = impl
	}
	return connection, nil
}

type BaseRepository interface {
}

type BaseRepositoryImpl[M BaseModel] struct {
	pool         postgres.IPostgres
	generateFunc func() *M
	tableName    string
}

func NewBaseRepository[M BaseModel](pool *postgres.Postgres, tablename string, generateFunc func() *M) BaseRepository {
	return &BaseRepositoryImpl[M]{
		pool:         pool,
		generateFunc: generateFunc,
		tableName:    tablename,
	}
}

func (repo *BaseRepositoryImpl[M]) SelectOne(ctx context.Context, sql string, args ...any) (*M, error) {
	connection, err := findExecutor(ctx, repo.pool)
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
	connection, err := findExecutor(ctx, repo.pool)
	if err != nil {
		return nil, err
	}
	row, err := connection.Query(ctx, sql, args...)
	defer row.Close()

	var objects []*M

	for row.Next() {
		object := repo.generateFunc()
		err = row.Scan(object)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func (repo *BaseRepositoryImpl[M]) Create(ctx context.Context, m *M) error {
	connection, err := findExecutor(ctx, repo.pool)
	if err != nil {
		return err
	}
	values := m.Params()
	squirell

	sql, args, err2 := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Insert(Self.table).
		SetMap(values).
		ToSql()
	if err2 != nil {
		return err2
	}

	conn, err3 := Self.db.Conn(ctx)
	if err3 != nil {
		return err3
	}

	if _, err = conn.Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}
