package transaction_manager

import (
	"context"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
)

type CtxManager struct {
	pool   postgres.IPostgres
	logger logger.Logger
}

func NewCtxManager(pool postgres.IPostgres, logger logger.Logger) CtxManager {
	return CtxManager{
		pool:   pool,
		logger: resolveLogger(logger).WithN("transaction_manager_ctx"),
	}
}

func (c CtxManager) GetDefaultOrTx(ctx context.Context) (postgres.SqlExecutor, func(), error) {
	return c.getConnectionOrTx(ctx, "master", c.pool.MasterConnection)
}

func (c CtxManager) GetReadOnlyOrTx(ctx context.Context) (postgres.SqlExecutor, func(), error) {
	return c.getConnectionOrTx(ctx, "read_only", c.pool.ReadOnlyConnection)
}

func (c CtxManager) getConnectionOrTx(
	ctx context.Context,
	source string,
	connectionProvider func(context.Context) (postgres.Connection, error),
) (postgres.SqlExecutor, func(), error) {
	value, ok := ctx.Value(TxKey).(postgres.SqlExecutor)
	if ok {
		c.logger.Debug("Using transaction executor from context", attribute.String("source", "transaction"))
		return value, func() {}, nil
	}
	conn, err := connectionProvider(ctx)
	if err != nil {
		c.logger.Error("Failed to acquire executor connection", attribute.String("source", source), attribute.String("error", err.Error()))
		return nil, func() {}, err
	}
	c.logger.Debug("Acquired executor connection", attribute.String("source", source))
	return conn, conn.Close, nil
}
