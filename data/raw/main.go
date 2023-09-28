package raw

import (
	"context"
	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob/cache"
)

const LatestVersion = "3.18"

type UpdateFunc func(data string)

func InitializeAll(version string, updateFunc UpdateFunc) error {
	return poe.InitializeAll(context.Background(), version, cache.Disk(), func(data string) {
		updateFunc(data)
	})
}
