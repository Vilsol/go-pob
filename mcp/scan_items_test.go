package mcp

import (
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
)

// staleCraftedHelmet is a PoB-crafted helmet whose explicit lines were edited away from its affix records, as
// imported builds often carry: its life line does not match the IncreasedLife7 record.
const staleCraftedHelmet = `Rarity: RARE
Test Mask
Torturer's Mask
Crafted: true
Prefix: {range:0.746}IncreasedLife7
Prefix: None
Prefix: None
Suffix: None
Suffix: None
Suffix: None
Item Level: 86
Quality: 20
Implicits: 0
+110 to maximum Life
+33% to Fire Resistance`

// slotText is the text of the item equipped in slot on the handle's current revision.
func slotText(t *testing.T, s *Server, h, slot string) string {
	t.Helper()
	hh, _ := s.get(h)
	b, _ := hh.parseRev(hh.cur)
	for _, e := range equippedItems(b) {
		if e.slot == slot {
			return e.text
		}
	}
	t.Fatalf("nothing in %s", slot)
	return ""
}

// addedLines checks that every line of before survives in after, in order and unchanged, and returns the lines after
// adds.
func addedLines(t *testing.T, before, after string) []string {
	t.Helper()
	old, next := strings.Split(before, "\n"), strings.Split(after, "\n")
	var added []string
	i := 0
	for _, l := range next {
		if i < len(old) && l == old[i] {
			i++
			continue
		}
		added = append(added, l)
	}
	if i != len(old) {
		t.Fatalf("line %q changed or lost:\n%s\n---\n%s", old[i], before, after)
	}
	return added
}

// TestScanAffixApplyKeepsOtherMods: applying an affix row adds that mod's lines to the item, fills a prefix or suffix
// into its first empty affix record and leaves every other line byte-identical, also on a PoB-crafted item whose
// lines no longer match its affix records.
func TestScanAffixApplyKeepsOtherMods(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "text": staleCraftedHelmet, "slot": "Helmet"})})
	before := slotText(t, s, h, "Helmet")
	list := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"affixes"}, "slots": []string{"Helmet"}, "stat": "Life", "per_slot": 5}))
	if len(list) == 0 {
		t.Fatal("no affix rows for the helmet")
	}
	for _, r := range list {
		call(t, s, "scan_build", map[string]any{"handle": h, "apply": r["id"]})
		want := before
		if src, rest, _ := strings.Cut(strings.SplitN(r["id"].(string), "|", 3)[2], ":"); src != "MASTER" {
			modID, rng, _ := strings.Cut(rest, "|")
			want = strings.Replace(before, r["type"].(string)+": None", r["type"].(string)+": {range:"+rng+"}"+modID, 1)
		}
		added := addedLines(t, want, slotText(t, s, h, "Helmet"))
		call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": 1})
		if len(added) == 0 || len(added) > 2 {
			t.Fatalf("%v added %q", r["id"], added)
		}
	}
}

// craftedStaff has a rolled added-physical prefix and a bench-crafted increased-physical prefix: two prefixes, its
// crafted mod used, and the LocalPhysicalDamagePercent group taken.
const craftedStaff = `Rarity: RARE
Test Staff
Maelstrom Staff
Crafted: true
Prefix: {range:0.669}LocalAddedPhysicalDamageTwoHand8
Prefix: None
Prefix: None
Suffix: None
Suffix: None
Suffix: None
Item Level: 84
Quality: 20
Implicits: 1
{tags:block}+25% Chance to Block Attack Damage
{tags:physical_damage,damage,physical,attack}{prefix}Adds 37 to 70 Physical Damage
{tags:physical_damage,damage,physical,attack}{crafted}{range:0.5}(100-129)% increased Physical Damage`

func modGroup(item *calculator.Item, id string) string {
	src, modID, _ := strings.Cut(id, ":")
	if src == "MASTER" {
		i, _ := strconv.Atoi(strings.TrimPrefix(modID, "master:"))
		return data.MasterMods()[i].Group
	}
	for _, m := range slices.Concat(affixPool(item), delveMods(item)) {
		if m.ID == modID {
			return m.Group
		}
	}
	return ""
}

