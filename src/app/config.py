import os


class Settings:
    app_name: str = "UPI Rail Simulator"
    database_url: str = os.getenv("DATABASE_URL", "sqlite:///./npci_upi.db")
    api_prefix: str = "/api/v1"
    default_currency: str = "INR"
    system_holding_account_id: str = "system-holding-account"
    default_seed_balance: str = "1000000.00"


settings = Settings()

