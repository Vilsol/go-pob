package builds

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type xmlNode struct {
	Name     string
	Attrs    map[string]string
	Children []*xmlNode
	Text     string
}

func decodeGeneric(raw []byte) (*xmlNode, error) {
	dec := xml.NewDecoder(bytes.NewReader(raw))

	var root *xmlNode
	var stack []*xmlNode

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("token: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			n := &xmlNode{Name: t.Name.Local, Attrs: make(map[string]string, len(t.Attr))}
			for _, a := range t.Attr {
				n.Attrs[a.Name.Local] = a.Value
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, n)
			} else if root == nil {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) == 0 {
				return nil, fmt.Errorf("unbalanced closing tag %s", t.Name.Local)
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(t)
			}
		}
	}

	if root == nil {
		return nil, fmt.Errorf("document has no root element")
	}
	return root, nil
}

func pathCounts(n *xmlNode, prefix string, elems, attrs map[string]int) {
	path := prefix + "/" + n.Name
	elems[path]++
	for k := range n.Attrs {
		attrs[path+"@"+k]++
	}
	for _, c := range n.Children {
		pathCounts(c, path, elems, attrs)
	}
}

type finding struct {
	Builds  int
	Total   int
	Example string
	Sample  string
}

type census struct {
	Builds      int
	ParseErrors map[string]string

	LostElem  map[string]*finding
	AddedElem map[string]*finding
	LostAttr  map[string]*finding
	AddedAttr map[string]*finding

	ValDiff        map[string]*finding
	ValDiffNumOnly map[string]*finding
	TextDiff       map[string]*finding

	SeenElem map[string]int
	SeenAttr map[string]int
}

func newCensus() *census {
	return &census{
		ParseErrors:    map[string]string{},
		LostElem:       map[string]*finding{},
		AddedElem:      map[string]*finding{},
		LostAttr:       map[string]*finding{},
		AddedAttr:      map[string]*finding{},
		ValDiff:        map[string]*finding{},
		ValDiffNumOnly: map[string]*finding{},
		TextDiff:       map[string]*finding{},
		SeenElem:       map[string]int{},
		SeenAttr:       map[string]int{},
	}
}

func bump(m map[string]*finding, key, example string, n int, sample string) {
	f, ok := m[key]
	if !ok {
		f = &finding{Example: example, Sample: sample}
		m[key] = f
	}
	f.Builds++
	f.Total += n
	if f.Sample == "" {
		f.Sample = sample
	}
}

func bumpOnce(m map[string]*finding, key, build, sample string, seen map[string]bool) {
	if seen[key] {
		if f := m[key]; f != nil {
			f.Total++
		}
		return
	}
	seen[key] = true
	bump(m, key, build, 1, sample)
}

func numericallyEqual(a, b string) bool {
	fa, ea := strconv.ParseFloat(a, 64)
	fb, eb := strconv.ParseFloat(b, 64)
	if ea != nil || eb != nil {
		return false
	}
	return fa == fb
}

func compareTree(in, out *xmlNode, path, build string, c *census, seen map[string]bool) {
	for k, v := range in.Attrs {
		ov, ok := out.Attrs[k]
		if !ok {
			continue
		}
		if v == ov {
			continue
		}
		sample := fmt.Sprintf("%q -> %q", v, ov)
		if numericallyEqual(v, ov) {
			bumpOnce(c.ValDiffNumOnly, path+"@"+k, build, sample, seen)
		} else {
			bumpOnce(c.ValDiff, path+"@"+k, build, sample, seen)
		}
	}

	if strings.TrimSpace(in.Text) != strings.TrimSpace(out.Text) {
		bumpOnce(c.TextDiff, path, build, fmt.Sprintf("%d chars -> %d chars",
			len(strings.TrimSpace(in.Text)), len(strings.TrimSpace(out.Text))), seen)
	}

	byName := func(n *xmlNode) map[string][]*xmlNode {
		m := map[string][]*xmlNode{}
		for _, ch := range n.Children {
			m[ch.Name] = append(m[ch.Name], ch)
		}
		return m
	}
	inKids, outKids := byName(in), byName(out)
	for name, list := range inKids {
		peer := outKids[name]
		for i, ch := range list {
			if i >= len(peer) {
				break
			}
			compareTree(ch, peer[i], path+"/"+name, build, c, seen)
		}
	}
}

