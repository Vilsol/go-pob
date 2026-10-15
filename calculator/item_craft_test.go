package calculator

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

const pobHelmet = `Rarity: RARE
Doom Visor
Hubris Circlet
Quality: 20
Implicits: 0
+80 to maximum Life
{crafted}+35% to Cold Resistance`

func helmetEnchant(t *testing.T, k int) string {
	t.Helper()
	table := data.Enchantments("Helmet")
	if table == nil || len(table.Skills["Fireball"]["MERCILESS"]) <= k {
		t.Fatal("REFUSING: no Fireball helmet enchantments in the data")
	}
	return table.Skills["Fireball"]["MERCILESS"][k]
}

// TestEditItemEnchant: ItemsTab.lua:2729-2753: Enchant puts the line in the slot, replacing what was there; Remove takes the slot's away.
func TestEditItemEnchant(t *testing.T) {
	initItemsTabData(t)
	first, second := helmetEnchant(t, 0), helmetEnchant(t, 1)
	for name, raw := range map[string]string{"pob": pobHelmet, "game": gameHelmet} {
		out, item := mustEdit(t, raw, ItemEdit{Kind: "enchant", Index: 1, Text: first})
		if len(item.EnchantModLines) != 1 || item.EnchantModLines[0].Line != first || len(item.ExplicitModLines) != len(ParseItem(raw).ExplicitModLines) {
			t.Fatalf("%s: enchants %v, %d explicits:\n%s", name, item.EnchantModLines, len(item.ExplicitModLines), out)
		}
		out, item = mustEdit(t, out, ItemEdit{Kind: "enchant", Index: 1, Text: second})
		if len(item.EnchantModLines) != 1 || item.EnchantModLines[0].Line != second {
			t.Fatalf("%s: replace gave %v:\n%s", name, item.EnchantModLines, out)
		}
		if info, _ := DisplayItem(out); info.Enchants != 1 || info.Anoints != 0 {
			t.Fatalf("%s: buttons %d enchant, %d anoint", name, info.Enchants, info.Anoints)
		}
		out, item = mustEdit(t, out, ItemEdit{Kind: "enchant", Index: 1, Value: 1})
		if len(item.EnchantModLines) != 0 || len(item.ExplicitModLines) != len(ParseItem(raw).ExplicitModLines) || !strings.Contains(out, "Implicits: 0") {
			t.Fatalf("%s: remove left %v:\n%s", name, item.EnchantModLines, out)
		}
	}
}

// TestEditItemEnchantTwoPart: ItemsTab.lua:2738-2740: a weapon's two-part enchantment ("a/b") is both enchant lines.
func TestEditItemEnchantTwoPart(t *testing.T) {
	initItemsTabData(t)
	var two string
	for _, line := range data.Enchantments("Bow").Sources["HARVEST"] {
		if strings.Contains(line, "/") {
			two = line
			break
		}
	}
	if two == "" {
		t.Fatal("REFUSING: no two-part bow enchantment in the data")
	}
	raw, _ := CraftItemText(2, "", "Imperial Bow")
	_, item := mustEdit(t, raw, ItemEdit{Kind: "enchant", Index: 1, Text: two})
	parts := strings.Split(two, "/")
	if len(item.EnchantModLines) != 2 || item.EnchantModLines[0].Line != parts[0] || item.EnchantModLines[1].Line != parts[1] {
		t.Fatalf("enchants %v for %q", item.EnchantModLines, two)
	}
}

const pobAmulet = `Rarity: RARE
Gale Pendant
Onyx Amulet
Implicits: 1
+12 to all Attributes
+70 to maximum Life`

// TestEditItemAnoint: ItemsTab.lua:2908-2920: Anoint puts "Allocates <notable>" in the slot in place of the old anoint; no notable removes it.
func TestEditItemAnoint(t *testing.T) {
	initItemsTabData(t)
	notables := AnointNotables()
	if len(notables) < 100 {
		t.Fatalf("REFUSING: %d anointable notables", len(notables))
	}
	a, b := notables[0].Name, notables[1].Name
	out, item := mustEdit(t, pobAmulet, ItemEdit{Kind: "anoint", Index: 1, Text: a})
	if got := ItemAnoints(item); !slices.Equal(got, []string{a}) || len(item.ImplicitModLines) != 1 || len(item.ExplicitModLines) != 1 {
		t.Fatalf("anoints %v, %d implicits, %d explicits:\n%s", got, len(item.ImplicitModLines), len(item.ExplicitModLines), out)
	}
	out, item = mustEdit(t, out, ItemEdit{Kind: "anoint", Index: 1, Text: b})
	if got := ItemAnoints(item); !slices.Equal(got, []string{b}) {
		t.Fatalf("re-anoint gave %v:\n%s", got, out)
	}
	if info, _ := DisplayItem(out); info.Anoints != 1 || info.Enchants != 0 || !slices.Equal(info.Anointed, []string{b}) {
		t.Fatalf("buttons %d anoint, %d enchant, anointed %v", info.Anoints, info.Enchants, info.Anointed)
	}
	_, item = mustEdit(t, out, ItemEdit{Kind: "anoint", Index: 1})
	if got := ItemAnoints(item); len(got) != 0 || len(item.ImplicitModLines) != 1 {
		t.Fatalf("remove left %v", got)
	}
}

