package config

import (
	"fmt"
	"os"

	"GolangTemplateProject/pkg/aesgcm"
	"GolangTemplateProject/pkg/email"
	"gopkg.in/yaml.v2"
)

const (
	ConfigFile = "./config/config.yaml"
)

var (
	cfg = &Config{}
)

// SERVER PARAMS

type Connection struct {
	Host string `json:"host" yaml:"host" validate:"required"`
	Port string `json:"port" yaml:"port" validate:"required"`
}

func (c *Connection) String() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

type TLS struct {
	Cert string `json:"cert" yaml:"cert" validate:"required"`
	Key  string `json:"key" yaml:"key" validate:"required"`
}

type Config struct {
	ServerInfo ServerInfo   `json:"server_info" yaml:"server_info" validate:"required"`
	Database   Database     `json:"database" yaml:"database" validate:"required"`
	Trace      Trace        `json:"trace" yaml:"trace" validate:"required"`
	Auth       Auth         `json:"auth" yaml:"auth" validate:"required"`
	Email      email.Config `json:"email" yaml:"email" validate:"required"`
}

type ServerInfo struct {
	Name           string     `json:"name" yaml:"name" validate:"required"`
	GrpcConnection Connection `json:"grpc_connection" yaml:"grpc_connection" validate:"required"`
	HttpConnection Connection `json:"http_connection" yaml:"http_connection" validate:"required"`
	TLS            TLS        `json:"tls" yaml:"tls" validate:"required"`
}

// DATABASE PARAMS

type Database struct {
	Postgres Postgres `json:"postgres" yaml:"postgres" validate:"required"`
	Minio    Minio    `json:"minio" yaml:"minio" validate:"required"`
}

type Postgres struct {
	Host     string `json:"host" yaml:"host" validate:"required"`
	Port     string `json:"port" yaml:"port" validate:"required"`
	User     string `json:"user" yaml:"user" validate:"required"`
	Password string `json:"password" yaml:"password" validate:"required"`
	Database string `json:"database" yaml:"database" validate:"required"`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode" validate:"required"`
	Settings struct {
		MaxOpenConnections int `json:"max_open_connections" yaml:"max_open_connections" validate:"required"`
		ConnMaxLifetime    int `json:"conn_max_lifetime" yaml:"conn_max_lifetime" validate:"required"`
		MaxIdleConnections int `json:"max_idle_connections" yaml:"max_idle_connections" validate:"required"`
		ConnMaxIdleTime    int `json:"conn_idle_time" yaml:"conn_idle_time" validate:"required"`
	} `json:"settings" yaml:"settings" validate:"required"`
}

type Minio struct {
	Host              string `json:"host" yaml:"host" validate:"required"`
	Port              string `json:"port" yaml:"port" validate:"required"`
	AccessKeyID       string `json:"access_key_id" yaml:"access_key_id" validate:"required"`
	SecretAccessKeyID string `json:"secret_access_key_id" yaml:"secret_access_key_id" validate:"required"`
	SSL               bool   `json:"ssl" yaml:"ssl"`
}

// TRACE

type Trace struct {
	Jaeger Jaeger `json:"jaeger" yaml:"jaeger" validate:"required"`
}

type Jaeger struct {
	Connection Connection `json:"connection" yaml:"connection" validate:"required"`
}

// Auth

type Auth struct {
	JWT    JWT           `json:"jwt" yaml:"jwt" validate:"required"`
	Bcrypt Bcrypt        `json:"bcrypt" yaml:"bcrypt" validate:"required"`
	Aesgcm aesgcm.Config `json:"aesgcm_256" yaml:"aesgcm_256" validate:"required"`
}

type JWT struct {
	SecretKey string `json:"secret_key" yaml:"secret_key" validate:"required"`
}

type Bcrypt struct {
	Secret string `json:"secret_key" yaml:"secret_key" validate:"required"`
}

func LoadConfig() error {
	file, err := os.Open(ConfigFile)
	if err != nil {
		return fmt.Errorf("config file not found: %s", err.Error())
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)
	err = yaml.NewDecoder(file).Decode(cfg)
	if err != nil {
		return fmt.Errorf("config file parse error: %s", err.Error())
	}
	return nil
}

func Get() *Config {
	return cfg
}
