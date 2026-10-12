package pob_test

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/pob"
)

const twoConfigSets = `<PathOfBuilding>
<Build level="90" className="Templar" ascendClassName="Hierophant" mainSocketGroup="1"/>
<Tree activeSpec="1"><Spec classId="5" ascendClassId="1" treeVersion="3_25" nodes=""/></Tree>
<Config activeConfigSet="3">
<ConfigSet title="Final" id="3">
<Input name="enemyIsBoss" string="Pinnacle"/>
<CustomModifierBlock title="Default" enabled="true">+10 to Strength</CustomModifierBlock>
</ConfigSet>
<ConfigSet title="Leveling" id="2">
<Input name="conditionLowLife" boolean="true"/>
<Placeholder name="enemyLevel" number="70"/>
<CustomModifierBlock title="Default" enabled="true">+20 to Dexterity</CustomModifierBlock>
</ConfigSet>
</Config>
</PathOfBuilding>`

func TestSetActiveConfigSetSwapsTheSetsInputs(t *testing.T) {
	b, err := builds.ParseBuildStr(twoConfigSets)
	if err != nil {
		t.Fatal(err)
	}
	if got := b.GetStringOption("enemyIsBoss"); got != "Pinnacle" {
		t.Fatalf("set 3 should be loaded: enemyIsBoss = %q", got)
	}

	b.SetActiveConfigSet(2)
	if b.Config.ActiveConfigSet != 2 {
		t.Fatalf("ActiveConfigSet = %d, want 2", b.Config.ActiveConfigSet)
	}
	if !b.GetBooleanOption("conditionLowLife") || b.GetStringOption("enemyIsBoss") != "" {
		t.Fatalf("set 2's inputs are not the live ones: %+v", b.Config.Inputs)
	}
	if got := b.GetNumberPlaceholder("enemyLevel"); got != 70 {
		t.Fatalf("set 2's placeholder enemyLevel = %v, want 70", got)
	}
	if blocks := b.GetCustomModifierBlocks(); len(blocks) != 1 || blocks[0].Text != "+20 to Dexterity" {
		t.Fatalf("set 2's custom modifiers are not the live ones: %+v", blocks)
	}

	b.SetConfigOption(inputString("enemyIsBoss", "Uber"))
	b.SetActiveConfigSet(3)
	if got := b.GetStringOption("enemyIsBoss"); got != "Pinnacle" || b.GetBooleanOption("conditionLowLife") {
		t.Fatalf("set 3 did not come back intact: %+v", b.Config.Inputs)
	}
	b.SetActiveConfigSet(2)
	if got := b.GetStringOption("enemyIsBoss"); got != "Uber" {
		t.Fatalf("the edit made on set 2 was lost: enemyIsBoss = %q", got)
	}
}

// TestSetActiveConfigSetUnknownIDFallsBackToTheFirstSet: ConfigTab.lua:1289-1291: an unknown id falls back to the first set in order.
func TestSetActiveConfigSetUnknownIDFallsBackToTheFirstSet(t *testing.T) {
	b, err := builds.ParseBuildStr(twoConfigSets)
	if err != nil {
		t.Fatal(err)
	}
	b.SetActiveConfigSet(2)
	b.SetActiveConfigSet(9)
	if b.Config.ActiveConfigSet != 3 || b.GetStringOption("enemyIsBoss") != "Pinnacle" {
		t.Fatalf("ActiveConfigSet = %d with inputs %+v, want set 3", b.Config.ActiveConfigSet, b.Config.Inputs)
	}
}

// TestSetActiveConfigSetSavesEverySetUnderItsID: Saving after a switch writes each set's own inputs under its own id, and the active id.
func TestSetActiveConfigSetSavesEverySetUnderItsID(t *testing.T) {
	b, err := builds.ParseBuildStr(twoConfigSets)
	if err != nil {
		t.Fatal(err)
	}
	b.SetActiveConfigSet(2)
	out, err := builds.SerializeBuild(b)
	if err != nil {
		t.Fatal(err)
	}
	xml := string(out)
	if !strings.Contains(xml, `activeConfigSet="2"`) {
		t.Fatalf("the active id is not saved: %s", xml)
	}
	set3 := xml[strings.Index(xml, `id="3"`):strings.Index(xml, `id="2"`)]
	if !strings.Contains(set3, "enemyIsBoss") || strings.Contains(set3, "conditionLowLife") {
		t.Fatalf("set 3 saved with the wrong inputs: %s", set3)
	}
	set2 := xml[strings.Index(xml, `id="2"`):]
	if !strings.Contains(set2, "conditionLowLife") || strings.Contains(set2, "enemyIsBoss") {
		t.Fatalf("set 2 saved with the wrong inputs: %s", set2)
	}
}

func inputString(name, value string) pob.Input {
	return pob.Input{Name: name, String: &value}
}
