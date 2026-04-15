package postgres

import "fmt"

// Config содержит параметры подключения и настройки пула для PostgreSQL.
// Используется для построения строки подключения и передачи настроек в pgxpool.
type Config struct {
	Host     string `json:"host" yaml:"host" validate:"required,hostname|ip"`
	Port     string `json:"port" yaml:"port" validate:"required,numeric,min=1,max=65535"`
	User     string `json:"user" yaml:"user" validate:"required,min=1"`
	Password string `json:"password" yaml:"password" validate:"required,min=1"`
	Database string `json:"database" yaml:"database" validate:"required,min=1"`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode" validate:"required,oneof=disable require verify-ca verify-full"`
	Settings struct {
		MaxOpenConnections int   `json:"max_open_connections" yaml:"max_open_connections" validate:"required,min=1"`
		ConnMaxLifetime    int64 `json:"conn_max_lifetime" yaml:"conn_max_lifetime" validate:"required,min=0"`
		MaxIdleConnections int   `json:"max_idle_connections" yaml:"max_idle_connections" validate:"required,min=0"`
		ConnMaxIdleTime    int64 `json:"conn_idle_time" yaml:"conn_idle_time" validate:"required,min=0"`
	} `json:"settings" yaml:"settings" validate:"required"`
}

// ConnectionString формирует строку подключения в формате, понятном pgx/psql.
// Пример: "host=... port=... user=... password=... dbname=... sslmode=..."
func (p *Config) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host,
		p.Port,
		p.User,
		p.Password,
		p.Database,
		p.SSLMode,
	)
}
