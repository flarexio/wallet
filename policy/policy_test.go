package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/flarexio/core/policy"
)

// Mirrors transport/http.Owner.
const ownerFlag = byte(1)

func claims(sub string, roles ...string) map[string]any {
	return map[string]any{
		"sub":   sub,
		"roles": roles,
	}
}

// data.json has to stay in step with the rule strings wired up in main.go.
func TestAccountPermissions(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	p, err := policy.NewRegoPolicy(ctx, "data.json")
	if !assert.NoError(err) {
		return
	}

	testCases := []struct {
		name    string
		action  string
		object  string
		roles   []string
		allowed bool
	}{
		{"owner reads own wallet", "get", "alice", []string{"user"}, true},
		{"owner signs message", "sign_message", "alice", []string{"user"}, true},
		{"owner signs transaction", "sign_transaction", "alice", []string{"user"}, true},

		{"non-owner reads", "get", "bob", []string{"user"}, false},
		{"non-owner signs message", "sign_message", "bob", []string{"user"}, false},
		{"non-owner signs transaction", "sign_transaction", "bob", []string{"user"}, false},

		{"unknown action", "transfer", "alice", []string{"user"}, false},
		{"unknown role", "sign_transaction", "alice", []string{"guest"}, false},
		{"no roles", "get", "alice", nil, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]any{
				"domain":    "wallet::accounts",
				"action":    tc.action,
				"who_flags": ownerFlag,
				"object":    tc.object,
				"claims":    claims("alice", tc.roles...),
			}

			allowed, err := p.Eval(ctx, input)
			assert.NoError(err)
			assert.Equal(tc.allowed, allowed)
		})
	}
}

// A role scoped to reads alone cannot sign.
func TestReadOnlyRoleCannotSign(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	p, err := policy.NewRegoPolicy(ctx, "testdata/readonly.json")
	if !assert.NoError(err) {
		return
	}

	eval := func(action string) bool {
		allowed, err := p.Eval(ctx, map[string]any{
			"domain":    "wallet::accounts",
			"action":    action,
			"who_flags": ownerFlag,
			"object":    "alice",
			"claims":    claims("alice", "reader"),
		})
		assert.NoError(err)
		return allowed
	}

	assert.True(eval("get"))
	assert.False(eval("sign_message"))
	assert.False(eval("sign_transaction"))
}

// rbac.rego lives in flarexio/core and allows any role-bearing token when
// who_flags is 0, so JWTAuthorizator refuses to wire a route up that way. If
// this starts failing, core changed the rule and that guard can be revisited.
func TestZeroWhoFlagsSkipsOwnership(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()

	p, err := policy.NewRegoPolicy(ctx, "data.json")
	if !assert.NoError(err) {
		return
	}

	input := func(whoFlags byte) map[string]any {
		return map[string]any{
			"domain":    "wallet::accounts",
			"action":    "sign_transaction",
			"who_flags": whoFlags,
			"object":    "alice",
			"claims":    claims("bob", "user"),
		}
	}

	allowed, err := p.Eval(ctx, input(ownerFlag))
	assert.NoError(err)
	assert.False(allowed, "bob must not sign for alice when ownership is checked")

	allowed, err = p.Eval(ctx, input(0))
	assert.NoError(err)
	assert.True(allowed, "who_flags 0 bypasses the ownership check -- this is why JWTAuthorizator panics on it")
}
