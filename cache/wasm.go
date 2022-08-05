//go:build js

package cache

// TODO Implement wasm cache

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
	log.Trace().Str("key", key).Msg("loading from cache")
	return d.get(key), nil
}

func (d *wasmCache) Set(key string, value []byte) error {
	log.Trace().Str("key", key).Int("len", len(value)).Msg("storing in cache")
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
