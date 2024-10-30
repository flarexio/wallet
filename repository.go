package wallet

import "errors"

var (
	ErrWalletNotFound = errors.New("wallet not found")
)

type Repository interface {
	Save(wallet *Wallet) error

	FindBySubject(subject string) (*Wallet, error)
}
