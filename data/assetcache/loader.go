package assetcache

import (
	"github.com/Vilsol/go-pob/storage"
)

type AssetLoaderWrapper struct {
	storage.Storage
	Prefetched map[string][]byte
}

func (a AssetLoaderWrapper) Get(key string) ([]byte, error) {
	if b, ok := a.Prefetched[key]; ok {
		return b, nil
	}
	return a.Storage.GetCache(CacheKey(key))
}

func (a AssetLoaderWrapper) Set(key string, value []byte) error {
	return a.Storage.SetCache(CacheKey(key), value)
}

func (a AssetLoaderWrapper) Exists(key string) bool {
	if _, ok := a.Prefetched[key]; ok {
		return true
	}
	return a.Storage.ExistsInCache(CacheKey(key))
}
