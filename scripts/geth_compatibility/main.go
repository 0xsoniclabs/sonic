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

// Command geth_compatibility writes a review file with a row for every
// go-ethereum declaration that Sonic's binaries can reach and that differs
// between the go-ethereum versions of two Sonic revisions. See README.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/0xsoniclabs/sonic/scripts/geth_compatibility/internal/analysis"
	"github.com/0xsoniclabs/sonic/scripts/geth_compatibility/internal/review"
)

func main() {
	base := flag.String("base", "origin/main", "Sonic revision with the old go-ethereum; its merge base with -head is used")
	head := flag.String("head", "", "Sonic branch or revision with the new go-ethereum")
	file := flag.String("review", "review.md", "review file to write; its rows are kept while their findings are unchanged")
	flag.Parse()
	if *head == "" || flag.NArg() > 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*base, *head, *file); err != nil {
		fmt.Fprintln(os.Stderr, "geth_compatibility:", err)
		os.Exit(1)
	}
}

func run(base, head, file string) error {
	top, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	mergeBase, err := git("merge-base", base, head)
	if err != nil {
		return err
	}
	baseDir, err := checkout(top, mergeBase)
	if err != nil {
		return fmt.Errorf("base: %w", err)
	}
	defer func() { _ = os.RemoveAll(baseDir) }()
	headDir, err := checkout(top, head)
	if err != nil {
		return fmt.Errorf("head: %w", err)
	}
	defer func() { _ = os.RemoveAll(headDir) }()
	baseSide, err := analysis.Analyze(baseDir, analysis.Sonic)
	if err != nil {
		return fmt.Errorf("base: %w", err)
	}
	headSide, err := analysis.Analyze(headDir, analysis.Sonic)
	if err != nil {
		return fmt.Errorf("head: %w", err)
	}
	if missing := analysis.Missing(baseSide, headSide); len(missing) > 0 {
		return fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	if err := analysis.Complete(baseDir, baseSide, headSide); err != nil {
		return fmt.Errorf("base: %w", err)
	}
	if err := analysis.Complete(headDir, headSide, baseSide); err != nil {
		return fmt.Errorf("head: %w", err)
	}
	findings := analysis.Compare(baseSide, headSide)
	todo, err := review.Update(file, baseSide, headSide, findings)
	if err != nil {
		return err
	}
	fmt.Printf("wrote %s: %d rows, %d todo\n", file, len(findings), todo)
	return nil
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// checkout extracts the committed files of rev into a new temporary directory.
func checkout(repo, rev string) (dir string, err error) {
	dir, err = os.MkdirTemp("", "geth_compatibility-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(dir)
		}
	}()
	archive := exec.Command("git", "-C", repo, "archive", rev)
	untar := exec.Command("tar", "-x", "-C", dir)
	tarball, err := archive.StdoutPipe()
	if err != nil {
		return "", err
	}
	untar.Stdin = tarball
	archive.Stderr, untar.Stderr = os.Stderr, os.Stderr
	err = untar.Start()
	// Only tar may hold the read end, so that git archive gets a broken pipe
	// instead of blocking if tar exits early.
	_ = tarball.Close()
	if err != nil {
		return "", err
	}
	if err := archive.Run(); err != nil {
		if tarErr := untar.Wait(); tarErr != nil {
			return "", fmt.Errorf("tar: %w", tarErr)
		}
		return "", fmt.Errorf("git archive %s: %w", rev, err)
	}
	if err := untar.Wait(); err != nil {
		return "", fmt.Errorf("tar: %w", err)
	}
	return dir, nil
}
