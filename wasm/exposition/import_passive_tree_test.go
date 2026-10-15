package exposition

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

type passiveTreeOracleState struct {
	Message                  string   `json:"message"`
	ClassID                  int      `json:"classId"`
	AscendClassID            int      `json:"ascendClassId"`
	SecondaryAscendClassID   int      `json:"secondaryAscendClassId"`
	TreeVersion              string   `json:"treeVersion"`
	ClusterHashFormatVersion int      `json:"clusterHashFormatVersion"`
	Nodes                    []int64  `json:"nodes"`
	Masteries                []string `json:"masteries"`
	Sockets                  []string `json:"sockets"`
	Overrides                []string `json:"overrides"`
	Items                    []struct {
		ID  int    `json:"id"`
		Raw string `json:"raw"`
	} `json:"items"`
	Level             int     `json:"level"`
	LevelAuto         bool    `json:"levelAuto"`
	Bandit            string  `json:"bandit"`
	PantheonMajorGod  string  `json:"pantheonMajorGod"`
	PantheonMinorGod  string  `json:"pantheonMinorGod"`
	ResistancePenalty float64 `json:"resistancePenalty"`
}

func gradePassiveTree(t *testing.T, label string, build *pob.PathOfBuilding, msg string, want passiveTreeOracleState) {
	t.Helper()
	spec := build.Tree.Specs[build.ActiveSpecIndex()]
	if "^x33FF77"+msg != want.Message {
		t.Errorf("%s: message %q, PoB %q", label, msg, want.Message)
	}
	secondary := 0
	if spec.SecondaryAscendClassID != nil {
		secondary = *spec.SecondaryAscendClassID
	}
	if int(spec.ClassID) != want.ClassID || int(spec.AscendClassID) != want.AscendClassID || secondary != want.SecondaryAscendClassID ||
		string(spec.TreeVersion) != want.TreeVersion || spec.ClusterHashFormatVersion == nil || *spec.ClusterHashFormatVersion != want.ClusterHashFormatVersion {
		t.Errorf("%s: class %d/%d/%d tree %s, PoB %d/%d/%d tree %s", label, spec.ClassID, spec.AscendClassID, secondary, spec.TreeVersion,
			want.ClassID, want.AscendClassID, want.SecondaryAscendClassID, want.TreeVersion)
	}
	if build.Build.ClassName != pob.ClassNameByID[spec.ClassID] || build.Build.AscendClassName != pob.AscendancyNameOf(spec.ClassID, spec.AscendClassID) {
		t.Errorf("%s: the build's class %s/%s is not the tree's", label, build.Build.ClassName, build.Build.AscendClassName)
	}

	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	var wantNodes []int64
	for _, id := range want.Nodes {
		node := tree.Nodes[strconv.FormatInt(id, 10)]
		if node.ClassStartIndex == nil && (node.IsAscendancyStart == nil || !*node.IsAscendancyStart) {
			wantNodes = append(wantNodes, id)
		}
	}
	gotNodes := pob.SpecNodes(spec)
	slices.Sort(gotNodes)
	if !slices.Equal(gotNodes, wantNodes) || !slices.Equal(build.Build.PassiveNodes, pob.SpecNodes(spec)) {
		t.Errorf("%s: nodes\n%v\nPoB\n%v", label, gotNodes, wantNodes)
	}

	selections := pob.SpecMasterySelections(spec)
	masteries := make([]string, 0, len(selections))
	sockets := make([]string, 0, len(spec.Sockets))
	overrides := make([]string, 0, len(spec.Overrides))
	for _, m := range selections {
		masteries = append(masteries, fmt.Sprintf("%d:%d", m.Node, m.Effect))
	}
	for _, s := range spec.Sockets {
		sockets = append(sockets, fmt.Sprintf("%d:%d", s.NodeID, s.ItemID))
	}
	for _, o := range spec.Overrides {
		overrides = append(overrides, fmt.Sprintf("%d:%s", o.NodeID, o.DN))
	}
	for _, pair := range []struct {
		name      string
		got, want []string
	}{{"masteries", masteries, want.Masteries}, {"sockets", sockets, want.Sockets}, {"overrides", overrides, want.Overrides}} {
		slices.Sort(pair.got)
		if !slices.Equal(pair.got, pair.want) {
			t.Errorf("%s: %s %v, PoB %v", label, pair.name, pair.got, pair.want)
		}
	}

	if len(build.Items.Items) != len(want.Items) {
		t.Errorf("%s: %d items, PoB %d", label, len(build.Items.Items), len(want.Items))
	}
	for k, item := range build.Items.Items {
		if k < len(want.Items) && (item.ID != want.Items[k].ID || html.UnescapeString(item.Text) != want.Items[k].Raw) {
			t.Errorf("%s: item %d (id %d):\n%s\nPoB (id %d):\n%s", label, k, item.ID, html.UnescapeString(item.Text), want.Items[k].ID, want.Items[k].Raw)
		}
	}

	penalty, _ := build.GetNumberOptionOK("resistancePenalty")
	if build.Build.Level != want.Level || build.Build.CharacterLevelAutoMode != want.LevelAuto || penalty != want.ResistancePenalty ||
		build.GetStringOption("bandit") != want.Bandit || build.GetStringOption("pantheonMajorGod") != want.PantheonMajorGod ||
		build.GetStringOption("pantheonMinorGod") != want.PantheonMinorGod {
		t.Errorf("%s: level %d auto %v penalty %v bandit %q gods %q %q, PoB %d %v %v %q %q %q", label, build.Build.Level, build.Build.CharacterLevelAutoMode,
			penalty, build.GetStringOption("bandit"), build.GetStringOption("pantheonMajorGod"), build.GetStringOption("pantheonMinorGod"),
			want.Level, want.LevelAuto, want.ResistancePenalty, want.Bandit, want.PantheonMajorGod, want.PantheonMinorGod)
	}
}

