package account

import "errors"

var (
	ErrAccountNotFound = errors.New("account not found")
)

type Repository interface {
	Save(a *Account) error
	FindBySubject(subject string) (*Account, error)
	Close() error
}
