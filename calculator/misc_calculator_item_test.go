package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

func activeSetSlot(build *pob.PathOfBuilding, slotName string) (pob.Slot, string) {
	set := build.Items.ItemSets[activeItemSetIndex(build.Items)]
	for _, s := range set.Slots {
		if s.Name != slotName || s.ItemID == 0 {
			continue
		}
		for _, it := range build.Items.Items {
			if it.ID == s.ItemID {
				return s, it.Text
			}
		}
	}
	return pob.Slot{}, ""
}

func equipText(build *pob.PathOfBuilding, slotName, text string, empty ...string) {
	id := 1
	for _, it := range build.Items.Items {
		id = max(id, it.ID+1)
	}
	build.Items.Items = append(build.Items.Items, pob.Item{ID: id, Text: text})
	build.SetSlotItem(slotName, id)
	for _, c := range empty {
		build.SetSlotItem(c, 0)
	}
}

type itemWhatIfCase struct {
	name     string
	override CalcOverride
	mutate   func(*pob.PathOfBuilding)
}

// TestMiscCalculatorItemEqualsRecompute: an item what-if (override repSlotName/repItem.
func TestMiscCalculatorItemEqualsRecompute(t *testing.T) {
	socketMatchesFollowOverride = true
	t.Cleanup(func() { socketMatchesFollowOverride = false })
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	donors := map[string]string{}
	for _, d := range []struct{ file, slot, key string }{
		{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Weapon 1 Swap", "staff"},
		{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Helmet", "helmet"},
		{"0280-elementalist-fireball.xml", "Helmet", "helmet2"},
		{"0339-reliquarian-shield-crush-of-the-chieftain-2.xml", "Body Armour", "body"},
		{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Body Armour", "body2"},
		{"0008-necromancer-vaal-summon-skeletons.xml", "Ring 1", "ring"},
		{"0280-elementalist-fireball.xml", "Ring 1", "ring2"},
		{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "Flask 5", "flask"},
	} {
		_, text := activeSetSlot(loadCorpusBuild(t, d.file), d.slot)
		if text == "" {
			t.Fatalf("REFUSING: %s has no item in %s", d.file, d.slot)
		}
		donors[d.key] = text
	}

	for _, tc := range []struct {
		file, helmet, body, ring string
	}{
		{"0339-reliquarian-shield-crush-of-the-chieftain-2.xml", "helmet", "body2", "ring"},
		{"0008-necromancer-vaal-summon-skeletons.xml", "helmet", "body", "ring2"},
		{"0280-elementalist-fireball.xml", "helmet", "body", "ring"},
		{"13276-ladder-cuba_5717.cubamrc.xml", "helmet", "body", "ring"},
		{"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml", "helmet2", "body", "ring"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			build := loadCorpusBuild(t, tc.file)
			misc := GetMiscCalculator(build)
			before := len(build.Items.Items)

			rep := func(name, slot, donor string, empty ...string) itemWhatIfCase {
				text := donors[donor]
				return itemWhatIfCase{
					name:     name,
					override: CalcOverride{RepSlotName: slot, RepItem: ParseItem(text)},
					mutate:   func(b *pob.PathOfBuilding) { equipText(b, slot, text, empty...) },
				}
			}
			cases := []itemWhatIfCase{
				rep("helmet", "Helmet", tc.helmet),
				rep("body armour", "Body Armour", tc.body),
				rep("ring 1", "Ring 1", tc.ring),
				{
					name:     "unequip gloves",
					override: CalcOverride{RepSlotName: "Gloves"},
					mutate:   func(b *pob.PathOfBuilding) { b.SetSlotItem("Gloves", 0) },
				},
			}
			if s, _ := activeSetSlot(build, "Weapon 2"); s.ItemID != 0 {
				cases = append(cases, rep("two-handed weapon", "Weapon 1", "staff", "Weapon 2"))
			}
			for _, slot := range []string{"Flask 1", "Flask 2"} {
				s, text := activeSetSlot(build, slot)
				if s.ItemID == 0 {
					continue
				}
				item := ParseItem(text)
				c := itemWhatIfCase{
					name:     "toggle " + slot,
					override: CalcOverride{ToggleFlask: item},
					mutate:   func(b *pob.PathOfBuilding) { b.SetFlaskActive(slot, !s.Active) },
				}
				if item.Type == "Tincture" {
					c.override = CalcOverride{ToggleTincture: item}
				}
				cases = append(cases, c)
			}

			for _, c := range cases {
				whatIf := misc.Calc(&c.override)
				if len(build.Items.Items) != before {
					t.Fatalf("%s: the what-if mutated the build's items", c.name)
				}

				mutated := loadCorpusBuild(t, tc.file)
				c.mutate(mutated)
				recompute := (&Calculator{PoB: mutated}).BuildOutput(OutputModeMain)

				base := misc.Base()
				if c.override.ToggleFlask != nil || c.override.ToggleTincture != nil {
					if len(whatIf.Flasks) == len(base.Flasks) && len(whatIf.Tinctures) == len(base.Tinctures) {
						t.Errorf("%s: the active flask and tincture counts did not move — the override did not land", c.name)
					}
				} else if n := changedOutputs(whatIf.Player.Output, base.Player.Output); n == 0 {
					t.Errorf("%s: the what-if changed no output — the override did not land", c.name)
				}
				if where, ok := outputsAgree(whatIf.Player.Output, recompute.Player.Output); !ok {
					t.Errorf("%s: player output differs from a recompute: %s", c.name, where)
				}
				if (whatIf.Minion == nil) != (recompute.Minion == nil) {
					t.Errorf("%s: minion presence differs from a recompute", c.name)
				} else if whatIf.Minion != nil {
					if where, ok := outputsAgree(whatIf.Minion.Output, recompute.Minion.Output); !ok {
						t.Errorf("%s: minion output differs from a recompute: %s", c.name, where)
					}
				}
			}

			if where, ok := outputsAgree(misc.Base().Player.Output, (&Calculator{PoB: loadCorpusBuild(t, tc.file)}).BuildOutput(OutputModeMain).Player.Output); !ok {
				t.Errorf("calcBase moved after the item what-ifs: %s", where)
			}
		})
	}
}

// TestMiscCalculatorToggleUnequippedFlask: A flask that is not equipped still toggles on (ItemsTab.lua:5003: toggleFlask = the hovered item).
func TestMiscCalculatorToggleUnequippedFlask(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	_, text := activeSetSlot(loadCorpusBuild(t, "4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml"), "Flask 5")
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	env := misc.Calc(&CalcOverride{ToggleFlask: ParseItem(text)})
	if len(env.Flasks) != len(misc.Base().Flasks)+1 {
		t.Fatalf("flasks: base %d, what-if %d; want one more", len(misc.Base().Flasks), len(env.Flasks))
	}
	if changedOutputs(env.Player.Output, misc.Base().Player.Output) == 0 {
		t.Error("toggling Stormblood on changed no output")
	}
}

// TestMiscCalculatorKeepsSocketMatches: by default a what-if grades gem colour matches against the items the build has now.
func TestMiscCalculatorKeepsSocketMatches(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}
	_, text := activeSetSlot(loadCorpusBuild(t, "0339-reliquarian-shield-crush-of-the-chieftain-2.xml"), "Body Armour")
	file := "0008-necromancer-vaal-summon-skeletons.xml"
	whatIf := GetMiscCalculator(loadCorpusBuild(t, file)).Calc(&CalcOverride{RepSlotName: "Body Armour", RepItem: ParseItem(text)})
	mutated := loadCorpusBuild(t, file)
	equipText(mutated, "Body Armour", text)
	recompute := (&Calculator{PoB: mutated}).BuildOutput(OutputModeMain)
	if _, ok := outputsAgree(whatIf.Player.Output, recompute.Player.Output); ok {
		t.Error("the what-if equals a recompute: socket matches followed the replaced item")
	}
}
