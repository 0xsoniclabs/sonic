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

// Command geth_compatibility lists every go-ethereum declaration that Sonic's binaries
// can reach and that differs between the go-ethereum version of a base revision
// and the one of the head, and fails until each difference carries a verdict in
// the review file. See README.md.
package main

import (
	"archive/tar"
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/tabwriter"

	"golang.org/x/mod/modfile"
)

const (
	gethModule = "github.com/ethereum/go-ethereum"
	reviewFile = "scripts/geth_compatibility/review.md"
)

type options struct {
	dir     string
	base    string
	head    string
	review  string
	write   bool
	all     bool
	diff    bool
	why     bool
	only    *regexp.Regexp
	prepare string
	verify  bool
	apply   string
	roots   []string
}

func main() {
	var o options
	var only string
	flag.StringVar(&o.dir, "dir", "", "Sonic checkout to check (default: the enclosing git repository)")
	flag.StringVar(&o.base, "base", "origin/main", "Sonic revision providing the old go-ethereum; its merge base with the head is used")
	flag.StringVar(&o.head, "head", "", "Sonic revision to check instead of the working tree")
	flag.StringVar(&o.review, "review", "", "review file (default: "+reviewFile+" of the head, read only with -head)")
	flag.BoolVar(&o.write, "write", false, "rewrite the review file: keep matching verdicts, add new findings as todo, drop stale rows")
	flag.BoolVar(&o.all, "all", false, "list reviewed findings too")
	flag.BoolVar(&o.diff, "diff", false, "print the source difference of every listed finding")
	flag.BoolVar(&o.why, "why", false, "print how Sonic reaches every listed finding")
	flag.StringVar(&only, "only", "", "list or prepare only findings whose symbol matches this regular expression")
	flag.StringVar(&o.prepare, "prepare", "", "write the findings that need a reviewer into this new directory, in groups, with a brief for the reviewers")
	flag.BoolVar(&o.verify, "verify", false, "with -prepare, take the reviewed findings that are consensus critical or open questions, for a second reviewer")
	flag.StringVar(&o.apply, "apply", "", "record the rows that reviewers filled in under a directory written by -prepare, and write the review file")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: geth_compatibility [flags] [main packages, default ./cmd/sonicd ./cmd/sonictool]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	o.roots = flag.Args()
	if len(o.roots) == 0 {
		o.roots = []string{"./cmd/sonicd", "./cmd/sonictool"}
	}
	if only != "" {
		re, err := regexp.Compile(only)
		if err != nil {
			fmt.Fprintln(os.Stderr, "geth_compatibility:", err)
			os.Exit(2)
		}
		o.only = re
	}
	switch {
	case o.verify && o.prepare == "":
		fmt.Fprintln(os.Stderr, "geth_compatibility: -verify needs -prepare")
		os.Exit(2)
	case o.prepare != "" && o.apply != "":
		fmt.Fprintln(os.Stderr, "geth_compatibility: -prepare and -apply exclude each other")
		os.Exit(2)
	case o.head != "" && o.review == "" && (o.write || o.apply != ""):
		fmt.Fprintln(os.Stderr, "geth_compatibility: with -head, -write and -apply need -review to name the file to write")
		os.Exit(2)
	}
	code, err := run(o)
	if err != nil {
		fmt.Fprintln(os.Stderr, "geth_compatibility:", err)
		os.Exit(2)
	}
	os.Exit(code)
}

