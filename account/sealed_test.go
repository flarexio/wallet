package account

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeCipher is AAD-faithful and nothing else: it records the aad alongside the
// plaintext and refuses to hand it back under a different one.
type fakeCipher struct{}

func (fakeCipher) Seal(plaintext, aad []byte) ([]byte, error) {
	return append(append([]byte{byte(len(aad))}, aad...), plaintext...), nil
}

func (fakeCipher) Open(ciphertext, aad []byte) ([]byte, error) {
	n := int(ciphertext[0])

	if !bytes.Equal(ciphertext[1:1+n], aad) {
		return nil, errors.New("aad mismatch")
	}

	return ciphertext[1+n:], nil
}

func testPubkey(n byte) ed25519.PublicKey {
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = n

	return ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
}

func TestSealSaltRoundTrip(t *testing.T) {
	assert := assert.New(t)

	a := &Account{Subject: "alice", Salt: "alice-salt", PublicKey: testPubkey(1)}

	sealed, err := SealSalt(fakeCipher{}, a)
	if !assert.NoError(err) {
		return
	}

	got, err := OpenSalt(fakeCipher{}, a.PublicKey, sealed)
	assert.NoError(err)
	assert.Equal(a.Salt, got)
}

// A salt lifted out of one record must not open under another wallet.
func TestSealSaltIsBoundToTheWallet(t *testing.T) {
	assert := assert.New(t)

	a := &Account{Subject: "alice", Salt: "alice-salt", PublicKey: testPubkey(1)}

	sealed, err := SealSalt(fakeCipher{}, a)
	if !assert.NoError(err) {
		return
	}

	_, err = OpenSalt(fakeCipher{}, testPubkey(2), sealed)
	assert.Error(err)
}

func TestSealSaltRefusesAnEmptySalt(t *testing.T) {
	assert := assert.New(t)

	a := &Account{Subject: "alice", PublicKey: testPubkey(1)}

	_, err := SealSalt(fakeCipher{}, a)
	assert.ErrorIs(err, ErrNoSalt)
}
