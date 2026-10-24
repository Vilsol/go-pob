package builds

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func TestSplitSections(t *testing.T) {
	for _, tc := range []struct {
		name, doc string
		ok        bool
		root      string
		sections  []string
	}{
		{"plain", `<?xml version="1.0"?><R a="1"><A x="y"/><B><C/><B>t</B></B><!-- <D> --><E q='>"' r=">">x</E></R>`, true, "R", []string{`<A x="y"/>`, `<B><C/><B>t</B></B>`, `<E q='>"' r=">">x</E>`}},
		{"cdata", "<R><A><![CDATA[</A><B>]]></A>\n<B></B></R>\n", true, "R", []string{"<A><![CDATA[</A><B>]]></A>", "<B></B>"}},
		{"doctype", `<!DOCTYPE R><R><A/></R>`, false, "", nil},
		{"unclosed", `<R><A></R>`, false, "", nil},
		{"two roots", `<R/><S/>`, false, "", nil},
		{"open tag", `<R><A`, false, "", nil},
		{"no root", `text`, false, "", nil},
	} {
		prefix, root, sections, ok := splitSections([]byte(tc.doc))
		if ok != tc.ok {
			t.Errorf("%s: ok %v", tc.name, ok)
			continue
		}
		if !ok {
			continue
		}
		var got []string
		for _, s := range sections {
			got = append(got, tc.doc[s.start:s.end])
		}
		if root != tc.root || !slices.Equal(got, tc.sections) || len(prefix) == 0 {
			t.Errorf("%s: root %q prefix %q sections %q", tc.name, root, prefix, got)
		}
	}
}

func parseCorpus(t *testing.T, name string) []byte {
	t.Helper()
	x, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), name))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	return x
}

// TestParseBuildOverMatchesParseBuild: An undo step parsed over the open build is the step parsed alone.
func TestParseBuildOverMatchesParseBuild(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(corpusDirForLoadedData(t), "*.xml"))
	if err != nil || len(files) < 1000 {
		t.Fatalf("REFUSING: %d corpus files (%v)", len(files), err)
	}
	picked := []string{"0001-necromancer-unknown.xml"}
	for i := 7; i < len(files); i += 211 {
		picked = append(picked, filepath.Base(files[i]))
	}
	picked = corpusSample(picked, 2)
	edits := []struct {
		name string
		edit func(b *pob.PathOfBuilding)
	}{
		{"none", func(*pob.PathOfBuilding) {}},
		{"level", func(b *pob.PathOfBuilding) { b.Build.Level = 1 + b.Build.Level%100 }},
		{"main group", func(b *pob.PathOfBuilding) { b.SetMainSocketGroup(b.Build.MainSocketGroup + 1) }},
		{"notes", func(b *pob.PathOfBuilding) { b.SetNotes(b.Notes + " edited") }},
		{"config", func(b *pob.PathOfBuilding) {
			yes := true
			b.SetConfigOption(pob.Input{Name: "conditionFortify", Boolean: &yes})
		}},
		{"tree", func(b *pob.PathOfBuilding) { b.CopySpec(1, "copy") }},
		{"skills", func(b *pob.PathOfBuilding) {
			if len(b.Skills.SkillSets) > 0 && len(b.Skills.SkillSets[0].Skills) > 0 {
				b.SetSkillGroupName(0, 0, "renamed")
			}
		}},
		{"items", func(b *pob.PathOfBuilding) {
			if len(b.Items.Items) > 0 {
				b.Items.Items[0].Text += "edited"
			}
		}},
	}
	differing := 0
	for _, name := range picked {
		x := parseCorpus(t, name)
		for _, e := range edits {
			live, err := ParseBuild(x)
			if err != nil {
				t.Fatal(err)
			}
			liveXML, err := SerializeBuild(live)
			if err != nil {
				t.Fatal(err)
			}
			edited, err := ParseBuild(x)
			if err != nil {
				t.Fatal(err)
			}
			e.edit(edited)
			target, err := SerializeBuild(edited)
			if err != nil {
				t.Fatal(err)
			}
			if e.name != "none" && string(target) != string(liveXML) {
				differing++
			}
			want, err := ParseBuild(target)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ParseBuildOver(live, string(liveXML), string(target))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s, edit %s: parsed over the open build differs from parsed alone", name, e.name)
			}
		}
	}
	if differing < len(picked)*(len(edits)-3) {
		t.Fatalf("vacuity: only %d edits changed the saved build", differing)
	}

	live, err := ParseBuild(parseCorpus(t, "0001-necromancer-unknown.xml"))
	if err != nil {
		t.Fatal(err)
	}
	liveXML, _ := SerializeBuild(live)
	other := parseCorpus(t, "0280-elementalist-fireball.xml")
	want, _ := ParseBuild(other)
	got, err := ParseBuildOver(live, string(liveXML), string(other))
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("another build parsed over 0001 differs from parsed alone (%v)", err)
	}
}

// TestParseBuildOverKeepsUnchangedSections: The parse over the open build takes an unchanged Items or Tree section from it.
func TestParseBuildOverKeepsUnchangedSections(t *testing.T) {
	live, err := ParseBuild(parseCorpus(t, "0001-necromancer-unknown.xml"))
	if err != nil {
		t.Fatal(err)
	}
	liveXML, _ := SerializeBuild(live)
	edited := *live
	edited.Build.Level--
	target, _ := SerializeBuild(&edited)
	got, err := ParseBuildOver(live, string(liveXML), string(target))
	if err != nil {
		t.Fatal(err)
	}
	if &got.Items.Items[0] != &live.Items.Items[0] || &got.Tree.Specs[0] != &live.Tree.Specs[0] {
		t.Fatal("an unchanged section was parsed again")
	}
	if &got.Skills.SkillSets[0] == &live.Skills.SkillSets[0] {
		t.Fatal("Skills, which a calculation writes into, was taken from the open build")
	}
	if got.Build.Level != live.Build.Level-1 {
		t.Fatalf("level %d, want %d", got.Build.Level, live.Build.Level-1)
	}
}
