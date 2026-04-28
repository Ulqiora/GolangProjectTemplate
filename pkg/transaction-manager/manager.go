package transaction_manager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/jackc/pgx/v5"
)

const (
	TxKey = "transaction-key"
)

type TransactionManager interface {
	Do(context.Context, func(context.Context) error) error
	Dox(context.Context, func(context.Context) error, pgx.TxOptions) error
}

type TransactionManagerImpl struct {
	pool    postgres.IPostgres
	log     logger.Logger
	metrics *metrics
}

func New(pool postgres.IPostgres, logger logger.Logger) TransactionManager {
	return NewWithOptions(pool, WithLogger(logger))
}

func NewWithOptions(pool postgres.IPostgres, opts ...Option) TransactionManager {
	resolved := buildOptions(opts...)

	return &TransactionManagerImpl{
		pool:    pool,
		log:     resolved.log.WithN("transaction_manager"),
		metrics: resolveMetrics(resolved.registerer),
	}
}

func (t TransactionManagerImpl) Do(ctx context.Context, f func(ctx context.Context) error) error {
	return t.execute(ctx, newDoMeta(), nil, f)
}

func (t TransactionManagerImpl) Dox(ctx context.Context, fn func(context.Context) error, opts pgx.TxOptions) error {
	return t.execute(ctx, newDoxMeta(opts), &opts, fn)
}

func (t TransactionManagerImpl) execute(ctx context.Context, meta txMeta, opts *pgx.TxOptions, fn func(context.Context) error) error {
	startedAt := time.Now()
	fields := []attribute.Field{
		attribute.String("method", meta.method),
		attribute.String("access_mode", meta.accessMode),
		attribute.String("isolation", meta.isolation),
	}

	if t.pool == nil {
		err := errors.New("transaction manager pool is nil")
		t.log.Error("Failed to execute transaction", append(fields, attribute.String("error", err.Error()))...)
		t.metrics.Observe(meta, "pool_missing", startedAt)
		return err
	}

	t.log.Debug("Acquiring master connection for transaction", fields...)
	connection, err := t.pool.MasterConnection(ctx)
	if err != nil {
		t.log.Error("Failed to acquire master connection for transaction", append(fields, attribute.String("error", err.Error()))...)
		t.metrics.Observe(meta, "acquire_connection_failed", startedAt)
		return err
	}
	defer connection.Close()

	tx, err := t.beginTransaction(ctx, connection, opts)
	if err != nil {
		t.log.Error("Failed to begin transaction", append(fields, attribute.String("error", err.Error()))...)
		t.metrics.Observe(meta, "begin_failed", startedAt)
		return err
	}

	t.metrics.IncActive(meta)
	defer t.metrics.DecActive(meta)

	t.log.Debug("Transaction started", fields...)

	defer func() {
		if recovered := recover(); recovered != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				t.log.Error("Failed to rollback transaction after panic", append(fields, attribute.String("error", rollbackErr.Error()))...)
				t.metrics.Observe(meta, "panic_rollback_failed", startedAt)
			} else {
				t.log.Warn("Rolled back transaction after panic", append(fields, attribute.String("panic", fmt.Sprint(recovered)))...)
				t.metrics.Observe(meta, "panic_rolled_back", startedAt)
			}
			panic(recovered)
		}
	}()

	var txExecutor postgres.SqlExecutor = tx
	ctx = context.WithValue(ctx, TxKey, txExecutor)
	err = fn(ctx)
	if err != nil {
		t.log.Warn("Transaction callback returned error, rolling back", append(fields, attribute.String("error", err.Error()))...)
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			t.log.Error("Failed to rollback transaction", append(fields, attribute.String("error", rollbackErr.Error()))...)
			t.metrics.Observe(meta, "rollback_failed", startedAt)
			return fmt.Errorf("%v: %w", rollbackErr, err)
		}

		t.metrics.Observe(meta, "rolled_back", startedAt)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.log.Error("Failed to commit transaction", append(fields, attribute.String("error", err.Error()))...)
		t.metrics.Observe(meta, "commit_failed", startedAt)
		return err
	}

	t.log.Debug("Transaction committed", fields...)
	t.metrics.Observe(meta, "committed", startedAt)

	return nil
}

func (t TransactionManagerImpl) beginTransaction(ctx context.Context, connection postgres.Connection, opts *pgx.TxOptions) (pgx.Tx, error) {
	if opts == nil {
		return connection.Begin(ctx)
	}

	return connection.BeginTx(ctx, *opts)
}
