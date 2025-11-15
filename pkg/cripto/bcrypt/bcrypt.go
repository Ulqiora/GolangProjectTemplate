package bcrypt

import "golang.org/x/crypto/bcrypt"

type BCrypt struct {
	cfg Config
}

func New(cfg Config) *BCrypt {
	return &BCrypt{
		cfg: cfg,
	}
}

func (b *BCrypt) Hash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), b.cfg.Cost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (b *BCrypt) Validate(password string, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
