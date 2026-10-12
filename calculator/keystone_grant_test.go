package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func keystoneNamesFromMods(mods []mod.Mod) []string {
	out := make([]string, 0)
	for _, m := range mods {
		if m.Name() != "Keystone" {
			continue
		}
		if s, ok := m.Value().List().(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// TestBareKeystoneNameParsesToKeystoneMod: PoB ModParser.lua:3476-3478 -- after the specialModList literal.
//
//	for _, name in pairs(data.keystones) do
//	end
func TestBareKeystoneNameParsesToKeystoneMod(t *testing.T) {
	for _, name := range []string{"Eldritch Battery", "Chaos Inoculation", "Zealot's Oath", "The Agnostic", "Vaal Pact"} {
		entry := ParseMod(name, false)
		testza.AssertNotNil(t, entry.ModList, name+" must parse")
		testza.AssertEqual(t, "", entry.Extra, name+" must be fully consumed")
		testza.AssertEqual(t, []string{name}, keystoneNamesFromMods(entry.ModList),
			name+" must yield exactly one Keystone LIST mod naming itself")
	}
}

// TestKeystoneNameIsAnchored: The keystone lines are exact-match only.
func TestKeystoneNameIsAnchored(t *testing.T) {
	entry := ParseMod("you have vaal pact if you've dealt a critical strike recently", false)
	testza.AssertNotNil(t, entry.ModList)
	testza.AssertEqual(t, []string{"Vaal Pact"}, keystoneNamesFromMods(entry.ModList))
	testza.AssertEqual(t, 1, len(entry.ModList))
	testza.AssertEqual(t, 1, len(entry.ModList[0].Tags()), "the conditional form must keep its Condition tag")

	entry = ParseMod("10% increased Blood Magic Duration", false)
	testza.AssertEqual(t, 0, len(keystoneNamesFromMods(entry.ModList)))
}

// TestAllDataKeystonesAreRegistered: Every name in data.keystones must be registered, and nothing else may be.
func TestAllDataKeystonesAreRegistered(t *testing.T) {
	testza.AssertEqual(t, 58, len(data.Keystones), "PoB Modules/Data.lua:352 lists 58 keystones")
	for _, name := range data.Keystones {
		entry := ParseMod(name, false)
		testza.AssertEqual(t, []string{name}, keystoneNamesFromMods(entry.ModList), name)
	}
}

// TestConfigOptionsGrantKeystones: PoB ConfigOptions.lua:478 / :523 / :1145 / :1150 / :1153 / :1158.
func TestConfigOptionsGrantKeystones(t *testing.T) {
	cases := []struct {
		option   string
		val      interface{}
		keystone string
		flag     string
	}{
		{"playerHasElementalEquilibrium", true, "Elemental Equilibrium", ""},
		{"playerHasPointBlank", true, "Point Blank", ""},
		{"conditionHaveArborix", true, "Iron Reflexes", "Condition:HaveIronReflexes"},
		{"conditionHaveAugyre", "EleOverload", "Elemental Overload", "Condition:HaveElementalOverload"},
		{"conditionHaveAugyre", "ResTechnique", "Resolute Technique", "Condition:HaveResoluteTechnique"},
		{"conditionHaveVulconus", true, "Avatar of Fire", "Condition:HaveAvatarOfFire"},
	}

	for _, c := range cases {
		apply, ok := configurations[c.option]
		testza.AssertTrue(t, ok, c.option+" must be a live config handler")
		if !ok {
			continue
		}

		modList := moddb.NewModList()
		enemyModList := moddb.NewModList()
		apply(c.val, modList, enemyModList)

		testza.AssertEqual(t, []interface{}{c.keystone}, modList.List(nil, "Keystone"), c.option)
		if c.flag != "" {
			found := false
			for _, m := range modList.Mods() {
				if m.Name() == c.flag {
					found = true
					testza.AssertEqual(t, 1, len(m.Tags()), c.flag+" keeps its Combat condition tag")
				}
			}
			testza.AssertTrue(t, found, c.option+" must also set "+c.flag)
		}
		testza.AssertEqual(t, 0, len(enemyModList.Mods()), c.option+" must not touch the enemy mod list")
	}

	modList := moddb.NewModList()
	configurations["conditionHaveAugyre"](nil, modList, moddb.NewModList())
	testza.AssertEqual(t, 0, len(modList.Mods()))
}

// TestKeystoneMergedOnlyOnce: PoB CalcPerform.lua:369-378 guards with env.keystonesAdded.
func TestKeystoneMergedOnlyOnce(t *testing.T) {
	env := &Environment{
		ModDB:          moddb.NewModDB(),
		KeystonesAdded: make(map[string]interface{}),
		Spec:           &PassiveSpec{TreeVersion: data.LatestTreeVersion},
	}
	env.ModDB.AddMod(mod.NewList("Keystone", "Iron Reflexes").Source("Tree10661"))
	env.ModDB.AddMod(mod.NewList("Keystone", "Iron Reflexes").Source("Item"))

	countMods := func() int {
		total := 0
		for _, mods := range env.ModDB.Mods {
			total += len(mods)
		}
		return total
	}

	before := countMods()
	mergeKeystones(env, env.ModDB)
	added := countMods() - before

	expected := len(keystoneModList(env, data.LatestTreeVersion, "Iron Reflexes").Mods())
	testza.AssertEqual(t, expected, added, "a doubly-granted keystone must merge its stat lines once")
}
