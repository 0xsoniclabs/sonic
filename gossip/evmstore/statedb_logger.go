package evmstore

import (
	"math/big"

	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

func WrapStateDbWithLogger(stateDb state.StateDB, logger *tracing.Hooks) state.StateDB {
	return &LoggingStateDB{
		stateDb,
		logger,
		make(map[common.Address]struct{}),
	}
}

type LoggingStateDB struct {
	state.StateDB
	logger         *tracing.Hooks
	selfDestructed map[common.Address]struct{}
}

func (l *LoggingStateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := l.StateDB.AddBalance(addr, amount, reason)
	if l.logger.OnBalanceChange != nil && !amount.IsZero() {
		l.logger.OnBalanceChange(addr, prev.ToBig(), l.GetBalance(addr).ToBig(), reason)
	}
	return prev
}

func (l *LoggingStateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := l.StateDB.SubBalance(addr, amount, reason)
	if l.logger.OnBalanceChange != nil && !amount.IsZero() {
		l.logger.OnBalanceChange(addr, prev.ToBig(), l.GetBalance(addr).ToBig(), reason)
	}
	return prev
}

func (l *LoggingStateDB) SetCode(addr common.Address, code []byte) []byte {
	prevCodeHash := l.GetCodeHash(addr)
	prevCode := l.StateDB.SetCode(addr, code)
	if l.logger.OnCodeChange != nil {
		l.logger.OnCodeChange(addr, prevCodeHash, prevCode, l.GetCodeHash(addr), code)
	}
	return prevCode
}

func (l *LoggingStateDB) SetNonce(addr common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	if l.logger.OnNonceChange != nil {
		prev := l.GetNonce(addr)
		l.logger.OnNonceChange(addr, prev, nonce)
	}
	l.StateDB.SetNonce(addr, nonce, reason)
}

func (l *LoggingStateDB) SetState(addr common.Address, slot common.Hash, value common.Hash) common.Hash {
	prev := l.StateDB.SetState(addr, slot, value)
	if l.logger.OnStorageChange != nil {
		l.logger.OnStorageChange(addr, slot, prev, value)
	}
	return prev
}

func (l *LoggingStateDB) AddLog(log *types.Log) {
	if l.logger.OnLog != nil {
		l.logger.OnLog(log)
	}
	l.StateDB.AddLog(log)
}

func (l *LoggingStateDB) SelfDestruct(addr common.Address) uint256.Int {
	if l.logger.OnBalanceChange != nil {
		prev := l.GetBalance(addr)
		if prev.Sign() > 0 {
			l.logger.OnBalanceChange(addr, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
		l.selfDestructed[addr] = struct{}{}
	}
	return l.StateDB.SelfDestruct(addr)
}

func (l *LoggingStateDB) EndTransaction() {
	// If tokens were sent to account post-selfdestruct it is burnt.
	if l.logger.OnBalanceChange != nil {
		for addr := range l.selfDestructed {
			if l.HasSelfDestructed(addr) {
				prev := l.GetBalance(addr)
				l.logger.OnBalanceChange(addr, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
			}
		}
		l.selfDestructed = make(map[common.Address]struct{})
	}
	l.StateDB.EndTransaction()
}
