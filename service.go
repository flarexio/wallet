package wallet

import (
	"errors"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/golang-jwt/jwt/v5"

	"github.com/flarexio/identity/passkeys"
	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/keys"
)

type Service interface {
	Wallet(subject string) (solana.PublicKey, error)
	Signature(subject string, data []byte) ([]byte, error)
	SignTransaction(subject string, transaction *solana.Transaction) ([]solana.Signature, error)
	InitializeTransaction(req *InitializeTransactionRequest) (*protocol.CredentialAssertion, string, error)
	FinalizeTransaction(req *protocol.ParsedCredentialAssertionData) (string, *solana.Transaction, error)
	Close() error
}

func NewService(accounts account.Repository, passkeys passkeys.Service, cfg conf.Config) (Service, error) {
	keys, err := keys.NewGoogleKeysService(cfg.Keys.Google)
	if err != nil {
		return nil, err
	}

	return &service{
		accounts: accounts,
		keys:     keys,
		passkeys: passkeys,
	}, nil
}

type service struct {
	accounts account.Repository
	keys     keys.Service
	passkeys passkeys.Service
}

func (svc *service) findOrCreate(subject string) (*account.Account, error) {
	// TODO: find wallet from solana

	key, err := svc.keys.Key()
	if err != nil {
		return nil, err
	}

	return account.NewAccount(subject, key)
}

func (svc *service) Wallet(subject string) (solana.PublicKey, error) {
	a, err := svc.accounts.Find(subject)
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

func (svc *service) Signature(subject string, data []byte) ([]byte, error) {
	a, err := svc.accounts.Find(subject)
	if err != nil {
		return nil, err
	}

	return a.Signature(data), nil
}

func (svc *service) SignTransaction(subject string, transaction *solana.Transaction) ([]solana.Signature, error) {
	a, err := svc.accounts.Find(subject)
	if err != nil {
		return nil, err
	}

	getter := func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(a.Wallet()) {
			privkey := solana.PrivateKey(a.PrivateKey)
			return &privkey
		}

		return nil
	}

	return transaction.Sign(getter)
}

func (svc *service) InitializeTransaction(req *InitializeTransactionRequest) (*protocol.CredentialAssertion, string, error) {
	data, err := req.Transaction.MarshalBinary()
	if err != nil {
		return nil, "", err
	}

	r := &passkeys.InitializeTransactionRequest{
		UserID:          req.UserID,
		TransactionID:   req.TransactionID,
		TransactionData: data,
	}

	opts, mediation, err := svc.passkeys.InitializeTransaction(r)
	if err != nil {
		return nil, "", err
	}

	if _, err := svc.SignTransaction(req.Subject, req.Transaction); err != nil {
		return nil, "", err
	}

	t, err := account.NewTransaction(req.TransactionID, req.Transaction)
	if err != nil {
		return nil, "", err
	}

	if err := svc.accounts.CacheTransaction(t, 120*time.Second); err != nil {
		return nil, "", err
	}

	return opts, mediation, nil
}

func (svc *service) FinalizeTransaction(req *protocol.ParsedCredentialAssertionData) (string, *solana.Transaction, error) {
	tokenStr, err := svc.passkeys.FinalizeTransaction(req)
	if err != nil {
		return "", nil, err
	}

	token, err := svc.passkeys.VerifyToken(tokenStr)
	if err != nil {
		return "", nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil, errors.New("invalid type")
	}

	tid, ok := claims["trans"].(string)
	if !ok {
		return "", nil, errors.New("invalid type")
	}

	id, err := account.ParseTransactionID(tid)
	if err != nil {
		return "", nil, err
	}

	t, err := svc.accounts.RemoveTransactionByID(id)
	if err != nil {
		return "", nil, err
	}

	return tokenStr, t.Transaction, nil
}

func (svc *service) Close() error {
	return svc.keys.Close()
}
