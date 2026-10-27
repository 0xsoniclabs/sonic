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

package bundles

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"testing"

	sonictool "github.com/0xsoniclabs/sonic/cmd/sonictool/app"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/bundle"
	"github.com/0xsoniclabs/sonic/integration"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/0xsoniclabs/sonic/tests/gas_subsidies"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestBundle_ExportFailedBundlesListsBundlesWithUnrevertedFailures(t *testing.T) {
	upgrades := opera.GetBrioUpgrades()
	upgrades.TransactionBundles = true
	upgrades.GasSubsidies = true
	net := tests.StartIntegrationTestNet(t, tests.IntegrationTestNetOptions{Upgrades: &upgrades})

	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)
	accounts := tests.MakeAccountsWithBalance(t, net, 20, big.NewInt(1e18))
	sponsoredAccount := tests.NewAccount()
	gas_subsidies.Fund(t, net, sponsoredAccount.Address(), big.NewInt(1e18))

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()
	blockNumber, err := client.BlockNumber(t.Context())
	require.NoError(t, err)

	signer := types.LatestSignerForChainID(net.GetChainId())
	reverting := func(account *tests.Account) bundle.BuilderStep {
		return Step(t, net, account, &types.AccessListTx{To: &revertAddress, Gas: 1_000_000, Data: revertInput})
	}
	transfer := func(account *tests.Account) bundle.BuilderStep {
		return Step(t, net, account, &types.AccessListTx{To: &common.Address{1}, Value: big.NewInt(1)})
	}
	nested := func(account *tests.Account, root bundle.BuilderStep) bundle.BuilderStep {
		envelope, _ := bundle.NewBuilder().WithSigner(signer).SetEarliest(blockNumber).With(root).BuildEnvelopeAndPlan()
		return bundle.Step(account.PrivateKey, envelope)
	}
	// A sponsored transaction is followed by an internal fee payment in the block.
	sponsoredTx := tests.SetTransactionDefaults(t, net, &types.AccessListTx{To: &common.Address{1}, Gas: 21_000}, sponsoredAccount)
	sponsoredTx.GasPrice = big.NewInt(0)
	sponsored := bundle.Step(sponsoredAccount.PrivateKey, sponsoredTx)
	// The nested envelope of a case is exported in place of the case's own
	// envelope, after offset block transactions of the case.
	cases := map[string]struct {
		root     bundle.BuilderStep
		exported bool
		nested   bool
		offset   int
	}{
		"FailedBareRootIsExported":                                {root: reverting(accounts[0]), exported: true},
		"FailedGroupIsSkipped":                                    {root: bundle.AllOf(reverting(accounts[1]))},
		"SucceededBundleIsSkipped":                                {root: transfer(accounts[2])},
		"FailedNestedBareRootIsExported":                          {root: nested(accounts[5], reverting(accounts[6])), exported: true, nested: true},
		"FailedNestedBareRootInGroupIsExported":                   {root: bundle.OneOf(nested(accounts[7], reverting(accounts[8])), transfer(accounts[9])), exported: true, nested: true},
		"FailedNestedBareRootAfterTransactionIsExported":          {root: bundle.AllOf(transfer(accounts[14]), nested(accounts[12], reverting(accounts[13])).WithFlags(bundle.EF_TolerateFailed)), exported: true, nested: true, offset: 1},
		"FailedNestedBareRootAfterSponsoredTransactionIsExported": {root: bundle.AllOf(sponsored, nested(accounts[18], reverting(accounts[19])).WithFlags(bundle.EF_TolerateFailed)), exported: true, nested: true, offset: 2},
		"FailedNestedBareRootInRevertedGroupIsSkipped":            {root: bundle.AllOf(nested(accounts[15], reverting(accounts[16])).WithFlags(bundle.EF_TolerateFailed), reverting(accounts[17]))},
		"FailedNestedGroupIsSkipped":                              {root: nested(accounts[10], bundle.AllOf(reverting(accounts[11])))},
	}

	txs := []*types.Transaction{tests.CreateTransaction(t, net, &types.LegacyTx{To: &common.Address{}, Value: big.NewInt(1)}, accounts[3])}
	type export struct {
		envelope *types.Transaction
		offset   int
	}
	var plans []common.Hash
	exports := map[common.Hash]export{}
	for _, c := range cases {
		envelope, plan := bundle.NewBuilder().WithSigner(signer).SetEarliest(blockNumber).With(c.root).BuildEnvelopeAndPlan()
		txs = append(txs, envelope)
		plans = append(plans, plan.Hash())
		if c.nested {
			txBundle, err := bundle.OpenEnvelope(signer, envelope)
			require.NoError(t, err)
			txs := txBundle.GetTransactionsInReferencedOrder()
			envelope = txs[slices.IndexFunc(txs, bundle.IsEnvelope)]
		}
		if c.exported {
			exports[plan.Hash()] = export{envelope, c.offset}
		}
	}
	// An envelope out of its block range is skipped as invalid.
	outdated, _ := bundle.NewBuilder().WithSigner(signer).SetEarliest(1).SetRangeLength(1).With(transfer(accounts[4])).BuildEnvelopeAndPlan()
	txs = append(txs, outdated)

	_, err = net.ForceEmitAll(t.Context(), txs)
	require.NoError(t, err)

	infos, err := WaitForBundleExecutions(t.Context(), client.Client(), plans)
	require.NoError(t, err)
	want := []failedBundleEntry{}
	for i, info := range infos {
		if e, ok := exports[plans[i]]; ok {
			tx, err := e.envelope.MarshalBinary()
			require.NoError(t, err)
			want = append(want, failedBundleEntry{Block: uint64(info.Block), Pos: int(info.Position) + e.offset, Tx: tx})
		}
	}

	// Sealing epochs checks the bundle history hash at their ends.
	net.AdvanceEpoch(t, 2)
	net.Stop()
	file := filepath.Join(t.TempDir(), "failed_bundles.json")
	require.NoError(t, sonictool.RunWithArgs([]string{
		"sonictool",
		"--datadir", filepath.Join(net.GetDirectory(), "state"),
		"events", "export-failed-bundles", file,
	}))

	data, err := os.ReadFile(file)
	require.NoError(t, err)
	var got []failedBundleEntry
	require.NoError(t, json.Unmarshal(data, &got))
	require.ElementsMatch(t, want, got)
}

