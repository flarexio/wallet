package wallet

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newSessionService() *service {
	seed := make([]byte, ed25519.SeedSize)

	return &service{
		privkey:  ed25519.NewKeyFromSeed(seed),
		sessions: make(map[string][]*Session),
	}
}

// A stream that has gone away must not wedge the service: AckSession used to
// block sending on an unbuffered channel while holding svc's mutex, which
// froze every other session operation until restart.
func TestAckSessionDoesNotBlockOtherSessions(t *testing.T) {
	assert := assert.New(t)

	svc := newSessionService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, _, err := svc.CreateSession(ctx, []byte("payload"))
	if !assert.NoError(err) {
		return
	}

	acked := make(chan error, 1)
	go func() {
		acked <- svc.AckSession(context.Background(), session, []byte("data"))
	}()

	select {
	case err := <-acked:
		assert.NoError(err)
	case <-time.After(2 * time.Second):
		assert.Fail("AckSession blocked with no reader on the session channel")
		return
	}

	other, _, err := svc.CreateSession(context.Background(), []byte("unrelated"))
	if !assert.NoError(err) {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.SessionData(context.Background(), other)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		assert.Fail("session mutex still held after an unread ack")
	}
}

// The ack path and the timeout path both terminate the channel, and used to be
// able to interleave into a "send on closed channel" panic.
func TestAckSessionRacingTimeout(t *testing.T) {
	assert := assert.New(t)

	svc := newSessionService()

	for i := range 200 {
		ctx, cancel := context.WithCancel(context.Background())

		data := []byte{byte(i), byte(i >> 8)}

		session, ch, err := svc.CreateSession(ctx, data)
		if !assert.NoError(err) {
			cancel()
			return
		}

		go svc.AckSession(context.Background(), session, []byte("data"))
		go cancel()

		<-ch // drains whichever outcome wins, then observes the close
	}
}

func TestAckSessionDeliversToStream(t *testing.T) {
	assert := assert.New(t)

	svc := newSessionService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, ch, err := svc.CreateSession(ctx, []byte("payload"))
	if !assert.NoError(err) {
		return
	}

	data, err := svc.SessionData(ctx, session)
	assert.NoError(err)
	assert.Equal([]byte("payload"), data)

	assert.NoError(svc.AckSession(ctx, session, []byte("result")))

	assert.Equal([]byte("result"), <-ch)

	_, ok := <-ch
	assert.False(ok, "channel should be closed after delivery")

	// A second ack must never deliver again. Whether it reports "closed" or
	// "not found" depends on whether unindexing has caught up yet.
	err = svc.AckSession(ctx, session, []byte("again"))
	assert.Error(err)
	assert.True(err == ErrSessionClosed || err == ErrSessionNotFound,
		"unexpected error for repeated ack: %v", err)
}

// Session ids index the map by their first two characters; anything shorter
// used to panic with a slice-bounds error on an unauthenticated route.
func TestShortSessionIDIsNotFound(t *testing.T) {
	assert := assert.New(t)

	svc := newSessionService()

	for _, id := range []string{"", "x"} {
		_, err := svc.SessionData(context.Background(), id)
		assert.ErrorIs(err, ErrSessionNotFound)

		err = svc.AckSession(context.Background(), id, []byte("data"))
		assert.ErrorIs(err, ErrSessionNotFound)
	}
}

func TestCancelledSessionClosesStream(t *testing.T) {
	assert := assert.New(t)

	svc := newSessionService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, ch, err := svc.CreateSession(ctx, []byte("payload"))
	if !assert.NoError(err) {
		return
	}

	cancel()

	data, ok := <-ch
	assert.False(ok, "cancelled session should close without delivering")
	assert.Nil(data)

	assert.Eventually(func() bool {
		_, err := svc.SessionData(context.Background(), session)
		return err == ErrSessionNotFound
	}, 2*time.Second, 10*time.Millisecond, "cancelled session should be unindexed")
}

// Session ids are deterministic Ed25519 signatures, so the same payload always
// collides. Documents the current contract rather than endorsing it.
func TestDuplicatePayloadIsRejected(t *testing.T) {
	assert := assert.New(t)

	svc := newSessionService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first, _, err := svc.CreateSession(ctx, []byte("same"))
	if !assert.NoError(err) {
		return
	}

	_, _, err = svc.CreateSession(ctx, []byte("same"))
	assert.ErrorIs(err, ErrSessionExists)

	// The rejected attempt must not have disturbed the live session.
	data, err := svc.SessionData(ctx, first)
	assert.NoError(err)
	assert.Equal([]byte("same"), data)
}
