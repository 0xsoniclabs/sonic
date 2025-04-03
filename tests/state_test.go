// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/tests"
)

var (
	testPaths = []string{
		filepath.Join(".", "testdata", "EIPTests", "StateTests"),
		filepath.Join(".", "testdata", "GeneralStateTests"),
		filepath.Join(".", "execution-spec-tests", "fixtures", "state_tests"),
	}

	unsupportedForks = map[string]struct{}{
		"ConstantinopleFix": {},
		"Constantinople":    {},
		"Byzantium":         {},
		"Frontier":          {},
		"Homestead":         {},
	}
)

func initMatcher(st *tests.TestMatcher) {
	// EOF is not yet supported by sonic.
	st.SkipLoad(`^stEOF/`)
}

// TestState runs the state tests from the Ethereum tests and Ethereum execution spec tests.
// In order to run the Ethereum tests, clone the `ethereum/tests` repository inside this tests directory as `testdata`.
// As the tests are pre-filled no further steps are needed.
// For the execution spec tests, clone the `ethereum/execution-spec-tests` repository inside this tests directory.
// Install the python dependencies (using uv is recommended):
// `uv sync --all-extras && uv run solc-select use 0.8.24 --always-install`
// Collect all desired test cases:
// `uv run fill --collect-only --from Istanbul --to Prague`
// Fill all state tests (these are the only ones we currently support):
// `uv run fill --from Istanbul --until Prague -m state_test`
func TestState(t *testing.T) {
	t.Parallel()

	st := new(tests.TestMatcher)
	initMatcher(st)
	for _, dir := range testPaths {
		// If the directory does not exist,
		// skip it but do not exit test without checking the other directories.
		dirinfo, err := os.Stat(dir)
		if os.IsNotExist(err) || !dirinfo.IsDir() {
			t.Logf("Skipping %s as it does not exist, did you clone/fill the tests?\n", dir)
			continue
		}

		st.Walk(t, dir, func(t *testing.T, name string, test *tests.StateTest) {
			execStateTest(t, st, test)
		})
	}
}

func execStateTest(t *testing.T, st *tests.TestMatcher, test *tests.StateTest) {
	for _, subtest := range test.Subtests() {
		subtest := subtest
		key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)

		t.Run(key, func(t *testing.T) {
			if _, ok := unsupportedForks[subtest.Fork]; ok {
				t.Skipf("unsupported fork %s", subtest.Fork)
			}

			factory := createCarmenFactory(t)

			config := opera.DefaultVMConfig
			config.ChargeExcessGas = false
			config.IgnoreGasFeeCap = false
			config.InsufficientBalanceIsNotAnError = false
			config.SkipTipPaymentToCoinbase = false

			err := test.RunWith(subtest, config, factory, func(err error, state *tests.StateTestState) {})
			if err := st.CheckFailure(t, err); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// createCarmenFactory creates a new factory, that initialises
// carmen implementation of the state database.
func createCarmenFactory(t *testing.T) carmenFactory {
	// ethereum tests creates extensively long test names, which causes t.TempDir fails
	// on a too long names. For this reason, we use os.MkdirTemp instead.
	dir, err := os.MkdirTemp("", "eth-tests-carmen-*")
	if err != nil {
		t.Fatalf("cannot create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatalf("cannot remove temp dir: %v", err)
		}
	})

	parameters := carmen.Parameters{
		Variant:   gostate.VariantGoMemory,
		Schema:    carmen.Schema(5),
		Archive:   carmen.NoArchive,
		Directory: dir,
	}

	st, err := carmen.NewState(parameters)
	if err != nil {
		t.Fatalf("cannot create state: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("cannot close state: %v", err)
		}
	})

	return carmenFactory{st: st}
}
