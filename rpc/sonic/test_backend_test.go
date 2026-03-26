package sonic

import (
	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/ethapi"
	testbackend "github.com/0xsoniclabs/sonic/rpc/test_backend"
	"github.com/ethereum/go-ethereum/common"
)

func Test_TestBackend(t *testing.T) {

	chain := testbackend.NewBlockchain()
	chain.SetAccount(common.Address{0x13}, &testbackend.AccountState{
		Nonce:   1,
		Balance: big.NewInt(1000),
	})

	api := ethapi.NewPublicBundleAPI(chain)

	api.PrepareBundle(t.Context(),
		ethapi.PrepareBundleArgs{})
}
