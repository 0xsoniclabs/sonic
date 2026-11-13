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

package app

import (
	"encoding"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/config"
	"github.com/stretchr/testify/require"
)

// updateGolden rewrites the golden files of the operator surface instead of
// comparing against them. Run it only for a change that adds a flag or a key:
// operators' command lines and configuration files break on a removed or
// renamed one, so those stay for a release as deprecated (accepted, warned,
// without effect).
var updateGolden = flag.Bool("update-golden", false, "rewrite the golden files of the operator surface")

func TestSurface_FlagsAreThoseOfTheGoldenFile(t *testing.T) {
	var names []string
	for _, flag := range initApp().Flags {
		names = append(names, flag.GetName())
	}
	compareWithGolden(t, "flags.golden", names)
}

func TestSurface_ConfigKeysAreThoseOfTheGoldenFile(t *testing.T) {
	var keys []string
	collectTomlKeys("", reflect.TypeFor[config.Config](), &keys)
	compareWithGolden(t, "config_keys.golden", keys)
}

func TestSurface_CollectTomlKeys_FollowsTheLoaderRules(t *testing.T) {
	type inner struct {
		Leaf int
	}
	type outer struct {
		Plain   string
		Renamed int `toml:"Other,omitempty"`
		Ignored int `toml:"-"`
		Nested  inner
		Pointer *inner
		Text    textValue
		hidden  int //nolint:unused // the shape under test
	}
	var keys []string
	collectTomlKeys("", reflect.TypeFor[outer](), &keys)
	slices.Sort(keys)
	require.Equal(t, []string{
		"Nested.Leaf",
		"Other",
		"Plain",
		"Pointer.Leaf",
		"Text",
	}, keys)
}

type textValue struct{ value string }

func (v *textValue) UnmarshalText(text []byte) error {
	v.value = string(text)
	return nil
}

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

// collectTomlKeys appends the dotted TOML keys the configuration loader
// accepts for a value of the given type: an exported field is a key named as
// the field or as its tag, a struct is a table of keys, and a value decoded
// from text is one key.
func collectTomlKeys(prefix string, valueType reflect.Type, keys *[]string) {
	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	if valueType.Kind() != reflect.Struct || reflect.PointerTo(valueType).Implements(textUnmarshalerType) {
		*keys = append(*keys, prefix)
		return
	}
	for field := range valueType.Fields() {
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		if prefix != "" {
			name = prefix + "." + name
		}
		collectTomlKeys(name, field.Type, keys)
	}
}

func compareWithGolden(t *testing.T, name string, lines []string) {
	t.Helper()
	slices.Sort(lines)
	if *updateGolden {
		content := strings.Join(lines, "\n") + "\n"
		require.NoError(t, os.WriteFile(goldenPath(name), []byte(content), 0o644))
		return
	}
	want := readGolden(t, name)
	require.Equal(t, want, lines,
		"the operator surface changed; a removed or renamed entry breaks existing "+
			"command lines and configuration files, so it stays deprecated for a release; "+
			"an added one is recorded with -update-golden")
}

func readGolden(t *testing.T, name string) []string {
	t.Helper()
	content, err := os.ReadFile(goldenPath(name))
	require.NoError(t, err)
	return strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
}

func goldenPath(name string) string {
	return filepath.Join("testdata", name)
}
