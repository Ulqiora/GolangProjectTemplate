package aesgcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
)

type Crypter interface {
	Encrypt(key, plaintext []byte) (ciphertext, nonce string, err error)
	Decrypt(key, nonce, ciphertext []byte) (plaintext []byte, err error)
}

type CryptAesgcm struct {
	secretKey string
}

func NewCrypt(config Config) *CryptAesgcm {
	return &CryptAesgcm{
		secretKey: config.SecretKey,
	}
}

func (c *CryptAesgcm) Encrypt(key, plaintext []byte) (ciphertext, nonce string, err error) {
	// Создание нового блока AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	// Создание нового GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	// Создание nonce (GCM требует уникальный nonce для каждой операции шифрования)
	nonce = string(make([]byte, aesGCM.NonceSize()))
	if _, err = io.ReadFull(rand.Reader, []byte(nonce)); err != nil {
		return "", "", err
	}

	// Шифрование данных
	ciphertext = string(aesGCM.Seal(nil, []byte(nonce), plaintext, nil))
	return hex.EncodeToString([]byte(ciphertext)), hex.EncodeToString([]byte(nonce)), nil
}

func (c *CryptAesgcm) Decrypt(key, nonce, ciphertext []byte) (plaintext []byte, err error) {
	// Создание нового блока AES
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Создание нового GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Расшифровка данных
	plaintext, err = aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