// TestScanAffixesRespectItemRules: a crafted mod counts toward the prefixes and uses the item's one bench craft, a
// group the item has is never offered again, and open_affixes reports what is open.
func TestScanAffixesRespectItemRules(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "text": craftedStaff, "slot": "Weapon 1"})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"affixes"}, "slots": []string{"Weapon 1"}, "per_slot": 5})
	open := res["open_affixes"].([]any)[0].(map[string]any)
	if open["open_prefixes"] != 1.0 || open["open_suffixes"] != 3.0 {
		t.Fatalf("open_affixes %v, want 1 prefix and 3 suffixes open", open)
	}
	list := rows(res)
	if len(list) == 0 {
		t.Fatalf("no rows: %v", res)
	}
	item := calculator.ParseItem(craftedStaff)
	have := countAffixes(item).groups
	prefixes := 0
	for _, r := range list {
		id := strings.Split(r["id"].(string), "|")[2]
		if strings.HasPrefix(id, "MASTER:") {
			t.Fatalf("bench craft offered on an item with a crafted mod: %v", r)
		}
		if g := modGroup(item, id); g == "" || have[g] {
			t.Fatalf("row %v adds group %q the item has", r["id"], g)
		}
		if r["type"] == "Prefix" {
			prefixes++
		}
	}
	if prefixes == 0 {
		t.Fatalf("no prefix row for the one open prefix: %v", list)
	}
}

// TestCountAffixesModsNotLines: a two-line mod is one affix, and a crafted mod counts as its bench type.
func TestCountAffixesModsNotLines(t *testing.T) {
	_ = testServer(t)
	flask := "Rarity: MAGIC\nTest Flask\nDivine Life Flask\nItem Level: 60\nImplicits: 0\n{tags:flask}{prefix}21% reduced Amount Recovered\n{tags:flask}{prefix}Instant Recovery when on Low Life\n{suffix}Grants Immunity to Bleeding for 4 seconds if used while Bleeding\n{suffix}Grants Immunity to Corrupted Blood for 4 seconds if used while affected by Corrupted Blood"
	if a := countAffixes(calculator.ParseItem(flask)); a.pre != 1 || a.suf != 1 {
		t.Fatalf("flask counts %d/%d, want 1/1", a.pre, a.suf)
	}
	if a := countAffixes(calculator.ParseItem(craftedStaff)); a.pre != 2 || a.suf != 0 || a.crafted != 1 {
		t.Fatalf("staff counts %d/%d crafted %d, want 2/0 crafted 1", a.pre, a.suf, a.crafted)
	}
}

// TestCountAffixesByRolledValue: an item's lines count as the mods whose rolls hold their values, so two lines that
// only fit a hybrid's wording stay two mods and a line worded like a prefix and a suffix counts as the one it rolls
// as; no candidate repeats a group the item has.
func TestCountAffixesByRolledValue(t *testing.T) {
	_ = testServer(t)
	gloves := "Rarity: RARE\nTest Grip\nSpiked Gloves\nItem Level: 80\nImplicits: 1\n30% increased Melee Damage\n70% increased Armour\n+100 to maximum Life\n+30% to Lightning Resistance"
	item := calculator.ParseItem(gloves)
	a := countAffixes(item)
	armour := ""
	for _, m := range affixPool(item) {
		if m.Type == "Prefix" && len(m.Lines) == 1 && affixTemplate(m.Lines[0]) == affixTemplate("70% increased Armour") {
			armour = m.Group
		}
	}
	if a.pre != 2 || a.suf != 1 || !a.groups["IncreasedLife"] || armour == "" || !a.groups[armour] {
		t.Fatalf("gloves count %d/%d groups %v, want 2/1 with IncreasedLife and %q", a.pre, a.suf, a.groups, armour)
	}
	for _, c := range affixCandidates(gloves, item, a, defaultRoll) {
		if c.group == "IncreasedLife" || c.group == armour {
			t.Fatalf("%s repeats group %s", c.id, c.group)
		}
	}
	body := "Rarity: RARE\nTest Garb\nTorturer Garb\nItem Level: 86\nImplicits: 0\n85% increased Evasion and Energy Shield\n+28 to maximum Energy Shield\n{crafted}+24 to maximum Mana"
	if a := countAffixes(calculator.ParseItem(body)); a.crafted != 1 || a.pre != 3 {
		t.Fatalf("body armour counts %d prefixes, %d crafted; want 3 and 1", a.pre, a.crafted)
	}
	shield := "Rarity: RARE\nTest Kite\nArchon Kite Shield\nItem Level: 84\nImplicits: 0\n+99 to maximum Life\n7% additional Physical Damage Reduction"
	if a := countAffixes(calculator.ParseItem(shield)); a.pre != 1 || a.suf != 1 {
		t.Fatalf("shield counts %d/%d, want 1/1", a.pre, a.suf)
	}
}

