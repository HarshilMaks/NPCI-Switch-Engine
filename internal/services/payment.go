package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"npci-upi/internal/state"
	"npci-upi/internal/types"
)

const idempotencyScopePayments = "payments:create"

type PaymentService struct {
	DB *sql.DB
}

func NewPaymentService(db *sql.DB) *PaymentService {
	return &PaymentService{DB: db}
}

type accountRow struct {
	ID       string
	Status   string
	Balance  decimal.Decimal
	Currency string
}

type transactionRow struct {
	ID            string
	PayerVPA      string
	PayeeVPA      string
	Amount        decimal.Decimal
	Currency      string
	Status        string
	Version       int
	Idempotency   string
	CreatedAtRFC  string
	UpdatedAtRFC  string
}

type idempotencyRow struct {
	RequestHash    string
	ResponseJSON   string
	StatusCode     int
}

func (s *PaymentService) CreatePayment(
	ctx context.Context,
	req types.PaymentCreateRequest,
	idempotencyKey string,
	correlationID string,
) (int, types.PaymentResponse, error) {
	if err := validatePaymentRequest(req); err != nil {
		return 0, types.PaymentResponse{}, NewAppError(400, "INVALID_REQUEST", err.Error())
	}
	requestHash, err := hashRequest(req)
	if err != nil {
		return 0, types.PaymentResponse{}, NewAppError(500, "HASH_ERROR", "failed to hash request")
	}

	if existing, err := s.getIdempotency(ctx, idempotencyKey); err != nil {
		return 0, types.PaymentResponse{}, NewAppError(500, "IDEMPOTENCY_READ_ERROR", err.Error())
	} else if existing != nil {
		if existing.RequestHash != requestHash {
			return 0, types.PaymentResponse{}, NewAppError(409, "IDEMPOTENCY_CONFLICT", "idempotency key reused")
		}
		var cached types.PaymentResponse
		if err := json.Unmarshal([]byte(existing.ResponseJSON), &cached); err != nil {
			return 0, types.PaymentResponse{}, NewAppError(500, "IDEMPOTENCY_READ_ERROR", "invalid cached response")
		}
		return existing.StatusCode, cached, nil
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, types.PaymentResponse{}, NewAppError(500, "DB_TX_ERROR", "unable to begin transaction")
	}
	defer tx.Rollback()

	amount, _ := decimal.NewFromString(req.Amount)
	now := time.Now().UTC()
	transactionID := uuid.NewString()
	createdAt := now.Format(time.RFC3339Nano)

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transactions
		 (id, payer_vpa, payee_vpa, amount, currency, status, version, idempotency_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		transactionID, req.PayerVPA, req.PayeeVPA, amount.StringFixed(2), req.Currency, "INITIATED", 1, idempotencyKey, createdAt, createdAt,
	)
	if err != nil {
		return 0, types.PaymentResponse{}, NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	if err := s.insertOutbox(ctx, tx, "transaction", transactionID, "payment.initiated", map[string]any{
		"transaction_id": transactionID,
		"correlation_id": correlationID,
	}); err != nil {
		return 0, types.PaymentResponse{}, NewAppError(500, "OUTBOX_WRITE_ERROR", err.Error())
	}

	row := transactionRow{
		ID:           transactionID,
		PayerVPA:     req.PayerVPA,
		PayeeVPA:     req.PayeeVPA,
		Amount:       amount,
		Currency:     req.Currency,
		Status:       "INITIATED",
		Version:      1,
		Idempotency:  idempotencyKey,
		CreatedAtRFC: createdAt,
		UpdatedAtRFC: createdAt,
	}

	if err := s.processPaymentFlow(ctx, tx, &row, correlationID); err != nil {
		return 0, types.PaymentResponse{}, err
	}

	response := types.PaymentResponse{
		TransactionID: row.ID,
		Status:        row.Status,
		AcceptedAt:    now,
	}

	responseJSON, _ := json.Marshal(response)
	if err := s.insertIdempotency(ctx, tx, idempotencyKey, requestHash, responseJSON, 202); err != nil {
		return 0, types.PaymentResponse{}, NewAppError(500, "IDEMPOTENCY_WRITE_ERROR", err.Error())
	}

	if err := tx.Commit(); err != nil {
		return 0, types.PaymentResponse{}, NewAppError(500, "DB_COMMIT_ERROR", err.Error())
	}
	return 202, response, nil
}

func (s *PaymentService) GetPaymentStatus(ctx context.Context, transactionID string) (*types.PaymentStatusResponse, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, amount, currency, status FROM transactions WHERE id = ?`, transactionID)
	var id, amountStr, currency, status string
	if err := row.Scan(&id, &amountStr, &currency, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewAppError(404, "NOT_FOUND", "transaction not found")
		}
		return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
	}

	eventsRows, err := s.DB.QueryContext(ctx, `SELECT from_status, to_status, reason_code, actor, created_at FROM transaction_events WHERE transaction_id = ? ORDER BY created_at ASC`, transactionID)
	if err != nil {
		return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
	}
	defer eventsRows.Close()

	events := make([]map[string]any, 0)
	for eventsRows.Next() {
		var fromStatus, toStatus, reason, actor, createdAt string
		if err := eventsRows.Scan(&fromStatus, &toStatus, &reason, &actor, &createdAt); err != nil {
			return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
		}
		events = append(events, map[string]any{
			"from_status": fromStatus,
			"to_status":   toStatus,
			"reason_code": reason,
			"actor":       actor,
			"timestamp":   createdAt,
		})
	}

	return &types.PaymentStatusResponse{
		TransactionID: id,
		Status:        status,
		Amount:        amountStr,
		Currency:      currency,
		Events:        events,
	}, nil
}

func (s *PaymentService) ConfirmPayment(ctx context.Context, transactionID, correlationID string) (*types.PaymentStatusResponse, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, NewAppError(500, "DB_TX_ERROR", "unable to begin transaction")
	}
	defer tx.Rollback()

	row, err := s.loadTransaction(ctx, tx, transactionID)
	if err != nil {
		return nil, err
	}
	if isTerminalStatus(row.Status) {
		if err := tx.Commit(); err != nil {
			return nil, NewAppError(500, "DB_COMMIT_ERROR", err.Error())
		}
		return s.GetPaymentStatus(ctx, transactionID)
	}
	if err := s.processPaymentFlow(ctx, tx, row, correlationID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, NewAppError(500, "DB_COMMIT_ERROR", err.Error())
	}
	return s.GetPaymentStatus(ctx, transactionID)
}

func (s *PaymentService) CancelPayment(ctx context.Context, transactionID, correlationID string) (*types.PaymentStatusResponse, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, NewAppError(500, "DB_TX_ERROR", "unable to begin transaction")
	}
	defer tx.Rollback()

	row, err := s.loadTransaction(ctx, tx, transactionID)
	if err != nil {
		return nil, err
	}
	if row.Status != "INITIATED" && row.Status != "AUTH_PENDING" {
		return nil, NewAppError(400, "INVALID_STATE", "only INITIATED/AUTH_PENDING can be canceled")
	}
	if err := s.transition(ctx, tx, row, "FAILED", "CANCELLED_BY_USER", "api", correlationID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, NewAppError(500, "DB_COMMIT_ERROR", err.Error())
	}
	return s.GetPaymentStatus(ctx, transactionID)
}

func (s *PaymentService) ManualReversal(ctx context.Context, req types.ReversalRequest, correlationID string) (map[string]any, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, NewAppError(500, "DB_TX_ERROR", "unable to begin transaction")
	}
	defer tx.Rollback()

	original, err := s.loadTransaction(ctx, tx, req.OriginalTransactionID)
	if err != nil {
		return nil, err
	}
	if original.Status != "COMPLETED" {
		return nil, NewAppError(400, "INVALID_STATE", "manual reversal only allowed for COMPLETED transactions")
	}

	payer, err := s.getAccountByVPA(ctx, tx, original.PayerVPA)
	if err != nil {
		return nil, err
	}
	if payer == nil {
		return nil, NewAppError(400, "PAYER_VPA_NOT_FOUND", "payer vpa not found")
	}
	payee, err := s.getAccountByVPA(ctx, tx, original.PayeeVPA)
	if err != nil {
		return nil, err
	}
	if payee == nil {
		return nil, NewAppError(400, "PAYEE_VPA_NOT_FOUND", "payee vpa not found")
	}
	holding, err := s.getAccountByID(ctx, tx, "SYSTEM_HOLDING_ACCOUNT")
	if err != nil {
		return nil, err
	}
	if payee.Balance.LessThan(original.Amount) {
		return nil, NewAppError(400, "INSUFFICIENT_FUNDS", "payee has insufficient balance for reversal")
	}

	reversalTxID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO transactions
		 (id, payer_vpa, payee_vpa, amount, currency, status, version, idempotency_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		reversalTxID, original.PayeeVPA, original.PayerVPA, original.Amount.StringFixed(2), original.Currency, "REVERSED", 1, fmt.Sprintf("manual-reversal-%s", uuid.NewString()), now, now,
	)
	if err != nil {
		return nil, NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	// Payee -> holding
	if err := s.applyBalanceUpdates(ctx, tx, payee.ID, holding.ID, original.Amount); err != nil {
		return nil, err
	}
	if err := s.insertLedgerPair(ctx, tx, reversalTxID, payee.ID, holding.ID, original.Amount, original.Currency); err != nil {
		return nil, err
	}
	// Holding -> payer
	if err := s.applyBalanceUpdates(ctx, tx, holding.ID, payer.ID, original.Amount); err != nil {
		return nil, err
	}
	if err := s.insertLedgerPair(ctx, tx, reversalTxID, holding.ID, payer.ID, original.Amount, original.Currency); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO reversals (id, original_transaction_id, reversal_transaction_id, reason, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), original.ID, reversalTxID, req.Reason, "REVERSED", now,
	); err != nil {
		return nil, NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	if err := s.insertOutbox(ctx, tx, "transaction", reversalTxID, "payment.reversed", map[string]any{
		"transaction_id":          reversalTxID,
		"original_transaction_id": original.ID,
		"correlation_id":          correlationID,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, NewAppError(500, "DB_COMMIT_ERROR", err.Error())
	}

	return map[string]any{
		"original_transaction_id": original.ID,
		"reversal_transaction_id": reversalTxID,
		"status":                  "REVERSED",
		"created_at":              now,
	}, nil
}

func (s *PaymentService) processPaymentFlow(ctx context.Context, tx *sql.Tx, row *transactionRow, correlationID string) error {
	if row.Status == "INITIATED" {
		if err := s.transition(ctx, tx, row, "AUTH_PENDING", "AUTH_REQUESTED", "orchestrator", correlationID); err != nil {
			return err
		}
	}
	if row.Status == "AUTH_PENDING" {
		if err := s.transition(ctx, tx, row, "AUTHORIZED", "AUTH_APPROVED", "orchestrator", correlationID); err != nil {
			return err
		}
	}
	if row.Status == "AUTHORIZED" {
		payer, err := s.getAccountByVPA(ctx, tx, row.PayerVPA)
		if err != nil {
			return err
		}
		if payer == nil {
			return s.transition(ctx, tx, row, "FAILED", "PAYER_VPA_NOT_FOUND", "orchestrator", correlationID)
		}
		holding, err := s.getAccountByID(ctx, tx, "SYSTEM_HOLDING_ACCOUNT")
		if err != nil {
			return err
		}
		if payer.Balance.LessThan(row.Amount) {
			return s.transition(ctx, tx, row, "FAILED", "INSUFFICIENT_FUNDS", "orchestrator", correlationID)
		}

		if err := s.applyBalanceUpdates(ctx, tx, payer.ID, holding.ID, row.Amount); err != nil {
			return err
		}
		if err := s.insertLedgerPair(ctx, tx, row.ID, payer.ID, holding.ID, row.Amount, row.Currency); err != nil {
			return err
		}
		if err := s.transition(ctx, tx, row, "DEBIT_POSTED", "DEBIT_SUCCESS", "orchestrator", correlationID); err != nil {
			return err
		}
	}

	if row.Status == "DEBIT_POSTED" {
		payer, err := s.getAccountByVPA(ctx, tx, row.PayerVPA)
		if err != nil {
			return err
		}
		holding, err := s.getAccountByID(ctx, tx, "SYSTEM_HOLDING_ACCOUNT")
		if err != nil {
			return err
		}
		payee, err := s.getAccountByVPA(ctx, tx, row.PayeeVPA)
		if err != nil {
			return err
		}
		if payee == nil || payee.Status != "ACTIVE" {
			if err := s.transition(ctx, tx, row, "REVERSAL_PENDING", "BENEFICIARY_UNAVAILABLE", "orchestrator", correlationID); err != nil {
				return err
			}
			if payer == nil {
				return s.transition(ctx, tx, row, "FAILED", "PAYER_VPA_NOT_FOUND", "orchestrator", correlationID)
			}
			return s.autoReversal(ctx, tx, row, payer, holding, correlationID)
		}
		if err := s.applyBalanceUpdates(ctx, tx, holding.ID, payee.ID, row.Amount); err != nil {
			return err
		}
		if err := s.insertLedgerPair(ctx, tx, row.ID, holding.ID, payee.ID, row.Amount, row.Currency); err != nil {
			return err
		}
		if err := s.transition(ctx, tx, row, "CREDIT_POSTED", "CREDIT_SUCCESS", "orchestrator", correlationID); err != nil {
			return err
		}
	}

	if row.Status == "CREDIT_POSTED" {
		return s.transition(ctx, tx, row, "COMPLETED", "PAYMENT_COMPLETE", "orchestrator", correlationID)
	}

	if row.Status == "REVERSAL_PENDING" {
		payer, err := s.getAccountByVPA(ctx, tx, row.PayerVPA)
		if err != nil {
			return err
		}
		holding, err := s.getAccountByID(ctx, tx, "SYSTEM_HOLDING_ACCOUNT")
		if err != nil {
			return err
		}
		if payer == nil {
			return s.transition(ctx, tx, row, "REVERSAL_FAILED", "PAYER_VPA_NOT_FOUND", "orchestrator", correlationID)
		}
		return s.autoReversal(ctx, tx, row, payer, holding, correlationID)
	}

	return nil
}

func (s *PaymentService) autoReversal(ctx context.Context, tx *sql.Tx, original *transactionRow, payer *accountRow, holding *accountRow, correlationID string) error {
	reversalTxID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO transactions
		 (id, payer_vpa, payee_vpa, amount, currency, status, version, idempotency_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		reversalTxID, original.PayeeVPA, original.PayerVPA, original.Amount.StringFixed(2), original.Currency, "REVERSED", 1, fmt.Sprintf("auto-reversal-%s", original.ID), now, now,
	)
	if err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	if err := s.applyBalanceUpdates(ctx, tx, holding.ID, payer.ID, original.Amount); err != nil {
		return err
	}
	if err := s.insertLedgerPair(ctx, tx, reversalTxID, holding.ID, payer.ID, original.Amount, original.Currency); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO reversals (id, original_transaction_id, reversal_transaction_id, reason, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), original.ID, reversalTxID, "AUTO_REVERSAL_AFTER_CREDIT_FAILURE", "REVERSED", now,
	); err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}
	if err := s.transition(ctx, tx, original, "REVERSED", "AUTO_REVERSAL_SUCCESS", "orchestrator", correlationID); err != nil {
		return err
	}
	return s.insertOutbox(ctx, tx, "transaction", original.ID, "payment.reversed", map[string]any{
		"transaction_id":          original.ID,
		"reversal_transaction_id": reversalTxID,
		"correlation_id":          correlationID,
	})
}

func (s *PaymentService) transition(ctx context.Context, tx *sql.Tx, row *transactionRow, toStatus, reason, actor, correlationID string) error {
	fromStatus := row.Status
	if err := state.EnsureTransitionAllowed(fromStatus, toStatus); err != nil {
		return NewAppError(400, "INVALID_STATE", err.Error())
	}
	row.Status = toStatus
	row.Version += 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	row.UpdatedAtRFC = now

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE transactions SET status = ?, version = ?, updated_at = ? WHERE id = ?`,
		row.Status, row.Version, row.UpdatedAtRFC, row.ID,
	); err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	meta := map[string]any{"correlation_id": correlationID}
	metaJSON, _ := json.Marshal(meta)
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO transaction_events (id, transaction_id, from_status, to_status, reason_code, actor, metadata_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), row.ID, fromStatus, toStatus, reason, actor, string(metaJSON), now,
	); err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	return s.insertOutbox(ctx, tx, "transaction", row.ID, fmt.Sprintf("payment.%s", strings.ToLower(toStatus)), map[string]any{
		"transaction_id": row.ID,
		"from_status":    fromStatus,
		"to_status":      toStatus,
		"reason_code":    reason,
		"correlation_id": correlationID,
	})
}

