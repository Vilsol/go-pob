package parsecache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

type Store struct {
	dir string
	fp  string
}

var ErrDisabled = errors.New("parsecache: disabled")

func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("parsecache: no home directory: %w", err)
	}
	return filepath.Join(home, "go-pob-corpus", "parsed-builds"), nil
}

func Open() (*Store, error) {
	dir := os.Getenv("PARSE_CACHE")
	switch {
	case strings.EqualFold(dir, "off"), strings.EqualFold(dir, "0"), strings.EqualFold(dir, "false"):
		return nil, fmt.Errorf("parsecache: %w", ErrDisabled)
	case dir == "":
		d, err := DefaultDir()
		if err != nil {
			return nil, fmt.Errorf("parsecache: no home directory: %w", err)
		}
		dir = d
	}

	fp, err := ParserFingerprint()
	if err != nil {
		return nil, err
	}
	//nolint:gosec // G703: PARSE_CACHE (or ~/go-pob-corpus) is an operator-supplied directory.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("parsecache: %w", err)
	}
	return &Store{dir: dir, fp: fp}, nil
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) path(rawXML []byte) string {
	h := sha256.New()
	h.Write([]byte(s.fp))
	h.Write([]byte{0})
	h.Write(rawXML)
	key := hex.EncodeToString(h.Sum(nil))
	return filepath.Join(s.dir, s.fp, key[:2], key[2:]+".gpbc")
}

func (s *Store) Parse(rawXML []byte) (*pob.PathOfBuilding, error) {
	p := s.path(rawXML)
	//nolint:gosec // G703: p is a SHA-256 hex name under the operator's own PARSE_CACHE directory;
	if blob, err := os.ReadFile(p); err == nil {
		var out pob.PathOfBuilding
		if err := Decode(blob, &out); err == nil {
			return &out, nil
		}
	}

	build, err := builds.ParseBuild(rawXML)
	if err != nil {
		return nil, err
	}
	s.store(p, build)
	return build, nil
}

func (s *Store) store(path string, build *pob.PathOfBuilding) {
	blob, err := Encode(build)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return
	}
	tmp := f.Name()
	if _, err := f.Write(blob); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
	}
}

var Shared = sync.OnceValue(func() *Store {
	s, err := Open()
	if err != nil {
		return nil
	}
	return s
})

func ParseCached(rawXML []byte) (*pob.PathOfBuilding, error) {
	if s := Shared(); s != nil {
		return s.Parse(rawXML)
	}
	return builds.ParseBuild(rawXML)
}
