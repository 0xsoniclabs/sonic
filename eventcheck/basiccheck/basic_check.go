package basiccheck

import (
	"errors"
	"math"

	base "github.com/Fantom-foundation/lachesis-base/eventcheck/basiccheck"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/0xsoniclabs/sonic/inter"
)

var (
	ErrWrongNetForkID = errors.New("wrong network fork ID")
	ErrZeroTime       = errors.New("event has zero timestamp")
	ErrNegativeValue  = errors.New("negative value")
	ErrIntrinsicGas   = errors.New("intrinsic gas too low")
	// ErrTipAboveFeeCap is a sanity error to ensure no one is able to specify a
	// transaction with a tip higher than the total fee cap.
	ErrTipAboveFeeCap = errors.New("max priority fee per gas higher than max fee per gas")
)

type Checker struct {
	base base.Checker
}

// New validator which performs checks which don't require anything except event
func New() *Checker {
	return &Checker{
		base: base.Checker{},
	}
}

// validateTx checks whether a transaction is valid according to the consensus
// rules
func validateTx(tx *types.Transaction) error {
	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 || tx.GasPrice().Sign() < 0 {
		return ErrNegativeValue
	}

	// Ensure the transaction has more gas than the basic tx fee.

	// NOTE: the call to intrinsicGas was migrated to use Geth's core package
	// Unfortunately, we do not have the information at this point to determine
	// the enabled revisions a this block height.
	// Transactions are correctly validated in the pool and in the processor,
	// therefore this check is called as the less strict version.
	// This check will be removed in future versions:
	// https://github.com/Fantom-foundation/lachesis-base-sonic/blob/main/eventcheck/basiccheck/basic_check.go
	intrGas, err := core.IntrinsicGas(
		tx.Data(),
		tx.AccessList(),
		tx.SetCodeAuthorizations(),
		tx.To() == nil, // is contract creation
		true,           // is homestead

		// is eip-2028 (transactional data gas cost reduction)
		// enabled to get the lower intrinsic gas cost of both options
		true,

		// is eip-3860 (limit and meter init-code )
		// Disable to get the lower intrinsic gas cost of both options
		false,
	)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return ErrIntrinsicGas
	}

	if tx.GasFeeCapIntCmp(tx.GasTipCap()) < 0 {
		return ErrTipAboveFeeCap
	}
	return nil
}

func (v *Checker) validateTxs(e inter.EventPayloadI) error {
	for _, tx := range e.Txs() {
		if err := validateTx(tx); err != nil {
			return err
		}
	}
	return nil
}

// Validate event
func (v *Checker) Validate(e inter.EventPayloadI) error {
	if e.NetForkID() != 0 {
		return ErrWrongNetForkID
	}
	if err := v.base.Validate(e); err != nil {
		return err
	}
	if e.GasPowerUsed() >= math.MaxInt64-1 || e.GasPowerLeft().Max() >= math.MaxInt64-1 {
		return base.ErrHugeValue
	}
	if e.CreationTime() <= 0 || e.MedianTime() <= 0 {
		return ErrZeroTime
	}
	if err := v.validateTxs(e); err != nil {
		return err
	}

	return nil
}