// TestAffixCandidatesItemLevelAndBench: no candidate, bench crafts included, needs a higher item level than the
// item's; a "Can have up to 3 Crafted Modifiers" item keeps bench crafts after its first crafted mod.
func TestAffixCandidatesItemLevelAndBench(t *testing.T) {
	_ = testServer(t)
	low := "Rarity: RARE\nTest Ring\nIron Ring\nItem Level: 30\nImplicits: 0\n+20 to maximum Life\n"
	item := calculator.ParseItem(low)
	cands := affixCandidates(low, item, countAffixes(item), defaultRoll)
	bench := 0
	for _, c := range cands {
		src, id, _ := strings.Cut(strings.Split(c.id, "|")[0], ":")
		lvl := 0
		if src == "MASTER" {
			i, _ := strconv.Atoi(strings.TrimPrefix(id, "master:"))
			lvl = data.MasterMods()[i].Level
			bench++
		} else {
			for _, m := range affixPool(item) {
				if m.ID == id {
					lvl = m.Level
				}
			}
		}
		if lvl == 0 || lvl > 30 {
			t.Fatalf("%s needs level %d on an item level 30 ring", c.id, lvl)
		}
	}
	if bench == 0 {
		t.Fatal("no bench craft offered on a ring without one")
	}
	multi := "Rarity: RARE\nTest Ring\nIron Ring\nItem Level: 84\nImplicits: 0\n+20 to maximum Life\n{crafted}Can have up to 3 Crafted Modifiers\n{crafted}+(29-35)% to Fire Resistance\n"
	item = calculator.ParseItem(multi)
	a := countAffixes(item)
	if a.crafted != 2 || a.craftLimit != 3 {
		t.Fatalf("multimod ring crafted %d of %d", a.crafted, a.craftLimit)
	}
	if !slices.ContainsFunc(affixCandidates(multi, item, a, defaultRoll), func(c affixCandidate) bool { return strings.HasPrefix(c.id, "MASTER:") }) {
		t.Fatal("no bench craft on a multimod ring with room")
	}
}

// TestScanAffixRollChoice: roll picks where an added mod rolls, avg by default, and tier picks a lower tier; the
// committed item carries that roll.
func TestScanAffixRollChoice(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	gains := map[string]map[string]float64{}
	for _, roll := range []string{"min", "", "max"} {
		args := map[string]any{"handle": h, "parts": []string{"affixes"}, "stat": "Life", "per_slot": 5}
		if roll != "" {
			args["roll"] = roll
		}
		for _, r := range rows(call(t, s, "scan_build", args)) {
			id := r["id"].(string)
			base := id[:strings.LastIndex(id, "|")]
			if gains[base] == nil {
				gains[base] = map[string]float64{}
			}
			gains[base][roll] = r["gain"].(float64)
			if want := map[string]string{"min": "|0", "": "|0.5", "max": "|1"}[roll]; !strings.HasSuffix(id, want) {
				t.Fatalf("roll %q row id %s", roll, id)
			}
		}
	}
	ranged := 0
	for id, g := range gains {
		if len(g) == 3 && g["min"] < g[""] && g[""] < g["max"] {
			ranged++
		} else if len(g) == 3 && !(g["min"] <= g[""] && g[""] <= g["max"]) {
			t.Fatalf("%s: min %v avg %v max %v", id, g["min"], g[""], g["max"])
		}
	}
	if ranged == 0 {
		t.Fatalf("no row gains more at a higher roll: %v", gains)
	}
	top := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"affixes"}, "stat": "Life", "roll": "max"}))[0]
	slot := top["slot"].(string)
	tier1 := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"affixes"}, "stat": "Life", "tier": 1, "slots": []string{slot}}))
	tier3 := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"affixes"}, "stat": "Life", "tier": 3, "slots": []string{slot}}))
	if len(tier1) == 0 || len(tier3) == 0 || !strings.Contains(tier3[0]["name"].(string), "(T3)") || tier3[0]["gain"].(float64) >= tier1[0]["gain"].(float64) {
		t.Fatalf("tier 1 %v, tier 3 %v", tier1, tier3)
	}
	before := slotText(t, s, h, slot)
	call(t, s, "scan_build", map[string]any{"handle": h, "apply": top["id"]})
	added := addedLines(t, before, slotText(t, s, h, slot))
	if !slices.ContainsFunc(added, func(l string) bool { return strings.Contains(l, "{range:1}") }) {
		t.Fatalf("%v added %q without its max roll", top["id"], added)
	}
}

