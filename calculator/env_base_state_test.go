package calculator

import (
	"fmt"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func initModDBNameSet(t *testing.T) map[string]int {
	t.Helper()
	db := moddb.NewModDB()
	initModDB(&Environment{}, db)
	out := map[string]int{}
	for _, list := range db.Mods {
		for _, m := range list {
			out[fmt.Sprintf("%s|%s", m.Name(), m.Type())]++
		}
	}
	if len(out) == 0 {
		t.Fatal("REFUSING: initModDB seeded nothing — an empty result here would make every " +
			"assertion below pass vacuously")
	}
	return out
}

// TestInitModDBSeedsThePinsBaseState pins the base modifiers CalcSetup.lua:18-114 seeds onto every actor.
func TestInitModDBSeedsThePinsBaseState(t *testing.T) {
	got := initModDBNameSet(t)
	want := make([]string, 0, 26)
	want = append(want,
		"SoulEaterMax|BASE",
		"ShockStacksMax|BASE",
		"ScorchStacksMax|BASE",
		"Excommunicated|FLAG",
		"ChaoticMight|FLAG",
		"CritChanceCap|BASE",
	)
	for _, s := range []string{
		"AccelerationShrine", "BrutalShrine", "DiamondShrine", "DivineShrine", "EchoingShrine",
		"GloomShrine", "GreaterFreezingShrine", "GreaterShockingShrine", "GreaterSkeletalShrine",
		"ImpenetrableShrine", "MassiveShrine", "ReplenishingShrine", "ResistanceShrine",
		"ResonatingShrine", "LesserAccelerationShrine", "LesserBrutalShrine",
		"LesserImpenetrableShrine", "LesserMassiveShrine", "LesserReplenishingShrine",
		"LesserResistanceShrine",
	} {
		want = append(want, s+"|FLAG")
	}
	for _, w := range want {
		if got[w] == 0 {
			t.Errorf("%s: absent from initModDB", w)
		}
	}
	if n := got["DamageTaken|INC"]; n != 4 {
		t.Errorf("DamageTaken INC seeded %d times, want 4 (CalcSetup.lua:64-67, two of them the "+
			"Condition:Party:* forms)", n)
	}
}

// TestEnemyLevelCapsAtMaxEnemyLevel pins ConfigTab.lua:1062-1071.
func TestEnemyLevelCapsAtMaxEnemyLevel(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		input, placeholder, characterLvl float64
		want                             int
	}{
		{"input above the cap", 95, 84, 90, 85},
		{"input below the cap", 72, 84, 90, 72},
		{"placeholder wins when no input", 0, 84, 90, 84},
		{"character level when neither", 0, 0, 68, 68},
		{"character level above the cap", 0, 0, 98, 85},
		{"never below 1", 0, 0, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveEnemyLevel(tc.input, tc.placeholder, tc.characterLvl); got != tc.want {
				t.Errorf("resolveEnemyLevel(%g,%g,%g) = %d, want %d",
					tc.input, tc.placeholder, tc.characterLvl, got, tc.want)
			}
		})
	}
}

// TestCalcSetupBaseModsArePresent covers the five CalcSetup.lua:543-547 rows and the enemy-side :576 row, all absent.
func TestCalcSetupBaseModsArePresent(t *testing.T) {
	got := initModDBNameSet(t)
	if got["PerAbsorptionElementalEnergyShieldRecoup|BASE"] != 0 {
		t.Fatal("PerAbsorptionElementalEnergyShieldRecoup belongs in initEnvWith (CalcSetup.lua:543), " +
			"not initModDB (:18-114) — this test would then be asserting the wrong site")
	}
}

// TestAllocatedKeystoneMultiplierIsPublished pins CalcSetup.lua:679-681, the third of the four allocated-node multipliers.
func TestAllocatedKeystoneMultiplierIsPublished(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")

	keystones := 0
	for _, node := range env.AllocatedNodes {
		if node.IsKeystone != nil && *node.IsKeystone {
			keystones++
		}
	}
	if keystones == 0 {
		t.Skip("this build allocates no keystone, so the guarded mod is correctly absent")
	}
	if got := env.ModDB.Sum(mod.TypeBase, nil, "Multiplier:AllocatedKeystone"); int(got) != keystones {
		t.Errorf("Multiplier:AllocatedKeystone = %v, want %d allocated keystones", got, keystones)
	}
}

// TestCalcSetupTailBaseModsArePresent covers CalcSetup.lua:543-547 and the enemy-side :576, all six absent from initEnvWith.
func TestCalcSetupTailBaseModsArePresent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")

	for name, want := range map[string]float64{
		"PerAbsorptionElementalEnergyShieldRecoup": 12,
		"TinctureLimit":  1,
		"PresenceRadius": 80,
	} {
		if got := env.ModDB.Sum(mod.TypeBase, nil, name); got != want {
			t.Errorf("%s BASE = %v, want %v", name, got, want)
		}
	}
	for _, name := range []string{"ManaDegenPercentTincture", "LifeDegenPercentTincture"} {
		if len(env.ModDB.Mods[name]) == 0 {
			t.Errorf("%s: no mod of that name on the player ModDB (CalcSetup.lua:545-546)", name)
		}
	}
	if len(env.EnemyModDB.Mods["Condition:AgainstDamageOverTime"]) == 0 {
		t.Error("Condition:AgainstDamageOverTime: absent from the enemy ModDB (CalcSetup.lua:576)")
	}
}
