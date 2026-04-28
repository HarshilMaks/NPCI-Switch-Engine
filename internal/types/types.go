package types

import "time"

type PaymentCreateRequest struct {
	PayerVPA  string `json:"payer_vpa"`
	PayeeVPA  string `json:"payee_vpa"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	ClientRef string `json:"client_ref,omitempty"`
}

type ConfirmPaymentRequest struct {
	AuthCode string `json:"auth_code"`
}

type CancelPaymentRequest struct {
	Reason string `json:"reason"`
}

type ManualReversalRequest struct {
	OriginalTransactionID string `json:"original_transaction_id"`
	Reason                string `json:"reason"`
}

type PaymentResponse struct {
	TransactionID string    `json:"transaction_id"`
	Status        string    `json:"status"`
	AcceptedAt    time.Time `json:"accepted_at"`
}

type PaymentStatusResponse struct {
	TransactionID string            `json:"transaction_id"`
	Status        string            `json:"status"`
	Amount        string            `json:"amount"`
	Currency      string            `json:"currency"`
	Events        []map[string]any  `json:"events"`
}

type ReversalRequest struct {
	OriginalTransactionID string `json:"original_transaction_id"`
	Reason                string `json:"reason"`
}

type ReconciliationRunResponse struct {
	RunID   string         `json:"run_id"`
	RunKey  string         `json:"run_key"`
	Status  string         `json:"status"`
	Summary map[string]any `json:"summary"`
}

type ErrorEnvelope struct {
	Code          string         `json:"code"`
	Message       string         `json:"message"`
	Details       map[string]any `json:"details,omitempty"`
	CorrelationID string         `json:"correlation_id"`
}

type LedgerEntry struct {
	EntryID       string    `json:"entry_id"`
	TransactionID string    `json:"transaction_id"`
	Account       string    `json:"account"`
	LegType       string    `json:"leg_type"`
	Amount        string    `json:"amount"`
	PostedAt      time.Time `json:"posted_at"`
}

type LedgerResponse struct {
	AccountID string         `json:"account_id"`
	Entries   []LedgerEntry  `json:"entries"`
	Total     string         `json:"total"`
}
