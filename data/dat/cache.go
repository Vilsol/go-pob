package dat

import (
	"io/fs"
	"os"
	"path/filepath"
)

// TODO Cache in browser when needed

var _ fs.FS = (*PullThroughCacheFS)(nil)

type PullThroughCacheFS struct {
	BaseCacheDir string
	Upstream     fs.FS
}

func NewPullThroughCacheFS(upstream fs.FS) *PullThroughCacheFS {
	dir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}

	baseCacheDir := filepath.Join(dir, "go-pob", "bundle-cache")
	if err := os.MkdirAll(baseCacheDir, 0777); err != nil {
		if !os.IsExist(err) {
			panic(err)
		}
	}

	return &PullThroughCacheFS{
		BaseCacheDir: baseCacheDir,
		Upstream:     upstream,
	}
}

func (p *PullThroughCacheFS) Open(name string) (fs.File, error) {
	fullPath := filepath.Join(p.BaseCacheDir, name)
	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	if stat != nil {
		return os.Open(fullPath)
	}

	return p.Upstream.Open(name)
}
