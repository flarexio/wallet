package persistence

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/conf"
)

// The example config is what anyone new to the repo copies, and nothing used
// to look at it past parsing — so it could name a main store whose every
// method returns "not implemented" and still pass. This opens the drivers it
// actually names and puts an account through them.
func TestExampleConfigServesAccounts(t *testing.T) {
	assert := assert.New(t)

	// Driver paths are backfilled from conf.Path while decoding, so this has
	// to be set before the file is read or the test writes to the real store.
	conf.Path = t.TempDir()

	f, err := os.Open("../config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var cfg conf.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		t.Fatal(err)
	}

	repo, err := NewAccountRepository(cfg.Persistence)
	if !assert.NoError(err) {
		return
	}
	defer repo.Close()

	a := testAccount("alice", 1)
	if !assert.NoError(repo.Save(a)) {
		return
	}

	got, err := repo.Find("alice")
	if !assert.NoError(err) {
		return
	}

	assert.Equal(a.Salt, got.Salt)
	assert.Equal(a.KeyVersion, got.KeyVersion)
	assert.Equal(a.Derivation, got.Derivation)
	assert.Equal([]byte(a.PublicKey), []byte(got.PublicKey))

	// service.findOrCreate reads this exact error to decide whether to create a
	// wallet on first access, so a driver that reports a miss any other way
	// silently turns off account creation.
	_, err = repo.Find("nobody")
	assert.ErrorIs(err, account.ErrAccountNotFound)
}
