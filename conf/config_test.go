package conf

import (
	"os"
	"testing"
	"time"

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

	assert.Len(cfg.Keys.Session.Key, 32)

	assert.Equal(PersistenceDriverBadger, cfg.Persistence.Driver)
	assert.NotNil(cfg.Persistence.Badger)

	badger := cfg.Persistence.Badger
	assert.Equal("wallets", badger.Name)
	assert.Equal(Path, badger.Path)
	assert.False(badger.InMem)

	assert.Equal("identity.flarex.io", cfg.JWT.Issuer)
	assert.Equal("talkix.flarex.io", cfg.JWT.Audience)
	assert.Equal("https://identity.flarex.io/.well-known/jwks.json", cfg.JWT.JWKsURL)
}

func TestCompositeConfig(t *testing.T) {
	assert := assert.New(t)

	Path = "~/.flarex/wallet"

	f, err := os.Open("testdata/composite.yaml")
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
}

func TestBackupConfig(t *testing.T) {
	assert := assert.New(t)

	f, err := os.Open("testdata/backup.yaml")
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

	assert.True(cfg.Backup.Enabled())
	assert.Equal(6*time.Hour, cfg.Backup.Interval)

	// Omitting keep must not mean "keep none".
	assert.Equal(DefaultBackupKeep, cfg.Backup.Keep)

	assert.Equal(BackupDestinationGCS, cfg.Backup.Destination.Driver)
	assert.NotNil(cfg.Backup.Destination.GCS)
	assert.Equal("flarex-wallet-backups", cfg.Backup.Destination.GCS.Bucket)
	assert.Equal("prod", cfg.Backup.Destination.GCS.Prefix)
}

// No backup section at all leaves the schedule off rather than half-configured.
func TestBackupOffByDefault(t *testing.T) {
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

	assert.False(cfg.Backup.Enabled())
	assert.Equal(BackupDestinationNone, cfg.Backup.Destination.Driver)
}