// TestAnointButtons: Item.lua:1194-1206, ItemsTab.lua:63-67, 524-553: "Can be Anointed" (Cord Belt) and the extra enchantment lines.
func TestAnointButtons(t *testing.T) {
	initItemsTabData(t)
	belt := "Rarity: RARE\nTest Belt\nCord Belt\nImplicits: 1\nCan be Anointed\n+40 to maximum Life"
	if info, _ := DisplayItem(belt); info.Anoints != 1 || info.Enchants != 1 {
		t.Fatalf("Cord Belt: %d anoint, %d enchant buttons", info.Anoints, info.Enchants)
	}
	amulet := "Rarity: UNIQUE\nTest\nOnyx Amulet\nImplicits: 2\n{crafted}Allocates Constitution\n{crafted}Allocates Heart of Oak\nCan have 3 additional Enchantment Modifiers"
	if info, _ := DisplayItem(amulet); info.Anoints != 3 {
		t.Fatalf("two anoints and three more allowed: %d anoint buttons, want 3", info.Anoints)
	}
	talisman := "Rarity: RARE\nTest\nAvian Twins Talisman\nImplicits: 0"
	if info, ok := DisplayItem(talisman); !ok || info.Anoints != 0 {
		t.Fatalf("talisman: %d anoint buttons (ok %v)", info.Anoints, ok)
	}
}

// TestItemEnchantChoices: ItemsTab.lua:2665-2706: a helmet's enchantments are by skill, the build's skills marked; a belt's are by source.
func TestItemEnchantChoices(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	c, ok := ItemEnchantChoices(build, pobHelmet)
	if !ok || !c.HaveSkills || len(c.Skills) < 50 || !slices.Contains(c.Used, "Herald of Ash") {
		t.Fatalf("helmet: ok %v, by skill %v, %d skills, used %v", ok, c.HaveSkills, len(c.Skills), c.Used)
	}
	if !slices.IsSortedFunc(c.Skills, func(a, b EnchantSkill) int { return strings.Compare(a.Skill, b.Skill) }) {
		t.Fatal("skills not sorted")
	}
	belt, ok := ItemEnchantChoices(build, "Rarity: RARE\nTest\nLeather Belt\nImplicits: 0")
	if !ok || belt.HaveSkills || len(belt.Skills) != 1 || len(belt.Skills[0].Sources) == 0 {
		t.Fatalf("belt: %+v", belt)
	}
	if _, ok := ItemEnchantChoices(build, pobAmulet); ok {
		t.Fatal("an amulet has enchantments")
	}
}

// TestEnchantAnointPower: ItemsTab.lua:2754-2779, NotableDBControl.lua:113-160, ItemsTab.lua:2925-2963: the popups' power sort and comparison.
func TestEnchantAnointPower(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, helmet := activeSetSlot(build, "Helmet")
	vals := misc.EnchantSortValues(helmet, 1, []string{"+50 to maximum Life", "1% increased Fireball Cast Speed"}, "Life")
	if vals[0] <= vals[1] {
		t.Fatalf("Life with +50 life %v, with cast speed %v", vals[0], vals[1])
	}
	_, amulet := activeSetSlot(build, "Amulet")
	var life, other int64
	for _, n := range AnointNotables() {
		switch n.Name {
		case "Constitution":
			life = n.ID
		case "Fervour":
			other = n.ID
		}
	}
	if life == 0 || other == 0 {
		t.Fatal("REFUSING: Constitution / Fervour are not anointable")
	}
	p := misc.AnointPowers(amulet, 1, []int64{life, other}, "Life")
	if p[0] <= 0 || p[0] <= p[1] {
		t.Fatalf("Life power: Constitution %v, Fervour %v", p[0], p[1])
	}
	c, ok := misc.CompareAnoint(amulet, 1, life, "")
	if !ok || c.Note != "" || c.New.Player.Output["Life"] <= c.Base.Player.Output["Life"] || !strings.Contains(c.Header, "Anointing Constitution will give you") {
		t.Fatalf("compare: ok %v, note %q, header %q", ok, c.Note, c.Header)
	}
	var alloc int64
	for _, id := range build.Build.PassiveNodes {
		if n, ok := anointNode(id); ok && len(n.Recipe) > 0 {
			alloc = id
		}
	}
	if alloc == 0 {
		t.Fatal("REFUSING: the build allocates no anointable notable")
	}
	if c, _ := misc.CompareAnoint(amulet, 1, alloc, ""); !strings.Contains(c.Note, "already allocated") {
		t.Fatalf("allocated %d: %q", alloc, c.Note)
	}
}

