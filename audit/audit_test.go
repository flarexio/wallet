package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func entry(subject string, action Action) *Entry {
	return &Entry{
		Subject:       subject,
		Action:        action,
		TransactionID: "b3f1c2d4-0000-4000-8000-000000000001",
		Wallet:        "So11111111111111111111111111111111111111112",
		KeyVersion:    1,
		PayloadHash:   "0f0e0d0c0b0a09080706050403020100f0e0d0c0b0a090807060504030201000",
		CredentialID:  "cred-1",
		SignCount:     7,
		Origin:        "https://wallet.flarex.io",
	}
}

func TestChainLinksEntries(t *testing.T) {
	assert := assert.New(t)

	l := NewMemoryLog()

	first, err := l.Append(entry("alice", SignMessage))
	if !assert.NoError(err) {
		return
	}

	second, err := l.Append(entry("alice", SignTransaction))
	if !assert.NoError(err) {
		return
	}

	assert.Equal(uint64(1), first.Seq)
	assert.Equal("", first.PrevHash)
	assert.Equal(uint64(2), second.Seq)
	assert.Equal(first.Hash, second.PrevHash)
	assert.NotEqual(first.Hash, second.Hash)
	assert.NoError(l.Verify())
}

func TestAppendRejectsIncompleteEntry(t *testing.T) {
	assert := assert.New(t)

	l := NewMemoryLog()

	_, err := l.Append(&Entry{Action: SignMessage})
	assert.ErrorIs(err, ErrNoSubject)

	_, err = l.Append(&Entry{Subject: "alice"})
	assert.ErrorIs(err, ErrNoAction)

	entries, err := l.Entries()
	assert.NoError(err)
	assert.Empty(entries, "a rejected entry must not consume a sequence number")
}

func TestVerifyDetectsEdit(t *testing.T) {
	assert := assert.New(t)

	l := NewMemoryLog()

	for _, sub := range []string{"alice", "bob", "carol"} {
		if _, err := l.Append(entry(sub, SignTransaction)); !assert.NoError(err) {
			return
		}
	}

	entries, err := l.Entries()
	if !assert.NoError(err) {
		return
	}

	assert.NoError(Verify(entries))

	entries[1].Subject = "mallory"
	assert.ErrorIs(Verify(entries), ErrBrokenChain)
}

func TestVerifyDetectsRemoval(t *testing.T) {
	assert := assert.New(t)

	l := NewMemoryLog()

	for _, sub := range []string{"alice", "bob", "carol"} {
		if _, err := l.Append(entry(sub, SignTransaction)); !assert.NoError(err) {
			return
		}
	}

	entries, err := l.Entries()
	if !assert.NoError(err) {
		return
	}

	shortened := []*Entry{entries[0], entries[2]}
	assert.ErrorIs(Verify(shortened), ErrBrokenChain)
}

func TestFileLogSurvivesReopen(t *testing.T) {
	assert := assert.New(t)

	path := filepath.Join(t.TempDir(), "audit.log")

	l, err := NewFileLog(path)
	if !assert.NoError(err) {
		return
	}

	first, err := l.Append(entry("alice", SignMessage))
	if !assert.NoError(err) {
		return
	}
	assert.NoError(l.Close())

	reopened, err := NewFileLog(path)
	if !assert.NoError(err) {
		return
	}
	defer reopened.Close()

	second, err := reopened.Append(entry("bob", SignTransaction))
	if !assert.NoError(err) {
		return
	}

	assert.Equal(uint64(2), second.Seq, "sequence continues across a restart")
	assert.Equal(first.Hash, second.PrevHash, "chain links across a restart")
	assert.NoError(reopened.Verify())
}

// Entries are hashed from their fields, so a hash computed before writing has
// to match one recomputed after a JSON round trip -- time formatting included.
func TestFileLogHashSurvivesJSON(t *testing.T) {
	assert := assert.New(t)

	path := filepath.Join(t.TempDir(), "audit.log")

	l, err := NewFileLog(path)
	if !assert.NoError(err) {
		return
	}
	defer l.Close()

	written, err := l.Append(entry("alice", SignMessage))
	if !assert.NoError(err) {
		return
	}

	entries, err := l.Entries()
	if !assert.NoError(err) {
		return
	}

	if !assert.Len(entries, 1) {
		return
	}

	assert.Equal(written.Hash, entries[0].Hash)
	assert.Equal(written.Hash, entries[0].sum())
	assert.True(written.Time.Equal(entries[0].Time))
}

func TestFileLogRefusesTamperedLog(t *testing.T) {
	assert := assert.New(t)

	path := filepath.Join(t.TempDir(), "audit.log")

	l, err := NewFileLog(path)
	if !assert.NoError(err) {
		return
	}

	for _, sub := range []string{"alice", "bob"} {
		if _, err := l.Append(entry(sub, SignTransaction)); !assert.NoError(err) {
			return
		}
	}
	assert.NoError(l.Close())

	bs, err := os.ReadFile(path)
	if !assert.NoError(err) {
		return
	}

	lines := strings.Split(strings.TrimRight(string(bs), "\n"), "\n")
	if !assert.Len(lines, 2) {
		return
	}

	var edited map[string]any
	if !assert.NoError(json.Unmarshal([]byte(lines[0]), &edited)) {
		return
	}
	edited["subject"] = "mallory"

	line, err := json.Marshal(edited)
	if !assert.NoError(err) {
		return
	}

	if !assert.NoError(os.WriteFile(path, []byte(string(line)+"\n"+lines[1]+"\n"), 0o600)) {
		return
	}

	_, err = NewFileLog(path)
	assert.ErrorIs(err, ErrBrokenChain, "a tampered log must not be reopened for appending")
}

func TestFileLogStartsEmpty(t *testing.T) {
	assert := assert.New(t)

	l, err := NewFileLog(filepath.Join(t.TempDir(), "audit.log"))
	if !assert.NoError(err) {
		return
	}
	defer l.Close()

	entries, err := l.Entries()
	assert.NoError(err)
	assert.Empty(entries)
	assert.NoError(l.Verify())
}
