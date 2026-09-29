// Package snapshot takes scheduled encrypted backups of a running wallet and
// ships them off the machine. The per-account salt lives nowhere but the
// account store, so one disk failure empties every wallet.
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

// Destination is where finished snapshots are kept. String is for logs and
// must not reveal credentials.
type Destination interface {
	Put(ctx context.Context, name string, r io.Reader) error
	List(ctx context.Context) ([]string, error)
	Delete(ctx context.Context, name string) error
	String() string
}

const (
	storePrefix = "wallet-"
	storeSuffix = ".bak"

	auditPrefix = "audit-"
	auditSuffix = ".log.enc"

	// Sorts lexically in chronological order, which is what retention needs.
	stamp = "20060102T150405Z"
)

var (
	ErrNoSource      = errors.New("snapshot needs a source")
	ErrNoDestination = errors.New("snapshot needs a destination")
	ErrNoInterval    = errors.New("snapshot needs a positive interval")
	ErrNoKeep        = errors.New("snapshot needs to keep at least one run")
)

type Scheduler struct {
	Source      Source
	Destination Destination
	Passphrase  string
	Interval    time.Duration

	Keep int

	// AuditPath is the signature log, backed up alongside the store. Empty
	// skips it.
	AuditPath string

	Log *zap.Logger
}

// Validate is called by Run, but Run is normally started in a goroutine where
// its error would go nowhere, so callers check first.
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
// logged and retried on the next tick rather than taking the wallet down.
// Nothing is taken at startup: a crash-looping service would push every good
// run out through retention.
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

// Once backs up the store and the audit log, then prunes. Snapshots are always
// full — records are around a hundred bytes, and a full run leaves every file
// independently restorable.
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

// copyAuditLog needs no locking: a prefix of a hash chain still verifies.
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

// prune counts store and audit files separately, so runs that predate the
// audit backup do not age the two sets at different rates.
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
