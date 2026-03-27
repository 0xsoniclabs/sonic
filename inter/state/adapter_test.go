package state

import (
	"testing"

	"go.uber.org/mock/gomock"
)

func TestMockStateDB_ImplementsInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockStateDB(ctrl)
	var _ StateDB = mock
}
