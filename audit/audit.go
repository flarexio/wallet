package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Action string

const (
	SignMessage     Action = "sign_message"
	SignTransaction Action = "sign_transaction"
)

var (
	ErrBrokenChain = errors.New("audit chain is broken")
	ErrNoAction    = errors.New("audit entry has no action")
	ErrNoSubject   = errors.New("audit entry has no subject")
)

// Entry records one released signature. Seq, Time, PrevHash and Hash are filled
// in by Append; everything else describes what was signed and what authorized
// it.
type Entry struct {
	Seq           uint64    `json:"seq"`
	Time          time.Time `json:"time"`
	Subject       string    `json:"subject"`
	Action        Action    `json:"action"`
	TransactionID string    `json:"transaction_id"`
	Wallet        string    `json:"wallet"`
	KeyVersion    int       `json:"key_version"`
	PayloadHash   string    `json:"payload_hash"`
	CredentialID  string    `json:"credential_id"`
	SignCount     uint32    `json:"sign_count"`
	Origin        string    `json:"origin"`
	PrevHash      string    `json:"prev_hash"`
	Hash          string    `json:"hash"`
}

// sum hashes every field except Hash itself. Fields are NUL-separated so that
// moving a character across a boundary cannot produce the same digest.
func (e *Entry) sum() string {
	h := sha256.New()

	for _, field := range []string{
		strconv.FormatUint(e.Seq, 10),
		e.Time.UTC().Format(time.RFC3339Nano),
		e.Subject,
		string(e.Action),
		e.TransactionID,
		e.Wallet,
		strconv.Itoa(e.KeyVersion),
		e.PayloadHash,
		e.CredentialID,
		strconv.FormatUint(uint64(e.SignCount), 10),
		e.Origin,
		e.PrevHash,
	} {
		h.Write([]byte(field))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil))
}

type Log interface {
	Append(e *Entry) (*Entry, error)
	Entries() ([]*Entry, error)
	Verify() error
	Close() error
}

// chain tracks the tail of the log so the next entry can link to it.
type chain struct {
	seq  uint64
	head string
}

func (c *chain) link(e *Entry) (*Entry, error) {
	if e == nil {
		return nil, errors.New("nil audit entry")
	}

	if e.Subject == "" {
		return nil, ErrNoSubject
	}

	if e.Action == "" {
		return nil, ErrNoAction
	}

	next := *e
	next.Seq = c.seq + 1

	if next.Time.IsZero() {
		next.Time = time.Now()
	}
	next.Time = next.Time.UTC()

	next.PrevHash = c.head
	next.Hash = next.sum()

	return &next, nil
}

func (c *chain) advance(e *Entry) {
	c.seq = e.Seq
	c.head = e.Hash
}

// Verify recomputes every hash and checks that each entry links to the one
// before it. A rewritten or removed entry breaks the chain from that point on.
func Verify(entries []*Entry) error {
	var prev string

	for i, e := range entries {
		if e.Seq != uint64(i+1) {
			return fmt.Errorf("%w: entry %d has seq %d", ErrBrokenChain, i+1, e.Seq)
		}

		if e.PrevHash != prev {
			return fmt.Errorf("%w: entry %d does not link to entry %d", ErrBrokenChain, e.Seq, i)
		}

		if sum := e.sum(); sum != e.Hash {
			return fmt.Errorf("%w: entry %d has been modified", ErrBrokenChain, e.Seq)
		}

		prev = e.Hash
	}

	return nil
}

func NewMemoryLog() Log {
	return &memoryLog{}
}

type memoryLog struct {
	chain   chain
	entries []*Entry
	mu      sync.Mutex
}

func (l *memoryLog) Append(e *Entry) (*Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	next, err := l.chain.link(e)
	if err != nil {
		return nil, err
	}

	l.entries = append(l.entries, next)
	l.chain.advance(next)

	return next, nil
}

func (l *memoryLog) Entries() ([]*Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]*Entry, len(l.entries))
	copy(out, l.entries)

	return out, nil
}

func (l *memoryLog) Verify() error {
	entries, err := l.Entries()
	if err != nil {
		return err
	}

	return Verify(entries)
}

func (l *memoryLog) Close() error {
	return nil
}
