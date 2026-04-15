package ports

import (
	"context"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	transaction_manager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v5"
)

type BaseRepositoryImpl[M BaseModel] struct {
	pool       postgres.IPostgres
	ctxManager transaction_manager.CtxManager
	dialect    goqu.DialectWrapper
	tableName  string
}

func NewBaseRepository[M BaseModel](pool postgres.IPostgres, tablename string) BaseRepository[M] {
	return &BaseRepositoryImpl[M]{
		pool:       pool,
		ctxManager: transaction_manager.NewCtxManager(pool, logger.DefaultLogger()),
		dialect:    goqu.Dialect("postgres"),
		tableName:  tablename,
	}
}

func (repo *BaseRepositoryImpl[M]) SelectOne(ctx context.Context, sql string, args ...any) (M, error) {
	var result M
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	defer fnClose()
	if err != nil {
		return result, err
	}
	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[M])
	if err != nil {
		return result, err
	}
	return result, nil
}

func (repo *BaseRepositoryImpl[M]) Select(ctx context.Context, sql string, args ...any) ([]M, error) {
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return nil, err
	}
	defer fnClose()
	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[M])
	if err != nil {
		return result, err
	}
	return result, nil
}

func (repo *BaseRepositoryImpl[M]) Update(ctx context.Context, m M) error {
	connenction, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer fnClose()
	nameKey, key := m.PrimaryKey()
	sql, args, err := repo.dialect.Update(repo.tableName).
		Set(m).Where(
		goqu.T(repo.tableName).Col(nameKey).Eq(key),
	).Prepared(true).ToSQL()
	if err != nil {
		return err
	}
	affected, err := connenction.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if affected.RowsAffected() == 0 {
		return ErrNoAffectedRows
	}
	return nil
}

func (repo *BaseRepositoryImpl[M]) Create(ctx context.Context, m M) error {
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer fnClose()
	sql, args, err := repo.dialect.Insert(repo.tableName).
		Rows(m).
		Prepared(true).
		ToSQL()

	if err != nil {
		//fmt.Println("err build sql build")
		return err
	}

	if _, err = connection.Exec(ctx, sql, args...); err != nil {
		//fmt.Println("err exec object: ", err.Error())
		return err
	}
	//fmt.Println("complete create object")
	return nil
}

func (repo *BaseRepositoryImpl[M]) CreateBatch(ctx context.Context, m []M) error {
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer fnClose()
	sql, args, err := repo.dialect.Insert(repo.tableName).
		Rows(m).
		Prepared(true).
		ToSQL()

	if err != nil {
		//fmt.Println("err build sql build")
		return err
	}

	if _, err = connection.Exec(ctx, sql, args...); err != nil {
		//fmt.Println("err exec object: ", err.Error())
		return err
	}
	//fmt.Println("complete create object")
	return nil
}

func (repo *BaseRepositoryImpl[M]) Delete(ctx context.Context, ids []any) error {
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer fnClose()

	sql, args, err := repo.dialect.Delete(repo.tableName).
		Where(goqu.T(repo.tableName).Col("id").In(ids)).
		Prepared(true).
		ToSQL()

	if err != nil {
		return err
	}

	if _, err = connection.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return nil
}
