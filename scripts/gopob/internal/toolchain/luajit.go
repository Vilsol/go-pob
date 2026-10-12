package toolchain

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LuaJIT int

const (
	Pinned LuaJIT = iota
	Deterministic
)

const DeterministicPath = ".luajit-det/bin/luajit"

type Binary struct {
	Path string
	MD5  string
	Size int64
}

func (b Binary) String() string { return fmt.Sprintf("%s (md5 %s, %d bytes)", b.Path, b.MD5, b.Size) }

func LuaJITBinary(ctx context.Context, root string, which LuaJIT) (Binary, error) {
	var path string
	switch which {
	case Deterministic:
		path = filepath.Join(root, DeterministicPath)
		if st, e := os.Stat(path); e != nil || st.IsDir() {
			return Binary{}, fmt.Errorf(
				"REFUSING: no deterministic luajit at %s — run `task luajit-det`. Stock LuaJIT's "+
					"pairs() order varies per process, so anything dumped with it is a mosaic",
				DeterministicPath)
		}
	case Pinned:
		if !HaveMise() {
			if os.Getenv(AllowUnpinnedEnv) == "" {
				return Binary{}, fmt.Errorf(
					"REFUSING: mise is not on PATH, so `luajit` would be whichever build the "+
						"system has; set %s=1 to accept that", AllowUnpinnedEnv)
			}
			p, e := exec.LookPath("luajit")
			if e != nil {
				return Binary{}, fmt.Errorf("REFUSING: no luajit on PATH: %w", e)
			}
			path = p
		} else {
			out, e := runMiseWhich(ctx, root, "luajit")
			if e != nil {
				return Binary{}, e
			}
			path = out
		}
	default:
		return Binary{}, fmt.Errorf("REFUSING: unknown luajit selector %d", int(which))
	}
	return fingerprint(path)
}

func runMiseWhich(ctx context.Context, root, tool string) (string, error) {
	//nolint:gosec // a build tool asking the pin manager where a pinned tool lives
	cmd := exec.CommandContext(ctx, "mise", "-C", root, "which", tool)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("REFUSING: `mise which %s` failed; the pin is not resolvable: %w", tool, err)
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return "", fmt.Errorf("REFUSING: `mise which %s` printed nothing", tool)
	}
	return p, nil
}

func fingerprint(path string) (Binary, error) {
	f, err := os.Open(path)
	if err != nil {
		return Binary{}, fmt.Errorf("REFUSING: cannot read %s to fingerprint it: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	h := md5.New() //nolint:gosec // a build fingerprint, not a security primitive
	n, err := io.Copy(h, f)
	if err != nil {
		return Binary{}, fmt.Errorf("REFUSING: cannot fingerprint %s: %w", path, err)
	}
	if n == 0 {
		return Binary{}, fmt.Errorf("REFUSING: %s is EMPTY; its md5 would fingerprint nothing", path)
	}
	return Binary{Path: path, MD5: hex.EncodeToString(h.Sum(nil)), Size: n}, nil
}
