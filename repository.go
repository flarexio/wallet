package wallet

import "errors"

var (
	ErrWalletNotFound = errors.New("wallet not found")
)

type Repository interface {
	Save(w *Wallet) error
	FindBySubject(subject string) (*Wallet, error)
	Close() error
}
