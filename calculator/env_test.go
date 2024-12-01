package calculator

import (
	"context"
	"os"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob/cache"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
)

func init() {
	config.InitLogging(false)
}

func TestEmptyEnv(t *testing.T) {
	testCache := &EnvironmentCache{}

	err := poe.InitializeAll(context.Background(), raw.LatestVersion, cache.Disk(), nil)
	testza.AssertNoError(t, err)

	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := InitEnv(build, testCache, OutputModeMain)

	testza.AssertEqual(t, 101, len(cachedPlayerDB.(*ModDB).Mods))
	testza.AssertEqual(t, 60, len(cachedEnemyDB.(*ModDB).Mods))
	testza.AssertNil(t, cachedMinionDB)
}
