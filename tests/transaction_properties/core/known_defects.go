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

// This file holds everything about the defects this package knows about and tolerates: where a check
// has to accept a range because of one, and how a run that reproduced one reports it. Deleting a
// defect from here once it is fixed should tighten the checks back to exact equality.

package core

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/urfave/cli.v1"
)

// FailOnKnownDefects decides whether a defect this run reproduced fails the test, which it does
// unless asked otherwise because a failing test is the only report that cannot be missed:
// tolerating a defect silently means a green run and a finding nobody sees again, and printing is no
// better because `go test` discards a passing package's output unless -v is given. Turn it off only
// where something else reports them, such as a CI job reading them out of the log.
var FailOnKnownDefects bool

// KnownDefectsFlag defines the flag above, as a cli.BoolTFlag from the flag library the client's own
// commands use: that is what makes it default to true, fall back to the environment when the command
// line is silent -- which is the only way to reach this package from a run of the whole module --
// and reject a value it cannot parse. TestMain applies it to the test binary's flag set.
var KnownDefectsFlag = cli.BoolTFlag{
	Name:        "properties.fail-on-known-defects",
	EnvVar:      "SONIC_TEST_FAIL_ON_KNOWN_DEFECTS",
	Usage:       "report a known defect reproduced by this run as a test failure rather than as a log entry",
	Destination: &FailOnKnownDefects,
}

// DefectNote is a defect a domain knows about and had to design around rather than tolerate: the
// input that provokes it is kept out of the draws, so a run cannot reproduce it and cannot report it
// as a sighting either. Reporting the avoidance is what keeps it from disappearing.
type DefectNote struct {
	// Summary is the one-line statement of the defect.
	Summary string

	// Avoidance completes "the input that provokes it is kept out of the draws, because ...", which
	// differs per defect: one crashes the node, another leaves state no block accounts for.
	Avoidance string

	// Detail says where it is, how it is reached, what the harness does about it, and what deleting
	// that workaround will take. It is indented and printed under the summary.
	Detail string
}

// DefectLog collects what a run witnessed of each defect. It exists because the checks accept a
// range wherever a defect makes the exact answer unpredictable, so a run that hit one would
// otherwise look exactly like a run that did not -- and, for a defect nothing can be drawn for at
// all, because a run that avoided one would otherwise look exactly like a fixed client.
type DefectLog struct {
	unreceiptedCharges []string
	notes              []DefectNote
}

// Note records a defect the domains under test had to design around. It is reported like a sighting,
// because a workaround nobody is reminded of outlives the defect it was written for.
func (l *DefectLog) Note(notes ...DefectNote) {
	l.notes = append(l.notes, notes...)
}

func (l *DefectLog) UnreceiptedCharge(account common.Address, amount *big.Int) {
	l.unreceiptedCharges = append(l.unreceiptedCharges, fmt.Sprintf(
		"%v: paid %v wei for transactions that produced no receipt", account, amount,
	))
}

// Report describes every defect the run reproduced, and every one the domains under test had to
// avoid, as a failure unless FailOnKnownDefects says otherwise.
func (l *DefectLog) Report(t *testing.T, fork string, upgrades opera.Upgrades) {
	t.Helper()

	mustFail := len(l.notes) > 0 ||
		(len(l.unreceiptedCharges) > 0 && !upgrades.Allegro) // Allegro fixed defect 2

	summary := l.Summary(fork)
	switch {
	case summary == "":
	case FailOnKnownDefects && mustFail:
		t.Errorf("%s", summary)
	default:
		t.Logf("%s", summary)
	}
}

// Summary is what Report says, or "" when the run reproduced nothing and had nothing to avoid.
func (l *DefectLog) Summary(fork string) string {
	if len(l.unreceiptedCharges) == 0 && len(l.notes) == 0 {
		return ""
	}

	// The header carries the word CI greps for, whichever kind of finding follows.
	headline := "KNOWN DEFECTS REPRODUCED"
	if len(l.notes) > 0 {
		headline = "KNOWN DEFECTS REPRODUCED OR AVOIDED"
	}

	var out strings.Builder
	fmt.Fprintf(&out, "\n=== %s on %s ===\n", headline, fork)

	if sightings := l.unreceiptedCharges; len(sightings) > 0 {
		fmt.Fprintf(&out,
			"\n  [2] A transaction the state processor cannot apply keeps the gas it bought\n"+
				"      without reaching a block or a receipt (%d occurrence(s)), so the state\n"+
				"      cannot be replayed from block data alone.\n"+
				"      Cause: the snapshot/revert around ApplyMessage in evmcore is gated on\n"+
				"      Allegro. Allegro and later forks fix it; on earlier ones this is\n"+
				"      reported without failing the test.\n"+
				"      Examples:\n", len(sightings))
		writeSightings(&out, sightings)
	}

	for _, note := range l.notes {
		fmt.Fprintf(&out,
			"\n  [-] %s\n"+
				"      Not reproduced by this run and not reproducible by it: the input that\n"+
				"      provokes it is kept out of the draws, because %s.\n",
			note.Summary, note.Avoidance)
		writeDetail(&out, note.Detail)
	}

	out.WriteString("\n=== end of known defects ===")
	return out.String()
}

// writeDetail indents a note's detail under its summary, keeping whatever relative indentation it
// has: a stack trace is only readable with it.
func writeDetail(out *strings.Builder, detail string) {
	lines := strings.Split(strings.Trim(detail, "\n"), "\n")

	margin := math.MaxInt
	for _, line := range lines {
		if trimmed := strings.TrimLeft(line, "\t"); trimmed != "" {
			margin = min(margin, len(line)-len(trimmed))
		}
	}

	for _, line := range lines {
		if len(line) <= margin {
			out.WriteString("\n") // a blank line, without a margin of trailing spaces
			continue
		}
		fmt.Fprintf(out, "      %s\n", strings.ReplaceAll(line[margin:], "\t", "    "))
	}
}

// writeSightings renders a few sightings, enough to act on without burying the run.
func writeSightings(out *strings.Builder, sightings []string) {
	const shown = 3
	for _, sighting := range sightings[:min(shown, len(sightings))] {
		fmt.Fprintf(out, "        %s\n", sighting)
	}
	if extra := len(sightings) - shown; extra > 0 {
		fmt.Fprintf(out, "        ... and %d more\n", extra)
	}
}
