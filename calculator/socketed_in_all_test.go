package calculator

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
)

// TestVaalPactWhileAllSocketedGemsAreRed: ModParser.lua:3246, CalcSetup.lua:1607-1610 / 1771-1773, ModStore.lua:741-746.
func TestVaalPactWhileAllSocketedGemsAreRed(t *testing.T) {
	requireGameData(t, "3.29")
	raw, err := os.ReadFile("../testdata/socketed-in/all-red-helmet.xml")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	keystones := func(xml string) []string {
		build, err := parsecache.ParseCached([]byte(xml))
		if err != nil {
			t.Fatal(err)
		}
		env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
		var out []string
		for _, v := range env.ModDB.List(nil, "Keystone") {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}

	if got := keystones(string(raw)); !slices.Contains(got, "Vaal Pact") {
		t.Errorf("all-red helmet group: keystones %v, want Vaal Pact", got)
	}

	mixed := strings.NewReplacer(
		`nameSpec="Determination" skillId="Determination" gemId="Metadata/Items/Gems/SkillGemDetermination"`,
		`nameSpec="Grace" skillId="Grace" gemId="Metadata/Items/Gems/SkillGemGrace"`,
	).Replace(string(raw))
	if mixed == string(raw) {
		t.Fatal("REFUSING: the control's gem swap did not apply")
	}
	if got := keystones(mixed); slices.Contains(got, "Vaal Pact") {
		t.Errorf("red+green helmet group: keystones %v, want no Vaal Pact", got)
	}
}
