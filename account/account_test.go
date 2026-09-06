package account

import (
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubKey struct{ version int }

func (k *stubKey) Signature(data []byte) ([]byte, error) {
	sum := sha512.Sum512(append([]byte("stub-kms/"), data...))
	return sum[:], nil
}

func (k *stubKey) Verify(data []byte, sig []byte) (bool, error) {
	want, err := k.Signature(data)
	if err != nil {
		return false, err
	}

	return string(want) == string(sig), nil
}

func (k *stubKey) Version() int             { return k.version }
func (k *stubKey) Public() crypto.PublicKey { return nil }

func (k *stubKey) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return k.Signature(digest)
}

// Recoverability rests on this: the same subject, salt and KMS key must always
// give back the same account key.
func TestDeriveIsDeterministic(t *testing.T) {
	assert := assert.New(t)

	key := &stubKey{version: 2}

	first, err := Derive("alice", "salt", key)
	if !assert.NoError(err) {
		return
	}

	second, err := Derive("alice", "salt", key)
	if !assert.NoError(err) {
		return
	}

	assert.Equal(first, second)
}

func TestDeriveSeparatesSubjectsAndSalts(t *testing.T) {
	assert := assert.New(t)

	key := &stubKey{version: 1}

	alice, err := Derive("alice", "salt", key)
	if !assert.NoError(err) {
		return
	}

	bob, err := Derive("bob", "salt", key)
	if !assert.NoError(err) {
		return
	}

	resalted, err := Derive("alice", "other-salt", key)
	if !assert.NoError(err) {
		return
	}

	assert.NotEqual(alice, bob)
	assert.NotEqual(alice, resalted)
}

// The seed used to be the first 32 bytes of the KMS signature. Those bytes are
// R, which the signature scheme publishes, so anything that exposed a KMS
// signature exposed the account key. HKDF breaks that link.
func TestDeriveDoesNotSeedFromTheRawSignature(t *testing.T) {
	assert := assert.New(t)

	key := &stubKey{version: 1}

	sig, err := key.Signature([]byte("alice" + "salt"))
	if !assert.NoError(err) {
		return
	}

	naive := ed25519.NewKeyFromSeed(sig[:ed25519.SeedSize])

	derived, err := Derive("alice", "salt", key)
	if !assert.NoError(err) {
		return
	}

	assert.NotEqual(naive, derived)
	assert.NotContains(string(derived.Seed()), string(sig[:ed25519.SeedSize]))
}

func TestNewAccountCarriesThePublicKey(t *testing.T) {
	assert := assert.New(t)

	key := &stubKey{version: 4}

	a, privkey, err := NewAccount("alice", key)
	if !assert.NoError(err) {
		return
	}

	assert.Equal("alice", a.Subject)
	assert.Equal(4, a.KeyVersion)
	assert.NotEmpty(a.Salt)

	pub, ok := privkey.Public().(ed25519.PublicKey)
	if !assert.True(ok) {
		return
	}

	assert.True(pub.Equal(a.PublicKey))
	assert.Equal(pub, ed25519.PublicKey(a.Wallet().Bytes()))

	// The salt is what makes the account reproducible.
	again, err := Derive("alice", a.Salt, key)
	if assert.NoError(err) {
		assert.Equal(privkey, again)
	}
}

// The persisted record must not carry key material: a stolen store should be
// useless without the KMS key.
func TestAccountRecordHasNoPrivateKey(t *testing.T) {
	assert := assert.New(t)

	a, privkey, err := NewAccount("alice", &stubKey{version: 1})
	if !assert.NoError(err) {
		return
	}

	bs, err := json.Marshal(a)
	if !assert.NoError(err) {
		return
	}

	record := string(bs)

	assert.NotContains(record, "PrivateKey")
	assert.NotContains(record, string(privkey))
	assert.NotContains(record, string(privkey.Seed()))

	var back *Account
	if !assert.NoError(json.Unmarshal(bs, &back)) {
		return
	}

	assert.Equal(a.Subject, back.Subject)
	assert.Equal(a.Salt, back.Salt)
	assert.Equal(a.KeyVersion, back.KeyVersion)
	assert.True(back.PublicKey.Equal(a.PublicKey))
	assert.True(strings.Contains(record, "PublicKey"))
}
