package keys

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testCipher(t *testing.T) Cipher {
	t.Helper()

	c, err := NewCipher([32]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestCipherRoundTrip(t *testing.T) {
	assert := assert.New(t)

	c := testCipher(t)
	salt := []byte("0f8fad5b-d9cb-469f-a165-70867728950e")

	sealed, err := c.Seal(salt, []byte("wallet-a"))
	if !assert.NoError(err) {
		return
	}

	assert.NotContains(string(sealed), string(salt))

	opened, err := c.Open(sealed, []byte("wallet-a"))
	assert.NoError(err)
	assert.Equal(salt, opened)
}

// A sealed salt must not open under another wallet.
func TestCipherIsBoundToTheAAD(t *testing.T) {
	c := testCipher(t)

	sealed, err := c.Seal([]byte("salt"), []byte("wallet-a"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Open(sealed, []byte("wallet-b"))
	assert.Error(t, err)
}

func TestCipherRejectsTampering(t *testing.T) {
	c := testCipher(t)

	sealed, err := c.Seal([]byte("salt"), nil)
	if err != nil {
		t.Fatal(err)
	}

	sealed[len(sealed)-1] ^= 1

	_, err = c.Open(sealed, nil)
	assert.Error(t, err)
}

// The example ships an all-zero key; sealing under it would publish salts
// anyone can open.
func TestCipherRefusesTheZeroKey(t *testing.T) {
	_, err := NewCipher([32]byte{})
	assert.ErrorIs(t, err, ErrNoCipherKey)
}

func TestCipherRejectsShortInput(t *testing.T) {
	_, err := testCipher(t).Open([]byte("short"), nil)
	assert.ErrorIs(t, err, ErrShortCiphertext)
}
