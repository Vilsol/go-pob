package calculator2

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob2"
)

// itemsEditStep is one row of testdata/items_edit_pob.jsonl, which testdata/items_edit_oracle.lua writes from
// PoB-PoE2 for testdata/items_edit.xml.
type itemsEditStep struct {
	Step               string             `json:"step"`
	ActiveItemSet      int                `json:"activeItemSet"`
	UseSecondWeaponSet bool               `json:"useSecondWeaponSet"`
	Slots              map[string]int     `json:"slots"`
	Output             map[string]float64 `json:"output"`
	XML                string             `json:"xml"`
}

func readItemsEditPastes(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "items_edit_pastes.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n=====\n")
}

// itemsEditSteps are the oracle's steps, through the tab of the last calculation; ring is that calculation's
// AdditionalRingSlot flag.
func itemsEditSteps(pastes []string) map[string]func(tab *ItemsTab, ring bool) error {
	paste := func(i int) func(tab *ItemsTab, ring bool) error {
		return func(tab *ItemsTab, ring bool) error {
			if err := tab.CreateDisplayItemFromRaw(pastes[i-1], true); err != nil {
				return err
			}
			return tab.AddDisplayItem(false, ring)
		}
	}
	return map[string]func(tab *ItemsTab, ring bool) error{
		"paste helmet":      paste(1),
		"paste wand":        paste(2),
		"paste second wand": paste(3),
		"weapon set II":     func(tab *ItemsTab, _ bool) error { tab.SetUseSecondWeaponSet(true); return nil },
		"equip second wand in set II": func(tab *ItemsTab, _ bool) error {
			return tab.SetSlotItem("Weapon 1 Swap", 6)
		},
		"weapon set I": func(tab *ItemsTab, _ bool) error { tab.SetUseSecondWeaponSet(false); return nil },
		"body armour runes": func(tab *ItemsTab, ring bool) error {
			if err := tab.EditItem(1); err != nil {
				return err
			}
			if err := tab.SetDisplayItemSocketCount(2); err != nil {
				return err
			}
			if err := tab.SetDisplayItemRune(1, "Desert Rune"); err != nil {
				return err
			}
			if err := tab.SetDisplayItemRune(2, "Glacial Rune"); err != nil {
				return err
			}
			return tab.AddDisplayItem(false, ring)
		},
		"wand rune": func(tab *ItemsTab, ring bool) error {
			if err := tab.EditItem(5); err != nil {
				return err
			}
			if err := tab.SetDisplayItemRune(1, "Storm Rune"); err != nil {
				return err
			}
			return tab.AddDisplayItem(false, ring)
		},
		"character rune": func(tab *ItemsTab, _ bool) error { return tab.SetCharacterRune("Helmet Rune #1", "Desert Rune") },
		"paste flask":    paste(4),
		"flask active":   func(tab *ItemsTab, _ bool) error { return tab.SetSlotActive("Flask 1", true) },
		"new item set":   func(tab *ItemsTab, _ bool) error { tab.NewItemSetService("Second"); return nil },
		"equip body armour in new set": func(tab *ItemsTab, _ bool) error {
			return tab.SetSlotItem("Body Armour", 1)
		},
		"copy item set": func(tab *ItemsTab, _ bool) error {
			_, err := tab.CopyItemSetService(tab.ActiveItemSetID, "Third")
			return err
		},
		"rename item set":    func(tab *ItemsTab, _ bool) error { return tab.RenameItemSet(tab.ActiveItemSetID, "Renamed") },
		"item set 1":         func(tab *ItemsTab, _ bool) error { tab.SetActiveItemSet(1); return nil },
		"delete item set":    func(tab *ItemsTab, _ bool) error { return tab.DeleteItemSet(4) },
		"item set 2":         func(tab *ItemsTab, _ bool) error { tab.SetActiveItemSet(2); return nil },
		"back to item set 1": func(tab *ItemsTab, _ bool) error { tab.SetActiveItemSet(1); return nil },
		"delete ring":        func(tab *ItemsTab, _ bool) error { return tab.DeleteItem(2) },
		"unequip helmet":     func(tab *ItemsTab, _ bool) error { return tab.SetSlotItem("Helmet", 0) },
		"edit wand text": func(tab *ItemsTab, ring bool) error {
			if err := tab.EditItem(5); err != nil {
				return err
			}
			return tab.EditDisplayItemText(pastes[4], "RARE", true, ring)
		},
	}
}

func readItemsEditOracle(t *testing.T) []itemsEditStep {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "items_edit_pob.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []itemsEditStep
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var row itemsEditStep
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	return rows
}

