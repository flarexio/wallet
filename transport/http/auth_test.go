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
