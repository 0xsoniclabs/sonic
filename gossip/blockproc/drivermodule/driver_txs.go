// Copyright 2026 Sonic Operations Ltd
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

package drivermodule

import (
	"fmt"
	"io"
	"math"
	"math/big"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/0xsoniclabs/sonic/evmcore/core_types"
	"github.com/0xsoniclabs/sonic/gossip/blockproc"
	"github.com/0xsoniclabs/sonic/gossip/blockproc/subsidies"
	"github.com/0xsoniclabs/sonic/gossip/gasprice"
	"github.com/0xsoniclabs/sonic/inter"
	"github.com/0xsoniclabs/sonic/inter/drivertype"
	"github.com/0xsoniclabs/sonic/inter/iblockproc"
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/0xsoniclabs/sonic/inter/validatorpk"
	"github.com/0xsoniclabs/sonic/opera"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/drivercall"
	"github.com/0xsoniclabs/sonic/opera/contracts/driver/driverpos"
)

const (
	// internalTransactionsGasLimit is the gas limit for internal transactions like epoch sealing. This constant MUST NOT be changed, as doing so will cause a network fork.
	internalTransactionsGasLimit = 500_000_000
	maxAdvanceEpochs             = 1 << 16
)

type DriverTxListenerModule struct{}

func NewDriverTxListenerModule() *DriverTxListenerModule {
	return &DriverTxListenerModule{}
}

func (m *DriverTxListenerModule) Start(block iblockproc.BlockCtx, bs iblockproc.BlockState, es iblockproc.EpochState, statedb state.StateDB) blockproc.TxListener {
	return &DriverTxListener{
		block:   block,
		es:      es,
		bs:      bs,
		statedb: statedb,
	}
}

type DriverTxListener struct {
	block   iblockproc.BlockCtx
	es      iblockproc.EpochState
	bs      iblockproc.BlockState
	statedb state.StateDB
}

type DriverTxTransactor struct{}

type DriverTxPreTransactor struct{}

func NewDriverTxTransactor() *DriverTxTransactor {
	return &DriverTxTransactor{}
}

func NewDriverTxPreTransactor() *DriverTxPreTransactor {
	return &DriverTxPreTransactor{}
}

func InternalTxBuilder(statedb state.StateDB) func(calldata []byte, addr common.Address) *types.Transaction {
	nonce := uint64(math.MaxUint64)
	return func(calldata []byte, addr common.Address) *types.Transaction {
		if nonce == math.MaxUint64 {
			nonce = statedb.GetNonce(common.Address{})
		}
		tx := types.NewTransaction(nonce, addr, common.Big0, internalTransactionsGasLimit, common.Big0, calldata)
		nonce++
		return tx
	}
}

func maxBlockIdx(a, b idx.Block) idx.Block {
	if a > b {
		return a
	}
	return b
}

func (p *DriverTxPreTransactor) PopInternalTxs(block iblockproc.BlockCtx, bs iblockproc.BlockState, es iblockproc.EpochState, sealing bool, statedb state.StateDB) types.Transactions {
	buildTx := InternalTxBuilder(statedb)
	internalTxs := make(types.Transactions, 0, 8)

	// write cheaters
	for _, validatorID := range bs.EpochCheaters[bs.CheatersWritten:] {
		calldata := drivercall.DeactivateValidator(validatorID, drivertype.DoublesignBit)
		internalTxs = append(internalTxs, buildTx(calldata, driver.ContractAddress))
	}

	// push data into Driver before epoch sealing
	if sealing {
		metrics := make([]drivercall.ValidatorEpochMetric, es.Validators.Len())
		for oldValIdx := idx.Validator(0); oldValIdx < es.Validators.Len(); oldValIdx++ {
			info := bs.ValidatorStates[oldValIdx]
			// forgive downtime if below BlockMissedSlack
			missed := opera.BlocksMissed{
				BlocksNum: maxBlockIdx(block.Idx, info.LastBlock) - info.LastBlock,
				Period:    inter.MaxTimestamp(block.Time, info.LastOnlineTime) - info.LastOnlineTime,
			}
			uptime := info.Uptime
			if missed.BlocksNum <= es.Rules.Economy.BlockMissedSlack {
				missed = opera.BlocksMissed{}
				prevOnlineTime := inter.MaxTimestamp(info.LastOnlineTime, es.EpochStart)
				uptime += inter.MaxTimestamp(block.Time, prevOnlineTime) - prevOnlineTime
			}
			metrics[oldValIdx] = drivercall.ValidatorEpochMetric{
				Missed:          missed,
				Uptime:          uptime,
				OriginatedTxFee: info.Originated,
			}
		}
		calldata := drivercall.SealEpoch(metrics)
		internalTxs = append(internalTxs, buildTx(calldata, driver.ContractAddress))
	}
	return internalTxs
}

