package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pgx *pgxpool.Pool
}

type IPostgres interface {
	Connection(ctx context.Context) (*Connection, error)
	Ping(ctx context.Context, pool *Postgres) error
	Close() error
}

func New(ctx context.Context, cfg *Config) (*Postgres, error) {
	connectionStr := cfg.ConnectionString()
	pool, err := pgxpool.New(ctx, connectionStr)
	if err != nil {
		return nil, err
	}
	pool.Config().MaxConnIdleTime = time.Duration(cfg.Settings.ConnMaxIdleTime * time.Second.Milliseconds())
	pool.Config().MaxConnLifetime = time.Duration(cfg.Settings.ConnMaxLifetime * time.Second.Milliseconds())
	pool.Config().MaxConns = int32(cfg.Settings.MaxOpenConnections)
	pool.Config().MaxConnIdleTime = time.Duration(cfg.Settings.ConnMaxIdleTime)

	return &Postgres{pgx: pool}, nil
}

func (p *Postgres) Close() error {
	p.pgx.Close()
	return nil
}

func (p *Postgres) Ping(ctx context.Context, pool *Postgres) error {
	return pool.pgx.Ping(ctx)
}

func (p *Postgres) Connection(ctx context.Context) (*Connection, error) {
	conn, err := p.pgx.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &Connection{connection: conn}, nil
}
