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
	baseDir      = filepath.Join(".", "testdata")
	stateTestDir = filepath.Join(baseDir, "GeneralStateTests")

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

func TestState(t *testing.T) {
	t.Parallel()

	st := new(tests.TestMatcher)
	initMatcher(st)
	for _, dir := range []string{
		filepath.Join(baseDir, "EIPTests", "StateTests"),
		stateTestDir,
	} {
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
