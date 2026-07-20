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

package testapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var methods = map[string]struct {
	call   func(*TestApi, context.Context, [][]byte) error
	expect func(*MockBackend) *gomock.Call
}{
	"ProposeTransactions": {
		call:   (*TestApi).ProposeTransactions,
		expect: func(b *MockBackend) *gomock.Call { return b.EXPECT().ProposeTransactions(gomock.Any()) },
	},
	"AddTransactions": {
		call:   (*TestApi).AddTransactions,
		expect: func(b *MockBackend) *gomock.Call { return b.EXPECT().AddTransactions(gomock.Any()) },
	},
}

func TestTestApi_FailsIfNotEnabled(t *testing.T) {
	for name, m := range methods {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)
			backend.EXPECT().IsTestOnlyApiEnabled().Return(false)

			api := &TestApi{backend: backend}
			err := m.call(api, t.Context(), nil)
			require.ErrorContains(t, err, "test-only API is not enabled")
		})
	}
}

func TestTestApi_FailsOnDecodingError(t *testing.T) {
	for name, m := range methods {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)
			backend.EXPECT().IsTestOnlyApiEnabled().Return(true)

			data := [][]byte{[]byte("not a valid RLP encoding")}

			api := &TestApi{backend: backend}
			err := m.call(api, t.Context(), data)
			require.ErrorContains(t, err, "typed transaction too short")
		})
	}
}

func TestTestApi_ForwardsTransactionsToBackend(t *testing.T) {
	for name, m := range methods {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)
			backend.EXPECT().IsTestOnlyApiEnabled().Return(true)

			tx := types.NewTx(&types.LegacyTx{Nonce: 1})
			txData, err := rlp.EncodeToBytes(tx)
			require.NoError(t, err)

			want := []*types.Transaction{tx}
			data := [][]byte{txData}

			m.expect(backend).DoAndReturn(
				func(got []*types.Transaction) error {
					require.Equal(t, len(want), len(got))
					for i := range want {
						require.Equal(t, want[i].Hash(), got[i].Hash())
					}
					return nil
				},
			)

			api := &TestApi{backend: backend}
			require.NoError(t, m.call(api, t.Context(), data))
		})
	}
}

func TestTestApi_ReturnsEncounteredIssueFromBackend(t *testing.T) {
	for name, m := range methods {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			backend := NewMockBackend(ctrl)
			backend.EXPECT().IsTestOnlyApiEnabled().Return(true)

			issue := fmt.Errorf("introduced test-issue")
			m.expect(backend).Return(issue)

			api := &TestApi{backend: backend}
			require.ErrorIs(t, m.call(api, t.Context(), nil), issue)
		})
	}
}
