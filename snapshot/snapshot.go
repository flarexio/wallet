// Package snapshot takes scheduled encrypted backups of a running wallet and
// ships them off the machine.
//
// The per-account salt lives nowhere but the account store, and the KMS key
// alone rebuilds nothing without it — so a store that exists on exactly one
// disk is a set of wallets that one disk failure empties. Keeping a copy
// somewhere else is the whole job of this package.
package snapshot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/flarexio/wallet/backup"
)

// Source is the account store, already open and serving.
type Source interface {
	Snapshot(w io.Writer, since uint64) (uint64, error)
}

// Destination is where finished snapshots are kept. Names are opaque to it and
// sort chronologically, which is what retention relies on.
type Destination interface {
	Put(ctx context.Context, name string, r io.Reader) error
	List(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, name string) error

	// String names the destination for logs. It must not reveal credentials.
	String() string
}

const (
	storePrefix = "wallet-"
	storeSuffix = ".bak"

	auditPrefix = "audit-"
	auditSuffix = ".log.enc"

	// stamp sorts lexically in the same order as it sorts chronologically,
	// which is what lets retention work off names alone.
	stamp = "20060102T150405Z"
)

var (
	ErrNoSource      = errors.New("snapshot needs a source")
	ErrNoDestination = errors.New("snapshot needs a destination")
	ErrNoInterval    = errors.New("snapshot needs a positive interval")
	ErrNoKeep        = errors.New("snapshot needs to keep at least one run")
)

// Scheduler runs Once on an interval for as long as its context lives.
type Scheduler struct {
	Source      Source
	Destination Destination
	Passphrase  string
	Interval    time.Duration

	// Keep is how many runs to leave at the destination. Older ones are
	// deleted after a successful upload, never before.
	Keep int

	// AuditPath is the hash-chained signature log, backed up alongside the
	// store. It lives on the same disk and has the same problem. Empty skips
	// it.
	AuditPath string

	Log *zap.Logger
}

// Validate reports whether the schedule is complete enough to run. Run calls
// it too, but Run is normally started in a goroutine where its error would go
// nowhere — so the caller checks first and refuses to start the server.
func (s *Scheduler) Validate() error {
	switch {
	case s.Source == nil:
		return ErrNoSource
	case s.Destination == nil:
		return ErrNoDestination
	case s.Interval <= 0:
		return ErrNoInterval
	case s.Keep < 1:
		return ErrNoKeep
	}

	return backup.CheckPassphrase(s.Passphrase)
}

// Run takes a snapshot every Interval until ctx is cancelled. A failed run is
// logged and retried at the next tick rather than stopping the schedule: a
// backup that cannot be written is not a reason to take the wallet down, but
// it is a reason to be loud about it.
//
// Nothing is taken at startup. A service that is crash-looping would otherwise
// write a snapshot per restart and push every good one out through retention,
// which is the opposite of what a backup is for.
func (s *Scheduler) Run(ctx context.Context) error {
	if err := s.Validate(); err != nil {
		return err
	}

	log := s.logger().With(
		zap.String("destination", s.Destination.String()),
		zap.Duration("interval", s.Interval),
		zap.Int("keep", s.Keep),
	)

	log.Info("scheduled backup started")

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("scheduled backup stopped")
			return ctx.Err()

		case <-ticker.C:
			if err := s.Once(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				log.Error("backup failed", zap.Error(err))
			}
		}
	}
}

// Once writes one snapshot of the store, one of the audit log, and then prunes
// whatever the retention no longer covers.
//
// Snapshots are always taken in full. badger can stream incrementally, but the
// records here are around a hundred bytes each, and a full run leaves every
// file independently restorable instead of only meaningful as part of a chain.
func (s *Scheduler) Once(ctx context.Context) error {
	if err := s.Validate(); err != nil {
		return err
	}

	at := time.Now().UTC().Format(stamp)
	log := s.logger()

	name := storePrefix + at + storeSuffix
	if err := s.put(ctx, name, func(w io.Writer) error {
		_, err := s.Source.Snapshot(w, 0)
		return err
	}); err != nil {
		return fmt.Errorf("store: %w", err)
	}

	log.Info("store backed up", zap.String("name", name))

	if s.AuditPath != "" {
		name := auditPrefix + at + auditSuffix

		if err := s.put(ctx, name, s.copyAuditLog); err != nil {
			return fmt.Errorf("audit log: %w", err)
		}

		log.Info("audit log backed up", zap.String("name", name))
	}

	return s.prune(ctx)
}

// copyAuditLog reads whatever is on disk right now. An append landing mid-read
// just means the copy stops a few entries short, and a prefix of a hash chain
// still verifies — so this does not need to stop the log to be consistent.
func (s *Scheduler) copyAuditLog(w io.Writer) error {
	f, err := os.Open(s.AuditPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)

	return err
}

func (s *Scheduler) put(ctx context.Context, name string, fn func(io.Writer) error) error {
	var buf bytes.Buffer

	if err := backup.Write(&buf, s.Passphrase, fn); err != nil {
		return err
	}

	return s.Destination.Put(ctx, name, &buf)
}

// prune deletes the runs the retention no longer covers. Store and audit files
// are counted separately so that a run that predates the audit backup does not
// make the two sets age at different rates.
func (s *Scheduler) prune(ctx context.Context) error {
	names, err := s.Destination.List(ctx)
	if err != nil {
		return err
	}

	stale := append(
		expired(names, storePrefix, storeSuffix, s.Keep),
		expired(names, auditPrefix, auditSuffix, s.Keep)...,
	)

	for _, name := range stale {
		if err := s.Destination.Delete(ctx, name); err != nil {
			return err
		}

		s.logger().Info("old backup removed", zap.String("name", name))
	}

	return nil
}

// expired returns the names beyond the newest keep, oldest first.
func expired(names []string, prefix, suffix string, keep int) []string {
	matched := make([]string, 0, len(names))

	for _, name := range names {
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			matched = append(matched, name)
		}
	}

	if len(matched) <= keep {
		return nil
	}

	sort.Strings(matched)

	return matched[:len(matched)-keep]
}

func (s *Scheduler) logger() *zap.Logger {
	if s.Log == nil {
		return zap.NewNop()
	}

	return s.Log
}
