package calculator2

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// BuySimilarListed is LISTED_STATUS_OPTIONS: the Listed dropdown's labels and the status each searches.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:30-40
var BuySimilarListed = []struct{ Label, APIValue string }{
	{"Instant Buyout", "securable"},
	{"Instant Buyout & In Person", "available"},
	{"In Person (Online)", "online"},
	{"Any", "any"},
}

// BuySimilarDefence is one defenceEntries row: an armour value the item has and its trade filter key.
type BuySimilarDefence struct {
	Label    string
	TradeKey string
	Value    float64
}

// BuySimilarDefences is the popup's defence rows for a non-unique item with an armour base.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:310-329
func BuySimilarDefences(item *Item) []BuySimilarDefence {
	defenceEntries := []BuySimilarDefence{}
	isUnique := item.Rarity == "UNIQUE" || item.Rarity == "RELIC"
	if isUnique || item.ArmourData == nil || item.Base == nil || item.Base.Armour == nil {
		return defenceEntries
	}
	defences := []struct{ key, label, tradeKey string }{
		{"Armour", "Armour", "ar"},
		{"Evasion", "Evasion", "ev"},
		{"EnergyShield", "Energy Shield", "es"},
		{"Ward", "Runic Ward", "ward"},
	}
	for _, def := range defences {
		if val := item.ArmourData[def.key]; val > 0 {
			defenceEntries = append(defenceEntries, BuySimilarDefence{Label: def.label, Value: val, TradeKey: def.tradeKey})
		}
	}
	return defenceEntries
}

func tradeItemType(item *Item) string {
	if item == nil {
		return ""
	}
	if item.Type != "" {
		return item.Type
	}
	if item.Base != nil {
		return item.Base.Type
	}
	return ""
}

// TradeCategory is the trade site's category for an item in slotName, "" for none.
//
//pob2:src/Classes/TradeHelpers.lua:317-363
func TradeCategory(slotName string, item *Item) string {
	if slotName == "" {
		return ""
	}
	itemType := tradeItemType(item)
	if weaponSlotPattern.MatchString(slotName) {
		if itemType == "" {
			return "weapon.one"
		}
		switch {
		case itemType == "Shield":
			return "armour.shield"
		case itemType == "Focus":
			return "armour.focus"
		case itemType == "Buckler":
			return "armour.buckler"
		case itemType == "Quiver":
			return "armour.quiver"
		case itemType == "Bow":
			return "weapon.bow"
		case itemType == "Crossbow":
			return "weapon.crossbow"
		case itemType == "Talisman":
			return "weapon.talisman"
		case itemType == "Staff" && item.Base != nil && item.Base.SubType == "Warstaff":
			return "weapon.warstaff"
		case itemType == "Staff":
			return "weapon.staff"
		case itemType == "Two Hand Sword":
			return "weapon.twosword"
		case itemType == "Two Hand Axe":
			return "weapon.twoaxe"
		case itemType == "Two Hand Mace":
			return "weapon.twomace"
		case itemType == "Fishing Rod":
			return "weapon.rod"
		case itemType == "One Hand Sword":
			return "weapon.onesword"
		case itemType == "Spear":
			return "weapon.spear"
		case itemType == "Flail":
			return "weapon.flail"
		case itemType == "One Hand Axe":
			return "weapon.oneaxe"
		case itemType == "One Hand Mace":
			return "weapon.onemace"
		case itemType == "Sceptre":
			return "weapon.sceptre"
		case itemType == "Wand":
			return "weapon.wand"
		case itemType == "Dagger":
			return "weapon.dagger"
		case itemType == "Claw":
			return "weapon.claw"
		case strings.Contains(itemType, "Two Hand"):
			return "weapon.twomelee"
		case strings.Contains(itemType, "One Hand"):
			return "weapon.one"
		}
		return ""
	}
	switch {
	case slotName == "Body Armour":
		return "armour.chest"
	case slotName == "Helmet":
		return "armour.helmet"
	case slotName == "Gloves":
		return "armour.gloves"
	case slotName == "Boots":
		return "armour.boots"
	case slotName == "Amulet":
		return "accessory.amulet"
	case slotName == "Ring 1" || slotName == "Ring 2" || slotName == "Ring 3":
		return "accessory.ring"
	case slotName == "Belt":
		return "accessory.belt"
	case strings.Contains(slotName, "Jewel"):
		return "jewel"
	case strings.Contains(slotName, "Flask 1"):
		return "flask.life"
	case strings.Contains(slotName, "Flask 2"):
		return "flask.mana"
	case strings.Contains(slotName, "Charm"):
		return "flask"
	}
	return ""
}

