package exposition

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/tree2"
)

func openFixture(t *testing.T, name string) (*Calculator, *Outputs) {
	t.Helper()
	xml, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	return calc, out
}

func click(t *testing.T, calc *Calculator, id int, hotkey int) tree2.ClickResult {
	t.Helper()
	res, err := calc.ClickNode(id, false, hotkey, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	return tree2.ClickResult(res)
}

// PoB-PoE2's sidebar for the fixture with node 44871 (+10 to maximum Energy Shield) allocated, from
// oracle/poe2_sidebar.lua: Energy Shield 10, Effective Hit Pool 546 (542 without it).
func TestClickNodeAllocatesAndRecalculates(t *testing.T) {
	calc, before := openFixture(t, "sorceress_spark.xml")
	if before.Output["EnergyShield"] != 0 {
		t.Fatalf("EnergyShield %v before the click", before.Output["EnergyShield"])
	}
	if got := calc.HoverNode(44871, 0); !slices.Equal(got.Path, []int{44871}) {
		t.Fatalf("hover path %v, want [44871]", got.Path)
	}
	if res := click(t, calc, 44871, 0); res != tree2.ClickChanged {
		t.Fatalf("click result %d", res)
	}
	after, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "sorceress_spark.xml +44871", "EnergyShield", after.Output["EnergyShield"], 10)
	assertClose(t, "sorceress_spark.xml +44871", "TotalEHP", float64(int(after.Output["TotalEHP"]+0.5)), 546)
	xml, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xml, `nodes="44871,54447"`) {
		t.Fatalf("the saved spec does not carry the allocation:\n%s", xml)
	}
	st, err := calc.TreeState()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.Alloc, []int{44871, 54447}) || st.Used != 1 {
		t.Fatalf("tree state alloc %v used %d", st.Alloc, st.Used)
	}

	if dep := calc.HoverNode(44871, 0).Depends; !slices.Equal(dep, []int{44871}) {
		t.Fatalf("depends %v, want [44871]", dep)
	}
	if res := click(t, calc, 44871, 0); res != tree2.ClickChanged {
		t.Fatalf("second click result %d", res)
	}
	again, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "sorceress_spark.xml", "EnergyShield", again.Output["EnergyShield"], 0)
}

// An attribute node asks for its attribute (TreeTab:ModifyAttributePopup); the choice is saved as an override.
func TestClickAttributeNodeAsksThenAllocates(t *testing.T) {
	calc, _ := openFixture(t, "sorceress_spark.xml")
	if res := click(t, calc, 56216, 0); res != tree2.ClickChooseAttribute {
		t.Fatalf("click result %d, want ClickChooseAttribute", res)
	}
	if err := calc.AllocateAttributeNode(56216, 3, 0); err != nil {
		t.Fatal(err)
	}
	xml, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xml, `nodes="44871,54447,56216"`) || !strings.Contains(xml, `intNodes="56216"`) {
		t.Fatalf("the saved spec lacks the attribute allocation:\n%s", xml)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	st, _ := calc.TreeState()
	if i := slices.IndexFunc(st.Text, func(n TreeNodeText) bool { return n.ID == 56216 }); i < 0 || st.Text[i].Name != "Intelligence" {
		t.Fatalf("node 56216 text %+v", st.Text)
	}
}

