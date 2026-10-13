package exposition

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
)

func treeNodeNamed(t *testing.T, want func(id int64, n data.Node) bool) int64 {
	t.Helper()
	ids := make([]int64, 0)
	for k, n := range data.TreeVersions[data.LatestTreeVersion].Tree().Nodes {
		id, _ := strconv.ParseInt(k, 10, 64)
		if want(id, n) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		t.Fatal("REFUSING: no tree node fits")
	}
	slices.Sort(ids)
	return ids[0]
}

func choiceNames(c []TattooChoice) []string {
	out := make([]string, len(c))
	for i, x := range c {
		out[i] = x.DN
	}
	return out
}

// TestTattooChoicesFilterByTarget: TreeTab.lua:846-878: the popup's list per node type, legacy rows only when asked, sorted by label.
func TestTattooChoicesFilterByTarget(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0007-elementalist-storm-burst-of-repulsion.xml")
	name := func(want string) func(int64, data.Node) bool {
		return func(_ int64, n data.Node) bool { return n.Name != nil && *n.Name == want && n.AscendancyName == nil }
	}

	str := GetTattooChoices(build, treeNodeNamed(t, name("Strength")), false)
	if len(str) == 0 {
		t.Fatal("a Strength node offers no tattoo")
	}
	if !sort.SliceIsSorted(str, func(i, j int) bool { return str[i].Label < str[j].Label }) {
		t.Error("the list is not sorted by label")
	}
	for _, c := range str {
		tt := data.TattooTargets[c.DN]
		if tt.Legacy {
			t.Errorf("%s is legacy but listed without Show Legacy Tattoos", c.DN)
		}
		if !strings.Contains("Strength", strings.TrimPrefix(tt.TargetType, "Small ")) && tt.TargetType != "Small Attribute" {
			t.Errorf("%s targets %q, not a Strength node", c.DN, tt.TargetType)
		}
		if !slices.Equal(c.Descriptions[:len(data.TattooNodes[c.DN].Stats)], data.TattooNodes[c.DN].Stats) {
			t.Errorf("%s: descriptions %q do not start with its lines", c.DN, c.Descriptions)
		}
	}
	if !slices.Contains(choiceNames(str), "Tattoo of the Ngamahu Firewalker") {
		t.Errorf("Tattoo of the Ngamahu Firewalker missing from %q", choiceNames(str))
	}
	legacy := GetTattooChoices(build, treeNodeNamed(t, name("Strength")), true)
	if len(legacy) <= len(str) {
		t.Errorf("Show Legacy Tattoos adds nothing: %d -> %d", len(str), len(legacy))
	}

	keystone := GetTattooChoices(build, treeNodeNamed(t, func(_ int64, n data.Node) bool { return n.IsKeystone != nil && *n.IsKeystone }), false)
	if !slices.Contains(choiceNames(keystone), "Acrobatics") {
		t.Errorf("a keystone is not offered Acrobatics: %q", choiceNames(keystone))
	}
	if len(keystone) == 0 || keystone[0].DN != "" || !slices.Equal(keystone[0].Descriptions, []string{"Limited to 1 Ascendancy Tattoo"}) {
		t.Errorf("a keystone's first choice is not the nameless row: %+v", keystone[:min(1, len(keystone))])
	}
	mastery := GetTattooChoices(build, treeNodeNamed(t, func(_ int64, n data.Node) bool {
		return n.IsMastery != nil && *n.IsMastery && len(n.MasteryEffects) > 0
	}), false)
	for _, c := range mastery {
		if !strings.HasPrefix(c.DN, "Runegraft") {
			t.Errorf("a mastery is offered %s", c.DN)
		}
	}
	if len(mastery) == 0 {
		t.Error("a mastery offers no runegraft")
	}
	few := GetTattooChoices(build, treeNodeNamed(t, func(_ int64, n data.Node) bool {
		return n.Name != nil && *n.Name == "Strength" && len(n.Out)+len(n.In) < 8
	}), true)
	if slices.Contains(choiceNames(few), "Honoured Tattoo of the Makanga") {
		t.Error("a node with fewer than 8 links is offered Honoured Tattoo of the Makanga")
	}
	notable := GetTattooChoices(build, treeNodeNamed(t, func(_ int64, n data.Node) bool {
		return n.IsNotable != nil && *n.IsNotable && len(n.Stats) == 0
	}), true)
	if len(notable) != 0 {
		t.Errorf("a notable with no lines is offered %q", choiceNames(notable))
	}
}