var weaponSlotPattern = regexp.MustCompile(`^Weapon \d`)

// TradeCategoryLabel is the popup's Category label.
//
//pob2:src/Classes/TradeHelpers.lua:368-372
func TradeCategoryLabel(item *Item) string {
	if item == nil || item.Base == nil {
		return "Item"
	}
	if item.Base.Type != "" {
		return item.Base.Type
	}
	if item.Type != "" {
		return item.Type
	}
	return "Item"
}

// BuySimilarOptions is the popup's controls: the league, the Listed index (1-based), the base type box, the item level
// boxes and, per defence row, whether it is checked and its boxes; a nil box is empty.
type BuySimilarOptions struct {
	League   string
	Listed   int
	BaseType bool
	IlvlMin  *int
	IlvlMax  *int
	Defences []BuySimilarDefenceFilter
	Mods     []BuySimilarModFilter
}

type BuySimilarDefenceFilter struct {
	Min     *int
	Max     *int
	Checked bool
}

// BuySimilarModFilter is a mod row's box and its Min and Max boxes, one per mod entry.
type BuySimilarModFilter struct {
	Min     *float64
	Max     *float64
	Checked bool
}

// B64GzipEncode is the string gzipped, then Base64URL encoded.
//
//pob2:src/Classes/TradeHelpers.lua:603-608
func B64GzipEncode(str string) string {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write([]byte(str))
	_ = w.Close()
	return strings.NewReplacer("+", "-", "/", "_").Replace(base64.StdEncoding.EncodeToString(buf.Bytes()))
}

