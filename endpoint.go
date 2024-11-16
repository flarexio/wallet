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

type SignMessageRequest struct {
	Subject string `json:"-"`
	Message []byte `json:"message"`
}

func SignMessageEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*SignMessageRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		return svc.SignMessage(req.Subject, req.Message)
	}
}

type InitializeSignatureRequest struct {
	Subject         string `json:"-"`
	UserID          string `json:"user_id"`
	TransactionID   string `json:"transaction_id"`
	TransactionData []byte `json:"transaction_data"`
}

func InitializeSignatureEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*InitializeSignatureRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		opts, mediation, err := svc.InitializeSignature(req)
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

type FinalizeSignatureResponse struct {
	Token     string           `json:"token"`
	Signature solana.Signature `json:"sig"`
}

func FinalizeSignatureEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*protocol.ParsedCredentialAssertionData)
		if !ok {
			return nil, errors.New("invalid type")
		}

		token, sig, err := svc.FinalizeSignature(req)
		if err != nil {
			return nil, err
		}

		resp := &FinalizeSignatureResponse{
			Token:     token,
			Signature: sig,
		}

		return resp, nil
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

type CreateSessionRequest struct {
	Data []byte `json:"data"`
}

type CreateSessionResponse struct {
	Session string
	Data    <-chan []byte
}

func CreateSessionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*CreateSessionRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		session, ch, err := svc.CreateSession(ctx, req.Data)
		if err != nil {
			return nil, err
		}

		resp := &CreateSessionResponse{
			Session: session,
			Data:    ch,
		}

		return resp, nil
	}
}

type SessionDataResponse struct {
	Data []byte `json:"data"`
}

func SessionDataEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := request.(string)
		if !ok {
			return nil, errors.New("invalid request")
		}

		data, err := svc.SessionData(ctx, session)
		if err != nil {
			return nil, err
		}

		return &SessionDataResponse{Data: data}, nil
	}
}

type AckSessionRequest struct {
	Session string `json:"-"`
	Data    []byte `json:"data"`
}

func AckSessionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*AckSessionRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		return nil, svc.AckSession(ctx, req.Session, req.Data)
	}
}
