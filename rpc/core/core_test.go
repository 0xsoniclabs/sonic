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
	"strings"
	"testing"
)

// frozenMethods is the surface of Core. Core grows only when a method proves
// it cannot be served from what is there; a change here is a decision of the
// plan, recorded with it.
var frozenMethods = []string{
	"Block",
	"BlockNumberByHash",
	"BundleInfo",
	"ChainInfo",
	"FindLogs",
	"Head",
	"Header",
	"OracleInputs",
	"PendingHeads",
	"PoolContent",
	"PoolContentFrom",
	"PoolNonce",
	"PoolStats",
	"PoolTransaction",
	"PoolTransactions",
	"Progress",
	"Receipts",
	"Rules",
	"SendTransaction",
	"State",
	"SubscribeHead",
	"SubscribeTransactions",
	"TransactionByHash",
}

func TestCore_MethodSetIsFrozen(t *testing.T) {
	coreType := reflect.TypeFor[Core]()
	var methods []string
	for index := range coreType.NumMethod() {
		methods = append(methods, coreType.Method(index).Name)
	}
	if !slices.Equal(methods, frozenMethods) {
		t.Errorf("Core has methods %v, want %v", methods, frozenMethods)
	}
}

func TestCore_NoMethodMutatesButSendTransaction(t *testing.T) {
	mutating := []string{"Send", "Set", "Add", "Put", "Write", "Delete", "Remove", "Commit", "Apply", "Update", "Insert"}
	coreType := reflect.TypeFor[Core]()
	for index := range coreType.NumMethod() {
		name := coreType.Method(index).Name
		if name == "SendTransaction" {
			continue
		}
		for _, prefix := range mutating {
			if strings.HasPrefix(name, prefix) {
				t.Errorf("Core method %s mutates the node", name)
			}
		}
	}
}

func TestCore_EveryRoleIsPartOfCore(t *testing.T) {
	coreType := reflect.TypeFor[Core]()
	roles := []reflect.Type{
		reflect.TypeFor[ChainReader](),
		reflect.TypeFor[HeadSource](),
		reflect.TypeFor[StateProvider](),
		reflect.TypeFor[Pool](),
		reflect.TypeFor[LogIndex](),
		reflect.TypeFor[NodeInfo](),
	}
	total := 0
	for _, role := range roles {
		if !coreType.Implements(role) {
			t.Errorf("Core does not embed %s", role.Name())
		}
		total += role.NumMethod()
	}
	if total != coreType.NumMethod() {
		t.Errorf("the roles hold %d methods, Core %d: a method of Core belongs to no role", total, coreType.NumMethod())
	}
}
