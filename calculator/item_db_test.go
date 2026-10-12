package calculator

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/utils"
)

func gradeDBItem(item *Item) data.ItemDBGraded {
	return data.ItemDBGraded{
		Name:        item.Name,
		Type:        dbItemType(item),
		BaseName:    item.BaseName,
		Rarity:      string(item.Rarity),
		League:      item.League,
		Source:      item.Source,
		Level:       int(item.Requirements["level"]),
		StrMod:      int(item.Requirements["strMod"]),
		DexMod:      int(item.Requirements["dexMod"]),
		IntMod:      int(item.Requirements["intMod"]),
		PrimarySlot: item.getPrimarySlot(),
		Variants:    len(item.VariantList),
		Explicits:   len(item.ExplicitModLines),
	}
}

// TestItemDBAgainstPoB grades our parse of every unique and crafted rare template against PoB's (oracle/item_db.lua).
func TestItemDBAgainstPoB(t *testing.T) {
	type miss struct{ field, ours, pob string }
	byField := map[string][]string{}
	check := func(label string, ours, pob data.ItemDBGraded) {
		for _, m := range []miss{
			{"name", ours.Name, pob.Name}, {"type", ours.Type, pob.Type}, {"baseName", ours.BaseName, pob.BaseName},
			{"rarity", ours.Rarity, pob.Rarity}, {"league", ours.League, pob.League}, {"source", ours.Source, pob.Source},
			{"level", fmt.Sprint(ours.Level), fmt.Sprint(pob.Level)}, {"strMod", fmt.Sprint(ours.StrMod), fmt.Sprint(pob.StrMod)},
			{"dexMod", fmt.Sprint(ours.DexMod), fmt.Sprint(pob.DexMod)}, {"intMod", fmt.Sprint(ours.IntMod), fmt.Sprint(pob.IntMod)},
			{"primarySlot", ours.PrimarySlot, pob.PrimarySlot}, {"variants", fmt.Sprint(ours.Variants), fmt.Sprint(pob.Variants)},
			{"explicits", fmt.Sprint(ours.Explicits), fmt.Sprint(pob.Explicits)},
		} {
			if m.ours != m.pob {
				byField[m.field] = append(byField[m.field], fmt.Sprintf("%s: ours %q, PoB %q", label, m.ours, m.pob))
			}
		}
	}
	n := 0
	for _, e := range data.Uniques() {
		item := &Item{}
		item.parseRaw(data.SanitiseItemText(e.Raw), RarityUnique, true)
		if e.PoB == nil || item.Base == nil {
			if (e.PoB == nil) != (item.Base == nil) {
				byField["base"] = append(byField["base"], fmt.Sprintf("%s/%s: ours has base %v, PoB %v", e.Kind, item.Name, item.Base != nil, e.PoB != nil))
			}
			continue
		}
		n++
		check(e.Kind+"/"+e.PoB.Name, gradeDBItem(item), *e.PoB)
	}
	for _, e := range data.Rares() {
		item := &Item{}
		item.parseRaw(data.SanitiseItemText(e.Raw), RarityRare, true)
		if e.PoB == nil || item.Base == nil {
			continue
		}
		if item.Crafted {
			item = craftDBItem(item)
		}
		n++
		check("rare/"+e.PoB.Name, gradeDBItem(item), *e.PoB)
	}
	if n < 1000 {
		t.Fatalf("graded only %d DB items; the generated table did not load", n)
	}
	knownExplicits := []string{"fishing/Reefbane", "fishing/Song of the Sirens", "jewel/Transcendent Mind"}
	var left []string
	for _, l := range byField["explicits"] {
		known := false
		for _, k := range knownExplicits {
			if strings.HasPrefix(l, k+",") {
				known = true
			}
		}
		if !known {
			left = append(left, l)
		}
	}
	if len(byField["explicits"])-len(left) != len(knownExplicits) {
		t.Errorf("explicits: %d of the %d known parser gaps remain; drop the fixed ones from the list", len(byField["explicits"])-len(left), len(knownExplicits))
	}
	if left == nil {
		delete(byField, "explicits")
	} else {
		byField["explicits"] = left
	}
	fields := make([]string, 0, len(byField))
	for f := range byField {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		list := byField[f]
		t.Errorf("%s: %d of %d differ", f, len(list), n)
		for k, l := range list {
			if k >= 8 {
				break
			}
			t.Log("  " + l)
		}
	}
}

