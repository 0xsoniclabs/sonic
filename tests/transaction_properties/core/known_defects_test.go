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
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/stretchr/testify/require"
)

func TestWriteSightings_ShowsAFewAndCountsTheRest(t *testing.T) {
	var out strings.Builder
	writeSightings(&out, []string{"one", "two", "three", "four", "five"})

	rendered := out.String()
	require.Contains(t, rendered, "one")
	require.Contains(t, rendered, "three")
	require.NotContains(t, rendered, "four", "the report must not bury the run")
	require.Contains(t, rendered, "... and 2 more")
}

func TestWriteSightings_CountsNothingExtraWhenEverythingFits(t *testing.T) {
	var out strings.Builder
	writeSightings(&out, []string{"one", "two"})

	require.NotContains(t, out.String(), "more")
}

// TestDefectLog_ReportsANoteNothingCouldHaveReproduced covers the reason notes exist: the run cannot
// draw the input that provokes the defect, so nothing but the note itself distinguishes this run from
// one against a client that had been fixed.
func TestDefectLog_ReportsANoteNothingCouldHaveReproduced(t *testing.T) {
	log := &DefectLog{}
	require.Empty(t, log.Summary("Brio"), "a log with nothing in it says nothing")

	log.Note(DefectNote{
		Summary:   "something takes the node down",
		Avoidance: "the node does not survive it",
		Detail: `
			stack trace
			  indented line
			what to delete once it is fixed`,
	})

	summary := log.Summary("Brio")
	require.Contains(t, summary, "KNOWN DEFECTS REPRODUCED OR AVOIDED on Brio")
	require.Contains(t, summary, "something takes the node down")
	require.Contains(t, summary, "not reproducible by it")
	require.Contains(t, summary, "because the node does not survive it",
		"why the input is kept out is the note's to say, since it differs per defect")
	require.Contains(t, summary, "  indented line", "a stack trace keeps its own indentation")
	require.Contains(t, summary, "what to delete once it is fixed")
}

func TestDefectLog_FailsOnANoteUnlessAskedNotTo(t *testing.T) {
	defer func(previous bool) { FailOnKnownDefects = previous }(FailOnKnownDefects)

	log := &DefectLog{}
	log.Note(DefectNote{Summary: "something takes the node down"})

	FailOnKnownDefects = true
	fake := &testing.T{}
	log.Report(fake, "Brio", opera.GetBrioUpgrades())
	require.True(t, fake.Failed(), "a defect the harness had to avoid must not pass silently")

	FailOnKnownDefects = false
	log.Report(t, "Brio", opera.GetBrioUpgrades()) // logged instead, and this test still passes
}
