package persistence

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/dgraph-io/badger/v4"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
)

func NewBadgerAccountRepository(cfg *conf.BadgerPersistenceConfig) (account.Repository, error) {
	opts := badger.DefaultOptions(cfg.Path + "/" + cfg.Name)
	if cfg.InMem {
		opts = badger.DefaultOptions("").WithInMemory(true)
	}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &badgerAccountRepository{db}, nil
}

type badgerAccountRepository struct {
	db *badger.DB
}

func (repo *badgerAccountRepository) Save(a *account.Account) error {
	key := []byte("sub:" + a.Subject)

	bs, err := json.Marshal(&a)
	if err != nil {
		return err
	}

	return repo.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, bs)
	})
}

func (repo *badgerAccountRepository) Find(subject string) (*account.Account, error) {
	var a *account.Account

	key := []byte("sub:" + subject)

	if err := repo.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return account.ErrAccountNotFound
			}

			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &a)
		})
	}); err != nil {
		return nil, err
	}

	return a, nil
}

func (repo *badgerAccountRepository) CacheTransaction(t *account.Transaction, ttl time.Duration) error {
	key := []byte("tx:" + t.TransactionID.String())

	bs, err := json.Marshal(&t)
	if err != nil {
		return err
	}

	return repo.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(key, bs).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func (repo *badgerAccountRepository) RemoveTransactionByID(id account.TransactionID) (*account.Transaction, error) {
	var t *account.Transaction

	key := []byte("tx:" + id.String())

	if err := repo.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return account.ErrTransactionNotFound
			}

			return err
		}

		if err := item.Value(func(val []byte) error {
			return json.Unmarshal(val, &t)
		}); err != nil {
			return err
		}

		return txn.Delete(key)
	}); err != nil {
		return nil, err
	}

	return t, nil
}

func (repo *badgerAccountRepository) Close() error {
	if repo.db != nil {
		return repo.db.Close()
	}

	return nil
}
