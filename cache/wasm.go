//go:build js

package cache

// TODO Implement wasm cache

type wasmCache struct {
}

func Disk() DiskCache {
	return wasmCache{}
}

func (d wasmCache) Get(key string) ([]byte, error) {
	return nil, nil
}

func (d wasmCache) Set(key string, value []byte) error {
	return nil
}

func (d wasmCache) Exists(key string) bool {
	return false
}
