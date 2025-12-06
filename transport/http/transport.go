package http

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-webauthn/webauthn/protocol"

	"github.com/flarexio/wallet"
)

func HealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func WalletHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			err := errors.New("user is required")
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, username)
		if err != nil {
			c.Abort()
			c.Error(err)
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
			err := errors.New("user is required")
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		var req *wallet.SignMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func InitializeSignMessageHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			err := errors.New("user is required")
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		var req *wallet.InitializeSignMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func FinalizeSignMessageHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := protocol.ParseCredentialRequestResponse(c.Request)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
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
			err := errors.New("user is required")
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		var req *wallet.SignTransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func InitializeSignTransactionHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			err := errors.New("user is required")
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		var req *wallet.InitializeSignTransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Subject = username

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func FinalizeSignTransactionHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, err := protocol.ParseCredentialRequestResponse(c.Request)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func CreateSessionHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req *wallet.CreateSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		result, ok := resp.(*wallet.CreateSessionResponse)
		if !ok {
			err := errors.New("invalid type")
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		var codeSent bool
		c.Stream(func(w io.Writer) bool {
			for {
				select {
				case <-ctx.Done():
					return false

				default:
					if !codeSent {
						c.SSEvent("session", result.Session)
						codeSent = true
						return true
					}

					data, ok := <-result.Data
					if !ok {
						c.SSEvent("fail", "session closed")
						return false
					}

					if data == nil {
						c.SSEvent("fail", "timeout")
						return false
					}

					based := base64.StdEncoding.EncodeToString(data)

					c.SSEvent("data", based)
					return false
				}
			}
		})
	}
}

func SessionDataHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := c.Param("session")
		if session == "" {
			err := errors.New("session is required")
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		ctx := c.Request.Context()
		resp, err := endpoint(ctx, session)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}

func AckSessionHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := c.Param("session")
		if session == "" {
			err := errors.New("session is required")
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		var req *wallet.AckSessionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		req.Session = session

		ctx := c.Request.Context()
		_, err := endpoint(ctx, req)
		if err != nil {
			c.Abort()
			c.Error(err)
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.String(http.StatusOK, "ok")
	}
}
