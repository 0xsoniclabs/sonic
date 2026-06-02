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

package execspec

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	carmen "github.com/0xsoniclabs/carmen/go/state"
	"github.com/0xsoniclabs/carmen/go/state/gostate"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/tosca/go/geth_adapter"
	"github.com/0xsoniclabs/tosca/go/interpreter/lfvm"
	"github.com/0xsoniclabs/tosca/go/interpreter/sfvm"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/stretchr/testify/require"
)

// In order to run these tests, provide the path to the fixtures directory via the
// -test-data flag:
//
//	go test ./tests/eth_exec_spec/ -test-data /path/to/fixtures
//
// Fixtures can be downloaded from the Ethereum execution-spec-tests releases page:
// https://github.com/ethereum/execution-spec-tests/releases
// Run either all permutations of supported VMs and stateDBs or a specific combination
// by calling a sub-test. In case of an error, TestState_RunSingleCase can be used
// to run a single test case.

var (
	fixturesDir = flag.String(
		"test-data",
		"",
		"path to the fixtures directory. If empty, tests will be skipped.",
	)

	unsupportedForks = map[string]struct{}{
		"ConstantinopleFix": {},
		"Constantinople":    {},
		"Byzantium":         {},
		"Frontier":          {},
		"Homestead":         {},
	}
)

// TestBlockProcessing_EthereumExecutionSpecTests runs the Ethereum execution spec tests
// using different VM and StateDB implementations.
func TestBlockProcessing_EthereumExecutionSpecTests(t *testing.T) {
	if *fixturesDir == "" {
		t.Skip("fixtures directory not provided; re-run with -test-data /path/to/fixtures")
	}
	if _, err := os.Stat(*fixturesDir); os.IsNotExist(err) {
		t.Fatalf("directory %s not does not exist", *fixturesDir)
	}
	if _, err := os.Stat(*fixturesDir + "/state_tests"); os.IsNotExist(err) {
		t.Fatalf("state_tests directory not found in %s", *fixturesDir)
	}

	defaultConfig := newEthSpecVmConfig()

	gethConfig := defaultConfig
	gethConfig.Interpreter = func(evm *vm.EVM) vm.Interpreter {
		return vm.NewEvmInterpreter(evm)
	}

	sfvmInterpreter, err := sfvm.NewInterpreter(sfvm.Config{})
	require.NoError(t, err)
	sfvmConfig := defaultConfig
	sfvmConfig.Interpreter = geth_adapter.NewGethInterpreterFactory(sfvmInterpreter)

	lfvmInterpreter, err := lfvm.NewInterpreter(lfvm.Config{})
	require.NoError(t, err)
	lfvmConfig := defaultConfig
	lfvmConfig.Interpreter = geth_adapter.NewGethInterpreterFactory(lfvmInterpreter)

	t.Run("VM: geth, StateDB: geth", func(t *testing.T) {
		runTestCases(t, gethConfig, false)
	})
	t.Run("VM: sfvm, StateDB: geth", func(t *testing.T) {
		runTestCases(t, sfvmConfig, false)
	})
	t.Run("VM: lfvm, StateDB: geth", func(t *testing.T) {
		runTestCases(t, lfvmConfig, false)
	})
	t.Run("VM: geth, StateDB: carmen", func(t *testing.T) {
		runTestCases(t, gethConfig, true)
	})
	t.Run("VM: sfvm, StateDB: carmen", func(t *testing.T) {
		runTestCases(t, sfvmConfig, true)
	})
	t.Run("VM: lfvm, StateDB: carmen", func(t *testing.T) {
		runTestCases(t, lfvmConfig, true)
	})
}

