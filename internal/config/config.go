package config

import (
	"errors"
	"os"
)

type Settings struct {
	AppName               string
	DatabaseURL           string
	APIPrefix             string
	DefaultCurrency       string
	SystemHoldingAccount  string
	DefaultSeedBalanceStr string
	RequestTimeout        int // seconds
}

func Load() (Settings, error) {
	cfg := Settings{
		AppName:               "UPI Rail Simulator",
		DatabaseURL:           env("DATABASE_URL", "file:npci_upi.db?_pragma=busy_timeout(5000)"),
		APIPrefix:             "/api/v1",
		DefaultCurrency:       env("DEFAULT_CURRENCY", "INR"),
		SystemHoldingAccount:  env("HOLDING_ACCOUNT_ID", "system-holding-account"),
		DefaultSeedBalanceStr: env("DEFAULT_SEED_BALANCE", "1000000.00"),
		RequestTimeout:        30,
	}

	// Validate critical fields
	if cfg.SystemHoldingAccount == "" {
		return cfg, errors.New("HOLDING_ACCOUNT_ID cannot be empty")
	}
	if cfg.DefaultCurrency == "" {
		return cfg, errors.New("DEFAULT_CURRENCY cannot be empty")
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL cannot be empty")
	}

	return cfg, nil
}

func env(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

