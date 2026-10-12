package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/fields"

	"github.com/Vilsol/go-pob/scripts/gopob/internal/datafields"
	"github.com/Vilsol/go-pob/scripts/gopob/internal/root"
)

const dataFieldsRefusal = 5

const dataFieldsUsage = `gopob data-fields --list FILE | --check [--list FILE]
      analyse which go-pob-data row fields go-pob reads and write the list to FILE (it goes upstream
      as go-pob-data's fields/<version>.json), or with --check exit 1 naming every read field the
      list lacks (default: the list go-pob-data publishes)
  gopob data-fields project --list FILE --from DIR --to DIR
      write FILE's projection of the cached full tuple assets (go-pob-data's fields.Project) into an
      EMPTY cache dir (GO_POB_CACHE_DIR layout), under the full assets' own URLs
  gopob data-fields sizes --list FILE --from DIR
      per asset: raw msgpack and brotli bytes, full vs projected
`

func dataFields(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "project":
			return dataFieldsProject(args[1:])
		case "sizes":
			return dataFieldsSizes(args[1:])
		}
	}
	fs := flag.NewFlagSet("data-fields", flag.ContinueOnError)
	check := fs.Bool("check", false, "compare against the committed list instead of writing it")
	listPath := fs.String("list", "", "list file to write, or to --check (default for --check: go-pob-data's published list)")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || (!*check && *listPath == "") {
		fmt.Fprint(os.Stderr, dataFieldsUsage)
		return 64
	}
	repo, err := root.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return root.RefusalCode
	}
	res, err := datafields.Analyze(repo)
	var refusal *datafields.RefusalError
	if errors.As(err, &refusal) {
		fmt.Fprintln(os.Stderr, err)
		return dataFieldsRefusal
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fresh := datafields.Build(res, datafields.DataModule+"@"+res.DataModuleVersion, res.DefaultVersion)
	if *check {
		committed, err := datafields.Published(res.DefaultVersion)
		source := "go-pob-data's published list for " + res.DefaultVersion
		if *listPath != "" {
			committed, err = datafields.Load(*listPath)
			source = *listPath
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		missing := datafields.Missing(committed, fresh)
		for _, m := range missing {
			fmt.Println("MISSING", m)
		}
		if len(missing) > 0 {
			fmt.Printf("FAIL: %d key paths go-pob reads are absent from %s; write a new list with `task data-fields -- --list FILE` and send it upstream\n", len(missing), source)
			return 1
		}
		fmt.Printf("OK: %s covers every field go-pob reads (%d assets)\n", source, len(fresh.Assets))
		return 0
	}
	b, err := fresh.Marshal()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(*listPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.WriteFile(*listPath, b, 0o644); err != nil { //nolint:gosec // G306: a committed source file
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	kept, total := 0, 0
	for p, a := range fresh.Assets {
		kept += len(a.Paths())
		total += len(a.Paths()) + len(fresh.Unread[p])
	}
	fmt.Printf("wrote %s: %d assets, %d of %d key paths kept, %d types kept whole\n", *listPath, len(fresh.Assets), kept, total, len(fresh.Whole))
	return 0
}

func cacheFile(dir, namespace, url string) string {
	sum := sha256.Sum256([]byte(namespace + "|" + url))
	return filepath.Join(dir, hex.EncodeToString(sum[:]))
}

type loadedAsset struct {
	path string
	list datafields.AssetList
	full []byte
	raw  []byte
	file string
}

func loadAssets(l *datafields.List, from string) ([]loadedAsset, error) {
	namespace := l.DataModule[strings.LastIndex(l.DataModule, "@")+1:]
	paths := make([]string, 0, len(l.Assets))
	for p := range l.Assets {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var out []loadedAsset
	for _, p := range paths {
		url, err := datafields.AssetURL(l.Version, p)
		if err != nil {
			return nil, err
		}
		file := cacheFile(from, namespace, url)
		full, err := os.ReadFile(file) //nolint:gosec // G304: a cache path built from the list
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "SKIPPED %s: not in %s (the CDN has no %s, or it was never loaded)\n", p, from, url)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		raw, err := datafields.Decompress(full)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, loadedAsset{path: p, list: l.Assets[p], full: full, raw: raw, file: filepath.Base(file)})
	}
	return out, nil
}

func dataFieldsProject(args []string) int {
	fs := flag.NewFlagSet("data-fields project", flag.ContinueOnError)
	listPath := fs.String("list", "", "list file")
	from := fs.String("from", "", "cache dir holding the full assets")
	to := fs.String("to", "", "empty cache dir to write the projected assets into")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || *listPath == "" || *from == "" || *to == "" {
		fmt.Fprint(os.Stderr, dataFieldsUsage)
		return 64
	}
	if entries, err := os.ReadDir(*to); err == nil && len(entries) > 0 {
		fmt.Fprintf(os.Stderr, "REFUSING: %s is not empty; a stale asset there would be served as if projected\n", *to)
		return dataFieldsRefusal
	}
	l, err := datafields.Load(*listPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	assets, err := loadAssets(l, *from)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.MkdirAll(*to, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for _, a := range assets {
		br, err := fields.Project(a.path, a.list.Keep, a.full)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		projected, err := datafields.Decompress(br)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := os.WriteFile(filepath.Join(*to, a.file), br, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("%-50s %10d -> %10d raw\n", a.path, len(a.raw), len(projected))
	}
	fmt.Printf("wrote %d projected assets to %s\n", len(assets), *to)
	return 0
}

func dataFieldsSizes(args []string) int {
	fs := flag.NewFlagSet("data-fields sizes", flag.ContinueOnError)
	listPath := fs.String("list", "", "list file")
	from := fs.String("from", "", "cache dir holding the full assets")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || *listPath == "" || *from == "" {
		fmt.Fprint(os.Stderr, dataFieldsUsage)
		return 64
	}
	l, err := datafields.Load(*listPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	assets, err := loadAssets(l, *from)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("asset\tkept_keys\traw_full\traw_min\tbr_cdn\tbr_full_q11\tbr_min_q11")
	var t [5]int
	for _, a := range assets {
		br, err := fields.Project(a.path, a.list.Keep, a.full)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		projected, err := datafields.Decompress(br)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fullQ, err := datafields.Compress(a.raw)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		minQ, err := datafields.Compress(projected)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		row := [5]int{len(a.raw), len(projected), len(a.full), len(fullQ), len(minQ)}
		for i := range t {
			t[i] += row[i]
		}
		fmt.Printf("%s\t%d\t%d\t%d\t%d\t%d\t%d\n", a.path, len(a.list.Keep), row[0], row[1], row[2], row[3], row[4])
	}
	fmt.Printf("TOTAL\t-\t%d\t%d\t%d\t%d\t%d\n", t[0], t[1], t[2], t[3], t[4])
	return 0
}
