package calculator2

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// itemCraftRow is one row of testdata/item_craft_pob.jsonl, which testdata/item_craft_oracle.lua writes from
// PoB-PoE2's Craft Item popup: a crafted item, or one data.itemBaseLists entry.
type itemCraftRow struct {
	Rarity int      `json:"rarity"`
	Title  string   `json:"title"`
	Base   string   `json:"base"`
	Raw    string   `json:"raw"`
	Type   string   `json:"type"`
	Bases  []string `json:"bases"`
}

func readItemCraftRows(t *testing.T) []itemCraftRow {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "item_craft_pob.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rows []itemCraftRow
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var r itemCraftRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	return rows
}

func TestCraftItemMatchesPoB(t *testing.T) {
	rows := readItemCraftRows(t)
	var types []string
	crafted := 0
	for _, r := range rows {
		if r.Type != "" {
			types = append(types, r.Type)
			list := ItemBaseLists()[r.Type]
			var got []string
			for _, e := range list {
				got = append(got, e.Name, e.Label)
			}
			if len(got) != len(r.Bases) {
				t.Errorf("%s: %d bases, PoB-PoE2 lists %d", r.Type, len(got)/2, len(r.Bases)/2)
				continue
			}
			for i := range got {
				if got[i] != r.Bases[i] {
					t.Errorf("%s: base %d is %q, PoB-PoE2's is %q", r.Type, i/2+1, got[i], r.Bases[i])
					break
				}
			}
			continue
		}
		crafted++
		item, ok := CraftItem(r.Rarity, r.Title, r.Base, ItemModSource(WithParser(ParseMod)))
		if !ok {
			t.Errorf("%s: not crafted", r.Base)
			continue
		}
		if want := r.Raw; item.BuildRaw() != want {
			t.Errorf("%s:\n got %q\nwant %q", r.Base, item.BuildRaw(), want)
		}
	}
	if crafted == 0 {
		t.Fatal("no crafted rows")
	}
	got := ItemBaseTypeList()
	if len(got) != len(types) {
		t.Fatalf("%d base types, PoB-PoE2 lists %d", len(got), len(types))
	}
	for i := range got {
		if got[i] != types[i] {
			t.Errorf("base type %d is %q, PoB-PoE2's is %q", i+1, got[i], types[i])
		}
	}
}
