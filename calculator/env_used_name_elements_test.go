package calculator

import (
	"strings"
	"testing"
)

func directModNameElements(name string) uint8 {
	lower := strings.ToLower(name)
	var bits uint8
	if fireOrIgnitePattern.MatchString(lower) {
		bits |= nameFireOrIgnite
	}
	if coldOrFreezePattern.MatchString(lower) {
		bits |= nameColdOrFreeze
	}
	return bits
}

func TestModNameElementsMatchTheRegexpsOverRealMods(t *testing.T) {
	initBenchData(t)

	for _, cold := range []string{"UnseenProbeFreezeIgnite", "UnseenProbeColdOnly", "UnseenProbeNeither"} {
		if _, seen := modNameElementCache.Load(cold); seen {
			t.Fatalf("%s is already cached; the fill path is not graded", cold)
		}
		if got, want := modNameElements(cold), directModNameElements(cold); got != want {
			t.Errorf("%s on a cold cache: got %02b, want %02b", cold, got, want)
		}
		if got, want := modNameElements(cold), directModNameElements(cold); got != want {
			t.Errorf("%s from the cache: got %02b, want %02b", cold, got, want)
		}
	}

	seen := map[uint8]int{}
	for _, name := range []string{"0280-elementalist-fireball.xml", "0001-necromancer-unknown.xml"} {
		c := &Calculator{PoB: loadHeavyBuild(t, name)}
		env := c.BuildOutput(OutputModeMain)
		for modName := range env.ModsUsed {
			want := directModNameElements(modName)
			if got := modNameElements(modName); got != want {
				t.Errorf("%s: %s got %02b, want %02b", name, modName, got, want)
			}
			seen[want]++
		}
	}
	for _, bits := range []uint8{0, nameFireOrIgnite, nameColdOrFreeze, nameFireOrIgnite | nameColdOrFreeze} {
		if seen[bits] == 0 {
			t.Errorf("no real mod name carries bits %02b; the differential does not cover that case", bits)
		}
	}
}
