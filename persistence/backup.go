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

// ErrSnapshotUnsupported is returned by a repository that has no way to stream
// itself — the solana driver, or a composite whose main store is one.
var ErrSnapshotUnsupported = errors.New("repository cannot be snapshotted")

// Snapshotter is a repository that can stream a copy of itself while it is
// still serving. This is what the scheduled backup runs against: badger's
// Backup is a Stream read, so it needs no downtime and no second handle on the
// directory — the CLI only stops the service because it opens badger itself
// and badger locks the directory exclusively.
type Snapshotter interface {
	// Snapshot writes everything newer than since, returning the version the
	// caller would pass next time to continue from here. Pass 0 for a full
	// snapshot.
	Snapshot(w io.Writer, since uint64) (uint64, error)
}

const restorePendingWrites = 256

// Backup writes a snapshot of the badger store to w. This one opens the
// directory itself, so the service must not be running against it — use a
// Snapshotter when the store is already open.
func Backup(cfg *conf.BadgerPersistenceConfig, w io.Writer) error {
	db, err := openQuietBadger(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Backup(w, 0)

	return err
}

// Restore loads a snapshot into an empty badger store.
func Restore(cfg *conf.BadgerPersistenceConfig, r io.Reader) error {
	db, err := openQuietBadger(cfg)
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
	return badger.Open(badgerOptions(cfg))
}

// openQuietBadger is for the CLI, where badger's level tables are noise around
// the one line the operator actually wants.
func openQuietBadger(cfg *conf.BadgerPersistenceConfig) (*badger.DB, error) {
	return badger.Open(badgerOptions(cfg).WithLogger(nil))
}

func badgerOptions(cfg *conf.BadgerPersistenceConfig) badger.Options {
	if cfg.InMem {
		return badger.DefaultOptions("").WithInMemory(true)
	}

	return badger.DefaultOptions(cfg.Path + "/" + cfg.Name)
}

// Accounts counts the account records in the store, so an operator can tell
// whether they are backing up the store they meant to.
func Accounts(cfg *conf.BadgerPersistenceConfig) (int, error) {
	db, err := openQuietBadger(cfg)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int

	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = []byte(subjectPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}

		return nil
	})

	return count, err
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

func (repo *badgerAccountRepository) Snapshot(w io.Writer, since uint64) (uint64, error) {
	return repo.db.Backup(w, since)
}

// Snapshot streams the main store. Accounts are written to main synchronously
// and only backfilled into the cache, so main is the copy that is guaranteed
// complete; snapshotting the cache could miss an account whose backfill had
// not landed.
func (repo *compositeAccountRepository) Snapshot(w io.Writer, since uint64) (uint64, error) {
	main, ok := repo.main.(Snapshotter)
	if !ok {
		return 0, ErrSnapshotUnsupported
	}

	return main.Snapshot(w, since)
}
