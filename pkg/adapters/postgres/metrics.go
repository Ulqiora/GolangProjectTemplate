package postgres

import (
	"context"
	"errors"
	"time"

	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

const metricsScrapeTimeout = 5 * time.Second

type poolCollector struct {
	postgres *Postgres
	descs    map[string]*prometheus.Desc
}

type clusterSnapshot struct {
	InRecovery         float64
	MaxConnections     int64
	DatabaseSizeBytes  int64
	NumBackends        int64
	XactCommit         int64
	XactRollback       int64
	BlksRead           int64
	BlksHit            int64
	TupReturned        int64
	TupFetched         int64
	TupInserted        int64
	TupUpdated         int64
	TupDeleted         int64
	TempFiles          int64
	TempBytes          int64
	Deadlocks          int64
	Conflicts          int64
	ReplicationClients int64
	WALReceivers       int64
}

func newPoolCollector(postgres *Postgres) prometheus.Collector {
	poolNamespace := "postgres_pool"
	clusterNamespace := "postgres_cluster"
	labels := []string{"role", "host", "port", "database"}
	constLabels := prometheus.Labels{
		"instance_id": postgres.instanceID,
	}
	newDesc := func(namespace string, name string, help string) *prometheus.Desc {
		return prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", name),
			help,
			labels,
			constLabels,
		)
	}

	return &poolCollector{
		postgres: postgres,
		descs: map[string]*prometheus.Desc{
			"pool_acquire_count":              newDesc(poolNamespace, "acquire_total", "Total number of successful connection acquires from the pool."),
			"pool_acquire_duration":           newDesc(poolNamespace, "acquire_duration_seconds_total", "Total time spent acquiring connections from the pool."),
			"pool_acquired_conns":             newDesc(poolNamespace, "acquired_connections", "Current number of acquired connections in the pool."),
			"pool_canceled_acquire_count":     newDesc(poolNamespace, "canceled_acquire_total", "Total number of canceled connection acquire attempts."),
			"pool_constructing_conns":         newDesc(poolNamespace, "constructing_connections", "Current number of connections being constructed."),
			"pool_empty_acquire_count":        newDesc(poolNamespace, "empty_acquire_total", "Total number of acquires that waited because the pool was empty."),
			"pool_idle_conns":                 newDesc(poolNamespace, "idle_connections", "Current number of idle connections in the pool."),
			"pool_max_conns":                  newDesc(poolNamespace, "max_connections", "Maximum number of connections configured for the pool."),
			"pool_total_conns":                newDesc(poolNamespace, "total_connections", "Current total number of connections in the pool."),
			"pool_new_conns_count":            newDesc(poolNamespace, "new_connections_total", "Total number of connections created by the pool."),
			"pool_max_lifetime_destroy_count": newDesc(poolNamespace, "max_lifetime_destroy_total", "Total number of connections destroyed because they exceeded max lifetime."),
			"pool_max_idle_destroy_count":     newDesc(poolNamespace, "max_idle_destroy_total", "Total number of connections destroyed because they exceeded max idle time."),
			"cluster_up":                      newDesc(clusterNamespace, "up", "Whether cluster metrics scrape succeeded for the postgres endpoint."),
			"cluster_scrape_duration":         newDesc(clusterNamespace, "scrape_duration_seconds", "Duration of postgres cluster metrics scrape in seconds."),
			"cluster_in_recovery":             newDesc(clusterNamespace, "in_recovery", "Whether postgres endpoint is in recovery mode."),
			"cluster_max_connections":         newDesc(clusterNamespace, "max_connections", "Configured postgres max_connections value."),
			"cluster_database_size_bytes":     newDesc(clusterNamespace, "database_size_bytes", "Current size of the selected database in bytes."),
			"cluster_numbackends":             newDesc(clusterNamespace, "numbackends", "Number of backends currently connected to the selected database."),
			"cluster_xact_commit":             newDesc(clusterNamespace, "xact_commit_total", "Total number of committed transactions for the selected database."),
			"cluster_xact_rollback":           newDesc(clusterNamespace, "xact_rollback_total", "Total number of rolled back transactions for the selected database."),
			"cluster_blks_read":               newDesc(clusterNamespace, "blks_read_total", "Total number of disk blocks read for the selected database."),
			"cluster_blks_hit":                newDesc(clusterNamespace, "blks_hit_total", "Total number of buffer hits for the selected database."),
			"cluster_tup_returned":            newDesc(clusterNamespace, "tup_returned_total", "Total number of rows returned by queries for the selected database."),
			"cluster_tup_fetched":             newDesc(clusterNamespace, "tup_fetched_total", "Total number of rows fetched by queries for the selected database."),
			"cluster_tup_inserted":            newDesc(clusterNamespace, "tup_inserted_total", "Total number of rows inserted into the selected database."),
			"cluster_tup_updated":             newDesc(clusterNamespace, "tup_updated_total", "Total number of rows updated in the selected database."),
			"cluster_tup_deleted":             newDesc(clusterNamespace, "tup_deleted_total", "Total number of rows deleted from the selected database."),
			"cluster_temp_files":              newDesc(clusterNamespace, "temp_files_total", "Total number of temporary files created by queries in the selected database."),
			"cluster_temp_bytes":              newDesc(clusterNamespace, "temp_bytes_total", "Total amount of data written to temporary files in bytes for the selected database."),
			"cluster_deadlocks":               newDesc(clusterNamespace, "deadlocks_total", "Total number of deadlocks detected in the selected database."),
			"cluster_conflicts":               newDesc(clusterNamespace, "conflicts_total", "Total number of queries canceled due to recovery conflicts in the selected database."),
			"cluster_replication_clients":     newDesc(clusterNamespace, "replication_clients", "Current number of replication clients visible from the postgres endpoint."),
			"cluster_wal_receivers":           newDesc(clusterNamespace, "wal_receivers", "Current number of WAL receivers visible from the postgres endpoint."),
		},
	}
}

