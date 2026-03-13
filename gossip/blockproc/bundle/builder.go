package bundle

import (
	"crypto/ecdsa"
	big "math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func Step(key *ecdsa.PrivateKey, tx types.TxData) step {
	return step{key: key, tx: tx}
}

func Nested(
	key *ecdsa.PrivateKey,
	tx *types.Transaction,
) step {
	// TODO: check that the tx is a valid bundle transaction;
	return step{
		key: key,
		tx: &types.AccessListTx{
			ChainID:    tx.ChainId(),
			Nonce:      tx.Nonce(),
			GasPrice:   tx.GasPrice(),
			Gas:        tx.Gas(),
			To:         tx.To(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
		},
	}
}

func NewAllOf(steps ...step) *types.Transaction {
	return NewBundle(AllOf, steps...)
}

func NewOneOf(steps ...step) *types.Transaction {
	return NewBundle(OneOf, steps...)
}

func NewBundle(
	flags ExecutionFlag,
	steps ...step,
) *types.Transaction {

	// Get chain ID from transactions, if any.
	var chainId *big.Int
	for _, step := range steps {
		tx := types.NewTx(step.tx)
		if curId := tx.ChainId(); curId != nil && curId.Sign() > 0 {
			chainId = curId
			break
		}
	}

	if chainId == nil {
		chainId = big.NewInt(1)
	}

	// Create an Execution Plan for the bundle.
	signer := types.LatestSignerForChainID(chainId)

	plan := ExecutionPlan{
		Steps: make([]ExecutionStep, len(steps)),
		Flags: flags,
	}
	for i, step := range steps {
		plan.Steps[i] = ExecutionStep{
			From: crypto.PubkeyToAddress(step.key.PublicKey),
			Hash: signer.Hash(types.NewTx(step.tx)),
		}
	}

	// Get hash of execution plan and annotate transactions with it.
	execPlanHash := plan.Hash()
	marker := types.AccessTuple{
		Address:     BundleOnly,
		StorageKeys: []common.Hash{execPlanHash},
	}
	for _, step := range steps {
		switch data := step.tx.(type) {
		case *types.DynamicFeeTx:
			data.AccessList = append(data.AccessList, marker)
		case *types.AccessListTx:
			data.AccessList = append(data.AccessList, marker)
		}
	}

	// Sign the modified TxData instances.
	txs := make([]*types.Transaction, len(steps))
	for i, step := range steps {
		txs[i] = types.MustSignNewTx(step.key, signer, step.tx)
	}

	// Build the bundle and wrap it in an envelope.
	return NewEnvelope(&TransactionBundle{
		Version: BundleV1,
		Bundle:  txs,
		Flags:   flags,
	})
}

func NewEnvelope(bundle *TransactionBundle) *types.Transaction {

	payload := Encode(*bundle)

	intrinsic, err := core.IntrinsicGas(
		payload,
		nil,   // access list is not used in the envelope transaction
		nil,   // code auth is not used in the bundle transaction
		false, // bundle transaction is not a contract creation
		true,  // is homestead
		true,  // is istanbul
		true,  // is shanghai
	)
	if err != nil {
		panic(err)
	}

	floorDataGas, err := core.FloorDataGas(payload)
	if err != nil {
		panic(err)
	}

	txGasSum := uint64(0)
	for _, tx := range bundle.Bundle {
		txGasSum += tx.Gas()
	}

	gasLimit := max(intrinsic, floorDataGas, txGasSum)

	chainId := big.NewInt(1)
	if len(bundle.Bundle) > 0 {
		chainId = bundle.Bundle[0].ChainId()
	}

	return types.NewTx(&types.AccessListTx{
		ChainID: chainId,
		To:      &BundleProcessor,
		Data:    payload,
		Gas:     gasLimit,
	})
}

type step struct {
	key *ecdsa.PrivateKey
	tx  types.TxData
}
