package calculator

import (
	"encoding/base64"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func sortedIDs(ids []int64) []int64 {
	out := slices.Clone(ids)
	slices.Sort(out)
	return slices.Compact(out)
}

// TestTreeURLsRoundTripPoBsOwn: PoB writes every spec's official link beside its node list.
func TestTreeURLsRoundTripPoBsOwn(t *testing.T) {
	_, names := corpusNamesForLoadedData(t)
	tree := specTree()
	checked, masteries := 0, 0
	for i, name := range names {
		if i%25 != 0 {
			continue
		}
		build := parseCorpusBuild(t, name)
		idx := build.ActiveSpecIndex()
		if idx < 0 {
			continue
		}
		spec := build.Tree.Specs[idx]
		url := strings.TrimSpace(spec.URL)
		if url == "" {
			continue
		}
		pobs, err := DecodeTreeURL(url)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		classID, ascendID, secondary := specClassIDs(spec)
		if pobs.ClassID != classID || pobs.AscendClassID != ascendID || pobs.SecondaryAscend != secondary&3 {
			t.Errorf("%s: class %d/%d/%d, spec %d/%d/%d", name, pobs.ClassID, pobs.AscendClassID, pobs.SecondaryAscend, classID, ascendID, secondary)
		}
		var want []int64
		bloodline := map[string]bool{}
		for _, a := range tree.AlternateAscendancies {
			bloodline[string(a.ID)] = true
		}
		for _, id := range pob.SpecNodes(spec) {
			if n, ok := tree.Nodes[nodeKey(id)]; ok && !(isAscendClassStartNode(n) && n.AscendancyName != nil && bloodline[*n.AscendancyName]) {
				want = append(want, id)
			}
		}
		wantSel := map[int64]int64{}
		for _, s := range pob.SpecMasterySelections(spec) {
			if slices.Contains(want, s.Node) && masteryHasEffect(tree, s.Node, s.Effect) {
				wantSel[s.Node] = s.Effect
			}
		}
		if got := sortedIDs(pobs.Nodes); !slices.Equal(got, sortedIDs(want)) {
			w := sortedIDs(want)
			var only, missing []int64
			for _, x := range got {
				if !slices.Contains(w, x) {
					only = append(only, x)
				}
			}
			for _, x := range w {
				if !slices.Contains(got, x) {
					missing = append(missing, x)
				}
			}
			t.Errorf("%s: PoB's link decodes to %d nodes, the spec has %d; link only %v, spec only %v", name, len(got), len(w), only, missing)
		}
		if len(pobs.MasterySelection) != len(wantSel) {
			t.Errorf("%s: PoB's link has %d mastery effects, the spec %d", name, len(pobs.MasterySelection), len(wantSel))
		}
		for n, eff := range wantSel {
			if pobs.MasterySelection[n] != eff {
				t.Errorf("%s: mastery %d: link %d, spec %d", name, n, pobs.MasterySelection[n], eff)
			}
		}
		masteries += len(wantSel)

		ours, err := DecodeTreeURL(EncodeTreeURL(build, "https://www.pathofexile.com/passive-skill-tree/"))
		if err != nil {
			t.Fatalf("%s: our link: %v", name, err)
		}
		if !slices.Equal(sortedIDs(ours.Nodes), sortedIDs(pobs.Nodes)) || ours.ClassID != pobs.ClassID || ours.AscendClassID != pobs.AscendClassID {
			t.Errorf("%s: our link decodes to %d nodes (class %d/%d), PoB's to %d (%d/%d)", name, len(ours.Nodes), ours.ClassID, ours.AscendClassID, len(pobs.Nodes), pobs.ClassID, pobs.AscendClassID)
		}
		if len(ours.MasterySelection) != len(pobs.MasterySelection) {
			t.Errorf("%s: our link has %d mastery effects, PoB's %d", name, len(ours.MasterySelection), len(pobs.MasterySelection))
		}
		checked++
	}
	if checked < 400 || masteries == 0 {
		t.Fatalf("only %d builds, %d mastery effects checked", checked, masteries)
	}
}

// TestDecodeTreeURL: PoB's messages for links it cannot read, and a link PoB's own comment gives (TreeTab.lua:797).
func TestDecodeTreeURL(t *testing.T) {
	for url, msg := range map[string]string{
		"https://www.pathofexile.com/passive-skill-tree/AAAA":         "Invalid tree link (unrecognised format)",
		"https://www.pathofexile.com/passive-skill-tree/AAAABwMAAA==": "Invalid tree link (unknown version number '7')",
		"https://www.pathofexile.com/passive-skill-tree/AAAABgkAAA==": "Invalid tree link (bad class ID '9')",
	} {
		if _, err := DecodeTreeURL(url); err == nil || err.Error() != msg {
			t.Errorf("%s: %v, want %q", url, err, msg)
		}
	}
	got, err := DecodeTreeURL("https://www.pathofexile.com/passive-skill-tree/3.15.0/AAAABgMADI6-HwKSwQQHLJwtH9-wTLNfKoP3ES3r5AAA")
	if err != nil {
		t.Fatal(err)
	}
	if got.ClassID != 3 || got.AscendClassID != 0 || len(got.MasterySelection) != 0 || got.Nodes[0] != startOf(t, "Witch") || len(got.Nodes) > 13 {
		t.Fatalf("%+v", got)
	}
	if v := treeLinkVersion("https://www.pathofexile.com/passive-skill-tree/3.20.0/AAAA"); v != "3_20" {
		t.Errorf("3.20 link: %s", v)
	}
	if v := treeLinkVersion("https://www.pathofexile.com/passive-skill-tree/ruthless/AAAA"); v != "3_29" {
		t.Errorf("ruthless link (no such tree here): %s", v)
	}
}

// TestImportTreeURLAddsTheTree: Import Tree adds the link's tree under its title and makes it the active one.
func TestImportTreeURLAddsTheTree(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	want := slices.Clone(build.Build.PassiveNodes)
	url := EncodeTreeURL(build, "https://www.pathofexile.com/passive-skill-tree/3.29.0/")
	build.SetActiveSpec(1)
	if err := ImportTreeURL(build, "Imported", url); err != nil {
		t.Fatal(err)
	}
	if n := len(build.Tree.Specs); n != 4 || build.Tree.ActiveSpec != 4 || pob.SpecTitle(build.Tree.Specs[3]) != "Imported" {
		t.Fatalf("%d trees, active %d", n, build.Tree.ActiveSpec)
	}
	if !slices.Equal(sortedIDs(build.Build.PassiveNodes), sortedIDs(want)) || build.Build.AscendClassName != "Elementalist" {
		t.Fatalf("imported %d nodes (%s), the tree had %d", len(build.Build.PassiveNodes), build.Build.AscendClassName, len(want))
	}
	if err := ImportTreeURL(build, "Bad", "https://www.pathofexile.com/passive-skill-tree/AAAA"); err == nil || len(build.Tree.Specs) != 4 {
		t.Fatalf("a bad link: %v, %d trees", err, len(build.Tree.Specs))
	}
}

func poePlannerLink(version, class, ascendancy int, nodes, cluster, ascNodes []int64, mastery [][2]int64) string {
	le := func(v int) []byte { return []byte{byte(v & 0xff), byte(v >> 8 & 0xff)} }
	b := append([]byte{}, le(1)...)
	b = append(b, 0, 0)
	b = append(b, le(1)...)
	b = append(b, le(version)...)
	b = append(b, le(class)[0], le(ascendancy)[0], 0)
	for _, l := range [][]int64{nodes, cluster, ascNodes} {
		b = append(b, le(len(l))...)
		for _, id := range l {
			b = append(b, le(int(id))...)
		}
	}
	b = append(b, le(len(mastery))...)
	for _, m := range mastery {
		b = append(b, le(int(m[0]))...)
		b = append(b, le(int(m[1]))...)
	}
	b = append(b, 0, 0, 0, 0)
	return "https://poeplanner.com/" + base64.URLEncoding.EncodeToString(b)
}

// TestDecodePoePlannerURL: A PoEPlanner link of a corpus tree reads back as that tree.
func TestDecodePoePlannerURL(t *testing.T) {
	build := parseCorpusBuild(t, "0280-elementalist-fireball.xml")
	spec := build.Tree.Specs[build.ActiveSpecIndex()]
	tree := specTree()
	var main, asc []int64
	for _, id := range pob.SpecNodes(spec) {
		n, ok := tree.Nodes[nodeKey(id)]
		switch {
		case !ok || isClassStartNode(n) || isAscendClassStartNode(n):
		case n.AscendancyName != nil:
			asc = append(asc, id)
		default:
			main = append(main, id)
		}
	}
	sel := pob.SpecMasterySelections(spec)
	mastery := make([][2]int64, 0, len(sel))
	for _, s := range sel {
		mastery = append(mastery, [2]int64{s.Node, s.Effect})
	}
	classID, ascendID, _ := specClassIDs(spec)
	link := poePlannerLink(51, classID, ascendID, main, nil, asc, mastery)
	got, err := decodeTreeLink(link)
	if err != nil {
		t.Fatal(err)
	}
	want := sortedIDs(append(append(append([]int64{}, main...), asc...), startOf(t, "Witch"), mustStart(t, "Witch", "Elementalist")))
	if !slices.Equal(sortedIDs(got.Nodes), want) || got.ClassID != classID || got.AscendClassID != ascendID || got.Version != "3_29" {
		t.Fatalf("%d nodes (class %d/%d, %s), want %d", len(got.Nodes), got.ClassID, got.AscendClassID, got.Version, len(want))
	}
	if len(got.MasterySelection) != len(mastery) || len(mastery) == 0 {
		t.Fatalf("%d mastery effects, want %d", len(got.MasterySelection), len(mastery))
	}
	if _, err := decodeTreeLink(poePlannerLink(99, classID, 0, nil, nil, nil, nil)); err == nil || err.Error() != "Invalid tree version found in link." {
		t.Fatalf("unknown version: %v", err)
	}
	official := EncodeTreeURL(build, "")
	got, err = decodeTreeLink("https://poeskilltree.com/?v=3.20.0#" + official)
	if err != nil || got.Version != "3_20" || len(got.Nodes) < len(main) {
		t.Fatalf("poeskilltree: %v, %s, %d nodes", err, got.Version, len(got.Nodes))
	}
}

func mustStart(t *testing.T, class pob.ClassName, ascendancy pob.AscendancyName) int64 {
	t.Helper()
	classID, _ := pob.ClassIDByName(class)
	id, err := strconv.ParseInt(classAscendancyStartNodeID(specTree(), classID, ascendancy), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
