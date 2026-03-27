// Copyright 2026 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

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
