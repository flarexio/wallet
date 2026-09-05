package persistence

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
)

func newTestRepo(t *testing.T) account.Repository {
	t.Helper()

	repo, err := NewBadgerAccountRepository(&conf.BadgerPersistenceConfig{InMem: true})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { repo.Close() })

	return repo
}

func cachedMessage(t *testing.T, subject string, id string, msg string) *account.Transaction {
	t.Helper()

	tx, err := account.NewSignMessageTransaction(subject, id, []byte(msg))
	if err != nil {
		t.Fatal(err)
	}

	return tx
}

// Transaction ids come from the client, so one account must not clobber another's.
func TestCacheTransactionIsolatesSubjects(t *testing.T) {
	assert := assert.New(t)

	repo := newTestRepo(t)
	id := uuid.NewString()

	assert.NoError(repo.CacheTransaction(cachedMessage(t, "alice", id, "alice-message"), time.Minute))
	assert.NoError(repo.CacheTransaction(cachedMessage(t, "mallory", id, "mallory-message"), time.Minute))

	got, err := repo.RemoveTransaction("alice", id2tid(t, id))
	if !assert.NoError(err) {
		return
	}

	assert.Equal("alice", got.Subject)
	assert.Equal([]byte("alice-message"), got.Message.Message)
}

// A matching id must not release another account's signature.
func TestRemoveTransactionRejectsOtherSubject(t *testing.T) {
	assert := assert.New(t)

	repo := newTestRepo(t)
	id := uuid.NewString()

	assert.NoError(repo.CacheTransaction(cachedMessage(t, "alice", id, "alice-message"), time.Minute))

	_, err := repo.RemoveTransaction("mallory", id2tid(t, id))
	assert.ErrorIs(err, account.ErrTransactionNotFound)

	got, err := repo.RemoveTransaction("alice", id2tid(t, id))
	assert.NoError(err)
	assert.Equal([]byte("alice-message"), got.Message.Message)
}

func TestRemoveTransactionRoundTrip(t *testing.T) {
	assert := assert.New(t)

	repo := newTestRepo(t)
	id := uuid.NewString()

	assert.NoError(repo.CacheTransaction(cachedMessage(t, "alice", id, "hello"), time.Minute))

	got, err := repo.RemoveTransaction("alice", id2tid(t, id))
	if !assert.NoError(err) {
		return
	}

	assert.Equal(id, got.TransactionID.String())

	_, err = repo.RemoveTransaction("alice", id2tid(t, id))
	assert.ErrorIs(err, account.ErrTransactionNotFound)
}

// transaction_id must be a plain UUID string on the wire.
func TestTransactionIDWireFormat(t *testing.T) {
	assert := assert.New(t)

	id := uuid.NewString()

	bs, err := json.Marshal(cachedMessage(t, "alice", id, "hello"))
	if !assert.NoError(err) {
		return
	}

	var raw struct {
		Subject       string `json:"subject"`
		TransactionID string `json:"transaction_id"`
	}
	if !assert.NoError(json.Unmarshal(bs, &raw)) {
		return
	}

	assert.Equal(id, raw.TransactionID, "transaction_id should be a plain UUID string")
	assert.Equal("alice", raw.Subject)

	var back *account.Transaction
	assert.NoError(json.Unmarshal(bs, &back))
	assert.Equal(id, back.TransactionID.String())
	assert.Equal("alice", back.Subject)
}

func id2tid(t *testing.T, id string) account.TransactionID {
	t.Helper()

	tid, err := account.ParseTransactionID(id)
	if err != nil {
		t.Fatal(err)
	}

	return tid
}
