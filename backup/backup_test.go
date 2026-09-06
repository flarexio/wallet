package backup

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const goodPass = "correct horse battery staple"

// Most tests are about the file format, not the work factor, and the real one
// costs seconds per derivation under the race detector.
const testIterations = minIterations

func sealed(t *testing.T, payload string) []byte {
	t.Helper()

	var buf bytes.Buffer

	err := write(&buf, goodPass, testIterations, func(w io.Writer) error {
		_, err := w.Write([]byte(payload))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func opened(t *testing.T, bs []byte, pass string) (string, error) {
	t.Helper()

	var got string

	err := Read(bytes.NewReader(bs), pass, func(r io.Reader) error {
		out, err := io.ReadAll(r)
		got = string(out)

		return err
	})

	return got, err
}

func TestRoundTrip(t *testing.T) {
	assert := assert.New(t)

	payload := strings.Repeat("account records ", 1000)

	got, err := opened(t, sealed(t, payload), goodPass)
	if assert.NoError(err) {
		assert.Equal(payload, got)
	}
}

// The snapshot is fund-bearing, so the file must not be readable without the
// passphrase.
func TestWrongPassphraseIsRefused(t *testing.T) {
	assert := assert.New(t)

	_, err := opened(t, sealed(t, "secrets"), "not the passphrase")
	assert.ErrorIs(err, ErrWrongPassphrase)
}

func TestPlaintextIsNotInTheFile(t *testing.T) {
	assert := assert.New(t)

	bs := sealed(t, "PRIVATE-KEY-MATERIAL")

	assert.NotContains(string(bs), "PRIVATE-KEY-MATERIAL")
}

func TestTamperedCiphertextIsRefused(t *testing.T) {
	assert := assert.New(t)

	bs := sealed(t, "secrets")
	bs[len(bs)-1] ^= 0xff

	_, err := opened(t, bs, goodPass)
	assert.ErrorIs(err, ErrWrongPassphrase)
}

// The salt and nonce are authenticated, so a file cannot be re-headed to make
// it decrypt to something else.
func TestTamperedHeaderIsRefused(t *testing.T) {
	assert := assert.New(t)

	bs := sealed(t, "secrets")
	bs[len(magic)+iterLen] ^= 0xff

	_, err := opened(t, bs, goodPass)
	assert.ErrorIs(err, ErrWrongPassphrase)
}

func TestNotABackup(t *testing.T) {
	assert := assert.New(t)

	for _, input := range []string{"", "short", "this is not a backup file at all"} {
		_, err := opened(t, []byte(input), goodPass)
		assert.ErrorIs(err, ErrNotABackup, "input %q", input)
	}
}

func TestTruncatedBackup(t *testing.T) {
	assert := assert.New(t)

	bs := sealed(t, "secrets")

	_, err := opened(t, bs[:headerLen-1], goodPass)
	assert.ErrorIs(err, ErrNotABackup)
}

func TestPassphraseRules(t *testing.T) {
	assert := assert.New(t)

	assert.ErrorIs(CheckPassphrase(""), ErrEmptyPassphrase)
	assert.ErrorIs(CheckPassphrase("short"), ErrShortPassphrase)
	assert.NoError(CheckPassphrase(goodPass))

	err := write(io.Discard, "short", testIterations, func(w io.Writer) error {
		assert.Fail("must not snapshot with a rejected passphrase")
		return nil
	})
	assert.ErrorIs(err, ErrShortPassphrase)

	_, err = opened(t, sealed(t, "secrets"), "")
	assert.ErrorIs(err, ErrEmptyPassphrase)
}

// Every backup gets its own salt and nonce, so two snapshots of the same data
// do not produce the same file.
func TestBackupsAreNotIdentical(t *testing.T) {
	assert := assert.New(t)

	first := sealed(t, "same payload")
	second := sealed(t, "same payload")

	assert.NotEqual(first, second)

	got, err := opened(t, second, goodPass)
	if assert.NoError(err) {
		assert.Equal("same payload", got)
	}
}

func TestSnapshotErrorIsReturned(t *testing.T) {
	assert := assert.New(t)

	var buf bytes.Buffer

	want := io.ErrClosedPipe
	err := write(&buf, goodPass, testIterations, func(w io.Writer) error { return want })

	assert.ErrorIs(err, want)
	assert.Empty(buf.Bytes(), "a failed snapshot must not leave a partial file")
}

// The work factor lives in the header so it can be raised without orphaning
// older backups -- but a file claiming an absurd one must not be honoured.
func TestImplausibleWorkFactorIsRefused(t *testing.T) {
	assert := assert.New(t)

	for _, iter := range []uint32{0, 1, minIterations - 1, maxIterations + 1} {
		bs := sealed(t, "secrets")
		binary.BigEndian.PutUint32(bs[len(magic):], iter)

		_, err := opened(t, bs, goodPass)
		assert.ErrorIs(err, ErrBadIterations, "iterations %d", iter)
	}
}

func TestHeaderRecordsTheWorkFactor(t *testing.T) {
	assert := assert.New(t)

	bs := sealed(t, "secrets")

	assert.Equal(uint32(testIterations), binary.BigEndian.Uint32(bs[len(magic):]))
}

// The only test that pays the real work factor, so the production path is
// covered without every other test spending seconds per derivation. Skipped
// under -short.
func TestRoundTripAtProductionCost(t *testing.T) {
	if testing.Short() {
		t.Skip("pbkdf2 at 600k iterations")
	}

	assert := assert.New(t)

	var buf bytes.Buffer
	if !assert.NoError(Write(&buf, goodPass, func(w io.Writer) error {
		_, err := w.Write([]byte("account records"))
		return err
	})) {
		return
	}

	assert.Equal(uint32(Iterations), binary.BigEndian.Uint32(buf.Bytes()[len(magic):]),
		"Write uses the production work factor")

	got, err := opened(t, buf.Bytes(), goodPass)
	if assert.NoError(err) {
		assert.Equal("account records", got)
	}
}
