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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Recognizes_GethLicense(t *testing.T) {
	// make a temporary file
	originalFileName := filepath.Join(t.TempDir(), "test_geth_license.go")

	gethHeader := `Copyright 2014 The go-ethereum Authors
				   This file is part of the go-ethereum library.
				   
				    The go-ethereum library is free software: you can redistribute it and/or modify
				    it under the terms of the GNU Lesser General Public License as published by
				    the Free Software Foundation, either version 3 of the License, or
				    (at your option) any later version.
				   
				    The go-ethereum library is distributed in the hope that it will be useful,
				    but WITHOUT ANY WARRANTY; without even the implied warranty of
				    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
				    GNU Lesser General Public License for more details.
				   
				    You should have received a copy of the GNU Lesser General Public License
				    along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.`

	originalContent := []byte(gethHeader)
	require.NoError(t, os.WriteFile(originalFileName, originalContent, 0644))
	require.NoError(t, processFiles(originalFileName, ".go", "//", gethHeader, false))

	contentAfter, err := os.ReadFile(originalFileName)
	require.NoError(t, err)
	require.Equal(t, originalContent, contentAfter)
}

func Test_Recognizes_CurrentSonicLicense(t *testing.T) {
	// make a file in a temp folder
	tmpFileName := filepath.Join(t.TempDir(), "test_license")
	originalContent := []byte(addPrefix(licenseHeader, "//") + "\npackage main")
	require.NoError(t, os.WriteFile(tmpFileName, originalContent, 0660))

	require.NoError(t, processFiles(tmpFileName, ".go", "//", licenseHeader, false))

	contentAfter, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)
	require.Equal(t, originalContent, contentAfter)
}

func Test_Replaces_OldLicenseHeader(t *testing.T) {

	tmpFileName := filepath.Join(t.TempDir(), "test_license.go")
	// write a sample license header to the file
	oldLicense := `Copyright 2024 Sonic Operations Ltd
				   This file is part of some old version
				   of the Sonic Client`
	require.NoError(t,
		os.WriteFile(
			tmpFileName,
			[]byte(addPrefix(oldLicense, "//")+"\npackage main\n"),
			0660))

	require.NoError(t, processFiles(tmpFileName, ".go", "//", licenseHeader, false))

	contentAfter, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)
	require.Contains(t, string(contentAfter), addPrefix(licenseHeader, "//"))

	require.NotContains(t, string(contentAfter), oldLicense)
}

func Test_Adds_LicenseHeader(t *testing.T) {
	tmpFileName := filepath.Join(t.TempDir(), "test_license.go")
	require.NoError(t, os.WriteFile(tmpFileName, []byte("package main\n\nfunc main() {}\n"), 0660))

	require.NoError(t, processFiles(tmpFileName, ".go", "//", licenseHeader, false))

	contentAfter, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)
	extendLicenseHeader := addPrefix(licenseHeader, "//")
	require.Contains(t, string(contentAfter), extendLicenseHeader)
}

func Test_Detects_DoubleHeader(t *testing.T) {
	tmpFileName := filepath.Join(t.TempDir(), "test_license.go")
	doubleHeaderString := addPrefix(licenseHeader, "//") +
		addPrefix(licenseHeader, "//") +
		"\npackage main"
	require.NoError(t, os.WriteFile(tmpFileName, []byte(doubleHeaderString), 0660))

	// Check for double license headers
	require.Error(t, checkDoubleHeader(tmpFileName, "//"))
}

func Test_OnlyOneEmptyLineAfterHeader(t *testing.T) {
	tmpFileName := filepath.Join(t.TempDir(), "test_license.go")

	fileWithHeader := "// This is some documentation\npackage main\nfunc main() {}\n"
	require.NoError(t, os.WriteFile(tmpFileName, []byte(fileWithHeader), 0660))

	require.NoError(t, processFiles(tmpFileName, ".go", "//", licenseHeader, false))

	contentAfter, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)

	alreadyFoundEmptyLine := false
	for i, line := range strings.Split(string(contentAfter), "\n") {
		if len(line) == 0 {
			if !alreadyFoundEmptyLine {
				alreadyFoundEmptyLine = true
				continue // first empty line after the header
			}
			// if we found a second empty line, fail the test
			require.Fail(t, "There should be only one empty line after the license header", i)
		}
		// there is a non-empty line after the first empty line
		if alreadyFoundEmptyLine {
			break // all is good
		}
	}
}

func Test_IgnoresGeneratedFiles(t *testing.T) {
	tmpFileName := filepath.Join(t.TempDir(), "some_generated_file.go")
	originalContent := []byte("// Code generated - DO NOT EDIT.\npackage main\n")
	require.NoError(t, os.WriteFile(tmpFileName, originalContent, 0660))

	require.NoError(t, processFiles(tmpFileName, ".go", "//", licenseHeader, false))

	contentAfter, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)
	require.Equal(t, string(originalContent), string(contentAfter))
}

func Test_IgnoresMockFile(t *testing.T) {
	tmpFileName := filepath.Join(t.TempDir(), "some_mock.go")
	originalContent := []byte("// Some header.\npackage main\n")
	require.NoError(t, os.WriteFile(tmpFileName, originalContent, 0660))

	require.NoError(t, processFiles(tmpFileName, ".go", "//", licenseHeader, false))

	contentAfter, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)
	require.Equal(t, string(originalContent), string(contentAfter))
}