// TestCorruptListsAgainstPoB: ItemsTab.lua:3129-3175, 3290-3332.
func TestCorruptListsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	var fixture struct {
		Items []string
		Pob   []struct {
			Corrupted, ScourgeUpside, ScourgeDownside []string
			Glimpse                                   []string `json:"Glimpse of Chaos"`
			Max                                       int
			Source                                    string
			CorruptedImplicits                        []string
		}
	}
	raw, err := os.ReadFile("../testdata/craft-ui/corrupt_lists.json")
	if err != nil || json.Unmarshal(raw, &fixture) != nil || len(fixture.Items) != len(fixture.Pob) || len(fixture.Items) < 8 {
		t.Fatalf("REFUSING: testdata/craft-ui/corrupt_lists.json: %v", err)
	}
	for k, text := range fixture.Items {
		c, ok := ItemCorruptChoices(text)
		if !ok {
			t.Fatalf("item %d not corruptible", k)
		}
		want := fixture.Pob[k]
		mods := map[string]data.AffixMod{}
		for _, m := range slices.Concat(itemAffixes(ParseItem(text)), data.SpecialCorruptionMods()) {
			mods[m.ID] = m
		}
		rank := func(ids []string) []string {
			out := make([]string, len(ids))
			for i, id := range ids {
				out[i] = corruptSortKey(mods[id]) + "\x00" + strconv.Itoa(mods[id].Level)
			}
			return out
		}
		for modType, got := range map[string][]CorruptMod{"Corrupted": c.Corrupted, "ScourgeUpside": c.ScourgeUpside,
			"ScourgeDownside": c.ScourgeDownside, "Glimpse of Chaos": c.Glimpse} {
			ids := make([]string, len(got))
			for i, m := range got {
				ids[i] = m.ID
			}
			w := map[string][]string{"Corrupted": want.Corrupted, "ScourgeUpside": want.ScourgeUpside, "ScourgeDownside": want.ScourgeDownside, "Glimpse of Chaos": want.Glimpse}[modType]
			if !slices.Equal(slices.Sorted(slices.Values(ids)), slices.Sorted(slices.Values(w))) || !slices.Equal(rank(ids), rank(w)) {
				t.Errorf("item %d %s:\n got %v\nwant %v", k, modType, ids, w)
			}
		}
		if c.MaxImplicits != want.Max {
			t.Errorf("item %d: %d implicit dropdowns, PoB %d", k, c.MaxImplicits, want.Max)
		}
		first := map[string][]string{"Corrupted": want.Corrupted, "Glimpse of Chaos": want.Glimpse}[want.Source]
		out, _ := CorruptItemText(text, want.Source, first[:min(2, len(first))], nil, nil)
		var lines []string
		for _, ml := range ParseItem(out).ImplicitModLines {
			lines = append(lines, ml.Line)
		}
		if !slices.Equal(lines, want.CorruptedImplicits) {
			t.Errorf("item %d corrupted with %s %v: implicits %q, PoB %q", k, want.Source, first[:min(2, len(first))], lines, want.CorruptedImplicits)
		}
	}
}

// TestCorruptItemText: ItemsTab.lua:3136-3175, 557-559.
func TestCorruptItemText(t *testing.T) {
	initItemsTabData(t)
	c, _ := ItemCorruptChoices(pobAmulet)
	if len(c.Corrupted) < 2 {
		t.Fatalf("REFUSING: %d corrupted implicits for an amulet", len(c.Corrupted))
	}
	a, b := c.Corrupted[0], c.Corrupted[len(c.Corrupted)-1]
	out, ok := CorruptItemText(pobAmulet, "Corrupted", []string{a.ID, b.ID}, nil, nil)
	item := ParseItem(out)
	got := make([]string, 0, len(item.ImplicitModLines))
	for _, ml := range item.ImplicitModLines {
		got = append(got, ml.Line)
	}
	if !ok || !item.Corrupted || !slices.Equal(got, slices.Concat(a.Lines, b.Lines)) || len(item.ExplicitModLines) != 1 {
		t.Fatalf("ok %v corrupted %v implicits %v:\n%s", ok, item.Corrupted, got, out)
	}
	if len(a.ModTags) > 0 && !slices.Equal(item.ImplicitModLines[0].ModTags, a.ModTags) {
		t.Fatalf("tags %v, want %v", item.ImplicitModLines[0].ModTags, a.ModTags)
	}
	out, _ = CorruptItemText(pobAmulet, "Scourge", []string{c.ScourgeUpside[0].ID}, []string{c.ScourgeDownside[0].ID}, nil)
	item = ParseItem(out)
	if len(item.ScourgeModLines) != len(c.ScourgeUpside[0].Lines)+len(c.ScourgeDownside[0].Lines) || len(item.ImplicitModLines) != 1 || !item.Corrupted {
		t.Fatalf("scourge: %d scourge lines, %d implicits:\n%s", len(item.ScourgeModLines), len(item.ImplicitModLines), out)
	}
	if out, _ := CorruptItemText(pobAmulet, "Corrupted", nil, nil, nil); !ParseItem(out).Corrupted || len(ParseItem(out).ImplicitModLines) != 1 {
		t.Fatalf("no implicit chosen keeps the implicits:\n%s", out)
	}
	if info, _ := DisplayItem(pobAmulet); !info.Corruptible {
		t.Fatal("an amulet is not corruptible")
	}
	if info, _ := DisplayItem("Rarity: MAGIC\nDivine Life Flask\nImplicits: 0"); info.Corruptible {
		t.Fatal("a flask is corruptible")
	}
}

// TestCorruptRolls: ItemsTab.lua:3176-3212, 3165-3172.
func TestCorruptRolls(t *testing.T) {
	initItemsTabData(t)
	unique := "Rarity: UNIQUE\nTest Crown\nHubris Circlet\nImplicits: 0\n{range:0.5}+(40-60) to maximum Life\nCannot be Frozen"
	c, _ := ItemCorruptChoices(unique)
	if !c.Unique || len(c.Rolls) != 1 || c.Rolls[0].Index != 0 || c.Rolls[0].Value != 1 || c.Rolls[0].Line != "+50 to maximum Life" {
		t.Fatalf("rolls %+v", c.Rolls)
	}
	if got := CorruptedRangeLine(unique, 0, 1.22); got != "+61 to maximum Life" {
		t.Fatalf("1.22 roll: %q", got)
	}
	out, _ := CorruptItemText(unique, "Corrupted", nil, nil, map[int]float64{0: 1.22})
	item := ParseItem(out)
	if r := item.ExplicitModLines[0].CorruptedRange; r == nil || *r != 1.22 || !item.Corrupted {
		t.Fatalf("roll %v:\n%s", r, out)
	}
	out, _ = CorruptItemText(out, "Corrupted", nil, nil, map[int]float64{0: 1})
	if ParseItem(out).ExplicitModLines[0].CorruptedRange != nil || strings.Contains(out, "corruptedRange") {
		t.Fatalf("roll 1 kept the tag:\n%s", out)
	}
}

