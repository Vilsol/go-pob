//go:build !js

package cache

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Vilsol/go-pob/utils"

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
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"loading from cache",
		slog.String("key", key),
	)

	r, _, err := cache.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get key from cache: %s: %w", key, err)
	}

	if r == nil {
		return nil, nil
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from cache: %w", err)
	}

	return b, nil
}

func (d desktopCache) Set(key string, value []byte) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"storing in cache",
		slog.String("key", key),
		slog.Int("len", len(value)),
	)

	_ = cache.Remove(key)

	_, w, err := cache.Get(key)
	if err != nil {
		return fmt.Errorf("failed to set key on cache: %s: %w", key, err)
	}

	if w == nil {
		return fmt.Errorf("could not write to cache")
	}

	defer w.Close()

	if _, err := w.Write(value); err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	return nil
}

func (d desktopCache) Exists(key string) bool {
	return cache.Exists(key)
}

func InitializeDiskCache(_ func(key string) []byte, _ func(key string, value []byte), _ func(key string) bool) {
	// Only used in WASM
}
