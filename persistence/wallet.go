package persistence

import (
	"errors"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
)

func NewAccountRepository(cfg conf.PersistenceConfig) (account.Repository, error) {
	switch cfg.Driver {
	case conf.PersistenceDriverBadger:
		return NewBadgerAccountRepository(cfg.Badger)

	case conf.PersistenceDriverSolana:
		return NewSolanaAccountRepository(cfg.Solana)

	case conf.PersistenceDriverComposite:
		return NewCompositeAccountRepository(cfg.Composite)

	default:
		return nil, errors.New("invalid persistence driver")
	}
}
