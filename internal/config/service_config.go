package config

import (
	"fmt"
	"os"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/cripto/bcrypt"
	"GolangTemplateProject/pkg/logger"
	"GolangTemplateProject/pkg/smart-span/tracing"
	"gopkg.in/yaml.v2"
)

type ServiceConfig struct {
	Env         logger.Env           `yaml:"env"`
	ServiceName string               `yaml:"service_name"`
	Database    ServiceDatabase      `yaml:"database"`
	Tracing     tracing.TracerConfig `yaml:"tracing"`
	Auth        AuthRuntimeConfig    `yaml:"auth"`
	AuthService AuthService          `yaml:"auth_service"`
}

type ServiceDatabase struct {
	Postgres postgres.Config `yaml:"postgres"`
}

type AuthRuntimeConfig struct {
	JWT    JWTConfig     `yaml:"jwt"`
	Bcrypt bcrypt.Config `yaml:"bcrypt"`
}

type JWTConfig struct {
	SecretKey         string `yaml:"secret_key"`
	AccessTTLSeconds  int64  `yaml:"access_ttl_seconds"`
	RefreshTTLSeconds int64  `yaml:"refresh_ttl_seconds"`
}

func LoadServiceConfig(path string) (*ServiceConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config %s: %w", path, err)
	}
	defer file.Close()

	cfg := &ServiceConfig{}
	if err = yaml.NewDecoder(file).Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", path, err)
	}
	return cfg, nil
}
