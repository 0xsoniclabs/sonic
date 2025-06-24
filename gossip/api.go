// Copyright 2025 Sonic Operations Ltd
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// PublicEthereumAPI provides an API to access Ethereum-like information.
// It is a github.com/ethereum/go-ethereum/eth simulation for console.
type PublicEthereumAPI struct {
	s *Service
}

// NewPublicEthereumAPI creates a new Ethereum protocol API for gossip.
func NewPublicEthereumAPI(s *Service) *PublicEthereumAPI {
	return &PublicEthereumAPI{s}
}

// Etherbase returns the zero address for web3 compatibility
func (api *PublicEthereumAPI) Etherbase() (common.Address, error) {
	return common.Address{}, nil
}

// Coinbase returns the zero address for web3 compatibility
func (api *PublicEthereumAPI) Coinbase() (common.Address, error) {
	return common.Address{}, nil
}

// Hashrate returns the zero POW hashrate for web3 compatibility
func (api *PublicEthereumAPI) Hashrate() hexutil.Uint64 {
	return hexutil.Uint64(0)
}

// ChainId is the EIP-155 replay-protection chain id for the current ethereum chain config.
func (api *PublicEthereumAPI) ChainId() hexutil.Uint64 {
	return hexutil.Uint64(api.s.store.GetRules().NetworkID)
}