// TestState_RunSingleCase runs a single test case from the execution
// spec tests, useful for debugging specific cases.
//
// It defaults to an example test case committed in the repository,
// The path can be modified to point to any test case in the fixtures directory,
// and the test will be run only if the name contains the targetName substring.
func TestState_RunSingleCase(t *testing.T) {
	path := "test_data/example_input.json"
	targetName := "fork_Osaka-state_test-inf_plus_generator-call_opcode_CALLCODE-"

	vmConfig := newEthSpecVmConfig()
	useCarmen := true

	testMatcher := &tests.TestMatcher{}
	testMatcher.RunTestFile(t, path, "",
		func(t *testing.T, name string, test *tests.StateTest) {
			if strings.Contains(name, targetName) {
				runSubtests(t, testMatcher, test, vmConfig, useCarmen)
			}
		})
}

// runTestCases iterates over all test directories and runs each discovered StateTest using the provided vm.Config.
func runTestCases(t *testing.T, config vm.Config, useCarmen bool) {
	matcher := &tests.TestMatcher{}
	walkTestDirs(t, matcher, func(t *testing.T, name string, test *tests.StateTest) {
		runSubtests(t, matcher, test, config, useCarmen)
	})
}

// runSubtests iterates over all subtests of a StateTest and runs each one.
func runSubtests(t *testing.T, matcher *tests.TestMatcher, test *tests.StateTest, config vm.Config, useCarmen bool) {
	t.Helper()
	for _, subtest := range test.Subtests() {
		key := fmt.Sprintf("%s/%d", subtest.Fork, subtest.Index)

		t.Run(key, func(t *testing.T) {
			if _, ok := unsupportedForks[subtest.Fork]; ok {
				t.Skipf("unsupported fork %s", subtest.Fork)
			}

			if !useCarmen {
				err := test.Run(subtest, config, false, "", func(err error, state *tests.StateTestState) {})
				require.NoError(t, matcher.CheckFailure(t, err))
			} else {
				// Carmen factory has to be created per test case.
				factory := createCarmenFactory(t)
				err := test.RunWith(subtest, config, factory, func(err error, state *tests.StateTestState) {})
				require.NoError(t, matcher.CheckFailure(t, err))
			}
		})
	}
}

// walkTestDirs walks the state_tests sub-directory inside fixturesDir and calls fn
// for each discovered test. The directory is skipped with a log message if absent.
func walkTestDirs(t *testing.T, matcher *tests.TestMatcher, fn func(t *testing.T, name string, test *tests.StateTest)) {
	t.Helper()
	dir := filepath.Join(*fixturesDir, "state_tests")
	dirinfo, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) || (err == nil && !dirinfo.IsDir()) {
		t.Logf("Skipping %s: directory does not exist", dir)
		return
	}
	matcher.Walk(t, dir, fn)
}

// newEthSpecVmConfig returns the base vm.Config used across Ethereum spec tests.
func newEthSpecVmConfig() vm.Config {
	config := opera.GetVmConfig(opera.Rules{})
	config.InterpreterForTracing = nil
	config.ChargeExcessGas = false
	config.IgnoreGasFeeCap = false
	config.InsufficientBalanceIsNotAnError = false
	config.SkipTipPaymentToCoinbase = false
	config.MaxTxGas = nil
	config.MaxCodeSize = nil
	config.MaxInitCodeSize = nil
	return config
}

// createCarmenFactory creates a new factory, that initializes
// the carmen implementation of the state database.
func createCarmenFactory(t *testing.T) carmenFactory {
	// ethereum tests creates extensively long test names, which causes t.TempDir fails
	// on a too long names. For this reason, we use os.MkdirTemp instead.
	dir, err := os.MkdirTemp("", "eth-tests-carmen-*")
	require.NoError(t, err, "cannot create temp dir for carmen state")

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
	require.NoError(t, err, "cannot create state")

	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("cannot close state: %v", err)
		}
	})

	return carmenFactory{st: st}
}

// TestMain is the entry point for the test suite. It registers test flags,
// silences the go-ethereum global logger by default, and forwards the exit code
// of m.Run() to the shell so CI can detect pass/fail.
func TestMain(m *testing.M) {
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "enable go-ethereum global logger output")
	flag.Parse()
	if !verbose {
		log.SetDefault(log.NewLogger(log.NewTerminalHandler(io.Discard, false)))
	}
	os.Exit(m.Run())
}
