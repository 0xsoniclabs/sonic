package verwatcher

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/logger"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/driverpos"
)

type VersionWatcher struct {
	store *Store

	done chan struct{}
	wg   sync.WaitGroup
	logger.Instance
}

func New(store *Store) *VersionWatcher {
	return &VersionWatcher{
		store:    store,
		done:     make(chan struct{}),
		Instance: logger.New(),
	}
}

func (w *VersionWatcher) Pause() error {
	have := getVersionNumber()
	needed := versionNumber(w.store.GetNetworkVersion())
	if needed > have {
		return fmt.Errorf("network upgrade %v was activated. Current node version is %v. "+
			"Please upgrade your node to continue syncing", needed, have)
	} else if w.store.GetMissedVersion() > 0 {
		return fmt.Errorf("node's state is dirty because node was upgraded after the network upgrade %v was activated. "+
			"Please re-sync the chain data to continue", versionNumber(w.store.GetMissedVersion()))
	}
	return nil
}

func (w *VersionWatcher) OnNewLog(l *types.Log) {
	if l.Address != driver.ContractAddress {
		return
	}
	if l.Topics[0] == driverpos.Topics.UpdateNetworkVersion && len(l.Data) >= 32 {
		netVersion := new(big.Int).SetBytes(l.Data[24:32]).Uint64()
		w.store.SetNetworkVersion(netVersion)
		w.log()
	}
}

func (w *VersionWatcher) log() {
	if err := w.Pause(); err != nil {
		w.Log.Warn(err.Error())
	}
}

func (w *VersionWatcher) Start() {
	w.log()
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.log()
			case <-w.done:
				return
			}
		}
	}()
}

func (w *VersionWatcher) Stop() {
	close(w.done)
	w.wg.Wait()
}
