package wallets

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-webauthn/webauthn/protocol"
)

func StartRegistrationHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req *StartRegistrationRequest
		if err := c.ShouldBind(&req); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.String(http.StatusExpectationFailed, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func FinishRegistrationHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req *protocol.ParsedCredentialCreationData
		if err := c.ShouldBind(&req); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.String(http.StatusExpectationFailed, err.Error())
			c.Error(err)
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}
