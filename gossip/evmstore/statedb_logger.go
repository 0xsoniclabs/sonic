package evmstore

import (
	"github.com/0xsoniclabs/sonic/inter/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"math/big"
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
	logger *tracing.Hooks
	selfDestructed map[common.Address]struct{}
}

func (l *LoggingStateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	prev := l.StateDB.GetBalance(addr)
	l.StateDB.AddBalance(addr, amount, reason)
	if l.logger.OnBalanceChange != nil && !amount.IsZero() {
		l.logger.OnBalanceChange(addr, prev.ToBig(), l.StateDB.GetBalance(addr).ToBig(), reason)
	}
}

func (l *LoggingStateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) {
	prev := l.StateDB.GetBalance(addr)
	l.StateDB.SubBalance(addr, amount, reason)
	if l.logger.OnBalanceChange != nil && !amount.IsZero() {
		l.logger.OnBalanceChange(addr, prev.ToBig(), l.StateDB.GetBalance(addr).ToBig(), reason)
	}
}

func (l *LoggingStateDB) SetCode(addr common.Address, code []byte) {
	prevCode := l.StateDB.GetCode(addr)
	prevCodeHash := l.StateDB.GetCodeHash(addr)
	l.StateDB.SetCode(addr, code)
	if l.logger.OnCodeChange != nil {
		l.logger.OnCodeChange(addr, prevCodeHash, prevCode, l.StateDB.GetCodeHash(addr), code)
	}
}

func (l *LoggingStateDB) SetNonce(addr common.Address, nonce uint64) {
	if l.logger.OnNonceChange != nil {
		prev := l.StateDB.GetNonce(addr)
		l.logger.OnNonceChange(addr, prev, nonce)
	}
	l.StateDB.SetNonce(addr, nonce)
}

func (l *LoggingStateDB) SetState(addr common.Address, slot common.Hash, value common.Hash) {
	if l.logger.OnStorageChange != nil {
		prev := l.StateDB.GetState(addr, slot)
		l.logger.OnStorageChange(addr, slot, prev, value)
	}
	l.StateDB.SetState(addr, slot, value)
}

func (l *LoggingStateDB) AddLog(log *types.Log) {
	if l.logger.OnLog != nil {
		l.logger.OnLog(log)
	}
	l.StateDB.AddLog(log)
}

func (l *LoggingStateDB) SelfDestruct(addr common.Address) {
	if l.logger.OnBalanceChange != nil {
		prev := l.StateDB.GetBalance(addr)
		if prev.Sign() > 0 {
			l.logger.OnBalanceChange(addr, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
		l.selfDestructed[addr] = struct{}{}
	}
	l.StateDB.SelfDestruct(addr)
}

func (l *LoggingStateDB) Finalise() {
	// If tokens were sent to account post-selfdestruct it is burnt.
	if l.logger.OnBalanceChange != nil {
		for addr := range l.selfDestructed {
			if l.HasSelfDestructed(addr) {
				prev := l.StateDB.GetBalance(addr)
				l.logger.OnBalanceChange(addr, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
			}
		}
		l.selfDestructed = make(map[common.Address]struct{})
	}
	l.StateDB.Finalise()
}
