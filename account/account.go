package account

import (
	"crypto/ed25519"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"

	"github.com/flarexio/core/model"
	"github.com/flarexio/wallet/keys"
)

func NewAccount(subject string, key keys.Key) (*Account, error) {
	salt := uuid.New().String()
	data := []byte(subject + salt)

	seed, err := key.Signature(data)
	if err != nil {
		return nil, err
	}

	privkey := ed25519.NewKeyFromSeed(seed[:32])

	return &Account{
		Subject:    subject,
		Salt:       salt,
		KeyVersion: key.Version(),
		PrivateKey: privkey,
		Model: model.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

type Account struct {
	Subject    string
	Salt       string
	KeyVersion int
	PrivateKey ed25519.PrivateKey
	model.Model
}

func (a *Account) Wallet() solana.PublicKey {
	pub, ok := a.PrivateKey.Public().(ed25519.PublicKey)
	if !ok {
		panic("invalid private key")
	}

	return solana.PublicKeyFromBytes(pub)
}

func (a *Account) Signature(data []byte) []byte {
	return ed25519.Sign(a.PrivateKey, data)
}