// TestCorruptSortValues: ItemsTab.lua:3064-3088: each implicit is valued on the item corrupted to it alone.
func TestCorruptSortValues(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, helmet := activeSetSlot(build, "Helmet")
	vals := misc.CorruptSortValues(helmet, "Corrupted", "Corrupted", []string{"V2IncreasedMaximumLifeCorrupted", "V2IncreasedLightRadiusCorrupted"}, "Life")
	if vals[0] <= vals[1] || vals[1] <= 0 {
		t.Fatalf("Life with increased life %v, with light radius %v", vals[0], vals[1])
	}
}

// TestImplicitListsAgainstPoB: AddImplicitPopup.lua:45-181: the Add Implicit popup's Exarch / Eater types and modifiers for four bases.
func TestImplicitListsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	type group struct {
		Label string
		IDs   []string
	}
	var fixture struct {
		Items []string
		Pob   []struct {
			EXARCH, EATER, VESTIGIAL []group
			Sources                  []string
		}
	}
	raw, err := os.ReadFile("../testdata/craft-ui/implicit_lists.json")
	if err != nil || json.Unmarshal(raw, &fixture) != nil || len(fixture.Items) != len(fixture.Pob) || len(fixture.Items) < 11 {
		t.Fatalf("REFUSING: testdata/craft-ui/implicit_lists.json: %v", err)
	}
	flat := func(gs []group) []string {
		out := make([]string, 0, len(gs))
		for _, g := range gs {
			out = append(out, g.Label+"="+strings.Join(g.IDs, ","))
		}
		return out
	}
	vestigial := 0
	for k, text := range fixture.Items {
		item := ParseItem(text)
		want := fixture.Pob[k]
		got := map[string][]group{}
		for _, source := range []string{"EXARCH", "EATER", "VESTIGIAL"} {
			for _, g := range buildImplicitMods(item, source) {
				ids := make([]string, len(g.Mods))
				for i, m := range g.Mods {
					ids[i] = m.ID
				}
				got[source] = append(got[source], group{g.Label, ids})
			}
		}
		for source, w := range map[string][]group{"EXARCH": want.EXARCH, "EATER": want.EATER} {
			a, b := flat(got[source]), flat(w)
			slices.Sort(a)
			slices.Sort(b)
			if !slices.Equal(a, b) {
				t.Errorf("item %d %s:\n got %v\nwant %v", k, source, a, b)
			}
		}
		offered := slices.ContainsFunc(implicitSources(item), func(s ImplicitSource) bool { return s.ID == "VESTIGIAL" })
		if offered != slices.Contains(want.Sources, "VESTIGIAL") {
			t.Errorf("item %d: Vestigial offered %v, PoB %v", k, offered, want.Sources)
		}
		if offered {
			vestigial++
			if a, b := flat(got["VESTIGIAL"]), flat(want.VESTIGIAL); !slices.Equal(a, b) {
				t.Errorf("item %d VESTIGIAL: %d entries, want %d; first %v, want %v", k, len(a), len(b), a[:min(1, len(a))], b[:min(1, len(b))])
			}
		}
	}
	if vestigial < 5 {
		t.Fatalf("REFUSING: only %d items offer Vestigial", vestigial)
	}
}

// TestImplicitText: ItemsTab.lua:571-576, 3920-3932.
func TestImplicitText(t *testing.T) {
	initItemsTabData(t)
	coat := "Rarity: RARE\nTest Coat\nVaal Regalia\nSearing Exarch Item\nEater of Worlds Item\nImplicits: 0\n+80 to maximum Life"
	c, ok := ItemImplicitChoices(coat)
	ids := make([]string, 0, len(c.Sources))
	for _, s := range c.Sources {
		ids = append(ids, s.ID)
	}
	if !ok || !slices.Equal(ids, []string{"EXARCH", "EATER", "CUSTOM"}) {
		t.Fatalf("sources %v", ids)
	}
	if info, _ := DisplayItem(coat); !info.AddImplicit {
		t.Fatal("a rare body armour has no Add Implicit")
	}
	if info, _ := DisplayItem("Rarity: UNIQUE\nTest\nVaal Regalia\nImplicits: 0\nImplicit Modifiers Cannot Be Changed"); info.AddImplicit {
		t.Fatal("Add Implicit on an item whose implicits cannot be changed")
	}
	first, second := c.Groups["EXARCH"][0].Mods[0], c.Groups["EXARCH"][1].Mods[0]
	out, _ := ImplicitText(coat, "EXARCH", first.ID, "")
	out, _ = ImplicitText(out, "EXARCH", second.ID, "")
	item := ParseItem(out)
	if len(item.ImplicitModLines) != len(second.Lines) || item.ImplicitModLines[0].Line != second.Lines[0] {
		t.Fatalf("a second Exarch implicit did not replace the first:\n%s", out)
	}
	out, _ = ImplicitText(out, "CUSTOM", "", "+5 to Strength")
	item = ParseItem(out)
	if last := item.ImplicitModLines[len(item.ImplicitModLines)-1]; last.Line != "+5 to Strength" || !last.Custom {
		t.Fatalf("custom implicit %+v:\n%s", last, out)
	}
	kaom := "Rarity: UNIQUE\nKaom's Heart\nGlorious Plate\nImplicits: 1\n+5 to Strength\nHas no Sockets\n+500 to maximum Life"
	vc, _ := ItemImplicitChoices(kaom)
	v := vc.Groups["VESTIGIAL"][0].Mods[0]
	out, _ = ImplicitText(kaom, "VESTIGIAL", v.ID, "")
	item = ParseItem(out)
	if len(item.ImplicitModLines) != len(v.Lines) || item.ImplicitModLines[0].Line != v.Lines[0] || !item.ImplicitModLines[0].Vestigial {
		t.Fatalf("vestigial implicit %+v:\n%s", item.ImplicitModLines, out)
	}
}

