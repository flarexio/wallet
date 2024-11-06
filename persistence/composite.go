package persistence

import (
	"errors"
	"time"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
)

func NewCompositeAccountRepository(cfg *conf.CompositePersistenceConfig) (account.Repository, error) {
	var (
		main  account.Repository
		cache account.Repository
	)

	switch cfg.Main.Driver {
	case conf.PersistenceDriverSolana:
		repo, err := NewSolanaAccountRepository(cfg.Main.Solana)
		if err != nil {
			return nil, err
		}

		main = repo

	case conf.PersistenceDriverBadger:
		repo, err := NewBadgerAccountRepository(cfg.Main.Badger)
		if err != nil {
			return nil, err
		}

		main = repo

	default:
		return nil, errors.New("invalid main driver")
	}

	switch cfg.Cache.Driver {
	case conf.PersistenceDriverBadger:
		repo, err := NewBadgerAccountRepository(cfg.Cache.Badger)
		if err != nil {
			return nil, err
		}

		cache = repo

	case conf.PersistenceDriverSolana:
		return nil, errors.New("solana is not supported as cache driver")

	default:
		return nil, errors.New("invalid cache driver")
	}

	return &compositeAccountRepository{main, cache}, nil
}

type compositeAccountRepository struct {
	main  account.Repository
	cache account.Repository
}

func (repo *compositeAccountRepository) Save(a *account.Account) error {
	err := repo.main.Save(a)
	if err != nil {
		return err
	}

	go repo.cache.Save(a)

	return nil
}

func (repo *compositeAccountRepository) Find(subject string) (*account.Account, error) {
	if a, err := repo.cache.Find(subject); err == nil {
		return a, nil
	}

	a, err := repo.main.Find(subject)
	if err != nil {
		return nil, err
	}

	go repo.cache.Save(a)

	return a, nil
}

func (repo *compositeAccountRepository) CacheTransaction(t *account.Transaction, ttl time.Duration) error {
	return repo.cache.CacheTransaction(t, ttl)
}

func (repo *compositeAccountRepository) RemoveTransactionByID(id account.TransactionID) (*account.Transaction, error) {
	return repo.cache.RemoveTransactionByID(id)
}

func (repo *compositeAccountRepository) Close() error {
	err := repo.main.Close()

	if repo.cache != nil {
		repo.cache.Close()
	}

	return err
}
