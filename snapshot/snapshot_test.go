package snapshot

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flarexio/wallet/account"
	"github.com/flarexio/wallet/backup"
	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/persistence"
)

const passphrase = "test-passphrase"

type fakeDestination struct {
	put     []string
	deleted []string
	names   []string
}

func (d *fakeDestination) Put(ctx context.Context, name string, r io.Reader) error {
	d.put = append(d.put, name)
	d.names = append(d.names, name)

	_, err := io.Copy(io.Discard, r)

	return err
}

func (d *fakeDestination) List(ctx context.Context) ([]string, error) {
	return d.names, nil
}

func (d *fakeDestination) Delete(ctx context.Context, name string) error {
	d.deleted = append(d.deleted, name)
	return nil
}

func (d *fakeDestination) String() string { return "fake" }

func TestExpiredKeepsTheNewest(t *testing.T) {
	assert := assert.New(t)

	names := []string{
		"wallet-20260101T000000Z.bak",
		"wallet-20260103T000000Z.bak",
		"wallet-20260102T000000Z.bak",
	}

	assert.Equal(
		[]string{"wallet-20260101T000000Z.bak"},
		expired(names, storePrefix, storeSuffix, 2),
	)

	assert.Nil(expired(names, storePrefix, storeSuffix, 3))
	assert.Nil(expired(names, storePrefix, storeSuffix, 9))
}

// The timestamp sorts lexically in the same order it sorts chronologically,
// which is the only reason retention can work off names.
func TestStampSortsChronologically(t *testing.T) {
	assert := assert.New(t)

	earlier := time.Date(2026, 9, 9, 23, 59, 59, 0, time.UTC).Format(stamp)
	later := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC).Format(stamp)

	assert.Less(earlier, later)
}

// A run that predates the audit backup must not make the two sets age at
// different rates, so each kind is counted on its own.
func TestPruneCountsStoreAndAuditSeparately(t *testing.T) {
	assert := assert.New(t)

	dst := &fakeDestination{names: []string{
		"wallet-20260101T000000Z.bak",
		"wallet-20260102T000000Z.bak",
		"wallet-20260103T000000Z.bak",
		"audit-20260103T000000Z.log.enc",
		"something-else.txt",
	}}

	s := &Scheduler{Destination: dst, Keep: 2}

	if !assert.NoError(s.prune(context.Background())) {
		return
	}

	assert.Equal([]string{"wallet-20260101T000000Z.bak"}, dst.deleted)
}

func TestCheckRejectsAnIncompleteScheduler(t *testing.T) {
	assert := assert.New(t)

	full := Scheduler{
		Source:      nil,
		Destination: &fakeDestination{},
		Passphrase:  passphrase,
		Interval:    time.Hour,
		Keep:        1,
	}

	assert.ErrorIs(full.Validate(), ErrNoSource)

	s := full
	s.Source = &fakeSource{}
	assert.NoError(s.Validate())

	s.Destination = nil
	assert.ErrorIs(s.Validate(), ErrNoDestination)

	s = full
	s.Source = &fakeSource{}
	s.Interval = 0
	assert.ErrorIs(s.Validate(), ErrNoInterval)

	s = full
	s.Source = &fakeSource{}
	s.Keep = 0
	assert.ErrorIs(s.Validate(), ErrNoKeep)

	// A scheduled backup nobody can decrypt is worse than none.
	s = full
	s.Source = &fakeSource{}
	s.Passphrase = "short"
	assert.ErrorIs(s.Validate(), backup.ErrShortPassphrase)
}

type fakeSource struct{}

func (fakeSource) Snapshot(w io.Writer, since uint64) (uint64, error) {
	_, err := w.Write([]byte("snapshot"))
	return 1, err
}

func TestCopyAuditLogToleratesAMissingFile(t *testing.T) {
	assert := assert.New(t)

	s := &Scheduler{AuditPath: filepath.Join(t.TempDir(), "audit.log")}

	var buf bytes.Buffer
	assert.NoError(s.copyAuditLog(&buf))
	assert.Zero(buf.Len())
}