// TestScanBuildRanksByWorstCase: with Full DPS counting more totems than the limit, rows rank by their gain with the
// count held to the limit, clean rows before unusable ones, and no clean row that gains nothing there is kept.
func TestScanBuildRanksByWorstCase(t *testing.T) {
	s := testServer(t)
	h := fullDPSTotems(t, s, 1)
	setGemCount(t, s, h, 1, 0, totemLimit(t, s, h)+2)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"uniques"}, "stat": "FullDPS", "slots": []string{"Weapon 2", "Helmet", "Amulet"}, "per_slot": 5})
	if res["worst_case"] == nil {
		t.Fatalf("no worst_case note: %v", res)
	}
	list := rows(res)
	if len(list) == 0 {
		t.Fatalf("no rows: %v", res)
	}
	last, unusable := math.Inf(1), false
	for _, r := range list {
		if r["unusable"] != nil && !unusable {
			last, unusable = math.Inf(1), true
		}
		w, ok := r["worst_gain"].(float64)
		if !ok || (w <= 0 && !unusable) || w > last || (unusable && r["unusable"] == nil) {
			t.Fatalf("row %v out of worst-case order after %v", r, last)
		}
		last = w
	}
}

// TestRankWorstDropsRowsWithoutWorstGain: a row that gains only with favourable conditions is dropped, but for an
// unusable one, kept last so its reason shows, and a part left without rows says so.
func TestRankWorstDropsRowsWithoutWorstGain(t *testing.T) {
	list := []map[string]any{
		{"id": "a", "part": "uniques", "gain": 10.0, "worst_gain": 0.0},
		{"id": "b", "part": "uniques", "gain": 5.0, "worst_gain": 4.0},
		{"id": "c", "part": "affixes", "gain": 6.0},
		{"id": "d", "part": "flasks", "gain": 9.0, "worst_gain": -1.0},
		{"id": "e", "part": "uniques", "gain": 8.0, "worst_gain": -2.0, "unusable": "You do not meet the Dexterity requirement", dirtyKey: true},
	}
	empty := map[string]string{}
	got := rankWorst(list, empty)
	ids := make([]string, 0, len(got))
	for _, r := range got {
		ids = append(ids, r["id"].(string))
	}
	if !slices.Equal(ids, []string{"c", "b", "e"}) || empty["flasks"] == "" || empty["uniques"] != "" {
		t.Fatalf("kept %v, empty parts %v", ids, empty)
	}
}

// TestScanBuildEmptyPartSaysWhy: a part with no rows names its reason.
func TestScanBuildEmptyPartSaysWhy(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"flasks"}, "stat": "Str"})
	if len(rows(res)) != 0 {
		t.Fatalf("a flask raises Str: %v", rows(res))
	}
	why, _ := res["empty_parts"].(map[string]any)["flasks"].(string)
	if !strings.Contains(why, "raises Str") {
		t.Fatalf("empty flask scan without a reason: %v", res)
	}
}

