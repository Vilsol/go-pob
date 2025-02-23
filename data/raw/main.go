package raw

import (
	"context"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

const LatestVersion = "3.18"

type UpdateFunc func(data string)

type AssetLoaderWrapper struct {
	storage.Storage
}

func (a AssetLoaderWrapper) Get(key string) ([]byte, error) {
	return a.Storage.GetCache(key)
}

func (a AssetLoaderWrapper) Set(key string, value []byte) error {
	return a.Storage.SetCache(key, value)
}

func (a AssetLoaderWrapper) Exists(key string) bool {
	return a.Storage.ExistsInCache(key)
}

func InitializeAll(version string, updateFunc UpdateFunc) error {
	//nolint:wrapcheck
	return poe.InitializeAll(context.Background(), version, AssetLoaderWrapper{
		Storage: storage.Get(),
	}, func(data string) {
		updateFunc(data)
	})
}
