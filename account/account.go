package account

import (
	"crypto/ed25519"

	"github.com/gagliardetto/solana-go"

	"github.com/flarexio/core/model"
)

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