// TestScanClusterSwapAllocatesItsNotables: with one notable of a jewel left unallocated, a swap can move the jewel's
// notables onto that node; after applying a row the jewel's notables, the new one included, sit on allocated nodes,
// and a row that allocates nodes for that says how many.
func TestScanClusterSwapAllocatesItsNotables(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("7.xml")})["handle"].(string)
	const socket = 2491
	hh, _ := s.get(h)
	b, _ := hh.parseRev(hh.cur)
	var drop int64
	before, _ := jewelSubgraph(b, socket)
	for id, n := range before {
		if n.Name == "Feed the Fury" && slices.Contains(b.Build.PassiveNodes, id) {
			drop = id
		}
	}
	if drop == 0 {
		t.Fatal("Feed the Fury is not allocated on the jewel in 2491")
	}
	b.DeallocateNodes([]int64{drop})
	if _, e := s.commit(hh, b); e != nil {
		t.Fatal(e.Message)
	}
	rev := hh.cur
	list := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"clusters"}, "per_slot": 5}))
	needs := 0
	for _, r := range list {
		if !strings.HasPrefix(r["id"].(string), "cluster|2491|") {
			continue
		}
		if r["points"] != nil {
			needs++
		}
		call(t, s, "scan_build", map[string]any{"handle": h, "apply": r["id"]})
		b, _ := hh.parseRev(hh.cur)
		call(t, s, "revert_build", map[string]any{"handle": h, "to_revision": rev})
		sub, _ := jewelSubgraph(b, socket)
		var text string
		for _, sock := range activeSpec(b).Sockets {
			if sock.NodeID == socket {
				for _, it := range b.Items.Items {
					if it.ID == sock.ItemID {
						text = it.Text
					}
				}
			}
		}
		for _, l := range strings.Split(text, "\n") {
			n, ok := strings.CutPrefix(strings.TrimSpace(tagRe.Replace(l)), notablePrefix)
			if !ok || n == "Feed the Fury" {
				continue
			}
			if !slices.ContainsFunc(slices.Collect(maps.Keys(sub)), func(id int64) bool { return sub[id].Name == n && slices.Contains(b.Build.PassiveNodes, id) }) {
				t.Fatalf("%v: %s is not allocated after the swap (row %v)", r["id"], n, r)
			}
		}
	}
	if needs == 0 {
		t.Fatalf("no row allocates nodes; the fixture no longer exercises it: %v", list)
	}
}

// recordWand is a PoB-crafted wand whose spell critical strike line carries an older wording than its suffix record.
const recordWand = `Rarity: RARE
Test Wand
Profane Wand
Crafted: true
Prefix: None
Prefix: {range:0.7}IncreasedManaWeapon7
Prefix: None
Suffix: {range:0.7}SpellCriticalStrikeChance5
Suffix: None
Suffix: None
Item Level: 84
Implicits: 1
14% increased Cast Speed
94% increased Critical Strike Chance for Spells
+96 to maximum Mana`

// recordPlate is a PoB-crafted plate whose hybrid armour-and-life record has its two lines apart; the life line alone
// reads as a life prefix.
const recordPlate = `Rarity: RARE
Test Plate
Astral Plate
Crafted: true
Prefix: {range:0.5}LocalBaseArmourAndLife4
Prefix: None
Prefix: None
Suffix: {range:0.5}FireResist4
Suffix: None
Suffix: None
Item Level: 84
Implicits: 0
+120 to Armour
+26% to Fire Resistance
+36 to maximum Life`

