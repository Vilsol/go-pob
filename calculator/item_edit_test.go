package calculator

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func corpusItemText(t *testing.T, file string, id int) string {
	t.Helper()
	for _, it := range loadCorpusBuild(t, file).Items.Items {
		if it.ID == id {
			return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&").Replace(it.Text)
		}
	}
	t.Fatalf("REFUSING: %s has no item %d", file, id)
	return ""
}

func mustEdit(t *testing.T, raw string, e ItemEdit) (string, *Item) {
	t.Helper()
	out, err := EditItem(raw, e)
	if err != nil {
		t.Fatalf("%+v: %v", e, err)
	}
	return out, ParseItem(out)
}

const gameHelmet = `Item Class: Helmets
Rarity: Rare
Doom Visor
Hubris Circlet
--------
Quality: +20% (augmented)
Energy Shield: 250 (augmented)
--------
Sockets: B-B-B-G
--------
Item Level: 84
--------
+80 to maximum Life
+40% to Fire Resistance
+35% to Cold Resistance (crafted)`

// TestEditItemQuality: ItemsTab.lua:637-641: the quality edit writes the item's quality; the game's own text format survives.
func TestEditItemQuality(t *testing.T) {
	initItemsTabData(t)
	out, item := mustEdit(t, gameHelmet, ItemEdit{Kind: "quality", Value: 7})
	if item.Quality == nil || *item.Quality != 7 || item.Name != ParseItem(gameHelmet).Name || len(item.ExplicitModLines) != 3 {
		t.Fatalf("quality %v, name %q, %d explicits:\n%s", item.Quality, item.Name, len(item.ExplicitModLines), out)
	}
	if strings.Count(out, "Quality: ") != 1 {
		t.Fatalf("two quality lines:\n%s", out)
	}
	info, _ := DisplayItem(out)
	if !info.HasQuality || info.Quality != 7 {
		t.Fatalf("display quality %+v", info.Quality)
	}
}

// TestEditItemSockets: ItemsTab.lua:459-503: socket colours, links and an added socket are the Sockets line.
func TestEditItemSockets(t *testing.T) {
	initItemsTabData(t)
	_, item := mustEdit(t, gameHelmet, ItemEdit{Kind: "sockets", Text: "R-G B-B"})
	got := ""
	for i, so := range item.Sockets {
		if i > 0 {
			if so.Group == item.Sockets[i-1].Group {
				got += "-"
			} else {
				got += " "
			}
		}
		got += so.Color
	}
	if got != "R-G B-B" {
		t.Fatalf("sockets %q", got)
	}
}

// TestEditItemInfluence: ItemsTab.lua:583-626: two influence dropdowns; choosing writes the "<Influence> Item" lines.
func TestEditItemInfluence(t *testing.T) {
	initItemsTabData(t)
	out, item := mustEdit(t, gameHelmet, ItemEdit{Kind: "influence", Text: "Shaper,Hunter"})
	if !item.Influence.Shaper || !item.Influence.Basilisk || item.Influence.Elder {
		t.Fatalf("influence %+v", item.Influence)
	}
	out, item = mustEdit(t, out, ItemEdit{Kind: "influence", Text: "Elder"})
	if item.Influence.Shaper || !item.Influence.Elder || strings.Contains(out, "Shaper Item") {
		t.Fatalf("influence %+v:\n%s", item.Influence, out)
	}
	if info, _ := DisplayItem(out); !slices.Equal(info.Influence, []string{"Elder"}) || !info.CanInfluence {
		t.Fatalf("display influence %v", info.Influence)
	}
}

// TestEditItemVariants: ItemsTab.lua:391-452: the variant and alt variant selectors.
func TestEditItemVariants(t *testing.T) {
	initItemsTabData(t)
	raw := corpusItemText(t, "0002-pathfinder-unknown.xml", 180)
	info, ok := DisplayItem(raw)
	if !ok || len(info.Variants) < 2 || len(info.AltVariants) == 0 {
		t.Fatalf("REFUSING: item 180 shows %d variants, %d alt", len(info.Variants), len(info.AltVariants))
	}
	pick := 1
	if info.Variant == 1 {
		pick = 2
	}
	_, item := mustEdit(t, raw, ItemEdit{Kind: "variant", Index: 0, Value: pick})
	if item.Variant == nil || *item.Variant != pick {
		t.Fatalf("variant %v, want %d", item.Variant, pick)
	}
	alt := 1
	if info.AltVariants[0] == 1 {
		alt = 2
	}
	_, item = mustEdit(t, raw, ItemEdit{Kind: "variant", Index: 1, Value: alt})
	if item.VariantAlt == nil || *item.VariantAlt != alt {
		t.Fatalf("alt variant %v, want %d", item.VariantAlt, alt)
	}
}