func run(o options) (int, error) {
	if o.dir == "" {
		top, err := git(".", "rev-parse", "--show-toplevel")
		if err != nil {
			return 0, err
		}
		o.dir = top
	}
	headRev, headName := "HEAD", "working tree"
	if o.head != "" {
		headRev, headName = o.head, o.head
	}
	mergeBase, err := git(o.dir, "merge-base", o.base, headRev)
	if err != nil {
		return 0, err
	}
	baseName := fmt.Sprintf("%s (merge base with %s)", mergeBase[:10], o.base)

	baseMod, err := git(o.dir, "show", mergeBase+":go.mod")
	if err != nil {
		return 0, err
	}
	var headMod string
	if o.head == "" {
		b, err := os.ReadFile(filepath.Join(o.dir, "go.mod"))
		if err != nil {
			return 0, err
		}
		headMod = string(b)
	} else if headMod, err = git(o.dir, "show", o.head+":go.mod"); err != nil {
		return 0, err
	}
	baseGeth, err := gethVersion(baseMod)
	if err != nil {
		return 0, err
	}
	headGeth, err := gethVersion(headMod)
	if err != nil {
		return 0, err
	}
	if baseGeth == headGeth {
		fmt.Printf("go-ethereum is unchanged since %s: %s\n", baseName, baseGeth)
		return 0, nil
	}
	prepared := false
	if o.prepare != "" {
		if o.prepare, err = newWorkDir(o.prepare); err != nil {
			return 0, err
		}
		defer func() {
			if !prepared {
				clearDir(o.prepare)
			}
		}()
	}

	baseDir, err := extract(o.dir, mergeBase)
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(baseDir)
	var explain func(string) bool
	if o.why || o.prepare != "" {
		explain = func(key string) bool { return o.only == nil || o.only.MatchString(key) }
	}
	base, err := analyze(baseDir, o.roots, sonicConsensus, explain)
	if err != nil {
		return 0, fmt.Errorf("base: %w", err)
	}
	headDir := o.dir
	if o.head != "" && o.prepare != "" {
		headDir = filepath.Join(o.prepare, "head")
		if err := archive(o.dir, o.head, headDir); err != nil {
			return 0, err
		}
	} else if o.head != "" {
		if headDir, err = extract(o.dir, o.head); err != nil {
			return 0, err
		}
		defer os.RemoveAll(headDir)
	}
	head, err := analyze(headDir, o.roots, sonicConsensus, explain)
	if err != nil {
		return 0, fmt.Errorf("head: %w", err)
	}
	if len(head.missing) > 0 {
		return 0, fmt.Errorf("head lacks %s: update sonicConsensus", strings.Join(head.missing, ", "))
	}

	reviewName := o.review
	if o.review == "" {
		o.review = filepath.Join(headDir, filepath.FromSlash(reviewFile))
		reviewName = o.review
		if o.head != "" {
			reviewName = o.head + ":" + reviewFile
		}
	}
	findings := compare(base, head)
	entries, err := readReview(o.review)
	if err != nil {
		return 0, err
	}
	var problems []string
	if o.apply != "" {
		var applied int
		if applied, problems, err = applyRows(o.apply, findings, entries); err != nil {
			return 0, err
		}
		fmt.Printf("applied %d rows from %s\n\n", applied, o.apply)
		o.write = true
	}
	var pending, untagged []finding
	open := 0
	matched := map[string]bool{}
	critical := map[string]int{}
	for _, f := range findings {
		e, ok := entries[f.key]
		if ok && e.matches(f) {
			matched[f.key] = true
		}
		switch {
		case needsReview(f, e, ok):
			pending = append(pending, f)
		case e.verdict == verdictTodo:
			pending = append(pending, f)
			open++
		case e.verdict == verdictAdopted && f.consensus && e.consensus == "":
			untagged = append(untagged, f)
		}
		if ok && isCritical(e.consensus) {
			critical[e.consensus]++
		}
	}
	stale := len(entries) - len(matched)

	fmt.Printf("go-ethereum  base %s  (sonic %s)\n", baseGeth, baseName)
	fmt.Printf("             head %s  (sonic %s)\n", headGeth, headName)
	fmt.Printf("reachable    base %s\n", base.summary())
	fmt.Printf("             head %s\n", head.summary())
	fmt.Printf("findings     %s\n", countStatus(findings))
	fmt.Printf("review       %d with a verdict, %d open questions, %d to review, %d stale  (%s)\n",
		len(findings)-len(pending), open, len(pending)-open, stale, reviewName)
	fmt.Printf("consensus    %d on the consensus path, %s, %d adopted without a classification\n",
		countConsensus(findings), describeCritical(critical), len(untagged))
	if len(base.missing) > 0 {
		fmt.Printf("             base lacks %s\n", strings.Join(base.missing, ", "))
	}

	if o.prepare != "" {
		data := briefData{
			Verify:      o.verify,
			Dir:         o.prepare,
			Head:        headName,
			Base:        baseName,
			HeadGeth:    headGeth,
			BaseGeth:    baseGeth,
			HeadTree:    headDir,
			Modules:     head.modules,
			BaseModules: base.modules,
		}
		details := func(f finding) string {
			var b strings.Builder
			d := f.diff()
			writeDetails(&b, f, &d, true, base, head)
			return b.String()
		}
		prepared, err = prepare(data, selectFindings(findings, entries, o.verify, o.only), entries, details)
		return 0, err
	}

	listed := append(slices.Clone(pending), untagged...)
	if o.all {
		listed = findings
	}
	if o.only != nil {
		var filtered []finding
		for _, f := range listed {
			if o.only.MatchString(f.key) {
				filtered = append(filtered, f)
			}
		}
		listed = filtered
	}
	if len(listed) > 0 {
		fmt.Println()
		printFindings(listed, entries, o.diff, o.why, base, head)
	}

	if o.write {
		if err := writeReview(o.review, baseGeth, headGeth, findings, entries); err != nil {
			return 0, err
		}
		fmt.Printf("\nwrote %s: %d rows, %d pending\n", o.review, len(findings), len(pending))
	}
	if len(pending) > 0 {
		fmt.Printf("\nFAIL  %d findings have no verdict. Record one per row in %s\n", len(pending), reviewName)
		fmt.Println("      (-write adds the missing rows as todo, -diff -only <regexp> shows the changes).")
	}
	if len(untagged) > 0 {
		fmt.Printf("\nFAIL  %d adopted findings on the consensus path have no consensus classification.\n", len(untagged))
		fmt.Println("      Record live, a hardfork name or no in their Consensus cell.")
	}
	if len(problems) > 0 {
		fmt.Printf("\nFAIL  rows of %s were not applied:\n", o.apply)
		for _, p := range problems {
			fmt.Printf("      %s\n", p)
		}
	}
	if len(pending) > 0 || len(untagged) > 0 || len(problems) > 0 {
		return 1, nil
	}
	fmt.Println("\nOK    every go-ethereum difference reachable from Sonic has a verdict.")
	return 0, nil
}