// TestAddTattooRoundTrips: Add on an allocated attribute node (TreeTab.lua:879-885) replaces its stats, and the build saves and reloads with it.
func TestAddTattooRoundTrips(t *testing.T) {
	initItemData(t)
	const file = "0007-elementalist-storm-burst-of-repulsion.xml"
	build := corpusBuild(t, file)
	node := treeNodeNamed(t, func(id int64, n data.Node) bool {
		return n.Name != nil && (*n.Name == "Intelligence" || *n.Name == "Dexterity") && slices.Contains(build.Build.PassiveNodes, id)
	})
	choices := GetTattooChoices(build, node, false)
	if len(choices) == 0 {
		t.Fatal("REFUSING: the allocated attribute node offers nothing")
	}
	base := GetMiscCalculator(corpusBuild(t, file)).Base().Output
	dn := choices[0].DN
	if err := AddTattoo(build, node, dn); err != nil {
		t.Fatal(err)
	}
	if got := build.TattooOn(int(node)); got != dn {
		t.Fatalf("TattooOn = %q after adding %q", got, dn)
	}
	if err := AddTattoo(build, node, "Acrobatics"); err == nil {
		t.Error("an attribute node took a keystone tattoo")
	}
	got := GetMiscCalculator(build).Base().Output
	xml, err := builds.SerializeBuild(build)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(xml), `nodeId="`+strconv.FormatInt(node, 10)+`" dn="`+dn+`"`) {
		t.Fatal("the saved build carries no override for the node")
	}
	saved, err := builds.ParseBuild(xml)
	if err != nil {
		t.Fatal(err)
	}
	want := GetMiscCalculator(saved).Base().Output
	differs := false
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %v after Add, %v reloaded", k, got[k], v)
		}
		differs = differs || base[k] != v
	}
	if !differs {
		t.Errorf("vacuity: %s on %d changes nothing", dn, node)
	}
	if err := AddTattoo(build, node, choices[len(choices)-1].DN); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, o := range build.Tree.Specs[build.ActiveSpecIndex()].Overrides {
		if int64(o.NodeID) == node {
			n++
		}
	}
	if n != 1 || build.TattooOn(int(node)) != choices[len(choices)-1].DN {
		t.Errorf("replacing left %d overrides, the node shows %q", n, build.TattooOn(int(node)))
	}
}

// TestTreeTattooArtComesFromTheRows: 0007's runegrafts draw the tattoo rows' art, not the attributes the build file saved.
func TestTreeTattooArtComesFromTheRows(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0007-elementalist-storm-burst-of-repulsion.xml")
	art := TreeTattooArt(build)
	if len(art) != len(build.Tree.Specs[build.ActiveSpecIndex()].Overrides) {
		t.Fatalf("%d art rows for %d overrides", len(art), len(build.Tree.Specs[build.ActiveSpecIndex()].Overrides))
	}
	for _, a := range art {
		want := data.TattooTargets[build.TattooOn(a.NodeID)]
		if a.Icon != want.Icon || a.ActiveEffectImage != want.ActiveEffectImage || a.Icon == "" {
			t.Errorf("node %d: %q / %q, the row has %q / %q", a.NodeID, a.Icon, a.ActiveEffectImage, want.Icon, want.ActiveEffectImage)
		}
	}
	build.Tree.Specs[build.ActiveSpecIndex()].Overrides[0].DN = "No such tattoo"
	if len(TreeTattooArt(build)) != len(art)-1 {
		t.Error("an unknown tattoo still draws")
	}
}
