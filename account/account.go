package account

import (
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"

	"github.com/flarexio/core/model"
	"github.com/flarexio/wallet/keys"
)

// CurrentDerivation is what new accounts are created with.
const CurrentDerivation = 1

// derivationInfo domain-separates the account key from anything else derived
// from the same KMS key. An entry here is permanent: every account created
// under it derives from it forever, so a new scheme gets a new number rather
// than an edit. Old entries can only be removed once no account uses them.
var derivationInfo = map[int]string{
	1: "flarex-wallet-account-v1",
}

var ErrUnsupportedDerivation = errors.New("unsupported derivation version")

func NewAccount(subject string, key keys.Key) (*Account, ed25519.PrivateKey, error) {
	salt := uuid.New().String()

	privkey, err := Derive(subject, salt, CurrentDerivation, key)
	if err != nil {
		return nil, nil, err
	}

	a := &Account{
		Subject:    subject,
		Salt:       salt,
		KeyVersion: key.Version(),
		Derivation: CurrentDerivation,
		PublicKey:  privkey.Public().(ed25519.PublicKey),
		Model: model.Model{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	return a, privkey, nil
}

// Derive rebuilds an account key from the KMS key and the account's salt.
//
// The KMS signature goes through HKDF rather than being used as the seed
// directly. The first 32 bytes of an ed25519 signature are R, a value the
// signature scheme publishes; seeding from it would make any exposure of a
// KMS signature an exposure of the account key.
func Derive(subject string, salt string, derivation int, key keys.Key) (ed25519.PrivateKey, error) {
	info, ok := derivationInfo[derivation]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedDerivation, derivation)
	}

	sig, err := key.Signature([]byte(subject + salt))
	if err != nil {
		return nil, err
	}

	seed, err := hkdf.Key(sha256.New, sig, []byte(salt), info+"|"+subject, ed25519.SeedSize)
	if err != nil {
		return nil, err
	}

	return ed25519.NewKeyFromSeed(seed), nil
}

// Account is the persisted record. It never carries private key material: the
// key is derived on demand from the KMS key plus Salt.
type Account struct {
	Subject    string
	Salt       string
	KeyVersion int
	Derivation int
	PublicKey  ed25519.PublicKey
	model.Model
}

func (a *Account) Wallet() solana.PublicKey {
	return solana.PublicKeyFromBytes(a.PublicKey)
}

func NewSignTransaction(subject string, id string, tx *solana.Transaction, versioned bool) (*Transaction, error) {
	tid, err := ParseTransactionID(id)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		Subject:       subject,
		TransactionID: tid,
		Transaction: &SignTransaction{
			Transaction: tx,
			Versioned:   versioned,
		},
	}, nil
}

func NewSignMessageTransaction(subject string, id string, msg []byte) (*Transaction, error) {
	tid, err := ParseTransactionID(id)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		Subject:       subject,
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
	return json.Marshal(id.String())
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
	Subject       string           `json:"subject"`
	TransactionID TransactionID    `json:"transaction_id"`
	Transaction   *SignTransaction `json:"transaction"`
	Message       *SignMessage     `json:"message"`
}

type SignMessage struct {
	Message []byte
}

type SignTransaction struct {
	Transaction *solana.Transaction
	Versioned   bool
}

func (tx *SignTransaction) UnmarshalJSON(data []byte) error {
	var raw struct {
		Transaction []byte `json:"transaction"`
		Versioned   bool   `json:"versioned"`
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

	return nil
}

func (tx *SignTransaction) MarshalJSON() ([]byte, error) {
	bs, err := tx.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		Transaction []byte `json:"transaction"`
		Versioned   bool   `json:"versioned"`
	}{
		Transaction: bs,
		Versioned:   tx.Versioned,
	})
}
