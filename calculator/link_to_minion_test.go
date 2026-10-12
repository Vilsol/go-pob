package calculator

import (
	"os"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

//	linksApplyToMinions = env.minion and modDB:Flag(nil, "Condition:CanLinkToMinions")
//	    and modDB:Flag(nil, "Condition:LinkedToMinion") and not <minion undamageable>

// TestLinkSkillsCanTargetDamageableMinionsRowParses pins ModParser.lua:4323.
func TestLinkSkillsCanTargetDamageableMinionsRowParses(t *testing.T) {
	e := ParseMod("Link Skills can target Damageable Minions", false)
	if e.Extra != "" {
		t.Fatalf("unparsed remainder %q — the line must parse WHOLE", e.Extra)
	}
	if len(e.ModList) != 1 {
		t.Fatalf("got %d mods, want 1", len(e.ModList))
	}
	m := e.ModList[0]
	if m.Name() != "Condition:CanLinkToMinions" || m.Type() != mod.TypeFlag {
		t.Fatalf("got %s %s, want Condition:CanLinkToMinions FLAG", m.Name(), m.Type())
	}
	if len(m.Tags()) != 1 {
		t.Fatalf("got %d tags, want 1 (Condition HaveDamageableMinion): %v", len(m.Tags()), m.Tags())
	}
	ct, ok := m.Tags()[0].(*mod.ConditionTag)
	if !ok || len(ct.VarList) != 1 || ct.VarList[0] != "HaveDamageableMinion" {
		t.Errorf("tag = %#v, want Condition HaveDamageableMinion", m.Tags()[0])
	}
}

// TestLinkToMinionRowDoesNotSwallowItsNeighbour is the control.
func TestLinkToMinionRowDoesNotSwallowItsNeighbour(t *testing.T) {
	e := ParseMod("Your Linked Minions take 70% less Damage", false)
	if e.Extra != "" || len(e.ModList) != 1 {
		t.Fatalf("remainder %q, %d mods", e.Extra, len(e.ModList))
	}
	if e.ModList[0].Name() != "MinionModifier" {
		t.Errorf("neighbour parsed to %s, want MinionModifier", e.ModList[0].Name())
	}
}

// TestLinkedToMinionConfigIsDispatched pins ConfigOptions.lua:513-514.
func TestLinkedToMinionConfigIsDispatched(t *testing.T) {
	if !slices.Contains(configOrder, "linkedToMinion") {
		t.Fatal("linkedToMinion is not in configOrder, so its handler is never dispatched")
	}
	handler, ok := configurations["linkedToMinion"]
	if !ok {
		t.Fatal("no configurations[\"linkedToMinion\"] handler")
	}

	checked := moddb.NewModList()
	handler(true, checked, moddb.NewModList())
	if n := len(checked.Mods()); n != 1 {
		t.Fatalf("checked: got %d mods, want 1", n)
	}
	m := checked.Mods()[0]
	if m.Name() != "Condition:LinkedToMinion" || m.Type() != mod.TypeFlag || m.GetSource() != "Config" {
		t.Errorf("got %s %s source %q, want Condition:LinkedToMinion FLAG source Config", m.Name(), m.Type(), m.GetSource())
	}
	if len(m.Tags()) != 0 {
		t.Errorf("got %d tags, want 0 — ConfigOptions.lua:514 adds the flag untagged", len(m.Tags()))
	}

	unchecked := moddb.NewModList()
	handler(false, unchecked, moddb.NewModList())
	if n := len(unchecked.Mods()); n != 0 {
		t.Errorf("unchecked: got %d mods, want 0", n)
	}
}

// TestLinkArmMarksTheLinkedMinion is the end-to-end.
func TestLinkArmMarksTheLinkedMinion(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const subject = "1007-luminary-animate-guardian-of-smiting-4.xml"
	env := corpusEnvForLoadedData(t, subject)
	if env.Minion == nil || env.Minion.ModDB == nil {
		t.Fatalf("REFUSING: %s builds no minion actor", subject)
	}
	if !env.ModDB.Flag(nil, "Condition:LinkedToMinion") {
		t.Fatalf("PREMISE BROKEN: %s does not carry Condition:LinkedToMinion", subject)
	}
	if !env.ModDB.Flag(nil, "Condition:CanLinkToMinions") {
		t.Fatalf("PREMISE BROKEN: %s does not carry Condition:CanLinkToMinions", subject)
	}
	for _, cond := range []string{"AffectedByLink", "AffectedByFlameLink"} {
		if !env.Minion.ModDB.Conditions[cond] {
			t.Errorf("%s: minion condition %s not set", subject, cond)
		}
	}

	const control = "0008-necromancer-vaal-summon-skeletons.xml"
	cenv := corpusEnvForLoadedData(t, control)
	if cenv.Minion == nil || cenv.Minion.ModDB == nil {
		t.Fatalf("REFUSING: control %s builds no minion actor, so it controls nothing", control)
	}
	if cenv.Minion.ModDB.Conditions["AffectedByLink"] {
		t.Errorf("control %s (a minion build with no link skill) is marked AffectedByLink — "+
			"the arm is applying unconditionally", control)
	}
}
