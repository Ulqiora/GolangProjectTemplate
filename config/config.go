package config

type Grpc struct {
	Host string `json:"host" yaml:"host" validate:"required"`
	Port string `json:"port" yaml:"port" validate:"required"`
}

type HTTP struct {
	Host string `json:"host" yaml:"host" validate:"required"`
	Port string `json:"port" yaml:"port" validate:"required"`
}
type TLS struct {
}

type Config struct {
	ServerInfo ServerInfo `json:"server_info" yaml:"server_info" validate:"required"`
}

type ServerInfo struct {
	GrpcConnection Grpc
	HttpConnection HTTP
}