func (p *DriverTxTransactor) PopInternalTxs(_ iblockproc.BlockCtx, _ iblockproc.BlockState, es iblockproc.EpochState, sealing bool, statedb state.StateDB) types.Transactions {
	buildTx := InternalTxBuilder(statedb)
	internalTxs := make(types.Transactions, 0, 1)
	// push data into Driver after epoch sealing
	if sealing {
		calldata := drivercall.SealEpochValidators(es.Validators.SortedIDs())
		internalTxs = append(internalTxs, buildTx(calldata, driver.ContractAddress))
	}
	return internalTxs
}

func (p *DriverTxListener) OnNewReceipt(tx *types.Transaction, r *types.Receipt, originator idx.ValidatorID, baseFee *big.Int, blobBaseFee *big.Int) {
	if p.es.Rules.Upgrades.Brio {
		p.onNewReceiptPostBrio(originator, tx, r)
	} else {
		p.onNewReceiptPreBrio(tx, r, originator, baseFee, blobBaseFee)
	}
}

func (p *DriverTxListener) onNewReceiptPreBrio(tx *types.Transaction, r *types.Receipt, originator idx.ValidatorID, baseFee *big.Int, blobBaseFee *big.Int) {
	if originator == 0 {
		return
	}
	originatorIdx := p.es.Validators.GetIdx(originator)

	// track originated fee
	var gasPrice *big.Int
	if p.es.Rules.Upgrades.Allegro {
		gasPrice = effectiveGasPrice(tx, baseFee)
	} else {
		gasPrice = tx.GasPrice()
	}
	txFee := new(big.Int).Mul(new(big.Int).SetUint64(r.GasUsed), gasPrice)

	if r.BlobGasUsed != 0 && blobBaseFee != nil {
		blobFee := new(big.Int).SetUint64(r.BlobGasUsed)
		blobFee.Mul(blobFee, blobBaseFee)
		txFee.Add(txFee, blobFee)
	}

	originated := p.bs.ValidatorStates[originatorIdx].Originated
	originated.Add(originated, txFee)

	// track gas power refunds
	notUsedGas := tx.Gas() - r.GasUsed
	if notUsedGas != 0 {
		p.bs.ValidatorStates[originatorIdx].DirtyGasRefund += notUsedGas
	}
}

func effectiveGasPrice(tx *types.Transaction, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return tx.GasPrice()
	}

	// To ensure backwards compatibility the gas tip calculation has to be
	// preserved in its current form. In case of an error due to a negative
	// result, the computed tip still needs to be used as reported by the
	// function for backward compatibility with previous client versions.
	gasTip, _ := gasprice.EffectiveGasTip(tx, baseFee)
	return new(big.Int).Add(baseFee, gasTip)
}

// onNewReceiptPostBrio is called for every transaction accepted in a block,
// providing the ID of the validator that has proposed the transaction in one of
// its events, the accepted transaction, and its receipt. It is used to keep
// track of validator-originated transaction fees.
//
// This function is an updated version of the `onNewReceiptPreBrio` function
// above, which is getting enabled with the Brio flag. Unlike the previous
// version, this function correctly accounts for fees of sponsored and bundled
// transactions. The old code is preserved for backward compatibility until the
// Brio hard-fork and to support full-history syncs.
func (p *DriverTxListener) onNewReceiptPostBrio(
	originator idx.ValidatorID,
	tx *types.Transaction,
	r *types.Receipt,
) {
	p.onNewReceiptPostBrioInternal(originator, tx, r, log.Root())
}

func (p *DriverTxListener) onNewReceiptPostBrioInternal(
	originator idx.ValidatorID,
	tx *types.Transaction,
	r *types.Receipt,
	log log.Logger,
) {
	fee, err := ComputeEffectiveFee(tx, r)
	if err != nil {
		// If there is an error in the fee computation, the safe default
		// is to avoid attributing it to any validator, so no fees are paid out.
		if r == nil {
			log.Warn("error in fee computation", "tx", tx.Hash(), "err", err)
		} else {
			log.Warn("error in fee computation", "tx", tx.Hash(),
				"usedGas", r.GasUsed, "gasPrice", r.EffectiveGasPrice,
				"blobGasUsed", r.BlobGasUsed, "blobGasPrice", r.BlobGasPrice,
				"err", err,
			)
		}
		return
	}

	var blockNumber uint64
	if r != nil && r.BlockNumber != nil {
		blockNumber = r.BlockNumber.Uint64()
	}

	if originator == 0 {
		log.Warn("failed to attribute transaction to validator, fees got burned", "tx", tx.Hash(), "block", blockNumber, "fees", fee)
		return
	}

	originatorIdx := p.es.Validators.GetIdx(originator)
	originated := p.bs.ValidatorStates[originatorIdx].Originated
	originated.Add(originated, fee)
}

