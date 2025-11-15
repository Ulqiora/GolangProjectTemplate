package aesgcm

type Config struct {
	SecretKey string `json:"secret_key" yaml:"secret_key" validate:"required"`
}

type Crypter interface {
	Encrypt(plaintext []byte) (ciphertext, nonce string, err error)
	Decrypt(nonce, ciphertext []byte) (plaintext []byte, err error)
}
