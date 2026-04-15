package transaction_manager

import (
	"context"
	"errors"
	"fmt"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
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
		if errRoll := tx.Rollback(ctx); errRoll != nil && (errors.Is(errRoll, pgx.ErrTxClosed)) {
			t.log.Error(err.Error())
			return fmt.Errorf("%v: %w", errRoll, err)
		}
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		t.log.Error(err.Error())
		return err
	}
	return nil
}

func (t TransactionManagerImpl) Dox(ctx context.Context, fn func(context.Context) error, opts pgx.TxOptions) error {
	connection, err := t.pool.Connection(ctx)
	if err != nil {
		t.log.Error(err.Error())
		return err
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
		if errRoll := tx.Rollback(ctx); errRoll != nil && errors.Is(errRoll, pgx.ErrTxClosed) {
			t.log.Error(err.Error())
			return fmt.Errorf("%v: %w", errRoll, err)
		}
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		t.log.Error(err.Error())
		return err
	}
	return nil
}
