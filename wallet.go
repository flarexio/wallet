package wallet

import (
	"crypto/ed25519"

	"github.com/gagliardetto/solana-go"

	"github.com/flarexio/core/model"
)

type Wallet struct {
	Subject    string
	Salt       string
	KeyVersion int
	PrivateKey ed25519.PrivateKey
	model.Model
}

func (w *Wallet) PublicKey() solana.PublicKey {
	pub, ok := w.PrivateKey.Public().(ed25519.PublicKey)
	if !ok {
		panic("invalid private key")
	}

	return solana.PublicKeyFromBytes(pub)
}

func (w *Wallet) Sign(data []byte) ([]byte, error) {
	return ed25519.Sign(w.PrivateKey, data), nil
}
