package core_types

import "testing"

func TestTransactionResult_Constants(t *testing.T) {
	// Verify the enum values are distinct and ordered as expected.
	if TransactionResultInvalid != 0 {
		t.Errorf("expected TransactionResultInvalid == 0, got %d", TransactionResultInvalid)
	}
	if TransactionResultFailed != 1 {
		t.Errorf("expected TransactionResultFailed == 1, got %d", TransactionResultFailed)
	}
	if TransactionResultSuccessful != 2 {
		t.Errorf("expected TransactionResultSuccessful == 2, got %d", TransactionResultSuccessful)
	}
}

func TestTransactionResult_Distinct(t *testing.T) {
	results := []TransactionResult{
		TransactionResultInvalid,
		TransactionResultFailed,
		TransactionResultSuccessful,
	}
	seen := make(map[TransactionResult]bool)
	for _, r := range results {
		if seen[r] {
			t.Errorf("duplicate TransactionResult value: %d", r)
		}
		seen[r] = true
	}
}