// TestImplicitSortValues: ItemsTab.lua:3957-3970: each modifier is valued on the item with it added.
func TestImplicitSortValues(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, helmet := activeSetSlot(build, "Helmet")
	var life, other string
	for _, g := range buildImplicitMods(ParseItem(helmet), "EATER") {
		for _, m := range g.Mods {
			switch {
			case life == "" && strings.HasSuffix(m.Label, "increased Mana Regeneration Rate"):
				life = m.ID
			case other == "" && strings.HasSuffix(m.Label, "chance to Freeze"):
				other = m.ID
			}
		}
	}
	if life == "" || other == "" {
		t.Fatalf("REFUSING: no Eater mana regeneration / freeze implicit for the helmet (%q, %q)", life, other)
	}
	vals := misc.ImplicitSortValues(helmet, "EATER", []string{life, other}, "ManaRegen")
	if vals[0] <= vals[1] {
		t.Fatalf("ManaRegen with %s %v, with %s %v", life, vals[0], other, vals[1])
	}
}

// TestCustomModListsAgainstPoB: ItemsTab.lua:3579-3797: the Add modifier popup's sources.
func TestCustomModListsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	var fixture struct {
		Items []string
		Pob   []map[string]json.RawMessage
	}
	raw, err := os.ReadFile("../testdata/craft-ui/custom_mod_lists.json")
	if err != nil || json.Unmarshal(raw, &fixture) != nil || len(fixture.Items) != len(fixture.Pob) || len(fixture.Items) < 9 {
		t.Fatalf("REFUSING: testdata/craft-ui/custom_mod_lists.json: %v", err)
	}
	for k, text := range fixture.Items {
		c, ok := ItemCustomModChoices(text)
		if !ok {
			t.Fatalf("item %d has no base", k)
		}
		var wantSources []string
		_ = json.Unmarshal(fixture.Pob[k]["sources"], &wantSources)
		gotSources := make([]string, 0, len(c.Sources))
		for _, s := range c.Sources {
			gotSources = append(gotSources, s.ID)
		}
		if !slices.Equal(gotSources, wantSources) {
			t.Errorf("item %d sources %v, want %v", k, gotSources, wantSources)
		}
		for _, s := range wantSources {
			if s == "CUSTOM" {
				continue
			}
			var want []string
			_ = json.Unmarshal(fixture.Pob[k][s], &want)
			want = craftDataDrift(s, want)
			got := make([]string, 0, len(c.Mods[s]))
			for _, m := range c.Mods[s] {
				got = append(got, m.Label)
			}
			if s == "PREFIX" || s == "SUFFIX" {
				slices.Sort(got)
				slices.Sort(want)
			}
			if !slices.Equal(got, want) {
				t.Errorf("item %d %s: %d entries, want %d\n got %v\nwant %v", k, s, len(got), len(want), got, want)
			}
		}
	}
}

func craftDataDrift(source string, want []string) []string {
	switch source {
	case "ESSENCE":
		return slices.DeleteFunc(want, func(l string) bool { return strings.HasPrefix(l, "Essence of Desolation ") })
	case "MASTER":
		power := slices.Index(want, "+1 to Minimum Power Charges ^8(Suffix)")
		if power >= 0 && power+1 < len(want) && want[power+1] == "+1 to Minimum Frenzy Charges ^8(Suffix)" {
			want[power], want[power+1] = want[power+1], want[power]
		}
	}
	return want
}

