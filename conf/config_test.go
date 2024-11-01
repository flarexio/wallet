package conf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestConfig(t *testing.T) {
	assert := assert.New(t)

	Path = "~/.flarex/wallet"

	f, err := os.Open("../config.example.yaml")
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Equal("flarex-439501", cfg.Keys.Google.ProjectID)
	assert.Equal("global", cfg.Keys.Google.Location)
	assert.Equal("wallet", cfg.Keys.Google.KeyRing)
	assert.Equal("main", cfg.Keys.Google.Key)

	assert.Equal(PersistenceDriverComposite, cfg.Persistence.Driver)
	assert.NotNil(cfg.Persistence.Composite)

	composite := cfg.Persistence.Composite
	assert.Equal(PersistenceDriverBadger, composite.Cache.Driver)
	assert.Equal("wallets", composite.Cache.Badger.Name)
	assert.Equal(Path, composite.Cache.Badger.Path)
	assert.False(composite.Cache.Badger.InMem)

	assert.Equal(PersistenceDriverSolana, composite.Main.Driver)
	assert.Equal("https://api.devnet.solana.com", composite.Main.Solana.RPC)
	assert.Equal("fx72MZ7SPxwePzFiMagFZakeXxaJn7oLGDd3wxLuENL", composite.Main.Solana.Program)
	assert.Equal(Path, composite.Main.Solana.Path)
	assert.Equal("id.json", composite.Main.Solana.Account)

	assert.Equal("identity.flarex.io", cfg.Identity.BaseURL)
}
