// Package postgres предоставляет адаптер для подключения к PostgreSQL через pgxpool.
// Документация описывает основные объекты: конфигурацию, пул соединений и интерфейсы
// для получения/использования соединений.
package postgres

import (
	"context"
	"time"

	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/logger/attribute"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// Postgres представляет пул подключений к PostgreSQL.
// Поля master и readOnly содержат внутренние *pgxpool.Pool и не экспортируются напрямую.
type Postgres struct {
	instanceID       string
	log              logger.Logger
	master           *pgxpool.Pool
	readOnly         *pgxpool.Pool
	masterCfg        EndpointConfig
	readOnlyCfg      EndpointConfig
	readOnlyIsMaster bool
	metrics          prometheus.Collector
}

// IPostgres описывает поведение адаптера postgres, которое требуется использовать
// в приложении: получение соединения, проверка доступности и корректное закрытие.
type IPostgres interface {
	// Connection возвращает обёртку Connection, которую нужно освободить после использования.
	Connection(ctx context.Context) (Connection, error)
	// MasterConnection возвращает подключение к master-пулу для write/transaction операций.
	MasterConnection(ctx context.Context) (Connection, error)
	// ReadOnlyConnection возвращает подключение к RO-пулу для read-only операций.
	ReadOnlyConnection(ctx context.Context) (Connection, error)
	// Ping проверяет доступность базы данных.
	Ping(ctx context.Context) error
	// Close закрывает пул соединений.
	Close() error
}

// New создаёт и настраивает новый пул соединений к PostgreSQL на основе Config.
// Возвращает готовый *Postgres или ошибку при создании пула.
func New(ctx context.Context, cfg *Config) (*Postgres, error) {
	return NewWithOptions(ctx, cfg)
}

func NewWithOptions(ctx context.Context, cfg *Config, opts ...Option) (*Postgres, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	resolvedOpts := buildOptions(opts...)
	instanceID := nextInstanceID()
	log := resolvedOpts.log.WithN("postgres",
		attribute.String("instance_id", instanceID),
	)

	log.Info("Initializing postgres adapter")

	masterPool, err := newPool(ctx, "master", cfg.Master, log, resolvedOpts)
	if err != nil {
		log.Error("Failed to initialize postgres master pool", attribute.String("error", err.Error()))
		return nil, err
	}

	readOnlyCfg, readOnlyConfigured := cfg.readOnlyEndpoint()
	readOnlyPool := masterPool
	if readOnlyConfigured {
		readOnlyPool, err = newPool(ctx, "read_only", readOnlyCfg, log, resolvedOpts)
		if err != nil {
			masterPool.Close()
			log.Error("Failed to initialize postgres read-only pool", attribute.String("error", err.Error()))
			return nil, err
		}
		log.Info("Initialized dedicated postgres read-only pool", endpointFields("read_only", readOnlyCfg)...)
	} else {
		log.Info("Read-only postgres pool is not configured, master pool will be reused", endpointFields("master", cfg.Master)...)
	}

	p := &Postgres{
		instanceID:       instanceID,
		log:              log,
		master:           masterPool,
		readOnly:         readOnlyPool,
		masterCfg:        cfg.Master,
		readOnlyCfg:      readOnlyCfg,
		readOnlyIsMaster: !readOnlyConfigured,
	}
	p.metrics = newPoolCollector(p)

	if err = registerPoolCollector(resolvedOpts.registerer, p.metrics); err != nil {
		log.Error("Failed to register postgres metrics collector", attribute.String("error", err.Error()))
		p.Close()
		return nil, err
	}

	log.Info("Postgres adapter initialized")

	return p, nil
}

func (p *Postgres) MetricsCollector() prometheus.Collector {
	return p.metrics
}

func (p *Postgres) RegisterMetrics(registerer prometheus.Registerer) error {
	return registerPoolCollector(registerer, p.metrics)
}

func newPool(ctx context.Context, role string, cfg EndpointConfig, log logger.Logger, opts options) (*pgxpool.Pool, error) {
	log.Debug("Creating postgres pool", endpointFields(role, cfg)...)

	poolCfg, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, err
	}

	applyPoolSettings(poolCfg, cfg.Settings)
	if opts.tracingEnabled {
		poolCfg.ConnConfig.Tracer = newPGXTracer(opts.tracerProvider, role, cfg)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	log.Info("Created postgres pool", endpointFields(role, cfg)...)

	return pool, nil
}

func applyPoolSettings(poolCfg *pgxpool.Config, settings PoolSettings) {
	if settings.MaxOpenConnections > 0 {
		poolCfg.MaxConns = int32(settings.MaxOpenConnections)
	}
	if settings.MaxIdleConnections > 0 {
		// В pgxpool есть MinConns — используем его как минимальное число подключений (аналог "idle").
		poolCfg.MinConns = int32(settings.MaxIdleConnections)
	}
	if settings.ConnMaxLifetime > 0 {
		poolCfg.MaxConnLifetime = time.Duration(settings.ConnMaxLifetime) * time.Second
	}
	if settings.ConnMaxIdleTime > 0 {
		poolCfg.MaxConnIdleTime = time.Duration(settings.ConnMaxIdleTime) * time.Second
	}
}

// Close закрывает пул соединений. Метод безопасен для повторного вызова.
func (p *Postgres) Close() error {
	p.log.Info("Closing postgres adapter")
	if p.readOnly != nil && p.readOnly != p.master {
		p.log.Debug("Closing postgres read-only pool", endpointFields("read_only", p.readOnlyCfg)...)
		p.readOnly.Close()
	}
	if p.master != nil {
		p.log.Debug("Closing postgres master pool", endpointFields("master", p.masterCfg)...)
		p.master.Close()
	}
	p.log.Info("Closed postgres adapter")
	return nil
}

// Ping выполняет простой ping к master и RO-пулам.
func (p *Postgres) Ping(ctx context.Context) error {
	p.log.Debug("Pinging postgres master pool", endpointFields("master", p.masterCfg)...)
	if err := p.master.Ping(ctx); err != nil {
		p.log.Error("Failed to ping postgres master pool", append(endpointFields("master", p.masterCfg), attribute.String("error", err.Error()))...)
		return err
	}
	if p.readOnly != nil && p.readOnly != p.master {
		p.log.Debug("Pinging postgres read-only pool", endpointFields("read_only", p.readOnlyCfg)...)
		if err := p.readOnly.Ping(ctx); err != nil {
			p.log.Error("Failed to ping postgres read-only pool", append(endpointFields("read_only", p.readOnlyCfg), attribute.String("error", err.Error()))...)
			return err
		}
	}

	p.log.Debug("Successfully pinged postgres pools")
	return nil
}

// Connection аккумулирует и возвращает подключение из пула в виде Connection.
// Получатель обязан вызвать Release/Close после использования.
func (p *Postgres) Connection(ctx context.Context) (Connection, error) {
	return p.MasterConnection(ctx)
}

func (p *Postgres) MasterConnection(ctx context.Context) (Connection, error) {
	return p.acquireConnection(ctx, "master", p.master, p.masterCfg)
}

func (p *Postgres) ReadOnlyConnection(ctx context.Context) (Connection, error) {
	role := "read_only"
	cfg := p.readOnlyCfg
	if p.readOnlyIsMaster {
		role = "master"
		cfg = p.masterCfg
	}

	return p.acquireConnection(ctx, role, p.readOnly, cfg)
}

func (p *Postgres) acquireConnection(ctx context.Context, role string, pool *pgxpool.Pool, cfg EndpointConfig) (Connection, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		p.log.Error("Failed to acquire postgres connection", append(endpointFields(role, cfg), attribute.String("error", err.Error()))...)
		return nil, err
	}

	return &ConnectionImpl{connection: conn}, nil
}

func endpointFields(role string, cfg EndpointConfig) []attribute.Field {
	return []attribute.Field{
		attribute.String("role", role),
		attribute.String("host", cfg.Host),
		attribute.String("port", cfg.Port),
		attribute.String("database", cfg.Database),
		attribute.String("user", cfg.User),
	}
}
