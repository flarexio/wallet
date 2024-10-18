package wallets

import (
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	ID              string     `json:"id"`
	Name            *string    `json:"name,omitempty"`
	PublicKey       string     `json:"public_key"`
	AttestationType string     `json:"attestation_type"`
	AAGUID          uuid.UUID  `json:"aaguid"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	Transports      []string   `json:"transports"`
	BackupEligible  bool       `json:"backup_eligible"`
	BackupState     bool       `json:"backup_state"`
	IsMFA           bool       `json:"is_mfa"`
}

type StartTransactionRequest struct {
	UserID          string `json:"user_id"`
	TransactionID   string `json:"transaction_id"`
	TransactionData any    `json:"transaction_data"`
}