// TestItemDBFiltersAgainstPoB grades the DB lists' dropdowns and every filter case of testdata/items-ui/item_db_filters.json against PoB's ItemDBControl run on the same case.
func TestItemDBFiltersAgainstPoB(t *testing.T) {
	raw, err := os.ReadFile("../testdata/items-ui/item_db_filters.json")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_db_filters.json: %v", err)
	}
	var want struct {
		Cases []struct {
			DB          string  `json:"db"`
			Slot        int     `json:"slot"`
			Type        int     `json:"type"`
			League      int     `json:"league"`
			Requirement int     `json:"requirement"`
			Obtainable  int     `json:"obtainable"`
			Search      string  `json:"search"`
			SearchMode  int     `json:"searchMode"`
			Level       int     `json:"level"`
			Str         float64 `json:"str"`
			Dex         float64 `json:"dex"`
			Int         float64 `json:"int"`
		} `json:"cases"`
		Results  [][]string                        `json:"results"`
		UniqueDB struct{ Leagues, Types []string } `json:"uniqueDB"`
		RareDB   struct{ Leagues, Types []string } `json:"rareDB"`
	}
	if err := json.Unmarshal(raw, &want); err != nil || len(want.Cases) < 50 || len(want.Cases) != len(want.Results) {
		t.Fatalf("REFUSING: the fixture does not decode to cases and results: %v", err)
	}
	for _, db := range []struct {
		kind  ItemDBKind
		lists struct{ Leagues, Types []string }
	}{{ItemDBUnique, want.UniqueDB}, {ItemDBRare, want.RareDB}} {
		leagues, types := ItemDBLists(db.kind)
		if db.kind == ItemDBUnique && !slices.Equal(leagues, db.lists.Leagues) {
			t.Errorf("%s leagues:\n ours %q\n PoB  %q", db.kind, leagues, db.lists.Leagues)
		}
		if !slices.Equal(types, db.lists.Types) {
			t.Errorf("%s types:\n ours %q\n PoB  %q", db.kind, types, db.lists.Types)
		}
	}
	knownParser := map[string]bool{}
	knownSeen := map[string]bool{}
	drop := func(list []string, ref []string) []string {
		in := map[string]bool{}
		for _, n := range ref {
			in[n] = true
		}
		var out []string
		for _, n := range list {
			if knownParser[n] && !in[n] {
				knownSeen[n] = true
				continue
			}
			out = append(out, n)
		}
		return out
	}
	for k, c := range want.Cases {
		f := ItemDBFilter{Slot: c.Slot, Type: c.Type, League: c.League, Requirement: c.Requirement, Obtainable: c.Obtainable, Search: c.Search, SearchMode: c.SearchMode}
		var names []string
		for _, item := range FilterItemDB(ItemDBKind(c.DB), f, ItemDBContext{CharacterLevel: c.Level, Str: c.Str, Dex: c.Dex, Int: c.Int}) {
			names = append(names, item.Name)
		}
		pob := want.Results[k]
		names, pob = drop(names, pob), drop(pob, names)
		if !slices.Equal(names, pob) {
			ours := map[string]bool{}
			for _, n := range names {
				ours[n] = true
			}
			var onlyPoB []string
			onlyOurs := make([]string, 0, len(ours))
			for _, n := range pob {
				if !ours[n] {
					onlyPoB = append(onlyPoB, n)
				}
				delete(ours, n)
			}
			for n := range ours {
				onlyOurs = append(onlyOurs, n)
			}
			t.Errorf("case %d %+v: ours %d, PoB %d; only PoB %q; only ours %q", k, c, len(names), len(pob), head(onlyPoB), head(onlyOurs))
		}
	}
	if len(knownSeen) != len(knownParser) {
		t.Errorf("only %d of the %d known parser differences still show (%v); drop the fixed ones", len(knownSeen), len(knownParser), knownSeen)
	}
}

func head(s []string) []string {
	if len(s) > 5 {
		return s[:5]
	}
	return s
}

