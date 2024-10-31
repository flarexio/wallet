package wallet

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
)

type WalletRequest struct {
	Subject string `json:"subject"`
}

func WalletEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(WalletRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		return svc.Wallet(req.Subject)
	}
}
