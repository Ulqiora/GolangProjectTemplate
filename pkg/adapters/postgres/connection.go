package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connection определяет набор операций, которые можно выполнять над соединением/транзакцией.
// Интерфейс включает исполнение запросов, работу с батчами, копирование и управление транзакциями.
type Connection interface {
	// Exec выполняет команду SQL (INSERT/UPDATE/DELETE и пр.).
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	// Query выполняет запрос возвращающий строки.
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	// QueryRow выполняет запрос, ожидаемый вернуть одну строку.
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	// SendBatch отправляет pgx.Batch и возвращает результаты.
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	// CopyFrom выполняет операцию COPY FROM для массовой вставки.
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	// Begin начинает транзакцию.
	Begin(ctx context.Context) (pgx.Tx, error)
	// BeginTx начинает транзакцию с опциями.
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	// Close освобождает соединение/ресурсы. Для pgxpool.Conn эквивалент Release.
	Close()
}

// SqlExecutor — упрощённый интерфейс для функций, которым нужен только набор операций исполнения SQL.
type SqlExecutor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// ConnectionImpl — конкретная реализация Connection, обёртка над *pgxpool.Conn.
type ConnectionImpl struct {
	connection *pgxpool.Conn
}

// Release освобождает соединение обратно в пул.
// (оставлено для обратной совместимости с кодом, использующим Release напрямую)
func (c *ConnectionImpl) Release() {
	c.connection.Release()
}

// Exec выполняет SQL-команду через внутреннее соединение.
func (c *ConnectionImpl) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return c.connection.Exec(ctx, sql, arguments...)
}

// Query выполняет запрос и возвращает строки.
func (c *ConnectionImpl) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return c.connection.Query(ctx, sql, args...)
}

// QueryRow выполняет запрос, который возвращает одну строку.
func (c *ConnectionImpl) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return c.connection.QueryRow(ctx, sql, args...)
}

// Begin начинает транзакцию.
func (c *ConnectionImpl) Begin(ctx context.Context) (pgx.Tx, error) {
	return c.connection.Begin(ctx)
}

// BeginTx начинает транзакцию с заданными опциями.
func (c *ConnectionImpl) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return c.connection.BeginTx(ctx, txOptions)
}

// SendBatch отправляет батч команд.
func (c *ConnectionImpl) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return c.connection.SendBatch(ctx, b)
}

// CopyFrom выполняет массовую вставку через COPY FROM.
func (c *ConnectionImpl) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return c.connection.CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// Close освобождает соединение обратно в пул (аналог Release).
func (c *ConnectionImpl) Close() {
	c.connection.Release()
}
