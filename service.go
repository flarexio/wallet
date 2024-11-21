package wallet

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mr-tron/base58"

	"github.com/flarexio/identity/passkeys"
	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/keys"
)

type Service interface {
	Wallet(subject string) (solana.PublicKey, error)

	SignMessage(subject string, message []byte) (solana.Signature, error)
	InitializeSignMessage(req *InitializeSignMessageRequest) (*protocol.CredentialAssertion, string, error)
	FinalizeSignMessage(req *protocol.ParsedCredentialAssertionData) (solana.Signature, error)

	SignTransaction(subject string, transaction *solana.Transaction) ([]solana.Signature, error)
	InitializeSignTransaction(req *InitializeSignTransactionRequest) (*protocol.CredentialAssertion, string, error)
	FinalizeSignTransaction(req *protocol.ParsedCredentialAssertionData) (*solana.Transaction, bool, error)

	CreateSession(ctx context.Context, data []byte) (string, <-chan []byte, error)
	SessionData(ctx context.Context, session string) ([]byte, error)
	AckSession(ctx context.Context, session string, data []byte) error

	Close() error
}

func NewService(accounts account.Repository, passkeys passkeys.Service, cfg conf.Config) (Service, error) {
	keys, err := keys.NewGoogleKeysService(cfg.Keys.Google)
	if err != nil {
		return nil, err
	}

	sessionKey := cfg.Keys.Session.Key
	privkey := ed25519.NewKeyFromSeed(sessionKey[:])

	return &service{
		accounts: accounts,
		keys:     keys,
		passkeys: passkeys,
		privkey:  privkey,
		sessions: make(map[string][]*Session),
	}, nil
}

type service struct {
	accounts account.Repository
	keys     keys.Service
	passkeys passkeys.Service
	privkey  ed25519.PrivateKey
	sessions map[string][]*Session
	sync.Mutex
}

