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

// This file holds what a run says about the defects this package knows about. Nothing here fails a
// test: a defect already known is not news, and a suite that goes red over one is a suite nobody
// reads. What provokes one is skipped instead -- see skip.go.

package core

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// DefectLog is what the run has to say about known defects: the ones its domains steered around, and
// what it saw of the one that can only be tolerated.
type DefectLog struct {
	unreceiptedCharges []string
	notes              []string
}

// Note records a defect a domain steered around, one line apiece.
func (l *DefectLog) Note(notes ...string) {
	l.notes = append(l.notes, notes...)
}

// UnreceiptedCharge records a sighting of defect 2: pre-Allegro, a transaction the state processor
// cannot apply keeps the gas it bought although it reaches no block. It is tolerated rather than
// skipped, because skipping it would mean skipping most of what a pre-Allegro run has to say.
func (l *DefectLog) UnreceiptedCharge(account common.Address, amount *big.Int) {
	l.unreceiptedCharges = append(l.unreceiptedCharges, fmt.Sprintf("%v paid %v wei", account, amount))
}

// Report logs the summary, or nothing when there is nothing to say.
func (l *DefectLog) Report(t *testing.T, fork string) {
	t.Helper()
	if summary := l.Summary(fork); summary != "" {
		t.Logf("%s", summary)
	}
}

// Summary is one line per defect.
func (l *DefectLog) Summary(fork string) string {
	if len(l.unreceiptedCharges) == 0 && len(l.notes) == 0 {
		return ""
	}

	var out strings.Builder
	fmt.Fprintf(&out, "known defects on %s:", fork)
	for _, note := range l.notes {
		fmt.Fprintf(&out, "\n    avoided: %s", note)
	}
	if seen := l.unreceiptedCharges; len(seen) > 0 {
		fmt.Fprintf(&out, "\n    seen [2]: %d account(s) charged by a transaction that got no receipt"+
			", e.g. %s", len(seen), seen[0])
	}
	return out.String()
}
