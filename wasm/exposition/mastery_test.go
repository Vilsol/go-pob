package exposition

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
)

// TestSelectMasteryEffectMatchesTheSavedChoice: Choosing another effect for 0280's mastery 63268 calculates as 0280 saved with that effect.
func TestSelectMasteryEffectMatchesTheSavedChoice(t *testing.T) {
	initItemData(t)
	const file = "0280-elementalist-fireball.xml"
	build := corpusBuild(t, file)
	var next int64
	for _, o := range GetMasteryOptions(build, 63268) {
		if !o.Selected {
			next = o.Effect
			break
		}
	}
	if next == 0 {
		t.Fatal("REFUSING: no other effect offered for 63268")
	}
	if err := SelectMasteryEffect(build, 63268, next, nil); err != nil {
		t.Fatal(err)
	}
	if GetMasterySelections(build)[63268] != next {
		t.Fatalf("selections = %v", GetMasterySelections(build))
	}
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, file))
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(xml), `masteryEffects="{11505,55308},{32241,38454},{63268,6216},`,
		`masteryEffects="{11505,55308},{32241,38454},{63268,`+strconv.FormatInt(next, 10)+`},`, 1)
	if edited == string(xml) {
		t.Fatal("REFUSING: tree 2's masteryEffects not found")
	}
	saved, err := builds.ParseBuild([]byte(edited))
	if err != nil {
		t.Fatal(err)
	}
	got, want := GetMiscCalculator(build).Base().Output, GetMiscCalculator(saved).Base().Output
	base := GetMiscCalculator(corpusBuild(t, file)).Base().Output
	differs := false
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v after choosing, %v saved with the choice", k, got[k], v)
		}
		if base[k] != v {
			differs = true
		}
	}
	if !differs {
		t.Errorf("vacuity: effect %d calculates as 6216 did", next)
	}
}