type Session struct {
	data   []byte
	sig    string
	ch     chan<- []byte
	cancel context.CancelFunc
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

func (svc *service) SignMessage(subject string, message []byte) (solana.Signature, error) {
	a, err := svc.accounts.Find(subject)
	if err != nil {
		return solana.Signature{}, err
	}

	privkey := solana.PrivateKey(a.PrivateKey)

	return privkey.Sign(message)
}

func (svc *service) InitializeSignMessage(req *InitializeSignMessageRequest) (*protocol.CredentialAssertion, string, error) {
	r := &passkeys.InitializeTransactionRequest{
		UserID:          req.UserID,
		TransactionID:   req.TransactionID,
		TransactionData: sha256.Sum256(req.Message),
	}

	opts, mediation, err := svc.passkeys.InitializeTransaction(r)
	if err != nil {
		return nil, "", err
	}

	sig, err := svc.SignMessage(req.Subject, req.Message)
	if err != nil {
		return nil, "", err
	}

	t, err := account.NewSignMessageTransaction(req.TransactionID, req.Message)
	if err != nil {
		return nil, "", err
	}

	t.Message.Signature = sig

	if err := svc.accounts.CacheTransaction(t, 120*time.Second); err != nil {
		return nil, "", err
	}

	return opts, mediation, nil
}

func (svc *service) FinalizeSignMessage(req *protocol.ParsedCredentialAssertionData) (solana.Signature, error) {
	var sig solana.Signature

	tokenStr, err := svc.passkeys.FinalizeTransaction(req)
	if err != nil {
		return sig, err
	}

	token, err := svc.passkeys.VerifyToken(tokenStr)
	if err != nil {
		return sig, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return sig, errors.New("invalid type")
	}

	tid, ok := claims["trans"].(string)
	if !ok {
		return sig, errors.New("invalid type")
	}

	id, err := account.ParseTransactionID(tid)
	if err != nil {
		return sig, err
	}

	t, err := svc.accounts.RemoveTransactionByID(id)
	if err != nil {
		return sig, err
	}

	sig = t.Message.Signature

	return sig, nil
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

func (svc *service) InitializeSignTransaction(req *InitializeSignTransactionRequest) (*protocol.CredentialAssertion, string, error) {
	data, err := req.Transaction.MarshalBinary()
	if err != nil {
		return nil, "", err
	}

	r := &passkeys.InitializeTransactionRequest{
		UserID:          req.UserID,
		TransactionID:   req.TransactionID,
		TransactionData: sha256.Sum256(data),
	}

	opts, mediation, err := svc.passkeys.InitializeTransaction(r)
	if err != nil {
		return nil, "", err
	}

	sigs, err := svc.SignTransaction(req.Subject, req.Transaction)
	if err != nil {
		return nil, "", err
	}

	t, err := account.NewSignTransaction(req.TransactionID, req.Transaction, req.Versioned)
	if err != nil {
		return nil, "", err
	}

	t.Transaction.Signatures = sigs

	if err := svc.accounts.CacheTransaction(t, 120*time.Second); err != nil {
		return nil, "", err
	}

	return opts, mediation, nil
}

func (svc *service) FinalizeSignTransaction(req *protocol.ParsedCredentialAssertionData) (*solana.Transaction, bool, error) {
	tokenStr, err := svc.passkeys.FinalizeTransaction(req)
	if err != nil {
		return nil, false, err
	}

	token, err := svc.passkeys.VerifyToken(tokenStr)
	if err != nil {
		return nil, false, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false, errors.New("invalid type")
	}

	tid, ok := claims["trans"].(string)
	if !ok {
		return nil, false, errors.New("invalid type")
	}

	id, err := account.ParseTransactionID(tid)
	if err != nil {
		return nil, false, err
	}

	t, err := svc.accounts.RemoveTransactionByID(id)
	if err != nil {
		return nil, false, err
	}

	tx := t.Transaction.Transaction
	versioned := t.Transaction.Versioned

	return tx, versioned, nil
}

func (svc *service) CreateSession(ctx context.Context, data []byte) (string, <-chan []byte, error) {
	sig := ed25519.Sign(svc.privkey, data)
	basedSig := base58.Encode(sig)

	ch := make(chan []byte)
	ctx, cancel := context.WithCancel(ctx)

	session := &Session{
		data:   data,
		sig:    basedSig,
		ch:     ch,
		cancel: cancel,
	}

	go svc.sessionTimeout(ctx, session)

	svc.Lock()
	defer svc.Unlock()

	index := basedSig[:2]

	sessions, ok := svc.sessions[index]
	if !ok {
		sessions = make([]*Session, 0)
	}

	for _, s := range sessions {
		if s.sig == session.sig {
			return "", nil, errors.New("session already exists")
		}
	}

	svc.sessions[index] = append(sessions, session)

	return basedSig, ch, nil
}

func (svc *service) sessionTimeout(ctx context.Context, session *Session) {
	for {
		select {
		case <-ctx.Done():
			close(session.ch)

			svc.Lock()

			index := session.sig[:2]
			sessions, ok := svc.sessions[index]
			if ok {
				for i, s := range sessions {
					if s == session {
						svc.sessions[index] = append(sessions[:i], sessions[i+1:]...)
						break
					}
				}

				if len(svc.sessions[index]) == 0 {
					delete(svc.sessions, index)
				}
			}

			svc.Unlock()
			return

		case <-time.After(120 * time.Second):
			session.ch <- nil
			session.cancel()
		}
	}
}

func (svc *service) SessionData(ctx context.Context, session string) ([]byte, error) {
	svc.Lock()
	defer svc.Unlock()

	index := session[:2]
	sessions, ok := svc.sessions[index]
	if !ok {
		return nil, errors.New("session not found")
	}

	for _, s := range sessions {
		if s.sig == session {
			return s.data, nil
		}
	}

	return nil, errors.New("session not found")
}

func (svc *service) AckSession(ctx context.Context, session string, data []byte) error {
	svc.Lock()
	defer svc.Unlock()

	index := session[:2]
	sessions, ok := svc.sessions[index]
	if !ok {
		return errors.New("session not found")
	}

	for _, s := range sessions {
		if s.sig == session {
			s.ch <- data
			defer s.cancel()

			return nil
		}
	}

	return errors.New("session not found")
}

func (svc *service) Close() error {
	return svc.keys.Close()
}
