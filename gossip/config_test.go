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

package gossip

import (
	"reflect"
	"testing"

	"github.com/Fantom-foundation/lachesis-base/utils/cachescale"
)

// TestConfigInstancesAreIndependent instantiates 100 Configs and verifies that
// all pointer, slice, map, or interface fields are not shared among instances.
func TestConfigInstancesAreIndependent(t *testing.T) {
	const n = 100
	configs := make([]Config, n)
	for i := 0; i < n; i++ {
		configs[i] = DefaultConfig(cachescale.Identity)
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			checkNoSharedReferences(t, configs[i], configs[j], "")
		}
	}
}

// checkNoSharedReferences recursively checks that no pointer, slice, map, or interface fields are shared.
func checkNoSharedReferences(t *testing.T, a, b interface{}, path string) {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	if va.Kind() == reflect.Ptr || va.Kind() == reflect.Interface {
		if va.IsNil() || vb.IsNil() {
			return
		}
		if va.Pointer() == vb.Pointer() {
			t.Errorf("shared reference at %s", path)
		}
		va = va.Elem()
		vb = vb.Elem()
	}
	if va.Kind() == reflect.Struct {
		for i := 0; i < va.NumField(); i++ {
			fieldA := va.Field(i)
			fieldB := vb.Field(i)
			fieldType := va.Type().Field(i)
			if !fieldA.CanInterface() || !fieldB.CanInterface() {
				continue // skip unexported fields
			}
			checkNoSharedReferences(t, fieldA.Interface(), fieldB.Interface(), path+"."+fieldType.Name)
		}
	}
	if va.Kind() == reflect.Slice || va.Kind() == reflect.Map {
		if va.IsNil() || vb.IsNil() {
			return
		}
		if va.Pointer() == vb.Pointer() {
			t.Errorf("shared reference at %s", path)
		}
	}
}
