package account

import (
	"errors"
	"time"
)

var (
	ErrAccountNotFound     = errors.New("account not found")
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrSubjectMismatch     = errors.New("transaction does not belong to subject")
)

type Repository interface {
	Save(a *Account) error
	Find(subject string) (*Account, error)

	CacheTransaction(t *Transaction, ttl time.Duration) error
	RemoveTransaction(subject string, id TransactionID) (*Transaction, error)

	Close() error
}
