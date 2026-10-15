package calculator

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

type timelessTradeIDs struct {
	keystone []string
	devotion []string
}

//pob:block:src/Data/TimelessJewelData/LegionTradeIds.lua:4-69
var timelessJewelTradeIDs = map[int]timelessTradeIDs{
	1: {keystone: []string{"explicit.pseudo_timeless_jewel_doryani", "explicit.pseudo_timeless_jewel_xibaqua", "explicit.pseudo_timeless_jewel_ahuana"}},
	2: {keystone: []string{"explicit.pseudo_timeless_jewel_kaom", "explicit.pseudo_timeless_jewel_rakiata", "explicit.pseudo_timeless_jewel_akoya"}},
	3: {keystone: []string{"explicit.pseudo_timeless_jewel_asenath", "explicit.pseudo_timeless_jewel_nasima", "explicit.pseudo_timeless_jewel_balbala"}},
	4: {
		keystone: []string{"explicit.pseudo_timeless_jewel_avarius", "explicit.pseudo_timeless_jewel_dominus", "explicit.pseudo_timeless_jewel_maxarius"},
		devotion: []string{
			"explicit.stat_2566390555", "explicit.stat_2697019412", "explicit.stat_970844066", "explicit.stat_1724614884",
			"explicit.stat_3103189267", "explicit.stat_1910205563", "explicit.stat_1810368194", "explicit.stat_730530528",
			"explicit.stat_4235333770", "explicit.stat_3808469650", "explicit.stat_2830135449", "explicit.stat_2042813020",
			"explicit.stat_3293275880", "explicit.stat_2585926696", "explicit.stat_2803981661", "explicit.stat_1355915086",
		},
	},
	5: {keystone: []string{"explicit.pseudo_timeless_jewel_cadiro", "explicit.pseudo_timeless_jewel_victario", "explicit.pseudo_timeless_jewel_caspiro"}},
	6: {keystone: []string{"explicit.pseudo_timeless_jewel_vorana", "explicit.pseudo_timeless_jewel_uhtred", "explicit.pseudo_timeless_jewel_medved"}},
}

//pob:block:src/Classes/TreeTab.lua:2412-2418
var TimelessTradeTypes = []string{
	"Instant buyout", "Instant buyout and in person", "In person (online in league)", "In person (online)", "Any (includes offline)",
}

//pob:block:src/Classes/TreeTab.lua:2351-2353
var TimelessRealms = []string{"PC", "Sony", "Xbox"}

type TimelessTradeQuery struct {
	JewelType         int
	ConquerorType     int
	DevotionVariant1  int
	DevotionVariant2  int
	Seeds             []int
	SelIndex          int
	HighlightIndex    int
	SearchMore        bool
	TradeTypeIndex    int
	Realm             string
	League            string
	LastSearchStart   int
	LastSearchCount   int
	LastSearchPresent bool
}

type TimelessTrade struct {
	URL               string
	SelIndex          int
	HighlightIndex    int
	LastSearchStart   int
	LastSearchCount   int
	LastSearchPresent bool
}

type timelessTradeFilter struct {
	ID    string             `json:"id"`
	Value *timelessTradeSeed `json:"value,omitempty"`
}

type timelessTradeSeed struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

