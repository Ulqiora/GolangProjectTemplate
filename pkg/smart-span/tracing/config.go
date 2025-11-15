package tracing

type TracerConfig struct {
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
	TLS      struct {
		Enable          bool   `yaml:"enable"`
		CertificatePath string `yaml:"certificate_path"`
		KayPath         string `yaml:"kay_path"`
	} `yaml:"tls"`
	Timeout     int64  `yaml:"timeout"`
	ServiceName string `yaml:"service_name"`
}
