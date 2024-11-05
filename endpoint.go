package wallet

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-webauthn/webauthn/protocol"

	"github.com/flarexio/identity/passkeys"
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

type SignatureRequest struct {
	Subject string `json:"-"`
	Data    []byte `json:"data"`
}

func SignatureEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*SignatureRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		return svc.Signature(req.Subject, req.Data)
	}
}

type SignTransactionRequest struct {
	Subject     string
	Transaction *solana.Transaction
}

func (req *SignTransactionRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Subject     string `json:"-"`
		Transaction []byte `json:"tx"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	transaction, err := solana.TransactionFromBytes(raw.Transaction)
	if err != nil {
		return err
	}

	req.Transaction = transaction

	return nil
}

type SignTransactionResponse struct {
	Transaction *solana.Transaction
}

func (resp *SignTransactionResponse) MarshalJSON() ([]byte, error) {
	bs, err := resp.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}

	out := struct {
		Transaction []byte `json:"tx"`
	}{Transaction: bs}

	return json.Marshal(out)
}

func SignTransactionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*SignTransactionRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		_, err := svc.SignTransaction(req.Subject, req.Transaction)
		if err != nil {
			return nil, err
		}

		resp := &SignTransactionResponse{
			Transaction: req.Transaction,
		}

		return resp, nil
	}
}

type InitializeTransactionRequest struct {
	Subject       string
	UserID        string
	TransactionID string
	Transaction   *solana.Transaction
}

func (req *InitializeTransactionRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Subject       string `json:"-"`
		UserID        string `json:"user_id"`
		TransactionID string `json:"transaction_id"`
		Transaction   []byte `json:"transaction_data"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	req.UserID = raw.UserID
	req.TransactionID = raw.TransactionID

	transaction, err := solana.TransactionFromBytes(raw.Transaction)
	if err != nil {
		return err
	}

	req.Transaction = transaction

	return nil
}

func InitializeTransactionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*InitializeTransactionRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		opts, mediation, err := svc.InitializeTransaction(req)
		if err != nil {
			return nil, err
		}

		resp := &passkeys.InitializeLoginResponse{
			Response:  opts.Response,
			Mediation: mediation,
		}

		return resp, err
	}
}

type FinalizeTransactionResponse struct {
	Token       string
	Transaction *solana.Transaction
}

func (resp *FinalizeTransactionResponse) MarshalJSON() ([]byte, error) {
	bs, err := resp.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}

	out := struct {
		Token       string `json:"token"`
		Transaction []byte `json:"tx"`
	}{
		Token:       resp.Token,
		Transaction: bs,
	}

	return json.Marshal(out)
}

func FinalizeTransactionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*protocol.ParsedCredentialAssertionData)
		if !ok {
			return nil, errors.New("invalid type")
		}

		token, transaction, err := svc.FinalizeTransaction(req)
		if err != nil {
			return nil, err
		}

		resp := &FinalizeTransactionResponse{
			Token:       token,
			Transaction: transaction,
		}

		return resp, nil
	}
}