// TestCustomModText: ItemsTab.lua:991-993, 3591-3609, 3356-3371: Add modifier...
func TestCustomModText(t *testing.T) {
	initItemsTabData(t)
	if info, _ := DisplayItem(pobHelmet); !info.AddCustom {
		t.Fatal("a rare helmet has no Add modifier")
	}
	if info, _ := DisplayItem("Rarity: UNIQUE\nTest\nHubris Circlet\nImplicits: 0"); info.AddCustom {
		t.Fatal("Add modifier on a unique")
	}
	crafted, _ := CraftItemText(2, "", "Hubris Circlet")
	if cc, _ := ItemCustomModChoices(crafted); slices.ContainsFunc(cc.Sources, func(s ImplicitSource) bool { return s.ID == "PREFIX" }) {
		t.Fatal("a crafted item offers Prefix (ItemsTab.lua:3572)")
	}
	c, _ := ItemCustomModChoices(pobHelmet)
	bench := c.Mods["MASTER"][0]
	out, ok := CustomModText(pobHelmet+"\nCorrupted", "MASTER", bench.ID, "")
	item := ParseItem(out)
	last := item.ExplicitModLines[len(item.ExplicitModLines)-1]
	if !ok || last.Line != bench.Lines[len(bench.Lines)-1] || !last.Crafted || !strings.HasSuffix(out, "Corrupted") {
		t.Fatalf("bench mod %+v:\n%s", last, out)
	}
	if len(bench.ModTags) > 0 && !slices.Equal(item.ExplicitModLines[len(item.ExplicitModLines)-len(bench.Lines)].ModTags, bench.ModTags) {
		t.Fatalf("tags lost:\n%s", out)
	}
	out, _ = CustomModText(pobHelmet, "CUSTOM", "", "+7 to Dexterity")
	if last := ParseItem(out).ExplicitModLines; last[len(last)-1].Line != "+7 to Dexterity" || !last[len(last)-1].Custom {
		t.Fatalf("custom:\n%s", out)
	}
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, helmet := activeSetSlot(build, "Helmet")
	var life, other string
	for _, m := range buildCustomMods(helmet, ParseItem(helmet), "PREFIX") {
		switch {
		case life == "" && strings.Contains(m.Label, "to maximum Life]"):
			life = m.ID
		case other == "" && strings.Contains(m.Label, "to maximum Mana]"):
			other = m.ID
		}
	}
	if life == "" || other == "" {
		t.Fatalf("REFUSING: no life / mana prefix for the helmet (%q, %q)", life, other)
	}
	if v := misc.CustomModSortValues(helmet, "PREFIX", []string{life, other}, "Life"); v[0] <= v[1] {
		t.Fatalf("Life with %s %v, with %s %v", life, v[0], other, v[1])
	}
}

// TestAffixControlsAgainstPoB: ItemsTab.lua:2129-2217 and Item:Craft.
func TestAffixControlsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	type series struct {
		Label string
		IDs   []string
	}
	var fixture struct {
		Items []struct {
			Raw   string
			Edits []struct {
				Slot  int
				ModID string
				Range float64
			}
		}
		Pob []struct {
			Limit    int
			Slots    [][]series
			Explicit []string
			Name     string
			Level    *int
		}
	}
	raw, err := os.ReadFile("../testdata/craft-ui/affix_controls.json")
	if err != nil || json.Unmarshal(raw, &fixture) != nil || len(fixture.Items) != len(fixture.Pob) || len(fixture.Items) < 6 {
		t.Fatalf("REFUSING: testdata/craft-ui/affix_controls.json: %v", err)
	}
	for k, it := range fixture.Items {
		want := fixture.Pob[k]
		c, ok := ItemAffixControls(it.Raw)
		if !ok || len(c.Controls) != want.Limit {
			t.Fatalf("item %d: ok %v, %d dropdowns, want %d", k, ok, len(c.Controls), want.Limit)
		}
		for i, ctl := range c.Controls {
			var got []series
			for _, s := range ctl.Series[1:] {
				got = append(got, series{s.Label, s.ModList})
			}
			if len(got) != len(want.Slots[i]) {
				t.Errorf("item %d slot %d: %d entries, want %d", k, i, len(got), len(want.Slots[i]))
				continue
			}
			for j := range got {
				if got[j].Label != want.Slots[i][j].Label || !slices.Equal(got[j].IDs, want.Slots[i][j].IDs) {
					t.Errorf("item %d slot %d entry %d: %v, want %v", k, i, j, got[j], want.Slots[i][j])
					break
				}
			}
		}
		item := ParseItem(it.Raw)
		st, _ := itemAffixState(it.Raw, item)
		for _, e := range it.Edits {
			a, _, _ := st.slot(e.Slot - 1)
			*a = affixSlot{modID: e.ModID, rng: e.Range, hasRange: true}
		}
		out := ParseItem(craftText(it.Raw, item, st))
		got := make([]string, 0, len(out.ExplicitModLines))
		for _, ml := range out.ExplicitModLines {
			got = append(got, ml.Line)
		}
		if !slices.Equal(got, want.Explicit) {
			t.Errorf("item %d explicit %v, want %v", k, got, want.Explicit)
		}
		if out.Rarity == RarityMagic && out.Name != want.Name {
			t.Errorf("item %d name %q, want %q", k, out.Name, want.Name)
		}
		if want.Level != nil && out.Requirements["level"] != float64(*want.Level) {
			t.Errorf("item %d level %v, want %d", k, out.Requirements["level"], *want.Level)
		}
	}
}

