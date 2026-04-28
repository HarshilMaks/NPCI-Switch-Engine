package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ReconciliationService struct {
	DB *sql.DB
}

func NewReconciliationService(db *sql.DB) *ReconciliationService {
	return &ReconciliationService{DB: db}
}

func (s *ReconciliationService) Run(ctx context.Context) (map[string]any, error) {
	runKey := time.Now().UTC().Format("recon-20060102150405")
	runID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, NewAppError(500, "DB_TX_ERROR", "unable to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO reconciliation_runs (id, run_key, status, summary_json, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		runID, runKey, "RUNNING", "{}", now, now,
	); err != nil {
		return nil, NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	diffCount := 0
	staleCount := 0
	mismatchCount := 0
	inspectedCount := 0

	txRows, err := tx.QueryContext(ctx, `SELECT id, status, updated_at FROM transactions`)
	if err != nil {
		return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
	}
	defer txRows.Close()

	for txRows.Next() {
		var txID, status, updatedAtStr string
		if err := txRows.Scan(&txID, &status, &updatedAtStr); err != nil {
			return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
		}
		inspectedCount++

		// Check for balanced ledger
		debitsRow := tx.QueryRowContext(ctx,
			`SELECT COUNT(*), SUM(amount) FROM ledger_entries WHERE transaction_id = ? AND leg_type = 'DEBIT'`,
			txID,
		)
		creditsRow := tx.QueryRowContext(ctx,
			`SELECT COUNT(*), SUM(amount) FROM ledger_entries WHERE transaction_id = ? AND leg_type = 'CREDIT'`,
			txID,
		)

		var debitCount int
		var debitSum sql.NullString
		if err := debitsRow.Scan(&debitCount, &debitSum); err != nil {
			return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
		}

		var creditCount int
		var creditSum sql.NullString
		if err := creditsRow.Scan(&creditCount, &creditSum); err != nil {
			return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
		}

		if status == "COMPLETED" || status == "REVERSED" {
			if debitCount == 0 || creditCount == 0 {
				details, _ := json.Marshal(map[string]any{
					"has_debit":  debitCount > 0,
					"has_credit": creditCount > 0,
				})
				if _, err := tx.ExecContext(
					ctx,
					`INSERT INTO reconciliation_diffs (id, run_id, transaction_id, diff_type, details_json, created_at)
					 VALUES (?, ?, ?, ?, ?, ?)`,
					uuid.NewString(), runID, txID, "missing_leg", string(details), now,
				); err != nil {
					return nil, NewAppError(500, "DB_WRITE_ERROR", err.Error())
				}
				diffCount++
			}
		}

		if debitSum.Valid && creditSum.Valid && debitSum.String != creditSum.String {
			details, _ := json.Marshal(map[string]any{
				"debit_total":  debitSum.String,
				"credit_total": creditSum.String,
			})
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO reconciliation_diffs (id, run_id, transaction_id, diff_type, details_json, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)`,
				uuid.NewString(), runID, txID, "amount_mismatch", string(details), now,
			); err != nil {
				return nil, NewAppError(500, "DB_WRITE_ERROR", err.Error())
			}
			diffCount++
			mismatchCount++
		}

		isTerminal := status == "COMPLETED" || status == "FAILED" || status == "REVERSED" || status == "REVERSAL_FAILED"
		if !isTerminal {
			updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)
			ageThreshold := time.Now().UTC().Add(-10 * time.Minute)
			if updatedAt.Before(ageThreshold) {
				details, _ := json.Marshal(map[string]any{
					"status":     status,
					"updated_at": updatedAtStr,
				})
				if _, err := tx.ExecContext(
					ctx,
					`INSERT INTO reconciliation_diffs (id, run_id, transaction_id, diff_type, details_json, created_at)
					 VALUES (?, ?, ?, ?, ?, ?)`,
					uuid.NewString(), runID, txID, "stale_pending", string(details), now,
				); err != nil {
					return nil, NewAppError(500, "DB_WRITE_ERROR", err.Error())
				}
				diffCount++
				staleCount++
			}
		}
	}

	var unpublishedCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE published_at IS NULL`).Scan(&unpublishedCount); err != nil {
		return nil, NewAppError(500, "DB_READ_ERROR", err.Error())
	}

	summary := map[string]any{
		"inspected_transactions":     inspectedCount,
		"diff_count":                 diffCount,
		"amount_mismatch_count":      mismatchCount,
		"stale_pending_count":        staleCount,
		"unpublished_outbox_events":  unpublishedCount,
	}
	summaryJSON, _ := json.Marshal(summary)

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE reconciliation_runs SET status = ?, summary_json = ?, completed_at = ? WHERE id = ?`,
		"COMPLETED", string(summaryJSON), now, runID,
	); err != nil {
		return nil, NewAppError(500, "DB_WRITE_ERROR", err.Error())
	}

	if err := tx.Commit(); err != nil {
		return nil, NewAppError(500, "DB_COMMIT_ERROR", err.Error())
	}

	return map[string]any{
		"run_id":   runID,
		"run_key":  runKey,
		"status":   "COMPLETED",
		"summary":  summary,
	}, nil
}
