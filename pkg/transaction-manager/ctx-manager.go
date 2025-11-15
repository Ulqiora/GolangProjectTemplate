package transaction_manager

import (
	"context"
	"fmt"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
)

type CtxManager struct {
	pool   postgres.IPostgres
	logger logger.Logger
}

func NewCtxManager(pool postgres.IPostgres, logger logger.Logger) CtxManager {
	return CtxManager{
		pool:   pool,
		logger: logger,
	}
}

func (c CtxManager) GetDefaultOrTx(ctx context.Context) (postgres.SqlExecutor, func(), error) {
	value, ok := ctx.Value(TxKey).(postgres.SqlExecutor)
	if ok {
		fmt.Println("tx exec")
		return value, func() {}, nil
	}
	conn, err := c.pool.Connection(ctx)
	if err != nil {
		fmt.Println("err get connection")
		return nil, func() {}, err
	}
	fmt.Println("single connection exec")
	return conn, conn.Close, nil
}
