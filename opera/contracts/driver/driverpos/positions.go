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

package driverpos

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Events
var (
	// Topics of Driver contract logs
	Topics = struct {
		UpdateValidatorWeight common.Hash
		UpdateValidatorPubkey common.Hash
		UpdateNetworkRules    common.Hash
		UpdateNetworkVersion  common.Hash
		AdvanceEpochs         common.Hash
	}{
		UpdateValidatorWeight: crypto.Keccak256Hash([]byte("UpdateValidatorWeight(uint256,uint256)")),
		UpdateValidatorPubkey: crypto.Keccak256Hash([]byte("UpdateValidatorPubkey(uint256,bytes)")),
		UpdateNetworkRules:    crypto.Keccak256Hash([]byte("UpdateNetworkRules(bytes)")),
		UpdateNetworkVersion:  crypto.Keccak256Hash([]byte("UpdateNetworkVersion(uint256)")),
		AdvanceEpochs:         crypto.Keccak256Hash([]byte("AdvanceEpochs(uint256)")),
	}
)
