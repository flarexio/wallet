package wallets

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-webauthn/webauthn/protocol"
)

type StartRegistrationRequest struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

func StartRegistrationEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*StartRegistrationRequest)
		if !ok {
			return nil, errors.New("invalid type")
		}

		opts, err := svc.StartRegistration(req.UserID, req.Username)
		if err != nil {
			return nil, err
		}

		return opts, err
	}
}

func FinishRegistrationEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*protocol.ParsedCredentialCreationData)
		if !ok {
			return nil, errors.New("invalid type")
		}

		token, err := svc.FinishRegistration(req)
		if err != nil {
			return nil, err
		}

		return token, nil
	}
}

type StartTransactionRequest struct {
	UserID          string `json:"user_id"`
	TransactionID   string `json:"transaction_id"`
	TransactionData any    `json:"transaction_data"`
}
