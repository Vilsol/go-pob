package datafields

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMissingNamesAssetAndPath(t *testing.T) {
	committed := &List{Assets: map[string]AssetList{
		"raw/Mods":  {Type: "Mod", Keep: []string{"ID"}},
		"raw/Whole": {Type: "W", Keep: []string{"N"}},
	}}
	fresh := &List{Assets: map[string]AssetList{
		"raw/Mods":  {Type: "Mod", Keep: []string{"ID", "Domain"}},
		"raw/Whole": {Type: "W", Keep: []string{"N", "N.x"}},
		"raw/New":   {Type: "New", Keep: []string{"A"}},
	}}
	got := Missing(committed, fresh)
	want := []string{
		"asset raw/New (New) is loaded by go-pob but absent from the list",
		"raw/Mods: go-pob reads Mod.Domain, absent from the list",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestBuildKeepsAWholeTypeAsKeepAll(t *testing.T) {
	res := &Result{
		Structs: map[string]*StructInfo{
			"W":   {Name: "W", Fields: []FieldInfo{{Name: "A"}, {Name: "B"}}},
			"Mod": {Name: "Mod", Fields: []FieldInfo{{Name: "ID"}, {Name: "Domain"}}},
		},
		Reads:  map[FieldKey][]Site{{Type: "Mod", Field: "ID"}: {{Pos: "x.go:1"}}},
		Whole:  map[string][]Whole{"W": {{Why: "msgp", Site: Site{Pos: "y.go:2"}}}},
		Assets: map[string]Asset{"raw/W": {Path: "raw/W", RowType: "W"}, "raw/Mods": {Path: "raw/Mods", RowType: "Mod"}},
	}
	l := Build(res, "m@v", "3.29")
	if l.Format != "go-pob-data-fields/2" {
		t.Errorf("format %q", l.Format)
	}
	if got := l.Assets["raw/W"].Keep; !reflect.DeepEqual(got, []string{"*"}) {
		t.Errorf("whole type keeps %q, want [*]", got)
	}
	if got := l.Assets["raw/Mods"].Keep; !reflect.DeepEqual(got, []string{"ID"}) {
		t.Errorf("Mods keeps %q", got)
	}
	if len(l.Unread["raw/W"]) != 0 || !reflect.DeepEqual(l.Unread["raw/Mods"], []string{"Domain"}) {
		t.Errorf("unread %q", l.Unread)
	}
	if m := Missing(&List{Assets: map[string]AssetList{"raw/W": {Keep: []string{"A"}}, "raw/Mods": {Keep: []string{"*"}}}}, l); !reflect.DeepEqual(m, []string{"raw/W: go-pob reads W.B, absent from the list"}) {
		t.Errorf("missing %q", m)
	}
	if m := Missing(&List{Assets: map[string]AssetList{"raw/W": {Keep: []string{"A", "B"}}, "raw/Mods": {Keep: []string{"ID"}}}}, l); len(m) != 0 {
		t.Errorf("a /1 list naming every field covers a whole type; missing %q", m)
	}
}

func TestLoadReadsBothFormats(t *testing.T) {
	dir := t.TempDir()
	for format, ok := range map[string]bool{"go-pob-data-fields/1": true, "go-pob-data-fields/2": true, "go-pob-data-fields/3": false} {
		p := filepath.Join(dir, "l.json")
		if err := os.WriteFile(p, []byte(`{"format":"`+format+`","assets":{}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(p); (err == nil) != ok {
			t.Errorf("%s: err %v", format, err)
		}
	}
}
