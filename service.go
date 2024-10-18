package wallets

import "github.com/go-webauthn/webauthn/protocol"

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
	FinalizeRegistration(req *protocol.ParsedCredentialCreationData) (string, error)
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
	return &service{cfg}
}

type service struct {
	cfg Config
}

func (svc *service) StartRegistration(userID string, username string) (*protocol.CredentialCreation, error) {
	return nil, nil
}

func (svc *service) FinalizeRegistration(req *protocol.ParsedCredentialCreationData) (string, error) {
	return "", nil
}
