package transaction_manager

import (
	"context"
	"errors"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	"github.com/jackc/pgx/v5"
)

const (
	TxKey = "transaction-key"
)

type TransactionManager interface {
	Do(context.Context, func(context.Context) error) error
	Dox(context.Context, func(context.Context) error, pgx.TxOptions)
}

type TransactionManagerImpl struct {
	pool postgres.IPostgres
	log  logger.Logger
}

func New(pool postgres.IPostgres, logger logger.Logger) TransactionManager {
	return &TransactionManagerImpl{
		pool: pool,
		log:  logger,
	}
}

func (t TransactionManagerImpl) Do(ctx context.Context, f func(ctx context.Context) error) error {
	connection, err := t.pool.Connection(ctx)
	if err != nil {
		return err
	}
	tx, err := connection.Begin(ctx)
	if err != nil {
		return err
	}
	var txWrap postgres.SqlExecutor = tx
	ctxTx := context.WithValue(ctx, TxKey, txWrap)

	err = f(ctxTx)
	if err != nil {
		if err = tx.Rollback(ctxTx); err != nil && (errors.Is(err, pgx.ErrTxClosed)) {
			t.log.Error(err.Error())
		}
		return err
	}
	err = tx.Commit(ctxTx)
	if err != nil {
		if err = tx.Rollback(ctx); err != nil && (errors.Is(err, pgx.ErrTxClosed)) {
			t.log.Error(err.Error())
		}
	}
	return nil
}

func (t TransactionManagerImpl) Dox(ctx context.Context, fn func(context.Context) error, opts pgx.TxOptions) {
	connection, err := t.pool.Connection(ctx)
	if err != nil {
		t.log.Error(err.Error())
		return
	}
	tx, err := connection.BeginTx(ctx, opts)
	defer func() {
		if err = tx.Rollback(ctx); err != nil && (errors.Is(err, pgx.ErrTxClosed)) {
			t.log.Error(err.Error())
		}
	}()
	ctx = context.WithValue(ctx, TxKey, tx)
	err = fn(ctx)
	if err != nil {
		if err = tx.Rollback(ctx); err != nil && (errors.Is(err, pgx.ErrTxClosed)) {
			t.log.Error(err.Error())
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		t.log.Error(err.Error())
	}
}
