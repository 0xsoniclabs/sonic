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
	tmpDir := t.TempDir()
	originalFileName := filepath.Join(tmpDir, "test_geth_license.go")

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

	require.NoError(t, os.WriteFile(originalFileName, []byte(gethHeader), 0644))

	originalContent, err := os.ReadFile(originalFileName)
	require.NoError(t, err)

	// create a copy of the file with the same content
	copyName := filepath.Join(tmpDir, "test_geth_license_copy.go")
	require.NoError(t, copyFile(originalFileName, copyName))

	require.NoError(t, processFiles(tmpDir, ".go", "//", gethHeader, false, false))

	contentAfter, err := os.ReadFile(copyName)
	require.NoError(t, err)

	require.Equal(t, originalContent, contentAfter)
}

func Test_Recognizes_CurrentSonicLicense(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()

	// make a temporary file in that folder
	tmpFileName := filepath.Join(tmpDir, "test_license")
	require.NoError(t, os.WriteFile(tmpFileName, []byte(addPrefix(licenseHeader, "//")+"\npackage main"), 0660))

	originalContent, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)

	// create a copy of the file with the same content
	copyName := filepath.Join(tmpDir, "test_geth_license_copy.go")
	require.NoError(t, copyFile(tmpFileName, copyName))

	require.NoError(t, processFiles(tmpDir, ".go", "//", licenseHeader, false, false))

	contentAfter, err := os.ReadFile(copyName)
	require.NoError(t, err)
	require.Equal(t, originalContent, contentAfter)
}

func Test_Replaces_OldLicenseHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFileName := filepath.Join(tmpDir, "test_license.go")
	// write a sample license header to the file
	oldLicense := `Copyright 2024 Sonic Operations Ltd
				   This file is part of some old version
				   of the Sonic Client`
	originalContent := []byte(addPrefix(oldLicense, "//") + "\npackage main\n")

	require.NoError(t, os.WriteFile(tmpFileName, originalContent, 0660))

	require.NoError(t, processFiles(tmpDir, ".go", "//", licenseHeader, false, false))

	content, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)
	require.Contains(t, string(content), addPrefix(licenseHeader, "//"))

	require.NotContains(t, string(content), oldLicense)
}

func Test_Adds_LicenseHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFileName := filepath.Join(tmpDir, "test_license.go")
	// write a file without license header
	require.NoError(t, os.WriteFile(tmpFileName, []byte("package main\n\nfunc main() {}\n"), 0660))

	require.NoError(t, processFiles(tmpDir, ".go", "//", licenseHeader, false, false))

	content, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)
	extendLicenseHeader := addPrefix(licenseHeader, "//")
	require.Contains(t, string(content), extendLicenseHeader)
}

func copyFile(src, dst string) error {
	originalContent, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, originalContent, 0440)
}

func Test_Detects_DoubleHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFileName := filepath.Join(tmpDir, "test_license.go")
	doubleHeaderString := addPrefix(licenseHeader, "//") +
		addPrefix(licenseHeader, "//") +
		"\npackage main"
	require.NoError(t, os.WriteFile(tmpFileName, []byte(doubleHeaderString), 0660))

	// Check for double license headers
	require.Error(t, checkDoubleHeader(tmpFileName, "//"))
}

func Test_OnlyOneEmptyLineAfterHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFileName := filepath.Join(tmpDir, "test_license.go")

	fileWithHeader := addPrefix(licenseHeader, "//") + "\npackage main\nfunc main() {}\n"
	require.NoError(t, os.WriteFile(tmpFileName, []byte(fileWithHeader), 0660))

	require.NoError(t, processFiles(tmpDir, ".go", "//", licenseHeader, false, false))

	content, err := os.ReadFile(tmpFileName)
	require.NoError(t, err)

	alreadyFoundEmptyLine := false
	for i, line := range strings.Split(string(content), "\n") {
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
