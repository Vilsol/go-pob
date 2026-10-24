package pob2

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// roundTrip parses, saves and re-parses one build: the saved bytes must equal the input and both parses must agree.
func roundTrip(raw []byte) error {
	first, err := ParseBuild(raw)
	if err != nil {
		return err
	}
	saved, err := SerializeBuild(first)
	if err != nil {
		return err
	}
	second, err := ParseBuild(saved)
	if err != nil {
		return fmt.Errorf("re-parse: %w", err)
	}
	if !sameValue(reflect.ValueOf(first), reflect.ValueOf(second)) {
		return fmt.Errorf("parse -> save -> parse changed the structure")
	}
	if !bytes.Equal(saved, raw) {
		return fmt.Errorf("save differs from the input at byte %d", firstDiff(saved, raw))
	}
	return nil
}

// sameValue is reflect.DeepEqual with a NaN equal to a NaN of the same sign: PoB saves "nan" for a 0/0 stat.
func sameValue(a, b reflect.Value) bool {
	if a.Kind() != b.Kind() {
		return false
	}
	switch a.Kind() {
	case reflect.Float64:
		x, y := a.Float(), b.Float()
		if math.IsNaN(x) || math.IsNaN(y) {
			return math.IsNaN(x) && math.IsNaN(y) && math.Signbit(x) == math.Signbit(y)
		}
		return x == y
	case reflect.Pointer:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() == b.IsNil()
		}
		return sameValue(a.Elem(), b.Elem())
	case reflect.Struct:
		for i := range a.NumField() {
			if !sameValue(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Slice:
		if a.Len() != b.Len() || a.IsNil() != b.IsNil() {
			return false
		}
		for i := range a.Len() {
			if !sameValue(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if a.Len() != b.Len() || a.IsNil() != b.IsNil() {
			return false
		}
		for _, k := range a.MapKeys() {
			if !b.MapIndex(k).IsValid() || !sameValue(a.MapIndex(k), b.MapIndex(k)) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a.Interface(), b.Interface())
}

func firstDiff(a, b []byte) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

func TestFixtureRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("../builds/testdata/poe2_minimal.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := roundTrip(raw); err != nil {
		t.Fatal(err)
	}
	b, err := ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	spec := b.Tree.Specs[0]
	if *spec.TreeVersion != "0_5" || *spec.AscendancyInternalID != "Druid1" || spec.WeaponSet2 == nil ||
		!reflect.DeepEqual(NodeIDs(spec.WeaponSet2.Nodes), []int{1433, 2244}) {
		t.Fatalf("spec read wrong: %+v", spec)
	}
	set := b.Items.ItemSets[0]
	if len(set.RuneSlots) != 2 || set.RuneSlots[1].SlotName != "Body Armour Rune #2" || !set.UseSecondWeaponSet.IsNil {
		t.Fatalf("item set read wrong: %+v", set)
	}
	gem := b.Skills.SkillSets[0].Skills[0].Gems[1]
	if !gem.Corrupted.IsNil || gem.CorruptLevel.IsNil || gem.Level.V != 20 || *gem.SkillMinion != "LivingLightning" {
		t.Fatalf("gem read wrong: %+v", gem)
	}
	if !strings.HasPrefix(b.Items.Items[0].Raw, "Rarity: RARE\n") || b.Items.Items[0].ModRanges[1].Range != 0.25 {
		t.Fatalf("item read wrong: %+v", b.Items.Items[0])
	}
	if b.Notes.Text != "line one\n\tline two & <three>" || b.Build.MainSocketGroup != 2 {
		t.Fatalf("notes/build read wrong: %q %+v", b.Notes.Text, b.Build)
	}
	if b.Build.ExtraAttrib["futureAttr"] != "kept" || len(b.Tree.Specs[0].ExtraKids) != 1 {
		t.Fatalf("unknown attribute or element dropped: %+v", b.Build.Unparsed)
	}
}

// An unknown element sitting between two known ones is written back where it was.
func TestUnknownKeepsItsPlace(t *testing.T) {
	in := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<PathOfBuilding2>\n\t<Build level=\"1\">\n\t\t<PlayerStat stat=\"A\" value=\"1\"/>\n" +
		"\t\t<Mystery x=\"1\">\n\t\t\tbody\n\t\t</Mystery>\n\t\t<PlayerStat stat=\"B\" value=\"2\"/>\n\t</Build>\n</PathOfBuilding2>\n"
	b, err := ParseBuild([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	out, err := SerializeBuild(b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "<PlayerStat stat=\"A\" value=\"1\"/>\n\t\t<Mystery x=\"1\">\n\t\t\tbody\n\t\t</Mystery>\n\t\t<PlayerStat stat=\"B\"") {
		t.Fatalf("unknown element moved:\n%s", out)
	}
}

func TestRefusesAPoE1Root(t *testing.T) {
	_, err := ParseBuild([]byte(`<?xml version="1.0" encoding="UTF-8"?><PathOfBuilding><Build/></PathOfBuilding>`))
	if err == nil || !strings.Contains(err.Error(), "'PathOfBuilding2' root element missing") {
		t.Fatalf("want the root refusal, got %v", err)
	}
}

func TestComposeMatchesLua(t *testing.T) {
	root := &Node{Elem: "R", Attrib: map[string]string{"b": `x"<'&>`, "a": "1\n2"}, Kids: []*Node{
		{Text: ""}, {Elem: "E", Attrib: map[string]string{}},
	}}
	got, err := ComposeXML(root)
	if err != nil {
		t.Fatal(err)
	}
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<R a=\"1\n2\" b=\"x&quot;&lt;&apos;&amp;&gt;\">\n\t\n\t<E/>\n</R>\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func collectUnparsed(v reflect.Value, path string, into map[string]int) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			collectUnparsed(v.Elem(), path, into)
		}
	case reflect.Slice:
		for i := range v.Len() {
			collectUnparsed(v.Index(i), path, into)
		}
	case reflect.Struct:
		if u, ok := v.Interface().(Unparsed); ok {
			for k := range u.ExtraAttrib {
				into[path+"@"+k]++
			}
			for _, kid := range u.ExtraKids {
				into[path+"/"+kid.Elem]++
			}
			return
		}
		for i := range v.NumField() {
			name := v.Type().Field(i).Name
			if tag := v.Type().Field(i).Tag.Get("pob"); tag != "" && tag[0] != ',' {
				name = strings.Split(tag, ",")[0]
			}
			collectUnparsed(v.Field(i), path+"/"+name, into)
		}
	}
}

// poe2CorpusDirs is the skills-on corpus and, beside its target, the skills-off one.
func poe2CorpusDirs(t *testing.T) []string {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus at testdata/corpus-poe2-0.5")
	}
	on, err := filepath.EvalSymlinks("../testdata/corpus-poe2-0.5")
	if err != nil {
		t.Skipf("no PoE2 corpus: %v", err)
	}
	dirs := []string{on}
	if off := filepath.Join(filepath.Dir(on), "corpus-skillsoff"); dirExists(off) {
		dirs = append(dirs, off)
	}
	return dirs
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func TestCorpusPoE2RoundTrip(t *testing.T) {
	step := 1
	if os.Getenv("POE2_ROUNDTRIP_ALL") != "true" {
		step = 20
	}
	for _, dir := range poe2CorpusDirs(t) {
		files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
		if err != nil || len(files) < 200 {
			t.Fatalf("REFUSING: %d PoE2 corpus files in %s (%v)", len(files), dir, err)
		}
		sort.Strings(files)
		ok, failed, unnamed := 0, 0, map[string]int{}
		for i := 0; i < len(files); i += step {
			raw, err := os.ReadFile(files[i])
			if err != nil {
				t.Fatal(err)
			}
			if b, err := ParseBuild(raw); err == nil {
				collectUnparsed(reflect.ValueOf(b), "PathOfBuilding2", unnamed)
			}
			if err := roundTrip(raw); err != nil {
				failed++
				if failed <= 5 {
					t.Errorf("%s: %v", filepath.Base(files[i]), err)
				}
				continue
			}
			ok++
		}
		t.Logf("%s: %d / %d round-trip byte-identical", filepath.Base(dir), ok, ok+failed)
		if len(unnamed) > 0 {
			t.Errorf("%s: attributes or elements the types do not name: %v", filepath.Base(dir), unnamed)
		}
		if ok < 200 {
			t.Fatalf("REFUSING: only %d builds round-tripped in %s", ok, dir)
		}
	}
}
