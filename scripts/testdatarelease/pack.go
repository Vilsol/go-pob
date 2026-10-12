package main

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
)

var defaultVersionRe = regexp.MustCompile(`(?m)^const DefaultVersion = "([^"]+)"$`)

var lfsPointer = []byte("version https://git-lfs.github.com/spec/v1")

var archiveTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

const repo = "Vilsol/go-pob-testdata"

func cmdPack(args []string) error {
	fl := flag.NewFlagSet("pack", flag.ExitOnError)
	out := fl.String("out", filepath.Join(os.TempDir(), "go-pob-testdata-release"), "output directory for the archives")
	date := fl.String("date", time.Now().UTC().Format("2006.01.02"), "release date in the tag (yyyy.mm.dd)")
	if err := fl.Parse(args); err != nil {
		return fmt.Errorf("pack: %w", err)
	}
	if fl.NArg() > 0 {
		return fmt.Errorf("unexpected arguments %v", fl.Args())
	}

	version, err := gameDataVersion()
	if err != nil {
		return err
	}
	rules, err := loadExportRules()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", *out, err)
	}

	m := manifest{Repo: repo, Release: version + "-" + *date, Assets: map[string]string{}}
	var files []string
	for _, dir := range []string{"corpus-" + version, "oracle-" + version} {
		asset := dir + archiveExt
		sum, n, err := packDir(filepath.Join("testdata", dir), dir, filepath.Join(*out, asset), rules)
		if err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
		m.Assets[asset] = sum
		files = append(files, filepath.Join(*out, asset))
		fmt.Printf("%s  %d files  %.1f MB  %s\n", asset, n, float64(fileSize(filepath.Join(*out, asset)))/1e6, sum)
	}

	var sums strings.Builder
	names := make([]string, 0, len(m.Assets))
	for name := range m.Assets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&sums, "%s  %s\n", m.Assets[name], name)
	}
	if err := os.WriteFile(filepath.Join(*out, sumsFile), []byte(sums.String()), 0o644); err != nil { //nolint:gosec // published with the release
		return fmt.Errorf("writing %s: %w", sumsFile, err)
	}
	files = append(files, filepath.Join(*out, sumsFile))

	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", manifestPath, err)
	}
	if err := os.WriteFile(filepath.FromSlash(manifestPath), append(b, '\n'), 0o644); err != nil { //nolint:gosec // a tracked file
		return fmt.Errorf("writing %s: %w", manifestPath, err)
	}
	fmt.Printf("wrote %s (release %s)\n\nPublish with:\n\n", manifestPath, m.Release)
	fmt.Printf("gh release create %s %s --repo %s --title %q --notes %q\n", m.Release, strings.Join(files, " "), repo,
		"Test data "+m.Release, "Corpus and PoB oracle for game data "+version+". Fetched by mise run testdata in go-pob.")
	return nil
}

func gameDataVersion() (string, error) {
	if v := os.Getenv("POE_VERSION"); v != "" {
		return v, nil
	}
	src, err := os.ReadFile(filepath.Join("data", "raw", "main.go"))
	if err != nil {
		return "", fmt.Errorf("reading the game-data version: %w", err)
	}
	m := defaultVersionRe.FindSubmatch(src)
	if m == nil {
		return "", fmt.Errorf("data/raw/main.go declares no `const DefaultVersion = \"...\"`")
	}
	return string(m[1]), nil
}

func packDir(src, prefix, dst string, rules *exportRules) (string, int, error) {
	var names []string
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is not a regular file", p)
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		names = append(names, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return "", 0, fmt.Errorf("listing %s: %w", src, err)
	}
	if len(names) == 0 {
		return "", 0, fmt.Errorf("%s holds no files", src)
	}
	sort.Strings(names)
	if err := rules.checkAll(src, prefix, names); err != nil {
		return "", 0, err
	}

	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", 0, fmt.Errorf("creating %s: %w", tmp, err)
	}
	defer os.Remove(tmp)
	h := sha256.New()
	err = writeArchive(io.MultiWriter(f, h), src, prefix, names, rules)
	if cerr := f.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("writing %s: %w", tmp, cerr)
	}
	if err != nil {
		return "", 0, err
	}
	if err := os.Rename(tmp, dst); err != nil {
		return "", 0, fmt.Errorf("renaming %s: %w", tmp, err)
	}
	return hex.EncodeToString(h.Sum(nil)), len(names), nil
}

func writeArchive(w io.Writer, src, prefix string, names []string, rules *exportRules) error {
	bw := brotli.NewWriterOptions(w, brotli.WriterOptions{Quality: 9, LGWin: 24})
	tw := tar.NewWriter(bw)
	for _, name := range names {
		if err := addFile(tw, src, prefix, name, rules); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("closing the tar: %w", err)
	}
	if err := bw.Close(); err != nil {
		return fmt.Errorf("closing the brotli stream: %w", err)
	}
	return nil
}

func addFile(tw *tar.Writer, src, prefix, name string, rules *exportRules) error {
	p := filepath.Join(src, filepath.FromSlash(name))
	b, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("reading %s: %w", p, err)
	}
	if bytes.HasPrefix(b, lfsPointer) {
		return fmt.Errorf("%s is a Git LFS pointer; run `git lfs pull` first", p)
	}
	b = rules.rewrite("testdata/"+prefix+"/"+name, b)
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     prefix + "/" + name,
		Mode:     0o644,
		Size:     int64(len(b)),
		ModTime:  archiveTime,
		Format:   tar.FormatPAX,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("%s: %w", hdr.Name, err)
	}
	if _, err := tw.Write(b); err != nil {
		return fmt.Errorf("%s: %w", hdr.Name, err)
	}
	return nil
}

func fileSize(p string) int64 {
	fi, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return fi.Size()
}
