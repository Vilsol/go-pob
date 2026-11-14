package exposition

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// The values are what the pinned PoB-PoE2 calculates and saves for a new build (HeadlessWrapper's newBuild with
// main.defaultCharLevel and main.defaultGemQuality set, two frames, then SaveDB).
func TestNewBuildMatchesPoB(t *testing.T) {
	for _, c := range []struct {
		level, quality int
		auto           bool
		want           map[string]float64
	}{
		{1, 0, true, map[string]float64{"Life": 65, "Mana": 50, "Spirit": 100, "Str": 7, "Dex": 15, "Int": 7, "Evasion": 7, "TotalDPS": 0.36544921875}},
		{90, 20, false, map[string]float64{"Life": 1187, "Mana": 424, "Spirit": 100, "Dex": 15, "Evasion": 7, "TotalDPS": 1.2012}},
	} {
		calc, err := NewBuild(c.level, c.quality)
		if err != nil {
			t.Fatal(err)
		}
		out, err := calc.Calculate()
		if err != nil {
			t.Fatal(err)
		}
		for stat, want := range c.want {
			assertClose(t, "new build", stat, out.Output[stat], want)
		}
		if out.ClassName != "Ranger" || out.AscendClassName != "None" || out.Level != c.level {
			t.Errorf("class %q ascendancy %q level %d, want Ranger None %d", out.ClassName, out.AscendClassName, out.Level, c.level)
		}
		xml, err := calc.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		saved, err := pob2.ParseBuild([]byte(xml))
		if err != nil {
			t.Fatal(err)
		}
		b := saved.Build
		if b.TargetVersion != "0_1" || b.ViewMode != "TREE" || b.CharacterLevelAutoMode != c.auto || b.MainSocketGroup != 1 {
			t.Errorf("Build %+v", b)
		}
		if q := saved.Skills.DefaultGemQuality; q.IsNil || q.V != c.quality {
			t.Errorf("defaultGemQuality %+v, want %d", q, c.quality)
		}
		if len(saved.Tree.Specs) != 1 || *saved.Tree.Specs[0].ClassID != 2 || *saved.Tree.Specs[0].Nodes != "50459" || saved.Tree.Specs[0].Title != nil {
			t.Errorf("spec %+v", saved.Tree.Specs)
		}
		if !strings.Contains(xml, `<ItemSet id="1"`) || !strings.Contains(xml, `<SkillSet id="1"`) || !strings.Contains(xml, `<ConfigSet id="1"`) {
			t.Errorf("sets missing:\n%s", xml)
		}
	}
}

// Build.lua:1185-1186: Save writes the spec's class and ascendancy, not the ones the file was loaded with.
func TestSaveWritesSpecClass(t *testing.T) {
	calc, err := NewBuild(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := TreeClassList()
	if err != nil {
		t.Fatal(err)
	}
	witch := slices.IndexFunc(classes, func(r TreeClassRow) bool { return r.Label == "Witch" })
	if witch < 0 {
		t.Fatal("no Witch in the tree")
	}
	for _, step := range []struct {
		change            func() error
		class, ascendancy string
	}{
		{func() error { return nil }, "Ranger", "None"},
		{func() error { _, err := calc.ChangeTreeClass(classes[witch].ClassID, TreeClassContinue); return err }, "Witch", "None"},
		{func() error { return calc.ChangeTreeAscendancy(1) }, "Witch", "Infernalist"},
	} {
		if _, err := calc.Calculate(); err != nil {
			t.Fatal(err)
		}
		if err := step.change(); err != nil {
			t.Fatal(err)
		}
		if _, err := calc.Calculate(); err != nil {
			t.Fatal(err)
		}
		for _, save := range []func() (string, error){calc.Serialize, func() (string, error) { return calc.SerializeWithStats(nil, nil, nil) }} {
			xml, err := save()
			if err != nil {
				t.Fatal(err)
			}
			saved, err := pob2.ParseBuild([]byte(xml))
			if err != nil {
				t.Fatal(err)
			}
			if saved.Build.ClassName != step.class || saved.Build.AscendClassName != step.ascendancy {
				t.Errorf("saved %q %q, want %q %q", saved.Build.ClassName, saved.Build.AscendClassName, step.class, step.ascendancy)
			}
		}
	}
}
