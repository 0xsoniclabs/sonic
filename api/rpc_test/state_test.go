package rpctest

import (
	"github.com/0xsoniclabs/sonic/inter/state"
)

var _ state.StateDB = &testState{}