// A node of another ascendancy of the same class switches to it (PassiveTreeView.lua:461-475).
func TestClickOtherAscendancySwitches(t *testing.T) {
	calc, _ := openFixture(t, "sorceress_spark.xml")
	if res := click(t, calc, 64789, 0); res != tree2.ClickChanged {
		t.Fatalf("click result %d", res)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if out.AscendClassName != "Stormweaver" {
		t.Fatalf("ascendancy %q, want Stormweaver", out.AscendClassName)
	}
	st, _ := calc.TreeState()
	if !slices.Contains(st.Alloc, 64789) || st.AscUsed != 1 {
		t.Fatalf("alloc %v ascUsed %d", st.Alloc, st.AscUsed)
	}
}

func TestPassiveTreeJSON(t *testing.T) {
	raw, err := PassiveTreeJSON()
	if err != nil {
		t.Fatal(err)
	}
	var tj treeJSON
	if err := json.Unmarshal([]byte(raw), &tj); err != nil {
		t.Fatal(err)
	}
	if len(tj.OrbitRadii) != 10 || len(tj.Nodes) < 4000 || len(tj.Groups) < 1000 {
		t.Fatalf("%d orbit radii, %d nodes, %d groups", len(tj.OrbitRadii), len(tj.Nodes), len(tj.Groups))
	}
	i := slices.IndexFunc(tj.Nodes, func(n treeNodeJSON) bool { return n.ID == 44871 })
	if i < 0 || tj.Nodes[i].Name != "Energy Shield" || tj.Nodes[i].Type != "Normal" {
		t.Fatalf("node 44871 %+v", tj.Nodes[i])
	}
}

// PoB-PoE2's TreeData/0_5/tree.lua: node 10636's icon, connection art and unlock constraint, node 10100's attribute
// options, and the nodeOverlay entry for Normal nodes
func TestPassiveTreeJSONCarriesTheArtNames(t *testing.T) {
	raw, err := PassiveTreeJSON()
	if err != nil {
		t.Fatal(err)
	}
	var tj treeJSON
	if err := json.Unmarshal([]byte(raw), &tj); err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(tj.Nodes, func(n treeNodeJSON) bool { return n.ID == 10636 })
	if i < 0 || !slices.Equal(tj.Nodes[i].UnlockConstraint, []int{5571}) {
		t.Fatalf("node 10636 unlock constraint %+v", tj.Nodes[i].UnlockConstraint)
	}
	var art struct {
		Sprites     map[string]map[string]json.RawMessage `json:"sprites"`
		NodeOverlay map[string]map[string]string          `json:"nodeOverlay"`
		Nodes       map[string]treeArtNode                `json:"nodes"`
	}
	if err := json.Unmarshal(tj.Art, &art); err != nil {
		t.Fatal(err)
	}
	if n := art.Nodes["10636"]; n.Icon != "Art/2DArt/SkillIcons/passives/dmgreduction.dds" || n.ConnectionArt != "CharacterPlanned" {
		t.Fatalf("node 10636 art %+v", n)
	}
	want := []treeArtOption{
		{Name: "Strength", Icon: "Art/2DArt/SkillIcons/passives/plusstrength.dds"},
		{Name: "Dexterity", Icon: "Art/2DArt/SkillIcons/passives/plusdexterity.dds"},
		{Name: "Intelligence", Icon: "Art/2DArt/SkillIcons/passives/plusintelligence.dds"},
	}
	if got := art.Nodes["10100"].Options; !slices.Equal(got, want) {
		t.Fatalf("node 10100 options %+v", got)
	}
	if got := art.NodeOverlay["Normal"]; got["alloc"] != "PSSkillFrameActive" || got["unalloc"] != "PSSkillFrame" {
		t.Fatalf("Normal overlay %v", got)
	}
	if _, ok := art.Sprites["frame"]; !ok {
		t.Fatalf("no frame sheet among %d sprite groups", len(art.Sprites))
	}
}

// PoB-PoE2's TreeData/0_5/tree.json classes, by integerId
func TestTreeClassesAreTheTreesClasses(t *testing.T) {
	got, err := TreeClasses()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Witch", "Ranger", "Warrior", "Sorceress", "Huntress", "Mercenary", "Monk", "Druid"}
	if !slices.Equal(got, want) {
		t.Errorf("classes %v, want %v", got, want)
	}
}

// savedSpec is the active <Spec>'s nodes and weapon sets as the build would save them.
func savedSpec(calc *Calculator) string {
	s := calc.build.Tree.Specs[calc.build.Tree.ActiveSpec-1]
	out := "nodes=" + *s.Nodes
	if s.WeaponSet1 != nil {
		out += " WeaponSet1=" + s.WeaponSet1.Nodes
	}
	if s.WeaponSet2 != nil {
		out += " WeaponSet2=" + s.WeaponSet2.Nodes
	}
	return out
}

// PoB-PoE2 on sorceress_spark.xml: AllocNode under spec.allocMode 1, 2 and 0, each followed by AddUndoState,
// then Undo, Undo, Redo and three more Undos, with PassiveSpec:Save and the calculated Energy Shield after
// each step (PoB-PoE2's Lua under the deterministic LuaJIT).
func TestWeaponSetAllocationUndoRedo(t *testing.T) {
	calc, _ := openFixture(t, "sorceress_spark.xml")
	step := func(name string, want string, es float64) {
		t.Helper()
		if got := savedSpec(calc); got != want {
			t.Fatalf("%s: saved %q, PoB %q", name, got, want)
		}
		out, err := calc.Calculate()
		if err != nil {
			t.Fatal(err)
		}
		assertClose(t, name, "EnergyShield", out.Output["EnergyShield"], es)
	}
	alloc := func(id, allocMode int) {
		t.Helper()
		if res, err := calc.ClickNode(id, false, 0, false, allocMode); err != nil || tree2.ClickResult(res) != tree2.ClickChanged {
			t.Fatalf("click %d: %d %v", id, res, err)
		}
	}
	undo := func(f func() (bool, error)) {
		t.Helper()
		if ok, err := f(); err != nil || !ok {
			t.Fatalf("undo/redo: %v %v", ok, err)
		}
	}
	if got := calc.HoverNode(18845, 2).Path; !slices.Equal(got, []int{18845, 4739}) {
		t.Fatalf("hover path %v, PoB [18845 4739]", got)
	}
	alloc(44871, 1)
	step("ws1 +44871", "nodes=44871,54447 WeaponSet1=44871", 10)
	alloc(18845, 2)
	step("ws2 +18845", "nodes=4739,18845,44871,54447 WeaponSet1=44871 WeaponSet2=4739,18845", 10)
	alloc(6686, 0)
	step("ws0 +6686", "nodes=4739,6686,18845,44871,54447,55807 WeaponSet1=44871", 10)
	undo(calc.Undo)
	step("undo1", "nodes=4739,18845,44871,54447 WeaponSet1=44871 WeaponSet2=4739,18845", 10)
	undo(calc.Undo)
	step("undo2", "nodes=44871,54447 WeaponSet1=44871", 10)
	undo(calc.Redo)
	step("redo1", "nodes=4739,18845,44871,54447 WeaponSet1=44871 WeaponSet2=4739,18845", 10)
	undo(calc.Undo)
	undo(calc.Undo)
	if ok, _ := calc.Undo(); ok {
		t.Fatal("an undo past the loaded state restored something")
	}
	step("undo x3", "nodes=54447", 0)
}

// A hand-made save in PassiveSpec:Save's format, with nodes on both weapon sets; PoB-PoE2 loads it with
// allocMode 2, 2, 1, 0 on 4739, 18845, 44871, 54447, counts 3 used (1 on set 1, 2 on set 2), Energy Shield 10,
// and saves it back unchanged.
func TestWeaponSetSaveRoundTrip(t *testing.T) {
	calc, out := openFixture(t, "sorceress_spark_weapon_sets.xml")
	assertClose(t, "sorceress_spark_weapon_sets.xml", "EnergyShield", out.Output["EnergyShield"], 10)
	st, err := calc.TreeState()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.AllocMode, []int{4739, 2, 18845, 2, 44871, 1}) || st.Used != 3 {
		t.Fatalf("alloc modes %v used %d", st.AllocMode, st.Used)
	}
	if _, w1, w2 := countSets(calc); w1 != 1 || w2 != 2 {
		t.Fatalf("weapon set counts %d %d", w1, w2)
	}
	calc.saveSpec(calc.env.Spec)
	if got := savedSpec(calc); got != "nodes=4739,18845,44871,54447 WeaponSet1=44871 WeaponSet2=4739,18845" {
		t.Fatalf("saved %q", got)
	}
	xml, err := calc.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(xml, `<WeaponSet1 nodes="44871"`) || !strings.Contains(xml, `<WeaponSet2 nodes="4739,18845"`) {
		t.Fatalf("the serialized spec lacks the weapon sets:\n%s", xml)
	}
}