func roundTripOne(raw []byte, build string, c *census) error {
	inTree, err := decodeGeneric(stripNilAttrs(raw))
	if err != nil {
		return fmt.Errorf("input is not decodable XML: %w", err)
	}

	parsed, err := ParseBuild(raw)
	if err != nil {
		c.ParseErrors[build] = err.Error()
		return nil //nolint:nilerr
	}

	serialized, err := SerializeBuild(parsed)
	if err != nil {
		c.ParseErrors[build] = "serialize: " + err.Error()
		return nil //nolint:nilerr
	}

	outTree, err := decodeGeneric(serialized)
	if err != nil {
		return fmt.Errorf("OUR OWN OUTPUT is not decodable XML: %w", err)
	}

	inElems, inAttrs := map[string]int{}, map[string]int{}
	outElems, outAttrs := map[string]int{}, map[string]int{}
	pathCounts(inTree, "", inElems, inAttrs)
	pathCounts(outTree, "", outElems, outAttrs)

	for p, n := range inElems {
		c.SeenElem[p]++
		if outElems[p] < n {
			bump(c.LostElem, p, build, n-outElems[p], fmt.Sprintf("%d -> %d", n, outElems[p]))
		}
	}
	for p, n := range outElems {
		if inElems[p] < n {
			bump(c.AddedElem, p, build, n-inElems[p], fmt.Sprintf("%d -> %d", inElems[p], n))
		}
	}
	for p, n := range inAttrs {
		c.SeenAttr[p]++
		if outAttrs[p] < n {
			bump(c.LostAttr, p, build, n-outAttrs[p], fmt.Sprintf("%d -> %d", n, outAttrs[p]))
		}
	}
	for p, n := range outAttrs {
		if inAttrs[p] < n {
			bump(c.AddedAttr, p, build, n-inAttrs[p], fmt.Sprintf("%d -> %d", inAttrs[p], n))
		}
	}

	compareTree(inTree, outTree, "/"+inTree.Name, build, c, map[string]bool{})

	c.Builds++
	return nil
}

func runCensus(t *testing.T, dir string, limit int) *census {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("REFUSING: corpus directory %s is unreadable: %v", dir, err)
	}

	c := newCensus()
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".xml" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		t.Fatalf("REFUSING: %s contains no .xml build — a zero from this census would be "+
			"indistinguishable from a corpus that was never read", dir)
	}

	if limit > 0 && len(names) > limit {
		names = names[:limit]
	}

	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("REFUSING: cannot read %s: %v", name, err)
		}
		if err := roundTripOne(raw, name, c); err != nil {
			t.Fatalf("REFUSING on %s: %v", name, err)
		}
	}
	return c
}

func reportSection(t *testing.T, title string, m map[string]*finding, denom map[string]int) {
	t.Helper()

	if len(m) == 0 {
		t.Logf("%s: none", title)
		return
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if m[keys[i]].Builds != m[keys[j]].Builds {
			return m[keys[i]].Builds > m[keys[j]].Builds
		}
		return keys[i] < keys[j]
	})

	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d distinct paths\n", title, len(m))
	fmt.Fprintf(&b, "%8s %8s %10s  %-58s %s\n", "BUILDS", "OF", "OCCURS", "PATH", "SAMPLE / EXAMPLE BUILD")
	for _, k := range keys {
		f := m[k]
		of := ""
		if denom != nil {
			of = strconv.Itoa(denom[k])
		}
		fmt.Fprintf(&b, "%8d %8s %10d  %-58s %s  [%s]\n", f.Builds, of, f.Total, k, f.Sample, f.Example)
	}
	t.Log("\n" + b.String())
}