func registerPoolCollector(registerer prometheus.Registerer, collector prometheus.Collector) error {
	if registerer == nil {
		return nil
	}
	if err := registerer.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			return nil
		}
		return err
	}

	return nil
}

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range c.descs {
		ch <- desc
	}
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	c.collectPool(ch, "master", c.postgres.master, c.postgres.masterCfg)
	c.collectCluster(ch, "master", c.postgres.master, c.postgres.masterCfg)

	if c.postgres.readOnly != nil && c.postgres.readOnly != c.postgres.master {
		c.collectPool(ch, "read_only", c.postgres.readOnly, c.postgres.readOnlyCfg)
		c.collectCluster(ch, "read_only", c.postgres.readOnly, c.postgres.readOnlyCfg)
	}
}

func (c *poolCollector) collectPool(ch chan<- prometheus.Metric, role string, pool *pgxpool.Pool, cfg EndpointConfig) {
	if pool == nil {
		return
	}

	stat := pool.Stat()
	labels := c.postgres.metricLabels(role, cfg)
	c.collectCounter(ch, "pool_acquire_count", float64(stat.AcquireCount()), labels...)
	c.collectCounter(ch, "pool_acquire_duration", stat.AcquireDuration().Seconds(), labels...)
	c.collectGauge(ch, "pool_acquired_conns", float64(stat.AcquiredConns()), labels...)
	c.collectCounter(ch, "pool_canceled_acquire_count", float64(stat.CanceledAcquireCount()), labels...)
	c.collectGauge(ch, "pool_constructing_conns", float64(stat.ConstructingConns()), labels...)
	c.collectCounter(ch, "pool_empty_acquire_count", float64(stat.EmptyAcquireCount()), labels...)
	c.collectGauge(ch, "pool_idle_conns", float64(stat.IdleConns()), labels...)
	c.collectGauge(ch, "pool_max_conns", float64(stat.MaxConns()), labels...)
	c.collectGauge(ch, "pool_total_conns", float64(stat.TotalConns()), labels...)
	c.collectCounter(ch, "pool_new_conns_count", float64(stat.NewConnsCount()), labels...)
	c.collectCounter(ch, "pool_max_lifetime_destroy_count", float64(stat.MaxLifetimeDestroyCount()), labels...)
	c.collectCounter(ch, "pool_max_idle_destroy_count", float64(stat.MaxIdleDestroyCount()), labels...)
}

