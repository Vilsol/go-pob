package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestLuckRollFlagsParse covers ModParser.lua:4034, :4095-4101 and :4102-4106.
func TestLuckRollFlagsParse(t *testing.T) {
	for _, tc := range []struct {
		line  string
		flags []string
	}{
		{"Chance to Block is Lucky", []string{
			"BlockChanceIsLucky", "ProjectileBlockChanceIsLucky",
			"SpellBlockChanceIsLucky", "SpellProjectileBlockChanceIsLucky"}},
		{"Chance to Block is Unlucky", []string{
			"BlockChanceIsUnlucky", "ProjectileBlockChanceIsUnlucky",
			"SpellBlockChanceIsUnlucky", "SpellProjectileBlockChanceIsUnlucky"}},
		{"Chance to Block Attack Damage is Lucky", []string{
			"BlockChanceIsLucky", "ProjectileBlockChanceIsLucky"}},
		{"Chance to Block Attack Damage is Unlucky", []string{
			"BlockChanceIsUnlucky", "ProjectileBlockChanceIsUnlucky"}},
		{"Chance to Block Spell Damage is Lucky", []string{
			"SpellBlockChanceIsLucky", "SpellProjectileBlockChanceIsLucky"}},
		{"Chance to Block Spell Damage is Unlucky", []string{
			"SpellBlockChanceIsUnlucky", "SpellProjectileBlockChanceIsUnlucky"}},
		{"Your Lucky or Unlucky effects use the best or worst from three rolls instead of two",
			[]string{"ExtremeLuck"}},
		{"Your Lucky or Unlucky effects are instead Unexciting", []string{"Unexciting"}},
		{"Chance to Block Attack or Spell Damage is Lucky if you've Blocked Recently", []string{
			"BlockChanceIsLucky", "ProjectileBlockChanceIsLucky",
			"SpellBlockChanceIsLucky", "SpellProjectileBlockChanceIsLucky"}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			mods, extra := parseMod(tc.line, 1)
			testza.AssertEqual(t, "", extra, "%q must parse completely", tc.line)
			got := make([]string, 0, len(mods))
			for _, m := range mods {
				got = append(got, m.Name())
			}
			testza.AssertEqual(t, tc.flags, got, "%q", tc.line)
		})
	}
}

// TestBlockLuckReachesEffectiveBlockChance is the ACTIVATION rung.
func TestBlockLuckReachesEffectiveBlockChance(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}

	for _, build := range []string{
		"0016-gladiator-dual-strike-of-ambidexterity.xml",
		"0092-gladiator-war-banner.xml",
		"0276-gladiator-eviscerate.xml",
	} {
		t.Run(build, func(t *testing.T) {
			rec, ok := oracle[build]
			if !ok || rec.TreeSources < oracleMinTreeSources {
				t.Fatalf("REFUSING: %s is not graded", build)
			}
			raw, eff := rec.Output["BlockChance"], rec.Output["EffectiveBlockChance"]
			if raw <= 0 || eff <= raw {
				t.Fatalf("REFUSING: PoB reports BlockChance %v and EffectiveBlockChance %v for %s; "+
					"no luck transform is being applied there", raw, eff, build)
			}

			env := corpusEnvForLoadedData(t, build)
			var absolute int
			for _, key := range []string{"BlockChance", "ProjectileBlockChance", "SpellBlockChance", "SpellProjectileBlockChance"} {
				ourRaw := env.Player.Output[key]
				got := env.Player.Output["Effective"+key]

				want := (1 - math.Pow(1-ourRaw/100, 2)) * 100
				if math.Abs(got-want) > 1e-9 {
					t.Errorf("Effective%s = %v, want %v = lucky roll of our own %s %v "+
						"(CalcDefence.lua:737-767)", key, got, want, key, ourRaw)
				}

				if oracleClose(ourRaw, rec.Output[key], oracleDumpTolerance) {
					absolute++
					if pobEff := rec.Output["Effective"+key]; !oracleClose(got, pobEff, oracleDumpTolerance) {
						t.Errorf("Effective%s = %v, want %v; our raw %s already matches PoB's",
							key, got, pobEff, key)
					}
				}
			}
			if absolute == 0 {
				t.Errorf("REFUSING: no key on %s has a raw value matching PoB's, so this fixture "+
					"can only check self-consistency", build)
			}
		})
	}
}
