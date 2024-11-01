package wallet

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
)

func WalletHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("user")
		if username == "" {
			c.Abort()
			c.String(http.StatusBadRequest, "user is required")
			return
		}

		resp, err := endpoint(c, username)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}
