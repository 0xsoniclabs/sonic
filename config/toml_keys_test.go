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
	"bufio"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// tomlKeys lists the dotted names of the fields the TOML loader reads from a
// configuration file. Slices, maps and fields without exported fields are leaves.
func tomlKeys(prefix string, t reflect.Type, seen map[reflect.Type]bool, out *[]string) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		*out = append(*out, prefix)
		return
	}
	seen[t] = true
	defer delete(seen, t)
	exported := 0
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() || field.Tag.Get("toml") == "-" {
			continue
		}
		exported++
		name := field.Name
		if tag, _, _ := strings.Cut(field.Tag.Get("toml"), ","); tag != "" {
			name = tag
		}
		if field.Anonymous && field.Tag.Get("toml") == "" {
			tomlKeys(prefix, field.Type, seen, out)
			continue
		}
		tomlKeys(prefix+"."+name, field.Type, seen, out)
	}
	if exported == 0 {
		*out = append(*out, prefix)
	}
}

func currentTomlKeys() []string {
	var keys []string
	tomlKeys("", reflect.TypeOf(Config{}), map[reflect.Type]bool{}, &keys)
	sort.Strings(keys)
	return keys
}

// To regenerate the golden file of a release, run the test with
// TOML_KEYS_OUT=<file> in a checkout of that release.
func TestTomlKeys_WriteGolden(t *testing.T) {
	out := os.Getenv("TOML_KEYS_OUT")
	if out == "" {
		t.Skip("TOML_KEYS_OUT not set")
	}
	require.NoError(t, os.WriteFile(out, []byte(strings.Join(currentTomlKeys(), "\n")+"\n"), 0o644))
}

// O4: every key of the configuration file of the reference release is still
// read. The loader rejects unknown keys, so a removed key breaks configurations.
func TestTomlKeys_EveryKeyOfTheReferenceReleaseIsStillRead(t *testing.T) {
	file, err := os.Open("testdata/toml_keys_v2.2.4.txt")
	require.NoError(t, err)
	defer file.Close()

	current := map[string]bool{}
	for _, key := range currentTomlKeys() {
		current[key] = true
	}
	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		key := strings.TrimSpace(scanner.Text())
		if key == "" {
			continue
		}
		count++
		require.True(t, current[key], "key %s of v2.2.4 is not read any more", key)
	}
	require.NoError(t, scanner.Err())
	require.Greater(t, count, 100)
}
