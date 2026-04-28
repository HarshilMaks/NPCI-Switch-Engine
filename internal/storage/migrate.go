package storage

import "database/sql"

func Migrate(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			currency TEXT NOT NULL,
			status TEXT NOT NULL,
			available_balance TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS vpas (
			id TEXT PRIMARY KEY,
			handle TEXT NOT NULL UNIQUE,
			account_id TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(account_id) REFERENCES accounts(id)
		);`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			payer_vpa TEXT NOT NULL,
			payee_vpa TEXT NOT NULL,
			amount TEXT NOT NULL,
			currency TEXT NOT NULL,
			status TEXT NOT NULL,
			version INTEGER NOT NULL,
			idempotency_key TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS transaction_events (
			id TEXT PRIMARY KEY,
			transaction_id TEXT NOT NULL,
			from_status TEXT NOT NULL,
			to_status TEXT NOT NULL,
			reason_code TEXT NOT NULL,
			actor TEXT NOT NULL,
			metadata_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(transaction_id) REFERENCES transactions(id)
		);`,
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id TEXT PRIMARY KEY,
			transaction_id TEXT NOT NULL,
			account_id TEXT NOT NULL,
			leg_type TEXT NOT NULL,
			amount TEXT NOT NULL,
			currency TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(transaction_id) REFERENCES transactions(id)
		);`,
		`CREATE TABLE IF NOT EXISTS reversals (
			id TEXT PRIMARY KEY,
			original_transaction_id TEXT NOT NULL,
			reversal_transaction_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS idempotency_records (
			id TEXT PRIMARY KEY,
			idempotency_key TEXT NOT NULL,
			scope_key TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			response_payload TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(idempotency_key, scope_key)
		);`,
		`CREATE TABLE IF NOT EXISTS outbox_events (
			id TEXT PRIMARY KEY,
			aggregate_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at TEXT NOT NULL,
			published_at TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS reconciliation_runs (
			id TEXT PRIMARY KEY,
			run_key TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			summary_json TEXT NOT NULL,
			started_at TEXT NOT NULL,
			completed_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS reconciliation_diffs (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			transaction_id TEXT NOT NULL,
			diff_type TEXT NOT NULL,
			details_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(run_id) REFERENCES reconciliation_runs(id)
		);`,
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

