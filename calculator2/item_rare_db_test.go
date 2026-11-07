package calculator2

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// itemRareDBRow is one row of testdata/item_rare_db_pob.jsonl.gz, which testdata/item_rare_db_oracle.lua writes from
// PoB-PoE2's rare database: a crafted item, or the list's Type dropdown.
type itemRareDBRow struct {
	Name        string   `json:"name"`
	Raw         string   `json:"raw"`
	Type        string   `json:"type"`
	PrimarySlot string   `json:"primarySlot"`
	Level       float64  `json:"level"`
	Types       []string `json:"types"`
}

// rareDBMatchedFloor is the rare items equal to PoB-PoE2's.
const rareDBMatchedFloor = 118

// TestRareDBMatchesPoB: the rare templates craft to PoB-PoE2's items, and the list's types are PoB-PoE2's.
func TestRareDBMatchesPoB(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "item_rare_db_pob.jsonl.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	db := ItemDB(ItemDBRare)
	sc := bufio.NewScanner(zr)
	sc.Buffer(nil, 1<<20)
	same, total := 0, 0
	for sc.Scan() {
		var r itemRareDBRow
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if r.Types != nil {
			if _, types := ItemDBLists(ItemDBRare); !slices.Equal(types, r.Types) {
				t.Errorf("types %q, PoB-PoE2's %q", types, r.Types)
			}
			continue
		}
		total++
		item := db[r.Name]
		if item == nil {
			t.Errorf("%s is not in the rare database", r.Name)
			continue
		}
		level := 0.0
		if item.Requirements.Level != nil {
			level = *item.Requirements.Level
		}
		if got := item.BuildRaw(); got != r.Raw {
			t.Logf("%s: item text %s", r.Name, firstDiffLine(got, r.Raw))
		} else if item.Type != r.Type || item.GetPrimarySlot() != r.PrimarySlot || level != r.Level {
			t.Errorf("%s: %s %s %v, PoB-PoE2's %s %s %v", r.Name, item.Type, item.GetPrimarySlot(), level, r.Type, r.PrimarySlot, r.Level)
		} else {
			same++
		}
	}
	t.Logf("%d / %d rare items equal PoB-PoE2's", same, total)
	if total == 0 || len(db) != total {
		t.Fatalf("%d items, PoB-PoE2's %d", len(db), total)
	}
	if same < rareDBMatchedFloor {
		t.Fatalf("%d rare items equal PoB-PoE2's, below the floor %d", same, rareDBMatchedFloor)
	}
}
