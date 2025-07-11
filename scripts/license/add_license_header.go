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

// add_license_header.go: Add or check license headers in project files
// Usage: go run add_license_header.go [--check]

package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed license_header.txt
var licenseHeader string

func main() {
	// process command line flags
	checkOnly := flag.Bool("check", false,
		"Check mode: only verify headers, do not modify files")

	var targetDir string
	flag.StringVar(&targetDir, "dir", "",
		"Target directory to start processing files from. This flag is required to run.")

	flag.Parse()

	// get root dir from args
	if len(targetDir) <= 0 {
		log.Fatal("Please provide a directory to look for files, use -dir\n")
	}
	// Check if the directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		log.Fatalf("Invalid target directory: '%s'\n", targetDir)
	}
	fmt.Printf("Processing files in directory: %s\n", targetDir)

	// This map defines the file patterns for files names or extensions, and
	// their corresponding comment prefixes that will be added to the license header.
	// Patterns that start with a dot (.) are treated as file extensions,
	// while others are treated as specific file names.
	patterns := map[string]string{
		".go":         "//",
		".yml":        "#",
		"go.mod":      "//",
		"Jenkinsfile": "//",
		"BUILD":       "#",
	}
	// Process files with specified extensions
	for ext, prefix := range patterns {
		fmt.Printf("Processing files with extension %s using prefix '%s'\n", ext, prefix)
		err := processFiles(targetDir, ext, prefix, licenseHeader, *checkOnly)
		if err != nil {
			log.Fatalf("Error processing files with extension %s: %v\n", ext, err)
		}
	}
}

// processFiles walks through the directory tree starting from dir,
// finds files with the specified extension and processes them by adding or
// checking the license header.
//
// checkOnly indicates whether to only check the headers without modifying files.
// doubleHeader indicates whether to only check for double license headers.
// if doubleHeader is true, checkOnly is ignored.
func processFiles(dir, ext, prefix, license string, checkOnly bool) error {
	licenseHeader := addPrefix(license, prefix)
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// build files should not be checked
		if shouldIgnore(path, []string{"/build/", "_mock.go", ".pb.go"}) {
			return nil
		}
		if matchPattern(path, ext) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory %s: %v", dir, err)
	}
	anyFails := false
	for _, f := range files {
		if err := processFile(f, licenseHeader, checkOnly); err != nil {
			fmt.Println(err)
			if !checkOnly {
				// if check mode is not enabled, return the error
				return err
			}
			// since check mode is enabled, record that there was a failure
			anyFails = true
		}
		if checkOnly {
			if err := checkDoubleHeader(f, prefix); err != nil {
				fmt.Println(err)
				anyFails = true
			}
		}
	}
	// return an error if there were any files that failed the check
	if anyFails {
		return fmt.Errorf("some files do not have the correct license header or have double headers")
	}
	return nil
}

// processFile checks if the file given has the correct license header and corrects it if requested.
// If checkOnly is true, it only checks the header without modifying the file.
// If the file has an old license header, it replaces it with the new one.
// If the file does not have a license header, it adds the new one.
// If the file has the correct license header, it does nothing.
func processFile(file, licenseHeader string, checkOnly bool) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", file, err)
	}

	if strings.Contains(strings.ToLower(string(content)), strings.ToLower("code generated")) {
		return nil // skip generated files
	}

	lines := strings.Split(string(content), "\n")
	licenseLines := strings.Split(strings.TrimSuffix(licenseHeader, "\n"), "\n")
	needsUpdate := false

	// check if the file has the first line of geth lincense header
	if strings.Contains(lines[0], "The go-ethereum Authors") {
		return nil
	}

	for i, l := range licenseLines {
		if i >= len(lines) || strings.TrimSpace(lines[i]) != strings.TrimSpace(l) {
			needsUpdate = true
			break
		}
	}
	if !needsUpdate {
		return nil
	}
	if checkOnly {
		return fmt.Errorf("missing or incorrect license header: %s", file)
	}

	// this means the file has an old license header, we need to replace it
	if strings.Contains(lines[0], "Sonic Operations Ltd") {
		// search for the first empty line after the old license header
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				// remove lines up to this point
				content = []byte(strings.Join(lines[i+1:], "\n"))
				break
			}
		}
	}

	// Add header
	newContent := licenseHeader + "\n" + string(content)
	return os.WriteFile(file, []byte(newContent), 0000)
}

func checkDoubleHeader(path, prefix string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", path, err)
	}

	lines := strings.Split(string(content), "\n")

	// if the first line does not contain "Copyright", we assume there is no license header
	if !strings.Contains(lines[0], "Copyright") {
		return nil
	}
	for i, line := range lines[1:] {
		if strings.Contains(line, prefix+" Copyright") {
			return fmt.Errorf("double license header found in %s at line %d: %s", path, i+1, strings.Split(line, "\n")[0])
		}
	}
	return nil
}

// shouldIgnore checks if the file path should be ignored based on certain patterns.
func shouldIgnore(path string, ignoredPaths []string) bool {
	// the scripts ignores everything inside a build directory
	for _, pathFragment := range ignoredPaths {
		if strings.Contains(path, pathFragment) {
			return true
		}
	}
	return false
}

func matchPattern(path, pat string) bool {
	if pat[0] == '.' {
		return strings.HasSuffix(path, pat)
	}
	return filepath.Base(path) == pat
}

func addPrefix(license, prefix string) string {
	var buf bytes.Buffer
	s := bufio.NewScanner(strings.NewReader(license))
	for s.Scan() {
		line := s.Text()
		if line == "" {
			buf.WriteString(prefix + "\n")
		} else {
			buf.WriteString(prefix + " " + line + "\n")
		}
	}
	return buf.String()
}
