package calculator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

// for id, node in pairs(self.nodes) do
//
//	-- If node is tattooed, replace it
//	if self.hashOverrides[node.id] then
//		self:ReplaceNode(node, self.hashOverrides[node.id])
//	end
//	-- If node is conquered, replace it or add mods
//	if node.conqueredBy and node.type ~= "Socket" then
const (
	tattooBuild  = "0001-necromancer-unknown.xml"
	tattooNodeID = 5296
	tattooDN     = "Tattoo of the Tawhoa Shaman"
	tattooSource = "Tree:Tawhoa5"
	origSource   = "Tree:5296"
)

// TestTattooOverrideIsParsed pins the input half on its own.
func TestTattooOverrideIsParsed(t *testing.T) {
	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tattooBuild))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}

	var found *string
	total := 0
	for _, spec := range parsed.Tree.Specs {
		total += len(spec.Overrides)
		for _, o := range spec.Overrides {
			if o.NodeID == tattooNodeID {
				dn := o.DN
				found = &dn
			}
		}
	}

	if total == 0 {
		t.Fatalf("REFUSING: %s parsed 0 <Override> elements. The file contains them, so this is a "+
			"parser gap, not a build without tattoos", tattooBuild)
	}
	if found == nil {
		t.Fatalf("node %d has no override among %d parsed", tattooNodeID, total)
	}
	if *found != tattooDN {
		t.Errorf("node %d override dn = %q, want %q", tattooNodeID, *found, tattooDN)
	}
}

// TestTattooReplacesNodeStats is the defect itself: the node must grant the TATTOO's stats, sourced to the tattoo's id, and must stop granting its own.
func TestTattooReplacesNodeStats(t *testing.T) {
	row, ok := data.TattooNodes[tattooDN]
	if !ok {
		t.Fatalf("REFUSING: %q is absent from data.TattooNodes — regenerate with "+
			"`task data-regen -- --only tattoo-passives`", tattooDN)
	}
	if len(row.Stats) == 0 {
		t.Fatalf("REFUSING: %q carries no stat lines, so replacing with it is indistinguishable "+
			"from not replacing at all", tattooDN)
	}

	xml, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), tattooBuild))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	env := (&Calculator{PoB: parsed}).BuildOutput(OutputModeMain)

	sources := map[string]int{}
	for _, mods := range env.ModDB.Mods {
		for _, m := range mods {
			sources[string(m.GetSource())]++
		}
	}

	if sources[tattooSource] == 0 {
		t.Errorf("%s: no mod sourced %q. PassiveSpec.lua:1188 replaces node %d with that row, so "+
			"its stat lines must carry the TATTOO's id as their source, not the node's.",
			tattooBuild, tattooSource, tattooNodeID)
	}
	if n := sources[origSource]; n != 0 {
		t.Errorf("%s: %d mod(s) still sourced %q. ReplaceNode overwrites node.sd wholesale, so the "+
			"original passive must stop granting anything.", tattooBuild, n, origSource)
	}
}