func (s *PaymentService) insertOutbox(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload map[string]any) error {
	payloadJSON, _ := json.Marshal(payload)
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_type, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), aggregateType, aggregateID, eventType, string(payloadJSON), time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return NewAppError(500, "OUTBOX_WRITE_ERROR", err.Error())
	}
	return nil
}

func (s *PaymentService) insertIdempotency(ctx context.Context, tx *sql.Tx, key, requestHash string, response []byte, statusCode int) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO idempotency_records (id, idempotency_key, scope_key, request_hash, response_payload, status_code, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), key, idempotencyScopePayments, requestHash, string(response), statusCode, time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return err
	}
	return nil
}

func (s *PaymentService) getIdempotency(ctx context.Context, key string) (*idempotencyRow, error) {
	row := s.DB.QueryRowContext(ctx,
		`SELECT request_hash, response_payload, status_code FROM idempotency_records WHERE idempotency_key = ? AND scope_key = ?`,
		key, idempotencyScopePayments,
	)
	record := idempotencyRow{}
	if err := row.Scan(&record.RequestHash, &record.ResponseJSON, &record.StatusCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (s *PaymentService) loadTransaction(ctx context.Context, tx *sql.Tx, id string) (*transactionRow, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT id, payer_vpa, payee_vpa, amount, currency, status, version, idempotency_key, created_at, updated_at
		 FROM transactions WHERE id = ?`,
		id,
	)
	var amountStr string
	var tr transactionRow
	if err := row.Scan(&tr.ID, &tr.PayerVPA, &tr.PayeeVPA, &amountStr, &tr.Currency, &tr.Status, &tr.Version, &tr.Idempotency, &tr.CreatedAtRFC, &tr.UpdatedAtRFC); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewAppError(404, "NOT_FOUND", "transaction not found")
		}
		return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
	}
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, NewAppError(500, "AMOUNT_PARSE_ERROR", "invalid amount in transaction")
	}
	tr.Amount = amount
	return &tr, nil
}

func (s *PaymentService) getAccountByID(ctx context.Context, tx *sql.Tx, id string) (*accountRow, error) {
	row := tx.QueryRowContext(ctx, `SELECT id, status, available_balance, currency FROM accounts WHERE id = ?`, id)
	var status, balanceStr, currency string
	var accountID string
	if err := row.Scan(&accountID, &status, &balanceStr, &currency); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewAppError(400, "ACCOUNT_NOT_FOUND", "account not found")
		}
		return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
	}
	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return nil, NewAppError(500, "BALANCE_PARSE_ERROR", "invalid balance value")
	}
	return &accountRow{ID: accountID, Status: status, Balance: balance, Currency: currency}, nil
}

func (s *PaymentService) getAccountByVPA(ctx context.Context, tx *sql.Tx, handle string) (*accountRow, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT a.id, a.status, a.available_balance, a.currency
		 FROM vpas v JOIN accounts a ON v.account_id = a.id
		 WHERE v.handle = ? AND v.status = 'ACTIVE'`,
		handle,
	)
	var accountID, status, balanceStr, currency string
	if err := row.Scan(&accountID, &status, &balanceStr, &currency); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
	}
	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return nil, NewAppError(500, "BALANCE_PARSE_ERROR", "invalid balance value")
	}
	return &accountRow{ID: accountID, Status: status, Balance: balance, Currency: currency}, nil
}

