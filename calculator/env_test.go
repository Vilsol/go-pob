package calculator

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"go-pob/builds"
)

func TestEmptyEnv(t *testing.T) {
	file, _ := os.ReadFile("./testdata/builds/Fireball.xml")
	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB := InitEnv(build, OutputModeMain)

	testza.AssertEqual(t, 101, len(cachedPlayerDB.Mods))
	testza.AssertEqual(t, 60, len(cachedEnemyDB.Mods))
	testza.AssertNil(t, cachedMinionDB)

	_ = cachedPlayerDB
	_ = cachedEnemyDB
	_ = cachedMinionDB

	marshal, err := json.MarshalIndent(env, "", "  ")
	os.WriteFile("out.json", marshal, 0777)
}
