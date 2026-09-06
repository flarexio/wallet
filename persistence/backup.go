package persistence

import (
	"errors"
	"io"

	"github.com/dgraph-io/badger/v4"

	"github.com/flarexio/wallet/conf"
)

// ErrStoreNotEmpty guards the restore path: loading into a store that already
// holds records would merge two histories rather than restore one.
var ErrStoreNotEmpty = errors.New("refusing to restore into a store that already holds records")

const restorePendingWrites = 256

// Backup writes a snapshot of the badger store to w. The service must not be
// running against the same directory.
func Backup(cfg *conf.BadgerPersistenceConfig, w io.Writer) error {
	db, err := openBadger(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Backup(w, 0)

	return err
}

// Restore loads a snapshot into an empty badger store.
func Restore(cfg *conf.BadgerPersistenceConfig, r io.Reader) error {
	db, err := openBadger(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	empty, err := isEmpty(db)
	if err != nil {
		return err
	}

	if !empty {
		return ErrStoreNotEmpty
	}

	return db.Load(r, restorePendingWrites)
}

func openBadger(cfg *conf.BadgerPersistenceConfig) (*badger.DB, error) {
	opts := badger.DefaultOptions(cfg.Path + "/" + cfg.Name)
	if cfg.InMem {
		opts = badger.DefaultOptions("").WithInMemory(true)
	}

	return badger.Open(opts)
}

func isEmpty(db *badger.DB) (bool, error) {
	empty := true

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false

		it := txn.NewIterator(opts)
		defer it.Close()

		it.Rewind()
		empty = !it.Valid()

		return nil
	})

	return empty, err
}

// BadgerConfig finds the badger store in a persistence config, following a
// composite down to whichever side actually holds the accounts.
func BadgerConfig(cfg conf.PersistenceConfig) (*conf.BadgerPersistenceConfig, error) {
	switch cfg.Driver {
	case conf.PersistenceDriverBadger:
		if cfg.Badger == nil {
			return nil, errors.New("badger driver has no badger config")
		}

		return cfg.Badger, nil

	case conf.PersistenceDriverComposite:
		if cfg.Composite == nil {
			return nil, errors.New("composite driver has no composite config")
		}

		if c, err := BadgerConfig(cfg.Composite.Main); err == nil {
			return c, nil
		}

		return BadgerConfig(cfg.Composite.Cache)

	default:
		return nil, errors.New("no badger store in this persistence config")
	}
}
