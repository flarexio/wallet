package http

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func contextWith(claims *Claims) *gin.Context {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if claims != nil {
		c.Set(claimsKey, claims)
	}

	return c
}

// A stolen token must not be usable with the attacker's own passkey.
func TestCheckPasskeyUserRejectsAnotherUsersPasskey(t *testing.T) {
	assert := assert.New(t)

	c := contextWith(&Claims{PasskeyUserID: "hanko-alice"})

	assert.ErrorIs(CheckPasskeyUser(c, "hanko-mallory"), ErrPasskeyUserMismatch)
	assert.NoError(CheckPasskeyUser(c, "hanko-alice"))
}

// Otherwise a caller opts out of the check by omitting the field.
func TestCheckPasskeyUserRejectsEmptyRequestID(t *testing.T) {
	assert := assert.New(t)

	c := contextWith(&Claims{PasskeyUserID: "hanko-alice"})

	assert.ErrorIs(CheckPasskeyUser(c, ""), ErrPasskeyUserMismatch)
}

// A token predating the claim passes, and the gap is recorded.
func TestCheckPasskeyUserAllowsTokenWithoutTheClaim(t *testing.T) {
	assert := assert.New(t)

	c := contextWith(&Claims{})
	c.Params = gin.Params{{Key: "user", Value: "alice"}}

	assert.NoError(CheckPasskeyUser(c, "hanko-anything"))
	assert.Len(c.Errors, 1, "an unchecked request must leave a trace")
}

// Failing open here would undo the whole check.
func TestCheckPasskeyUserFailsWithoutClaims(t *testing.T) {
	assert := assert.New(t)

	assert.Error(CheckPasskeyUser(contextWith(nil), "hanko-alice"))

	c := contextWith(nil)
	c.Set(claimsKey, "not claims")
	assert.Error(CheckPasskeyUser(c, "hanko-alice"))
}

func TestClaimsCarryThePasskeyUserID(t *testing.T) {
	assert := assert.New(t)

	var claims Claims
	assert.NoError(unmarshalClaims(`{"sub":"alice","roles":["user"],"passkey_user_id":"hanko-abc"}`, &claims))

	assert.Equal("alice", claims.Subject)
	assert.Equal("hanko-abc", claims.PasskeyUserID)

	var legacy Claims
	assert.NoError(unmarshalClaims(`{"sub":"alice","roles":["user"]}`, &legacy))
	assert.Equal("", legacy.PasskeyUserID)
}

func unmarshalClaims(raw string, claims *Claims) error {
	return json.Unmarshal([]byte(raw), claims)
}
