package subsidies

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/evmcore/subsidies/registry"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestIsSponsorshipRequest_DetectsSponsorshipRequest(t *testing.T) {
	require := require.New(t)

	tx := types.NewTransaction(0, common.Address{}, nil, 21000, nil, nil)
	require.True(IsSponsorshipRequest(tx))

	tx = types.NewTransaction(0, common.Address{}, nil, 21000, common.Big1, nil)
	require.False(IsSponsorshipRequest(tx))
}

func TestIsCovered_ConsultsSubsidiesRegistry(t *testing.T) {

	tests := map[string]struct {
		availableFunds uint64
		expectCovered  bool
	}{
		"no funds available": {
			availableFunds: 0,
			expectCovered:  false,
		},
		"some funds available": {
			availableFunds: 1_000_000_000_000_000,
			expectCovered:  true,
		},
		"too little funds available": {
			availableFunds: 10, // < not enough to cover any fees
			expectCovered:  false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {

			require := require.New(t)
			ctrl := gomock.NewController(t)
			state := state.NewMockStateDB(ctrl)

			registryAddress := common.Address{1, 2, 3}
			code := registry.GetCode()
			hash := crypto.Keccak256Hash(code)

			any := gomock.Any()
			state.EXPECT().Snapshot().Return(1).AnyTimes()
			state.EXPECT().Exist(registryAddress).Return(true).AnyTimes()
			state.EXPECT().GetCode(registryAddress).Return(code).AnyTimes()
			state.EXPECT().GetCodeHash(registryAddress).Return(hash).AnyTimes()
			state.EXPECT().AddRefund(gomock.Any()).AnyTimes()
			state.EXPECT().SubRefund(gomock.Any()).AnyTimes()
			state.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()
			state.EXPECT().SlotInAccessList(any, any).AnyTimes()
			state.EXPECT().AddSlotToAccessList(any, any).AnyTimes()

			funds := common.Hash(big.NewInt(int64(test.availableFunds)).FillBytes(make([]byte, 32)))
			state.EXPECT().GetState(any, any).Return(funds).AnyTimes()

			upgrades := opera.GetSonicUpgrades()
			upgrades.GasSubsidies = true
			rules := opera.FakeNetRules(upgrades)

			var updateHeights []opera.UpgradeHeight
			chainConfig := opera.CreateTransientEvmChainConfig(
				rules.NetworkID,
				updateHeights,
				1,
			)

			key, err := crypto.GenerateKey()
			require.NoError(err)

			signer := types.LatestSigner(chainConfig)
			tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
				To:  &common.Address{},
				Gas: 21000,
			})

			blockContext := vm.BlockContext{
				BlockNumber: big.NewInt(1),
				BaseFee:     big.NewInt(2),
				Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int) {
					require.Equal(0, amount.Sign())
				},
				Random: &common.Hash{}, // < signals Revision >= Merge
			}

			covered, err := IsCovered(
				registryAddress,
				blockContext,
				signer,
				chainConfig,
				rules,
				state,
				tx,
			)
			require.NoError(err)
			require.Equal(test.expectCovered, covered)
		})
	}
}

func BenchmarkIsCovered_NoCoverage(b *testing.B) {
	require := require.New(b)
	ctrl := gomock.NewController(b)
	state := state.NewMockStateDB(ctrl)

	registryAddress := common.Address{1, 2, 3}
	code := registry.GetCode()
	hash := crypto.Keccak256Hash(code)

	any := gomock.Any()
	state.EXPECT().Snapshot().Return(1).AnyTimes()
	state.EXPECT().Exist(registryAddress).Return(true).AnyTimes()
	state.EXPECT().GetCode(registryAddress).Return(code).AnyTimes()
	state.EXPECT().GetCodeHash(registryAddress).Return(hash).AnyTimes()
	state.EXPECT().AddRefund(gomock.Any()).AnyTimes()
	state.EXPECT().SubRefund(gomock.Any()).AnyTimes()
	state.EXPECT().GetRefund().Return(uint64(0)).AnyTimes()
	state.EXPECT().SlotInAccessList(any, any).AnyTimes()
	state.EXPECT().AddSlotToAccessList(any, any).AnyTimes()

	state.EXPECT().GetState(any, any).Return(common.Hash{}).AnyTimes()

	upgrades := opera.GetSonicUpgrades()
	upgrades.GasSubsidies = true
	rules := opera.FakeNetRules(upgrades)

	var updateHeights []opera.UpgradeHeight
	chainConfig := opera.CreateTransientEvmChainConfig(
		rules.NetworkID,
		updateHeights,
		1,
	)

	key, err := crypto.GenerateKey()
	require.NoError(err)

	signer := types.LatestSigner(chainConfig)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		To:  &common.Address{},
		Gas: 21000,
	})

	blockContext := vm.BlockContext{
		BlockNumber: big.NewInt(1),
		BaseFee:     big.NewInt(2),
		Transfer: func(_ vm.StateDB, _ common.Address, _ common.Address, amount *uint256.Int) {
			require.Equal(0, amount.Sign())
		},
		Random: &common.Hash{}, // < signals Revision >= Merge
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := IsCovered(
			registryAddress,
			blockContext,
			signer,
			chainConfig,
			rules,
			state,
			tx,
		)
		require.NoError(err)
	}
}

// TODO: test those cases:
// - registry contract not deployed
// - registry contract deployed with wrong code
// - base fees are correctly considered
// - extra subsidy overhead is correctly considered
// - transaction with nil "to" address
// - transaction with data shorter than 4 bytes
// - transaction with data exactly 4 bytes
// - transaction with data longer than 4 bytes
// - make sure that internal transactions use correct nonces
