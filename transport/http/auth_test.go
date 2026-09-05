package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Rule strings are wired up as literals in cmd/wallet/main.go. A malformed one
// used to fail with "index out of range [1] with length 1", which says nothing
// about the actual mistake.
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

// who_flags of 0 makes rbac.rego skip the ownership check entirely (see
// TestZeroWhoFlagsSkipsOwnership in the policy package), so wiring a route
// without a Who has to fail at registration rather than at runtime.
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
