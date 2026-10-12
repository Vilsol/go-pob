package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestAurasAffectEnemiesReachesTheEnemyModDB pins ModParser.lua:3996 and CalcPerform.lua:2451-2464 together.
func TestAurasAffectEnemiesReachesTheEnemyModDB(t *testing.T) {
	t.Run("the flag parses", func(t *testing.T) {
		requireGameData(t, "3.29")

		const line = "Auras from your Skills which affect Allies also affect Enemies"
		entry := ParseMod(line, false)
		if entry.Extra != "" {
			t.Fatalf("ModParser.lua:3996 — %q left %q unparsed", line, entry.Extra)
		}
		if len(entry.ModList) != 1 {
			t.Fatalf("ModParser.lua:3996 — %q produced %d mods, want 1", line, len(entry.ModList))
		}
		if got := entry.ModList[0].Name(); got != "AurasAffectEnemies" {
			t.Fatalf("mod name = %s, want AurasAffectEnemies", got)
		}
	})

	t.Run("the aura reaches the enemy", func(t *testing.T) {
		if os.Getenv("CORPUS") != "true" {
			t.SkipNow()
		}
		requireGameData(t, "3.29")

		const name = "0161-assassin-viper-strike-of-the-mamba.xml"
		xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatal(err)
		}
		env := (&Calculator{PoB: build, AlignCalcsSkillToMain: true}).BuildOutput(OutputModeCalcs)
		if env == nil || env.Player == nil {
			t.Fatal("no CALCS env")
		}
		got := env.Player.OutputTable[OutTableMainHand]
		if len(got) == 0 {
			t.Fatal("no MainHand table — this build is an attack build and must have one")
		}

		for _, tc := range []struct {
			key  string
			want float64
			role string
		}{
			{"LightningEffMult", 0.25, "the subject: 50 + 36 aura, clamped to LightningResistMax 75"},
			{"ChaosEffMult", 1.01, "control: 30 EnemyConfig - 31 Despair, takes no ElementalResist"},
			{"PhysicalEffMult", 0.2975, "control: armour and PhysicalDamageReduction, no resist"},
		} {
			if math.Abs(got[tc.key]-tc.want) > math.Abs(tc.want)*0.005 {
				t.Errorf("%s = %g, want %g (%s)", tc.key, got[tc.key], tc.want, tc.role)
			}
		}
	})
}