func (s *PaymentService) applyBalanceUpdates(ctx context.Context, tx *sql.Tx, debitAccountID, creditAccountID string, amount decimal.Decimal) error {
	debitAccount, err := s.getAccountByID(ctx, tx, debitAccountID)
	if err != nil {
		return err
	}
	creditAccount, err := s.getAccountByID(ctx, tx, creditAccountID)
	if err != nil {
		return err
	}
	if debitAccount.Balance.LessThan(amount) {
		return NewAppError(400, "INSUFFICIENT_FUNDS", "insufficient balance for debit")
	}
	newDebit := debitAccount.Balance.Sub(amount)
	newCredit := creditAccount.Balance.Add(amount)

	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET available_balance = ? WHERE id = ?`, newDebit.StringFixed(2), debitAccountID); err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}
	if _, err := tx.ExecContext(ctx, `UPDATE accounts SET available_balance = ? WHERE id = ?`, newCredit.StringFixed(2), creditAccountID); err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}
	return nil
}

func (s *PaymentService) insertLedgerPair(ctx context.Context, tx *sql.Tx, transactionID, debitAccountID, creditAccountID string, amount decimal.Decimal, currency string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO ledger_entries (id, transaction_id, account_id, leg_type, amount, currency, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), transactionID, debitAccountID, "DEBIT", amount.StringFixed(2), currency, now,
	); err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO ledger_entries (id, transaction_id, account_id, leg_type, amount, currency, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), transactionID, creditAccountID, "CREDIT", amount.StringFixed(2), currency, now,
	); err != nil {
		return NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}
	return nil
}

func hashRequest(req types.PaymentCreateRequest) (string, error) {
	raw, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func validatePaymentRequest(req types.PaymentCreateRequest) error {
	if req.PayerVPA == "" || req.PayeeVPA == "" {
		return errors.New("payer_vpa and payee_vpa are required")
	}
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("amount must be a positive decimal string")
	}
	if req.Currency == "" {
		return errors.New("currency is required")
	}
	return nil
}

func isTerminalStatus(status string) bool {
	return status == "COMPLETED" || status == "FAILED" || status == "REVERSED" || status == "REVERSAL_FAILED"
}
