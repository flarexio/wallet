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

	assert.True(cfg.Persistences.Cache.Enabled)
	assert.Equal("wallets", cfg.Persistences.Cache.Name)
	assert.Equal(Path, cfg.Persistences.Cache.Path)
	assert.False(cfg.Persistences.Cache.InMem)

	assert.False(cfg.Persistences.Main.Enabled)
	assert.Equal("https://api.devnet.solana.com", cfg.Persistences.Main.RPC)
	assert.Equal("fx72MZ7SPxwePzFiMagFZakeXxaJn7oLGDd3wxLuENL", cfg.Persistences.Main.Program)
	assert.Equal(Path, cfg.Persistences.Main.Path)
	assert.Equal("id.json", cfg.Persistences.Main.Account)
}
