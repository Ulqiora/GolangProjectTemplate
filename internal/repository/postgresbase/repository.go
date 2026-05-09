package postgresbase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	transaction_manager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jackc/pgx/v5"
)

type Model interface {
	Params() map[string]interface{}
	Fields() []string
	PrimaryKey() (string, any)
}

type Repository[M Model] interface {
	SelectOne(ctx context.Context, sql string, args ...any) (M, error)
	Select(ctx context.Context, sql string, args ...any) ([]M, error)
	SelectBy(ctx context.Context, where goqu.Expression, opts ...SelectOption) ([]M, error)
	SelectOneBy(ctx context.Context, where goqu.Expression, opts ...SelectOption) (M, error)
	FindByPrimaryKey(ctx context.Context, id any) (M, error)
	FindByPrimaryKeys(ctx context.Context, ids []any, opts ...SelectOption) ([]M, error)
	Exists(ctx context.Context, where goqu.Expression) (bool, error)
	Count(ctx context.Context, where ...goqu.Expression) (int64, error)
	Create(ctx context.Context, m M) error
	CreateBatch(ctx context.Context, models []M) error
	CreateReturning(ctx context.Context, m M) (M, error)
	Upsert(ctx context.Context, m M, conflictTarget string, update map[string]interface{}) error
	UpsertReturning(ctx context.Context, m M, conflictTarget string, update map[string]interface{}) (M, error)
	UpsertByPrimaryKey(ctx context.Context, m M) error
	UpsertByPrimaryKeyReturning(ctx context.Context, m M) (M, error)
	Update(ctx context.Context, m M) error
	UpdateReturning(ctx context.Context, m M) (M, error)
	UpdateFieldsByPrimaryKey(ctx context.Context, id any, fields map[string]interface{}) error
	UpdateFieldsWhere(ctx context.Context, where goqu.Expression, fields map[string]interface{}) (int64, error)
	Delete(ctx context.Context, ids []any) error
	DeleteByPrimaryKey(ctx context.Context, id any) error
	DeleteByPrimaryKeyReturning(ctx context.Context, id any) (M, error)
	DeleteWhere(ctx context.Context, where goqu.Expression) (int64, error)
	DeleteWhereReturning(ctx context.Context, where goqu.Expression) ([]M, error)
	SelectForLock(ctx context.Context, where goqu.Expression, lock LockOptions, opts ...SelectOption) ([]M, error)
	SelectOneForLock(ctx context.Context, where goqu.Expression, lock LockOptions, opts ...SelectOption) (M, error)
	LockByPrimaryKey(ctx context.Context, id any, lock LockOptions) (M, error)
	LockByPrimaryKeys(ctx context.Context, ids []any, lock LockOptions, opts ...SelectOption) ([]M, error)
	AdvisoryTransactionLock(ctx context.Context, key int64) error
	TryAdvisoryTransactionLock(ctx context.Context, key int64) (bool, error)
	AdvisoryTransactionLock2(ctx context.Context, key1 int32, key2 int32) error
	TryAdvisoryTransactionLock2(ctx context.Context, key1 int32, key2 int32) (bool, error)
}

type RepositoryImpl[M Model] struct {
	ctxManager transaction_manager.CtxManager
	dialect    goqu.DialectWrapper
	tableName  string
}

type LockMode string

const (
	LockForUpdate      LockMode = "FOR UPDATE"
	LockForNoKeyUpdate LockMode = "FOR NO KEY UPDATE"
	LockForShare       LockMode = "FOR SHARE"
	LockForKeyShare    LockMode = "FOR KEY SHARE"
)

type LockWaitPolicy string

const (
	LockWait       LockWaitPolicy = ""
	LockNoWait     LockWaitPolicy = "NOWAIT"
	LockSkipLocked LockWaitPolicy = "SKIP LOCKED"
)

type LockOptions struct {
	Mode       LockMode
	WaitPolicy LockWaitPolicy
	OfTables   []string
}

type OrderBy struct {
	Column string
	Desc   bool
}

type SelectOptions struct {
	Limit   uint
	Offset  uint
	OrderBy []OrderBy
}

type SelectOption func(*SelectOptions)

func WithLimit(limit uint) SelectOption {
	return func(opts *SelectOptions) {
		opts.Limit = limit
	}
}

func WithOffset(offset uint) SelectOption {
	return func(opts *SelectOptions) {
		opts.Offset = offset
	}
}

func WithOrderBy(orderBy ...OrderBy) SelectOption {
	return func(opts *SelectOptions) {
		opts.OrderBy = append(opts.OrderBy, orderBy...)
	}
}

