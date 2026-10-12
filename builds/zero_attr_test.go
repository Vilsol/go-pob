package builds

import (
	"strings"
	"testing"
)

// TestUnsetAttributesAreOmittedNotZeroed pins the ADDED-attribute class.
func TestUnsetAttributesAreOmittedNotZeroed(t *testing.T) {
	const in = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>` +
		`<Skills activeSkillSet="1"><SkillSet id="1">` +
		`<Skill enabled="true" label="Zap"><Gem nameSpec="Spark" level="20" quality="0" enabled="true"/></Skill>` +
		`</SkillSet></Skills>` +
		`<Items activeItemSet="1"><ItemSet id="1"><Slot name="Weapon 1" itemId="0"/></ItemSet></Items>` +
		`<Config activeConfigSet="1"><ConfigSet id="1"/></Config>` +
		`</PathOfBuilding>`

	build, err := ParseBuildStr(in)
	if err != nil {
		t.Fatalf("ParseBuild: %v", err)
	}
	out, err := SerializeBuild(build)
	if err != nil {
		t.Fatalf("SerializeBuild: %v", err)
	}
	saved := string(out)

	for _, tc := range []struct{ spelling, why string }{
		{`active="false"`, "ItemsTab.lua:1382 writes `active = ... and \"true\"`, so unset is ABSENT"},
		{`mainActiveSkill="0"`, "SkillsTab.lua:426 `tonumber(x) or 1` — absent is 1, \"0\" is 0"},
		{`mainActiveSkillCalcs="0"`, "SkillsTab.lua:427, same `or 1`"},
		{`skillPart="0"`, "SkillsTab.lua:475 `tonumber(x)` — absent is nil, \"0\" is a truthy 0"},
		{`skillPartCalcs="0"`, "SkillsTab.lua:476"},
		{`skillStageCount="0"`, "SkillsTab.lua:477"},
		{`skillStageCountCalcs="0"`, "SkillsTab.lua:478"},
		{`skillMinionSkill="0"`, "CalcActiveSkill.lua:727-733 treats absent as 1"},
		{`skillMinionItemSet="0"`, "unset is nil upstream, not 0"},
		{`skillMinion=""`, "SkillsTab.lua:481 takes the attribute raw; \"\" is TRUTHY in Lua"},
		{`qualityId=""`, "a truthy \"\" defeats every `qualityId or \"Default\"`"},
		{`gemId=""`, "SkillsTab.lua:437 `if child.attrib.gemId then` is TRUE for \"\""},
		{`skillId=""`, "same truthy-empty-string read"},
		{`slot=""`, "SkillsTab.lua:424; a truthy \"\" makes an unslotted group look slotted"},
		{`matchGemLevelToCharacterLevel="false"`, "absent is false upstream"},
		{`showAltQualityGems="false"`, "absent is false upstream"},
	} {
		if strings.Contains(saved, tc.spelling) {
			t.Errorf("saved XML writes %s for an attribute the input omitted — %s", tc.spelling, tc.why)
		}
	}

	for _, want := range []string{`label="Zap"`, `nameSpec="Spark"`, `level="20"`, `name="Weapon 1"`} {
		if !strings.Contains(saved, want) {
			t.Errorf("saved XML lost %s — the value was present in the input", want)
		}
	}

	if !strings.Contains(saved, `enableGlobal1="false"`) {
		t.Error(`saved XML omits enableGlobal1="false"; SkillsTab.lua:472 reads an absent one as TRUE`)
	}
}

// TestExplicitZeroAttributesSurvive is the CONTROL for the test above.
func TestExplicitZeroAttributesSurvive(t *testing.T) {
	const in = `<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding>` +
		`<Build level="90" className="Witch" ascendClassName="Occultist"/>` +
		`<Tree activeSpec="1"><Spec treeVersion="3_29" nodes=""/></Tree>` +
		`<Skills activeSkillSet="1"><SkillSet id="1">` +
		`<Skill enabled="true" label=""><Gem nameSpec="Spark" level="20" quality="0" enabled="true"/></Skill>` +
		`</SkillSet></Skills>` +
		`<Config activeConfigSet="1"><ConfigSet id="1" title=""/></Config>` +
		`</PathOfBuilding>`

	build, err := ParseBuildStr(in)
	if err != nil {
		t.Fatalf("ParseBuild: %v", err)
	}
	out, err := SerializeBuild(build)
	if err != nil {
		t.Fatalf("SerializeBuild: %v", err)
	}
	saved := string(out)
	for _, want := range []string{`label=""`, `title=""`} {
		if !strings.Contains(saved, want) {
			t.Errorf("saved XML dropped an explicit %s that the input carried", want)
		}
	}

	build2, err := ParseBuildStr(strings.Replace(in, `<ConfigSet id="1" title=""/>`, `<ConfigSet id="1"/>`, 1))
	if err != nil {
		t.Fatalf("ParseBuild: %v", err)
	}
	out2, err := SerializeBuild(build2)
	if err != nil {
		t.Fatalf("SerializeBuild: %v", err)
	}
	if strings.Contains(string(out2), `title=""`) {
		t.Error(`saved XML added title="" to a <ConfigSet> that had no title; ConfigTab.lua:854 reads absent as "Default"`)
	}
}
