package wallet

import (
	"crypto/ed25519"
)

type Service interface {
	Wallet(subject string, salt string) (ed25519.PrivateKey, error)
	Transaction()
}
