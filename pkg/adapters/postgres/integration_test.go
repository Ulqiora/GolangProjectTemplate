package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/logger"
	transaction_manager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestPostgresConnectionExecQueryAndReadOnlyConnection(t *testing.T) {
	ctx := context.Background()
	db := newIntegrationPostgres(t, ctx)

	tableName := createIntegrationTable(t, ctx, db)

	masterConn, err := db.MasterConnection(ctx)
	require.NoError(t, err)
	defer masterConn.Close()

	_, err = masterConn.Exec(ctx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1), ($2)", tableName), "first", "second")
	require.NoError(t, err)

	var count int
	err = masterConn.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	readOnlyConn, err := db.ReadOnlyConnection(ctx)
	require.NoError(t, err)
	defer readOnlyConn.Close()

	rows, err := readOnlyConn.Query(ctx, fmt.Sprintf("SELECT name FROM %s ORDER BY id", tableName))
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"first", "second"}, names)
}

func TestPostgresQueriesInsideTransactionManager(t *testing.T) {
	ctx := context.Background()
	db := newIntegrationPostgres(t, ctx)

	tableName := createIntegrationTable(t, ctx, db)
	log := newTestLogger(t)
	txManager := transaction_manager.New(db, log)
	ctxManager := transaction_manager.NewCtxManager(db, log)

	err := txManager.Do(ctx, func(txCtx context.Context) error {
		executor, closeFn, err := ctxManager.GetDefaultOrTx(txCtx)
		require.NoError(t, err)
		defer closeFn()

		_, err = executor.Exec(txCtx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "committed")
		return err
	})
	require.NoError(t, err)

	requireNames(t, ctx, db, tableName, []string{"committed"})

	err = txManager.Do(ctx, func(txCtx context.Context) error {
		executor, closeFn, err := ctxManager.GetDefaultOrTx(txCtx)
		require.NoError(t, err)
		defer closeFn()

		_, err = executor.Exec(txCtx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "rolled-back")
		require.NoError(t, err)

		return fmt.Errorf("force rollback")
	})
	require.Error(t, err)

	requireNames(t, ctx, db, tableName, []string{"committed"})
}

func TestPostgresDoxUsesTransactionOptions(t *testing.T) {
	ctx := context.Background()
	db := newIntegrationPostgres(t, ctx)

	tableName := createIntegrationTable(t, ctx, db)
	log := newTestLogger(t)
	txManager := transaction_manager.New(db, log)
	ctxManager := transaction_manager.NewCtxManager(db, log)

	err := txManager.Dox(ctx, func(txCtx context.Context) error {
		executor, closeFn, err := ctxManager.GetDefaultOrTx(txCtx)
		require.NoError(t, err)
		defer closeFn()

		_, err = executor.Exec(txCtx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "read-only-insert")
		return err
	}, pgx.TxOptions{
		AccessMode: pgx.ReadOnly,
	})
	require.Error(t, err)

	requireNames(t, ctx, db, tableName, nil)

	err = txManager.Dox(ctx, func(txCtx context.Context) error {
		executor, closeFn, err := ctxManager.GetDefaultOrTx(txCtx)
		require.NoError(t, err)
		defer closeFn()

		_, err = executor.Exec(txCtx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "read-write-insert")
		return err
	}, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	require.NoError(t, err)

	requireNames(t, ctx, db, tableName, []string{"read-write-insert"})
}

func TestPostgresReadOnlyConnectionFallsBackToMasterWhenReadOnlyIsNotConfigured(t *testing.T) {
	ctx := context.Background()
	db := newIntegrationPostgresWithoutReadOnly(t, ctx)

	tableName := createIntegrationTable(t, ctx, db)

	masterConn, err := db.MasterConnection(ctx)
	require.NoError(t, err)
	defer masterConn.Close()

	_, err = masterConn.Exec(ctx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "master-write")
	require.NoError(t, err)

	readOnlyConn, err := db.ReadOnlyConnection(ctx)
	require.NoError(t, err)
	defer readOnlyConn.Close()

	var name string
	err = readOnlyConn.QueryRow(ctx, fmt.Sprintf("SELECT name FROM %s", tableName)).Scan(&name)
	require.NoError(t, err)
	require.Equal(t, "master-write", name)
}

func TestPostgresTransactionManagerRollsBackOnPanic(t *testing.T) {
	ctx := context.Background()
	db := newIntegrationPostgres(t, ctx)

	tableName := createIntegrationTable(t, ctx, db)
	log := newTestLogger(t)
	txManager := transaction_manager.New(db, log)
	ctxManager := transaction_manager.NewCtxManager(db, log)

	require.Panics(t, func() {
		_ = txManager.Do(ctx, func(txCtx context.Context) error {
			executor, closeFn, err := ctxManager.GetDefaultOrTx(txCtx)
			require.NoError(t, err)
			defer closeFn()

			_, err = executor.Exec(txCtx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "panic-row")
			require.NoError(t, err)

			panic("boom")
		})
	})

	requireNames(t, ctx, db, tableName, nil)
}

func integrationEndpoint(host string, port string) *postgres.EndpointConfig {
	return &postgres.EndpointConfig{
		Host:     host,
		Port:     port,
		User:     "postgres",
		Password: "postgres",
		Database: "postgres",
		SSLMode:  "disable",
		Settings: postgres.PoolSettings{
			MaxOpenConnections: 4,
			ConnMaxLifetime:    30,
			MaxIdleConnections: 1,
			ConnMaxIdleTime:    30,
		},
	}
}

func createIntegrationTable(t *testing.T, ctx context.Context, db *postgres.Postgres) string {
	t.Helper()

	tableName := fmt.Sprintf("postgres_pkg_test_%d", time.Now().UnixNano())
	conn, err := db.MasterConnection(ctx)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE %s (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL
		)
	`, tableName))
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cleanupConn, err := db.MasterConnection(cleanupCtx)
		if err != nil {
			return
		}
		defer cleanupConn.Close()

		_, _ = cleanupConn.Exec(cleanupCtx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	})

	return tableName
}

func requireNames(t *testing.T, ctx context.Context, db *postgres.Postgres, tableName string, expected []string) {
	t.Helper()

	conn, err := db.ReadOnlyConnection(ctx)
	require.NoError(t, err)
	defer conn.Close()

	rows, err := conn.Query(ctx, fmt.Sprintf("SELECT name FROM %s ORDER BY id", tableName))
	require.NoError(t, err)
	defer rows.Close()

	var actual []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		actual = append(actual, name)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, expected, actual)
}

func newTestLogger(t *testing.T) logger.Logger {
	t.Helper()

	log, err := logger.NewLogger(logger.EnvStage, nil)
	require.NoError(t, err)

	return log
}
