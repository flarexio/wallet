package http

import (
	"errors"
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
		// Rules are wired up as literals in main.go, so a malformed one is a
		// programming error: fail at registration with a message that says so
		// rather than an index-out-of-range from the split.
		domain, action, ok := strings.Cut(rule, ".")
		if !ok || domain == "" || action == "" || strings.Contains(action, ".") {
			panic(`invalid authorization rule "` + rule + `": want "domain.action"`)
		}

		var flags byte
		for _, w := range who {
			flags = flags | byte(w)
		}

		// who_flags of 0 leaves rbac.rego's authorized_users set empty, and its
		// "count(authorized_users) == 0" clause then allows any token carrying a
		// permitted role through with no ownership check at all. No wallet route
		// wants that, so refuse to wire one up rather than silently downgrade.
		if flags == 0 {
			panic(`authorization rule "` + rule + `" needs at least one Who: who_flags 0 skips the ownership check`)
		}

		return func(c *gin.Context) {
			var claims Claims
			if err := ParseToken(c, &claims); err != nil {
				unauthorized(c, http.StatusUnauthorized, err)
				return
			}

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