// TestItemDBTooltipAgainstPoB grades each DB item's dbMode tooltip against the pin's.
func TestItemDBTooltipAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/items-ui/item_db_tooltip.json.gz")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_db_tooltip.json.gz: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var want map[string][]string
	if err := json.NewDecoder(zr).Decode(&want); err != nil || len(want) < 1000 {
		t.Fatalf("REFUSING: the fixture does not decode to the DB's tooltips: %v", err)
	}
	dbField := regexp.MustCompile(`^(\^xFFFF30Variants?: |\^xFF5555Exclusive to: |\^xDD0022Not yet available|\^x88FFFF)`)
	fields := func(lines []string) []string {
		var out []string
		for _, l := range lines {
			if dbField.MatchString(l) {
				out = append(out, l)
			}
		}
		return out
	}
	collapse := func(lines []string) []string {
		var out []string
		for _, l := range lines {
			if strings.Contains(l, "Tip: ") {
				break
			}
			if l == "--" && len(out) > 0 && out[len(out)-1] == "--" {
				continue
			}
			out = append(out, l)
		}
		for len(out) > 0 && out[len(out)-1] == "--" {
			out = out[:len(out)-1]
		}
		return out
	}
	ctx := newBuildTooltipContext()
	seen, whole := 0, 0
	for _, kind := range []ItemDBKind{ItemDBUnique, ItemDBRare} {
		for _, item := range ItemDB(kind) {
			pob, ok := want[item.Name]
			if !ok {
				continue
			}
			seen++
			ours := ourTooltipLines(AddItemTooltip(item, ctx), string(item.Rarity))
			if o, p := fields(ours), fields(pob); !slices.Equal(o, p) {
				t.Errorf("%s DB fields:\n ours %q\n PoB  %q", item.Name, o, p)
			}
			if slices.Equal(collapse(ours), collapse(pob)) {
				whole++
			}
		}
	}
	if seen < 1500 {
		t.Fatalf("graded only %d of the fixture's %d tooltips", seen, len(want))
	}
	t.Logf("%d of %d DB tooltips match the pin line for line", whole, seen)
	if whole < itemDBTooltipFloor {
		t.Errorf("only %d DB tooltips match the pin line for line; the floor is %d", whole, itemDBTooltipFloor)
	}
}

const itemDBTooltipFloor = 1443

// TestItemDBTextAgainstPoB grades the item each DB item enters the build as.
func TestItemDBTextAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/items-ui/item_db_added.json.gz")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_db_added.json.gz: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var want map[string]struct {
		Name      string `json:"name"`
		Explicits int    `json:"explicits"`
		Quality   int    `json:"quality"`
		Raw       string `json:"raw"`
	}
	if err := json.NewDecoder(zr).Decode(&want); err != nil || len(want) < 1000 {
		t.Fatalf("REFUSING: the fixture does not decode to the DB's items: %v", err)
	}
	n, same, lines := 0, 0, 0
	for _, kind := range []ItemDBKind{ItemDBUnique, ItemDBRare} {
		for _, item := range ItemDB(kind) {
			pob, ok := want[item.Name]
			if !ok {
				continue
			}
			text, ok := ItemDBText(kind, item.Name)
			if !ok {
				t.Fatalf("%s: no text", item.Name)
			}
			back := ParseItem(text)
			quality := -1
			if back.Quality != nil {
				quality = *back.Quality
			}
			if back.Name != pob.Name || quality != pob.Quality {
				t.Errorf("%s: ours %q at quality %d; PoB %q at %d", item.Name, back.Name, quality, pob.Name, pob.Quality)
			}
			if len(back.ExplicitModLines) != pob.Explicits {
				lines++
				t.Logf("%s: %d explicit lines, PoB %d", item.Name, len(back.ExplicitModLines), pob.Explicits)
			}
			if text == pob.Raw {
				same++
			}
			n++
		}
	}
	if n < 1500 {
		t.Fatalf("only %d DB items graded", n)
	}
	t.Logf("%d of %d texts match the pin's BuildRaw byte for byte", same, n)
	if lines > itemDBTextLineCeiling {
		t.Errorf("%d DB items enter the build with another explicit line count than the pin's; the ceiling is %d", lines, itemDBTextLineCeiling)
	}
	if same < itemDBTextSameFloor {
		t.Errorf("only %d texts match the pin's byte for byte; the floor is %d", same, itemDBTextSameFloor)
	}
}

const (
	itemDBTextLineCeiling = 23
	itemDBTextSameFloor   = 690
)

