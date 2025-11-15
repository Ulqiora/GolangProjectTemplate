package ports

import (
	"context"
	"fmt"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	transaction_manager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type BaseRepositoryImpl[M BaseModel] struct {
	pool         postgres.IPostgres
	ctxManager   transaction_manager.CtxManager
	generateFunc func() M
	tableName    string
}

func NewBaseRepository[M BaseModel](pool postgres.IPostgres, tablename string, generateFunc func() M) BaseRepository[M] {
	return &BaseRepositoryImpl[M]{
		pool:         pool,
		ctxManager:   transaction_manager.NewCtxManager(pool, logger.DefaultLogger()),
		generateFunc: generateFunc,
		tableName:    tablename,
	}
}

func (repo *BaseRepositoryImpl[M]) SelectOne(ctx context.Context, sql string, args ...any) (M, error) {
	var result M
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	defer fnClose()
	if err != nil {
		return result, err
	}
	row := connection.QueryRow(ctx, sql, args...)

	result = repo.generateFunc()
	err = row.Scan(result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (repo *BaseRepositoryImpl[M]) Select(ctx context.Context, sql string, args ...any) ([]M, error) {
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	defer fnClose()
	if err != nil {
		return nil, err
	}

	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fields := repo.fields(rows)

	var objects []M

	for rows.Next() {
		object := repo.generateFunc()
		err := object.Scan(fields, rows.Scan)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func (repo *BaseRepositoryImpl[M]) Create(ctx context.Context, m M) (M, error) {
	connection, fnClose, err := repo.ctxManager.GetDefaultOrTx(ctx)
	defer fnClose()
	if err != nil {
		fmt.Println("err fetch connection ")
		return repo.generateFunc(), err
	}
	values := m.Params()

	sql, args, err2 := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Insert(repo.tableName).
		SetMap(values).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err2 != nil {
		fmt.Println("err build sql build")
		return repo.generateFunc(), err2
	}

	if _, err = connection.Exec(ctx, sql, args...); err != nil {
		fmt.Println("err exec object: ", err.Error())
		return repo.generateFunc(), err
	}
	fmt.Println("complete create object")
	return m, nil
}

func (repo *BaseRepositoryImpl[M]) fields(rows pgx.Rows) []string {
	columns := rows.FieldDescriptions()
	fields := make([]string, 0, len(columns))

	for _, d := range columns {
		fields = append(fields, d.Name)
	}
	return fields
}