// TestAffixEditText: ItemsTab.lua:796-811, 931-968.
func TestAffixEditText(t *testing.T) {
	initItemsTabData(t)
	circlet, _ := CraftItemText(2, "", "Hubris Circlet")
	c, _ := ItemAffixControls(circlet)
	life := slices.IndexFunc(c.Controls[0].Series, func(s AffixSeries) bool { return s.Label == "+# to maximum Life" })
	if life < 0 {
		t.Fatal("REFUSING: no maximum Life prefix on the circlet")
	}
	tiers := c.Controls[0].Series[life].ModList
	out, ok := AffixEditText(circlet, 0, life, 0.999)
	if !ok || !strings.Contains(out, "Prefix: {range:") || !strings.Contains(out, "}"+tiers[len(tiers)-1]) {
		t.Fatalf("top tier not set:\n%s", out)
	}
	c2, _ := ItemAffixControls(out)
	if c2.Controls[0].Sel != life || !c2.Controls[0].SliderShown || c2.Controls[0].SliderVal < 0.9 {
		t.Fatalf("control after the edit: sel %d slider %v %v", c2.Controls[0].Sel, c2.Controls[0].SliderShown, c2.Controls[0].SliderVal)
	}
	if tip := AffixRollTooltip(out, 0, 0.999); len(tip) < 3 || !strings.HasPrefix(tip[2], "^7Affix: Tier 1 (") {
		t.Fatalf("slider tooltip %q", tip)
	}
	cleared, _ := AffixEditText(out, 0, 0, 0)
	if strings.Contains(cleared, tiers[len(tiers)-1]) || len(ParseItem(cleared).ExplicitModLines) != 0 {
		t.Fatalf("None kept the affix:\n%s", cleared)
	}
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	_, helmet := activeSetSlot(build, "Helmet")
	crafted, _ := CraftItemText(2, "", ParseItem(helmet).BaseName)
	hc, _ := ItemAffixControls(crafted)
	powers := misc.AffixPowers(crafted, 0, "Life")
	lifeAt := slices.IndexFunc(hc.Controls[0].Series, func(s AffixSeries) bool { return s.Label == "+# to maximum Life" })
	manaAt := slices.IndexFunc(hc.Controls[0].Series, func(s AffixSeries) bool { return s.Label == "+# to maximum Mana" })
	if lifeAt < 0 || manaAt < 0 || powers[lifeAt] <= powers[manaAt] {
		t.Fatalf("Life power of the life prefix %v, of the mana prefix %v (%d, %d)", powers[max(lifeAt, 0)], powers[max(manaAt, 0)], lifeAt, manaAt)
	}
}

// TestCraftItemImplicitsAndHiddenBases: ItemsTab.lua:2554-2561, Data.lua:1224-1234.
func TestCraftItemImplicitsAndHiddenBases(t *testing.T) {
	initItemsTabData(t)
	text, _ := CraftItemText(2, "", "Two-Stone Ring (Cold/Lightning)")
	item := ParseItem(text)
	if len(item.ImplicitModLines) != 1 || item.ImplicitModLines[0].Line != "+(12-16)% to Cold and Lightning Resistances" || len(item.ImplicitModLines[0].ModTags) == 0 {
		t.Fatalf("ring implicits %+v:\n%s", item.ImplicitModLines, text)
	}
	var hidden string
	for _, b := range ItemBases() {
		if pb, ok := data.CraftBaseKnown(b.Name); ok && pb.Hidden {
			hidden = b.Name
			break
		}
	}
	if hidden == "" {
		t.Fatal("REFUSING: no hidden base among Go's bases")
	}
	clusters := false
	for _, g := range CraftBaseGroups() {
		if slices.Contains(g.Bases, hidden) {
			t.Fatalf("hidden base %q listed under %s", hidden, g.Type)
		}
		clusters = clusters || (g.Type == "Jewel: Cluster" && slices.Contains(g.Bases, "Large Cluster Jewel"))
	}
	if !clusters {
		t.Fatal(`no "Jewel: Cluster" type with the Large Cluster Jewel (PoB's base type and subType)`)
	}
}

// TestClusterCraftAgainstPoB: ItemsTab.lua:696-715, 2059-2105.
func TestClusterCraftAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	type series struct {
		Label string
		IDs   []string
	}
	var fixture struct {
		Cases []struct {
			Raw   string
			Skill string
			Count int
		}
		Pob []struct {
			Enchant  []string
			Explicit []string
			Slots    [][]series
		}
	}
	raw, err := os.ReadFile("../testdata/craft-ui/cluster_craft.json")
	if err != nil || json.Unmarshal(raw, &fixture) != nil || len(fixture.Cases) != len(fixture.Pob) || len(fixture.Cases) < 3 {
		t.Fatalf("REFUSING: testdata/craft-ui/cluster_craft.json: %v", err)
	}
	for k, c := range fixture.Cases {
		if info, ok := DisplayItem(c.Raw); !ok || info.Cluster == nil || len(info.Cluster.Skills) == 0 {
			t.Fatalf("case %d: no cluster controls", k)
		}
		out, item := mustEdit(t, c.Raw, ItemEdit{Kind: "cluster", Text: c.Skill, Value: c.Count})
		want := fixture.Pob[k]
		got := make([]string, 0, len(item.EnchantModLines))
		for _, ml := range item.EnchantModLines {
			got = append(got, ml.Line)
		}
		if !slices.Equal(got, want.Enchant) || len(item.ExplicitModLines) != len(want.Explicit) {
			t.Errorf("case %d: enchant %q (%d explicit), want %q (%d)\n%s", k, got, len(item.ExplicitModLines), want.Enchant, len(want.Explicit), out)
		}
		info, _ := DisplayItem(out)
		if info.Cluster.Skill != c.Skill || info.Cluster.NodeCount != c.Count {
			t.Errorf("case %d: controls read back %s / %d", k, info.Cluster.Skill, info.Cluster.NodeCount)
		}
		ctl, _ := ItemAffixControls(out)
		for i, cc := range ctl.Controls {
			var gotS []series
			for _, s := range cc.Series[1:] {
				gotS = append(gotS, series{s.Label, s.ModList})
			}
			if len(gotS) != len(want.Slots[i]) {
				t.Errorf("case %d slot %d: %d entries, want %d", k, i, len(gotS), len(want.Slots[i]))
				continue
			}
			for j := range gotS {
				if gotS[j].Label != want.Slots[i][j].Label || !slices.Equal(gotS[j].IDs, want.Slots[i][j].IDs) {
					t.Errorf("case %d slot %d entry %d: %v, want %v", k, i, j, gotS[j], want.Slots[i][j])
					break
				}
			}
		}
	}
}

