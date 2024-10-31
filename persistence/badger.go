package persistence

import (
	"encoding/json"
	"errors"

	"github.com/dgraph-io/badger/v4"

	"github.com/flarexio/wallet"
	"github.com/flarexio/wallet/conf"
)

func NewBadgerWalletRepository(cfg conf.CacheConfig) (wallet.Repository, error) {
	opts := badger.DefaultOptions(cfg.Path + "/" + cfg.Name)
	if cfg.InMem {
		opts = badger.DefaultOptions("").WithInMemory(true)
	}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &badgerWalletRepository{db}, nil
}

type badgerWalletRepository struct {
	db *badger.DB
}

func (repo *badgerWalletRepository) Save(w *wallet.Wallet) error {
	key := []byte("sub:" + w.Subject)

	bs, err := json.Marshal(w)
	if err != nil {
		return err
	}

	return repo.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, bs)
	})
}

func (repo *badgerWalletRepository) FindBySubject(subject string) (*wallet.Wallet, error) {
	var w *wallet.Wallet

	key := []byte("sub:" + subject)

	if err := repo.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return wallet.ErrWalletNotFound
			}

			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &w)
		})
	}); err != nil {
		return nil, err
	}

	return w, nil
}

func (repo *badgerWalletRepository) Close() error {
	if repo.db != nil {
		return repo.db.Close()
	}

	return nil
}
