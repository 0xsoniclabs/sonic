package tests

import (
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestMaxEmptyBlockPeriodIsEnforced(t *testing.T) {
	hardFork := map[string]opera.Upgrades{
		"sonic":   opera.GetSonicUpgrades(),
		"allegro": opera.GetAllegroUpgrades(),
	}
	modes := map[string]bool{
		"single proposer":      true,
		"distributed proposer": false,
	}

	for name, upgrades := range hardFork {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for mode, singleProposer := range modes {
				upgrades := upgrades
				upgrades.SingleProposerBlockFormation = singleProposer
				t.Run(mode, func(t *testing.T) {
					t.Parallel()
					testMaxEmptyBlockPeriodIsEnforced(t, upgrades)
				})
			}
		})
	}
}

func testMaxEmptyBlockPeriodIsEnforced(
	t *testing.T,
	upgrades opera.Upgrades,
) {
	require := require.New(t)
	net := StartIntegrationTestNet(t, IntegrationTestNetOptions{
		Upgrades: &upgrades,
	})

	maxEmptyInterval := 4 * time.Second

	rules := getNetworkRules(t, net)
	rules.Blocks.MaxEmptyBlockSkipPeriod = inter.Timestamp(maxEmptyInterval)
	updateNetworkRules(t, net, rules)

	rules = getNetworkRules(t, net)
	require.Equal(inter.Timestamp(maxEmptyInterval), rules.Blocks.MaxEmptyBlockSkipPeriod)

	client, err := net.GetClient()
	require.NoError(err)
	defer client.Close()

	start, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// wait for a few empty blocks to be created; these empty blocks should
	// be produced every maxEmptyInterval seconds
	time.Sleep(5 * maxEmptyInterval)

	end, err := client.BlockNumber(t.Context())
	require.NoError(err)

	// there should be a few empty blocks
	require.Greater(end-start, uint64(3))

	getUnixTime := func(header *types.Header) time.Time {
		t.Helper()
		nanos, _, err := inter.DecodeExtraData(header.Extra)
		require.NoError(err)
		return time.Unix(int64(header.Time), int64(nanos))
	}

	var last time.Time
	for i := start + 1; i <= end; i++ {
		block, err := client.BlockByNumber(t.Context(), big.NewInt(int64(i)))
		require.NoError(err)

		// check if the block is empty
		require.Equal(0, len(block.Transactions()), "Block %d should be empty", i)

		blockTime := getUnixTime(block.Header())
		if last != (time.Time{}) {
			delay := blockTime.Sub(last)
			require.Less(maxEmptyInterval, delay)
			require.Less(delay, maxEmptyInterval+time.Second)
		}
		last = blockTime
	}
}
