package persistence

import (
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/flarexio/wallet"
	"github.com/flarexio/wallet/conf"
)

func NewSolanaWalletRepository(cfg conf.MainConfig) (wallet.Repository, error) {
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

	return &solanaWalletRepository{client, account, program}, nil
}

type solanaWalletRepository struct {
	client  *rpc.Client
	account solana.PrivateKey
	program solana.PublicKey
}

func (repo *solanaWalletRepository) Save(wallet *wallet.Wallet) error {
	return errors.New("not implemented")
}

func (repo *solanaWalletRepository) FindBySubject(subject string) (*wallet.Wallet, error) {
	return nil, errors.New("not implemented")
}

func (repo *solanaWalletRepository) Close() error {
	return repo.client.Close()
}