func (c *poolCollector) collectCluster(ch chan<- prometheus.Metric, role string, pool *pgxpool.Pool, cfg EndpointConfig) {
	if pool == nil {
		return
	}

	labels := c.postgres.metricLabels(role, cfg)
	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), metricsScrapeTimeout)
	defer cancel()

	snapshot, err := c.collectClusterSnapshot(ctx, pool)
	duration := time.Since(startedAt).Seconds()
	c.collectGauge(ch, "cluster_scrape_duration", duration, labels...)
	if err != nil {
		c.collectGauge(ch, "cluster_up", 0, labels...)
		c.postgres.log.Warn(
			"Failed to scrape postgres cluster metrics",
			append(endpointFields(role, cfg), attribute.String("error", err.Error()))...,
		)
		return
	}

	c.collectGauge(ch, "cluster_up", 1, labels...)
	c.collectGauge(ch, "cluster_in_recovery", snapshot.InRecovery, labels...)
	c.collectGauge(ch, "cluster_max_connections", float64(snapshot.MaxConnections), labels...)
	c.collectGauge(ch, "cluster_database_size_bytes", float64(snapshot.DatabaseSizeBytes), labels...)
	c.collectGauge(ch, "cluster_numbackends", float64(snapshot.NumBackends), labels...)
	c.collectCounter(ch, "cluster_xact_commit", float64(snapshot.XactCommit), labels...)
	c.collectCounter(ch, "cluster_xact_rollback", float64(snapshot.XactRollback), labels...)
	c.collectCounter(ch, "cluster_blks_read", float64(snapshot.BlksRead), labels...)
	c.collectCounter(ch, "cluster_blks_hit", float64(snapshot.BlksHit), labels...)
	c.collectCounter(ch, "cluster_tup_returned", float64(snapshot.TupReturned), labels...)
	c.collectCounter(ch, "cluster_tup_fetched", float64(snapshot.TupFetched), labels...)
	c.collectCounter(ch, "cluster_tup_inserted", float64(snapshot.TupInserted), labels...)
	c.collectCounter(ch, "cluster_tup_updated", float64(snapshot.TupUpdated), labels...)
	c.collectCounter(ch, "cluster_tup_deleted", float64(snapshot.TupDeleted), labels...)
	c.collectCounter(ch, "cluster_temp_files", float64(snapshot.TempFiles), labels...)
	c.collectCounter(ch, "cluster_temp_bytes", float64(snapshot.TempBytes), labels...)
	c.collectCounter(ch, "cluster_deadlocks", float64(snapshot.Deadlocks), labels...)
	c.collectCounter(ch, "cluster_conflicts", float64(snapshot.Conflicts), labels...)
	c.collectGauge(ch, "cluster_replication_clients", float64(snapshot.ReplicationClients), labels...)
	c.collectGauge(ch, "cluster_wal_receivers", float64(snapshot.WALReceivers), labels...)
}

func (c *poolCollector) collectClusterSnapshot(ctx context.Context, pool *pgxpool.Pool) (clusterSnapshot, error) {
	var snapshot clusterSnapshot

	if err := pool.QueryRow(ctx, `
		SELECT
			CASE WHEN pg_is_in_recovery() THEN 1 ELSE 0 END,
			current_setting('max_connections')::bigint,
			pg_database_size(current_database())::bigint
	`).Scan(&snapshot.InRecovery, &snapshot.MaxConnections, &snapshot.DatabaseSizeBytes); err != nil {
		return snapshot, err
	}

	if err := pool.QueryRow(ctx, `
		SELECT
			numbackends,
			xact_commit,
			xact_rollback,
			blks_read,
			blks_hit,
			tup_returned,
			tup_fetched,
			tup_inserted,
			tup_updated,
			tup_deleted,
			temp_files,
			temp_bytes,
			deadlocks,
			conflicts
		FROM pg_stat_database
		WHERE datname = current_database()
	`).Scan(
		&snapshot.NumBackends,
		&snapshot.XactCommit,
		&snapshot.XactRollback,
		&snapshot.BlksRead,
		&snapshot.BlksHit,
		&snapshot.TupReturned,
		&snapshot.TupFetched,
		&snapshot.TupInserted,
		&snapshot.TupUpdated,
		&snapshot.TupDeleted,
		&snapshot.TempFiles,
		&snapshot.TempBytes,
		&snapshot.Deadlocks,
		&snapshot.Conflicts,
	); err != nil {
		return snapshot, err
	}

	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_stat_replication`).Scan(&snapshot.ReplicationClients); err != nil {
		return snapshot, err
	}

	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM pg_stat_wal_receiver`).Scan(&snapshot.WALReceivers); err != nil {
		return snapshot, err
	}

	return snapshot, nil
}

func (c *poolCollector) collectGauge(ch chan<- prometheus.Metric, name string, value float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(c.descs[name], prometheus.GaugeValue, value, labels...)
}

func (c *poolCollector) collectCounter(ch chan<- prometheus.Metric, name string, value float64, labels ...string) {
	ch <- prometheus.MustNewConstMetric(c.descs[name], prometheus.CounterValue, value, labels...)
}