// BuySimilarURL is the trade search URL for the popup's selections over its mod entries. The realm is PoE2's, which
// the popup does not let change.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:44-195
//pob2:block:src/Classes/CompareBuySimilar.lua:385-396
func BuySimilarURL(item *Item, slotName string, modEntries []*BuySimilarModEntry, o BuySimilarOptions) string {
	isUnique := item.Rarity == "UNIQUE" || item.Rarity == "RELIC"
	realm := "poe2"
	league := o.League
	if league == "" || league == "Loading..." {
		league = "Standard"
	}
	hostName := "https://www.pathofexile.com/"

	listedAPIValue := "available"
	if o.Listed >= 1 && o.Listed <= len(BuySimilarListed) {
		listedAPIValue = BuySimilarListed[o.Listed-1].APIValue
	} else if o.Listed == 0 {
		listedAPIValue = BuySimilarListed[0].APIValue
	}

	queryTable := map[string]any{
		"status": map[string]any{"option": listedAPIValue},
		"stats":  []any{map[string]any{"type": "and", "filters": []any{}}},
	}
	queryFilters := map[string]any{}

	if isUnique {
		title := item.Title
		if title == "" {
			title = item.Name
		}
		queryTable["name"] = foulbornPrefix.ReplaceAllString(title, "")
		queryTable["type"] = item.BaseName
	} else {
		if categoryStr := TradeCategory(slotName, item); categoryStr != "" {
			queryFilters["type_filters"] = map[string]any{"filters": map[string]any{"category": map[string]any{"option": categoryStr}}}
		}
		if o.BaseType {
			queryTable["type"] = item.BaseName
		}
		if o.IlvlMin != nil || o.IlvlMax != nil {
			ilvlFilter := map[string]any{}
			if o.IlvlMin != nil {
				ilvlFilter["min"] = *o.IlvlMin
			}
			if o.IlvlMax != nil {
				ilvlFilter["max"] = *o.IlvlMax
			}
			queryFilters["misc_filters"] = map[string]any{"filters": map[string]any{"ilvl": ilvlFilter}}
		}
		armourFilters := map[string]any{}
		for i, def := range BuySimilarDefences(item) {
			if i >= len(o.Defences) || !o.Defences[i].Checked {
				continue
			}
			filter := map[string]any{}
			if v := o.Defences[i].Min; v != nil {
				filter["min"] = *v
			}
			if v := o.Defences[i].Max; v != nil {
				filter["max"] = *v
			}
			if len(filter) > 0 {
				armourFilters[def.TradeKey] = filter
			}
		}
		if len(armourFilters) > 0 {
			queryFilters["equipment_filters"] = map[string]any{"filters": armourFilters}
		}
	}

	// Mod filters
	for i, entry := range modEntries {
		var row BuySimilarModFilter
		if i < len(o.Mods) {
			row = o.Mods[i]
		}
		getFilter := func(tradeID string) map[string]any {
			filter := map[string]any{"id": tradeID}
			switch {
			case entry.IsOption:
				// an empty Lua table encodes as an array
				value := map[string]any{}
				if entry.Value != nil {
					value["min"], value["max"] = *entry.Value, *entry.Value
					filter["value"] = value
				} else {
					filter["value"] = []any{}
				}
			case entry.Value != nil:
				minVal, maxVal := row.Min, row.Max
				if entry.Invert != nil && *entry.Invert {
					minVal, maxVal = maxVal, minVal
					if minVal != nil {
						minVal = floatp(-*minVal)
					}
					if maxVal != nil {
						maxVal = floatp(-*maxVal)
					}
				}
				value := map[string]any{}
				if minVal != nil {
					value["min"] = *minVal
				}
				if maxVal != nil {
					value["max"] = *maxVal
				}
				if len(value) > 0 {
					filter["value"] = value
				}
			}
			return filter
		}
		if !row.Checked {
			continue
		}
		stats := queryTable["stats"].([]any)
		if len(entry.TradeIDs) == 1 {
			// 1 id entries are added to the stat filters section
			first := stats[0].(map[string]any)
			first["filters"] = append(first["filters"].([]any), getFilter(entry.TradeIDs[0]))
		} else if len(entry.TradeIDs) > 1 {
			// ambiguous entries are added as a separate count filter
			countFilter := map[string]any{"type": "count", "value": map[string]any{"min": 1}, "filters": []any{}}
			for _, tradeID := range entry.TradeIDs {
				countFilter["filters"] = append(countFilter["filters"].([]any), getFilter(tradeID))
			}
			queryTable["stats"] = append(stats, countFilter)
		}
	}

	if len(queryFilters) > 0 {
		queryTable["filters"] = queryFilters
	}

	queryJSON, _ := json.Marshal(queryTable)
	url := hostName + "trade2/search"
	if realm != "" && realm != "pc" {
		url += "/" + realm
	}
	var encodedLeague strings.Builder
	for _, c := range []byte(league) {
		if isAlnum(c) || c == '-' || c == '.' || c == '_' || c == '~' {
			encodedLeague.WriteByte(c)
		} else {
			fmt.Fprintf(&encodedLeague, "%%%02X", c)
		}
	}
	url += "/" + encodedLeague.String()
	url += "/" + B64GzipEncode(string(queryJSON))
	return url
}

var foulbornPrefix = regexp.MustCompile(`^Foulborn\s+`)

func isAlnum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// BuySimilarModEntry is one mod row of the popup: the formatted lines of the mods it sums, their source type, the trade
// ids it searches (more than one when the line is ambiguous), whether the trade site has an option for it, whether
// the site wants its value negated (nil when no stat description said), and its value.
type BuySimilarModEntry struct {
	Value          *float64
	Invert         *bool
	FormattedLines []string
	TradeIDs       []string
	Type           string
	IsOption       bool
}

// existingStats is every trade id the site has and its text, lowered; a repeated id keeps its last text.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:12-18
var existingStats = sync.OnceValue(func() map[string]string {
	existing := map[string]string{}
	for _, cat := range tradeStats() {
		for _, entry := range cat.Entries {
			existing[entry.ID] = asciiLower(entry.Text)
		}
	}
	return existing
})

