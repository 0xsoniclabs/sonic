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

package config

import (
	"math/big"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The golden file holds the TOML keys of the reference release, v2.2.5, as
// the loader decodes them; it was made by walking its config.Config as
// tomlKeys walks this one. The loader rejects unknown fields, so a key of the
// reference that is gone breaks the configuration files of its operators.
const referenceTomlKeys = "testdata/reference_toml_keys.txt"

// deprecatedTomlKeys are keys of the reference that are still accepted but
// have no effect, each with its reason; none so far.
var deprecatedTomlKeys = map[string]string{}

// tomlKeys returns the dotted TOML keys a type decodes, by the field names
// TomlSettings uses.
func tomlKeys(prefix string, kind reflect.Type) []string {
	for kind.Kind() == reflect.Pointer {
		kind = kind.Elem()
	}
	if kind.Kind() != reflect.Struct || kind == reflect.TypeFor[big.Int]() {
		return nil
	}
	var keys []string
	for index := range kind.NumField() {
		field := kind.Field(index)
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		if name == "-" {
			continue
		}
		if field.Anonymous {
			keys = append(keys, tomlKeys(prefix, field.Type)...)
			continue
		}
		if name == "" {
			name = field.Name
		}
		key := prefix + name
		keys = append(keys, key)
		keys = append(keys, tomlKeys(key+".", field.Type)...)
	}
	return keys
}

func TestTomlKeys_KeepTheKeysOfTheReference(t *testing.T) {
	golden, err := os.ReadFile(referenceTomlKeys)
	require.NoError(t, err)
	current := tomlKeys("", reflect.TypeFor[Config]())
	for _, key := range strings.Fields(string(golden)) {
		if _, deprecated := deprecatedTomlKeys[key]; deprecated {
			continue
		}
		require.True(t, slices.Contains(current, key), "the TOML key %s of the reference is gone", key)
	}
}
