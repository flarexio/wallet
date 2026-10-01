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
