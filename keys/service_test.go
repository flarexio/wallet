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

	svc, err := NewGoogleKeysService(cfg.Keys.Google)
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer svc.Close()

	data := []byte("test")

	sign, err := svc.Signature(data)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.True(svc.Verify(data, sign))
}
