package account

import (
	"errors"

	"github.com/flarexio/wallet/keys"
)

var ErrNoSalt = errors.New("account has no salt to seal")

// SealSalt encrypts the salt for storage somewhere the account store is not —
// on chain, in a shared bucket. The wallet address is the AAD, so a salt lifted
// out of one record cannot be opened under another.
func SealSalt(cipher keys.Cipher, a *Account) ([]byte, error) {
	if a.Salt == "" {
		return nil, ErrNoSalt
	}

	return cipher.Seal([]byte(a.Salt), a.PublicKey)
}

// OpenSalt reverses SealSalt. It takes the public key rather than the account
// because the account being rebuilt does not have a salt yet.
func OpenSalt(cipher keys.Cipher, pubkey []byte, sealed []byte) (string, error) {
	salt, err := cipher.Open(sealed, pubkey)
	if err != nil {
		return "", err
	}

	return string(salt), nil
}
