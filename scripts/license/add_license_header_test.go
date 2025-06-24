package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Recognizes_GethLicense(t *testing.T) {
	// make a temporary file
	tmpDir := t.TempDir()
	originalFile, err := os.Create(tmpDir + "/test_geth_license")
	require.NoError(t, err)
	defer func() { _ = originalFile.Close() }()

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

	// write a sample license header to the file
	_, err = originalFile.WriteString(addPrefix(gethHeader, "//") + "package main")
	require.NoError(t, err)
	originalContent, err := os.ReadFile(originalFile.Name())
	require.NoError(t, err)

	// create a copy of the file with the same content
	copy, err := os.Create(tmpDir + "/test_geth_license_copy.go")
	require.NoError(t, err)
	defer func() { _ = copy.Close() }()

	err = copyFile(originalFile.Name(), copy.Name())
	require.NoError(t, err)

	err = processFiles(tmpDir, ".go", "//", gethHeader, false, false)
	require.NoError(t, err)

	contentAfter, err := os.ReadFile(copy.Name())
	require.NoError(t, err)

	require.Equal(t, string(originalContent), string(contentAfter))
}

func Test_Recognizes_CurrentSonicLicense(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()

	// make a temporary file in that folder
	tmpFile, err := os.Create(tmpDir + "/test_license")
	require.NoError(t, err)
	defer func() { _ = tmpFile.Close() }()

	_, err = tmpFile.WriteString(addPrefix(licenseHeader, "//") + "\npackage main")
	require.NoError(t, err)
	originalContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	// create a copy of the file with the same content
	copy, err := os.Create(tmpDir + "/test_geth_license_copy.go")
	require.NoError(t, err)
	defer func() { _ = copy.Close() }()

	err = copyFile(tmpFile.Name(), copy.Name())
	require.NoError(t, err)

	err = processFiles(tmpDir, ".go", "//", licenseHeader, false, false)
	require.NoError(t, err)

	contentAfter, err := os.ReadFile(copy.Name())
	require.NoError(t, err)
	require.Equal(t, string(originalContent), string(contentAfter))
}

func Test_Replaces_OldLicenseHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFile, err := os.Create(tmpDir + "/test_license.go")
	require.NoError(t, err)
	defer func() { _ = tmpFile.Close() }()

	// write a sample license header to the file
	oldLicense := `Copyright 2024 Sonic Operations Ltd
				   This file is part of some old version
				   of the Sonic Client`
	_, err = tmpFile.WriteString(addPrefix(oldLicense, "//") + "\n\npackage main\n\nfunc main() {}\n")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	err = processFiles(tmpDir, ".go", "//", licenseHeader, false, false)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	require.Contains(t, string(content), addPrefix(licenseHeader, "//"))

	require.NotContains(t, string(content), oldLicense)
}

func Test_Adds_LicenseHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFile, err := os.Create(tmpDir + "/test_license.go")
	require.NoError(t, err)
	defer func() { _ = tmpFile.Close() }()

	// write a sample license header to the file
	_, err = tmpFile.WriteString("package main\n\nfunc main() {}\n")
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	err = processFiles(tmpDir, ".go", "//", licenseHeader, false, false)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	extendLicenseHeader := addPrefix(licenseHeader, "//")
	require.Contains(t, string(content), extendLicenseHeader)
}

func copyFile(src, dst string) error {
	originalContent, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, originalContent, 0644)
}

func Test_Detects_DoubleHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFile, err := os.Create(tmpDir + "/test_license.go")
	require.NoError(t, err)
	defer func() { _ = tmpFile.Close() }()

	// write a sample license header to the file
	_, err = tmpFile.WriteString(addPrefix(licenseHeader, "//") + addPrefix(licenseHeader, "//") + "\npackage main")
	require.NoError(t, err)

	// Check for double license headers
	require.Error(t, checkDoubleHeader(tmpFile.Name(), "//"))
}

func Test_OnlyOnEmptyLineAfterHeader(t *testing.T) {
	// make a temporary folder
	tmpDir := t.TempDir()
	// make a temporary file in that folder
	tmpFile, err := os.Create(tmpDir + "/test_license.go")
	require.NoError(t, err)
	defer func() { _ = tmpFile.Close() }()

	// write a sample license header to the file
	_, err = tmpFile.WriteString(addPrefix(licenseHeader, "//") + "\npackage main\n\nfunc main() {}\n")
	require.NoError(t, err)

	// Check for double license headers
	err = processFiles(tmpDir, ".go", "//", licenseHeader, false, false)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	firstEmptyLine := false
	for _, line := range string(content) {
		if firstEmptyLine && line != '\n' {
			require.Fail(t, "There should be an empty line after the license header")
		} else {
			break // found code after the header
		}
		if !firstEmptyLine && line == '\n' {
			firstEmptyLine = true
		}
	}
}
