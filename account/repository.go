package account

import (
	"errors"
	"time"
)

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrTransactionNotFound = errors.New("transaction not found")
)

type Repository interface {
	Save(a *Account) error
	Find(subject string) (*Account, error)

	CacheTransaction(t *Transaction, ttl time.Duration) error
	RemoveTransactionByID(id TransactionID) (*Transaction, error)

	Close() error
}
