package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

func init() {
	config.InitLogging()

	if err := raw.InitializeAll(raw.LatestVersion); err != nil {
		panic(err)
	}
}

func TestEmptyBuild(t *testing.T) {
	testza.AssertNoError(t, raw.InitializeAll(raw.LatestVersion))

	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	// Delete all skills
	build.Skills.SkillSets = []pob.SkillSet{}

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, 0.9523809523809523, env.Player.OutputTable[OutTableMainHand]["TotalMin"])
	testza.AssertEqual(t, 2.8571428571428568, env.Player.OutputTable[OutTableMainHand]["TotalMax"])
	testza.AssertEqual(t, 1.9047619047619047, env.Player.OutputTable[OutTableMainHand]["AverageHit"])
	testza.AssertEqual(t, 1.8857142857142855, env.Player.OutputTable[OutTableMainHand]["AverageDamage"])
	testza.AssertEqual(t, 2.2628571428571425, env.Player.OutputTable[OutTableMainHand]["TotalDPS"])
}

func TestFireballLevel1(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	build.WithMainSocketGroup(2)

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, float64(9), env.Player.Output["TotalMin"])
	testza.AssertEqual(t, float64(14), env.Player.Output["TotalMax"])
	testza.AssertEqual(t, 11.844999999999999, env.Player.Output["AverageHit"])
	testza.AssertEqual(t, 11.845, env.Player.Output["AverageDamage"])
	testza.AssertEqual(t, 15.793333333333333, env.Player.Output["TotalDPS"])

	//marshal, err := json.MarshalIndent(env, "", "  ")
	//testza.AssertNoError(t, err)
	//
	//err = os.WriteFile("out.json", marshal, 0777)
	//testza.AssertNoError(t, err)
}
