package wallet

import (
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"

	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/keys"
)

type Service interface {
	Wallet(subject string) (solana.PublicKey, error)
}

func NewService(wallets Repository, cfg conf.Config) (Service, error) {
	keysSvc, err := keys.NewGoogleKeysService(cfg.Keys.Google)
	if err != nil {
		return nil, err
	}

	return &service{
		wallets: wallets,
		keys:    keysSvc,
	}, nil
}

type service struct {
	wallets Repository
	keys    keys.Service
}

func (svc *service) findOrCreate(subject string) (*Wallet, error) {
	// TODO: find wallet from solana

	key, err := svc.keys.Key()
	if err != nil {
		return nil, err
	}

	salt := uuid.New().String()
	data := []byte(subject + salt)

	sig, err := key.Signature(data)
	if err != nil {
		return nil, err
	}

	w := &Wallet{
		Subject:    subject,
		Salt:       salt,
		KeyVersion: key.Version(),
		PrivateKey: sig,
	}

	return w, nil
}

func (svc *service) Wallet(subject string) (solana.PublicKey, error) {
	wallet, err := svc.wallets.FindBySubject(subject)
	if err != nil {
		if !errors.Is(err, ErrWalletNotFound) {
			return solana.PublicKey{}, err
		}

		w, err := svc.findOrCreate(subject)
		if err != nil {
			return solana.PublicKey{}, err
		}

		if err := svc.wallets.Save(w); err != nil {
			return solana.PublicKey{}, err
		}

		wallet = w
	}

	return wallet.PublicKey(), nil
}