// TestImportPassiveTreeAndJewelsAgainstPoB: ImportTab.lua:1130-1252 on PoB's sample character.
func TestImportPassiveTreeAndJewelsAgainstPoB(t *testing.T) {
	initItemData(t)
	_, char := readImportOracle(t)
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "import", "import_passive_tree.json"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var oracle struct{ First, Second, Third passiveTreeOracleState }
	if err := json.Unmarshal(b, &oracle); err != nil {
		t.Fatal(err)
	}
	if len(oracle.First.Nodes) == 0 || len(oracle.First.Sockets) == 0 || len(oracle.First.Overrides) == 0 {
		t.Fatal("REFUSING: the oracle imported no tree, jewels or tattoos")
	}
	build := builds.EmptyBuild()
	for _, step := range []struct {
		label        string
		deleteJewels bool
		want         passiveTreeOracleState
	}{{"first", true, oracle.First}, {"second", false, oracle.Second}, {"third", true, oracle.Third}} {
		msg, err := ImportPassiveTreeAndJewels(build, char, step.deleteJewels)
		if err != nil {
			t.Fatalf("%s: %v", step.label, err)
		}
		gradePassiveTree(t, step.label, build, msg, step.want)
	}
}

// TestImportAccountPassiveTree: ImportTab.lua:829-847: the account import keeps the build's quest choices when the answer has none.
func TestImportAccountPassiveTree(t *testing.T) {
	initItemData(t)
	_, char := readImportOracle(t)
	var wrapper struct {
		Character map[string]json.RawMessage `json:"character"`
	}
	if err := json.Unmarshal([]byte(char), &wrapper); err != nil {
		t.Fatal(err)
	}
	var passives map[string]json.RawMessage
	if err := json.Unmarshal(wrapper.Character["passives"], &passives); err != nil {
		t.Fatal(err)
	}
	passives["items"] = wrapper.Character["jewels"]
	delete(passives, "bandit_choice")
	delete(passives, "pantheon_major")
	delete(passives, "pantheon_minor")
	answer, _ := json.Marshal(passives)
	entry, _ := json.Marshal(map[string]any{"name": "Sample", "class": "Hierophant", "league": "Standard", "level": 99})

	build := builds.EmptyBuild()
	oak, lunaris := "Oak", "Lunaris"
	build.SetConfigOption(pob.Input{Name: "bandit", String: &oak})
	build.SetConfigOption(pob.Input{Name: "pantheonMajorGod", String: &lunaris})
	msg, err := ImportAccountPassiveTree(build, string(entry), string(answer), true)
	if err != nil || msg != "Passive tree and jewels successfully imported." {
		t.Fatalf("%q %v", msg, err)
	}
	if build.GetStringOption("bandit") != "Oak" || build.GetStringOption("pantheonMajorGod") != "Lunaris" || build.GetStringOption("pantheonMinorGod") != "None" {
		t.Errorf("quest choices %q %q %q, want the build's Oak Lunaris None", build.GetStringOption("bandit"),
			build.GetStringOption("pantheonMajorGod"), build.GetStringOption("pantheonMinorGod"))
	}
	if len(build.Tree.Specs[build.ActiveSpecIndex()].Sockets) != 11 {
		t.Errorf("%d jewels socketed, want the answer's 11", len(build.Tree.Specs[build.ActiveSpecIndex()].Sockets))
	}
	if _, err := ImportAccountPassiveTree(build, string(entry), "false", true); err == nil || err.Error() != "Failed to retrieve character data, try again." {
		t.Errorf("false answer: %v", err)
	}
	if _, err := ImportAccountItems(build, string(entry), "false", true, true, false); err == nil || err.Error() != "Failed to retrieve character data, try again." {
		t.Errorf("false answer: %v", err)
	}
}