// TestEditItemCatalyst: ItemsTab.lua:654-694 on a rare ring with mod tags: a catalyst and its quality.
func TestEditItemCatalyst(t *testing.T) {
	initItemsTabData(t)
	raw := corpusItemText(t, "0001-necromancer-unknown.xml", 445)
	if info, _ := DisplayItem(raw); !info.ShowsCatalyst {
		t.Fatal("REFUSING: item 445 shows no catalyst selector")
	}
	out, item := mustEdit(t, raw, ItemEdit{Kind: "catalyst", Value: 8, Second: 15})
	if item.Catalyst == nil || *item.Catalyst != 8 || item.CatalystQuality == nil || *item.CatalystQuality != 15 {
		t.Fatalf("catalyst %v / %v:\n%s", item.Catalyst, *item.CatalystQuality, out)
	}
	_, item = mustEdit(t, out, ItemEdit{Kind: "catalyst", Value: 0})
	if item.Catalyst != nil && *item.Catalyst != 0 {
		t.Fatalf("catalyst kept %v", *item.Catalyst)
	}
}

// TestEditItemDisableAndRemove: ItemsTab.lua:2034-2047 and 2344-2371: a mod line toggles disabled; a rare's crafted line has Remove.
func TestEditItemDisableAndRemove(t *testing.T) {
	initItemsTabData(t)
	info, _ := DisplayItem(gameHelmet)
	crafted := slices.IndexFunc(info.Mods, func(m DisplayModLine) bool { return m.Removable })
	if crafted < 0 || info.Mods[crafted].Removal != "Crafted:" {
		t.Fatalf("no removable crafted line: %+v", info.Mods)
	}
	out, item := mustEdit(t, gameHelmet, ItemEdit{Kind: "disable", Index: 0})
	if !item.ExplicitModLines[0].Disabled {
		t.Fatalf("not disabled:\n%s", out)
	}
	out, item = mustEdit(t, out, ItemEdit{Kind: "disable", Index: 0})
	if item.ExplicitModLines[0].Disabled {
		t.Fatalf("still disabled:\n%s", out)
	}
	_, item = mustEdit(t, gameHelmet, ItemEdit{Kind: "remove", Index: crafted})
	if len(item.ExplicitModLines) != 2 || slices.ContainsFunc(item.ExplicitModLines, func(m ModLine) bool { return m.Crafted }) {
		t.Fatalf("explicits after Remove: %+v", item.ExplicitModLines)
	}
	if _, err := EditItem(gameHelmet, ItemEdit{Kind: "remove", Index: 0}); err == nil {
		t.Fatal("removed a line that is not custom or crafted")
	}
}

// TestEditItemRange: ItemsTab.lua:1025-1033: a ranged line's roll.
func TestEditItemRange(t *testing.T) {
	initItemsTabData(t)
	const unique = "Rarity: UNIQUE\nTest Ring\nRuby Ring\nImplicits: 1\n+(20-30)% to Fire Resistance\n+(30-50) to maximum Life"
	info, _ := DisplayItem(unique)
	k := slices.IndexFunc(info.Mods, func(m DisplayModLine) bool { return m.HasRange && m.Section == "explicit" })
	if k < 0 {
		t.Fatalf("no ranged explicit: %+v", info.Mods)
	}
	out, _ := mustEdit(t, unique, ItemEdit{Kind: "range", Index: k, Float: 1})
	info, _ = DisplayItem(out)
	if info.Mods[k].Range != 1 || info.Mods[k].Line != "+50 to maximum Life" {
		t.Fatalf("after range 1: %+v\n%s", info.Mods[k], out)
	}
}

// TestCraftItemText: ItemsTab.lua:2521-2565: a crafted rare is named, its base and quality set; a flask cannot be rare.
func TestCraftItemText(t *testing.T) {
	initItemsTabData(t)
	text, ok := CraftItemText(2, "", "Hubris Circlet")
	item := ParseItem(text)
	if !ok || item.Rarity != RarityRare || item.Title != "New Item" || item.BaseName != "Hubris Circlet" || item.Quality == nil || *item.Quality != 20 {
		t.Fatalf("%v %+v:\n%s", ok, item, text)
	}
	text, _ = CraftItemText(2, "x", "Divine Life Flask")
	if item := ParseItem(text); item.Rarity != RarityMagic {
		t.Fatalf("flask rarity %s", item.Rarity)
	}
	text, _ = CraftItemText(0, "", "Ruby Ring")
	if item := ParseItem(text); item.Quality != nil {
		t.Fatalf("a ring took quality %d", *item.Quality)
	}
	groups := CraftBaseGroups()
	if !slices.IsSortedFunc(groups, func(a, b CraftBaseGroup) int { return strings.Compare(a.Type, b.Type) }) || len(groups) < 20 {
		t.Fatalf("%d groups", len(groups))
	}
}

