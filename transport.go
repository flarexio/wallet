package wallet

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-webauthn/webauthn/protocol"
)

func WalletHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			c.Abort()
			c.String(http.StatusBadRequest, "user is required")
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, username)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func SignMessageHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			c.Abort()
			c.String(http.StatusBadRequest, "user is required")
			return
		}

		var req *SignMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func InitializeSignatureHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			c.Abort()
			c.String(http.StatusBadRequest, "user is required")
			return
		}

		var req *InitializeSignatureRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func FinalizeSignatureHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := protocol.ParseCredentialRequestResponse(c.Request)
		if err != nil {
			c.Abort()
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func SignTransactionHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			c.Abort()
			c.String(http.StatusBadRequest, "user is required")
			return
		}

		var req *SignTransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func InitializeTransactionHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			c.Abort()
			c.String(http.StatusBadRequest, "user is required")
			return
		}

		var req *InitializeTransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func FinalizeTransactionHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := protocol.ParseCredentialRequestResponse(c.Request)
		if err != nil {
			c.Abort()
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}
