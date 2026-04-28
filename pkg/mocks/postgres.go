package mocks

import (
	"context"

	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Postgres struct {
	ConnectionFunc         func(ctx context.Context) (postgres.Connection, error)
	MasterConnectionFunc   func(ctx context.Context) (postgres.Connection, error)
	ReadOnlyConnectionFunc func(ctx context.Context) (postgres.Connection, error)
	PingFunc               func(ctx context.Context) error
	CloseFunc              func() error

	MasterCalls   int
	ReadOnlyCalls int
	CloseCalls    int
}

func (m *Postgres) Connection(ctx context.Context) (postgres.Connection, error) {
	if m.ConnectionFunc != nil {
		return m.ConnectionFunc(ctx)
	}
	return m.MasterConnection(ctx)
}

func (m *Postgres) MasterConnection(ctx context.Context) (postgres.Connection, error) {
	m.MasterCalls++
	if m.MasterConnectionFunc != nil {
		return m.MasterConnectionFunc(ctx)
	}
	return &Connection{}, nil
}

func (m *Postgres) ReadOnlyConnection(ctx context.Context) (postgres.Connection, error) {
	m.ReadOnlyCalls++
	if m.ReadOnlyConnectionFunc != nil {
		return m.ReadOnlyConnectionFunc(ctx)
	}
	return &Connection{}, nil
}

func (m *Postgres) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

func (m *Postgres) Close() error {
	m.CloseCalls++
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

type Connection struct {
	ExecFunc      func(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryFunc     func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRowFunc  func(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatchFunc func(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFromFunc  func(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	BeginFunc     func(ctx context.Context) (pgx.Tx, error)
	BeginTxFunc   func(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	CloseFunc     func()

	CloseCalls int
}

func (m *Connection) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, arguments...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *Connection) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, sql, args...)
	}
	return nil, nil
}

func (m *Connection) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return nil
}

func (m *Connection) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	if m.SendBatchFunc != nil {
		return m.SendBatchFunc(ctx, b)
	}
	return nil
}

func (m *Connection) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	if m.CopyFromFunc != nil {
		return m.CopyFromFunc(ctx, tableName, columnNames, rowSrc)
	}
	return 0, nil
}

func (m *Connection) Begin(ctx context.Context) (pgx.Tx, error) {
	if m.BeginFunc != nil {
		return m.BeginFunc(ctx)
	}
	return &Tx{}, nil
}

func (m *Connection) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	if m.BeginTxFunc != nil {
		return m.BeginTxFunc(ctx, txOptions)
	}
	return &Tx{}, nil
}

func (m *Connection) Close() {
	m.CloseCalls++
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

type Tx struct {
	Connection

	CommitFunc   func(ctx context.Context) error
	RollbackFunc func(ctx context.Context) error
	PrepareFunc  func(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)

	CommitCalls   int
	RollbackCalls int
}

func (m *Tx) Commit(ctx context.Context) error {
	m.CommitCalls++
	if m.CommitFunc != nil {
		return m.CommitFunc(ctx)
	}
	return nil
}

func (m *Tx) Rollback(ctx context.Context) error {
	m.RollbackCalls++
	if m.RollbackFunc != nil {
		return m.RollbackFunc(ctx)
	}
	return nil
}

func (m *Tx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (m *Tx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	if m.PrepareFunc != nil {
		return m.PrepareFunc(ctx, name, sql)
	}
	return nil, nil
}

func (m *Tx) Conn() *pgx.Conn {
	return nil
}