// TestAddDBItem: ItemDBControl.lua:335-360: Ctrl+click adds and equips in the primary slot.
func TestAddDBItem(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	active := func(slot string) int {
		set, _ := setItems(build)
		return setSlotItemID(set, slot)
	}
	count := func() int { return len(build.Items.Items) }
	before := count()
	id, ok := AddDBItem(build, ItemDBUnique, "Kaom's Heart, Glorious Plate", true, false)
	if !ok || active("Body Armour") != id || count() != before+1 {
		t.Fatalf("Body Armour = %d, want %d (items %d -> %d)", active("Body Armour"), id, before, count())
	}
	if id, ok = AddDBItem(build, ItemDBUnique, "Berek's Grip, Two-Stone Ring", true, true); !ok || active("Ring 2") != id {
		t.Fatalf("Shift: Ring 2 = %d, want %d", active("Ring 2"), id)
	}
	ring1 := active("Ring 1")
	if id, ok = AddDBItem(build, ItemDBRare, "Armour Helmet, Giantslayer Helmet", false, false); !ok || active("Helmet") == id || active("Ring 1") != ring1 {
		t.Fatalf("a drop equipped %d", id)
	}
	build.SetUseSecondWeaponSet(true)
	if id, ok = AddDBItem(build, ItemDBUnique, "Doryani's Fist, Vaal Gauntlets", true, false); !ok || active("Gloves") != id {
		t.Fatalf("Gloves = %d, want %d", active("Gloves"), id)
	}
	if id, ok = AddDBItem(build, ItemDBUnique, "Lifesprig, Driftwood Wand", true, false); !ok || active("Weapon 1 Swap") != id {
		t.Fatalf("Weapon 1 Swap = %d, want %d", active("Weapon 1 Swap"), id)
	}
	before = count()
	if _, ok = AddDBItem(build, ItemDBUnique, "Watcher's Eye, Prismatic Jewel", true, false); !ok || count() != before+1 {
		t.Fatal("a jewel was not added")
	}
	if _, ok = AddDBItem(build, ItemDBUnique, "No Such Item, Nothing", true, false); ok {
		t.Fatal("an unknown name was added")
	}
}

// TestSortItemDBByPower: ItemDBControl.lua:239-281: sorting by a stat puts the item that raises it most first.
func TestSortItemDBByPower(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	misc := GetMiscCalculator(build)
	life, ok := powerStatByName("Life")
	if !ok {
		t.Fatal("REFUSING: no Life power stat")
	}
	names := []string{"Abyssus, Ezomyte Burgonet", "Kaom's Heart, Glorious Plate", "Watcher's Eye, Prismatic Jewel", "Rumi's Concoction, Granite Flask"}
	list := make([]*Item, 0, len(names))
	for _, n := range names {
		item := ItemDBItem(ItemDBUnique, n)
		if item == nil {
			t.Fatalf("REFUSING: %s is not in the DB", n)
		}
		list = append(list, item)
	}
	powers := misc.SortItemDB(list, life)
	if list[0].Name != "Kaom's Heart, Glorious Plate" || powers[0] <= powers[1] {
		t.Fatalf("first %s at %v, then %v", list[0].Name, powers[0], powers[1])
	}
	want := powerStatValue(misc.Calc(&CalcOverride{RepSlotName: "Body Armour", RepItem: list[0]}), life)
	if powers[0] != want {
		t.Fatalf("Kaom's Heart power %v, the Body Armour what-if %v", powers[0], want)
	}
	if last := len(list) - 1; list[last].Name != "Watcher's Eye, Prismatic Jewel" || !math.IsInf(powers[last], -1) {
		t.Fatalf("last %s at %v", list[last].Name, powers[last])
	}
}

