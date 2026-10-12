package calculator

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEnergyBladeIsDerivedFromSocketedGem pins CalcSetup.lua:1812-1822.
func TestEnergyBladeIsDerivedFromSocketedGem(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	load := func(t *testing.T, name string) []byte {
		t.Helper()
		file, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			t.Fatalf("%v", err)
		}
		return file
	}
	run := func(t *testing.T, xml []byte) *Environment {
		t.Helper()
		build, err := builds.ParseBuild(xml)
		if err != nil {
			t.Fatalf("%v", err)
		}
		return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	}

	t.Run("one-handed-speed", func(t *testing.T) {
		const subject = "10050-ladder-Fujindesu_5297.yamatonerfs.xml"
		rec, ok := oracle[subject]
		if !ok {
			t.Fatalf("REFUSING: %s is not in the oracle blob", subject)
		}
		want, ok := rec.MainHand["Speed"]
		if !ok || want == 0 {
			t.Fatalf("REFUSING: %s publishes no MainHand.Speed; the fixture has moved", subject)
		}

		env := run(t, load(t, subject))
		if !env.ModDB.Conditions[conditionAffectedByEnergyBlade] {
			t.Fatalf("%s: AffectedByEnergyBlade is not set; the gem sweep did not fire", subject)
		}
		if got := env.Player.WeaponData1.CritChance; got != 7 {
			t.Errorf("%s: weaponData1.CritChance = %v, want the Energy Blade base's 7", subject, got)
		}
		if got := env.Player.OutputTable[OutTableMainHand]["Speed"]; math.Abs(got-want) > math.Abs(want)*1e-9 {
			t.Errorf("%s: MainHand.Speed = %v, want %v (PoB)", subject, got, want)
		}
	})

	for _, tc := range []struct {
		name  string
		build string
		want  string
	}{
		{"two-handed-base", "0309-berserker-flicker-strike-2.xml", energyBlade + " Two Handed"},
		{"one-handed-base", "4224-ladder-Blargamus20_2236.Blargnado.xml", energyBlade + " One Handed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := run(t, load(t, tc.build))
			if !env.ModDB.Conditions[conditionAffectedByEnergyBlade] {
				t.Fatalf("%s: AffectedByEnergyBlade is not set; the gem sweep did not fire", tc.build)
			}
			got := env.Player.ItemList["Weapon 1"]
			if got == nil || got.Name != tc.want {
				t.Errorf("%s: Weapon 1 is %v, want %q", tc.build, got, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name  string
		build string
		stat  string
	}{
		{"abyssal-jewel-life", "10050-ladder-Fujindesu_5297.yamatonerfs.xml", "Life"},
		{"abyssal-jewel-life-2", "13574-ladder-frozenstar_studio_0831.StackSomeHP.xml", "Life"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", tc.build)
			}
			want, ok := rec.Output[tc.stat]
			if !ok || want == 0 {
				t.Fatalf("REFUSING: %s publishes no %s; the fixture has moved", tc.build, tc.stat)
			}
			env := run(t, load(t, tc.build))
			if got := env.Player.Output[tc.stat]; math.Abs(got-want) > math.Abs(want)*1e-9 {
				t.Errorf("%s: %s = %v, want %v (PoB)", tc.build, tc.stat, got, want)
			}
		})
	}

	t.Run("disabled-gem-control", func(t *testing.T) {
		const subject = "10050-ladder-Fujindesu_5297.yamatonerfs.xml"
		xml := load(t, subject)
		enabled := []byte(`<Gem gemId="Metadata/Items/Gems/SkillGemEnergyBlade" enabled="true" skillId="EnergyBlade"`)
		if !bytes.Contains(xml, enabled) {
			t.Fatalf("REFUSING: %s no longer contains the enabled Energy Blade gem this control mutates", subject)
		}
		disabled := bytes.Replace(xml, enabled,
			[]byte(`<Gem gemId="Metadata/Items/Gems/SkillGemEnergyBlade" enabled="false" skillId="EnergyBlade"`), 1)

		env := run(t, disabled)
		if env.ModDB.Conditions[conditionAffectedByEnergyBlade] {
			t.Fatalf("%s: AffectedByEnergyBlade set from a DISABLED gem", subject)
		}
		if got := env.Player.ItemList["Weapon 1"]; got == nil || got.Name == energyBlade+" One Handed" {
			t.Errorf("%s: Weapon 1 was replaced with an Energy Blade despite the gem being disabled", subject)
		}
	})
}
