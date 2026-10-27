package calculator2

import (
	"os"
	"sort"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's for the same file, loaded headless: the main skill's
// baseSkillModList, skillData and mainOutput. A level 15, 10% quality corrupted Falling Thunder in a
// Quarterstaff with Close Combat II and Pinpoint Critical, under "+2 to Level of all Melee Skills" and
// "+5% to Quality of all Skills".
func TestBuildActiveSkillModListFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/skill_modlist.xml")
	testza.AssertNoError(t, err)
	build, err := pob2.ParseBuild(raw)
	testza.AssertNoError(t, err)
	tree := loadTestTree(t)
	env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
	testza.AssertNoError(t, err)

	mainSkill := env.Player.MainSkill
	testza.AssertEqual(t, "Falling Thunder", mainSkill.ActiveEffect.GrantedEffect.Name)
	got := make([]string, 0, len(mainSkill.BaseSkillModList.Mods()))
	for _, m := range mainSkill.BaseSkillModList.Mods() {
		got = append(got, ourModKey(m))
	}
	sort.Strings(got)
	want := []string{
		"CritChance|MORE|60|||",
		"CritMultiplier|MORE|-30|||",
		"Damage|MORE|30|||DistanceRamp",
		"GemCorruptionLevel|BASE|1|||",
		"GemGlobalLevel|BASE|2|||",
		"GemGlobalQuality|BASE|5|||",
		"GemLevel|BASE|15|||",
		"GemQuality|BASE|10|||",
		"Multiplier:SupportCount|BASE|1|||",
		"Multiplier:SupportCount|BASE|1|||",
		"SkillData|LIST|radius=40|||",
		"SkillPhysicalDamageConvertToLightning|BASE|60|||",
		"SupportManaMultiplier|MORE|20|||",
	}
	sort.Strings(want)
	testza.AssertEqual(t, want, got)
	testza.AssertEqual(t, 40.0, mainSkill.SkillData["radius"])
	testza.AssertEqual(t, -40.0, mainSkill.SkillData["attackSpeedMultiplier"])
	testza.AssertEqual(t, 18.0, env.Player.Output["GemLevel"])
	testza.AssertEqual(t, 15.0, env.Player.Output["GemQuality"])
}
