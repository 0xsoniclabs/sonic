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

package evmcore

import (
	"testing"

	"github.com/0xsoniclabs/sonic/opera"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"go.uber.org/mock/gomock"
)

func TestBundleIntegration_chainAdapter_ForwardsCallsToChainReader(t *testing.T) {

	t.Run("GetCurrentNetworkRules", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		chain := NewMockStateReader(ctrl)
		adapter := chainAdapter{chain: chain}

		chain.EXPECT().CurrentRules().Return(opera.Rules{})
		adapter.GetCurrentNetworkRules()
	})

	t.Run("GetCurrentChainConfig", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		chain := NewMockStateReader(ctrl)
		adapter := chainAdapter{chain: chain}

		chain.EXPECT().CurrentConfig().Return(&params.ChainConfig{})
		adapter.GetCurrentChainConfig()
	})

	t.Run("GetLatestHeader", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		chain := NewMockStateReader(ctrl)
		adapter := chainAdapter{chain: chain}

		chain.EXPECT().CurrentBlock().Return(&EvmBlock{})
		adapter.GetLatestHeader()
	})

	t.Run("Header", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		chain := NewMockStateReader(ctrl)
		adapter := chainAdapter{chain: chain}

		hash := common.Hash{123}
		number := uint64(456)
		chain.EXPECT().Header(hash, number)
		adapter.Header(hash, number)
	})
}