func Asc(column string) OrderBy {
	return OrderBy{Column: column}
}

func Desc(column string) OrderBy {
	return OrderBy{Column: column, Desc: true}
}

func init() {
	options := goqu.DefaultDialectOptions()
	options.PlaceHolderFragment = []byte("$")
	options.IncludePlaceholderNum = true
	goqu.RegisterDialect("postgres", options)
}

func NewRepository[M Model](pool postgres.IPostgres, tableName string) Repository[M] {
	return &RepositoryImpl[M]{
		ctxManager: transaction_manager.NewCtxManager(pool, logger.DefaultLogger()),
		dialect:    goqu.Dialect("postgres"),
		tableName:  tableName,
	}
}

func (repo *RepositoryImpl[M]) SelectBy(ctx context.Context, where goqu.Expression, opts ...SelectOption) ([]M, error) {
	sql, args, err := repo.selectDataset(where, opts...).Prepared(true).ToSQL()
	if err != nil {
		return nil, err
	}
	return repo.Select(ctx, sql, args...)
}

func (repo *RepositoryImpl[M]) SelectOneBy(ctx context.Context, where goqu.Expression, opts ...SelectOption) (M, error) {
	sql, args, err := repo.selectDataset(where, append(opts, WithLimit(1))...).Prepared(true).ToSQL()
	if err != nil {
		var result M
		return result, err
	}
	return repo.SelectOne(ctx, sql, args...)
}

func (repo *RepositoryImpl[M]) FindByPrimaryKey(ctx context.Context, id any) (M, error) {
	primaryKey := repo.primaryKeyColumn()
	return repo.SelectOneBy(ctx, goqu.T(repo.tableName).Col(primaryKey).Eq(id), WithLimit(1))
}

func (repo *RepositoryImpl[M]) FindByPrimaryKeys(ctx context.Context, ids []any, opts ...SelectOption) ([]M, error) {
	if len(ids) == 0 {
		return []M{}, nil
	}

	primaryKey := repo.primaryKeyColumn()
	return repo.SelectBy(ctx, goqu.T(repo.tableName).Col(primaryKey).In(ids), opts...)
}

func (repo *RepositoryImpl[M]) Exists(ctx context.Context, where goqu.Expression) (bool, error) {
	sql, args, err := repo.dialect.From(repo.tableName).
		Select(goqu.L("1")).
		Where(where).
		Limit(1).
		Prepared(true).
		ToSQL()
	if err != nil {
		return false, err
	}

	connection, closeConnection, err := repo.ctxManager.GetReadOnlyOrTx(ctx)
	if err != nil {
		return false, err
	}
	defer closeConnection()

	var exists int
	err = connection.QueryRow(ctx, sql, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (repo *RepositoryImpl[M]) Count(ctx context.Context, where ...goqu.Expression) (int64, error) {
	dataset := repo.dialect.From(repo.tableName).Select(goqu.COUNT("*"))
	if len(where) > 0 {
		dataset = dataset.Where(where...)
	}

	sql, args, err := dataset.Prepared(true).ToSQL()
	if err != nil {
		return 0, err
	}

	connection, closeConnection, err := repo.ctxManager.GetReadOnlyOrTx(ctx)
	if err != nil {
		return 0, err
	}
	defer closeConnection()

	var count int64
	err = connection.QueryRow(ctx, sql, args...).Scan(&count)
	return count, err
}

func (repo *RepositoryImpl[M]) SelectOne(ctx context.Context, sql string, args ...any) (M, error) {
	var result M

	connection, closeConnection, err := repo.ctxManager.GetReadOnlyOrTx(ctx)
	if err != nil {
		return result, err
	}
	defer closeConnection()

	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[M])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return result, ports.ErrNotFound
		}
		return result, err
	}
	return result, nil
}

func (repo *RepositoryImpl[M]) Select(ctx context.Context, sql string, args ...any) ([]M, error) {
	connection, closeConnection, err := repo.ctxManager.GetReadOnlyOrTx(ctx)
	if err != nil {
		return nil, err
	}
	defer closeConnection()

	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[M])
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl[M]) Update(ctx context.Context, m M) error {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer closeConnection()

	primaryKey, primaryValue := m.PrimaryKey()
	sql, args, err := repo.dialect.Update(repo.tableName).
		Set(m.Params()).
		Where(goqu.T(repo.tableName).Col(primaryKey).Eq(primaryValue)).
		Prepared(true).
		ToSQL()
	if err != nil {
		return err
	}

	affected, err := connection.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if affected.RowsAffected() == 0 {
		return ports.ErrNoAffectedRows
	}
	return nil
}

