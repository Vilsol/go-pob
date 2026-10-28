package tree2

import (
	"reflect"
	"strings"
	"testing"
)

func strp(s string) *string { return &s }

// druidSpec is a Druid/Oracle <Spec> on tree 0_5; the expectations below are PoB-PoE2's own
// allocNodes, allocMode and hashOverrides after loading the same element headless.
func druidSpec(nodes string, weaponSets map[int]string, dex, intel string) SpecInput {
	return SpecInput{
		TreeVersion: "0_5", Nodes: strp(nodes), ClassID: strp("11"), AscendClassID: strp("1"),
		ClassInternalID: strp("11"), AscendancyInternalID: strp("Druid1"), SecondaryAscendClassID: strp("nil"),
		MasteryEffects: strp(""), WeaponSets: weaponSets,
		AttributeOverrides: []AttributeOverrideInput{{DexNodes: dex, IntNodes: intel}},
	}
}

// oracleRadiusRows stands in for the calc's mod parse of the one AllocateFromNodeRadius row.
//
//pob2:block:src/Modules/ModParser.lua:5596-5598
func oracleRadiusRows(node *SpecNode) []NodeRadius {
	var out []NodeRadius
	for _, line := range node.Sd {
		if strings.EqualFold(line, "Non-Keystone Passive Skills in Medium Radius of allocated Keystone Passive Skills can be allocated without being connected to your tree") {
			out = append(out, NodeRadius{From: "Keystone", RadiusIndex: 2, To: []string{"Notable", "Normal"}})
		}
	}
	return out
}

func loadSpec(t *testing.T, in SpecInput) *PassiveSpec {
	t.Helper()
	spec := NewPassiveSpec(dataTree(t))
	spec.AllocateFromNodeRadius = oracleRadiusRows
	if err := spec.Load(in); err != nil {
		t.Fatal(err)
	}
	return spec
}

func TestSpecPrunesNodesWithNoPathToAStart(t *testing.T) {
	spec := loadSpec(t, druidSpec("722,1143,99999", map[int]string{1: "722"}, "", "722"))
	if got := sortedKeys(spec.AllocNodes); !reflect.DeepEqual(got, []int{42761, 61525}) {
		t.Errorf("allocNodes = %v, PoB [42761 61525]", got)
	}
	if spec.CurClassName != "Druid" || spec.CurAscendClassName != "Oracle" {
		t.Errorf("class = %s/%s, PoB Druid/Oracle", spec.CurClassName, spec.CurAscendClassName)
	}
	if o := spec.HashOverrides[722]; o == nil || o.Dn != "Intelligence" {
		t.Errorf("override 722 = %+v, PoB keeps Intelligence after the prune", o)
	}
	if !reflect.DeepEqual(spec.AllocSubgraphNodes, []int{99999}) {
		t.Errorf("allocSubgraphNodes = %v, want [99999]", spec.AllocSubgraphNodes)
	}
}

func TestSpecWeaponSetNodesPathOnlyThroughTheirOwnSet(t *testing.T) {
	// 1143 (mode 0) reaches the start only through 50084, a weapon set 1 node, so PoB prunes it;
	// 64370 (weapon set 2) paths through 13855 (mode 0).
	spec := loadSpec(t, druidSpec("13855,64370,50084,1143,722,99999", map[int]string{1: "50084", 2: "64370"}, "64370", "722"))
	if got := sortedKeys(spec.AllocNodes); !reflect.DeepEqual(got, []int{13855, 42761, 50084, 61525, 64370}) {
		t.Errorf("allocNodes = %v, PoB [13855 42761 50084 61525 64370]", got)
	}
	modes := map[int]int{}
	for id, n := range spec.AllocNodes {
		if n.AllocMode != 0 {
			modes[id] = n.AllocMode
		}
	}
	if !reflect.DeepEqual(modes, map[int]int{50084: 1, 64370: 2}) {
		t.Errorf("allocModes = %v, PoB map[50084:1 64370:2]", modes)
	}
	if got := spec.AllocNodes[64370].Dn; got != "Dexterity" {
		t.Errorf("64370 = %q, PoB Dexterity", got)
	}
	// A switchable node takes the option named for the class even unallocated.
	if got := spec.Nodes[1143].Dn; got != "Life Regeneration" {
		t.Errorf("1143 for a Druid = %q, want its Druid option Life Regeneration", got)
	}
	used, asc, _, _, ws1, ws2 := spec.CountAllocNodes()
	if used != 3 || asc != 0 || ws1 != 1 || ws2 != 1 {
		t.Errorf("CountAllocNodes = used %d asc %d ws1 %d ws2 %d, want 3 0 1 1", used, asc, ws1, ws2)
	}
}

func TestSpecLoadRefusesAMissingClass(t *testing.T) {
	spec := NewPassiveSpec(dataTree(t))
	if err := spec.Load(SpecInput{Nodes: strp("1"), AscendClassID: strp("0")}); err == nil {
		t.Fatal("a <Spec> with no class loaded")
	}
}

func TestResolveGrantedPassiveNodes(t *testing.T) {
	spec := NewPassiveSpec(dataTree(t))
	for name, want := range map[string]int{"  Saqawal's Guidance ": 42070, "giant's blood": 32349} {
		got := spec.ResolveGrantedPassiveNodes(GrantedPassive{Name: name})
		if len(got) != 1 || got[0].ID != want {
			t.Errorf("%q -> %v, want node %d", name, got, want)
		}
	}
	if got := spec.ResolveGrantedPassiveNodes(GrantedPassive{Name: "no such passive"}); len(got) != 0 {
		t.Errorf("unknown name resolved to %v", got)
	}
}