// TestItemSkillTooltipsAgainstPoB grades the granted-skill child tooltips.
func TestItemSkillTooltipsAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/items-ui/item_skill_tooltips.json.gz")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_skill_tooltips.json.gz: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var want map[string][][]string
	if err := json.NewDecoder(zr).Decode(&want); err != nil || len(want) < 100 {
		t.Fatalf("REFUSING: the fixture does not decode to child tooltips: %v", err)
	}
	ctx := newBuildTooltipContext()
	lines := func(tt ItemTooltip) []string {
		var out []string
		for _, l := range tt.Lines {
			if l.Separator {
				out = append(out, "--")
			} else {
				out = append(out, l.Text)
			}
		}
		return out
	}
	seen, same, empty := 0, 0, 0
	for name, pob := range want {
		item := ItemDBItem(ItemDBUnique, name)
		if item == nil {
			t.Errorf("%s: not in our DB", name)
			continue
		}
		ours := GrantedSkillTooltips(AddItemTooltip(item, ctx), ctx)
		for k, p := range pob {
			seen++
			switch {
			case k >= len(ours) || len(ours[k]) == 0:
				empty++
				t.Logf("%s #%d: none drawn; PoB starts %q", name, k, p[0])
			case slices.Equal(lines(ItemTooltip{Lines: ours[k]}), p):
				same++
			default:
				t.Logf("%s #%d:\n ours %q\n PoB  %q", name, k, lines(ItemTooltip{Lines: ours[k]}), p)
			}
		}
	}
	t.Logf("%d of %d child tooltips match the pin line for line; %d skills without a gem drew none", same, seen, empty)
	if same < itemSkillTooltipFloor {
		t.Errorf("only %d child tooltips match the pin; the floor is %d", same, itemSkillTooltipFloor)
	}
}

const itemSkillTooltipFloor = 229

func newBuildTooltipContext() ItemTooltipContext {
	return ItemTooltipContext{CharacterLevel: 1, ClassName: "Scion", ShowFlavourText: true, DBMode: true, Stats: ItemTooltipStats{Str: 20, Dex: 20, Int: 20}}
}

// TestDeleteSocketedJewelAgainstPoB deletes each jewel socketed in an allocated socket of the active tree.
func TestDeleteSocketedJewelAgainstPoB(t *testing.T) {
	initItemsTabData(t)
	raw, err := os.ReadFile("../testdata/items-ui/item_delete_tree.json")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_delete_tree.json: %v", err)
	}
	var want map[string]struct {
		Before []int64
		After  map[string][]int64
	}
	if err := json.Unmarshal(raw, &want); err != nil || len(want) < 2 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	cases := 0
	for file, w := range want {
		build := loadCorpusBuild(t, file)
		before := sortedNodeIDs(build.Build.PassiveNodes)
		if !slices.Equal(before, w.Before) {
			t.Fatalf("%s: our tree loads %d nodes, the pin's %d", file, len(before), len(w.Before))
		}
		for id, after := range w.After {
			cases++
			build := loadCorpusBuild(t, file)
			n, _ := strconv.Atoi(id)
			if !DeleteItems(build, []int{n}) {
				t.Fatalf("%s: item %d not deleted", file, n)
			}
			got := sortedNodeIDs(build.Build.PassiveNodes)
			if !slices.Equal(got, after) {
				t.Errorf("%s delete %s: ours drops %v and keeps %v that the pin drops", file, id, setMinus(after, got), setMinus(got, after))
			}
		}
	}
	if cases < 20 {
		t.Fatalf("graded only %d deletes", cases)
	}
}

func sortedNodeIDs(in []int64) []int64 {
	out := slices.Clone(in)
	slices.Sort(out)
	return out
}