func (repo *RepositoryImpl[M]) UpdateReturning(ctx context.Context, m M) (M, error) {
	var result M

	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return result, err
	}
	defer closeConnection()

	primaryKey, primaryValue := m.PrimaryKey()
	sql, args, err := repo.dialect.Update(repo.tableName).
		Set(m.Params()).
		Where(goqu.T(repo.tableName).Col(primaryKey).Eq(primaryValue)).
		Returning(goqu.Star()).
		Prepared(true).
		ToSQL()
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
		if errors.Is(err, pgx.ErrNoRows) {
			return result, ports.ErrNoAffectedRows
		}
		return result, err
	}
	return result, nil
}

func (repo *RepositoryImpl[M]) UpdateFieldsByPrimaryKey(ctx context.Context, id any, fields map[string]interface{}) error {
	primaryKey := repo.primaryKeyColumn()
	affected, err := repo.UpdateFieldsWhere(ctx, goqu.T(repo.tableName).Col(primaryKey).Eq(id), fields)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ports.ErrNoAffectedRows
	}
	return nil
}

func (repo *RepositoryImpl[M]) UpdateFieldsWhere(ctx context.Context, where goqu.Expression, fields map[string]interface{}) (int64, error) {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return 0, err
	}
	defer closeConnection()

	if len(fields) == 0 {
		return 0, nil
	}

	sql, args, err := repo.dialect.Update(repo.tableName).
		Set(fields).
		Where(where).
		Prepared(true).
		ToSQL()
	if err != nil {
		return 0, err
	}

	affected, err := connection.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return affected.RowsAffected(), nil
}

func (repo *RepositoryImpl[M]) Create(ctx context.Context, m M) error {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer closeConnection()

	sql, args, err := repo.dialect.Insert(repo.tableName).
		Rows(m.Params()).
		Prepared(true).
		ToSQL()
	if err != nil {
		return err
	}

	_, err = connection.Exec(ctx, sql, args...)
	return err
}

func (repo *RepositoryImpl[M]) CreateReturning(ctx context.Context, m M) (M, error) {
	var result M

	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return result, err
	}
	defer closeConnection()

	sql, args, err := repo.dialect.Insert(repo.tableName).
		Rows(m.Params()).
		Returning(goqu.Star()).
		Prepared(true).
		ToSQL()
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

func (repo *RepositoryImpl[M]) Upsert(ctx context.Context, m M, conflictTarget string, update map[string]interface{}) error {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer closeConnection()

	sql, args, err := repo.upsertDataset(m, conflictTarget, update).Prepared(true).ToSQL()
	if err != nil {
		return err
	}

	_, err = connection.Exec(ctx, sql, args...)
	return err
}

func (repo *RepositoryImpl[M]) UpsertReturning(ctx context.Context, m M, conflictTarget string, update map[string]interface{}) (M, error) {
	var result M

	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return result, err
	}
	defer closeConnection()

	sql, args, err := repo.upsertDataset(m, conflictTarget, update).
		Returning(goqu.Star()).
		Prepared(true).
		ToSQL()
	if err != nil {
		return result, err
	}

	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	result, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[M])
	return result, err
}

func (repo *RepositoryImpl[M]) UpsertByPrimaryKey(ctx context.Context, m M) error {
	primaryKey, _ := m.PrimaryKey()
	return repo.Upsert(ctx, m, primaryKey, paramsWithout(m.Params(), primaryKey))
}

func (repo *RepositoryImpl[M]) UpsertByPrimaryKeyReturning(ctx context.Context, m M) (M, error) {
	primaryKey, _ := m.PrimaryKey()
	return repo.UpsertReturning(ctx, m, primaryKey, paramsWithout(m.Params(), primaryKey))
}

func (repo *RepositoryImpl[M]) CreateBatch(ctx context.Context, models []M) error {
	if len(models) == 0 {
		return nil
	}

	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer closeConnection()

	rows := make([]interface{}, 0, len(models))
	for _, model := range models {
		rows = append(rows, model.Params())
	}

	sql, args, err := repo.dialect.Insert(repo.tableName).
		Rows(rows...).
		Prepared(true).
		ToSQL()
	if err != nil {
		return err
	}

	_, err = connection.Exec(ctx, sql, args...)
	return err
}

