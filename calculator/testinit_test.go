package calculator

import (
	"context"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func init() {
	config.InitLogging(false)

	if err := poe.InitializeAll(raw.DataContext(context.Background()), raw.LatestVersion, assetcache.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		panic(err)
	}
}