//pob:block:src/Classes/TreeTab.lua:2221-2333
func TimelessTradeURL(q TimelessTradeQuery) TimelessTrade {
	ids, ok := timelessJewelTradeIDs[q.JewelType]
	if !ok || len(q.Seeds) == 0 {
		return TimelessTrade{}
	}
	var startRow, endRow int
	if q.HighlightIndex > 0 && q.SelIndex > 0 && !q.SearchMore {
		startRow = min(q.SelIndex, q.HighlightIndex)
		endRow = max(q.SelIndex, q.HighlightIndex)
	} else {
		startRow = q.SelIndex
		if startRow <= 0 {
			startRow = 1
		}
		maxFilters := 10
		if q.SearchMore {
			maxFilters = 180
		}
		div := 1
		if q.ConquerorType == 1 {
			div = 3
		}
		endRow = startRow + maxFilters/div
	}
	seedCount := min(len(q.Seeds)-startRow, endRow-startRow) + 1
	if q.LastSearchPresent && q.LastSearchStart == startRow && q.LastSearchCount == seedCount {
		startRow = endRow + 1
		if startRow > len(q.Seeds) {
			return TimelessTrade{SelIndex: q.SelIndex, HighlightIndex: q.HighlightIndex, LastSearchStart: q.LastSearchStart, LastSearchCount: q.LastSearchCount, LastSearchPresent: true}
		}
		seedCount = min(len(q.Seeds)-startRow+1, seedCount)
		endRow = startRow + seedCount - 1
	}
	out := TimelessTrade{SelIndex: startRow, HighlightIndex: endRow, LastSearchStart: startRow, LastSearchCount: seedCount, LastSearchPresent: true}

	var conquerorTradeIDs []string
	if q.ConquerorType > 1 && q.ConquerorType-2 < len(ids.keystone) {
		conquerorTradeIDs = []string{ids.keystone[q.ConquerorType-2]}
	} else {
		conquerorTradeIDs = ids.keystone
	}
	var seedTrades []timelessTradeFilter
	for i := startRow; i <= startRow+seedCount-1; i++ {
		if i < 1 || i > len(q.Seeds) {
			continue
		}
		for _, id := range conquerorTradeIDs {
			seedTrades = append(seedTrades, timelessTradeFilter{ID: id, Value: &timelessTradeSeed{Min: q.Seeds[i-1], Max: q.Seeds[i-1]}})
		}
	}
	tradeTypes := []string{"securable", "available", "onlineleague", "online", "any"}
	tradeType := ""
	if q.TradeTypeIndex >= 1 && q.TradeTypeIndex <= len(tradeTypes) {
		tradeType = tradeTypes[q.TradeTypeIndex-1]
	}
	stats := []any{map[string]any{"filters": seedTrades, "type": "count", "value": map[string]int{"min": 1}}}
	if ids.devotion != nil {
		var devotionFilters []timelessTradeFilter
		for _, v := range []int{timelessDevotionID(q.DevotionVariant1), timelessDevotionID(q.DevotionVariant2)} {
			if v > 1 && v-2 < len(ids.devotion) {
				devotionFilters = append(devotionFilters, timelessTradeFilter{ID: ids.devotion[v-2]})
			}
		}
		if len(devotionFilters) > 0 {
			stats = append(stats, map[string]any{"filters": devotionFilters, "type": "and"})
		}
	}
	search := map[string]any{
		"query": map[string]any{"status": map[string]string{"option": tradeType}, "stats": stats},
		"sort":  map[string]string{"price": "asc"},
	}
	encoded, _ := json.Marshal(search) //nolint:errchkjson // plain maps and structs
	var q2 strings.Builder
	for _, b := range encoded {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			q2.WriteByte(b)
		} else {
			fmt.Fprintf(&q2, "%%%02X", b)
		}
	}
	realm := strings.ToLower(q.Realm)
	realmPath := realm + "/"
	if realm == "pc" || realm == "" {
		realmPath = ""
	}
	out.URL = "https://www.pathofexile.com/trade/search/" + realmPath + q.League + "/?q=" + q2.String()
	return out
}

var timelessItemHeads = map[int]struct {
	name, variants, line, conquered string
	leaders                         [3]string
}{
	1: {"Glorious Vanity", "Doryani (Corrupted Soul)|Xibaqua (Divine Flesh)|Ahuana (Immortal Ambition)", "Bathed in the blood of %d sacrificed in the name of %s", "Vaal", [3]string{"Doryani", "Xibaqua", "Ahuana"}},
	2: {"Lethal Pride", "Kaom (Strength of Blood)|Rakiata (Tempered by War)|Akoya (Chainbreaker)", "Commanded leadership over %d warriors under %s", "Karui", [3]string{"Kaom", "Rakiata", "Akoya"}},
	3: {"Brutal Restraint", "Asenath (Dance with Death)|Nasima (Second Sight)|Balbala (The Traitor)", "Denoted service of %d dekhara in the akhara of %s", "Maraketh", [3]string{"Asenath", "Nasima", "Balbala"}},
	5: {"Elegant Hubris", "Cadiro (Supreme Decadence)|Victario (Supreme Grandstanding)|Caspiro (Supreme Ostentation)", "Commissioned %d coins to commemorate %s", "Eternal Empire", [3]string{"Cadiro", "Victario", "Caspiro"}},
	6: {"Heroic Tragedy", "Vorana (Black Scythe Training)|Uhtred (Celestial Mathematics)|Medved (The Unbreaking Circle)", "Remembrancing %d songworthy deeds by the line of %s", "Kalguur", [3]string{"Vorana", "Uhtred", "Medved"}},
}

//pob:block:src/Classes/TimelessJewelListControl.lua:150-165
var timelessDevotionItemVariants = []string{
	"Totem Damage", "Brand Damage", "Channelling Damage", "Area Damage", "Elemental Damage", "Elemental Resistances",
	"Effect of non-Damaging Ailments", "Elemental Ailment Duration", "Duration of Curses", "Minion Attack and Cast Speed",
	"Minions Accuracy Rating", "Mana Regen", "Mana Cost (legacy)", "Non-Curse Aura Effect", "Defences from Shield",
	"Mana Cost Efficiency",
}

