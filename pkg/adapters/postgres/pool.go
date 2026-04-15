// Package postgres предоставляет адаптер для подключения к PostgreSQL через pgxpool.
// Документация описывает основные объекты: конфигурацию, пул соединений и интерфейсы
// для получения/использования соединений.
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres представляет пул подключений к PostgreSQL.
// Поле pgx содержит внутренний *pgxpool.Pool и не экспортируется напрямую.
type Postgres struct {
	pgx *pgxpool.Pool
}

// IPostgres описывает поведение адаптера postgres, которое требуется использовать
// в приложении: получение соединения, проверка доступности и корректное закрытие.
type IPostgres interface {
	// Connection возвращает обёртку Connection, которую нужно освободить после использования.
	Connection(ctx context.Context) (Connection, error)
	// Ping проверяет доступность базы данных.
	Ping(ctx context.Context, pool *Postgres) error
	// Close закрывает пул соединений.
	Close() error
}

// New создаёт и настраивает новый пул соединений к PostgreSQL на основе Config.
// Возвращает готовый *Postgres или ошибку при создании пула.
func New(ctx context.Context, cfg *Config) (*Postgres, error) {
	connectionStr := cfg.ConnectionString()

	// Парсим конфиг, чтобы корректно задать параметры пула перед созданием
	poolCfg, err := pgxpool.ParseConfig(connectionStr)
	if err != nil {
		return nil, err
	}

	// Применяем настройки из cfg.Settings (с проверкой на значения > 0)
	if cfg.Settings.MaxOpenConnections > 0 {
		poolCfg.MaxConns = int32(cfg.Settings.MaxOpenConnections)
	}
	if cfg.Settings.MaxIdleConnections > 0 {
		// В pgxpool есть MinConns — используем его как минимальное число подключений (аналог "idle")
		poolCfg.MinConns = int32(cfg.Settings.MaxIdleConnections)
	}
	if cfg.Settings.ConnMaxLifetime > 0 {
		poolCfg.MaxConnLifetime = time.Duration(cfg.Settings.ConnMaxLifetime) * time.Second
	}
	if cfg.Settings.ConnMaxIdleTime > 0 {
		poolCfg.MaxConnIdleTime = time.Duration(cfg.Settings.ConnMaxIdleTime) * time.Second
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	return &Postgres{pgx: pool}, nil
}

// Close закрывает пул соединений. Метод безопасен для повторного вызова.
func (p *Postgres) Close() error {
	p.pgx.Close()
	return nil
}

// Ping выполняет простой ping к базе, используя внутренний пул.
func (p *Postgres) Ping(ctx context.Context, pool *Postgres) error {
	return pool.pgx.Ping(ctx)
}

// Connection аккумулирует и возвращает подключение из пула в виде Connection.
// Получатель обязан вызвать Release/Close после использования.
func (p *Postgres) Connection(ctx context.Context) (Connection, error) {
	conn, err := p.pgx.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &ConnectionImpl{connection: conn}, nil
}