// TestCountAffixesReadsRecords: a PoB-crafted item's Prefix:/Suffix: records decide its mods, so a line worded unlike
// its record still holds its group and a hybrid's lines stay one mod wherever they sit.
func TestCountAffixesReadsRecords(t *testing.T) {
	_ = testServer(t)
	item := calculator.ParseItem(recordWand)
	a := countAffixes(item)
	if a.pre != 1 || a.suf != 1 || !a.groups["SpellCriticalStrikeChance"] {
		t.Fatalf("wand counts %d/%d groups %v, want 1/1 with SpellCriticalStrikeChance", a.pre, a.suf, a.groups)
	}
	for _, c := range affixCandidates(recordWand, item, a, defaultRoll) {
		if c.group == "SpellCriticalStrikeChance" {
			t.Fatalf("%s repeats the wand's spell critical strike suffix", c.id)
		}
	}
	a = countAffixes(calculator.ParseItem(recordPlate))
	if a.pre != 1 || a.suf != 1 || a.groups["IncreasedLife"] || !a.groups["LocalBaseArmourAndLife"] {
		t.Fatalf("plate counts %d/%d groups %v, want 1/1 with LocalBaseArmourAndLife only", a.pre, a.suf, a.groups)
	}
}

// anointedAmulet carries its anoint in the implicit lines after a plain implicit; PoB files a crafted implicit line
// as an enchant and writes enchants first.
const anointedAmulet = `Rarity: RARE
Test Heart
Jade Amulet
Item Level: 78
Implicits: 2
+22 to Dexterity
{crafted}Allocates Whispers of Doom
+25 to Dexterity
+18 to Intelligence
+39 to maximum Mana
+20% to Fire Resistance`

// TestAffixCandidatesOnAnointedAmulet: an anointed amulet with an open prefix gets candidates, each adding its lines
// and leaving the item's own lines where they were.
func TestAffixCandidatesOnAnointedAmulet(t *testing.T) {
	_ = testServer(t)
	item := calculator.ParseItem(anointedAmulet)
	a := countAffixes(item)
	cands := affixCandidates(anointedAmulet, item, a, defaultRoll)
	if !slices.ContainsFunc(cands, func(c affixCandidate) bool { return c.typ == "Prefix" }) {
		t.Fatalf("no prefix candidate for %d open prefixes", a.limit-a.pre)
	}
	for _, c := range cands {
		if added := addedLines(t, anointedAmulet, c.text); len(added) == 0 {
			t.Fatalf("%s adds nothing", c.id)
		}
	}
}

// TestAffixCandidatesOfferDelveMods: an item's Delve drop-restricted mods (PoB's "Delve Drop-restricted" modifier
// source) are candidates only with delve:true, in their affix type, can be applied, and count once added.
func TestAffixCandidatesOfferDelveMods(t *testing.T) {
	_ = testServer(t)
	plate := "Rarity: RARE\nTest Plate\nAstral Plate\nItem Level: 84\nImplicits: 0\n+100 to maximum Life\n"
	item := calculator.ParseItem(plate)
	a := countAffixes(item)
	isDelve := func(c affixCandidate) bool { return strings.HasPrefix(c.id, "DELVE:") }
	if slices.ContainsFunc(affixCandidates(plate, item, a, defaultRoll), isDelve) {
		t.Fatal("Delve drop-restricted mods are offered without delve:true")
	}
	withDelve := defaultRoll
	withDelve.delve = true
	i := slices.IndexFunc(affixCandidates(plate, item, a, withDelve), func(c affixCandidate) bool { return strings.HasPrefix(c.id, "DELVE:FireDamageTakenDelve|") })
	if i < 0 {
		t.Fatal("no Delve candidate for the plate's open affixes with delve:true")
	}
	c := affixCandidates(plate, item, a, withDelve)[i]
	if c.typ != "Prefix" || !strings.Contains(c.label, "Delve") || !affixAllowed(item, "DELVE", "FireDamageTakenDelve") {
		t.Fatalf("Delve candidate %+v", c)
	}
	if after := countAffixes(calculator.ParseItem(c.text)); after.pre != a.pre+1 || !after.groups[c.group] {
		t.Fatalf("after adding %s the plate counts %d prefixes, groups %v", c.id, after.pre, after.groups)
	}
}

