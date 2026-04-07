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

// Package ethapi implements the general Ethereum API functions.
package ethapi

import (
	"context"
	"fmt"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/0xsoniclabs/sonic/api/backend"
	"github.com/0xsoniclabs/sonic/opera"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
//
//go:generate mockgen -source=backend.go -destination=backend_mock.go -package=ethapi
type Backend interface {
	backend.Backend
}

// GetVmConfig is a utility function resolving the VM configuration for a block
// height based on the network rules.
func GetVmConfig(
	ctx context.Context,
	backend Backend,
	blockHeight idx.Block,
) (vm.Config, error) {
	rules, err := backend.GetNetworkRules(ctx, blockHeight)
	if err != nil {
		return vm.Config{}, err
	}
	if rules == nil {
		return vm.Config{}, fmt.Errorf("no network rules found for block height %d", blockHeight)
	}
	return opera.GetVmConfig(*rules), nil
}
