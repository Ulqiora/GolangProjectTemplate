package server_grpc

import "fmt"

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func (s ServerConfig) String() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