// TestImportPassiveTreeExtraPointsFromItems: Build.lua:883: the act estimate reads the last calculation's ExtraPoints.
func TestImportPassiveTreeExtraPointsFromItems(t *testing.T) {
	initItemData(t)
	_, char := readImportOracle(t)
	var wrapper map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(char), &wrapper); err != nil {
		t.Fatal(err)
	}
	var passives characterPassives
	if err := json.Unmarshal(wrapper["character"]["passives"], &passives); err != nil {
		t.Fatal(err)
	}
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	var hashes []int64
	for _, id := range passives.Hashes {
		node := tree.Nodes[strconv.FormatInt(id, 10)]
		if len(hashes) < 46 && (node.IsMastery == nil || !*node.IsMastery) && (node.AscendancyName == nil || *node.AscendancyName == "") {
			hashes = append(hashes, id)
		}
	}
	answer, _ := json.Marshal(map[string]any{"hashes": hashes, "items": []any{}})
	entry, _ := json.Marshal(map[string]any{"name": "Sample", "class": "Hierophant", "league": "Standard", "level": 50})

	for _, step := range []struct {
		item    bool
		penalty float64
	}{{false, -60}, {true, 0}} {
		build := builds.EmptyBuild()
		alira := "Alira"
		build.SetConfigOption(pob.Input{Name: "bandit", String: &alira})
		build.Build.Bandit = alira
		if step.item {
			build.SetSlotItem("Ring 1", build.AddItem("Rarity: RARE\nTest Band\nIron Ring\nGrants 1 Passive Skill Point"))
		}
		extra := calculator.NewCalculator(*build).BuildOutput(calculator.OutputModeMain).Player.Output["ExtraPoints"]
		if extra != map[bool]float64{false: 0, true: 1}[step.item] {
			t.Fatalf("REFUSING: item %v: ExtraPoints %v", step.item, extra)
		}
		if _, err := ImportAccountPassiveTree(build, string(entry), string(answer), true); err != nil {
			t.Fatal(err)
		}
		if used := calculator.TreeSpecsPoints(build)[build.ActiveSpecIndex()].Used; used != 46 {
			t.Fatalf("REFUSING: %d points used, want 46", used)
		}
		if penalty, _ := build.GetNumberOptionOK("resistancePenalty"); penalty != step.penalty {
			t.Errorf("item %v: resistance penalty %v, PoB %v", step.item, penalty, step.penalty)
		}
	}
}

// TestImportedKeystoneClusterNode: PassiveSpec.lua:2055-2057 (#10272): an imported keystone cluster's node is matched by isKeystone, whatever its orbit index.
func TestImportedKeystoneClusterNode(t *testing.T) {
	initItemData(t)
	build := builds.EmptyBuild()
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	var socketID, proxy string
	for id, node := range tree.Nodes {
		if node.ExpansionJewel != nil && node.ExpansionJewel.Size == 2 && (socketID == "" || id < socketID) {
			socketID, proxy = id, node.ExpansionJewel.Proxy
		}
	}
	itemID := build.AddItem("Rarity: UNIQUE\nOne With Nothing\nSmall Cluster Jewel\nImplicits: 0\nAdds Hollow Palm Technique")
	build.Tree.Specs[build.ActiveSpecIndex()].NodesAttr = socketID
	socket, _ := strconv.Atoi(socketID)
	if ok, err := SocketTreeJewel(build, socket, itemID); !ok || err != nil {
		t.Fatalf("REFUSING: socketing in %s: %v", socketID, err)
	}
	var keystone *calculator.TreeClusterNode
	for _, n := range calculator.TreeClusterNodes(build) {
		if n.Type == "Keystone" {
			keystone = &n
		}
	}
	if keystone == nil {
		t.Fatal("REFUSING: no keystone node in the subgraph")
	}
	for _, c := range []struct {
		isKeystone bool
		want       int
	}{{false, 0}, {true, 1}} {
		answer := fmt.Sprintf(`{"hashes_ex":[123],"jewel_data":{%q:{"subgraph":{"groups":{"keystone":{"proxy":%q,"nodes":["123"]}},"nodes":{"123":{"group":"keystone","isKeystone":%v,"orbitIndex":%d}}}}}}`,
			socketID, proxy, c.isKeystone, keystone.OrbitIndex+1)
		var passives characterPassives
		if err := json.Unmarshal([]byte(answer), &passives); err != nil {
			t.Fatal(err)
		}
		if got := importedSubgraphNodes(build, tree, passives); len(got) != c.want || (c.want == 1 && got[0] != keystone.ID) {
			t.Errorf("isKeystone %v: allocated %v, want %d node(s) (keystone %d)", c.isKeystone, got, c.want, keystone.ID)
		}
	}
}