func (repo *RepositoryImpl[M]) Delete(ctx context.Context, ids []any) error {
	if len(ids) == 0 {
		return nil
	}

	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer closeConnection()

	primaryKey := repo.primaryKeyColumn()
	sql, args, err := repo.dialect.Delete(repo.tableName).
		Where(goqu.T(repo.tableName).Col(primaryKey).In(ids)).
		Prepared(true).
		ToSQL()
	if err != nil {
		return err
	}

	affected, err := connection.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if affected.RowsAffected() == 0 {
		return ports.ErrNoAffectedRows
	}
	return nil
}

func (repo *RepositoryImpl[M]) DeleteByPrimaryKey(ctx context.Context, id any) error {
	return repo.Delete(ctx, []any{id})
}

func (repo *RepositoryImpl[M]) DeleteByPrimaryKeyReturning(ctx context.Context, id any) (M, error) {
	primaryKey := repo.primaryKeyColumn()
	return repo.deleteOneReturning(ctx, goqu.T(repo.tableName).Col(primaryKey).Eq(id))
}

func (repo *RepositoryImpl[M]) DeleteWhere(ctx context.Context, where goqu.Expression) (int64, error) {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return 0, err
	}
	defer closeConnection()

	sql, args, err := repo.dialect.Delete(repo.tableName).
		Where(where).
		Prepared(true).
		ToSQL()
	if err != nil {
		return 0, err
	}

	affected, err := connection.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return affected.RowsAffected(), nil
}

func (repo *RepositoryImpl[M]) DeleteWhereReturning(ctx context.Context, where goqu.Expression) ([]M, error) {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return nil, err
	}
	defer closeConnection()

	sql, args, err := repo.dialect.Delete(repo.tableName).
		Where(where).
		Returning(goqu.Star()).
		Prepared(true).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[M])
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl[M]) SelectForLock(ctx context.Context, where goqu.Expression, lock LockOptions, opts ...SelectOption) ([]M, error) {
	sql, args, err := repo.selectForLockSQL(where, lock, opts...)
	if err != nil {
		return nil, err
	}

	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return nil, err
	}
	defer closeConnection()

	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[M])
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (repo *RepositoryImpl[M]) LockByPrimaryKey(ctx context.Context, id any, lock LockOptions) (M, error) {
	primaryKey := repo.primaryKeyColumn()
	return repo.SelectOneForLock(ctx, goqu.T(repo.tableName).Col(primaryKey).Eq(id), lock, WithLimit(1))
}

func (repo *RepositoryImpl[M]) LockByPrimaryKeys(ctx context.Context, ids []any, lock LockOptions, opts ...SelectOption) ([]M, error) {
	if len(ids) == 0 {
		return []M{}, nil
	}

	primaryKey := repo.primaryKeyColumn()
	return repo.SelectForLock(ctx, goqu.T(repo.tableName).Col(primaryKey).In(ids), lock, opts...)
}

func (repo *RepositoryImpl[M]) SelectOneForLock(ctx context.Context, where goqu.Expression, lock LockOptions, opts ...SelectOption) (M, error) {
	sql, args, err := repo.selectForLockSQL(where, lock, append(opts, WithLimit(1))...)
	if err != nil {
		var result M
		return result, err
	}

	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		var result M
		return result, err
	}
	defer closeConnection()

	rows, err := connection.Query(ctx, sql, args...)
	if err != nil {
		var result M
		return result, err
	}
	defer rows.Close()

	result, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[M])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return result, ports.ErrNotFound
		}
		return result, err
	}
	return result, nil
}

func (repo *RepositoryImpl[M]) AdvisoryTransactionLock(ctx context.Context, key int64) error {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer closeConnection()

	_, err = connection.Exec(ctx, "select pg_advisory_xact_lock($1)", key)
	return err
}

func (repo *RepositoryImpl[M]) TryAdvisoryTransactionLock(ctx context.Context, key int64) (bool, error) {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return false, err
	}
	defer closeConnection()

	var locked bool
	err = connection.QueryRow(ctx, "select pg_try_advisory_xact_lock($1)", key).Scan(&locked)
	return locked, err
}

func (repo *RepositoryImpl[M]) AdvisoryTransactionLock2(ctx context.Context, key1 int32, key2 int32) error {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return err
	}
	defer closeConnection()

	_, err = connection.Exec(ctx, "select pg_advisory_xact_lock($1, $2)", key1, key2)
	return err
}

func (repo *RepositoryImpl[M]) TryAdvisoryTransactionLock2(ctx context.Context, key1 int32, key2 int32) (bool, error) {
	connection, closeConnection, err := repo.ctxManager.GetDefaultOrTx(ctx)
	if err != nil {
		return false, err
	}
	defer closeConnection()

	var locked bool
	err = connection.QueryRow(ctx, "select pg_try_advisory_xact_lock($1, $2)", key1, key2).Scan(&locked)
	return locked, err
}

