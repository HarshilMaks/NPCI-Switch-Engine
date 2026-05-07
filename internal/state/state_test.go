package state

import "testing"

func TestEnsureTransitionAllowed_ValidTransitions(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{"INITIATED", "AUTH_PENDING"},
		{"INITIATED", "FAILED"},
		{"AUTH_PENDING", "AUTHORIZED"},
		{"AUTHORIZED", "DEBIT_POSTED"},
		{"DEBIT_POSTED", "CREDIT_POSTED"},
		{"DEBIT_POSTED", "REVERSAL_PENDING"},
		{"CREDIT_POSTED", "COMPLETED"},
		{"REVERSAL_PENDING", "REVERSED"},
		{"REVERSAL_PENDING", "REVERSAL_FAILED"},
	}

	for _, tt := range tests {
		if err := EnsureTransitionAllowed(tt.from, tt.to); err != nil {
			t.Fatalf("expected transition %s -> %s to be valid, got error: %v", tt.from, tt.to, err)
		}
	}
}

func TestEnsureTransitionAllowed_InvalidTransitions(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{"INITIATED", "COMPLETED"},
		{"AUTHORIZED", "COMPLETED"},
		{"FAILED", "INITIATED"},
		{"COMPLETED", "FAILED"},
		{"UNKNOWN", "INITIATED"},
	}

	for _, tt := range tests {
		if err := EnsureTransitionAllowed(tt.from, tt.to); err == nil {
			t.Fatalf("expected transition %s -> %s to be invalid", tt.from, tt.to)
		}
	}
}
