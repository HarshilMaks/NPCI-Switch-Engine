package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"npci-upi/internal/config"
	"npci-upi/internal/storage"
	"npci-upi/internal/types"
)

func setupTestDB(t *testing.T) (*PaymentService, *ReconciliationService) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open("file:" + dbPath + "?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := storage.Migrate(db); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	cfg := config.Settings{
		DefaultCurrency:       "INR",
		SystemHoldingAccount:  "system-holding-account",
		DefaultSeedBalanceStr: "1000000.00",
	}
	if err := storage.Seed(db, cfg); err != nil {
		t.Fatalf("seed test db: %v", err)
	}

	return NewPaymentService(db, cfg), NewReconciliationService(db)
}

func TestCreatePayment_Idempotent(t *testing.T) {
	paymentSvc, _ := setupTestDB(t)

	req := types.PaymentCreateRequest{
		PayerVPA: "alice@bank",
		PayeeVPA: "bob@bank",
		Amount:   "100.00",
		Currency: "INR",
	}

	code1, resp1, err := paymentSvc.CreatePayment(context.Background(), req, "idem-1", "corr-1")
	if err != nil {
		t.Fatalf("first create payment failed: %v", err)
	}
	if code1 != 201 {
		t.Fatalf("expected first status code 201, got %d", code1)
	}

	code2, resp2, err := paymentSvc.CreatePayment(context.Background(), req, "idem-1", "corr-2")
	if err != nil {
		t.Fatalf("second create payment failed: %v", err)
	}
	if code2 != 201 {
		t.Fatalf("expected second status code 201, got %d", code2)
	}

	if resp1.TransactionID != resp2.TransactionID {
		t.Fatalf("expected same transaction id for idempotent request, got %s and %s", resp1.TransactionID, resp2.TransactionID)
	}
}

func TestCreatePayment_IdempotencyConflict(t *testing.T) {
	paymentSvc, _ := setupTestDB(t)

	req1 := types.PaymentCreateRequest{
		PayerVPA: "alice@bank",
		PayeeVPA: "bob@bank",
		Amount:   "100.00",
		Currency: "INR",
	}
	req2 := types.PaymentCreateRequest{
		PayerVPA: "alice@bank",
		PayeeVPA: "bob@bank",
		Amount:   "101.00",
		Currency: "INR",
	}

	if _, _, err := paymentSvc.CreatePayment(context.Background(), req1, "idem-conflict", "corr-1"); err != nil {
		t.Fatalf("first create payment failed: %v", err)
	}

	_, _, err := paymentSvc.CreatePayment(context.Background(), req2, "idem-conflict", "corr-2")
	if err == nil {
		t.Fatal("expected idempotency conflict error")
	}
	appErr, ok := err.(AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Status != 409 || appErr.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("unexpected conflict error: %+v", appErr)
	}
}

func TestValidatePaymentRequest(t *testing.T) {
	valid := types.PaymentCreateRequest{
		PayerVPA: "alice@bank",
		PayeeVPA: "bob@bank",
		Amount:   "1.00",
		Currency: "INR",
	}
	if err := validatePaymentRequest(valid); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	invalid := []types.PaymentCreateRequest{
		{PayeeVPA: "bob@bank", Amount: "1.00", Currency: "INR"},
		{PayerVPA: "alice@bank", Amount: "1.00", Currency: "INR"},
		{PayerVPA: "alice@bank", PayeeVPA: "bob@bank", Amount: "0", Currency: "INR"},
		{PayerVPA: "alice@bank", PayeeVPA: "bob@bank", Amount: "abc", Currency: "INR"},
		{PayerVPA: "alice@bank", PayeeVPA: "bob@bank", Amount: "1.00"},
	}

	for i, req := range invalid {
		if err := validatePaymentRequest(req); err == nil {
			t.Fatalf("expected invalid request #%d to fail validation", i)
		}
	}
}

func TestReconciliation_MissingLegAndStalePendingDetected(t *testing.T) {
	paymentSvc, reconciliationSvc := setupTestDB(t)
	db := paymentSvc.DB

	now := time.Now().UTC()
	old := now.Add(-11 * time.Minute).Format(time.RFC3339Nano)
	curr := now.Format(time.RFC3339Nano)

	_, err := db.Exec(
		`INSERT INTO transactions (id, payer_vpa, payee_vpa, amount, currency, status, version, idempotency_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"tx-missing-leg", "alice@bank", "bob@bank", "10.00", "INR", "COMPLETED", 1, "idem-missing-leg", curr, curr,
	)
	if err != nil {
		t.Fatalf("insert completed transaction: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO transactions (id, payer_vpa, payee_vpa, amount, currency, status, version, idempotency_key, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"tx-stale-pending", "alice@bank", "bob@bank", "9.00", "INR", "AUTH_PENDING", 1, "idem-stale-pending", old, old,
	)
	if err != nil {
		t.Fatalf("insert stale transaction: %v", err)
	}

	out, err := reconciliationSvc.Run(context.Background())
	if err != nil {
		t.Fatalf("reconciliation run failed: %v", err)
	}

	summary, ok := out["summary"].(map[string]any)
	if !ok {
		t.Fatalf("summary missing or invalid type: %T", out["summary"])
	}

	diffCount, ok := summary["diff_count"].(int)
	if !ok {
		t.Fatalf("diff_count invalid type: %T", summary["diff_count"])
	}
	if diffCount < 2 {
		t.Fatalf("expected at least 2 diffs, got %d", diffCount)
	}

	staleCount, ok := summary["stale_pending_count"].(int)
	if !ok {
		t.Fatalf("stale_pending_count invalid type: %T", summary["stale_pending_count"])
	}
	if staleCount < 1 {
		t.Fatalf("expected at least 1 stale pending diff, got %d", staleCount)
	}
}
