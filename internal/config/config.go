package config

import "os"

type Settings struct {
	AppName               string
	DatabaseURL           string
	APIPrefix             string
	DefaultCurrency       string
	SystemHoldingAccount  string
	DefaultSeedBalanceStr string
}

func Load() Settings {
	return Settings{
		AppName:               "UPI Rail Simulator",
		DatabaseURL:           env("DATABASE_URL", "file:npci_upi.db?_pragma=busy_timeout(5000)"),
		APIPrefix:             "/api/v1",
		DefaultCurrency:       env("DEFAULT_CURRENCY", "INR"),
		SystemHoldingAccount:  env("HOLDING_ACCOUNT_ID", "system-holding-account"),
		DefaultSeedBalanceStr: env("DEFAULT_SEED_BALANCE", "1000000.00"),
	}
}

func env(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

