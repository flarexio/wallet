package wallet

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
)

func WalletHandler(endpoint endpoint.Endpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req WalletRequest
		if err := c.ShouldBind(&req); err != nil {
			c.Abort()
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		resp, err := endpoint(c, req)
		if err != nil {
			c.Abort()
			c.String(http.StatusExpectationFailed, err.Error())
			return
		}

		c.JSON(http.StatusOK, &resp)
	}
}
