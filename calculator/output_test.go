package calculator

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"go-pob/builds"
)

func TestOutput(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	marshal, err := json.MarshalIndent(env, "", "  ")
	testza.AssertNoError(t, err)

	err = os.WriteFile("out.json", marshal, 0777)
	testza.AssertNoError(t, err)
}
