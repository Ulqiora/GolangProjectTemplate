package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Connection interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
}

type SqlExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ConnectionImpl struct {
	connection *pgxpool.Conn
}

func (c *ConnectionImpl) Release() {
	c.connection.Release()
}

func (c *ConnectionImpl) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return c.connection.Exec(ctx, sql, arguments...)
}

func (c *ConnectionImpl) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return c.connection.Query(ctx, sql, args...)
}

func (c *ConnectionImpl) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.connection.QueryRow(ctx, sql, args...)
}

func (c *ConnectionImpl) Begin(ctx context.Context) (pgx.Tx, error) {
	return c.connection.Begin(ctx)
}

func (c *ConnectionImpl) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return c.connection.BeginTx(ctx, txOptions)
}
