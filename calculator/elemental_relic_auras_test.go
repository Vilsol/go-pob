package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSummonElementalRelicAurasAreEnabled pins ConfigOptions.lua:667-675.
func TestSummonElementalRelicAurasAreEnabled(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "5882-ladder-RandomBK_6066.RBK_FLAME_I.xml")

	if env.Player == nil || env.Player.ModDB == nil {
		t.Fatal("fixture is useless: no player actor")
	}

	want := map[string]bool{"Anger": false, "Hatred": false, "Wrath": false}
	for _, m := range env.Player.ModDB.Mods["SkillData"] {
		for _, tg := range m.Tags() {
			if sid, ok := tg.(*mod.SkillIDTag); ok {
				if _, tracked := want[sid.IDTag]; tracked {
					want[sid.IDTag] = true
				}
			}
		}
	}
	for k, got := range want {
		if !got {
			t.Errorf("no SkillData enable carrying SkillId %q — ConfigOptions.lua:667-675 enables "+
				"all three relic auras by default", k)
		}
	}
}
