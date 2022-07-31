package calculator

import (
	"encoding/json"
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

	env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := InitEnv(build, OutputModeMain)

	testza.AssertEqual(t, 101, len(cachedPlayerDB.(*ModDB).Mods))
	testza.AssertEqual(t, 60, len(cachedEnemyDB.(*ModDB).Mods))
	testza.AssertNil(t, cachedMinionDB)

	_ = cachedPlayerDB
	_ = cachedEnemyDB
	_ = cachedMinionDB

	marshal, err := json.MarshalIndent(env, "", "  ")
	testza.AssertNoError(t, err)

	err = os.WriteFile("out.json", marshal, 0777)
	testza.AssertNoError(t, err)
}
