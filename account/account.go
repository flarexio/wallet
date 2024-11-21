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

func (a *Account) Sign(data []byte) []byte {
	return ed25519.Sign(a.PrivateKey, data)
}

func NewSignTransaction(id string, tx *solana.Transaction, versioned bool) (*Transaction, error) {
	tid, err := ParseTransactionID(id)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		TransactionID: tid,
		Transaction: &SignTransaction{
			Transaction: tx,
			Versioned:   versioned,
		},
	}, nil
}

func NewSignMessageTransaction(id string, msg []byte) (*Transaction, error) {
	tid, err := ParseTransactionID(id)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		TransactionID: tid,
		Message: &SignMessage{
			Message: msg,
		},
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

func (id TransactionID) MarshalJSON() ([]byte, error) {
	jsonStr := `"` + id.String() + `"`
	return json.Marshal(jsonStr)
}

func (id *TransactionID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	tid, err := ParseTransactionID(s)
	if err != nil {
		return err
	}

	*id = tid
	return nil
}

type Transaction struct {
	TransactionID TransactionID    `json:"transaction_id"`
	Transaction   *SignTransaction `json:"transaction"`
	Message       *SignMessage     `json:"message"`
}

type SignMessage struct {
	Message   []byte
	Signature solana.Signature
}

type SignTransaction struct {
	Transaction *solana.Transaction
	Versioned   bool
	Signatures  []solana.Signature
}

func (tx *SignTransaction) UnmarshalJSON(data []byte) error {
	var raw struct {
		Transaction []byte             `json:"transaction"`
		Versioned   bool               `json:"versioned"`
		Signatures  []solana.Signature `json:"signatures"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	transaction, err := solana.TransactionFromBytes(raw.Transaction)
	if err != nil {
		return err
	}

	tx.Transaction = transaction
	tx.Versioned = raw.Versioned
	tx.Signatures = raw.Signatures

	return nil
}

func (tx *SignTransaction) MarshalJSON() ([]byte, error) {
	bs, err := tx.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		Transaction []byte             `json:"transaction"`
		Versioned   bool               `json:"versioned"`
		Signatures  []solana.Signature `json:"signatures"`
	}{
		Transaction: bs,
		Versioned:   tx.Versioned,
		Signatures:  tx.Signatures,
	})
}