// TestScanOffersFossilModsOnlyWithDelve: a base's fossil-only mods (Subterranean, of the Underground) sit in PoB's
// prefix and suffix lists but only a fossil craft rolls them, so the affix part offers them only with delve:true. (No
// fossil mod rolls a resistance, so the resistance planner never meets one.)
func TestScanOffersFossilModsOnlyWithDelve(t *testing.T) {
	_ = testServer(t)
	texts := map[string]string{
		"Weapon 1":    "Rarity: RARE\nTest Wand\nProphecy Wand\nItem Level: 84\nImplicits: 0\n+10 to maximum Life\n",
		"Body Armour": "Rarity: RARE\nTest Plate\nAstral Plate\nItem Level: 84\nImplicits: 0\n+100 to maximum Life\n",
		"Helmet":      "Rarity: RARE\nTest Helm\nEternal Burgonet\nItem Level: 84\nImplicits: 0\n+10 to maximum Life\n",
		"Gloves":      "Rarity: RARE\nTest Gloves\nTitan Gauntlets\nItem Level: 84\nImplicits: 0\n+10 to maximum Life\n",
		"Boots":       "Rarity: RARE\nTest Boots\nTitan Greaves\nItem Level: 84\nImplicits: 0\n+10 to maximum Life\n",
		"Ring 1":      "Rarity: RARE\nTest Ring\nTwo-Stone Ring\nItem Level: 84\nImplicits: 0\n+10 to maximum Life\n",
		"Amulet":      "Rarity: RARE\nTest Amulet\nOnyx Amulet\nItem Level: 84\nImplicits: 0\n+10 to maximum Life\n",
		"Belt":        "Rarity: RARE\nTest Belt\nLeather Belt\nItem Level: 84\nImplicits: 0\n+10 to maximum Life\n",
	}
	offered := 0
	for slot, text := range texts {
		item := calculator.ParseItem(text)
		choices, _ := calculator.ItemCustomModChoices(modChoicesText(text))
		fossil := fossilMods(choices)
		isFossil := func(c affixCandidate) bool {
			id := strings.SplitN(strings.SplitN(c.id, "|", 2)[0], ":", 2)[1]
			return fossil[id] && !strings.HasPrefix(c.id, "DELVE:")
		}
		if i := slices.IndexFunc(affixCandidates(text, item, countAffixes(item), defaultRoll), isFossil); i >= 0 {
			t.Fatalf("%s: fossil mod offered without delve:true: %+v", slot, affixCandidates(text, item, countAffixes(item), defaultRoll)[i])
		}
		withDelve := defaultRoll
		withDelve.delve = true
		for _, c := range affixCandidates(text, item, countAffixes(item), withDelve) {
			if isFossil(c) {
				offered++
				if !strings.Contains(c.label, "Delve fossil") {
					t.Fatalf("%s: fossil candidate unlabelled: %+v", slot, c)
				}
			}
		}
	}
	if offered == 0 {
		t.Fatal("no fossil candidate with delve:true: the check above proves nothing")
	}
}

// TestScanAffixesFillsFlasks: a default affix scan offers magic flask affixes for a flask's open affix, and counts a
// magic flask's affixes by its name, so a flask named with a prefix and a suffix has none open.
func TestScanAffixesFillsFlasks(t *testing.T) {
	s := testServer(t)
	h := importFixture(t, s)
	full := calculator.ParseItem("Rarity: MAGIC\nDabbler's Quicksilver Flask of the Eagle\nItem Level: 84\nImplicits: 0\n29% reduced Duration\n25% increased effect\n14% increased Attack Speed during Flask effect\n")
	if a := countAffixes(full); a.open("Prefix") || a.open("Suffix") {
		t.Fatalf("Dabbler's Quicksilver Flask of the Eagle counts %dP/%dS", a.pre, a.suf)
	}
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "slot": "Flask 5", "text": "Rarity: MAGIC\nDiamond Flask\nItem Level: 84\nImplicits: 0\n"})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"affixes"}})
	var flaskRows, filled []string
	for _, r := range rows(res) {
		switch r["slot"] {
		case "Flask 5":
			flaskRows = append(flaskRows, r["name"].(string))
		case "Flask 1", "Flask 3", "Flask 4":
			filled = append(filled, r["id"].(string))
		}
	}
	if len(flaskRows) == 0 {
		t.Fatalf("no affix row for the empty Diamond Flask: %v", rows(res))
	}
	if len(filled) > 0 {
		t.Fatalf("affix rows for flasks whose names show both affixes: %v", filled)
	}
}