func countConsensus(fs []finding) int {
	n := 0
	for _, f := range fs {
		if f.consensus {
			n++
		}
	}
	return n
}

func printFindings(fs []finding, entries map[string]entry, withDiff, withWhy bool, base, head *side) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tSYMBOL\tLOCATION\tLINES\tSONIC\tCONSENSUS\tVERDICT")
	diffs := make([]textDiff, len(fs))
	for i, f := range fs {
		diffs[i] = f.diff()
		add, del := diffs[i].stat()
		direct := ""
		if f.direct {
			direct = "direct"
		}
		consensus := ""
		if f.consensus {
			consensus = "path"
		}
		verdict := ""
		if e, ok := entries[f.key]; ok {
			if e.consensus != "" {
				consensus = e.consensus
			}
			if e.matches(f) {
				verdict = e.verdict
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t+%d -%d\t%s\t%s\t%s\n", f.status, f.key, f.location(), add, del, direct, consensus, verdict)
	}
	tw.Flush()
	if !withDiff && !withWhy {
		return
	}
	for i, f := range fs {
		var d *textDiff
		if withDiff {
			d = &diffs[i]
		}
		writeDetails(os.Stdout, f, d, withWhy, base, head)
	}
}

func writeDetails(w io.Writer, f finding, d *textDiff, withWhy bool, base, head *side) {
	fmt.Fprintf(w, "\n=== %s %s\n", f.status, f.key)
	if withWhy {
		label, chain := "head", head.why[f.key]
		if chain == nil {
			label, chain = "base", base.why[f.key]
		}
		fmt.Fprintf(w, "used by %s through:\n", label)
		for _, step := range chain {
			fmt.Fprintf(w, "  <- %s\n", step)
		}
		label, chain = "head", head.consensusWhy[f.key]
		if chain == nil {
			label, chain = "base", base.consensusWhy[f.key]
		}
		if chain != nil {
			fmt.Fprintf(w, "on the consensus path of %s through:\n", label)
			for _, step := range chain {
				fmt.Fprintf(w, "  <- %s\n", step)
			}
		}
	}
	if d != nil {
		if path, ok := strings.CutPrefix(f.key, "module "); ok {
			fmt.Fprintf(w, "sources: base %s, head %s\n", cmp.Or(base.modules[path], "-"), cmp.Or(head.modules[path], "-"))
		}
		fmt.Fprint(w, d.unified())
	}
}

func countStatus(fs []finding) string {
	n := map[string]int{}
	for _, f := range fs {
		n[f.status]++
	}
	return fmt.Sprintf("%d changed, %d added, %d removed", n[statusChanged], n[statusAdded], n[statusRemoved])
}

func gethVersion(gomod string) (string, error) {
	f, err := modfile.Parse("go.mod", []byte(gomod), nil)
	if err != nil {
		return "", err
	}
	var req *modfile.Require
	for _, r := range f.Require {
		if r.Mod.Path == gethModule {
			req = r
		}
	}
	if req == nil {
		return "", fmt.Errorf("go.mod does not require %s", gethModule)
	}
	var rep *modfile.Replace
	for _, r := range f.Replace {
		if r.Old.Path == gethModule && (r.Old.Version == req.Mod.Version || r.Old.Version == "" && rep == nil) {
			rep = r
		}
	}
	if rep == nil {
		return req.Mod.Path + " " + req.Mod.Version, nil
	}
	return strings.TrimSpace(rep.New.Path + " " + rep.New.Version), nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func extract(repo, rev string) (string, error) {
	dir, err := os.MkdirTemp("", "geth_compatibility-")
	if err != nil {
		return "", err
	}
	if err := archive(repo, rev, dir); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func archive(repo, rev, dir string) error {
	cmd := exec.Command("git", "-C", repo, "archive", "--format=tar", rev)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := untar(out, dir); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive %s: %v: %s", rev, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func untar(r io.Reader, dir string) error {
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(h.Name))
		switch h.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			err = writeFile(target, tr, os.FileMode(h.Mode).Perm())
		case tar.TypeSymlink:
			if err = os.MkdirAll(filepath.Dir(target), 0o755); err == nil {
				err = os.Symlink(h.Linkname, target)
			}
		}
		if err != nil {
			return err
		}
	}
}

func writeFile(path string, r io.Reader, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
