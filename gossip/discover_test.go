package gossip

import (
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/0xsoniclabs/sonic/evmcore"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
)

// TestStartENRUpdater_ShutdownWaitBlocksUntilLoopStoppedReadingTheStore checks
// that the store can be safely closed once the function returned by
// StartENRUpdater has returned. Run with -race, this test fails if the updater
// loop is still accessing the store during or after the store is closed.
func TestStartENRUpdater_ShutdownWaitBlocksUntilLoopStoppedReadingTheStore(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 1)
	svc := &Service{store: store}
	svc.feed.Start(store.evm)

	waitForShutdown := StartENRUpdater(svc, newTestLocalNode(t))

	// Keep the updater busy with head notifications while the shutdown runs, so
	// that the loop is likely to be in the middle of a store access when the
	// feed is stopped.
	stopFeeding := make(chan struct{})
	feederDone := make(chan struct{})
	go func() {
		defer close(feederDone)
		for i := 1; ; i++ {
			select {
			case <-stopFeeding:
				return
			default:
			}
			svc.feed.newBlock.Send(evmcore.ChainHeadNotify{
				Block: &evmcore.EvmBlock{
					EvmHeader: evmcore.EvmHeader{
						Number: big.NewInt(int64(i)),
						Time:   inter.Timestamp(i),
					},
				},
			})
		}
	}()
	time.Sleep(10 * time.Millisecond)

	svc.feed.Stop() // < closes the subscription scope, terminating the updater
	waitForShutdown()

	close(stopFeeding)
	<-feederDone

	// From here on no goroutine may touch the store any more; it is closed by
	// the clean-up registered by newInMemoryStoreWithGenesisData.
}

// TestStartENRUpdater_ShutdownWaitReturnsIfNoEventsAreProduced checks that the
// updater loop terminates on an idle chain.
func TestStartENRUpdater_ShutdownWaitReturnsIfNoEventsAreProduced(t *testing.T) {
	store := newInMemoryStoreWithGenesisData(t, opera.GetSonicUpgrades(), 1, 1)
	svc := &Service{store: store}
	svc.feed.Start(store.evm)

	waitForShutdown := StartENRUpdater(svc, newTestLocalNode(t))
	svc.feed.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		waitForShutdown()
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("ENR updater did not terminate after the feed was stopped")
	}
}

func newTestLocalNode(t *testing.T) *enode.LocalNode {
	t.Helper()
	db, err := enode.OpenDB("")
	require.NoError(t, err)
	t.Cleanup(db.Close)

	var key *ecdsa.PrivateKey
	key, err = crypto.GenerateKey()
	require.NoError(t, err)

	return enode.NewLocalNode(db, key)
}
