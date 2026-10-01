package keys

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/wallet/conf"
)

func TestGoogleKeyService(t *testing.T) {
	assert := assert.New(t)

	f, err := os.Open("../config.example.yaml")
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer f.Close()

	var cfg conf.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		assert.Fail(err.Error())
		return
	}

	_, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	if !ok {
		t.Skip(`"GOOGLE_APPLICATION_CREDENTIALS" is not set`)
		return
	}

	svc, err := NewGoogleKeysService(cfg.Keys.Google)
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer svc.Close()

	key, err := svc.Key()
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Equal(1, key.Version())

	data := []byte("test")

	sig, err := key.Signature(data)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Len(sig, 64)
	assert.True(key.Verify(data, sig))
}

func TestGoogleCipher(t *testing.T) {
	assert := assert.New(t)

	f, err := os.Open("../config.example.yaml")
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer f.Close()

	var cfg conf.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		assert.Fail(err.Error())
		return
	}

	if _, ok := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS"); !ok {
		t.Skip(`"GOOGLE_APPLICATION_CREDENTIALS" is not set`)
		return
	}

	cipher, err := NewGoogleCipher(cfg.Keys.Records)
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer cipher.Close()

	salt := []byte("0f8fad5b-d9cb-469f-a165-70867728950e")
	wallet := []byte("wallet-address")

	sealed, err := cipher.Seal(salt, wallet)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.NotEqual(salt, sealed)

	opened, err := cipher.Open(sealed, wallet)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Equal(salt, opened)

	// The AAD is what stops a sealed salt from opening under another wallet.
	_, err = cipher.Open(sealed, []byte("another-address"))
	assert.Error(err)
}

func TestGoogleCipherNeedsAKey(t *testing.T) {
	_, err := NewGoogleCipher(conf.GoogleKeyConfig{})
	assert.ErrorIs(t, err, ErrNoCipherKey)
}