// TestCrucibleListsAgainstPoB: ItemsTab.lua:3664-3758, Item:CanHaveMod.
func TestCrucibleListsAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	var fixture struct {
		Items []string
		Pob   [][]struct {
			Labels   []string
			Selected string
		}
	}
	raw, err := os.ReadFile("../testdata/craft-ui/crucible_lists.json")
	if err != nil || json.Unmarshal(raw, &fixture) != nil || len(fixture.Items) != len(fixture.Pob) || len(fixture.Items) < 4 {
		t.Fatalf("REFUSING: testdata/craft-ui/crucible_lists.json: %v", err)
	}
	for k, text := range fixture.Items {
		c, ok := ItemCrucibleChoices(text)
		if !ok {
			t.Fatalf("item %d has no crucible tree", k)
		}
		for n, want := range fixture.Pob[k] {
			got := make([]string, 0, len(c.Nodes[n]))
			for _, e := range c.Nodes[n] {
				got = append(got, e.Label)
			}
			if !slices.Equal(got, want.Labels) || c.Selected[n] != want.Selected {
				t.Errorf("item %d node %d: %d entries (selected %q), want %d (%q)", k, n+1, len(got), c.Selected[n], len(want.Labels), want.Selected)
			}
		}
	}
	c, _ := ItemCrucibleChoices(fixture.Items[2])
	out, ok := CrucibleText(fixture.Items[2], []string{c.Nodes[0][0].ID, "", "", "", ""})
	item := ParseItem(out)
	if !ok || strings.Contains(out, "10% increased Attack Speed") || !strings.Contains(out, "{crucible}") {
		t.Fatalf("crucible lines not replaced:\n%s", out)
	}
	if len(item.CrucibleModLines) == 0 || item.CrucibleModLines[len(item.CrucibleModLines)-1].Line != c.Nodes[0][0].Lines[len(c.Nodes[0][0].Lines)-1] {
		t.Fatalf("crucible lines %+v:\n%s", item.CrucibleModLines, out)
	}
	if info, _ := DisplayItem(fixture.Items[3]); !info.AddCrucible {
		t.Fatal("a bow has no Add Crucible mod")
	}
	if info, _ := DisplayItem(pobHelmet); info.AddCrucible {
		t.Fatal("a helmet has Add Crucible mod")
	}
}

var parserGapNotables = []string{}

// TestAnointNotableSupportAgainstPoB: NotableDBControl.lua:246-256.
func TestAnointNotableSupportAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	var pob map[string]struct {
		Name        string
		Sd          []string
		Unsupported []bool
	}
	raw, err := os.ReadFile("../testdata/craft-ui/notable_support.json")
	if err != nil || json.Unmarshal(raw, &pob) != nil || len(pob) < 400 {
		t.Fatalf("REFUSING: testdata/craft-ui/notable_support.json: %v", err)
	}
	compared, differ := 0, 0
	for _, n := range AnointNotables() {
		want, ok := pob[strconv.FormatInt(n.ID, 10)]
		if !ok || !slices.Equal(want.Sd, n.Stats) {
			differ++
			continue
		}
		compared++
		if slices.Contains(parserGapNotables, n.Name) {
			if slices.Equal(want.Unsupported, n.Unsupported) {
				t.Errorf("%s now matches the pin: drop it from parserGapNotables", n.Name)
			}
			continue
		}
		if !slices.Equal(want.Unsupported, n.Unsupported) {
			t.Errorf("%s: unsupported %v, want %v (%q)", n.Name, n.Unsupported, want.Unsupported, n.Stats)
		}
	}
	if compared < 400 {
		t.Fatalf("only %d notables have the pin's stat text (%d differ)", compared, differ)
	}
}

// TestRareLikeUniqueCustomModSources: ItemsTab.lua:991-993, 3577-3588.
func TestRareLikeUniqueCustomModSources(t *testing.T) {
	initItemsTabData(t)
	text, ok := ItemDBText(ItemDBUnique, "Dread Captain's Cutlass, Ghostflame Blade")
	if !ok {
		t.Fatal("no Dread Captain's Cutlass in the DB")
	}
	choices, ok := ItemCustomModChoices(text)
	if !ok {
		t.Fatal("no choices")
	}
	ids := make([]string, 0, len(choices.Sources))
	for _, s := range choices.Sources {
		ids = append(ids, s.ID)
	}
	if want := []string{"ESSENCE", "VEILED", "CUSTOM"}; !slices.Equal(ids, want) {
		t.Errorf("sources %q, want %q", ids, want)
	}
	if !addCustomShown(ParseItem(text)) {
		t.Error("Add modifier... hidden on the cutlass")
	}
	subsume, ok := ItemDBText(ItemDBUnique, "Subsume the Source, Faithful Helmet")
	if !ok {
		t.Fatal("no Subsume the Source in the DB")
	}
	if addCustomShown(ParseItem(subsume)) {
		t.Error("Add modifier... shown on a rare-like unique without supportsCustomModifiers")
	}
}
