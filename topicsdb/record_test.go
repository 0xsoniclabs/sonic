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

package topicsdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_logRecLess_Examples(t *testing.T) {
	tests := map[string]struct {
		a, b ID
		want bool
	}{
		"equal IDs": {
			a:    ID{0x01, 0x02, 0x03},
			b:    ID{0x01, 0x02, 0x03},
			want: false,
		},
		"a < b": {
			a:    ID{0x01, 0x02, 0x03},
			b:    ID{0x01, 0x02, 0x04},
			want: true,
		},
		"a > b": {
			a:    ID{0x01, 0x02, 0x04},
			b:    ID{0x01, 0x02, 0x03},
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := logRecLess(logrec{ID: tc.a}, logrec{ID: tc.b})
			require.Equal(t, tc.want, got)
		})
	}
}
