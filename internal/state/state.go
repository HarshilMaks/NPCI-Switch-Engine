package state

import "fmt"

var allowedTransitions = map[string]map[string]struct{}{
	"INITIATED":       {"AUTH_PENDING": {}, "FAILED": {}},
	"AUTH_PENDING":    {"AUTHORIZED": {}, "FAILED": {}},
	"AUTHORIZED":      {"DEBIT_POSTED": {}, "FAILED": {}},
	"DEBIT_POSTED":    {"CREDIT_POSTED": {}, "REVERSAL_PENDING": {}},
	"CREDIT_POSTED":   {"COMPLETED": {}},
	"REVERSAL_PENDING": {"REVERSED": {}, "REVERSAL_FAILED": {}},
	"COMPLETED":       {},
	"FAILED":          {},
	"REVERSED":        {},
	"REVERSAL_FAILED": {},
}

func EnsureTransitionAllowed(current, target string) error {
	if allowed, ok := allowedTransitions[current]; ok {
		if _, exists := allowed[target]; exists {
			return nil
		}
	}
	return fmt.Errorf("illegal transition: %s -> %s", current, target)
}

