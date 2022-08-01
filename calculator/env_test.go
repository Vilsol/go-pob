package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"go-pob/builds"
)

func TestEmptyEnv(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	_, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := InitEnv(build, OutputModeMain)

	testza.AssertEqual(t, 101, len(cachedPlayerDB.(*ModDB).Mods))
	testza.AssertEqual(t, 60, len(cachedEnemyDB.(*ModDB).Mods))
	testza.AssertNil(t, cachedMinionDB)
}
