package main

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andybalholm/brotli"
)

const stagingDir = ".testdata-tmp"

func cmdFetch(args []string) error {
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	if err := fl.Parse(args); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	if fl.NArg() > 0 {
		return fmt.Errorf("unexpected arguments %v", fl.Args())
	}

	m, err := readManifest()
	if err != nil {
		return err
	}
	assets := make([]string, 0, len(m.Assets))
	dirs := map[string]string{}
	for asset := range m.Assets {
		dir, err := assetDir(asset)
		if err != nil {
			return err
		}
		assets = append(assets, asset)
		dirs[asset] = dir
	}
	sort.Strings(assets)

	if installed(m.Release, dirs) {
		fmt.Printf("testdata %s already installed\n", m.Release)
		return nil
	}

	base := os.Getenv("TESTDATA_BASE_URL")
	if base == "" {
		base = "https://github.com/" + m.Repo + "/releases/download/" + m.Release
	}

	tmp := filepath.Join("testdata", stagingDir)
	stage := filepath.Join(tmp, "stage")
	if err := os.RemoveAll(tmp); err != nil {
		return fmt.Errorf("clearing %s: %w", tmp, err)
	}
	if err := os.MkdirAll(stage, 0o750); err != nil {
		return fmt.Errorf("creating %s: %w", stage, err)
	}
	defer os.RemoveAll(tmp)

	for _, asset := range assets {
		local := filepath.Join(tmp, asset)
		fmt.Printf("fetching %s\n", asset)
		if err := download(strings.TrimSuffix(base, "/")+"/"+asset, local, m.Assets[asset]); err != nil {
			return fmt.Errorf("%s: %w", asset, err)
		}
		if err := unpack(local, stage, dirs[asset]); err != nil {
			return fmt.Errorf("%s: %w", asset, err)
		}
		if err := os.Remove(local); err != nil {
			return fmt.Errorf("removing %s: %w", local, err)
		}
	}

	marker := filepath.Join("testdata", installedFile)
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("removing %s: %w", marker, err)
	}
	for _, asset := range assets {
		dir := dirs[asset]
		live := filepath.Join("testdata", dir)
		if _, err := os.Stat(live); err == nil {
			if err := os.Rename(live, filepath.Join(tmp, "old-"+dir)); err != nil {
				return fmt.Errorf("moving %s aside: %w", live, err)
			}
		}
		if err := os.Rename(filepath.Join(stage, dir), live); err != nil {
			return fmt.Errorf("installing %s: %w", live, err)
		}
	}
	if err := os.WriteFile(marker, []byte(m.Release+"\n"), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", marker, err)
	}
	fmt.Printf("testdata %s installed\n", m.Release)
	return nil
}

func installed(release string, dirs map[string]string) bool {
	b, err := os.ReadFile(filepath.Join("testdata", installedFile))
	if err != nil || strings.TrimSpace(string(b)) != release {
		return false
	}
	for _, dir := range dirs {
		if fi, err := os.Stat(filepath.Join("testdata", dir)); err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}

func download(src, dst, want string) error {
	body, err := open(src)
	if err != nil {
		return err
	}
	defer body.Close()

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), body); err != nil {
		f.Close()
		return fmt.Errorf("reading %s: %w", src, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", dst, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		os.Remove(dst)
		return fmt.Errorf("checksum mismatch: got %s, %s pins %s", got, manifestPath, want)
	}
	return nil
}

func open(src string) (io.ReadCloser, error) {
	u, err := url.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", src, err)
	}
	switch u.Scheme {
	case "http", "https":
		resp, err := http.Get(src) //nolint:gosec,noctx // the URL is the pinned release or TESTDATA_BASE_URL
		if err != nil {
			return nil, fmt.Errorf("GET %s: %w", src, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("GET %s: %s", src, resp.Status)
		}
		return resp.Body, nil
	case "file":
		p := u.Path
		if len(p) > 2 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		f, err := os.Open(filepath.FromSlash(p)) //nolint:gosec // TESTDATA_BASE_URL is the caller's own mirror
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", src, err)
		}
		return f, nil
	default:
		return nil, fmt.Errorf("unsupported URL scheme in %s", src)
	}
}

func unpack(archive, stage, dir string) error {
	f, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("opening %s: %w", archive, err)
	}
	defer f.Close()
	tr := tar.NewReader(brotli.NewReader(f))
	n := 0
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading %s: %w", archive, err)
		}
		name := path.Clean(hdr.Name)
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return fmt.Errorf("entry %s is not a regular file", hdr.Name)
		}
		if !strings.HasPrefix(name, dir+"/") || strings.Contains(name, "..") || strings.Contains(name, `\`) {
			return fmt.Errorf("entry %s is outside %s/", hdr.Name, dir)
		}
		if err := portableName(name); err != nil {
			return err
		}
		if err := extract(tr, filepath.Join(stage, filepath.FromSlash(name))); err != nil {
			return err
		}
		n++
	}
	if n == 0 {
		return fmt.Errorf("%s holds no files", archive)
	}
	fmt.Printf("  %d files\n", n)
	return nil
}

func extract(r io.Reader, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(dst), err)
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	if _, err := io.Copy(out, r); err != nil { //nolint:gosec // the archive's sha256 is pinned
		out.Close()
		return fmt.Errorf("writing %s: %w", dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", dst, err)
	}
	return nil
}
