package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

// Cipher seals values stored outside the account store. Its key lives on the
// server, apart from KMS, so a leaked KMS credential alone cannot read salts.
type Cipher interface {
	Seal(plaintext, aad []byte) ([]byte, error)
	Open(ciphertext, aad []byte) ([]byte, error)
}

var (
	ErrNoCipherKey     = errors.New("no cipher key configured")
	ErrShortCiphertext = errors.New("ciphertext too short")
)

func NewCipher(key [32]byte) (Cipher, error) {
	if key == [32]byte{} {
		return nil, ErrNoCipherKey
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &aesCipher{gcm}, nil
}

type aesCipher struct {
	gcm cipher.AEAD
}

func (c *aesCipher) Seal(plaintext, aad []byte) ([]byte, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return c.gcm.Seal(nonce, nonce, plaintext, aad), nil
}

func (c *aesCipher) Open(ciphertext, aad []byte) ([]byte, error) {
	n := c.gcm.NonceSize()
	if len(ciphertext) < n {
		return nil, ErrShortCiphertext
	}

	return c.gcm.Open(nil, ciphertext[:n], ciphertext[n:], aad)
}
