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

package gossip

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
	"github.com/stretchr/testify/require"

	"github.com/0xsoniclabs/sonic/gossip/gasprice"
)

// TestDefaultConfig_KeepsTheDefaultsOfTheReference holds O1: the defaults
// the node configuration takes from the RPC module are those of the
// reference (plan, section 8).
func TestDefaultConfig_KeepsTheDefaultsOfTheReference(t *testing.T) {
	config := DefaultConfig(cachescale.Identity)
	require.Equal(t, uint64(50_000_000), config.RPCGasCap)
	require.Equal(t, 5*time.Second, config.RPCEVMTimeout)
	require.Equal(t, 5*time.Second, config.RPCTimeout)
	require.Equal(t, 100.0, config.RPCTxFeeCap)
	require.Equal(t, 25*1024*1024, config.MaxResponseSize)
	require.Equal(t, 2000, config.StructLogLimit)
	require.Equal(t, uint64(1000), config.MaxTraceFilterRange)
	require.Equal(t, 5*time.Second, config.TxSyncDefaultTimeout)
	require.Equal(t, 30*time.Second, config.TxSyncMaxTimeout)
	require.Equal(t, idx.Block(999999999999999999), config.FilterAPI.IndexedLogsBlockRangeLimit)
	require.Equal(t, idx.Block(100), config.FilterAPI.UnindexedLogsBlockRangeLimit)
	require.Equal(t, uint(1000), config.FilterAPI.LogQueryParameterLimit)
	require.Equal(t, uint(5000), config.FilterAPI.LogQueryResultLimit)
	require.Equal(t, gasprice.Config{
		MaxGasPrice:      big.NewInt(10_000_000_000_000_000),
		MinGasPrice:      new(big.Int),
		DefaultCertainty: 500_000,
	}, config.GPO)
}

// TestConfigInstancesAreIndependent instantiates 100 Configs and verifies that
// all pointer, slice, map, or interface fields are not shared among instances.
func TestConfigInstancesAreIndependent(t *testing.T) {

	configA := DefaultConfig(cachescale.Identity)
	configB := DefaultConfig(cachescale.Identity)
	checkNoSharedReferences(t, configA, configB, "")

}

// checkNoSharedReferences recursively checks that no pointer, slice, map, or interface fields are shared.
func checkNoSharedReferences(t *testing.T, a, b interface{}, path string) {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	if va.Kind() == reflect.Pointer || va.Kind() == reflect.Interface {
		if va.IsNil() || vb.IsNil() {
			return
		}
		if va.Pointer() == vb.Pointer() {
			t.Errorf("shared reference at %s", path)
		}
		va = va.Elem()
		vb = vb.Elem()
	}
	if va.Kind() == reflect.Struct {
		for i := 0; i < va.NumField(); i++ {
			fieldA := va.Field(i)
			fieldB := vb.Field(i)
			fieldType := va.Type().Field(i)
			if !fieldA.CanInterface() || !fieldB.CanInterface() {
				continue // skip unexported fields
			}
			checkNoSharedReferences(t, fieldA.Interface(), fieldB.Interface(), path+"."+fieldType.Name)
		}
	}
	if va.Kind() == reflect.Slice || va.Kind() == reflect.Map {
		if va.IsNil() || vb.IsNil() {
			return
		}
		if va.Pointer() == vb.Pointer() {
			t.Errorf("shared reference at %s", path)
		}
	}
}
