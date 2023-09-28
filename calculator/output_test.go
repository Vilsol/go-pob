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
	"github.com/Vilsol/go-pob/pob"
)

func init() {
	config.InitLogging(false)

	if err := poe.InitializeAll(context.Background(), raw.LatestVersion, cache.Disk(), nil); err != nil {
		panic(err)
	}
}

func TestEmptyBuild(t *testing.T) {
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

	build = build.WithMainSocketGroup(2)

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, float64(9), env.Player.Output["TotalMin"])
	testza.AssertEqual(t, float64(14), env.Player.Output["TotalMax"])
	testza.AssertEqual(t, 11.844999999999999, env.Player.Output["AverageHit"])
	testza.AssertEqual(t, 11.845, env.Player.Output["AverageDamage"])
	testza.AssertEqual(t, 15.793333333333333, env.Player.Output["TotalDPS"])
}

func TestFireballLevel20(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	build = build.WithMainSocketGroup(3)

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, float64(1640), env.Player.Output["TotalMin"])
	testza.AssertEqual(t, float64(2460), env.Player.Output["TotalMax"])
	testza.AssertEqual(t, 2111.5, env.Player.Output["AverageHit"])
	testza.AssertEqual(t, 2111.5, env.Player.Output["AverageDamage"])
	testza.AssertEqual(t, 2815.333333333333, env.Player.Output["TotalDPS"])
}

func TestFireballLevel1AddedColdLevel1(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	build = build.WithMainSocketGroup(4)

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, float64(24), env.Player.Output["TotalMin"])
	testza.AssertEqual(t, float64(36), env.Player.Output["TotalMax"])
	testza.AssertEqual(t, 30.9, env.Player.Output["AverageHit"])
	testza.AssertEqual(t, 30.9, env.Player.Output["AverageDamage"])
	testza.AssertEqual(t, 41.199999999999996, env.Player.Output["TotalDPS"])
}

func TestFireballLevel20AddedColdLevel1(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	build = build.WithMainSocketGroup(5)

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, float64(1655), env.Player.Output["TotalMin"])
	testza.AssertEqual(t, float64(2482), env.Player.Output["TotalMax"])
	testza.AssertEqual(t, 2130.555, env.Player.Output["AverageHit"])
	testza.AssertEqual(t, 2130.555, env.Player.Output["AverageDamage"])
	testza.AssertEqual(t, 2840.74, env.Player.Output["TotalDPS"])
}

func TestFireballLevel20AddedColdLevel20(t *testing.T) {
	file, err := os.ReadFile("../testdata/builds/Fireball.xml")
	testza.AssertNoError(t, err)

	build, err := builds.ParseBuild(file)
	testza.AssertNoError(t, err)

	build = build.WithMainSocketGroup(6)

	calculator := &Calculator{PoB: build}
	env := calculator.BuildOutput(OutputModeMain)

	testza.AssertEqual(t, float64(2202), env.Player.Output["TotalMin"])
	testza.AssertEqual(t, float64(3304), env.Player.Output["TotalMax"])
	testza.AssertEqual(t, 2835.5899999999997, env.Player.Output["AverageHit"])
	testza.AssertEqual(t, 2835.5899999999992, env.Player.Output["AverageDamage"])
	testza.AssertEqual(t, 3780.7866666666655, env.Player.Output["TotalDPS"])
}
