package keys

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/kms/apiv1/kmspb"

	kms "cloud.google.com/go/kms/apiv1"

	"github.com/flarexio/wallet/conf"
)

// Cipher seals field values that have to live somewhere less trusted than the
// account store — on chain, in a shared bucket.
//
// This is deliberately a different KMS key from the signing one, so that it can
// carry its own IAM. Whoever can derive account keys still cannot read the
// salts, and whoever can read the salts still cannot derive.
type Cipher interface {
	// Seal binds the ciphertext to aad, which must be reproducible at Open and
	// must identify the record: a salt lifted from one account must not open
	// under another.
	Seal(plaintext, aad []byte) ([]byte, error)
	Open(ciphertext, aad []byte) ([]byte, error)
	Close() error
}

var ErrNoCipherKey = errors.New("no cipher key configured")

func NewGoogleCipher(cfg conf.GoogleKeyConfig) (Cipher, error) {
	if cfg.Key == "" || cfg.KeyRing == "" {
		return nil, ErrNoCipherKey
	}

	client, err := kms.NewKeyManagementClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &googleCipher{client, cfg.Path()}, nil
}

type googleCipher struct {
	client *kms.KeyManagementClient
	name   string
}

func (c *googleCipher) Seal(plaintext, aad []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("nothing to seal")
	}

	resp, err := c.client.Encrypt(context.Background(), &kmspb.EncryptRequest{
		Name:                        c.name,
		Plaintext:                   plaintext,
		AdditionalAuthenticatedData: aad,
	})
	if err != nil {
		return nil, err
	}

	return resp.Ciphertext, nil
}

func (c *googleCipher) Open(ciphertext, aad []byte) ([]byte, error) {
	resp, err := c.client.Decrypt(context.Background(), &kmspb.DecryptRequest{
		Name:                        c.name,
		Ciphertext:                  ciphertext,
		AdditionalAuthenticatedData: aad,
	})
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}

	return resp.Plaintext, nil
}

func (c *googleCipher) Close() error {
	return c.client.Close()
}
