package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
)

func init() {
	config.InitLogging()
}

func TestEmptyEnv(t *testing.T) {
	err := raw.InitializeAll(raw.LatestVersion)
	testza.AssertNoError(t, err)

	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := InitEnv(build, OutputModeMain)

	testza.AssertEqual(t, 101, len(cachedPlayerDB.(*ModDB).Mods))
	testza.AssertEqual(t, 60, len(cachedEnemyDB.(*ModDB).Mods))
	testza.AssertNil(t, cachedMinionDB)
}
