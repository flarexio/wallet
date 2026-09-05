package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Rules are wired up as literals in main.go, so a malformed one must not start.
func TestJWTAuthorizatorRejectsMalformedRule(t *testing.T) {
	assert := assert.New(t)

	auth := JWTAuthorizator(nil)

	for _, rule := range []string{
		"",
		"wallet::accounts",
		"wallet::accounts_get",
		".get",
		"wallet::accounts.",
		"wallet::accounts.get.extra",
	} {
		t.Run(rule, func(t *testing.T) {
			assert.PanicsWithValue(
				`invalid authorization rule "`+rule+`": want "domain.action"`,
				func() { auth(rule, Owner) },
			)
		})
	}
}

func TestJWTAuthorizatorAcceptsWiredRules(t *testing.T) {
	assert := assert.New(t)

	auth := JWTAuthorizator(nil)

	for _, rule := range []string{
		"wallet::accounts.get",
		"wallet::accounts.sign_message",
		"wallet::accounts.sign_transaction",
	} {
		t.Run(rule, func(t *testing.T) {
			assert.NotPanics(func() { auth(rule, Owner) })
		})
	}
}

// who_flags 0 skips the ownership check; see policy.TestZeroWhoFlagsSkipsOwnership.
func TestJWTAuthorizatorRequiresWho(t *testing.T) {
	assert := assert.New(t)

	auth := JWTAuthorizator(nil)

	const rule = "wallet::accounts.get"
	want := `authorization rule "` + rule + `" needs at least one Who: who_flags 0 skips the ownership check`

	t.Run("no who at all", func(t *testing.T) {
		assert.PanicsWithValue(want, func() { auth(rule) })
	})

	t.Run("explicit zero", func(t *testing.T) {
		assert.PanicsWithValue(want, func() { auth(rule, Who(0)) })
	})

	t.Run("a real who is accepted", func(t *testing.T) {
		assert.NotPanics(func() { auth(rule, Owner) })
	})
}
