package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

func init() {
	config.InitLogging(false)

	if err := raw.InitializeAll(raw.LatestVersion, nil); err != nil {
		panic(err)
	}
}

func TestOnslaught(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	// Delete all skills
	build.Skills.SkillSets = []pob.SkillSet{}

	build.SetConfigOption(pob.Input{
		Name:    "buffOnslaught",
		Boolean: utils.Ptr(true),
	})

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, 0.9523809523809523, env.Player.OutputTable[OutTableMainHand]["TotalMin"])
	testza.AssertEqual(t, 2.8571428571428568, env.Player.OutputTable[OutTableMainHand]["TotalMax"])
	testza.AssertEqual(t, 1.9047619047619047, env.Player.OutputTable[OutTableMainHand]["AverageHit"])
	testza.AssertEqual(t, 1.8857142857142855, env.Player.OutputTable[OutTableMainHand]["AverageDamage"])
	testza.AssertEqual(t, 2.715428571428571, env.Player.OutputTable[OutTableMainHand]["TotalDPS"])
}
