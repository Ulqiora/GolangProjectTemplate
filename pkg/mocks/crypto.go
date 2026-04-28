package mocks

type Crypter struct {
	EncryptFunc func(plaintext []byte) (ciphertext, nonce string, err error)
	DecryptFunc func(nonce, ciphertext []byte) (plaintext []byte, err error)

	EncryptCalls int
	DecryptCalls int
}

func (m *Crypter) Encrypt(plaintext []byte) (string, string, error) {
	m.EncryptCalls++
	if m.EncryptFunc != nil {
		return m.EncryptFunc(plaintext)
	}
	return "", "", nil
}

func (m *Crypter) Decrypt(nonce, ciphertext []byte) ([]byte, error) {
	m.DecryptCalls++
	if m.DecryptFunc != nil {
		return m.DecryptFunc(nonce, ciphertext)
	}
	return nil, nil
}

type Hasher struct {
	HashFunc     func(password string) (string, error)
	ValidateFunc func(password string, hash string) error

	HashCalls     int
	ValidateCalls int
}

func (m *Hasher) Hash(password string) (string, error) {
	m.HashCalls++
	if m.HashFunc != nil {
		return m.HashFunc(password)
	}
	return "", nil
}

func (m *Hasher) Validate(password string, hash string) error {
	m.ValidateCalls++
	if m.ValidateFunc != nil {
		return m.ValidateFunc(password, hash)
	}
	return nil
}
