package gassubsidies

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	gassubsidies_contract "github.com/0xsoniclabs/sonic/tests/gas_subsidies/contract"

	"math/big"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/tests"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestGasSubsidies_DeployContract(t *testing.T) {

	net := tests.StartIntegrationTestNet(t,
		tests.IntegrationTestNetOptions{
			// The contract will be automatically installed for Allegro and later upgrades
			// using the genesis, for this test, Sonic is required.
			Upgrades: tests.AsPointer(opera.GetSonicUpgrades()),
			ModifyConfig: func(config *config.Config) {
				// The transaction to deploy the contract is not replay protected
				// This has the benefit that the same tx will work in both ethereum and sonic.
				// Nevertheless the default RPC configuration rejects this sort of transaction.
				config.Opera.AllowUnprotectedTxs = true
			},
		},
	)

	client, err := net.GetClient()
	require.NoError(t, err)
	defer client.Close()

	// deploy the contract
	opts, err := net.GetTransactOptions(net.GetSessionSponsor())
	require.NoError(t, err)

	// call deploy, to compute init data, this will not send the tx
	opts.NoSend = true
	_, orgTx, _, err := gassubsidies_contract.DeployGassubsidiesContract(opts, client)
	require.NoError(t, err)
	require.NotNil(t, orgTx)

	payload := &types.LegacyTx{
		Nonce:    0, // deployer address is generated from synthetic signature
		Gas:      orgTx.Gas(),
		GasPrice: orgTx.GasPrice(),
		Data:     orgTx.Data(),
		To:       nil, // contract creation
	}

	tx, err := types.NewTx(payload).WithSignature(&SyntheticTransactionSigner{}, make([]byte, 65))
	require.NoError(t, err)

	// validate synthetic signature values
	v, r, _ := tx.RawSignatureValues()
	require.Equal(t, big.NewInt(28), v)
	expectedR, ok := new(big.Int).SetString("539", 16)
	require.True(t, ok)
	require.Equal(t, expectedR, r)

	deployerAddress, err := SyntheticTransactionSigner{}.Sender(tx)
	require.NoError(t, err)

	// fund the deployer account
	cost := new(big.Int).Mul(big.NewInt(int64(tx.Gas())), tx.GasPrice())
	receipt, err := net.EndowAccount(deployerAddress, cost)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	// Deploy the contract
	receipt, err = net.Run(tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

	tests.WaitForProofOf(t, client, int(receipt.BlockNumber.Int64()))

	contractAddress := crypto.CreateAddress(deployerAddress, tx.Nonce())
	code, err := client.CodeAt(t.Context(), contractAddress, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code, "no code was deployed")

	t.Log("Sender address:", deployerAddress.Hex())
	t.Log("Contract address:", contractAddress.Hex())
	t.Log("Tx:", SerializedAsInEIP(t, deployerAddress, tx))
	t.Log("Contract code:", common.Bytes2Hex(code))
}

type SyntheticTransactionSigner struct {
	inner types.HomesteadSigner
}

func (s SyntheticTransactionSigner) Sender(tx *types.Transaction) (common.Address, error) {
	return s.inner.Sender(tx)
}

func (SyntheticTransactionSigner) SignatureValues(tx *types.Transaction, sig []byte) (r, s, v *big.Int, err error) {

	// NOTE: ethereum uses 27 for its system deployed transactions.
	// we use 28 to avoid any sort of confusion with ethereum transactions.
	//
	// Homestead signatures set v to 27 or 28, depending on the parity of the y
	// value of the R point. This is then used to recover the public key
	// from the signature.
	v = big.NewInt(28)

	r = new(big.Int)
	_, ok := r.SetString("539", 16)
	if !ok {
		err = fmt.Errorf("failed to set r")
		return
	}

	s = generate18BytesRandomNumber()
	return
}

func (s SyntheticTransactionSigner) ChainID() *big.Int {
	return nil
}

func (s SyntheticTransactionSigner) Hash(tx *types.Transaction) common.Hash {
	return tx.Hash()
}

func (s SyntheticTransactionSigner) Equal(other types.Signer) bool {
	return false
}

func generate18BytesRandomNumber() *big.Int {
	randomBytes := make([]byte, 18)
	_, _ = rand.Read(randomBytes)
	randomNumber := new(big.Int).SetBytes(randomBytes)
	return randomNumber
}

func SerializedAsInEIP(t testing.TB, from common.Address, tx *types.Transaction) string {

	type TxJson struct {
		Type  string `json:"type"`
		Nonce string `json:"nonce"`
		From  string `json:"from"`
		To    string `json:"to"`

		Gas      string `json:"gas"`
		GasPrice string `json:"gasPrice"`

		Value string `json:"value"`
		Input string `json:"input"`

		V    string `json:"v"`
		R    string `json:"r"`
		S    string `json:"s"`
		Hash string `json:"hash"`
	}

	v, r, s := tx.RawSignatureValues()

	txJson := TxJson{
		Type:     fmt.Sprintf("0x%x", tx.Type()),
		Nonce:    fmt.Sprintf("0x%x", tx.Nonce()),
		From:     from.Hex(),
		Gas:      fmt.Sprintf("0x%x", tx.Gas()),
		GasPrice: fmt.Sprintf("0x%x", tx.GasPrice()),
		Value:    fmt.Sprintf("0x%x", tx.Value()),
		Input:    common.Bytes2Hex(tx.Data()),
		V:        fmt.Sprintf("0x%x", v),
		R:        fmt.Sprintf("0x%x", r),
		S:        fmt.Sprintf("0x%x", s),
		Hash:     tx.Hash().Hex(),
	}

	// serialized and print
	res, err := json.MarshalIndent(txJson, "", "  ")
	require.NoError(t, err)
	return string(res)
}
