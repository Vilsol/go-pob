package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
)

// TestMineStackDoesNotLeakAcrossBuilds: stackMineMods once stripped effectStackVar from the skill data's shared tag.
func TestMineStackDoesNotLeakAcrossBuilds(t *testing.T) {
	requireGameData(t, "3.29")
	dir := corpusDirForLoadedData(t)
	eval := func(name string) *Environment {
		file, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := parsecache.ParseCached(file)
		if err != nil {
			t.Fatal(err)
		}
		return BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, NewEnvironmentCache())
	}
	const victim = "15603-elementalist-glacial-cascade-of-the-fissure.xml"
	fresh := eval(victim).Player.Output["DoubleDamageChance"]
	if fresh == 0 {
		t.Fatalf("control: %s has no DoubleDamageChance", victim)
	}
	eval("15531-elementalist-flame-wall.xml")
	if got := eval(victim).Player.Output["DoubleDamageChance"]; got != fresh {
		t.Fatalf("DoubleDamageChance after 15531: %v, fresh: %v", got, fresh)
	}
}
