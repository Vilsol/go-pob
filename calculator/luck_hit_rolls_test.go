package calculator

import (
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"
)

// TestLuckHitRollsParse covers the five ModParser.lua:4027-4032 rows that had no counterpart here.
func TestLuckHitRollsParse(t *testing.T) {
	for _, tc := range []struct {
		line  string
		flags []string
	}{
		{"Damage with Hits is Unlucky", []string{"UnluckyHits"}},
		{"Chaos Damage with Hits is Lucky", []string{"ChaosLuckyHits"}},
		{"Your Damage with Hits is Lucky while on Low Life", []string{"LuckyHits"}},
		{"Lightning Damage with Hits is Lucky if you've Blocked Spell Damage Recently",
			[]string{"LightningLuckHits"}},
		{"Cold Damage with Hits is Lucky if you have Suppressed Spell Damage Recently",
			[]string{"ColdLuckyHits"}},
		{"Fire Damage with Hits is Lucky if you've Blocked an Attack Recently",
			[]string{"FireLuckyHits"}},
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

// TestLuckHitRollsOutput pins CalcOffence.lua:3378-3397 against the oracle, on the real pipeline.
func TestLuckHitRollsOutput(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	for _, tc := range []struct {
		build string
		key   string
		want  float64
	}{
		{"0633-saboteur-lancing-steel-of-spraying", "DamageRolls", -1},
		{"0633-saboteur-lancing-steel-of-spraying", "CritDamageRolls", -1},
		{"0039-hierophant-static-strike", "DamageRolls", 1},
		{"0230-warden-spectral-throw-of-materialising", "DamageRolls", 1},
	} {
		t.Run(tc.build+"/"+tc.key, func(t *testing.T) {
			rec, ok := oracle[tc.build+".xml"]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", tc.build)
			}
			if rec.MainHand[tc.key] != tc.want {
				t.Fatalf("REFUSING: the oracle says MainHand.%s = %v on %s, not %v — the fixture "+
					"is describing a build the blob no longer contains",
					tc.key, rec.MainHand[tc.key], tc.build, tc.want)
			}
			env := corpusEnvForLoadedData(t, tc.build)
			if got := env.Player.OutputTable[OutTableMainHand][tc.key]; got != tc.want {
				t.Errorf("%s: MainHand.%s = %v, want %v", tc.build, tc.key, got, tc.want)
			}
		})
	}
}
