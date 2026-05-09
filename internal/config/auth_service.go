package config

import (
	"time"

	kafkaproducer "GolangTemplateProject/pkg/adapters/kafka/producer"
	servergrpc "GolangTemplateProject/pkg/adapters/server_grpc"
)

type AuthService struct {
	Server         servergrpc.AppConfig `json:"server" yaml:"server"`
	Observability  Observability        `json:"observability" yaml:"observability"`
	YandexID       YandexID             `json:"yandex_id" yaml:"yandex_id"`
	KafkaProducer  kafkaproducer.Config `json:"kafka_producer" yaml:"kafka_producer"`
	VKUserConsumer KafkaConsumerConfig  `json:"vk_user_consumer" yaml:"vk_user_consumer"`
}

type KafkaConsumerConfig struct {
	Enabled       bool     `json:"enabled" yaml:"enabled"`
	Topic         string   `json:"topic" yaml:"topic"`
	Brokers       []string `json:"brokers" yaml:"brokers"`
	GroupID       string   `json:"group_id" yaml:"group_id"`
	OffsetInitial string   `json:"offset_initial" yaml:"offset_initial"`
}

type Observability struct {
	Metrics  HTTPServer `json:"metrics" yaml:"metrics"`
	Profiler HTTPServer `json:"profiler" yaml:"profiler"`
}

type HTTPServer struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Host    string `json:"host" yaml:"host"`
	Port    int    `json:"port" yaml:"port"`
}

type YandexID struct {
	ClientID     string   `json:"client_id" yaml:"client_id"`
	ClientSecret string   `json:"client_secret" yaml:"client_secret"`
	RedirectURI  string   `json:"redirect_uri" yaml:"redirect_uri"`
	Scopes       []string `json:"scopes" yaml:"scopes"`
}

func (c YandexID) RequestTimeout() time.Duration {
	return 10 * time.Second
}
