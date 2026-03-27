package rpctest

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Account struct {
	PrivateKey *ecdsa.PrivateKey
}

func NewAccount() *Account {
	key, _ := crypto.GenerateKey()
	return &Account{
		PrivateKey: key,
	}
}

func (a *Account) Address() *common.Address {
	addr := crypto.PubkeyToAddress(a.PrivateKey.PublicKey)
	return &addr
}

func ToHexUint64(i uint64) *hexutil.Uint64 {
	hu := hexutil.Uint64(i)
	return &hu
}

func ToHexUint(i uint) *hexutil.Uint {
	hu := hexutil.Uint(i)
	return &hu
}

func ToHexBig(i big.Int) *hexutil.Big {
	return (*hexutil.Big)(&i)
}

func ToHexBigInt(i int64) *hexutil.Big {
	hu := hexutil.Big(*big.NewInt(i))
	return &hu
}