func countSets(calc *Calculator) (int, int, int) {
	used, _, _, _, w1, w2 := calc.env.Spec.CountAllocNodes()
	return used, w1, w2
}

// PoB-PoE2's radius_jewel.xml: socket 26725 holds a Large (radius index 3) jewel, socket 62152 a jewel with
// no radius; node ids in radius 3 of socket 26725 from its nodesInRadius.
func TestJewelRadiusData(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join("..", "..", "calculator2", "testdata", "radius_jewel.xml"))
	if err != nil {
		t.Fatal(err)
	}
	calc, err := ParseBuild(string(xml))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	st, err := calc.TreeState()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.JewelRadiusIndex, []int{26725, 3}) {
		t.Fatalf("jewel radius index %v, PoB [26725 3]", st.JewelRadiusIndex)
	}
	// the fixture's <Socket> items 8 and 16
	jewels := map[int]TreeJewel{}
	for _, j := range st.Jewels {
		jewels[j.Socket] = j
	}
	if jewels[26725] != (TreeJewel{Socket: 26725, Title: "Sol Spark", BaseName: "Time-Lost Ruby"}) || jewels[62152].BaseName != "Sapphire" {
		t.Fatalf("jewels %+v", st.Jewels)
	}
	raw, err := PassiveTreeJSON()
	if err != nil {
		t.Fatal(err)
	}
	var tj treeJSON
	if err := json.Unmarshal([]byte(raw), &tj); err != nil {
		t.Fatal(err)
	}
	// Data.lua:670-682 times PassiveTreeJewelDistanceMultiplier (Misc.lua:37)
	if len(tj.JewelRadius) != 12 || tj.JewelRadius[0] != (jewelRadiusJSON{Inner: 0, Outer: 1200, Col: "#bb6600"}) ||
		tj.JewelRadius[4] != (jewelRadiusJSON{Inner: 780, Outer: 1140, Col: "#d35400"}) {
		t.Fatalf("jewel radii %+v", tj.JewelRadius)
	}
	i := slices.IndexFunc(tj.Nodes, func(n treeNodeJSON) bool { return n.ID == 26725 })
	want := []int{1286, 1352, 5800, 6514, 6900, 8881, 9698, 12817, 13856, 14176, 17138, 17903, 18004, 18073, 18496, 18505, 19674, 21070, 21885,
		21982, 22967, 22975, 25482, 26092, 26176, 27439, 29041, 29993, 31388, 31805, 35048, 36027, 38130, 38921, 39621, 40292, 41493, 43149, 43507,
		43650, 44017, 44461, 45075, 45090, 47371, 47722, 49023, 49198, 49952, 50253, 50616, 51394, 51702, 52126, 52298, 52392, 53194, 53216, 53308,
		53386, 53921, 54485, 54998, 57388, 57703, 58838, 63114}
	if i < 0 || len(tj.Nodes[i].NodesInRadius) != 12 || !slices.Equal(tj.Nodes[i].NodesInRadius[2], want) {
		t.Fatalf("socket 26725 nodes in radius 3: %v", tj.Nodes[i].NodesInRadius)
	}
	if j := slices.IndexFunc(tj.Nodes, func(n treeNodeJSON) bool { return n.Type == "Keystone" }); tj.Nodes[j].NodesInRadius != nil {
		t.Fatal("a keystone carries nodesInRadius")
	}
}

