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
	"net/http"
	"net/http/httptest"
	"testing"
)

// FuzzEnvelope sends the same body to go-ethereum's server and to the
// module's endpoint and requires the same answer, the wording of a parameter
// the module cannot decode aside. The module never crashes on a body.
//
//	go test -run '^$' -fuzz FuzzEnvelope ./rpc/transport
func FuzzEnvelope(fuzz *testing.F) {
	for _, body := range bodies {
		fuzz.Add(body)
	}
	servers := newEndpoints(&testing.T{}, endpointOptions{})
	fuzz.Cleanup(func() {
		servers.reference.Close()
		servers.module.Close()
	})
	fuzz.Fuzz(func(t *testing.T, body string) {
		build := func(server *httptest.Server) func() *http.Request {
			return jsonRequest(server, body)
		}
		requireSameExchange(t, servers, "fuzzed body", build)
	})
}
