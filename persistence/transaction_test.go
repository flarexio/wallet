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

// Transaction ids are supplied by the client. One account reusing another's id
// must not clobber the pending transaction that id refers to.
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

// Even with a matching id, finalizing against another account's transaction
// must fail rather than release its signature.
func TestRemoveTransactionRejectsOtherSubject(t *testing.T) {
	assert := assert.New(t)

	repo := newTestRepo(t)
	id := uuid.NewString()

	assert.NoError(repo.CacheTransaction(cachedMessage(t, "alice", id, "alice-message"), time.Minute))

	_, err := repo.RemoveTransaction("mallory", id2tid(t, id))
	assert.ErrorIs(err, account.ErrTransactionNotFound)

	// Alice's transaction survives the failed attempt.
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

	// Removal is destructive: the same transaction cannot be finalized twice.
	_, err = repo.RemoveTransaction("alice", id2tid(t, id))
	assert.ErrorIs(err, account.ErrTransactionNotFound)
}

// TransactionID used to marshal a quoted string inside a JSON string, which
// only round-tripped because uuid.Parse tolerates the stray leading quote.
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
