package email

type SMTPConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
}

type Config struct {
	SMTPConfig
	Email    string `json:"email" yaml:"email"`
	Password string `json:"password" yaml:"password"`
}