func TestDirDestinationRoundTrip(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "backups")

	dst, err := NewDir(path)
	if !assert.NoError(err) {
		return
	}

	if !assert.NoError(dst.Put(ctx, "wallet-1.bak", strings.NewReader("hello"))) {
		return
	}

	names, err := dst.List(ctx)
	if !assert.NoError(err) {
		return
	}

	// The temporary file Put writes through must not survive the rename.
	assert.Equal([]string{"wallet-1.bak"}, names)

	bs, err := os.ReadFile(filepath.Join(path, "wallet-1.bak"))
	if !assert.NoError(err) {
		return
	}

	assert.Equal("hello", string(bs))

	info, err := os.Stat(filepath.Join(path, "wallet-1.bak"))
	if !assert.NoError(err) {
		return
	}

	// A snapshot reproduces every account key given KMS access.
	assert.Equal(os.FileMode(0o600), info.Mode().Perm())

	if !assert.NoError(dst.Delete(ctx, "wallet-1.bak")) {
		return
	}

	names, err = dst.List(ctx)
	assert.NoError(err)
	assert.Empty(names)
}

func testAccount(subject string, n byte) *account.Account {
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = n

	privkey := ed25519.NewKeyFromSeed(seed)

	return &account.Account{
		Subject:    subject,
		Salt:       subject + "-salt",
		KeyVersion: 1,
		Derivation: account.CurrentDerivation,
		PublicKey:  privkey.Public().(ed25519.PublicKey),
	}
}

// The drill nobody runs until the day it matters: take a snapshot of a store
// that is open and serving, then put it back into an empty one and check the
// accounts came through. A backup that has never been restored is not a
// backup.
func TestSnapshotRestoresWhileTheStoreIsOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("encrypts and decrypts at the production work factor")
	}

	assert := assert.New(t)
	ctx := context.Background()

	live := &conf.BadgerPersistenceConfig{Path: t.TempDir(), Name: "wallet"}

	repo, err := persistence.NewBadgerAccountRepository(live)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	alice := testAccount("alice", 1)
	bob := testAccount("bob", 2)

	if !assert.NoError(repo.Save(alice)) || !assert.NoError(repo.Save(bob)) {
		return
	}

	auditPath := filepath.Join(t.TempDir(), "audit.log")
	if err := os.WriteFile(auditPath, []byte(`{"seq":1}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()

	dst, err := NewDir(dir)
	if !assert.NoError(err) {
		return
	}

	s := &Scheduler{
		Source:      repo.(persistence.Snapshotter),
		Destination: dst,
		Passphrase:  passphrase,
		Interval:    time.Hour,
		Keep:        3,
		AuditPath:   auditPath,
	}

	if !assert.NoError(s.Once(ctx)) {
		return
	}

	// The store is still open and serving — that is the point of running the
	// snapshot in process rather than through the CLI.
	if _, err := repo.Find("alice"); !assert.NoError(err) {
		return
	}

	names, err := dst.List(ctx)
	if !assert.NoError(err) {
		return
	}

	var store, auditLog string
	for _, name := range names {
		switch {
		case strings.HasPrefix(name, storePrefix):
			store = name
		case strings.HasPrefix(name, auditPrefix):
			auditLog = name
		}
	}

	if !assert.NotEmpty(store) || !assert.NotEmpty(auditLog) {
		return
	}

	f, err := os.Open(filepath.Join(dir, store))
	if !assert.NoError(err) {
		return
	}
	defer f.Close()

	restored := &conf.BadgerPersistenceConfig{Path: t.TempDir(), Name: "wallet"}

	if !assert.NoError(backup.Read(f, passphrase, func(r io.Reader) error {
		return persistence.Restore(restored, r)
	})) {
		return
	}

	back, err := persistence.NewBadgerAccountRepository(restored)
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()

	for _, want := range []*account.Account{alice, bob} {
		got, err := back.Find(want.Subject)
		if !assert.NoError(err) {
			continue
		}

		assert.Equal(want.Salt, got.Salt)
		assert.Equal(want.KeyVersion, got.KeyVersion)
		assert.Equal(want.Derivation, got.Derivation)
		assert.Equal([]byte(want.PublicKey), []byte(got.PublicKey))
	}

	// The audit log travels with the store: it lives on the same disk and has
	// the same problem.
	af, err := os.Open(filepath.Join(dir, auditLog))
	if !assert.NoError(err) {
		return
	}
	defer af.Close()

	var got bytes.Buffer
	if !assert.NoError(backup.Read(af, passphrase, func(r io.Reader) error {
		_, err := io.Copy(&got, r)
		return err
	})) {
		return
	}

	assert.Equal(`{"seq":1}`+"\n", got.String())
}
