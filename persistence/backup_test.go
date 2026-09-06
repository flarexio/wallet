package persistence

import (
	"bytes"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
)

func onDiskConfig(t *testing.T) *conf.BadgerPersistenceConfig {
	t.Helper()

	return &conf.BadgerPersistenceConfig{Path: t.TempDir(), Name: "wallet"}
}

func testAccount(subject string, n byte) *account.Account {
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = n

	privkey := ed25519.NewKeyFromSeed(seed)

	return &account.Account{
		Subject:    subject,
		Salt:       subject + "-salt",
		KeyVersion: 1,
		Derivation: account.CurrentDerivation,
		PublicKey:  privkey.Public().(ed25519.PublicKey),
	}
}

func seed(t *testing.T, cfg *conf.BadgerPersistenceConfig, subjects ...string) {
	t.Helper()

	repo, err := NewBadgerAccountRepository(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	for i, subject := range subjects {
		if err := repo.Save(testAccount(subject, byte(i+1))); err != nil {
			t.Fatal(err)
		}
	}
}

// The whole point: a snapshot taken now can rebuild the store later.
func TestBackupRestoreRoundTrip(t *testing.T) {
	assert := assert.New(t)

	src := onDiskConfig(t)
	seed(t, src, "alice", "bob", "carol")

	var snapshot bytes.Buffer
	if !assert.NoError(Backup(src, &snapshot)) {
		return
	}

	assert.NotEmpty(snapshot.Bytes())

	dst := onDiskConfig(t)
	if !assert.NoError(Restore(dst, bytes.NewReader(snapshot.Bytes()))) {
		return
	}

	repo, err := NewBadgerAccountRepository(dst)
	if !assert.NoError(err) {
		return
	}
	defer repo.Close()

	for _, subject := range []string{"alice", "bob", "carol"} {
		got, err := repo.Find(subject)
		if !assert.NoError(err, subject) {
			continue
		}

		want := testAccount(subject, 0)

		assert.Equal(subject, got.Subject)
		assert.Equal(want.Salt, got.Salt)
		assert.Equal(1, got.KeyVersion)
		assert.Equal(account.CurrentDerivation, got.Derivation)
		assert.NotEmpty(got.PublicKey, "the public key has to survive, or the wallet address is lost")
	}

	_, err = repo.Find("mallory")
	assert.ErrorIs(err, account.ErrAccountNotFound)
}

// Restoring on top of live data would merge two histories. Refuse instead.
func TestRestoreRefusesNonEmptyStore(t *testing.T) {
	assert := assert.New(t)

	src := onDiskConfig(t)
	seed(t, src, "alice")

	var snapshot bytes.Buffer
	if !assert.NoError(Backup(src, &snapshot)) {
		return
	}

	dst := onDiskConfig(t)
	seed(t, dst, "bob")

	err := Restore(dst, bytes.NewReader(snapshot.Bytes()))
	assert.ErrorIs(err, ErrStoreNotEmpty)

	// bob is untouched, alice was not written.
	repo, err := NewBadgerAccountRepository(dst)
	if !assert.NoError(err) {
		return
	}
	defer repo.Close()

	_, err = repo.Find("bob")
	assert.NoError(err)

	_, err = repo.Find("alice")
	assert.ErrorIs(err, account.ErrAccountNotFound)
}

func TestBackupOfEmptyStore(t *testing.T) {
	assert := assert.New(t)

	src := onDiskConfig(t)

	var snapshot bytes.Buffer
	if !assert.NoError(Backup(src, &snapshot)) {
		return
	}

	dst := onDiskConfig(t)
	assert.NoError(Restore(dst, bytes.NewReader(snapshot.Bytes())))
}

// Pending transactions carry a TTL; a snapshot must not resurrect them as
// permanent records.
func TestBackupKeepsTransactionExpiry(t *testing.T) {
	assert := assert.New(t)

	src := onDiskConfig(t)

	repo, err := NewBadgerAccountRepository(src)
	if !assert.NoError(err) {
		return
	}

	tx, err := account.NewSignMessageTransaction("alice", "b3f1c2d4-0000-4000-8000-000000000001", []byte("hello"))
	if !assert.NoError(err) {
		repo.Close()
		return
	}

	assert.NoError(repo.CacheTransaction(tx, time.Second))
	assert.NoError(repo.Close())

	var snapshot bytes.Buffer
	if !assert.NoError(Backup(src, &snapshot)) {
		return
	}

	dst := onDiskConfig(t)
	if !assert.NoError(Restore(dst, bytes.NewReader(snapshot.Bytes()))) {
		return
	}

	restored, err := NewBadgerAccountRepository(dst)
	if !assert.NoError(err) {
		return
	}
	defer restored.Close()

	id, err := account.ParseTransactionID("b3f1c2d4-0000-4000-8000-000000000001")
	if !assert.NoError(err) {
		return
	}

	assert.Eventually(func() bool {
		_, err := restored.RemoveTransaction("alice", id)
		return err != nil
	}, 5*time.Second, 100*time.Millisecond, "a restored transaction must still expire")
}

func TestBadgerConfigResolution(t *testing.T) {
	assert := assert.New(t)

	badger := &conf.BadgerPersistenceConfig{Path: "/tmp", Name: "wallet"}

	t.Run("badger driver", func(t *testing.T) {
		got, err := BadgerConfig(conf.PersistenceConfig{
			Driver: conf.PersistenceDriverBadger,
			Badger: badger,
		})

		if assert.NoError(err) {
			assert.Equal(badger, got)
		}
	})

	t.Run("composite with badger as cache", func(t *testing.T) {
		got, err := BadgerConfig(conf.PersistenceConfig{
			Driver: conf.PersistenceDriverComposite,
			Composite: &conf.CompositePersistenceConfig{
				Main:  conf.PersistenceConfig{Driver: conf.PersistenceDriverSolana},
				Cache: conf.PersistenceConfig{Driver: conf.PersistenceDriverBadger, Badger: badger},
			},
		})

		if assert.NoError(err) {
			assert.Equal(badger, got)
		}
	})

	t.Run("no badger anywhere", func(t *testing.T) {
		_, err := BadgerConfig(conf.PersistenceConfig{Driver: conf.PersistenceDriverSolana})
		assert.Error(err)
	})
}
