package postgres

import (
	"fmt"
	"sync/atomic"
)

var postgresInstanceCounter atomic.Uint64

func nextInstanceID() string {
	return fmt.Sprintf("postgres-%d", postgresInstanceCounter.Add(1))
}

func (p *Postgres) metricLabels(role string, cfg EndpointConfig) []string {
	return []string{
		role,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	}
}
