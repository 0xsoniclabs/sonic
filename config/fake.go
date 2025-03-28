package config

import (
	"crypto/ecdsa"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xsoniclabs/consensus/consensus"

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

func ParseFakeGen(s string) (id consensus.ValidatorID, num consensus.ValidatorIndex, err error) {
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
	id = consensus.ValidatorID(u32)

	u32, err = strconv.ParseUint(parts[1], 10, 32)
	num = consensus.ValidatorIndex(u32)
	if consensus.ValidatorIndex(id) > num {
		err = fmt.Errorf("key-num should be in range from 1 to validators (<key-num>/<validators>), or should be zero for non-validator node")
		return
	}

	return
}
