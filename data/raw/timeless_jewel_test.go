package raw

import (
	"math"
	"testing"

	tjcalc "github.com/Vilsol/timeless-jewels/calculator"
	tjdata "github.com/Vilsol/timeless-jewels/data"
)

func mustU32(t *testing.T, field string, v int64) uint32 {
	t.Helper()

	if v < 0 || v > math.MaxUint32 {
		t.Fatalf("REFUSING: %s = %d does not fit uint32", field, v)
	}

	return uint32(v) //nolint:gosec
}

func mustInit(t *testing.T) {
	t.Helper()

	if err := InitializeAll(LatestVersion, nil); err != nil {
		t.Fatalf("REFUSING: game data %s did not initialise: %v", LatestVersion, err)
	}
	if err := EnsureTimelessSources(); err != nil {
		t.Fatalf("REFUSING: timeless sources did not load: %v", err)
	}
	if !tjdata.Initialized() {
		t.Fatal("REFUSING: EnsureTimelessSources succeeded but the library reports itself uninitialised — " +
			"initTimelessJewelCalculator is meant to run as part of the load")
	}
}

// TestGloriousVanityXibaquaReplacesStrengthNodes is the acceptance criterion for injecting our tables instead of linking the library's embedded copies.
func TestGloriousVanityXibaquaReplacesStrengthNodes(t *testing.T) {
	mustInit(t)

	const seed = 7094

	for _, graphID := range []int64{31875, 34171, 46910, 23027} {
		skill := PassiveSkillByGraphID(graphID)
		if skill == nil {
			t.Errorf("node %d: no PassiveSkill row", graphID)
			continue
		}

		result := tjcalc.Calculate(mustU32(t, "PassiveSkills._key", skill.Key), seed,
			tjdata.GloriousVanity, tjdata.Xibaqua)
		if result.AlternatePassiveSkill == nil {
			t.Errorf("node %d (%s, skill index %d): NOT replaced — the library answered from an empty or stale index",
				graphID, skill.Name, skill.Key)
			continue
		}

		t.Logf("node %5d %-18s -> %-34s rolls %v",
			graphID, skill.Name, result.AlternatePassiveSkill.ID, result.StatRolls)
	}
}

// TestConquerorJoinLinesUp pins the join our tree code depends on: PoB names a conqueror type and the library wants a JewelType enum value.
func TestConquerorJoinLinesUp(t *testing.T) {
	mustInit(t)

	for conqueror, want := range map[string]tjdata.JewelType{
		"vaal":     tjdata.GloriousVanity,
		"karui":    tjdata.LethalPride,
		"maraketh": tjdata.BrutalRestraint,
		"templar":  tjdata.MilitantFaith,
		"eternal":  tjdata.ElegantHubris,
	} {
		got := JewelTypeForConqueror(conqueror)
		if tjdata.JewelType(got) != want {
			t.Errorf("conqueror %q resolves to jewel type %d, want %d (%s)", conqueror, got, want, want)
		}
		if v := tjdata.GetAlternateTreeVersionIndex(mustU32(t, "JewelTypeForConqueror", int64(got))); v == nil {
			t.Errorf("conqueror %q: jewel type %d is absent from the injected tree versions", conqueror, got)
		}
	}
}

// TestInjectedTablesMatchOurRowCounts is the control for the two tests above.
func TestInjectedTablesMatchOurRowCounts(t *testing.T) {
	mustInit(t)

	for _, c := range []struct {
		name string
		ours int
		lib  int
	}{
		{"PassiveSkills", len(PassiveSkills), len(tjdata.PassiveSkills)},
		{"AlternatePassiveSkills", len(AlternatePassiveSkills), len(tjdata.AlternatePassiveSkills)},
		{"AlternatePassiveAdditions", len(AlternatePassiveAdditions), len(tjdata.AlternatePassiveAdditions)},
		{"AlternateTreeVersions", len(AlternateTreeVersions), len(tjdata.AlternateTreeVersions)},
	} {
		if c.ours == 0 {
			t.Errorf("REFUSING: %s is empty on OUR side — the comparison below would be vacuous", c.name)
			continue
		}
		if c.ours != c.lib {
			t.Errorf("%s: injected %d rows, we hold %d", c.name, c.lib, c.ours)
		}
	}
}

// TestInjectedTablesCarryPerTypeColumns pins the columns the library reads beside Var5/6.
func TestInjectedTablesCarryPerTypeColumns(t *testing.T) {
	mustInit(t)

	for _, c := range []struct {
		key                                    uint32
		minAttr, maxAttr, minNotable, maxNotab uint32
	}{
		{2, 1, 1, 1, 1},
		{6, 1, 1, 0, 0},
		{11, 0, 0, 1, 1},
	} {
		v := tjdata.GetAlternateTreeVersionIndex(c.key)
		if v == nil {
			t.Errorf("tree version %d absent", c.key)
			continue
		}
		got := [4]uint32{v.MinimumAdditionsSmallAttribute, v.MaximumAdditionsSmallAttribute, v.MinimumAdditionsNotable, v.MaximumAdditionsNotable}
		want := [4]uint32{c.minAttr, c.maxAttr, c.minNotable, c.maxNotab}
		if got != want {
			t.Errorf("tree version %d (%s): Var3/4/7/8 = %v, want %v", c.key, v.ID, got, want)
		}
	}

	ascendancy, descendancy := 0, 0
	for _, p := range tjdata.PassiveSkills {
		if p.AscendancyKey != nil {
			ascendancy++
		}
		if p.DescendancyKey != nil {
			descendancy++
		}
	}
	if ascendancy == 0 || descendancy == 0 {
		t.Errorf("injected PassiveSkills carry %d AscendancyKey and %d DescendancyKey rows, want both nonzero", ascendancy, descendancy)
	}
}
