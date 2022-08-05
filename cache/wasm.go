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
	return cache
}

func (d *wasmCache) Get(key string) ([]byte, error) {
	return d.get(key), nil
}

func (d *wasmCache) Set(key string, value []byte) error {
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
