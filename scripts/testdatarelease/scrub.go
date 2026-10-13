package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

type exportRules struct {
	rewrites [][2]string
	leaks    *regexp.Regexp
	allow    *regexp.Regexp
}

func loadExportRules() (*exportRules, error) {
	dir := filepath.Join("scripts", "export")
	leaks, err := readPatterns(filepath.Join(dir, "leaks.txt"))
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("NOTE: no scripts/export/leaks.txt here; the archives are not leak-checked")
		return nil, nil //nolint:nilnil // nil rules mean no export here; every method accepts them
	}
	if err != nil {
		return nil, err
	}
	allow, err := readPatterns(filepath.Join(dir, "leaks-allow.txt"))
	if err != nil {
		return nil, err
	}
	r := &exportRules{leaks: regexp.MustCompile("(?i)" + leaks), allow: regexp.MustCompile(allow)}
	b, err := os.ReadFile(filepath.Join(dir, "replace.txt"))
	if err != nil {
		return nil, fmt.Errorf("reading the export rewrites: %w", err)
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		from, to, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("replace.txt line %q has no tab", line)
		}
		r.rewrites = append(r.rewrites, [2]string{from, to})
	}
	return r, nil
}

func readPatterns(p string) (string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", p, err)
	}
	var alts []string
	for _, line := range strings.Split(string(b), "\n") {
		if line != "" {
			alts = append(alts, "(?:"+line+")")
		}
	}
	return strings.Join(alts, "|"), nil
}

func (r *exportRules) rewrite(path string, b []byte) []byte {
	if r == nil || strings.HasSuffix(path, ".xml") {
		return b
	}
	for _, rw := range r.rewrites {
		b = bytes.ReplaceAll(b, []byte(rw[0]), []byte(rw[1]))
	}
	return b
}

func (r *exportRules) leak(path string, b []byte) error {
	if r == nil {
		return nil
	}
	if r.leaks.MatchString(path) && !r.allow.MatchString(path) {
		return fmt.Errorf("%s: the path matches scripts/export/leaks.txt", path)
	}
	if bytes.IndexByte(b[:min(len(b), 8000)], 0) >= 0 || !r.leaks.Match(b) {
		return nil
	}
	for i, line := range strings.Split(string(b), "\n") {
		hit := fmt.Sprintf("./%s:%d:%s", path, i+1, line)
		if r.leaks.MatchString(line) && !r.allow.MatchString(hit) {
			return fmt.Errorf("leak (scripts/export/leaks.txt) at %s", hit[:min(len(hit), 240)])
		}
	}
	return nil
}

func (r *exportRules) checkAll(fsys fs.FS, src, prefix string, names []string) error {
	if r == nil {
		return nil
	}
	work := make(chan string)
	errs := make(chan error, len(names))
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range work {
				p := filepath.Join(src, filepath.FromSlash(name))
				b, err := fs.ReadFile(fsys, name)
				if err != nil {
					errs <- fmt.Errorf("reading %s: %w", p, err)
					continue
				}
				path := "testdata/" + prefix + "/" + name
				if err := r.leak(path, r.rewrite(path, b)); err != nil {
					errs <- err
				}
			}
		}()
	}
	for _, name := range names {
		work <- name
	}
	close(work)
	wg.Wait()
	close(errs)
	return <-errs
}
