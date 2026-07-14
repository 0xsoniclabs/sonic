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

package errlock

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// Environment variables used to drive Permanent() inside a re-executed test
// subprocess. Permanent() terminates the process via os.Exit, so it cannot be
// exercised in-process; the test re-runs its own binary and inspects the exit
// code and stderr of the child.
const (
	envRunPermanent = "ERRLOCK_TEST_RUN_PERMANENT"
	envPermanentDir = "ERRLOCK_TEST_PERMANENT_DIR"
)

// TestMain intercepts the re-executed subprocess and invokes Permanent() with
// the data directory provided by the parent. In the normal (parent) case it
// just runs the test suite.
func TestMain(m *testing.M) {
	if os.Getenv(envRunPermanent) == "1" {
		New(os.Getenv(envPermanentDir)).Permanent(errors.New("boom: triggering permanent halt"))
		// Permanent() must not return; if it does the test fails via the
		// unexpected exit code observed by the parent.
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runPermanentInSubprocess re-executes the test binary so that TestMain calls
// Permanent() with the given data directory, and returns the child's exit code
// and captured stderr.
func runPermanentInSubprocess(t *testing.T, dir string) (exitCode int, stderr string) {
	t.Helper()

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		envRunPermanent+"=1",
		envPermanentDir+"="+dir,
	)
	var buf bytes.Buffer
	cmd.Stderr = &buf
	err := cmd.Run()

	if err == nil {
		return 0, buf.String()
	}
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "expected the subprocess to exit with a non-zero status")
	return exitErr.ExitCode(), buf.String()
}

func TestPermanent_WriteSucceeds_WritesLockFileExitsWith74AndPrintsStack(t *testing.T) {
	dir := t.TempDir()

	exitCode, stderr := runPermanentInSubprocess(t, dir)

	require.Equal(t, 74, exitCode, "Permanent must exit with the distinct code 74")

	// The halt-lock file must be written with the original error message so a
	// restart is refused by Check().
	data, err := os.ReadFile(path.Join(dir, "errlock"))
	require.NoError(t, err, "the halt-lock file should have been written")
	require.Contains(t, string(data), "boom: triggering permanent halt")

	// The stack trace must be printed so the crash location is recorded.
	require.Contains(t, stderr, "errlock.(*ErrorLock).Permanent",
		"the stack trace must include the Permanent call site")
}

func TestPermanent_WriteFails_ExitsWith74AndPrintsStack(t *testing.T) {
	// Use a regular file as the "data directory" so that writing
	// <dir>/errlock fails (the path component is not a directory).
	f, err := os.CreateTemp(t.TempDir(), "not-a-dir")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	exitCode, stderr := runPermanentInSubprocess(t, f.Name())

	require.Equal(t, 74, exitCode,
		"Permanent must exit with 74 even when the halt-lock file cannot be written")

	// The stack trace must be printed independently of the write result.
	require.Contains(t, stderr, "errlock.(*ErrorLock).Permanent",
		"the stack trace must be printed even when writing the lock file fails")

	// No lock file should have been created at the (invalid) location.
	_, statErr := os.Stat(path.Join(f.Name(), "errlock"))
	require.Error(t, statErr)
}
