package postgres

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestNewWithNilConfigReturnsError(t *testing.T) {
	db, err := NewWithOptions(context.Background(), nil)
	require.ErrorIs(t, err, ErrNilConfig)
	require.Nil(t, db)
}

func TestMetricsCollectorReportsClusterDownWhenDatabaseUnavailable(t *testing.T) {
	registry := prometheus.NewRegistry()
	db, err := NewWithOptions(context.Background(), testMetricsConfig(), WithRegisterer(registry))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	metrics, err := registry.Gather()
	require.NoError(t, err)

	up := findGaugeValue(t, metrics, "postgres_cluster_up", map[string]string{
		"role":     "master",
		"host":     "127.0.0.1",
		"port":     "1",
		"database": "postgres",
	})
	require.Zero(t, up)
}

func TestMetricsCollectorCanRegisterMultipleInstancesInSingleRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()

	db1, err := NewWithOptions(context.Background(), testMetricsConfig(), WithRegisterer(registry))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db1.Close())
	})

	db2, err := NewWithOptions(context.Background(), testMetricsConfig(), WithRegisterer(registry))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db2.Close())
	})

	metrics, err := registry.Gather()
	require.NoError(t, err)

	require.Len(t, findMetrics(metrics, "postgres_cluster_up"), 2)
}

func testMetricsConfig() *Config {
	return &Config{
		Master: EndpointConfig{
			Host:     "127.0.0.1",
			Port:     "1",
			User:     "postgres",
			Password: "postgres",
			Database: "postgres",
			SSLMode:  "disable",
			Settings: PoolSettings{
				MaxOpenConnections: 1,
				ConnMaxLifetime:    1,
				MaxIdleConnections: 0,
				ConnMaxIdleTime:    1,
			},
		},
	}
}

func findGaugeValue(t *testing.T, metricFamilies []*dto.MetricFamily, name string, labels map[string]string) float64 {
	t.Helper()

	metrics := findMetrics(metricFamilies, name)
	require.NotEmpty(t, metrics)

	for _, metric := range metrics {
		if matchesLabels(metric, labels) && metric.Gauge != nil {
			return metric.Gauge.GetValue()
		}
	}

	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func findMetrics(metricFamilies []*dto.MetricFamily, name string) []*dto.Metric {
	for _, family := range metricFamilies {
		if family.GetName() == name {
			return family.GetMetric()
		}
	}

	return nil
}

func matchesLabels(metric *dto.Metric, labels map[string]string) bool {
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