//pob:block:src/Classes/TimelessJewelListControl.lua:174-189
var timelessDevotionLines = []string{
	"4% increased Totem Damage per 10 Devotion", "4% increased Brand Damage per 10 Devotion",
	"Channelling Skills deal 4% increased Damage per 10 Devotion", "4% increased Area Damage per 10 Devotion",
	"4% increased Elemental Damage per 10 Devotion", "+2% to all Elemental Resistances per 10 Devotion",
	"3% increased Effect of non-Damaging Ailments on Enemies per 10 Devotion", "4% reduced Elemental Ailment Duration on you per 10 Devotion",
	"4% reduced Duration of Curses on you per 10 Devotion", "1% increased Minion Attack and Cast Speed per 10 Devotion",
	"Minions have +60 to Accuracy Rating per 10 Devotion", "Regenerate 0.6 Mana per Second per 10 Devotion",
	"1% reduced Mana Cost of Skills per 10 Devotion", "1% increased effect of Non-Curse Auras per 10 Devotion",
	"3% increased Defences from Equipped Shield per 10 Devotion", "3% increased Mana Cost Efficiency per 10 Devotion",
}

//pob:block:src/Classes/TimelessJewelListControl.lua:73-286
func TimelessResultItem(jewelType, conquerorType, devotion1, devotion2, seed int, total float64, socketLabel string, rolled *[2]int) string {
	if socketLabel == "" {
		socketLabel = "Unknown"
	}
	label := "[" + strconv.Itoa(seed) + "; " + luaNumber(total) + "; " + socketLabel + "]\n"
	variant := "1"
	if conquerorType != 1 {
		variant = strconv.Itoa(conquerorType-1) + "\n"
	}
	if jewelType == 4 {
		if rolled == nil {
			rolled = &[2]int{}
		}
		altVariant := timelessDevotionID(devotion1)
		if altVariant == 1 {
			altVariant = rolled[0]
			if altVariant == 0 {
				altVariant = 2 + rand.IntN(16) //nolint:gosec // PoB's m_random
			}
		}
		altVariant2 := timelessDevotionID(devotion2)
		if altVariant2 == 1 {
			altVariant2 = rolled[1]
			if altVariant2 == 0 {
				altVariant2 = 2 + rand.IntN(16) //nolint:gosec // PoB's m_random
			}
		}
		if altVariant == altVariant2 {
			altVariant = altVariant%16 + 2
		}
		rolled[0], rolled[1] = altVariant, altVariant2
		var b strings.Builder
		b.WriteString("Militant Faith " + label + "Timeless Jewel\nLeague: Legion\nLimited to: 1 Historic\nHas Alt Variant: true\nHas Alt Variant Two: true\n")
		for _, v := range []string{"Avarius (Power of Purpose)", "Dominus (Inner Conviction)", "Maxarius (Transcendence)"} {
			b.WriteString("Variant: " + v + "\n")
		}
		for _, v := range timelessDevotionItemVariants {
			b.WriteString("Variant: " + v + "\n")
		}
		b.WriteString("Selected Variant: " + variant + "\n")
		b.WriteString("Selected Alt Variant: " + strconv.Itoa(altVariant+2) + "\n")
		b.WriteString("Selected Alt Variant Two: " + strconv.Itoa(altVariant2+2) + "\n")
		b.WriteString("Radius: Large\nImplicits: 0\n")
		for i, name := range []string{"Avarius", "Dominus", "Maxarius"} {
			fmt.Fprintf(&b, "{variant:%d}Carved to glorify %d new faithful converted by High Templar %s\n", i+1, seed, name)
		}
		for i, line := range timelessDevotionLines {
			fmt.Fprintf(&b, "{variant:%d}%s\n", i+4, line)
		}
		b.WriteString("Passives in radius are Conquered by the Templars\nHistoric\n")
		return b.String()
	}
	head, ok := timelessItemHeads[jewelType]
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(head.name + " " + label + "Timeless Jewel\nLeague: Legion\nLimited to: 1 Historic\n")
	for _, v := range strings.Split(head.variants, "|") {
		b.WriteString("Variant: " + v + "\n")
	}
	b.WriteString("Selected Variant: " + variant + "\n")
	b.WriteString("Radius: Large\nImplicits: 0\n")
	for i, leader := range head.leaders {
		fmt.Fprintf(&b, "{variant:%d}"+head.line+"\n", i+1, seed, leader)
	}
	b.WriteString("Passives in radius are Conquered by the " + head.conquered + "\nHistoric\n")
	return b.String()
}
