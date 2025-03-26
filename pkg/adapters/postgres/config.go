package postgres

import "fmt"

type Config struct {
	Host     string `json:"host" yaml:"host" validate:"required"`
	Port     string `json:"port" yaml:"port" validate:"required"`
	User     string `json:"user" yaml:"user" validate:"required"`
	Password string `json:"password" yaml:"password" validate:"required"`
	Database string `json:"database" yaml:"database" validate:"required"`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode" validate:"required"`
	Settings struct {
		MaxOpenConnections int   `json:"max_open_connections" yaml:"max_open_connections" validate:"required"`
		ConnMaxLifetime    int64 `json:"conn_max_lifetime" yaml:"conn_max_lifetime" validate:"required"`
		MaxIdleConnections int   `json:"max_idle_connections" yaml:"max_idle_connections" validate:"required"`
		ConnMaxIdleTime    int64 `json:"conn_idle_time" yaml:"conn_idle_time" validate:"required"`
	} `json:"settings" yaml:"settings" validate:"required"`
}

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
