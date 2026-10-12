package calculator

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

// TestKalandrasTouchMirrorsTheOtherRing pins CalcSetup.lua:1087-1125.
func TestKalandrasTouchMirrorsTheOtherRing(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	oracle, err := loadOutputOracle("../testdata/oracle-3.29/output-3.29.jsonl.gz")
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	corpusDir, _ := corpusNamesForLoadedData(t)

	const name = "1953-ladder-dominator_269_6824.John_Sanctum_.xml"
	rec, ok := oracle[name]
	if !ok {
		t.Fatalf("REFUSING: %s is not in the oracle blob", name)
	}
	file, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !containsBytes(file, "Kalandra&apos;s Touch") && !containsBytes(file, "Kalandra's Touch") {
		t.Fatalf("REFUSING: %s no longer equips Kalandra's Touch", name)
	}
	build, err := builds.ParseBuild(file)
	if err != nil {
		t.Fatalf("%v", err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())

	for _, k := range []string{"Int", "EnergyShield"} {
		want := rec.Output[k]
		got := env.Player.Output[k]
		if math.Abs(got-want) > 0.005*math.Abs(want) {
			t.Errorf("%s: %s = %v, want %v", name, k, got, want)
		}
	}
}

func containsBytes(hay []byte, needle string) bool {
	return len(needle) > 0 && len(hay) >= len(needle) && indexOfBytes(hay, needle) >= 0
}

func indexOfBytes(hay []byte, needle string) int {
	n := len(needle)
	for i := 0; i+n <= len(hay); i++ {
		if string(hay[i:i+n]) == needle {
			return i
		}
	}
	return -1
}
