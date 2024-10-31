package persistence

import (
	"github.com/flarexio/wallet"
	"github.com/flarexio/wallet/conf"
)

func NewWalletRepository(cfg conf.PersistenceConfig) (wallet.Repository, error) {
	var (
		repo  wallet.Repository
		cache wallet.Repository
		main  wallet.Repository
	)

	if cfg.Cache.Enabled {
		r, err := NewBadgerWalletRepository(cfg.Cache)
		if err != nil {
			return nil, err
		}

		cache = r

		// If main is not enabled, use cache as the repository
		if !cfg.Main.Enabled {
			repo = cache
		}
	}

	if cfg.Main.Enabled {
		r, err := NewSolanaWalletRepository(cfg.Main)
		if err != nil {
			return nil, err
		}

		main = r

		// If cache is not enabled, use main as the repository
		if cache != nil {
			repo = main
		}
	}

	// If both cache and main are enabled, use the composite repository
	if cache != nil && main != nil {
		repo = NewCompositeWalletRepository(cache, main)
	}

	return repo, nil
}
