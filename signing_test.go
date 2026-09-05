package wallet

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/flarexio/identity/passkeys"
	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/audit"
	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/persistence"
)

const testOrigin = "https://wallet.flarex.io"

// fakePasskeys stands in for Hanko. It records what was put in front of the
// user at Initialize so tests can compare it with what the audit log claims was
// approved, and hands back the transaction id at Finalize.
type fakePasskeys struct {
	approved      any
	transactionID string
	finalizeErr   error
}

func (p *fakePasskeys) InitializeTransaction(req *passkeys.InitializeTransactionRequest) (*protocol.CredentialAssertion, string, error) {
	p.approved = req.TransactionData
	p.transactionID = req.TransactionID

	return &protocol.CredentialAssertion{}, "optional", nil
}

func (p *fakePasskeys) FinalizeTransaction(req *protocol.ParsedCredentialAssertionData) (string, error) {
	if p.finalizeErr != nil {
		return "", p.finalizeErr
	}

	return "token", nil
}

func (p *fakePasskeys) VerifyToken(token string) (*jwt.Token, error) {
	return &jwt.Token{Claims: jwt.MapClaims{"trans": p.transactionID}}, nil
}

func (p *fakePasskeys) InitializeRegistration(userID string, username string) (*protocol.CredentialCreation, error) {
	return nil, errors.New("not used")
}

func (p *fakePasskeys) FinalizeRegistration(req *protocol.ParsedCredentialCreationData) (string, error) {
	return "", errors.New("not used")
}

func (p *fakePasskeys) InitializeLogin(userID string) (*protocol.CredentialAssertion, string, error) {
	return nil, "", errors.New("not used")
}

func (p *fakePasskeys) FinalizeLogin(req *protocol.ParsedCredentialAssertionData) (string, error) {
	return "", errors.New("not used")
}

var errAuditDown = errors.New("audit log is unavailable")

type failingAudit struct{ audit.Log }

func (failingAudit) Append(e *audit.Entry) (*audit.Entry, error) { return nil, errAuditDown }
func (failingAudit) Close() error                                { return nil }

type signingFixture struct {
	svc       *service
	passkeys  *fakePasskeys
	audit     audit.Log
	account   *account.Account
	assertion *protocol.ParsedCredentialAssertionData
}

func newSigningFixture(t *testing.T, auditLog audit.Log) *signingFixture {
	t.Helper()

	repo, err := persistence.NewBadgerAccountRepository(&conf.BadgerPersistenceConfig{InMem: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })

	seed := make([]byte, ed25519.SeedSize)
	seed[0] = 42

	a := &account.Account{
		Subject:    "alice",
		Salt:       "salt",
		KeyVersion: 3,
		PrivateKey: ed25519.NewKeyFromSeed(seed),
	}

	if err := repo.Save(a); err != nil {
		t.Fatal(err)
	}

	pk := &fakePasskeys{}

	assertion := &protocol.ParsedCredentialAssertionData{}
	assertion.ID = "credential-abc"
	assertion.Response.AuthenticatorData.Counter = 11
	assertion.Response.CollectedClientData.Origin = testOrigin

	return &signingFixture{
		svc: &service{
			accounts: repo,
			passkeys: pk,
			audit:    auditLog,
			sessions: make(map[string][]*Session),
		},
		passkeys:  pk,
		audit:     auditLog,
		account:   a,
		assertion: assertion,
	}
}

func unsignedTransaction(wallet solana.PublicKey) *solana.Transaction {
	return &solana.Transaction{
		Message: solana.Message{
			Header:          solana.MessageHeader{NumRequiredSignatures: 1},
			AccountKeys:     []solana.PublicKey{wallet},
			RecentBlockhash: solana.Hash{1, 2, 3},
		},
	}
}

// The passkey is supposed to authorize the signature, so no signature may exist
// before the assertion comes back.
func TestInitializeDoesNotSign(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, audit.NewMemoryLog())
	id := uuid.NewString()

	_, _, err := f.svc.InitializeSignMessage(&InitializeSignMessageRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Message:       []byte("hello"),
	})
	if !assert.NoError(err) {
		return
	}

	tid, err := account.ParseTransactionID(id)
	if !assert.NoError(err) {
		return
	}

	t2, err := f.svc.accounts.RemoveTransaction("alice", tid)
	if !assert.NoError(err) {
		return
	}

	assert.Equal(solana.Signature{}, t2.Message.Signature, "no signature may be parked before approval")

	entries, err := f.audit.Entries()
	assert.NoError(err)
	assert.Empty(entries, "nothing is signed yet, so nothing to audit")
}

