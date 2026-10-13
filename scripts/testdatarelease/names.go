package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/internal/oraclebin"
)

var (
	windowsIllegalChar = regexp.MustCompile(`[<>:"\\|?*\x00-\x1f]`)
	windowsReserved    = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[0-9]|LPT[0-9])(\..*)?$`)
)

func portableName(name string) error {
	for _, el := range strings.Split(name, "/") {
		switch {
		case windowsIllegalChar.MatchString(el):
			return fmt.Errorf("%s: %q is not a legal Windows file name", name, el)
		case strings.HasSuffix(el, ".") || strings.HasSuffix(el, " "):
			return fmt.Errorf("%s: %q ends in a dot or a space, which Windows drops", name, el)
		case windowsReserved.MatchString(el):
			return fmt.Errorf("%s: %q is a reserved Windows device name", name, el)
		}
	}
	return nil
}

func portableRenames(names []string) (map[string]string, error) {
	renames := map[string]string{}
	folded := map[string]string{}
	for _, name := range names {
		out := windowsIllegalChar.ReplaceAllString(name, "_")
		if err := portableName(out); err != nil {
			return nil, err
		}
		key := strings.ToLower(out)
		if prev, ok := folded[key]; ok {
			return nil, fmt.Errorf("%s and %s are one file on a case-insensitive filesystem", prev, name)
		}
		folded[key] = name
		if out != name {
			renames[name] = out
		}
	}
	return renames, nil
}

func renameOracle(fsys fs.FS, name string, b []byte, renames map[string]string) ([]byte, error) {
	switch {
	case len(renames) == 0:
		return b, nil
	case strings.HasSuffix(name, ".jsonl.gz"):
		return renameJSONL(b, renames)
	case strings.HasSuffix(name, oraclebin.BinExt):
		return regenerateBin(fsys, oraclebin.SourcePath(name), renames)
	}
	for old := range renames {
		if bytes.Contains(b, []byte(strings.TrimSuffix(old, ".xml"))) {
			return nil, fmt.Errorf("%s names the renamed build %s and pack does not know how to rewrite it", name, old)
		}
	}
	return b, nil
}

func renameJSONL(gz []byte, renames map[string]string) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("oracle blob: %w", err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("oracle blob: %w", err)
	}
	olds := make([]string, 0, len(renames))
	for old := range renames {
		olds = append(olds, old)
	}
	sort.Strings(olds)
	for _, old := range olds {
		from, _ := json.Marshal(old)
		to, _ := json.Marshal(renames[old])
		from = append([]byte(`{"build":`), from...)
		to = append([]byte(`{"build":`), to...)
		if n := bytes.Count(b, from); n != 1 {
			return nil, fmt.Errorf("oracle blob: %d records lead with build %s, want 1", n, old)
		}
		b = bytes.Replace(b, from, to, 1)
	}
	var out bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&out, gzip.BestCompression)
	if _, err := zw.Write(b); err != nil {
		return nil, fmt.Errorf("oracle blob: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("oracle blob: %w", err)
	}
	return out.Bytes(), nil
}

func regenerateBin(fsys fs.FS, jsonlPath string, renames map[string]string) ([]byte, error) {
	src, err := fs.ReadFile(fsys, jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", jsonlPath, err)
	}
	b, err := renameJSONL(src, renames)
	if err != nil {
		return nil, err
	}
	tmp := filepath.Join(os.TempDir(), "testdatarelease-"+path.Base(jsonlPath))
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return nil, fmt.Errorf("writing %s: %w", tmp, err)
	}
	defer os.Remove(tmp)
	recs, err := oraclebin.DecodeJSONL(tmp)
	if err != nil {
		return nil, err
	}
	digest, err := oraclebin.DigestFile(tmp)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := oraclebin.Encode(&out, recs, digest); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
