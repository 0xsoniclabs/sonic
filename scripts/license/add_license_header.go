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

var (
	licenseFile = "license_header.txt"
	ignorePaths = []string{"/build/"}
	extensions  = map[string]string{
		".go":         "//",
		"Jenkinsfile": "//",
		"go.mod":      "//",
		".yml":        "#",
		"BUILD":       "#",
	}
	//go:embed license_header.txt
	licenseHeader string
)

func main() {
	// process optional flag
	checkOnly := flag.Bool("check", false, "Check mode: only verify headers, do not modify files")
	checkDoubleHeader := flag.Bool("double-header", false, "Check for double license headers")
	var targetDir string
	flag.StringVar(&targetDir, "dir", "", "Target directory to start processing files from. This flag is required to run.")
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

	// Process files with specified extensions
	result := 0
	for ext, prefix := range extensions {
		fmt.Printf("Processing files with extension %s using prefix '%s'\n", ext, prefix)
		err := processFiles(targetDir, ext, prefix, licenseHeader, *checkOnly, *checkDoubleHeader)
		if err != nil {
			log.Fatalf("Error processing files with extension %s: %v\n", ext, err)
		}
	}
	os.Exit(result)
}

func processFiles(root, ext, prefix, license string, checkOnly, doubleHeader bool) error {
	licenseHeader := addPrefix(license, prefix)
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if shouldIgnore(path) {
			return nil
		}
		if matchExtension(path, ext) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory %s: %v", root, err)
	}
	for _, f := range files {
		if doubleHeader {
			if err := checkDoubleHeader(f, prefix); err != nil {
				fmt.Println(err)
			}
			continue
		}
		if err := processFile(f, licenseHeader, checkOnly); err != nil {
			fmt.Println(err)
			if checkOnly {
				continue
			}
			return err
		}
	}
	return nil
}

func processFile(path, licenseHeader string, checkOnly bool) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", path, err)
	}
	// we check the first 20 lines because the license header should be the top 15 lines.
	lines := strings.SplitN(string(content), "\n", 20)
	licenseLines := strings.Split(strings.TrimSuffix(licenseHeader, "\n"), "\n")
	needsUpdate := false

	// check if the file has the first line of geth lincense header
	if strings.Contains(lines[0], "The go-ethereum Authors") {
		// license header is 15 lines, 16th should be empty
		if strings.TrimSpace(lines[15]) != "" {
			lines[14] += "\n"
		}
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
		return fmt.Errorf("missing or incorrect license header: %s", path)
	}

	// this means the file has an old license header, we need to replace it
	if strings.Contains(lines[0], "Sonic Operations Ltd") {
		// search for the first empty line after the old license header
		counter := 0
		for i, line := range lines {
			counter += len(line)
			if strings.TrimSpace(line) == "" {
				// remove lines up to this point
				content = []byte(strings.Join(lines[i+1:], "\n"))
				break
			}
		}
	}

	// Add header
	newContent := licenseHeader + "\n" + string(content)
	return os.WriteFile(path, []byte(newContent), 0222)
}

func checkDoubleHeader(path, prefix string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", path, err)
	}
	// we check the first 20 lines because the license header should be the top 15 lines.
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

func shouldIgnore(path string) bool {
	for _, pat := range ignorePaths {
		if strings.Contains(path, pat) {
			return true
		}
	}
	return false
}

func matchExtension(path, ext string) bool {
	if ext[0] == '.' {
		return strings.HasSuffix(path, ext)
	}
	return filepath.Base(path) == ext
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
