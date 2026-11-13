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

package transport

import (
	"strings"
	"testing"
)

// FuzzHTTP_AnswersAsGoEthereumServer feeds any body to the module's HTTP
// handler and to go-ethereum's server and requires the same answer. The slow
// and the large method are left out: one ignores the fuzzer's pace, the
// other would allocate what an input asks.
func FuzzHTTP_AnswersAsGoEthereumServer(f *testing.F) {
	for _, body := range envelopeBodies {
		f.Add(body)
	}
	reference := newReference(f)
	reference.SetBatchLimits(testLimits.Items, testLimits.ResponseBytes)
	module := &httpHandler{front: NewFront(newTestDispatcher(f), nil, "http.api"), limits: testLimits}
	f.Fuzz(func(t *testing.T, body string) {
		if strings.Contains(body, "test_slow") || strings.Contains(body, "test_large") {
			t.Skip()
		}
		want := serveRecorded(reference, body)
		got := serveRecorded(module, body)
		if want.Code != got.Code || want.Body.String() != got.Body.String() {
			t.Fatalf("for %q\nwant %d %s\ngot  %d %s", body, want.Code, want.Body, got.Code, got.Body)
		}
	})
}
