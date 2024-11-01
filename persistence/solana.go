package persistence

import (
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
)

func NewSolanaAccountRepository(cfg *conf.SolanaPersistenceConfig) (account.Repository, error) {
	client := rpc.New(cfg.RPC)

	path := cfg.Path + "/" + cfg.Account

	account, err := solana.PrivateKeyFromSolanaKeygenFile(path)
	if err != nil {
		return nil, err
	}

	program, err := solana.PublicKeyFromBase58(cfg.Program)
	if err != nil {
		return nil, err
	}

	return &solanaAccountRepository{client, account, program}, nil
}

type solanaAccountRepository struct {
	client  *rpc.Client
	account solana.PrivateKey
	program solana.PublicKey
}

func (repo *solanaAccountRepository) Save(a *account.Account) error {
	return errors.New("not implemented")
}

func (repo *solanaAccountRepository) FindBySubject(subject string) (*account.Account, error) {
	return nil, errors.New("not implemented")
}

func (repo *solanaAccountRepository) Close() error {
	if repo.client != nil {
		return repo.client.Close()
	}

	return nil
}