// BuySimilarAddModEntries is the popup's mod rows for item's enchant, implicit and explicit lines; formatModLine is
// itemLib.formatModLine outside dbMode.
//
//pob2:src/Classes/CompareBuySimilar.lua:200-281
//pob2:block:src/Classes/CompareBuySimilar.lua:301-308
func BuySimilarAddModEntries(item *Item, formatModLine func(*ModLine) (string, bool)) []*BuySimilarModEntry {
	var modEntries []*BuySimilarModEntry
	// this adds a single aggregated entry for matching stats (e.g. transformed flat dmg mods) which
	// avoids issues with confusing results. mods with different types are not summed as e.g.
	// implicit and explicit mods are separate in the search. options are also avoided as they don't
	// represent values that can be added combined
	insertOrAddToExisting := func(entry *BuySimilarModEntry) {
		for _, existingFilter := range modEntries {
			// check if all result trade ids are equal
			sameHashes := len(entry.TradeIDs) > 0 && slices.Equal(entry.TradeIDs, existingFilter.TradeIDs)
			if sameHashes && existingFilter.Type == entry.Type {
				if entry.Value != nil {
					value := *entry.Value
					if !sameLuaBool(entry.Invert, existingFilter.Invert) {
						value = -value
					}
					existing := 0.0
					if existingFilter.Value != nil {
						existing = *existingFilter.Value
					}
					existingFilter.Value = floatp(existing + value)
				}
				existingFilter.FormattedLines = append(existingFilter.FormattedLines, entry.FormattedLines[0])
				return
			}
		}
		modEntries = append(modEntries, entry)
	}
	modTypeSources := []struct {
		list []*ModLine
		typ  string
	}{
		{item.EnchantModLines, "enchant"},
		{item.ImplicitModLines, "implicit"},
		{item.ExplicitModLines, "explicit"},
	}
	for _, source := range modTypeSources {
		for _, modLine := range source.list {
			if !item.CheckModLineVariant(modLine) {
				continue
			}
			// remove unsupported data. the formatting of unsupported
			// mods is confusing here
			copied := *modLine
			copied.Extra = ""
			formatted, ok := formatModLine(&copied)
			if !ok {
				continue
			}
			// Use range-resolved text for matching
			resolvedLine := copied.Line
			if copied.Range != nil {
				resolvedLine = applyRange(copied.Line, *copied.Range, nil, copied.ValueScalar, nil, item.modSource)
			}
			// check option first, because even if we match a line via the descriptors, the trade id formatting is different for options. e.g.: explicit.stat_345345|33
			tradeID, value := findTradeIDOption(resolvedLine, source.typ)
			entry := &BuySimilarModEntry{
				// this array will always start with one line, but if multiple mods are
				// aggregated together it will contain the original mod lines for each
				FormattedLines: []string{formatted},
				Type:           source.typ,
				IsOption:       tradeID != "",
				Invert:         new(false),
				Value:          value,
			}
			if tradeID != "" {
				entry.TradeIDs = []string{tradeID}
			} else {
				resultHashes, value, invert := findTradeHash(resolvedLine)
				// convert hashes to string ids
				var resultIDs []string
				exactMatch := false
				tradeLine := asciiLower(strings.ReplaceAll(resolvedLine, "\n", " "))
				existing := existingStats()
				for _, hash := range resultHashes {
					id := fmt.Sprintf("%s.stat_%d", source.typ, hash)
					text, ok := existing[id]
					if ok && text == tradeLine {
						// Prefer literal trade text over alternate numeric forms, e.g. Instant Recovery.
						if !exactMatch {
							resultIDs = nil
							value, invert = nil, new(false)
							exactMatch = true
						}
						resultIDs = append(resultIDs, id)
					} else if ok && !exactMatch {
						resultIDs = append(resultIDs, id)
					}
				}
				entry.TradeIDs = resultIDs
				entry.Value = value
				entry.Invert = invert
			}
			insertOrAddToExisting(entry)
		}
	}
	return modEntries
}

// sameLuaBool is a == b over Lua's nil, false and true.
func sameLuaBool(a, b *bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
