package snapshot

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Dir keeps snapshots in a directory. Only a backup if that directory is not
// the disk the wallet runs on.
type Dir struct {
	path string
}

func NewDir(path string) (*Dir, error) {
	if path == "" {
		return nil, errors.New("directory destination needs a path")
	}

	// A snapshot reproduces every account key given KMS.
	if err := os.MkdirAll(path, 0o700); err != nil {
		return nil, err
	}

	return &Dir{path}, nil
}

// Put renames into place, so a crash mid-write leaves nothing that looks
// complete.
func (d *Dir) Put(ctx context.Context, name string, r io.Reader) error {
	f, err := os.CreateTemp(d.path, name+".*.part")
	if err != nil {
		return err
	}

	tmp := f.Name()

	defer func() {
		f.Close()
		os.Remove(tmp)
	}()

	if err := f.Chmod(0o600); err != nil {
		return err
	}

	if _, err := io.Copy(f, r); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(tmp, filepath.Join(d.path, name))
}

func (d *Dir) List(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(d.path)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		names = append(names, entry.Name())
	}

	return names, nil
}

func (d *Dir) Delete(ctx context.Context, name string) error {
	return os.Remove(filepath.Join(d.path, name))
}

func (d *Dir) String() string {
	return "dir:" + d.path
}
