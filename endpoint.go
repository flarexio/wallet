package wallet

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
)

func WalletEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		sub, ok := request.(string)
		if !ok {
			return nil, errors.New("invalid request")
		}

		return svc.Wallet(sub)
	}
}
