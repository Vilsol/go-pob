package calculator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
)

const focusedAmuletRaw = `Rarity: RARE
+4 Phys | Cast Speed Amulet
Focused Amulet
Crafted: true
LevelReq: 60
Implicits: 5
{crafted}Allocates Brutal Skewering
-1 Prefix Modifier allowed
-2 Suffix Modifiers allowed
Implicit Modifiers Cannot Be Changed
100% increased Explicit Modifier magnitudes
{prefix}+2 to Level of all Skill Gems
{prefix}+2 to Level of all Physical Skill Gems
{suffix}44% increased Cast Speed`

// TestModMagnitudeScalesExplicitLines pins Item.lua:1183-1283 and :1377-1432, which had no counterpart in this port at all.
func TestModMagnitudeScalesExplicitLines(t *testing.T) {
	requireGameData(t, "3.29")

	t.Run("a crafted item scales its explicit lines", func(t *testing.T) {
		item := &Item{}
		item.ParseRaw(focusedAmuletRaw)

		if len(item.ModMagnitudeMods) != 1 {
			t.Fatalf("collected %d magnitude mods, want 1: %+v", len(item.ModMagnitudeMods), item.ModMagnitudeMods)
		}
		if got := item.ModMagnitudeMods[0]; got.Quality != 100 || got.ModType != "explicit" ||
			len(got.Tags) != 0 || got.Multiplier != nil {
			t.Fatalf("magnitude mod = %+v, want quality 100 / modType explicit / no tags / no multiplier", got)
		}

		want := map[string]float64{"active_skill": 4, "physical": 4}
		got := map[string]float64{}
		for _, ml := range item.ExplicitModLines {
			for _, m := range ml.ModList {
				if m.Name() != "GemProperty" {
					continue
				}
				prop, ok := m.Value().List().(mod.GemProperty)
				if !ok || prop.Key != "level" {
					continue
				}
				key := "active_skill"
				for _, k := range prop.KeywordList {
					if k == "physical" {
						key = "physical"
					}
				}
				got[key] = prop.Value
			}
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("gem level for %s = %v, want %v (the unscaled value is 2)", k, got[k], v)
			}
		}

		if !hasIncMod(item.ExplicitModLines, "Speed", 88) {
			t.Error("cast speed did not reach 88% — the magnitude entry matched only some of its lines")
		}
	})

	t.Run("an ungated item is left alone", func(t *testing.T) {
		raw := strings.ReplaceAll(focusedAmuletRaw, "Crafted: true\n", "")
		item := &Item{}
		item.ParseRaw(raw)

		if item.Crafted || item.AdvancedCopy {
			t.Fatalf("control is not ungated: crafted=%v advancedCopy=%v", item.Crafted, item.AdvancedCopy)
		}
		if len(item.ModMagnitudeMods) != 1 {
			t.Fatalf("the LINE must still be collected — only its APPLICATION is gated; got %d", len(item.ModMagnitudeMods))
		}
		if !hasIncMod(item.ExplicitModLines, "Speed", 44) {
			t.Error("cast speed moved off 44% — the gate did not hold")
		}
	})

	t.Run("a range tag alone opens the gate", func(t *testing.T) {
		raw := strings.ReplaceAll(focusedAmuletRaw, "Crafted: true\n", "")
		raw = strings.ReplaceAll(raw, "{suffix}44% increased Cast Speed", "{suffix}{range:1}(40-44)% increased Cast Speed")
		item := &Item{}
		item.ParseRaw(raw)

		if item.Crafted {
			t.Fatal("this case must reach the gate through AdvancedCopy alone")
		}
		if !item.AdvancedCopy {
			t.Fatal("a {range:N} mod line did not set AdvancedCopy")
		}
		if !hasIncMod(item.ExplicitModLines, "Speed", 88) {
			t.Error("cast speed did not reach 88% — AdvancedCopy is not opening the gate")
		}
	})

	t.Run("0397's gem levels land on PoB's", func(t *testing.T) {
		if os.Getenv("CORPUS") != "true" {
			t.SkipNow()
		}
		const name = "0397-hierophant-bladefall-of-impaling.xml"
		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := (&Calculator{PoB: build}).BuildOutput(OutputModeMain)

		want := map[string]int{
			"Determination":  32,
			"MoltenShell":    32,
			"BladefallAltY":  36,
			"FrostblinkAltX": 26,
		}
		for _, as := range env.Player.ActiveSkillList {
			ae := as.ActiveEffect
			if ae == nil || ae.GrantedEffect == nil {
				continue
			}
			if w, ok := want[ae.GrantedEffect.Raw.ID]; ok {
				if ae.Level != w {
					t.Errorf("%s effective level = %d, want %d", ae.GrantedEffect.Raw.ID, ae.Level, w)
				}
				delete(want, ae.GrantedEffect.Raw.ID)
			}
		}
		for id := range want {
			t.Errorf("%s never appeared in the active skill list — the fixture is asking about nothing", id)
		}
	})
}

func hasIncMod(lines []ModLine, name string, value float64) bool {
	for _, ml := range lines {
		for _, m := range ml.ModList {
			if m.Name() == name && m.Type() == mod.TypeIncrease && m.Value().Float() == value {
				return true
			}
		}
	}
	return false
}
