package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Connection struct {
	connection *pgxpool.Conn
}

func (c *Connection) Release() {
	c.connection.Release()
}

func (c *Connection) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return c.connection.Exec(ctx, sql, arguments...)
}
func (c *Connection) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return c.connection.Query(ctx, sql, args...)
}
func (c *Connection) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.connection.QueryRow(ctx, sql, args...)
}

func (c *Connection) Begin(ctx context.Context) (pgx.Tx, error) {
	return c.connection.Begin(ctx)
}
func (c *Connection) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return c.connection.BeginTx(ctx, txOptions)
}
