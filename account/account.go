package account

import (
	"crypto/ed25519"
	"encoding/json"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"

	"github.com/flarexio/core/model"
	"github.com/flarexio/wallet/keys"
)

func NewAccount(subject string, key keys.Key) (*Account, error) {
	salt := uuid.New().String()
	data := []byte(subject + salt)

	seed, err := key.Signature(data)
	if err != nil {
		return nil, err
	}

	privkey := ed25519.NewKeyFromSeed(seed[:ed25519.SeedSize])

	return &Account{
		Subject:    subject,
		Salt:       salt,
		KeyVersion: key.Version(),
		PrivateKey: privkey,
		Model: model.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}, nil
}

type Account struct {
	Subject    string
	Salt       string
	KeyVersion int
	PrivateKey ed25519.PrivateKey
	model.Model
}

func (a *Account) Wallet() solana.PublicKey {
	pub, ok := a.PrivateKey.Public().(ed25519.PublicKey)
	if !ok {
		panic("invalid private key")
	}

	return solana.PublicKeyFromBytes(pub)
}

func (a *Account) Signature(data []byte) []byte {
	return ed25519.Sign(a.PrivateKey, data)
}

func NewTransaction(id string, tx *solana.Transaction) (*Transaction, error) {
	tid, err := ParseTransactionID(id)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		TransactionID: tid,
		Transaction:   tx,
	}, nil
}

type TransactionID uuid.UUID

func ParseTransactionID(id string) (TransactionID, error) {
	tid, err := uuid.Parse(id)
	if err != nil {
		return TransactionID{}, err
	}

	return TransactionID(tid), nil
}

func (id TransactionID) String() string {
	return uuid.UUID(id).String()
}

type Transaction struct {
	TransactionID TransactionID
	Transaction   *solana.Transaction
}

func (tx *Transaction) UnmarshalJSON(data []byte) error {
	var raw struct {
		TransactionID string `json:"transaction_id"`
		Transaction   []byte `json:"transaction"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	id, err := ParseTransactionID(raw.TransactionID)
	if err != nil {
		return err
	}

	tx.TransactionID = id

	transaction, err := solana.TransactionFromBytes(raw.Transaction)
	if err != nil {
		return err
	}

	tx.Transaction = transaction

	return nil
}

func (tx *Transaction) MarshalJSON() ([]byte, error) {
	bs, err := tx.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}

	out := struct {
		TransactionID string `json:"transaction_id"`
		Transaction   []byte `json:"transaction"`
	}{
		TransactionID: tx.TransactionID.String(),
		Transaction:   bs,
	}

	return json.Marshal(out)
}
