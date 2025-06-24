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

package config

import (
	"crypto/ecdsa"
	"fmt"
	"strconv"
	"strings"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	cli "gopkg.in/urfave/cli.v1"

	"github.com/0xsoniclabs/sonic/integration/makefakegenesis"
)

// FakeNetFlag enables special testnet, where validators are automatically created
var FakeNetFlag = cli.StringFlag{
	Name:  "fakenet",
	Usage: "'n/N' - sets coinbase as fake n-th key from genesis of N validators.",
}

func getFakeValidatorKey(ctx *cli.Context) *ecdsa.PrivateKey {
	id, _, err := ParseFakeGen(ctx.GlobalString(FakeNetFlag.Name))
	if err != nil || id == 0 {
		return nil
	}
	return makefakegenesis.FakeKey(id)
}

func ParseFakeGen(s string) (id idx.ValidatorID, num idx.Validator, err error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		err = fmt.Errorf("use %%d/%%d format")
		return
	}

	var u32 uint64
	u32, err = strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return
	}
	id = idx.ValidatorID(u32)

	u32, err = strconv.ParseUint(parts[1], 10, 32)
	num = idx.Validator(u32)
	if idx.Validator(id) > num {
		err = fmt.Errorf("key-num should be in range from 1 to validators (<key-num>/<validators>), or should be zero for non-validator node")
		return
	}

	return
}
