package calculator

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func retainedAffix(t *testing.T, raw string) (string, string) {
	t.Helper()
	item := ParseItem(raw)
	st, ok := itemAffixState(raw, item)
	if !ok {
		t.Fatal("REFUSING: not a crafted item")
	}
	ids := slices.Sorted(maps.Keys(st.affixes))
	for _, id := range ids {
		mod := st.affixes[id]
		if mod.Type == "Prefix" && mod.Affix != "Subterranean" && modSpawnWeightTags(item, mod, map[string]bool{}, nil, nil) == 0 {
			if strings.Contains(raw, "Prefix: None") {
				return id, strings.Replace(raw, "Prefix: None", "Prefix: {range:0.5}"+id, 1)
			}
			return id, strings.Replace(raw, "Crafted: true\n", "Crafted: true\nPrefix: {range:0.5}"+id+"\n", 1)
		}
	}
	return "", raw
}

// TestAffixRetainedLabel: ItemsTab.lua:2297-2300 (#10222): a selected prefix the item can no longer roll is listed, labelled [Retained] only on a cluster jewel.
func TestAffixRetainedLabel(t *testing.T) {
	initItemsTabData(t)
	cluster, _ := mustEdit(t, "Rarity: RARE\nNew Item\nLarge Cluster Jewel\nCrafted: true\nImplicits: 0", ItemEdit{Kind: "cluster", Text: "affliction_axe_and_sword_damage", Value: 8})
	circlet, _ := CraftItemText(2, "", "Hubris Circlet")
	for _, c := range []struct {
		raw      string
		retained bool
	}{{cluster, true}, {circlet, false}} {
		raw := c.raw
		id, withAffix := retainedAffix(t, raw)
		if id == "" {
			t.Fatalf("REFUSING: no prefix outside the base's pool for\n%s", raw)
		}
		ctl, ok := ItemAffixControls(withAffix)
		if !ok || ctl.Controls[0].Sel == 0 {
			t.Fatalf("%s: the retained prefix is not selected\n%s", id, withAffix)
		}
		label := ctl.Controls[0].Series[ctl.Controls[0].Sel].Label
		if strings.HasPrefix(label, "^8[Retained] ") != c.retained {
			t.Errorf("%s on %s: label %q, retained label wanted %v", id, ParseItem(raw).BaseName, label, c.retained)
		}
	}
}
