package exposition

import (
	"reflect"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
)

// TestParseBuildOverACalculatedBuild: An undo step parsed over an open build that has been calculated.
func TestParseBuildOverACalculatedBuild(t *testing.T) {
	initItemData(t)
	for _, file := range []string{"0001-necromancer-unknown.xml", "0011-saboteur-cyclone.xml", "0280-elementalist-fireball.xml"} {
		live := corpusBuild(t, file)
		step, err := builds.SerializeBuild(live)
		if err != nil {
			t.Fatal(err)
		}
		calculator.NewCalculator(*live).BuildOutput(calculator.OutputModeMain)
		live.Build.Level--
		live.SetMainSocketGroup(1 + live.Build.MainSocketGroup%max(1, len(live.Skills.SkillSets[0].Skills)))
		calculator.NewCalculator(*live).BuildOutput(calculator.OutputModeMain)
		liveXML, err := builds.SerializeBuild(live)
		if err != nil {
			t.Fatal(err)
		}

		want, err := builds.ParseBuild(step)
		if err != nil {
			t.Fatal(err)
		}
		got, err := builds.ParseBuildOver(live, string(liveXML), string(step))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: parsed over the calculated build differs from parsed alone", file)
		}
		a := calculator.NewCalculator(*got).BuildOutput(calculator.OutputModeMain).Player.Output
		b := calculator.NewCalculator(*want).BuildOutput(calculator.OutputModeMain).Player.Output
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s: calculates differently", file)
		}
	}
}
