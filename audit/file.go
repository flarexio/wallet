package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
)

// NewFileLog opens an append-only JSONL log at path, replaying what is already
// there to recover the tail of the chain. An existing log that does not verify
// is refused rather than extended.
func NewFileLog(path string) (Log, error) {
	entries, err := readEntries(path)
	if err != nil {
		return nil, err
	}

	if err := Verify(entries); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}

	l := &fileLog{path: path, f: f}
	if n := len(entries); n > 0 {
		l.chain.advance(entries[n-1])
	}

	return l, nil
}

type fileLog struct {
	path  string
	f     *os.File
	chain chain
	mu    sync.Mutex
}

func (l *fileLog) Append(e *Entry) (*Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	next, err := l.chain.link(e)
	if err != nil {
		return nil, err
	}

	line, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}

	if _, err := l.f.Write(append(line, '\n')); err != nil {
		return nil, err
	}

	// The caller only releases a signature once this returns, so the entry has
	// to be on disk and not just in the page cache.
	if err := l.f.Sync(); err != nil {
		return nil, err
	}

	l.chain.advance(next)

	return next, nil
}

func (l *fileLog) Entries() ([]*Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return readEntries(l.path)
}

func (l *fileLog) Verify() error {
	entries, err := l.Entries()
	if err != nil {
		return err
	}

	return Verify(entries)
}

func (l *fileLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.f.Close()
}

func readEntries(path string) ([]*Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}
	defer f.Close()

	entries := make([]*Entry, 0)

	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}

		var e *Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("%w: entry %d is unreadable: %w", ErrBrokenChain, len(entries)+1, err)
		}

		entries = append(entries, e)
	}

	if err := s.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