func TestInitializeSignTransactionDoesNotSign(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, audit.NewMemoryLog())
	id := uuid.NewString()

	tx := unsignedTransaction(f.account.Wallet())

	_, _, err := f.svc.InitializeSignTransaction(&InitializeSignTransactionRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Transaction:   tx,
	})
	if !assert.NoError(err) {
		return
	}

	assert.Empty(tx.Signatures, "the transaction must not be signed at initialize")

	tid, err := account.ParseTransactionID(id)
	if !assert.NoError(err) {
		return
	}

	cached, err := f.svc.accounts.RemoveTransaction("alice", tid)
	if !assert.NoError(err) {
		return
	}

	assert.Empty(cached.Transaction.Transaction.Signatures)
	assert.Empty(cached.Transaction.Signatures)
}

func TestFinalizeSignsAfterApproval(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, audit.NewMemoryLog())
	id := uuid.NewString()
	message := []byte("transfer everything")

	_, _, err := f.svc.InitializeSignMessage(&InitializeSignMessageRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Message:       message,
	})
	if !assert.NoError(err) {
		return
	}

	sig, err := f.svc.FinalizeSignMessage("alice", f.assertion)
	if !assert.NoError(err) {
		return
	}

	pub := ed25519.PublicKey(f.account.Wallet().Bytes())
	assert.True(ed25519.Verify(pub, message, sig[:]), "signature must verify against the account key")
}

func TestFinalizeRecordsWhatWasApproved(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, audit.NewMemoryLog())
	id := uuid.NewString()
	message := []byte("transfer everything")

	_, _, err := f.svc.InitializeSignMessage(&InitializeSignMessageRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Message:       message,
	})
	if !assert.NoError(err) {
		return
	}

	if _, err := f.svc.FinalizeSignMessage("alice", f.assertion); !assert.NoError(err) {
		return
	}

	entries, err := f.audit.Entries()
	if !assert.NoError(err) {
		return
	}

	if !assert.Len(entries, 1) {
		return
	}

	e := entries[0]
	assert.Equal("alice", e.Subject)
	assert.Equal(audit.SignMessage, e.Action)
	assert.Equal(id, e.TransactionID)
	assert.Equal(f.account.Wallet().String(), e.Wallet)
	assert.Equal(3, e.KeyVersion)
	assert.Equal("credential-abc", e.CredentialID)
	assert.Equal(uint32(11), e.SignCount)
	assert.Equal(testOrigin, e.Origin)

	// The hash in the log has to be the hash the passkey was shown.
	approved, ok := f.passkeys.approved.([32]byte)
	if !assert.True(ok, "passkey was given %T", f.passkeys.approved) {
		return
	}

	assert.Equal(hex.EncodeToString(approved[:]), e.PayloadHash)
	assert.Equal(sha256.Sum256(message), approved)
	assert.NoError(f.audit.Verify())
}

// The transaction path hashes the unsigned bytes, which are exactly the bytes
// the passkey challenge was built from.
func TestFinalizeTransactionRecordsApprovedPayload(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, audit.NewMemoryLog())
	id := uuid.NewString()

	tx := unsignedTransaction(f.account.Wallet())

	_, _, err := f.svc.InitializeSignTransaction(&InitializeSignTransactionRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Transaction:   tx,
	})
	if !assert.NoError(err) {
		return
	}

	signed, versioned, err := f.svc.FinalizeSignTransaction("alice", f.assertion)
	if !assert.NoError(err) {
		return
	}

	assert.False(versioned)
	if !assert.Len(signed.Signatures, 1, "the transaction is signed at finalize") {
		return
	}

	entries, err := f.audit.Entries()
	if !assert.NoError(err) {
		return
	}

	if !assert.Len(entries, 1) {
		return
	}

	e := entries[0]
	assert.Equal(audit.SignTransaction, e.Action)

	approved, ok := f.passkeys.approved.([32]byte)
	if !assert.True(ok, "passkey was given %T", f.passkeys.approved) {
		return
	}

	assert.Equal(hex.EncodeToString(approved[:]), e.PayloadHash)
}