// ComputeEffectiveFee returns the effective fee charged for the given
// transaction and its receipt. For regular transactions, this is simply
// gasUsed * effectiveGasPrice + blobGasUsed * blobGasPrice. For fee charge
// transactions used to charge the fees of sponsored transactions, the fee is
// extracted from the transaction input data. The function returns an error if
// the effective price could not be determined, for example due to missing
// receipt data.
func ComputeEffectiveFee(
	tx *types.Transaction,
	r *types.Receipt,
) (*big.Int, error) {
	// pre-checks
	if r == nil {
		return nil, fmt.Errorf("missing receipt")
	}

	// Special case: fee charge transactions have their effective fee specified
	// in the input data, and not in the receipt gas used and gas price fields.
	// The fees only get charged if the transaction succeeded.
	if subsidies.IsFeeChargeTransaction(tx) && r.Status == types.ReceiptStatusSuccessful {
		if fee, err := subsidies.ParseFeeChargeAmount(tx); err == nil {
			return fee.ToBig(), nil
		}
	}

	// Track transactions for network sponsored transactions with tracking have
	// gasPrice=0, so the effective fee is naturally zero. No special case is
	// needed; they fall through to the calculation below.

	if r.EffectiveGasPrice == nil {
		return nil, fmt.Errorf("missing effective gas price in receipt")
	}
	if r.BlobGasUsed > 0 && r.BlobGasPrice == nil {
		return nil, fmt.Errorf("missing blob gas price in receipt")
	}

	gasFee := new(big.Int).Mul(
		new(big.Int).SetUint64(r.GasUsed),
		r.EffectiveGasPrice,
	)

	blobGasFee := big.NewInt(0)
	if r.BlobGasUsed > 0 {
		blobGasFee = new(big.Int).Mul(
			new(big.Int).SetUint64(r.BlobGasUsed),
			r.BlobGasPrice,
		)
	}

	return new(big.Int).Add(gasFee, blobGasFee), nil
}

func decodeDataBytes(l *core_types.Log) ([]byte, error) {
	if len(l.Data) < 32 {
		return nil, io.ErrUnexpectedEOF
	}
	start := new(big.Int).SetBytes(l.Data[24:32]).Uint64()
	if start+32 > uint64(len(l.Data)) {
		return nil, io.ErrUnexpectedEOF
	}
	size := new(big.Int).SetBytes(l.Data[start+24 : start+32]).Uint64()
	if start+32+size > uint64(len(l.Data)) {
		return nil, io.ErrUnexpectedEOF
	}
	return l.Data[start+32 : start+32+size], nil
}

func (p *DriverTxListener) OnNewLog(l *core_types.Log) {
	if l.Address != driver.ContractAddress {
		return
	}
	// Track validator weight changes
	if l.Topics[0] == driverpos.Topics.UpdateValidatorWeight && len(l.Topics) > 1 && len(l.Data) >= 32 {
		validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())
		weight := new(big.Int).SetBytes(l.Data[0:32])

		if weight.Sign() == 0 {
			delete(p.bs.NextValidatorProfiles, validatorID)
		} else {
			profile, ok := p.bs.NextValidatorProfiles[validatorID]
			if !ok {
				profile.PubKey = validatorpk.PubKey{
					Type: 0,
					Raw:  []byte{},
				}
			}
			profile.Weight = weight
			p.bs.NextValidatorProfiles[validatorID] = profile
		}
	}
	// Track validator pubkey changes
	if l.Topics[0] == driverpos.Topics.UpdateValidatorPubkey && len(l.Topics) > 1 {
		validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())
		pubkey, err := decodeDataBytes(l)
		if err != nil {
			log.Warn("Malformed UpdatedValidatorPubkey Driver event")
			return
		}

		profile, ok := p.bs.NextValidatorProfiles[validatorID]
		if !ok {
			log.Warn("Unexpected UpdatedValidatorPubkey Driver event")
			return
		}
		profile.PubKey, _ = validatorpk.FromBytes(pubkey)
		p.bs.NextValidatorProfiles[validatorID] = profile
	}
	// Update rules
	if l.Topics[0] == driverpos.Topics.UpdateNetworkRules && len(l.Data) >= 64 {
		diff, err := decodeDataBytes(l)
		if err != nil {
			log.Warn("Malformed UpdateNetworkRules Driver event")
			return
		}

		last := &p.es.Rules
		if p.bs.DirtyRules != nil {
			last = p.bs.DirtyRules
		}
		updated, err := opera.UpdateRules(*last, diff)
		if err != nil {
			log.Warn("Network rules update error", "err", err)
			return
		}
		p.bs.DirtyRules = &updated
	}
	// Advance epochs
	if l.Topics[0] == driverpos.Topics.AdvanceEpochs && len(l.Data) >= 32 {
		// epochsNum < 2^24 to avoid overflow
		epochsNum := new(big.Int).SetBytes(l.Data[29:32]).Uint64()

		p.bs.AdvanceEpochs += idx.Epoch(epochsNum)
		if p.bs.AdvanceEpochs > maxAdvanceEpochs {
			p.bs.AdvanceEpochs = maxAdvanceEpochs
		}
	}
}

func (p *DriverTxListener) Update(bs iblockproc.BlockState, es iblockproc.EpochState) {
	p.bs, p.es = bs, es
}

func (p *DriverTxListener) Finalize() iblockproc.BlockState {
	return p.bs
}
