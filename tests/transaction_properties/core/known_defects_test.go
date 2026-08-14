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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestDefectLog_ReportsWhatItSkippedAndWhatItSaw covers why the log exists: the run cannot draw the
// input that provokes a skipped defect, so nothing but the note itself distinguishes this run from one
// against a client that had been fixed. It never fails a test.
func TestDefectLog_ReportsWhatItSkippedAndWhatItSaw(t *testing.T) {
	log := &DefectLog{}
	require.Empty(t, log.Summary("Brio"), "a log with nothing in it says nothing")

	log.Note("[9] something takes the node down (somewhere/skip.go)")
	log.UnreceiptedCharge(common.Address{0xab}, big.NewInt(1234))

	summary := log.Summary("Brio")
	require.Contains(t, summary, "known defects on Brio")
	require.Contains(t, summary, "avoided: [9] something takes the node down")
	require.Contains(t, summary, "seen [2]: 1 account(s)")
	require.Contains(t, summary, "paid 1234 wei")

	fake := &testing.T{}
	log.Report(fake, "Brio")
	require.False(t, fake.Failed(), "a known defect must never fail a run")
}
