package wallets

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
	// LoginService
	// CredentialServie
	// TransactionService
}

type RegistrationService interface {
	StartRegistration(userID string, username string) (*protocol.CredentialCreation, error)
	FinishRegistration(req *protocol.ParsedCredentialCreationData) (string, error)
}

type LoginService interface {
	StartLogin(userID string) (*protocol.CredentialAssertion, string, error)
	FinishLogin(req *protocol.ParsedCredentialAssertionData) (string, error)
}

type CredentialServie interface {
	ListCredentials(userID string) ([]*Credential, error)
	UpdateCredential(credentialID string, name string) error
	RemoveCredential(credentialID string) error
}

type TransactionService interface {
	StartTransaction(req *StartTransactionRequest) (*protocol.CredentialAssertion, string, error)
	FinishTransaction(req *protocol.ParsedCredentialAssertionData) (string, error)
}

type WalletService interface {
	Wallet()
	Transaction()
}

func NewService(cfg Config) Service {
	baseURL := cfg.PasskeysConfig.BaseURL + "/" + cfg.PasskeysConfig.TenantID
	apiSecret := cfg.PasskeysConfig.PasskeyAPI.Secret

	client := resty.New().
		SetHeader("Content-Type", "application/json").
		SetHeader("apiKey", apiSecret).
		SetBaseURL(baseURL)

	return &service{cfg, client}
}

type service struct {
	cfg    Config
	client *resty.Client
}

func (svc *service) StartRegistration(userID string, username string) (*protocol.CredentialCreation, error) {
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

func (svc *service) FinishRegistration(req *protocol.ParsedCredentialCreationData) (string, error) {
	var (
		successResult *TokenResult
		failureResult *FailureResult
	)

	resp, err := svc.client.R().
		SetBody(&req).
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