func (repo *RepositoryImpl[M]) selectDataset(where goqu.Expression, opts ...SelectOption) *goqu.SelectDataset {
	dataset := repo.dialect.From(repo.tableName).Select(goqu.Star())
	if where != nil {
		dataset = dataset.Where(where)
	}
	return applySelectOptions(dataset, opts...)
}

func (repo *RepositoryImpl[M]) upsertDataset(m M, conflictTarget string, update map[string]interface{}) *goqu.InsertDataset {
	if len(update) == 0 {
		return repo.dialect.Insert(repo.tableName).
			Rows(m.Params()).
			OnConflict(goqu.DoNothing())
	}

	return repo.dialect.Insert(repo.tableName).
		Rows(m.Params()).
		OnConflict(goqu.DoUpdate(conflictTarget, update))
}

func (repo *RepositoryImpl[M]) deleteOneReturning(ctx context.Context, where goqu.Expression) (M, error) {
	var result M

	rows, err := repo.DeleteWhereReturning(ctx, where)
	if err != nil {
		return result, err
	}
	if len(rows) == 0 {
		return result, ports.ErrNotFound
	}
	return rows[0], nil
}

func (repo *RepositoryImpl[M]) selectForLockSQL(where goqu.Expression, lock LockOptions, opts ...SelectOption) (string, []any, error) {
	sql, args, err := repo.selectDataset(where, opts...).Prepared(true).ToSQL()
	if err != nil {
		return "", nil, err
	}

	lockSQL, err := lock.SQL()
	if err != nil {
		return "", nil, err
	}
	return sql + " " + lockSQL, args, nil
}

func (repo *RepositoryImpl[M]) primaryKeyColumn() string {
	model := newModel[M]()
	column, _ := model.PrimaryKey()
	return column
}

func newModel[M Model]() M {
	var zero M
	modelType := reflect.TypeOf(zero)
	if modelType != nil && modelType.Kind() == reflect.Ptr {
		return reflect.New(modelType.Elem()).Interface().(M)
	}
	return zero
}

func paramsWithout(params map[string]interface{}, columns ...string) map[string]interface{} {
	result := make(map[string]interface{}, len(params))
	for key, value := range params {
		result[key] = value
	}
	for _, column := range columns {
		delete(result, column)
	}
	return result
}

func applySelectOptions(dataset *goqu.SelectDataset, opts ...SelectOption) *goqu.SelectDataset {
	resolved := SelectOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&resolved)
		}
	}

	if len(resolved.OrderBy) > 0 {
		expressions := make([]exp.OrderedExpression, 0, len(resolved.OrderBy))
		for _, orderBy := range resolved.OrderBy {
			if orderBy.Column == "" {
				continue
			}
			column := goqu.C(orderBy.Column)
			if orderBy.Desc {
				expressions = append(expressions, column.Desc())
			} else {
				expressions = append(expressions, column.Asc())
			}
		}
		if len(expressions) > 0 {
			dataset = dataset.Order(expressions...)
		}
	}
	if resolved.Limit > 0 {
		dataset = dataset.Limit(resolved.Limit)
	}
	if resolved.Offset > 0 {
		dataset = dataset.Offset(resolved.Offset)
	}
	return dataset
}

func (lock LockOptions) SQL() (string, error) {
	mode := lock.Mode
	if mode == "" {
		mode = LockForUpdate
	}

	switch mode {
	case LockForUpdate, LockForNoKeyUpdate, LockForShare, LockForKeyShare:
	default:
		return "", fmt.Errorf("unsupported postgres lock mode: %s", mode)
	}

	var builder strings.Builder
	builder.WriteString(string(mode))
	if len(lock.OfTables) > 0 {
		builder.WriteString(" OF ")
		for index, table := range lock.OfTables {
			if strings.TrimSpace(table) == "" {
				return "", errors.New("lock table name is empty")
			}
			if strings.ContainsAny(table, " \t\n\r;\"'") {
				return "", fmt.Errorf("unsafe lock table name: %s", table)
			}
			if index > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(table)
		}
	}

	switch lock.WaitPolicy {
	case LockWait:
	case LockNoWait, LockSkipLocked:
		builder.WriteString(" ")
		builder.WriteString(string(lock.WaitPolicy))
	default:
		return "", fmt.Errorf("unsupported postgres lock wait policy: %s", lock.WaitPolicy)
	}

	return builder.String(), nil
}
