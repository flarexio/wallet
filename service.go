package wallet

import (
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/go-webauthn/webauthn/protocol"
)

type Service interface {
	PasskeyService
	// WalletService
}

type PasskeyService interface {
	RegistrationService
	LoginService
	// CredentialServie
	// TransactionService
}

type RegistrationService interface {
	InitializeRegistration(userID string, username string) (*protocol.CredentialCreation, error)
	FinalizeRegistration(req *protocol.ParsedCredentialCreationData) (string, error)
}

type LoginService interface {
	InitializeLogin(userID string) (*protocol.CredentialAssertion, string, error)
	FinalizeLogin(req *protocol.ParsedCredentialAssertionData) (string, error)
}

type CredentialServie interface {
	ListCredentials(userID string) ([]*Credential, error)
	UpdateCredential(credentialID string, name string) error
	RemoveCredential(credentialID string) error
}

type TransactionService interface {
	InitializeTransaction(req *InitializeTransactionRequest) (*protocol.CredentialAssertion, string, error)
	FinalizeTransaction(req *protocol.ParsedCredentialAssertionData) (string, error)
}

type WalletService interface {
	Wallet()
	Transaction()
}

func NewService(cfg Config) Service {
	passkeys := cfg.Providers.Passkeys

	client := resty.New().
		SetHeader("Content-Type", "application/json").
		SetHeader("apiKey", passkeys.PasskeyAPI.Secret).
		SetBaseURL(passkeys.BaseURL + "/" + passkeys.TenantID)

	return &service{cfg, client}
}

type service struct {
	cfg    Config
	client *resty.Client
}

func (svc *service) InitializeRegistration(userID string, username string) (*protocol.CredentialCreation, error) {
	params := map[string]string{
		"user_id":  userID,
		"username": username,
	}

	var (
		successResult *protocol.CredentialCreation
		failureResult *FailureResult
	)

	resp, err := svc.client.R().
		SetBody(params).
		SetResult(&successResult).
		SetError(&failureResult).
		Post("/registration/initialize")

	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, failureResult
	}

	return successResult, nil
}

func (svc *service) FinalizeRegistration(req *protocol.ParsedCredentialCreationData) (string, error) {
	var (
		successResult *TokenResult
		failureResult *FailureResult
	)

	resp, err := svc.client.R().
		SetBody(&req.Raw).
		SetResult(&successResult).
		SetError(&failureResult).
		Post("/registration/finalize")

	if err != nil {
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		return "", failureResult
	}

	return successResult.Token, nil
}

func (svc *service) InitializeLogin(userID string) (*protocol.CredentialAssertion, string, error) {
	params := map[string]string{
		"user_id": userID,
	}

	var (
		successResult *protocol.CredentialAssertion
		failureResult *FailureResult
	)

	resp, err := svc.client.R().
		SetBody(params).
		SetResult(&successResult).
		SetError(&failureResult).
		Post("/login/initialize")

	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, "", failureResult
	}

	return successResult, "optional", nil
}

func (svc *service) FinalizeLogin(req *protocol.ParsedCredentialAssertionData) (string, error) {
	var (
		successResult *TokenResult
		failureResult *FailureResult
	)

	resp, err := svc.client.R().
		SetBody(&req.Raw).
		SetResult(&successResult).
		SetError(&failureResult).
		Post("/login/finalize")

	if err != nil {
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		return "", failureResult
	}

	return successResult.Token, nil
}