// Fail closed: a signature that cannot be recorded is not released.
func TestSignatureWithheldWhenAuditFails(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, failingAudit{})
	id := uuid.NewString()

	_, _, err := f.svc.InitializeSignMessage(&InitializeSignMessageRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Message:       []byte("hello"),
	})
	if !assert.NoError(err) {
		return
	}

	sig, err := f.svc.FinalizeSignMessage("alice", f.assertion)
	assert.ErrorIs(err, ErrNotAudited)
	assert.ErrorIs(err, errAuditDown, "the underlying cause is preserved")
	assert.Equal(solana.Signature{}, sig, "no signature may be returned alongside the error")
}

func TestSignTransactionWithheldWhenAuditFails(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, failingAudit{})
	id := uuid.NewString()

	_, _, err := f.svc.InitializeSignTransaction(&InitializeSignTransactionRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Transaction:   unsignedTransaction(f.account.Wallet()),
	})
	if !assert.NoError(err) {
		return
	}

	tx, _, err := f.svc.FinalizeSignTransaction("alice", f.assertion)
	assert.ErrorIs(err, ErrNotAudited)
	assert.Nil(tx, "no transaction may be returned alongside the error")
}

// A failed assertion must not reach the signing key at all.
func TestRejectedAssertionNeverSigns(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, audit.NewMemoryLog())
	id := uuid.NewString()

	_, _, err := f.svc.InitializeSignMessage(&InitializeSignMessageRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Message:       []byte("hello"),
	})
	if !assert.NoError(err) {
		return
	}

	f.passkeys.finalizeErr = errors.New("assertion rejected")

	_, err = f.svc.FinalizeSignMessage("alice", f.assertion)
	assert.Error(err)

	entries, err := f.audit.Entries()
	assert.NoError(err)
	assert.Empty(entries)

	// The parked transaction is still there, so a genuine approval can follow.
	tid, err := account.ParseTransactionID(id)
	if !assert.NoError(err) {
		return
	}

	_, err = f.svc.accounts.RemoveTransaction("alice", tid)
	assert.NoError(err)
}

// Finalizing against another account's transaction must not sign or audit.
func TestFinalizeForOtherSubjectIsNotAudited(t *testing.T) {
	assert := assert.New(t)

	f := newSigningFixture(t, audit.NewMemoryLog())
	id := uuid.NewString()

	_, _, err := f.svc.InitializeSignMessage(&InitializeSignMessageRequest{
		Subject:       "alice",
		UserID:        "alice",
		TransactionID: id,
		Message:       []byte("hello"),
	})
	if !assert.NoError(err) {
		return
	}

	_, err = f.svc.FinalizeSignMessage("mallory", f.assertion)
	assert.ErrorIs(err, account.ErrTransactionNotFound)

	entries, err := f.audit.Entries()
	assert.NoError(err)
	assert.Empty(entries)
}

// The transaction id comes from the passkey token, so a record parked by one
// endpoint can be popped by the other. Finalizing a transaction record through
// the message endpoint used to dereference a nil t.Message.
func TestFinalizeAcrossEndpointsDoesNotPanic(t *testing.T) {
	assert := assert.New(t)

	t.Run("message finalize over a transaction record", func(t *testing.T) {
		f := newSigningFixture(t, audit.NewMemoryLog())

		_, _, err := f.svc.InitializeSignTransaction(&InitializeSignTransactionRequest{
			Subject:       "alice",
			UserID:        "alice",
			TransactionID: uuid.NewString(),
			Transaction:   unsignedTransaction(f.account.Wallet()),
		})
		if !assert.NoError(err) {
			return
		}

		assert.NotPanics(func() {
			_, err := f.svc.FinalizeSignMessage("alice", f.assertion)
			assert.ErrorIs(err, account.ErrTransactionNotFound)
		})
	})

	t.Run("transaction finalize over a message record", func(t *testing.T) {
		f := newSigningFixture(t, audit.NewMemoryLog())

		_, _, err := f.svc.InitializeSignMessage(&InitializeSignMessageRequest{
			Subject:       "alice",
			UserID:        "alice",
			TransactionID: uuid.NewString(),
			Message:       []byte("hello"),
		})
		if !assert.NoError(err) {
			return
		}

		assert.NotPanics(func() {
			_, _, err := f.svc.FinalizeSignTransaction("alice", f.assertion)
			assert.ErrorIs(err, account.ErrTransactionNotFound)
		})
	})
}