// TreeTab.lua:149-156: Reset leaves the class and ascendancy starts (PassiveSpec.lua:884-892), drops attribute
// choices, and is an undo state of its own.
func TestResetTree(t *testing.T) {
	calc, _ := openFixture(t, "sorceress_spark.xml")
	click(t, calc, 56216, 0)
	if err := calc.AllocateAttributeNode(56216, 3, 0); err != nil {
		t.Fatal(err)
	}
	click(t, calc, 64789, 0)
	if _, err := calc.Calculate(); err != nil {
		t.Fatal(err)
	}
	if err := calc.ResetTree(); err != nil {
		t.Fatal(err)
	}
	out, err := calc.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	st, _ := calc.TreeState()
	if !slices.Equal(st.Alloc, []int{40721, 54447}) || st.Used != 0 || st.AscUsed != 0 || out.Output["EnergyShield"] != 0 {
		t.Fatalf("after reset: alloc %v used %d asc %d ES %v", st.Alloc, st.Used, st.AscUsed, out.Output["EnergyShield"])
	}
	if i := slices.IndexFunc(st.Text, func(n TreeNodeText) bool { return n.ID == 56216 }); i >= 0 {
		t.Fatalf("node 56216 kept its attribute: %+v", st.Text[i])
	}
	xml, _ := calc.Serialize()
	if !strings.Contains(xml, `nodes="40721,54447"`) || strings.Contains(xml, `intNodes="56216"`) {
		t.Fatalf("the saved spec still carries the allocation:\n%s", xml)
	}
	if ok, err := calc.Undo(); err != nil || !ok {
		t.Fatalf("undo %v %v", ok, err)
	}
	if !strings.Contains(savedSpec(calc), "64789") {
		t.Fatalf("undo did not restore the tree: %s", savedSpec(calc))
	}
}

