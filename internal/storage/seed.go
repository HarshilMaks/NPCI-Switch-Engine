package storage

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"npci-upi/internal/config"
)

func Seed(db *sql.DB, cfg config.Settings) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	seedBalance := decimal.RequireFromString(cfg.DefaultSeedBalanceStr).String()

	accounts := []struct {
		ID      string
		UserID  string
		Status  string
		Balance string
	}{
		{cfg.SystemHoldingAccount, "system", "ACTIVE", "0.00"},
		{"alice-account", "alice", "ACTIVE", seedBalance},
		{"bob-account", "bob", "ACTIVE", seedBalance},
		{"inactive-merchant-account", "inactive_merchant", "INACTIVE", "0.00"},
	}

	for _, account := range accounts {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO accounts (id, user_id, currency, status, available_balance, created_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			account.ID, account.UserID, cfg.DefaultCurrency, account.Status, account.Balance, now,
		)
		if err != nil {
			return err
		}
	}

	vpas := []struct {
		Handle    string
		AccountID string
	}{
		{"alice@bank", "alice-account"},
		{"bob@bank", "bob-account"},
		{"inactive@bank", "inactive-merchant-account"},
		{"system@bank", cfg.SystemHoldingAccount},
	}

	for _, vpa := range vpas {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO vpas (id, handle, account_id, status, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			uuid.NewString(), vpa.Handle, vpa.AccountID, "ACTIVE", now,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

