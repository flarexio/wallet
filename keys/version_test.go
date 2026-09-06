package keys

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"cloud.google.com/go/kms/apiv1/kmspb"
)

const keyPath = "projects/p/locations/l/keyRings/r/cryptoKeys/k/cryptoKeyVersions/"

func versionedService(vers ...int) *googleKeyService {
	versions := make([]*kmspb.CryptoKeyVersion, 0, len(vers))
	for _, v := range vers {
		versions = append(versions, &kmspb.CryptoKeyVersion{Name: keyPath + strconv.Itoa(v)})
	}

	return &googleKeyService{keyVersions: versions}
}

// Accounts persist the KMS version number, so lookup goes by that number.
func TestKeyVersionResolution(t *testing.T) {
	assert := assert.New(t)

	svc := versionedService(1, 2, 3)

	testCases := []struct {
		name string
		ask  []int
		want int
		err  bool
	}{
		{"latest when unspecified", nil, 3, false},
		{"first version", []int{1}, 1, false},
		{"last version", []int{3}, 3, false},
		{"zero is not a version", []int{0}, 0, true},
		{"negative", []int{-1}, 0, true},
		{"past the end", []int{4}, 0, true},
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

			assert.Equal(tc.want, versionOf(got.Name))
		})
	}
}

// A version that is missing from the list, or a list that is not in version
// order, must not silently resolve to the wrong key: an account derived from
// the wrong KMS version produces the wrong wallet address.
func TestKeyVersionIsNotAListIndex(t *testing.T) {
	assert := assert.New(t)

	t.Run("gap in the list", func(t *testing.T) {
		svc := versionedService(1, 3, 4)

		got, err := svc.keyVersion(3)
		if assert.NoError(err) {
			assert.Equal(3, versionOf(got.Name))
		}

		_, err = svc.keyVersion(2)
		assert.Error(err, "version 2 is gone, not position 2")
	})

	t.Run("list out of order", func(t *testing.T) {
		svc := versionedService(3, 1, 2)

		got, err := svc.keyVersion(1)
		if assert.NoError(err) {
			assert.Equal(1, versionOf(got.Name))
		}

		latest, err := svc.keyVersion()
		if assert.NoError(err) {
			assert.Equal(3, versionOf(latest.Name), "latest is the highest version, not the last entry")
		}
	})
}

func TestKeyVersionEmpty(t *testing.T) {
	assert := assert.New(t)

	_, err := versionedService().keyVersion()
	assert.EqualError(err, "key empty")
}

// keyVersions is read from every request goroutine.
func TestKeyVersionConcurrentReads(t *testing.T) {
	assert := assert.New(t)

	svc := versionedService(1, 2, 3)

	done := make(chan int, 50)
	for range cap(done) {
		go func() {
			v, err := svc.keyVersion()
			if err != nil {
				done <- 0
				return
			}
			done <- versionOf(v.Name)
		}()
	}

	for range cap(done) {
		assert.Equal(3, <-done)
	}
}
