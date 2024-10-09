package aesgcm

type Config struct {
	SecretKey string `json:"secret_key" yaml:"secret_key" validate:"required"`
}