// PassiveTreeView.lua:503-529 (PoB-PoE2): with a point spent off the Warrior's start, a click on Titan's start asks
// first; Connect Path then paths the tree to the Warrior, keeps the spent point and allocates the clicked node.
func TestClickNodeConnectPathConnectsTheTreeToTheClickedClass(t *testing.T) {
	calc, _ := openFixture(t, "sorceress_spark.xml")
	if res := click(t, calc, 44871, 0); res != tree2.ClickChanged {
		t.Fatalf("click 44871: %d", res)
	}
	if res := click(t, calc, 32534, 0); res != tree2.ClickConfirmClassChange {
		t.Fatalf("click on Titan's start: %d, want the Class Change popup", res)
	}
	res, err := calc.ClickNodeConnectPath(32534, 0, 0)
	if err != nil || tree2.ClickResult(res) != tree2.ClickChanged {
		t.Fatalf("Connect Path: %d %v", res, err)
	}
	spec, _ := calc.spec()
	if spec.CurClass.Name != "Warrior" || !spec.Nodes[32534].Alloc || !spec.Nodes[44871].Alloc {
		t.Fatalf("class %s, Titan start %v, 44871 %v", spec.CurClass.Name, spec.Nodes[32534].Alloc, spec.Nodes[44871].Alloc)
	}
	if !spec.IsClassConnected(spec.CurClassID) {
		t.Fatal("the tree is not connected to the Warrior start")
	}
}

// PassiveTreeView.lua:2086-2093 (PoB-PoE2): the tree state carries the note a node's tooltip shows
func TestTreeStateCarriesNodeNotes(t *testing.T) {
	calc, _ := openFixture(t, "sorceress_spark.xml")
	calc.SetTreeNodeNote(44871, "take this first")
	st, err := calc.TreeState()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.Notes, []TreeNote{{ID: 44871, Text: "take this first"}}) {
		t.Fatalf("notes %+v", st.Notes)
	}
}
