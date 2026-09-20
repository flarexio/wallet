package snapshot

import (
	"context"
	"errors"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCS keeps snapshots in a Cloud Storage bucket.
//
// The bucket is as fund-bearing as the snapshots in it: give it its own
// service account, keep the wallet's identity write-only on it if you can, and
// turn on object versioning and a retention policy so that losing the wallet
// host cannot also mean losing the copies.
type GCS struct {
	client *storage.Client
	bucket string
	prefix string
}

func NewGCS(ctx context.Context, bucket, prefix string) (*GCS, error) {
	if bucket == "" {
		return nil, errors.New("gcs destination needs a bucket")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return &GCS{client, bucket, prefix}, nil
}

func (g *GCS) Put(ctx context.Context, name string, r io.Reader) error {
	w := g.object(name).NewWriter(ctx)

	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		return err
	}

	// The object only becomes visible when Close succeeds, so a failure here
	// leaves nothing behind that looks like a finished snapshot.
	return w.Close()
}

func (g *GCS) List(ctx context.Context) ([]string, error) {
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{Prefix: g.prefix})

	var names []string
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return names, nil
		}

		if err != nil {
			return nil, err
		}

		names = append(names, strings.TrimPrefix(attrs.Name, g.prefix))
	}
}

func (g *GCS) Delete(ctx context.Context, name string) error {
	return g.object(name).Delete(ctx)
}

func (g *GCS) Close() error {
	return g.client.Close()
}

func (g *GCS) String() string {
	return "gs://" + g.bucket + "/" + g.prefix
}

func (g *GCS) object(name string) *storage.ObjectHandle {
	return g.client.Bucket(g.bucket).Object(g.prefix + name)
}
