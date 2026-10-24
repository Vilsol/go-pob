package tree2

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// specXML is the slice of a PoE2 build file this package needs; the build-file types live elsewhere.
type specXML struct {
	Tree struct {
		ActiveSpec string `xml:"activeSpec,attr"`
		Specs      []struct {
			Attrs    []xml.Attr `xml:",any,attr"`
			Children []struct {
				XMLName xml.Name
				Attrs   []xml.Attr `xml:",any,attr"`
				Inner   []struct {
					XMLName xml.Name
					Attrs   []xml.Attr `xml:",any,attr"`
				} `xml:",any"`
			} `xml:",any"`
		} `xml:"Spec"`
	} `xml:"Tree"`
}

func attrMap(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

func ptr(m map[string]string, k string) *string {
	if v, ok := m[k]; ok {
		return &v
	}
	return nil
}

var weaponSetRE = regexp.MustCompile(`^WeaponSet(\d)`)

// activeSpecInput reads the active <Spec> the way TreeTab:Load picks it.
func activeSpecInput(t *testing.T, raw []byte) SpecInput {
	t.Helper()
	var doc specXML
	if err := xml.Unmarshal(raw, &doc); err != nil { //nolint:gosec // a local build file
		t.Fatal(err)
	}
	idx := 1
	if v, err := strconv.Atoi(doc.Tree.ActiveSpec); err == nil {
		idx = v
	}
	if idx < 1 || idx > len(doc.Tree.Specs) {
		t.Fatalf("activeSpec %d of %d specs", idx, len(doc.Tree.Specs))
	}
	spec := doc.Tree.Specs[idx-1]
	a := attrMap(spec.Attrs)
	in := SpecInput{
		Title: a["title"], TreeVersion: a["treeVersion"], Nodes: ptr(a, "nodes"), ClassID: ptr(a, "classId"),
		AscendClassID: ptr(a, "ascendClassId"), ClassInternalID: ptr(a, "classInternalId"),
		AscendancyInternalID: ptr(a, "ascendancyInternalId"), SecondaryAscendClassID: ptr(a, "secondaryAscendClassId"),
		MasteryEffects: ptr(a, "masteryEffects"), WeaponSets: map[int]string{},
	}
	for _, c := range spec.Children {
		ca := attrMap(c.Attrs)
		switch name := c.XMLName.Local; {
		case name == "Sockets":
			for _, s := range c.Inner {
				if s.XMLName.Local == "Socket" {
					sa := attrMap(s.Attrs)
					in.Sockets = append(in.Sockets, SocketInput{NodeID: sa["nodeId"], ItemID: sa["itemId"]})
				}
			}
		case name == "Overrides":
			for _, o := range c.Inner {
				if o.XMLName.Local == "AttributeOverride" {
					oa := attrMap(o.Attrs)
					in.AttributeOverrides = append(in.AttributeOverrides, AttributeOverrideInput{StrNodes: oa["strNodes"], DexNodes: oa["dexNodes"], IntNodes: oa["intNodes"]})
				}
			}
		case weaponSetRE.MatchString(name):
			n, _ := strconv.Atoi(weaponSetRE.FindStringSubmatch(name)[1])
			in.WeaponSets[n] = ca["nodes"]
		}
	}
	return in
}

type dumpTree struct {
	Meta struct {
		Class      string `json:"class"`
		Ascendancy string `json:"ascendancy"`
	} `json:"meta"`
	ModDB map[string][]struct {
		Source any `json:"source"`
	} `json:"modDB"`
}

var allocatesRE = regexp.MustCompile(`(?m)^(?:\{[^}]*\})*Allocates (.+?)\s*$`)

var itemLineRE = regexp.MustCompile(`(?m)^(?:\{[^}]*\})*([^<>\n]+?)\s*$`)

var treeSourceRE = regexp.MustCompile(`^Tree:(\d+)(?: \((.*)\))?$`)

// treeSources maps each Tree:<id> mod source in a dump to the spec node name PoB printed beside it.
func (d *dumpTree) treeSources() map[int]string {
	out := map[int]string{}
	for _, mods := range d.ModDB {
		for _, m := range mods {
			s, ok := m.Source.(string)
			if !ok {
				continue
			}
			if sm := treeSourceRE.FindStringSubmatch(s); sm != nil {
				id, _ := strconv.Atoi(sm[1])
				out[id] = sm[2]
			}
		}
	}
	return out
}

type probeRow struct {
	Build      string `json:"build"`
	Error      string `json:"error"`
	Class      string `json:"class"`
	Ascend     string `json:"ascend"`
	Alloc      []int  `json:"alloc"`
	AllocModes luaMap `json:"allocModes"`
}

// luaMap reads a Lua table dkjson wrote: an empty one comes out as [].
type luaMap map[string]int

func (m *luaMap) UnmarshalJSON(b []byte) error {
	if string(b) == "[]" {
		*m = luaMap{}
		return nil
	}
	return json.Unmarshal(b, (*map[string]int)(m)) //nolint:wrapcheck // decode error passes through as is
}

func readProbe(t *testing.T, path string) map[string]probeRow {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // an operator-supplied file
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out := map[string]probeRow{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var r probeRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		out[r.Build] = r
	}
	return out
}

// TestPoE2SpecAllocationVsOracle loads every corpus build's active <Spec> and checks it against PoB-PoE2:
// every Tree:<id> mod source in the build's dump must be allocated here, under the same node name
// (switchable and attribute replacements included), with the same class and ascendancy. With
// POE2_ALLOC_PROBE=<jsonl> (one PoB allocNodes row per build) the whole allocated set is compared too.
// Gated by CORPUS=true and the local-only testdata/corpus-poe2-0.5; dumps from POE2_DUMPS or beside the corpus.
func TestPoE2SpecAllocationVsOracle(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true not set")
	}
	corpus := filepath.Join("..", "..", "testdata", "corpus-poe2-0.5")
	files, _ := filepath.Glob(filepath.Join(corpus, "*.xml"))
	if len(files) == 0 {
		t.Skipf("no PoE2 corpus at %s", corpus)
	}
	sort.Strings(files)
	dumps := os.Getenv("POE2_DUMPS")
	if dumps == "" {
		if resolved, err := filepath.EvalSymlinks(corpus); err == nil {
			dumps = filepath.Join(filepath.Dir(resolved), "dumps")
		}
	}
	if _, err := os.Stat(dumps); err != nil {
		t.Skipf("no PoE2 dumps at %q — set POE2_DUMPS", dumps)
	}
	var probe map[string]probeRow
	if p := os.Getenv("POE2_ALLOC_PROBE"); p != "" {
		probe = readProbe(t, p)
	}
	tree := pinTree(t)

	offTreeIDs := map[int]int{}
	optionOwner := map[int]int{}
	for id, n := range tree.Nodes {
		for _, o := range n.SwitchOptions {
			if o.ID != n.ID {
				optionOwner[o.ID] = id
			}
		}
	}
	var optionSourced, itemGranted int
	var builds, srcNodes, offTree, missing, nameWrong, classWrong, probed, setAgree, setDisagree, modeWrong int
	first := map[string]string{}
	note := func(kind, msg string) {
		if _, ok := first[kind]; !ok {
			first[kind] = msg
		}
	}
	for _, path := range files {
		base := filepath.Base(path)
		raw, err := os.ReadFile(path) //nolint:gosec // corpus file
		if err != nil {
			t.Fatal(err)
		}
		in := activeSpecInput(t, raw)
		bi := readBuildItems(t, tree, raw)
		spec := NewPassiveSpec(tree)
		spec.AllocateFromNodeRadius = oracleRadiusRows
		spec.Items = func(id int) *Item { return bi.items[id] }
		if err := spec.Load(in); err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		grantSockets(spec, bi)
		builds++

		dumpRaw, err := os.ReadFile(filepath.Join(dumps, strings.TrimSuffix(base, ".xml")+".json")) //nolint:gosec // dump file
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		var d dumpTree
		if err := json.Unmarshal(dumpRaw, &d); err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		if d.Meta.Class != spec.CurClassName || d.Meta.Ascendancy != spec.CurAscendClassName {
			classWrong++
			note("class", base+": PoB "+d.Meta.Class+"/"+d.Meta.Ascendancy+", ours "+spec.CurClassName+"/"+spec.CurAscendClassName)
		}
		granted := map[int]bool{}
		for _, m := range allocatesRE.FindAllStringSubmatch(string(raw), -1) {
			for _, n := range spec.ResolveGrantedPassiveNodes(GrantedPassive{Name: html.UnescapeString(m[1])}) {
				granted[n.ID] = true
			}
		}
		// A bare keystone name on an item grants that keystone.
		for _, m := range itemLineRE.FindAllStringSubmatch(string(raw), -1) {
			if k, ok := tree.KeystoneMap[strings.ToLower(html.UnescapeString(m[1]))]; ok {
				granted[k.ID] = true
			}
		}
		for id, name := range d.treeSources() {
			srcNodes++
			node, ok := spec.AllocNodes[id]
			if !ok {
				if owner, isOption := optionOwner[id]; isOption {
					// ReplaceNode copies the option's mod list, whose sources carry the option's id.
					if n, ok := spec.AllocNodes[owner]; ok && spec.SwitchableNodes[id] == n {
						optionSourced++
						continue
					}
				}
				if _, onTree := tree.Nodes[id]; !onTree {
					offTree++
					offTreeIDs[id]++
					continue
				}
				if granted[id] {
					itemGranted++
					continue
				}
				missing++
				note("missing", base+": Tree:"+strconv.Itoa(id)+" ("+name+") is a PoB mod source but not allocated here")
				continue
			}
			if name != "" && node.Dn != name {
				nameWrong++
				note("name", base+": node "+strconv.Itoa(id)+" PoB "+strconv.Quote(name)+", ours "+strconv.Quote(node.Dn))
			}
		}

		row, ok := probe[base]
		if !ok || row.Error != "" {
			continue
		}
		probed++
		ours := sortedKeys(spec.AllocNodes)
		if len(ours) == len(row.Alloc) && equalInts(ours, row.Alloc) {
			setAgree++
		} else {
			setDisagree++
			note("set", base+": PoB "+strconv.Itoa(len(row.Alloc))+" allocated, ours "+strconv.Itoa(len(ours)))
		}
		for _, id := range ours {
			want := row.AllocModes[strconv.Itoa(id)]
			if spec.AllocNodes[id].AllocMode != want {
				modeWrong++
				note("mode", base+": node "+strconv.Itoa(id)+" PoB allocMode "+strconv.Itoa(want)+", ours "+strconv.Itoa(spec.AllocNodes[id].AllocMode))
			}
		}
	}
	t.Logf("builds %d; dump Tree sources %d: on the tree but not allocated here %d, name differs %d; class/ascendancy differs %d",
		builds, srcNodes, missing, nameWrong, classWrong)
	t.Logf("of which: sourced by the option id an allocated switchable node switched to here %d; granted by an item's \"Allocates <name>\" or keystone line %d; "+
		"neither a tree node nor an option %d over %d distinct ids %v",
		optionSourced, itemGranted, offTree, len(offTreeIDs), offTreeIDs)
	if probe != nil {
		t.Logf("PoB allocNodes probe: %d builds compared, allocated set agrees %d / disagrees %d, allocMode differs on %d nodes",
			probed, setAgree, setDisagree, modeWrong)
		if probed == 0 {
			t.Errorf("POE2_ALLOC_PROBE matched no corpus build")
		}
	}
	for _, k := range []string{"missing", "name", "class", "set", "mode"} {
		if m, ok := first[k]; ok {
			t.Logf("first %s: %s", k, m)
		}
	}
	if srcNodes == 0 {
		t.Fatalf("the dumps carried no Tree:<id> source — the dump shape changed")
	}
	if missing+offTree+nameWrong+classWrong+setDisagree+modeWrong != 0 {
		t.Errorf("allocation disagrees with PoB-PoE2")
	}
}

func equalInts(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