// TestEditItemRangeOverModRange: ItemsTab.lua:1239-1256: a build item's <ModRange> rolls are its lines' ranges.
func TestEditItemRangeOverModRange(t *testing.T) {
	initItemsTabData(t)
	const saved = "Rarity: UNIQUE\nTest Ring\nRuby Ring\nImplicits: 1\n+(20-30)% to Fire Resistance\n+(10-20) to Strength\n+(30-50) to maximum Life\n" +
		"\t\t\t<ModRange id=\"1\" range=\"0\"/>\n\t\t\t<ModRange id=\"2\" range=\"1\"/>\n\t\t\t<ModRange id=\"3\" range=\"0.25\"/>"
	info, _ := DisplayItem(saved)
	k := slices.IndexFunc(info.Mods, func(m DisplayModLine) bool { return m.Line == "+35 to maximum Life" })
	if k < 0 {
		t.Fatalf("REFUSING: the <ModRange> roll is not read: %+v", info.Mods)
	}
	out, _ := mustEdit(t, saved, ItemEdit{Kind: "range", Index: k, Float: 1})
	info, _ = DisplayItem(out)
	got := make([]string, 0, len(info.Mods))
	for _, m := range info.Mods {
		got = append(got, m.Line)
	}
	if want := []string{"+20% to Fire Resistance", "+20 to Strength", "+50 to maximum Life"}; !slices.Equal(got, want) {
		t.Fatalf("after the edit %q, want %q:\n%s", got, want, out)
	}
}

// TestCrucibleLines: Item.lua:1293-1294: a {crucible} line is a crucible line.
func TestCrucibleLines(t *testing.T) {
	initItemsTabData(t)
	const text = "Rarity: UNIQUE\nTest Staff\nLong Staff\nImplicits: 0\n+(30-50) to maximum Life\n{crucible}+20 to Strength"
	item := ParseItem(text)
	if len(item.ExplicitModLines) != 1 || len(item.CrucibleModLines) != 1 || item.CrucibleModLines[0].Line != "+20 to Strength" {
		t.Fatalf("explicit %d, crucible %+v", len(item.ExplicitModLines), item.CrucibleModLines)
	}
	if !slices.ContainsFunc(item.Mods, func(m mod.Mod) bool { return m.Name() == "Str" }) {
		t.Fatal("the crucible line's mod is not on the item")
	}
	tip := AddItemTooltip(item, ItemTooltipContext{})
	if last := tip.Lines[len(tip.Lines)-1]; !slices.ContainsFunc(tip.Lines, func(l TooltipLine) bool { return l.Text == "^xFFA500+20 to Strength" }) {
		t.Fatalf("no crucible-coloured line; last %q", last.Text)
	}
	if raw := buildRawText(item); !strings.Contains(raw, "{crucible}+20 to Strength") {
		t.Fatalf("BuildRaw lost the tag:\n%s", raw)
	}
	info, _ := DisplayItem(text)
	k := slices.IndexFunc(info.Mods, func(m DisplayModLine) bool { return m.Section == "crucible" })
	if k < 0 || !info.Mods[k].Removable || info.Mods[k].Removal != "Crucible:" {
		t.Fatalf("crucible line controls: %+v", info.Mods)
	}
	out, _ := mustEdit(t, text, ItemEdit{Kind: "remove", Index: k})
	if after := ParseItem(out); len(after.CrucibleModLines) != 0 || len(after.ExplicitModLines) != 1 {
		t.Fatalf("Remove left %+v:\n%s", after.CrucibleModLines, out)
	}
}

// TestEditItemVariantGroups: Item.lua:335-403, 2081-2099; ItemsTab.lua:1871-1938.
func TestEditItemVariantGroups(t *testing.T) {
	initItemsTabData(t)
	text, ok := ItemDBText(ItemDBUnique, "Zana's Ingenuity, Prismatic Ring")
	if !ok {
		t.Fatal("REFUSING: Zana's Ingenuity is not in the DB")
	}
	info, _ := DisplayItem(text)
	vg := info.VariantGroups
	if vg == nil || len(vg.Groups) != 3 || vg.Groups[0].Selected != 1 || vg.Groups[1].Selected != 4 || vg.Groups[2].Selected != 9 {
		t.Fatalf("groups %+v", vg)
	}
	out, _ := mustEdit(t, text, ItemEdit{Kind: "variantGroup", Index: 1, Value: 3})
	item := ParseItem(out)
	if item.VariantGroupSelections[1] != 3 || !strings.Contains(out, "Selected Variant Group: 1=3") {
		t.Fatalf("selections %v:\n%s", item.VariantGroupSelections, out)
	}
	var shown []string
	for _, ml := range item.ExplicitModLines {
		if item.checkModLineVariant(ml) {
			shown = append(shown, ml.Line)
		}
	}
	if !slices.Contains(shown, "20% increased maximum Mana") || slices.Contains(shown, "10% increased maximum Life") {
		t.Fatalf("shown %q", shown)
	}
	if !slices.ContainsFunc(item.Mods, func(m mod.Mod) bool { return m.Name() == "Mana" }) || slices.ContainsFunc(item.Mods, func(m mod.Mod) bool { return m.Name() == "Life" }) {
		t.Fatal("the mod list does not follow the group")
	}
}
