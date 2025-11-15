package bcrypt

type Hasher interface {
	Hash(password string) (string, error)
	Validate(password string, hash string) error
}

type Config struct {
	Cost int `json:"cost" yaml:"cost" validate:"required"`
}
