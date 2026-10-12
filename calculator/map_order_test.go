package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/storage"
)

// TestScanTieBreakIsTotal pins scan's winner against the order its candidates arrive in.
func TestScanTieBreakIsTotal(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	const line = "+1 to level of socketed active skill gems"

	tied := make([]string, 0, 2)
	for pattern, compiled := range specialModListCompiled {
		if compiled.Regex.MatchString(line) {
			tied = append(tied, pattern)
		}
	}
	testza.AssertEqual(t, 2, len(tied), "expected the known ambiguous pattern pair")

	winner := func(order []string) string {
		state := newScanState[interface{}](line)
		for _, pattern := range order {
			state.consider(pattern, specialModListCompiled[pattern])
		}
		return state.bestPattern
	}

	forward := winner([]string{tied[0], tied[1]})
	reverse := winner([]string{tied[1], tied[0]})
	testza.AssertEqual(t, forward, reverse, "scan's winner must not depend on candidate order")
}

// TestClusterJewelSkillsAreDeterministic pins the enchant -> skill-id table.
func TestClusterJewelSkillsAreDeterministic(t *testing.T) {
	const collidingEnchant = "added small passive skills grant: 2% increased effect of your curses"

	want := buildClusterJewelSkills()
	testza.AssertNotNil(t, want[collidingEnchant])

	skillID := func(table map[string][]mod.Mod) interface{} {
		mods := table[collidingEnchant]
		testza.AssertEqual(t, 1, len(mods))
		data, isJewelData := mods[0].Value().ValueList.(mod.JewelData)
		testza.AssertTrue(t, isJewelData)
		return data.Value
	}

	testza.AssertEqual(t, "affliction_curse_effect", skillID(want))

	for i := 0; i < 60; i++ {
		got := buildClusterJewelSkills()
		testza.AssertEqual(t, len(want), len(got))
		testza.AssertEqual(t, skillID(want), skillID(got))
	}
}

// TestGemIsTypeFoldsCase pins the keyword comparison against the case a mod pattern captured it in.
func TestGemIsTypeFoldsCase(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	base := poe.BaseItemTypeByIDMap["Metadata/Items/Gems/SkillGemFireball"]
	testza.AssertNotNil(t, base)
	gem := base.SkillGem()
	testza.AssertNotNil(t, gem)

	testza.AssertTrue(t, CalcGemIsType(gem, "active skill"))
	testza.AssertTrue(t, CalcGemIsType(gem, "Active Skill"))
	testza.AssertTrue(t, CalcGemIsType(gem, "active_skill"))
	testza.AssertTrue(t, CalcGemIsType(gem, "Spell"))
	testza.AssertFalse(t, CalcGemIsType(gem, "Attack"))
}
