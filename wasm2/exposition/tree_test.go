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
	res, err := calc.ClickNode(id, false, hotkey, false)
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
	if got := calc.HoverNode(44871); !slices.Equal(got.Path, []int{44871}) {
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

	if dep := calc.HoverNode(44871).Depends; !slices.Equal(dep, []int{44871}) {
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
	if err := calc.AllocateAttributeNode(56216, 3); err != nil {
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
