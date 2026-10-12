package exposition

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func initItemData(t *testing.T) {
	t.Helper()
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
}

// TestSetActiveItemSetMatchesTheSavedSet: Switching 0339 from its "enmity" set.
func TestSetActiveItemSetMatchesTheSavedSet(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml")
	if build.Items.ActiveItemSet != 2 {
		t.Fatalf("REFUSING: 0339 opens on set %d, want 2", build.Items.ActiveItemSet)
	}
	xml, err := os.ReadFile(filepath.Join("..", "..", "testdata", "corpus-"+raw.LatestVersion, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := builds.ParseBuild([]byte(strings.Replace(string(xml), `activeItemSet="2"`, `activeItemSet="1"`, 1)))
	if err != nil || saved.Items.ActiveItemSet != 1 {
		t.Fatalf("REFUSING: the set-1 copy did not load (%v)", err)
	}

	SetActiveItemSet(build, 1)
	helmet := func(eq []EquippedSlot) int {
		for _, e := range eq {
			if e.SlotName == "Helmet" {
				return e.ItemID
			}
		}
		return 0
	}
	if got := helmet(EquippedItems(build)); got != 16 {
		t.Errorf("Helmet = %d after switching to set 1, want its 16", got)
	}
	got, want := GetMiscCalculator(build).Base().Output, GetMiscCalculator(saved).Base().Output
	for _, k := range []string{"Life", "EnergyShield", "Armour", "FireResist", "TotalEHP", "CombinedDPS"} {
		if got[k] != want[k] {
			t.Errorf("%s: %v after switching, %v saved on set 1", k, got[k], want[k])
		}
	}
	set2 := GetMiscCalculator(corpusBuild(t, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml")).Base().Output
	if got["Life"] == set2["Life"] && got["Armour"] == set2["Armour"] {
		t.Error("vacuity: set 1 and set 2 calculate the same Life and Armour")
	}
}

// TestEquipItemIDAndSlotItemLists: EquipItemID equips an item of the build's list for that slot and refuses one that does not fit.
func TestEquipItemIDAndSlotItemLists(t *testing.T) {
	initItemData(t)
	build := corpusBuild(t, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml")
	lists := map[string][]int{}
	for _, s := range SlotItemLists(build) {
		lists[s.SlotName] = s.ItemIDs
	}
	if !slices.Contains(lists["Helmet"], 16) || !slices.Contains(lists["Helmet"], 61) || slices.Contains(lists["Helmet"], 58) {
		t.Fatalf("Helmet's list %v: want 16 and 61, not the ring 58", lists["Helmet"])
	}
	if _, err := EquipItemID(build, "Helmet", 58); err == nil {
		t.Error("a ring was equipped in Helmet")
	}
	if changed, err := EquipItemID(build, "Helmet", 61); err != nil || changed {
		t.Errorf("choosing the equipped helmet: changed %v, err %v", changed, err)
	}
	if changed, err := EquipItemID(build, "Helmet", 16); err != nil || !changed {
		t.Fatalf("choosing helmet 16: changed %v, err %v", changed, err)
	}
	for _, e := range EquippedItems(build) {
		if e.SlotName == "Helmet" && e.ItemID != 16 {
			t.Errorf("Helmet = %d, want 16", e.ItemID)
		}
	}
}
