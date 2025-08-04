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

package tests

import (
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestEthCall_CodeLargerThanMaxInitCodeSizeIsAccepted(t *testing.T) {
	tests := map[string]struct {
		codeSize int
	}{
		"max code size": {
			params.MaxCodeSize,
		},
		"max code size + 1": {
			params.MaxCodeSize + 1,
		},
		"max code size lfvm": {
			math.MaxUint16, // max code size supported by the LFVM
		},
		"max code size lfvm + 1": {
			math.MaxUint16 + 1,
		},
	}
	net := StartIntegrationTestNet(t)

	client, err := net.GetClient()
	require.NoError(t, err, "Failed to connect to the integration test network")
	defer client.Close()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			accountWithHugeCode := "0x5555555555555555555555555555555555555555"

			txArguments := map[string]string{
				"to":   accountWithHugeCode,
				"gas":  "0xffffffffffffffff",
				"data": "0x00",
			}
			requestedBlock := "latest"
			stateOverrides := map[string]map[string]hexutil.Bytes{
				accountWithHugeCode: {
					"code": make([]byte, test.codeSize),
				},
			}

			var res interface{}
			err = client.Client().Call(&res, "eth_call", txArguments, requestedBlock, stateOverrides)
			require.NoError(t, err)
		})
	}
}