func TestBundle_ExportFailedBundlesResumesAfterEarlierExport(t *testing.T) {
	net := GetIntegrationTestNetWithBundlesEnabled(t)

	revertAddress, revertInput := tests.MustDeployRevertContractAndGetMethodCallParameters(t, net)
	accounts := tests.MakeAccountsWithBalance(t, net, 6, big.NewInt(1e18))
	signer := types.LatestSignerForChainID(net.GetChainId())
	transfer := &types.LegacyTx{To: &common.Address{1}, Value: big.NewInt(1)}

	blockNumber := func() uint64 {
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()
		n, err := client.BlockNumber(t.Context())
		require.NoError(t, err)
		return n
	}
	fillUntil := func(block uint64) {
		for blockNumber() < block {
			_, err := net.ForceEmitAll(t.Context(), []*types.Transaction{tests.CreateTransaction(t, net, transfer, accounts[0])})
			require.NoError(t, err)
		}
	}
	build := func(account *tests.Account, tx *types.AccessListTx) (*types.Transaction, *bundle.TransactionBundle, bundle.ExecutionPlan) {
		return bundle.NewBuilder().WithSigner(signer).SetEarliest(blockNumber()).With(Step(t, net, account, tx)).BuildEnvelopeBundleAndPlan()
	}
	// run emits a failing bundle along the given transactions and returns its block.
	var want []failedBundleEntry
	run := func(account *tests.Account, others ...*types.Transaction) uint64 {
		failed, _, plan := build(account, &types.AccessListTx{To: &revertAddress, Gas: 1_000_000, Data: revertInput})
		_, err := net.ForceEmitAll(t.Context(), append([]*types.Transaction{failed}, others...))
		require.NoError(t, err)
		client, err := net.GetClient()
		require.NoError(t, err)
		defer client.Close()
		info, err := WaitForBundleExecution(t.Context(), client.Client(), plan.Hash())
		require.NoError(t, err)
		tx, err := failed.MarshalBinary()
		require.NoError(t, err)
		want = append(want, failedBundleEntry{Block: uint64(info.Block), Pos: int(info.Position), Tx: tx})
		return uint64(info.Block)
	}
	// digest hashes the contents of the node's databases.
	digest := func() common.Hash {
		dir := filepath.Join(net.GetDirectory(), "state")
		hasher := sha256.New()
		dbs := integration.GetRawDbProducer(filepath.Join(dir, "chaindata"), integration.DBCacheConfig{Cache: 16 << 20, Fdlimit: 100})
		for _, name := range dbs.Names() {
			db, err := dbs.OpenDB(name)
			require.NoError(t, err)
			it := db.NewIterator(nil, nil)
			for it.Next() {
				fmt.Fprintf(hasher, "%s %x %x\n", name, it.Key(), it.Value())
			}
			require.NoError(t, it.Error())
			it.Release()
			require.NoError(t, db.Close())
		}
		require.NoError(t, filepath.WalkDir(filepath.Join(dir, "carmen"), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			fmt.Fprintf(hasher, "%s %x\n", path, data)
			return err
		}))
		return common.Hash(hasher.Sum(nil))
	}
	export := func(file string, args ...string) []failedBundleEntry {
		require.NoError(t, sonictool.RunWithArgs(append([]string{
			"sonictool",
			"--datadir", filepath.Join(net.GetDirectory(), "state"),
			"events", "export-failed-bundles",
		}, append(args, file)...)))
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		var entries []failedBundleEntry
		require.NoError(t, json.Unmarshal(data, &entries))
		return entries
	}

	// The second export resumes in a later epoch than the one of the bundles
	// processed within the block range before it, which it replays to skip the
	// rescheduled bundle, and takes the history hash from the end of that epoch.
	run(accounts[5])
	net.AdvanceEpoch(t, 1)
	fillUntil(blockNumber() + 200)
	scheduled, rescheduled, _ := build(accounts[1], &types.AccessListTx{To: &common.Address{1}, Value: big.NewInt(1)})
	first := run(accounts[2], scheduled)
	fillUntil(first + 400)
	net.AdvanceEpoch(t, 1)
	fillUntil(first + 900)
	run(accounts[3])

	// The export leaves the databases of the stopped node unchanged.
	net.Stop()
	file := filepath.Join(t.TempDir(), "failed_bundles.json")
	before := digest()
	require.Equal(t, want, export(file))
	require.Equal(t, before, digest())
	require.NoError(t, net.Restart())

	require.Less(t, run(accounts[4], bundle.MustWrapIntoEnvelope(signer, rescheduled)), first+bundle.MaxBlockRangeLength)
	// Sealing epochs checks the bundle history hash at their ends.
	net.AdvanceEpoch(t, 2)
	net.Stop()

	require.Equal(t, want, export(file, "--append"))
	require.Equal(t, want, export(filepath.Join(t.TempDir(), "full.json")))
}

type failedBundleEntry struct {
	Block uint64        `json:"block"`
	Pos   int           `json:"pos"`
	Tx    hexutil.Bytes `json:"tx"`
}
