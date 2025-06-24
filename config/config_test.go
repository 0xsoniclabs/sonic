// Copyright 2025 Sonic Operations Ltd
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

package config

import (
	"strings"
	"testing"
)

func TestBootstrapNodes_AreValid(t *testing.T) {

	fakeIp := "1.2.3.4"
	fakeResolver := func(url string) (string, error) {
		return fakeIp, nil
	}

	for name, node := range Bootnodes {
		t.Run(name, func(t *testing.T) {
			for _, url := range node {
				t.Run(url, func(t *testing.T) {
					hostname, modified, err := resolveHostNameInEnodeURLInternal(url, fakeResolver)
					if err != nil {
						t.Fatalf("Failed to resolve hostname in enode URL: %v", err)
					}
					if !strings.Contains(url, hostname) {
						t.Fatalf("Hostname %q not found in URL", hostname)
					}
					if strings.Contains(modified, hostname) {
						t.Fatalf("failed to replace hostname in URL %q", modified)
					}
					if !strings.Contains(modified, fakeIp) {
						t.Fatalf("failed to insert IP in URL %q", modified)
					}
				})
			}
		})
	}
}