func setMinus(a, b []int64) []int64 {
	var out []int64
	for _, x := range a {
		if !slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

// TestMutateModAgainstPoB toggles the first Foulborn line of every unique DB item that has one.
func TestMutateModAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/items-ui/item_mutate.json.gz")
	if err != nil {
		t.Fatalf("REFUSING: testdata/items-ui/item_mutate.json.gz: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	var want map[string]struct{ Before, Mutated, Back []string }
	if err := json.NewDecoder(zr).Decode(&want); err != nil || len(want) < 100 {
		t.Fatalf("REFUSING: the fixture does not decode: %v", err)
	}
	explicits := func(text string) []string {
		item := ParseItem(text)
		var out []string
		for _, ml := range item.ExplicitModLines {
			if item.checkModLineVariant(ml) {
				out = append(out, utils.Ternary(ml.Mutated, "{mutated}", "")+ml.Line)
			}
		}
		return out
	}
	toggle := func(text string, mutated bool) (string, bool) {
		info, ok := DisplayItem(text)
		if !ok {
			return text, false
		}
		k := slices.IndexFunc(info.Mods, func(m DisplayModLine) bool { return m.Mutable && (!mutated || m.Mutated) })
		if k < 0 {
			return text, false
		}
		out, err := EditItem(text, ItemEdit{Kind: "mutate", Index: k})
		return out, err == nil
	}
	same, seen := 0, 0
	for name, w := range want {
		seen++
		text, ok := ItemDBText(ItemDBUnique, name)
		if !ok {
			t.Errorf("%s: not in our DB", name)
			continue
		}
		before := explicits(text)
		mutated, ok := toggle(text, false)
		if !ok {
			t.Logf("%s: no Foulborn line to toggle", name)
			continue
		}
		back, _ := toggle(mutated, true)
		if slices.Equal(before, w.Before) && slices.Equal(explicits(mutated), w.Mutated) && slices.Equal(explicits(back), w.Back) {
			same++
		} else {
			t.Logf("%s:\n mutated ours %q\n         PoB  %q\n back    ours %q\n         PoB  %q", name, explicits(mutated), w.Mutated, explicits(back), w.Back)
		}
	}
	t.Logf("%d of %d Foulborn toggles match the pin both ways", same, seen)
	if same < mutateFloor {
		t.Errorf("only %d Foulborn toggles match the pin; the floor is %d", same, mutateFloor)
	}
}

const mutateFloor = 233

// TestShownConditionalSlots: ItemsTab.lua:166-174: Ring 3 shows with an additional ring slot (The Unseen Hand, "+1 Ring Slot"), the Grafts only on a 3.27 tree.
func TestShownConditionalSlots(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, "0280-elementalist-fireball.xml")
	if got := GetMiscCalculator(build).ShownConditionalSlots(); len(got) != 0 {
		t.Fatalf("0280 shows %q", got)
	}
	build.AllocateNodes([]int64{43902})
	if got := GetMiscCalculator(build).ShownConditionalSlots(); !slices.Equal(got, []string{"Ring 3"}) {
		t.Fatalf("with The Unseen Hand: %q", got)
	}
}

// TestDropItemOnSlotAndSet: ItemSlotControl.lua:114-130: a dragged item drops only on a slot it fits; a DB item enters the build first.
func TestDropItemOnSlotAndSet(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	n := len(build.Items.Items)
	if DropItemOnSlot(build, "Helmet", 0, ItemDBUnique, "Kaom's Heart, Glorious Plate") || len(build.Items.Items) != n {
		t.Fatal("a body armour dropped on the Helmet")
	}
	if !DropItemOnSlot(build, "Body Armour", 0, ItemDBUnique, "Kaom's Heart, Glorious Plate") || len(build.Items.Items) != n+1 {
		t.Fatal("Kaom's Heart did not drop on Body Armour")
	}
	set, _ := setItems(build)
	kaom := setSlotItemID(set, "Body Armour")
	if text, _ := buildItemText(build, kaom); !strings.Contains(text, "Kaom's Heart") {
		t.Fatalf("Body Armour holds %d", kaom)
	}
	ring := setSlotItemID(set, "Ring 1")
	if ring == 0 || !DropItemOnSlot(build, "Ring 2", ring, "", "") {
		t.Fatalf("ring %d did not drop on Ring 2", ring)
	}
	set, _ = setItems(build)
	if setSlotItemID(set, "Ring 2") != ring || len(build.Items.Items) != n+1 {
		t.Fatal("a build item was added again or not equipped")
	}
	other := 1
	if build.Items.ActiveItemSet == 1 {
		other = 2
	}
	activeHelmet := setSlotItemID(set, "Helmet")
	if !EquipItemInSet(build, other, 0, ItemDBUnique, "Abyssus, Ezomyte Burgonet", false) {
		t.Fatal("Abyssus did not go to the other set")
	}
	if text, _ := buildItemText(build, build.SetSlotItemID(other, "Helmet")); !strings.Contains(text, "Abyssus") {
		t.Fatalf("set %d Helmet: %q", other, text)
	}
	if set, _ = setItems(build); setSlotItemID(set, "Helmet") != activeHelmet {
		t.Fatal("the active set's Helmet changed")
	}
}

// TestClusterNotableAffixTooltip: ItemsTab.lua:860-914: a cluster jewel notable's affix entry shows the node's name.
func TestClusterNotableAffixTooltip(t *testing.T) {
	initItemsTabData(t)
	base, _ := CraftItemText(2, "", "Large Cluster Jewel")
	info, _ := DisplayItem(base)
	if info.Cluster == nil || len(info.Cluster.Skills) == 0 {
		t.Fatal("REFUSING: the crafted large cluster jewel has no skills")
	}
	text, ok := craftClusterText(base, info.Cluster.Skills[0].ID, 8)
	if !ok {
		t.Fatal("REFUSING: the cluster skill did not apply")
	}
	controls, ok := ItemAffixControls(text)
	if !ok {
		t.Fatal("REFUSING: no affix controls")
	}
	found := false
	for _, c := range controls.Controls {
		for _, s := range c.Series {
			if len(s.ModList) != 1 || !strings.HasPrefix(s.Label, "1 Added Passive Skill is ") {
				continue
			}
			name := strings.TrimPrefix(s.Label, "1 Added Passive Skill is ")
			if !slices.Contains(s.Tooltip, "^xF8E6CA"+name) || !slices.Contains(s.Tooltip, "^7"+name+" can appear on:") || !slices.Contains(s.Tooltip, colorMagic+"Large Cluster Jewel:") {
				t.Fatalf("%s: %q", name, s.Tooltip)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("no notable entry among the affix dropdowns")
	}
}

// TestSharedItemSets: SharedItemSetListControl.lua:76-93 / ItemSetListControl.lua:93-106.
func TestSharedItemSets(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	shared, ok := ShareItemSet(build, build.Items.ActiveItemSet)
	if !ok || len(shared.Slots) < 5 {
		t.Fatalf("shared %+v", shared)
	}
	if tip := SharedItemSetTooltip(shared); len(tip) != len(shared.Slots) || !strings.HasPrefix(tip[0], "^7") {
		t.Fatalf("tooltip %q", tip)
	}
	sets, items := len(build.Items.ItemSets), len(build.Items.Items)
	id := ImportSharedItemSet(build, shared)
	if len(build.Items.ItemSets) != sets+1 || len(build.Items.Items) != items+len(shared.Slots) {
		t.Fatalf("sets %d -> %d, items %d -> %d", sets, len(build.Items.ItemSets), items, len(build.Items.Items))
	}
	for slot, text := range shared.Slots {
		got, ok := buildItemText(build, build.SetSlotItemID(id, slot))
		if !ok || ParseItem(got).Name != ParseItem(text).Name {
			t.Fatalf("%s: %q", slot, got)
		}
	}
}

// TestSharedItems: SharedItemListControl.lua:44-53.
func TestSharedItems(t *testing.T) {
	initItemsTabData(t)
	build := loadCorpusBuild(t, itemList0339)
	id := build.AddItem("Rarity: RARE\nShared Shell\nVaal Regalia\nQuality: 5\n+50 to maximum Life")
	shared, ok := ShareItem(build, id, "", "")
	if !ok || ParseItem(shared).Quality == nil || *ParseItem(shared).Quality != 5 {
		t.Fatalf("build item shared as %q", shared)
	}
	db, ok := ShareItem(build, 0, ItemDBUnique, "Kaom's Heart, Glorious Plate")
	if !ok || ParseItem(db).Name != "Kaom's Heart, Glorious Plate" || ParseItem(db).Quality == nil {
		t.Fatalf("DB item shared as %q", db)
	}
	if _, ok := ShareItem(build, 0, ItemDBUnique, "No Such Item"); ok {
		t.Fatal("shared a missing DB item")
	}
	items := len(build.Items.Items)
	added, ok := AddSharedItem(build, shared, "")
	got, _ := buildItemText(build, added)
	if !ok || len(build.Items.Items) != items+1 || *ParseItem(got).Quality != 20 {
		t.Fatalf("dropped on the list: %v %q", ok, got)
	}
	if _, ok := AddSharedItem(build, shared, "Helmet"); ok || len(build.Items.Items) != items+1 {
		t.Fatal("a body armour went into the helmet slot")
	}
	added, ok = AddSharedItem(build, shared, "Body Armour")
	if !ok || func() bool { set, _ := setItems(build); return setSlotItemID(set, "Body Armour") != added }() {
		t.Fatalf("dropped on Body Armour: %v %d", ok, added)
	}
}
