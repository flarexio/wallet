package wallet

import (
	"errors"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"

	"github.com/flarexio/core/model"
	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/keys"
)

type Service interface {
	Wallet(subject string) (solana.PublicKey, error)
	Close() error
}

func NewService(repo account.Repository, cfg conf.Config) (Service, error) {
	keysSvc, err := keys.NewGoogleKeysService(cfg.Keys.Google)
	if err != nil {
		return nil, err
	}

	return &service{
		accounts: repo,
		keys:     keysSvc,
	}, nil
}

type service struct {
	accounts account.Repository
	keys     keys.Service
}

func (svc *service) findOrCreate(subject string) (*account.Account, error) {
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

	a := &account.Account{
		Subject:    subject,
		Salt:       salt,
		KeyVersion: key.Version(),
		PrivateKey: sig,
		Model: model.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	return a, nil
}

func (svc *service) Wallet(subject string) (solana.PublicKey, error) {
	a, err := svc.accounts.FindBySubject(subject)
	if err != nil {
		if !errors.Is(err, account.ErrAccountNotFound) {
			return solana.PublicKey{}, err
		}

		newAccount, err := svc.findOrCreate(subject)
		if err != nil {
			return solana.PublicKey{}, err
		}

		if err := svc.accounts.Save(newAccount); err != nil {
			return solana.PublicKey{}, err
		}

		a = newAccount
	}

	return a.Wallet(), nil
}

func (svc *service) Close() error {
	return svc.keys.Close()
}
