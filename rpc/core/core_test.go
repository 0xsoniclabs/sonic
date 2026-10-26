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
package core

import (
	"reflect"
	"slices"
	"testing"
)

// TestCore_MethodSetIsFrozen lists the methods of Core. A change to the
// interface changes this list in the same commit, and a method that mutates
// the node other than SendTx is refused.
func TestCore_MethodSetIsFrozen(t *testing.T) {
	want := []string{
		"Block", "BlockNumberByHash", "BundleInfo", "ChainInfo", "FindLogs",
		"Head", "Header", "OracleInputs", "PendingHeads", "PoolContent",
		"PoolContentFrom", "PoolNonce", "PoolStats", "PoolTransaction",
		"PoolTransactions", "Progress", "Receipts", "Rules", "SendTx", "State",
		"SubscribeHead", "SubscribeTxs", "TxByHash",
	}
	typ := reflect.TypeFor[Core]()
	var got []string
	for i := range typ.NumMethod() {
		got = append(got, typ.Method(i).Name)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("Core methods changed:\n got %v\nwant %v", got, want)
	}
}

// TestCore_SendTxIsTheOnlySetter refuses methods whose names say they change
// the node.
func TestCore_SendTxIsTheOnlySetter(t *testing.T) {
	typ := reflect.TypeFor[Core]()
	for i := range typ.NumMethod() {
		name := typ.Method(i).Name
		for _, verb := range []string{"Set", "Add", "Remove", "Delete", "Put", "Write", "Commit", "Apply", "Send"} {
			if len(name) >= len(verb) && name[:len(verb)] == verb && name != "SendTx" {
				t.Errorf("Core method %s mutates the node; only SendTx may", name)
			}
		}
	}
}

// TestCore_RolesComposeCore checks that the role interfaces make up Core and
// share no method, so that each method has one role.
func TestCore_RolesComposeCore(t *testing.T) {
	roles := []reflect.Type{
		reflect.TypeFor[ChainReader](), reflect.TypeFor[HeadSource](), reflect.TypeFor[StateProvider](),
		reflect.TypeFor[Pool](), reflect.TypeFor[LogIndex](), reflect.TypeFor[NodeInfo](),
	}
	seen := map[string]string{}
	total := 0
	for _, role := range roles {
		for i := range role.NumMethod() {
			name := role.Method(i).Name
			if other, dup := seen[name]; dup {
				t.Errorf("method %s in both %s and %s", name, other, role.Name())
			}
			seen[name] = role.Name()
			total++
		}
	}
	if core := reflect.TypeFor[Core]().NumMethod(); total != core {
		t.Errorf("roles hold %d methods, Core %d", total, core)
	}
}
