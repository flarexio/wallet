package persistence

import "github.com/flarexio/wallet"

func NewCompositeWalletRepository(cache, main wallet.Repository) wallet.Repository {
	return &compositeWalletRepository{cache, main}
}

type compositeWalletRepository struct {
	cache wallet.Repository
	main  wallet.Repository
}

func (repo *compositeWalletRepository) Save(wallet *wallet.Wallet) error {
	go repo.cache.Save(wallet)

	return repo.main.Save(wallet)
}

func (repo *compositeWalletRepository) FindBySubject(subject string) (*wallet.Wallet, error) {
	if w, err := repo.cache.FindBySubject(subject); err == nil {
		return w, nil
	}

	w, err := repo.main.FindBySubject(subject)
	if err != nil {
		return nil, err
	}

	go repo.cache.Save(w)

	return w, nil
}

func (repo *compositeWalletRepository) Close() error {
	if repo.cache != nil {
		repo.cache.Close()
	}

	return repo.main.Close()
}
