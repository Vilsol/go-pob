//go:build !js

package cache

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/djherbis/fscache.v0"
)

var cache *fscache.FSCache

func init() {
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

	cache, err = fscache.New(baseCacheDir, 0755, time.Hour*24*30) // 30 day cache
	if err != nil {
		panic(err)
	}
}

type desktopCache struct {
}

func Disk() DiskCache {
	return desktopCache{}
}

func (d desktopCache) Get(key string) ([]byte, error) {
	log.Trace().Str("key", key).Msg("loading from cache")

	r, _, err := cache.Get(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get key from cache: "+key)
	}

	if r == nil {
		return nil, nil
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read from cache")
	}

	return b, nil
}

func (d desktopCache) Set(key string, value []byte) error {
	log.Trace().Str("key", key).Int("len", len(value)).Msg("storing in cache")

	_ = cache.Remove(key)

	_, w, err := cache.Get(key)
	if err != nil {
		return errors.Wrap(err, "failed to set key on cache: "+key)
	}

	if w == nil {
		return errors.New("could not write to cache")
	}

	defer w.Close()

	if _, err := w.Write(value); err != nil {
		return errors.Wrap(err, "failed to write to cache")
	}

	return nil
}

func (d desktopCache) Exists(key string) bool {
	return cache.Exists(key)
}
