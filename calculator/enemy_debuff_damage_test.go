package calculator

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestEnemyDebuffsLowerEnemyDamage pins CalcSetup.lua:68-71 together with the two config options that set their conditions.
func TestEnemyDebuffsLowerEnemyDamage(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	for _, tc := range []struct {
		name   string
		build  string
		config string
	}{
		{"debilitated", "0060-assassin-icicle-mine.xml", "conditionEnemyDebilitated"},
		{"malignant-madness", "0397-hierophant-bladefall-of-impaling.xml", "conditionEnemyMalignantMadness"},
		{"neither", "0103-elementalist-mana-infused-staff.xml", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok := oracle[tc.build]
			if !ok {
				t.Fatalf("REFUSING: %s is not in the oracle blob", tc.build)
			}
			file, err := os.ReadFile(filepath.Join(corpusDir, tc.build))
			if err != nil {
				t.Fatalf("%v", err)
			}
			if tc.config != "" && !bytes.Contains(file, []byte(tc.config)) {
				t.Fatalf("REFUSING: %s no longer sets %s; the fixture has stopped exercising it",
					tc.build, tc.config)
			}

			build, err := builds.ParseBuild(file)
			if err != nil {
				t.Fatalf("%v", err)
			}
			env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

			for _, key := range []string{
				"PhysicalEnemyDamageMult",
				"FireEnemyDamageMult",
				"PhysicalEnemyDamage",
				"totalEnemyDamage",
			} {
				want := rec.Output[key]
				got := env.Player.Output[key]
				if math.Abs(got-want) > math.Abs(want)*1e-9 {
					t.Errorf("%s: %s = %v, want %v (PoB)", tc.build, key, got, want)
				}
			}
		})
	}
}
