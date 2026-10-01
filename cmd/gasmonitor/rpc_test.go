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

package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRedactURL(t *testing.T) {
	tests := map[string]string{
		"https://rpc.soniclabs.com":                "https://rpc.soniclabs.com",
		"https://rpc.soniclabs.com/":               "https://rpc.soniclabs.com",
		"https://rpc.soniclabs.com/secret-key":     "https://rpc.soniclabs.com/***",
		"https://rpc.soniclabs.com?apikey=secret":  "https://rpc.soniclabs.com/***",
		"https://user:secret@rpc.soniclabs.com":    "https://rpc.soniclabs.com/***",
		"http://127.0.0.1:8545/v1/secret?x=secret": "http://127.0.0.1:8545/***",
		"rpc.soniclabs.com/secret":                 "<redacted URL>",
		"://secret":                                "<redacted URL>",
	}
	for raw, want := range tests {
		if got := redactURL(raw); got != want {
			t.Errorf("redactURL(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestRPCClient_ErrorsDoNotContainURLSecret(t *testing.T) {
	// Nothing listens on port 1, so the request fails with a net/http error
	// that would normally include the full URL.
	c := newRPCClient("http://127.0.0.1:1/secret-key?apikey=secret", time.Second)
	_, err := c.blockNumber(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("error leaks the URL secret: %v", err)
	}
	if !strings.Contains(err.Error(), "http://127.0.0.1:1/***") {
		t.Errorf("error does not name the redacted endpoint: %v", err)
	}
}
