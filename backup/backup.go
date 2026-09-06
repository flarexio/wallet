// Package backup writes and reads encrypted snapshots of the account store.
//
// A snapshot is fund-bearing on its own: records written before keys stopped
// being persisted carry the private key outright, and newer ones carry the
// salts, which reproduce every key given KMS access. So the file is always
// encrypted, and the passphrase is never derived from anything in the store.
package backup

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	magic    = "FLAREXWB1"
	iterLen  = 4
	saltLen  = 16
	nonceLen = 12
	keyLen   = 32

	headerLen = len(magic) + iterLen + saltLen + nonceLen

	// Iterations follows the OWASP guidance for PBKDF2-HMAC-SHA256. It is
	// written into the header so that raising it later does not orphan older
	// backups, and so tests can use a cheaper factor.
	Iterations = 600_000

	minIterations = 1_000
	maxIterations = 10_000_000

	// MinPassphrase is short enough to type and long enough that the KDF is
	// doing the work rather than the alphabet.
	MinPassphrase = 12
)

var (
	ErrNotABackup      = errors.New("not a flarex wallet backup")
	ErrWrongPassphrase = errors.New("wrong passphrase, or the backup is corrupt")
	ErrShortPassphrase = fmt.Errorf("passphrase must be at least %d characters", MinPassphrase)
	ErrEmptyPassphrase = errors.New("passphrase is required")
	ErrBadIterations   = errors.New("backup declares an implausible work factor")
)

// Write runs fn to produce the snapshot and writes it to w, encrypted.
func Write(w io.Writer, passphrase string, fn func(io.Writer) error) error {
	return write(w, passphrase, Iterations, fn)
}

func write(w io.Writer, passphrase string, iter int, fn func(io.Writer) error) error {
	if err := CheckPassphrase(passphrase); err != nil {
		return err
	}

	var plain bytes.Buffer
	if err := fn(&plain); err != nil {
		return err
	}

	header := make([]byte, headerLen)
	copy(header, magic)
	binary.BigEndian.PutUint32(header[len(magic):], uint32(iter))

	salt := header[len(magic)+iterLen : len(magic)+iterLen+saltLen]
	nonce := header[len(magic)+iterLen+saltLen:]

	if _, err := rand.Read(salt); err != nil {
		return err
	}

	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	gcm, err := newCipher(passphrase, salt, iter)
	if err != nil {
		return err
	}

	if _, err := w.Write(header); err != nil {
		return err
	}

	// The header is authenticated, so the work factor, salt and nonce cannot be
	// rewritten to make the file open more cheaply or decrypt to something else.
	_, err = w.Write(gcm.Seal(nil, nonce, plain.Bytes(), header))

	return err
}

// Read decrypts r and hands the snapshot to fn.
func Read(r io.Reader, passphrase string, fn func(io.Reader) error) error {
	if passphrase == "" {
		return ErrEmptyPassphrase
	}

	header := make([]byte, headerLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return ErrNotABackup
	}

	if string(header[:len(magic)]) != magic {
		return ErrNotABackup
	}

	iter := int(binary.BigEndian.Uint32(header[len(magic):]))
	if iter < minIterations || iter > maxIterations {
		return ErrBadIterations
	}

	salt := header[len(magic)+iterLen : len(magic)+iterLen+saltLen]
	nonce := header[len(magic)+iterLen+saltLen:]

	sealed, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	gcm, err := newCipher(passphrase, salt, iter)
	if err != nil {
		return err
	}

	plain, err := gcm.Open(nil, nonce, sealed, header)
	if err != nil {
		return ErrWrongPassphrase
	}

	return fn(bytes.NewReader(plain))
}

func CheckPassphrase(passphrase string) error {
	if passphrase == "" {
		return ErrEmptyPassphrase
	}

	if len([]rune(passphrase)) < MinPassphrase {
		return ErrShortPassphrase
	}

	return nil
}

func newCipher(passphrase string, salt []byte, iter int) (cipher.AEAD, error) {
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, iter, keyLen)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}
