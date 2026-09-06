package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/flarexio/core/policy"
)

type Who byte

const (
	Owner Who = 1 << iota
	Group
	Others
	Admin
	All
)

type JWTAuth func(rule string, who ...Who) gin.HandlerFunc

func JWTAuthorizator(policy policy.Policy) JWTAuth {
	return func(rule string, who ...Who) gin.HandlerFunc {
		domain, action, ok := strings.Cut(rule, ".")
		if !ok || domain == "" || action == "" || strings.Contains(action, ".") {
			panic(`invalid authorization rule "` + rule + `": want "domain.action"`)
		}

		var flags byte
		for _, w := range who {
			flags = flags | byte(w)
		}

		// rbac.rego skips the ownership check entirely when who_flags is 0.
		if flags == 0 {
			panic(`authorization rule "` + rule + `" needs at least one Who: who_flags 0 skips the ownership check`)
		}

		return func(c *gin.Context) {
			var claims Claims
			if err := ParseToken(c, &claims); err != nil {
				unauthorized(c, http.StatusUnauthorized, err)
				return
			}

			c.Set(claimsKey, &claims)

			input := map[string]any{
				"domain":    domain,
				"action":    action,
				"who_flags": flags,
				"claims":    claims.Map(),
			}

			if username := c.Param("user"); username != "" {
				input["object"] = username
			}

			ctx := c.Request.Context()
			allowed, err := policy.Eval(ctx, input)
			if err != nil {
				unauthorized(c, http.StatusExpectationFailed, err)
				return
			}

			if !allowed {
				err := errors.New("access denied")
				unauthorized(c, http.StatusForbidden, err)
				return
			}

			c.Next()
		}
	}
}

func unauthorized(c *gin.Context, code int, err error) {
	c.Abort()
	c.Header("WWW-Authenticate", "Bearer realm=wallet")
	c.String(code, err.Error())
}

func ParseToken(c *gin.Context, claims jwt.Claims) error {
	bearerToken := c.GetHeader("Authorization")

	tokenStr, ok := strings.CutPrefix(bearerToken, "Bearer ")
	if !ok {
		return errors.New("invalid authorization header format")
	}

	_, err := jwt.ParseWithClaims(tokenStr, claims, KeyFunc,
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
		jwt.WithLeeway(10*time.Second),
	)

	return err
}

// claimsKey is where JWTAuthorizator leaves the verified claims for handlers
// that need more than the policy decision.
const claimsKey = "wallet.claims"

var ErrPasskeyUserMismatch = errors.New("passkey does not belong to this account")

// CheckPasskeyUser rejects a passkey user id that is not the token subject's.
//
// The passkey challenge is built around the id the caller supplies, so without
// this a stolen token can be finalized with the attacker's own passkey and the
// second factor protects nothing.
//
// Tokens issued before identity published the claim carry nothing to check
// against. Those are let through and flagged, until every token in circulation
// has been reissued.
func CheckPasskeyUser(c *gin.Context, userID string) error {
	value, ok := c.Get(claimsKey)
	if !ok {
		return errors.New("claims not found")
	}

	claims, ok := value.(*Claims)
	if !ok {
		return errors.New("claims not found")
	}

	if claims.PasskeyUserID == "" {
		c.Error(fmt.Errorf("no passkey_user_id claim for %q: passkey ownership unchecked", claims.Subject))
		return nil
	}

	if claims.PasskeyUserID != userID {
		return ErrPasskeyUserMismatch
	}

	return nil
}
