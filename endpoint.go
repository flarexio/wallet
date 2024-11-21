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

type InitializeSignMessageRequest struct {
	Subject       string `json:"-"`
	UserID        string `json:"user_id"`
	TransactionID string `json:"transaction_id"`
	Message       []byte `json:"message"`
}

func InitializeSignMessageEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*InitializeSignMessageRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		opts, mediation, err := svc.InitializeSignMessage(req)
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

type FinalizeSignMessageResponse struct {
	Signature solana.Signature `json:"signature"`
}

func FinalizeSignMessageEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*protocol.ParsedCredentialAssertionData)
		if !ok {
			return nil, errors.New("invalid type")
		}

		sig, err := svc.FinalizeSignMessage(req)
		if err != nil {
			return nil, err
		}

		resp := &FinalizeSignMessageResponse{
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
		Transaction []byte `json:"transaction"`
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
	Signatures  []solana.Signature
}

func (resp *SignTransactionResponse) MarshalJSON() ([]byte, error) {
	bs, err := resp.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}

	out := struct {
		Transaction []byte             `json:"transaction"`
		Signatures  []solana.Signature `json:"signatures"`
	}{
		Transaction: bs,
		Signatures:  resp.Signatures,
	}

	return json.Marshal(out)
}

func SignTransactionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*SignTransactionRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		sigs, err := svc.SignTransaction(req.Subject, req.Transaction)
		if err != nil {
			return nil, err
		}

		resp := &SignTransactionResponse{
			Transaction: req.Transaction,
			Signatures:  sigs,
		}

		return resp, nil
	}
}

type InitializeSignTransactionRequest struct {
	Subject       string
	UserID        string
	TransactionID string
	Transaction   *solana.Transaction
	Versioned     bool
}

func (req *InitializeSignTransactionRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Subject       string `json:"-"`
		UserID        string `json:"user_id"`
		TransactionID string `json:"transaction_id"`
		Transaction   []byte `json:"transaction"`
		Versioned     bool   `json:"versioned"`
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
	req.Versioned = raw.Versioned

	return nil
}

func InitializeSignTransactionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*InitializeSignTransactionRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		opts, mediation, err := svc.InitializeSignTransaction(req)
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

type FinalizeSignTransactionResponse struct {
	Transaction *solana.Transaction
	Versioned   bool
	Signatures  []solana.Signature
}

func (resp *FinalizeSignTransactionResponse) MarshalJSON() ([]byte, error) {
	bs, err := resp.Transaction.MarshalBinary()
	if err != nil {
		return nil, err
	}

	out := struct {
		Transaction []byte             `json:"transaction"`
		Versioned   bool               `json:"versioned"`
		Signatures  []solana.Signature `json:"signatures"`
	}{
		Transaction: bs,
		Versioned:   resp.Versioned,
		Signatures:  resp.Signatures,
	}

	return json.Marshal(out)
}

func FinalizeSignTransactionEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(*protocol.ParsedCredentialAssertionData)
		if !ok {
			return nil, errors.New("invalid type")
		}

		transaction, versioned, err := svc.FinalizeSignTransaction(req)
		if err != nil {
			return nil, err
		}

		resp := &FinalizeSignTransactionResponse{
			Transaction: transaction,
			Versioned:   versioned,
			Signatures:  transaction.Signatures,
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
