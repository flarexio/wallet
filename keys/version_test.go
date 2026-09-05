package keys

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"cloud.google.com/go/kms/apiv1/kmspb"
)

func versionedService(names ...string) *googleKeyService {
	versions := make([]*kmspb.CryptoKeyVersion, 0, len(names))
	for _, name := range names {
		versions = append(versions, &kmspb.CryptoKeyVersion{Name: name})
	}

	return &googleKeyService{keyVersions: versions}
}

// Versions are 1-indexed against KMS; omitting one means "latest".
func TestKeyVersionResolution(t *testing.T) {
	assert := assert.New(t)

	svc := versionedService("v1", "v2", "v3")

	testCases := []struct {
		name string
		ask  []int
		want string
		err  bool
	}{
		{"latest when unspecified", nil, "v3", false},
		{"first version", []int{1}, "v1", false},
		{"last version", []int{3}, "v3", false},
		{"zero is not a version", []int{0}, "", true},
		{"negative", []int{-1}, "", true},
		{"past the end", []int{4}, "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.keyVersion(tc.ask...)
			if tc.err {
				assert.Error(err)
				return
			}

			if !assert.NoError(err) {
				return
			}

			assert.Equal(tc.want, got.Name)
		})
	}
}

func TestKeyVersionEmpty(t *testing.T) {
	assert := assert.New(t)

	_, err := versionedService().keyVersion()
	assert.EqualError(err, "key empty")
}

// keyVersions is read from every request goroutine, so the whole read has to
// sit under the lock -- half of it used to be outside.
func TestKeyVersionConcurrentReads(t *testing.T) {
	assert := assert.New(t)

	svc := versionedService("v1", "v2", "v3")

	done := make(chan string, 50)
	for range cap(done) {
		go func() {
			v, err := svc.keyVersion()
			if err != nil {
				done <- ""
				return
			}
			done <- v.Name
		}()
	}

	for range cap(done) {
		assert.Equal("v3", <-done)
	}
}
