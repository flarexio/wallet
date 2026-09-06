package wallet

import (
	"container/list"
	"crypto/ed25519"
	"sync"
	"time"
)

const (
	keyCacheTTL     = 5 * time.Minute
	keyCacheEntries = 4096
)

// keyCache holds derived account keys so that a burst of signing does not
// re-derive from KMS on every request. Entries are not zeroed on eviction:
// a caller may still be holding the slice, and zeroing under it would corrupt
// a signature in flight.
type keyCache struct {
	ttl     time.Duration
	max     int
	entries map[string]*list.Element
	order   *list.List
	now     func() time.Time
	mu      sync.Mutex
}

type cachedKey struct {
	subject string
	privkey ed25519.PrivateKey
	expires time.Time
}

func newKeyCache(ttl time.Duration, max int) *keyCache {
	return &keyCache{
		ttl:     ttl,
		max:     max,
		entries: make(map[string]*list.Element),
		order:   list.New(),
		now:     time.Now,
	}
}

func (c *keyCache) get(subject string) (ed25519.PrivateKey, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.entries[subject]
	if !ok {
		return nil, false
	}

	entry, ok := el.Value.(*cachedKey)
	if !ok {
		c.drop(el)
		return nil, false
	}

	if !c.now().Before(entry.expires) {
		c.drop(el)
		return nil, false
	}

	c.order.MoveToFront(el)

	return entry.privkey, true
}

func (c *keyCache) put(subject string, privkey ed25519.PrivateKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expires := c.now().Add(c.ttl)

	if el, ok := c.entries[subject]; ok {
		if entry, ok := el.Value.(*cachedKey); ok {
			entry.privkey = privkey
			entry.expires = expires
			c.order.MoveToFront(el)
			return
		}

		c.drop(el)
	}

	c.entries[subject] = c.order.PushFront(&cachedKey{
		subject: subject,
		privkey: privkey,
		expires: expires,
	})

	for c.max > 0 && c.order.Len() > c.max {
		c.drop(c.order.Back())
	}
}

func (c *keyCache) drop(el *list.Element) {
	if el == nil {
		return
	}

	if entry, ok := el.Value.(*cachedKey); ok {
		delete(c.entries, entry.subject)
	}

	c.order.Remove(el)
}

func (c *keyCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.order.Len()
}
