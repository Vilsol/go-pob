//go:build js

package cache

import (
	"context"
	"github.com/Vilsol/go-pob/utils"
	"log/slog"
)

var cache *wasmCache

type wasmCache struct {
	get    func(key string) []byte
	set    func(key string, value []byte)
	exists func(key string) bool
}

func Disk() DiskCache {
	// This gets invoked by tests
	if cache == nil {
		return &wasmCache{
			get: func(key string) []byte {
				return nil
			},
			set: func(key string, value []byte) {
			},
			exists: func(key string) bool {
				return false
			},
		}
	}

	return cache
}

func (d *wasmCache) Get(key string) ([]byte, error) {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"loading from cache",
		slog.String("key", key),
	)
	return d.get(key), nil
}

func (d *wasmCache) Set(key string, value []byte) error {
	slog.Log(
		context.Background(),
		utils.LevelTrace,
		"storing in cache",
		slog.String("key", key),
		slog.Int("len", len(value)),
	)
	d.set(key, value)
	return nil
}

func (d *wasmCache) Exists(key string) bool {
	return d.exists(key)
}

func InitializeDiskCache(get func(key string) []byte, set func(key string, value []byte), exists func(key string) bool) {
	cache = &wasmCache{
		get:    get,
		set:    set,
		exists: exists,
	}
}
