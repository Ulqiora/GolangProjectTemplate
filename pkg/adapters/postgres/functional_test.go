package postgres_test

import (
	"context"
	"fmt"
	"testing"

	transaction_manager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestFunctionalPostgresAndTransactionManagerFlowExportsMetrics(t *testing.T) {
	ctx := context.Background()
	registry := prometheus.NewRegistry()
	log := newTestLogger(t)
	env := newFunctionalPostgres(t, ctx, registry, log)
	db := env.db

	tableName := createIntegrationTable(t, ctx, db)
	txManager := transaction_manager.NewWithOptions(
		db,
		transaction_manager.WithLogger(log),
		transaction_manager.WithRegisterer(registry),
	)
	ctxManager := transaction_manager.NewCtxManager(db, log)

	err := txManager.Do(ctx, func(txCtx context.Context) error {
		executor, closeFn, err := ctxManager.GetDefaultOrTx(txCtx)
		require.NoError(t, err)
		defer closeFn()

		_, err = executor.Exec(txCtx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "committed-row")
		return err
	})
	require.NoError(t, err)

	err = txManager.Dox(ctx, func(txCtx context.Context) error {
		executor, closeFn, err := ctxManager.GetDefaultOrTx(txCtx)
		require.NoError(t, err)
		defer closeFn()

		_, err = executor.Exec(txCtx, fmt.Sprintf("INSERT INTO %s (name) VALUES ($1)", tableName), "rolled-back-row")
		return err
	}, pgx.TxOptions{
		AccessMode: pgx.ReadOnly,
	})
	require.Error(t, err)

	executor, closeFn, err := ctxManager.GetReadOnlyOrTx(ctx)
	require.NoError(t, err)
	defer closeFn()

	rows, err := executor.Query(ctx, fmt.Sprintf("SELECT name FROM %s ORDER BY id", tableName))
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"committed-row"}, names)

	metricFamilies, err := registry.Gather()
	require.NoError(t, err)

	require.Equal(t, 1.0, findCounterValue(t, metricFamilies, "transaction_manager_transactions_total", map[string]string{
		"method":      "do",
		"outcome":     "committed",
		"access_mode": "read_write",
		"isolation":   "default",
	}))
	require.Equal(t, 1.0, findCounterValue(t, metricFamilies, "transaction_manager_transactions_total", map[string]string{
		"method":      "dox",
		"outcome":     "rolled_back",
		"access_mode": "read_only",
		"isolation":   "default",
	}))
	require.Equal(t, 1.0, findGaugeValueByLabels(t, metricFamilies, "postgres_cluster_up", map[string]string{
		"role":     "master",
		"host":     env.host,
		"port":     env.port,
		"database": "postgres",
	}))
	requireMetricExists(t, metricFamilies, "postgres_pool_total_connections", map[string]string{
		"role":     "master",
		"host":     env.host,
		"port":     env.port,
		"database": "postgres",
	})
}
func findCounterValue(t *testing.T, metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	t.Helper()

	metrics := findMetricFamily(t, metricFamilies, name)
	for _, metric := range metrics.GetMetric() {
		if functionalMatchesLabels(metric, labels) && metric.Counter != nil {
			return metric.Counter.GetValue()
		}
	}

	t.Fatalf("counter metric %s with labels %v not found", name, labels)
	return 0
}

func findGaugeValueByLabels(t *testing.T, metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	t.Helper()

	metrics := findMetricFamily(t, metricFamilies, name)
	for _, metric := range metrics.GetMetric() {
		if functionalMatchesLabels(metric, labels) && metric.Gauge != nil {
			return metric.Gauge.GetValue()
		}
	}

	t.Fatalf("gauge metric %s with labels %v not found", name, labels)
	return 0
}

func requireMetricExists(t *testing.T, metricFamilies []*dto.MetricFamily, name string, labels map[string]string) {
	t.Helper()

	metrics := findMetricFamily(t, metricFamilies, name)
	for _, metric := range metrics.GetMetric() {
		if functionalMatchesLabels(metric, labels) {
			return
		}
	}

	t.Fatalf("metric %s with labels %v not found", name, labels)
}

func findMetricFamily(t *testing.T, metricFamilies []*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()

	for _, family := range metricFamilies {
		if family.GetName() == name {
			return family
		}
	}

	t.Fatalf("metric family %s not found", name)
	return nil
}

func functionalMatchesLabels(metric *dto.Metric, labels map[string]string) bool {
	for key, expected := range labels {
		found := false
		for _, label := range metric.GetLabel() {
			if label.GetName() == key && label.GetValue() == expected {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
