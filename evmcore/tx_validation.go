package evmcore

import (
	"fmt"
	"math/big"

	"github.com/0xsoniclabs/sonic/gossip/gasprice/gaspricelimits"
	"github.com/0xsoniclabs/sonic/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// validationOptions is a set of options to adjust the validation of transactions
// according to the current state of the transaction pool.
type validationOptions struct {
	istanbul bool // Fork indicator whether we are in the istanbul revision.
	shanghai bool // Fork indicator whether we are in the shanghai revision.

	eip2718 bool // Fork indicator whether we are using EIP-2718 type transactions.
	eip1559 bool // Fork indicator whether we are using EIP-1559 type transactions.
	eip4844 bool // Fork indicator whether we are using EIP-4844 type transactions.
	eip7623 bool // Fork indicator whether we are using EIP-7623 floor gas validation.
	eip7702 bool // Fork indicator whether we are using EIP-7702 set code transactions.

	currentState   TxPoolStateDB // Current state in the blockchain head
	currentMaxGas  uint64        // Current gas limit for transaction caps
	currentBaseFee *big.Int      // Current base fee for transaction caps
	minTip         *big.Int      // Minimum gas tip to enforce for acceptance into the pool

	locals  *accountSet // Set of local transaction to exempt from eviction rules
	isLocal bool        // Whether the transaction came from a local source

	signer types.Signer
}

// validateTx checks whether a transaction is valid according to the current
// options and adheres to some heuristic limits of the local node (price and size).
func validateTx(tx *types.Transaction, opt validationOptions) error {

	// Accept only legacy transactions until EIP-2718/2930 activates.
	// Since both eip-2718 and eip-2930 are activated in the berlin fork
	// (https://blog.ethereum.org/2021/03/08/ethereum-berlin-upgrade-announcement),
	// they can be grouped in a single flag.
	if !opt.eip2718 && tx.Type() != types.LegacyTxType {
		return ErrTxTypeNotSupported
	}
	// Reject dynamic fee transactions until EIP-1559 activates.
	if !opt.eip1559 && tx.Type() == types.DynamicFeeTxType {
		return ErrTxTypeNotSupported
	}
	// Reject blob transactions until EIP-4844 activates or if is already EIP-4844 and they are not empty
	if tx.Type() == types.BlobTxType {
		if !opt.eip4844 {
			return ErrTxTypeNotSupported
		}
		// For now, Sonic only supports Blob transactions without blob data.
		if len(tx.BlobHashes()) > 0 ||
			(tx.BlobTxSidecar() != nil && len(tx.BlobTxSidecar().BlobHashes()) > 0) {
			return ErrTxTypeNotSupported
		}
	}
	// validate EIP-7702 transactions, part of prague revision
	if tx.Type() == types.SetCodeTxType {
		// Check minimum revision
		if !opt.eip7702 {
			return ErrTxTypeNotSupported
		}

		// Check non-empty authorization list
		if len(tx.SetCodeAuthorizations()) == 0 {
			return ErrEmptyAuthorizations
		}
	}

	// Reject transactions over defined size to prevent DOS attacks
	if uint64(tx.Size()) > txMaxSize {
		return ErrOversizedData
	}

	// Check whether the init code size has been exceeded, introduced in EIP-3860
	if opt.shanghai && tx.To() == nil &&
		len(tx.Data()) > params.MaxInitCodeSize {
		return fmt.Errorf("%w: code size %v, limit %v", ErrMaxInitCodeSizeExceeded, len(tx.Data()), params.MaxInitCodeSize)
	}

	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return ErrNegativeValue
	}

	// Ensure the transaction doesn't exceed the current block limit gas.
	if opt.currentMaxGas < tx.Gas() {
		return ErrGasLimit
	}

	// Sanity check for extremely large numbers
	if tx.GasFeeCap().BitLen() > 256 {
		return ErrFeeCapVeryHigh
	}
	if tx.GasTipCap().BitLen() > 256 {
		return ErrTipVeryHigh
	}

	// Ensure gasFeeCap is greater than or equal to gasTipCap.
	if tx.GasFeeCapIntCmp(tx.GasTipCap()) < 0 {
		return ErrTipAboveFeeCap
	}

	// Make sure the transaction is signed properly.
	from, err := types.Sender(opt.signer, tx)
	if err != nil {
		return ErrInvalidSender
	}

	// Drop non-local transactions under our own minimal accepted gas price or tip
	local := opt.isLocal || opt.locals.contains(from) // account may be local even if the transaction arrived from the network
	if !local && tx.GasTipCapIntCmp(opt.minTip) < 0 {
		log.Trace("Rejecting underpriced tx: pool.gasPrice", "pool.gasPrice",
			opt.minTip, "tx.GasTipCap", tx.GasTipCap())
		return ErrUnderpriced
	}
	// Ensure Opera-specific hard bounds
	if baseFee := opt.currentBaseFee; baseFee != nil {
		limit := gaspricelimits.GetMinimumFeeCapForTransactionPool(baseFee)
		if tx.GasFeeCapIntCmp(limit) < 0 {
			log.Trace("Rejecting underpriced tx: minimumBaseFee", "minimumBaseFee", baseFee, "limit", limit, "tx.GasFeeCap", tx.GasFeeCap())
			return ErrUnderpriced
		}
	}

	// Ensure the transaction adheres to nonce ordering
	if opt.currentState.GetNonce(from) > tx.Nonce() {
		return ErrNonceTooLow
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if utils.Uint256ToBigInt(opt.currentState.GetBalance(from)).Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}

	// Ensure the transaction has more gas than the basic tx fee.
	intrGas, err := core.IntrinsicGas(
		tx.Data(),
		tx.AccessList(),
		tx.SetCodeAuthorizations(),
		tx.To() == nil, // is contract creation
		true,           // is homestead
		opt.istanbul,   // is eip-2028 (transactional data gas cost reduction)
		opt.shanghai,   // is eip-3860 (limit and meter init-code )
	)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return ErrIntrinsicGas
	}

	// EIP-7623 part of Prague revision: Floor data gas
	// see: https://eips.ethereum.org/EIPS/eip-7623
	if opt.eip7623 {
		floorDataGas, err := core.FloorDataGas(tx.Data())
		if err != nil {
			return err
		}
		if tx.Gas() < floorDataGas {
			return fmt.Errorf("%w: have %d, want %d", ErrFloorDataGas, tx.Gas(), floorDataGas)
		}
	}

	return nil
}
