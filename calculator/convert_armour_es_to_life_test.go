package calculator

import (
	"math"
	"os"
	"strings"
	"testing"
)

// TestConvertArmourESToLife pins BOTH halves of "Gain Maximum Life instead of Maximum Energy Shield from Equipped Armour Items".
func TestConvertArmourESToLife(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	t.Run("producer/0791-elementalist-immortal-call", func(t *testing.T) {
		const build = "0791-elementalist-immortal-call.xml"
		env := corpusEnvForLoadedData(t, build)

		if !env.ModDB.Flag(nil, "ConvertArmourESToLife") {
			t.Fatalf("REFUSING: %s no longer sets ConvertArmourESToLife", build)
		}

		want := map[string]float64{}
		for _, slot := range []string{"Helmet", "Gloves", "Boots", "Body Armour", "Weapon 2", "Weapon 3"} {
			item := env.Player.ItemList[slot]
			if item == nil || item.ArmourData == nil || item.ArmourData.EnergyShield <= 0 {
				continue
			}
			want[slot+" ES to Life Conversion"] = item.ArmourData.EnergyShield
		}
		if len(want) == 0 {
			t.Fatalf("REFUSING: %s equips no armour piece with energy shield — the fixture "+
				"cannot exercise the producer", build)
		}

		got := map[string]float64{}
		for _, m := range env.ModDB.Tabulate("BASE", nil, "Life") {
			if src := string(m.Mod.GetSource()); strings.HasSuffix(src, " ES to Life Conversion") {
				got[src] += m.Value
			}
		}
		if len(got) != len(want) {
			t.Errorf("%s: %d ES-to-Life sources, expected %d (%v vs %v)", build, len(got), len(want), got, want)
		}
		for src, value := range want {
			if got[src] != value {
				t.Errorf("%s: %q contributes %v BASE Life, the slot carries %v energy shield",
					build, src, got[src], value)
			}
		}
	})

	t.Run("guard/0780-reliquarian-vaal-arc-of-oscillating-2", func(t *testing.T) {
		const build = "0780-reliquarian-vaal-arc-of-oscillating-2.xml"
		rec, ok := oracle[build]
		if !ok {
			t.Fatalf("REFUSING: %s has no record in the oracle blob", build)
		}
		env := corpusEnvForLoadedData(t, build)
		if !env.ModDB.Flag(nil, "ConvertArmourESToLife") {
			t.Fatalf("REFUSING: %s no longer sets ConvertArmourESToLife", build)
		}
		want, ok := rec.Output["EnergyShield"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: PoB reports no nonzero EnergyShield for %s", build)
		}
		got := env.Player.Output["EnergyShield"]
		if math.Abs(got-want) > math.Abs(want)*oracleDumpTolerance {
			t.Errorf("%s: EnergyShield = %v, PoB reports %v (ratio %.4f)", build, got, want, got/want)
		}
	})
}