// TestWritePathRoundTripCensus is the report.
func TestWritePathRoundTripCensus(t *testing.T) {
	dir := corpusDirForLoadedData(t)

	limit := 0
	if os.Getenv("CORPUS") == "" && os.Getenv("WRITEPATH_FULL") == "" {
		limit = 300
	}

	c := runCensus(t, dir, limit)

	t.Logf("round-tripped %d builds from %s (limit=%d, 0 = whole corpus)", c.Builds, dir, limit)
	if len(c.ParseErrors) > 0 {
		t.Logf("ParseBuild/SerializeBuild refused on %d builds", len(c.ParseErrors))
		shown := 0
		for k, v := range c.ParseErrors {
			t.Logf("  %s: %s", k, v)
			if shown++; shown >= 5 {
				break
			}
		}
	}

	reportSection(t, "LOST ELEMENTS (in input, missing from our save)", c.LostElem, c.SeenElem)
	reportSection(t, "LOST ATTRIBUTES (in input, missing from our save)", c.LostAttr, c.SeenAttr)
	reportSection(t, "ADDED ELEMENTS (we write what PoB never wrote)", c.AddedElem, c.SeenElem)
	reportSection(t, "ADDED ATTRIBUTES (we write what PoB never wrote)", c.AddedAttr, c.SeenAttr)
	reportSection(t, "CHANGED VALUES (survives, but the value differs)", c.ValDiff, c.SeenAttr)
	reportSection(t, "REFORMATTED NUMBERS (same value, different spelling)", c.ValDiffNumOnly, c.SeenAttr)
	reportSection(t, "CHANGED ELEMENT TEXT", c.TextDiff, c.SeenElem)
}

// TestWritePathCensusSeesADeliberateLoss is the CONTROL.
func TestWritePathCensusSeesADeliberateLoss(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<PathOfBuilding>
	<Build level="93" className="Witch" ascendClassName="Necromancer" targetVersion="3_0" zzUnmodelledAttr="keep-me">
		<ZzUnmodelledElement id="7"/>
	</Build>
	<Tree activeSpec="1"><Spec treeVersion="3_18" nodes="1,2,3"/></Tree>
	<Skills activeSkillSet="1"><SkillSet id="1"><Skill enabled="true"/></SkillSet></Skills>
	<Notes>keep this prose</Notes>
</PathOfBuilding>`

	c := newCensus()
	if err := roundTripOne([]byte(doc), "control.xml", c); err != nil {
		t.Fatalf("control document did not round-trip at all: %v", err)
	}

	if _, ok := c.LostAttr["/PathOfBuilding/Build@zzUnmodelledAttr"]; !ok {
		t.Fatalf("CONTROL FAILED: the census cannot see a lost ATTRIBUTE. Lost attrs seen: %v",
			sortedKeys(c.LostAttr))
	}
	if _, ok := c.LostElem["/PathOfBuilding/Build/ZzUnmodelledElement"]; !ok {
		t.Fatalf("CONTROL FAILED: the census cannot see a lost ELEMENT. Lost elems seen: %v",
			sortedKeys(c.LostElem))
	}

	if _, ok := c.LostAttr["/PathOfBuilding/Build@className"]; ok {
		t.Fatal("CONTROL FAILED: className is modelled by pob.Build and must survive the round trip")
	}
	if strings.TrimSpace(c.TextDiff["/PathOfBuilding/Notes"].safeSample()) != "" {
		t.Fatalf("CONTROL FAILED: <Notes> text changed: %s", c.TextDiff["/PathOfBuilding/Notes"].Sample)
	}
}

func (f *finding) safeSample() string {
	if f == nil {
		return ""
	}
	return f.Sample
}

func sortedKeys(m map[string]*finding) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