// itemsXML is the build's <Items> as the build file writes it, without what a load keeps for a round trip.
func itemsXML(t *testing.T, b *pob2.PathOfBuilding2) string {
	t.Helper()
	out := &pob2.PathOfBuilding2{}
	items := b.Items
	items.Unparsed = pob2.Unparsed{}
	items.Items = append([]pob2.Item(nil), items.Items...)
	for i := range items.Items {
		items.Items[i].Unparsed = pob2.Unparsed{}
		items.Items[i].Raw = strings.TrimSpace(items.Items[i].Raw)
		for j := range items.Items[i].ModRanges {
			items.Items[i].ModRanges[j].Unparsed = pob2.Unparsed{}
		}
	}
	items.ItemSets = append([]pob2.ItemSet(nil), items.ItemSets...)
	for i := range items.ItemSets {
		set := &items.ItemSets[i]
		set.Unparsed = pob2.Unparsed{}
		set.Slots = append([]pob2.Slot(nil), set.Slots...)
		for j := range set.Slots {
			set.Slots[j].Unparsed = pob2.Unparsed{}
		}
		set.RuneSlots = append([]pob2.RuneSlot(nil), set.RuneSlots...)
		for j := range set.RuneSlots {
			set.RuneSlots[j].Unparsed = pob2.Unparsed{}
		}
	}
	if items.TradeSearchWeights != nil {
		w := *items.TradeSearchWeights
		w.Unparsed = pob2.Unparsed{}
		items.TradeSearchWeights = &w
	}
	out.Items = items
	text, err := pob2.SerializeBuild(out)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// Every edit recalculates; the sidebar's numbers, the active item set, what each slot holds and the saved items are
// PoB-PoE2's for the same edits through its Items tab controls.
func TestItemsEditMatchesPoB(t *testing.T) {
	steps := itemsEditSteps(readItemsEditPastes(t))
	rows := readItemsEditOracle(t)
	raw, err := os.ReadFile(filepath.Join("testdata", "items_edit.xml"))
	if err != nil {
		t.Fatal(err)
	}
	build, err := pob2.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := loadTestTree(t)
	var tab *ItemsTab
	var env *Environment
	for _, row := range rows {
		if row.Step == "saved" {
			saved, err := pob2.ParseBuild([]byte(row.XML))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := itemsXML(t, build), itemsXML(t, saved); got != want {
				t.Errorf("saved items:\n%s\nPoB-PoE2:\n%s", got, want)
			}
			continue
		}
		if row.Step != "load" {
			edit := steps[row.Step]
			if edit == nil {
				t.Fatalf("no edit for step %q", row.Step)
			}
			if err := edit(tab, env.ModDB.Flag(nil, "AdditionalRingSlot")); err != nil {
				t.Fatalf("%s: %v", row.Step, err)
			}
			tab.Save(build)
		}
		next := NewItemsTab(build, tree, nil)
		if tab != nil {
			next.DisplayItem = tab.DisplayItem
		}
		tab = next
		env, err = BuildOutput(build, Options{Tree: tree, Items: tab})
		if err != nil {
			t.Fatalf("%s: %v", row.Step, err)
		}
		if tab.ActiveItemSetID != row.ActiveItemSet || tab.UseSecondWeaponSet != row.UseSecondWeaponSet {
			t.Errorf("%s: item set %d weapon set II %v, PoB-PoE2 %d %v", row.Step, tab.ActiveItemSetID, tab.UseSecondWeaponSet, row.ActiveItemSet, row.UseSecondWeaponSet)
		}
		slots := map[string]int{}
		for _, slot := range tab.OrderedSlots {
			if slot.SelItemID != 0 {
				slots[slot.SlotName] = slot.SelItemID
			}
		}
		if got, want := jsonString(t, slots), jsonString(t, row.Slots); got != want {
			t.Errorf("%s: slots %s, PoB-PoE2 %s", row.Step, got, want)
		}
		for stat, want := range row.Output {
			if got := env.Player.Output[stat]; math.Abs(got-want) > 1e-9*math.Max(1, math.Abs(want)) {
				t.Errorf("%s: %s %v, PoB-PoE2 %v", row.Step, stat, got, want)
			}
		}
	}
}

func jsonString(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The committed oracle rows are what the oracles write from the pinned PoB-PoE2; CORPUS=true runs them, as they
// start PoB-PoE2.
func TestItemsEditOracleIsCurrent(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true runs PoB-PoE2")
	}
	pob, luajit := pobPoE2Probe(t)
	abs := func(name string) string {
		p, err := filepath.Abs(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, c := range []struct {
		probe, want string
		args        []string
	}{
		{"items_edit_oracle.lua", "items_edit_pob.jsonl", []string{abs("items_edit.xml"), abs("items_edit_pastes.txt")}},
		{"item_build_raw_oracle.lua", "item_build_raw_pob.jsonl", []string{abs("items_edit.xml")}},
	} {
		out := filepath.Join(t.TempDir(), "out.jsonl")
		args := append([]string{"-R", luajit, abs(c.probe), out}, c.args...)
		cmd := exec.CommandContext(context.Background(), "setarch", args...)
		cmd.Dir = filepath.Join(pob, "src")
		cmd.Env = append(os.Environ(),
			"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
			"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s failed: %v\n%s", c.probe, err, stderr.String())
		}
		got, err := os.ReadFile(out) //nolint:gosec // the test's own temp dir
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(abs(c.want))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("testdata/%s is stale; PoB-PoE2 now writes:\n%s", c.want, got)
		}
	}
}
