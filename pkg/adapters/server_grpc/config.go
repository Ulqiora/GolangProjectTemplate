package server_grpc

import (
	"fmt"
	"net"
)

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

func (s ServerConfig) String() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DialAddress returns an address suitable for dialing from the same process.
// For wildcard/unspecified listen hosts (0.0.0.0, ::), it returns 127.0.0.1.
func (s ServerConfig) DialAddress() string {
	if ip := net.ParseIP(s.Host); ip != nil && ip.IsUnspecified() {
		return fmt.Sprintf("127.0.0.1:%d", s.Port)
	}
	if s.Host == "" {
		return fmt.Sprintf("127.0.0.1:%d", s.Port)
	}
	return s.String()
}

type ProxyConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
}

func (p ProxyConfig) String() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}
