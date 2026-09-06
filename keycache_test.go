package wallet

import (
	"crypto/ed25519"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testKey(n byte) ed25519.PrivateKey {
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = n

	return ed25519.NewKeyFromSeed(seed)
}

func TestKeyCacheRoundTrip(t *testing.T) {
	assert := assert.New(t)

	c := newKeyCache(time.Minute, 8)
	key := testKey(1)

	_, ok := c.get("alice")
	assert.False(ok)

	c.put("alice", key)

	got, ok := c.get("alice")
	if assert.True(ok) {
		assert.Equal(key, got)
	}

	_, ok = c.get("bob")
	assert.False(ok, "one subject's key must not answer for another")
}

func TestKeyCacheExpires(t *testing.T) {
	assert := assert.New(t)

	now := time.Now()

	c := newKeyCache(5*time.Minute, 8)
	c.now = func() time.Time { return now }

	c.put("alice", testKey(1))

	now = now.Add(5*time.Minute - time.Nanosecond)
	_, ok := c.get("alice")
	assert.True(ok, "still inside the ttl")

	now = now.Add(time.Nanosecond)
	_, ok = c.get("alice")
	assert.False(ok, "ttl reached")

	assert.Equal(0, c.len(), "an expired entry is dropped, not left behind")
}

func TestKeyCacheEvictsLeastRecentlyUsed(t *testing.T) {
	assert := assert.New(t)

	c := newKeyCache(time.Minute, 3)

	for i := range 3 {
		c.put("user"+strconv.Itoa(i), testKey(byte(i)))
	}

	// Touching user0 makes user1 the coldest.
	_, ok := c.get("user0")
	assert.True(ok)

	c.put("user3", testKey(3))

	assert.Equal(3, c.len())

	_, ok = c.get("user1")
	assert.False(ok, "the least recently used entry is evicted")

	for _, subject := range []string{"user0", "user2", "user3"} {
		_, ok := c.get(subject)
		assert.True(ok, subject+" should still be cached")
	}
}

func TestKeyCacheReplacesInPlace(t *testing.T) {
	assert := assert.New(t)

	c := newKeyCache(time.Minute, 8)

	c.put("alice", testKey(1))
	c.put("alice", testKey(2))

	assert.Equal(1, c.len(), "replacing must not grow the cache")

	got, ok := c.get("alice")
	if assert.True(ok) {
		assert.Equal(testKey(2), got)
	}
}

func TestKeyCacheConcurrentAccess(t *testing.T) {
	c := newKeyCache(time.Minute, 16)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			subject := "user" + strconv.Itoa(i%8)
			c.put(subject, testKey(byte(i)))
			c.get(subject)
			c.len()
		}()
	}

	wg.Wait()
}
