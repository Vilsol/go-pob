package calculator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
)

// BuySimilarListed is LISTED_STATUS_OPTIONS: the Listed dropdown's labels and the status each searches.
//
//pob:block:src/Classes/CompareBuySimilar.lua:34-39
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
//pob:block:src/Classes/CompareBuySimilar.lua:314-333
func BuySimilarDefences(item *Item) []BuySimilarDefence {
	defenceEntries := []BuySimilarDefence{}
	isUnique := item.Rarity == RarityUnique || item.Rarity == RarityRelic
	if isUnique || item.ArmourData == nil || item.Base == nil || item.Base.Armour == nil {
		return defenceEntries
	}
	defences := []struct{ key, label, tradeKey string }{
		{"Armour", "Armour", "ar"},
		{"Evasion", "Evasion", "ev"},
		{"EnergyShield", "Energy Shield", "es"},
		{"Ward", "Ward", "ward"},
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

var tradeWeaponSlotRe = regexp.MustCompile(`^Weapon \d`)

// TradeCategory is the trade site's category for an item in slotName, "" for none.
//
//pob:src/Classes/TradeHelpers.lua:351-387
func TradeCategory(slotName string, item *Item) string {
	if slotName == "" {
		return ""
	}
	itemType := tradeItemType(item)
	if tradeWeaponSlotRe.MatchString(slotName) {
		if itemType == "" {
			return "weapon.one"
		}
		switch {
		case itemType == "Shield":
			return "armour.shield"
		case itemType == "Quiver":
			return "armour.quiver"
		case itemType == "Bow":
			return "weapon.bow"
		case itemType == "Staff":
			return "weapon.staff"
		case itemType == "Two Handed Sword":
			return "weapon.twosword"
		case itemType == "Two Handed Axe":
			return "weapon.twoaxe"
		case itemType == "Two Handed Mace":
			return "weapon.twomace"
		case itemType == "Fishing Rod":
			return "weapon.rod"
		case itemType == "One Handed Sword":
			return "weapon.onesword"
		case itemType == "One Handed Axe":
			return "weapon.oneaxe"
		case itemType == "One Handed Mace":
			return "weapon.onemace"
		case itemType == "Sceptre":
			return "weapon.sceptre"
		case itemType == "Wand":
			return "weapon.wand"
		case itemType == "Dagger":
			return "weapon.dagger"
		case itemType == "Claw":
			return "weapon.claw"
		case strings.Contains(itemType, "Two Handed"):
			return "weapon.twomelee"
		case strings.Contains(itemType, "One Handed"):
			return "weapon.one"
		}
		return "weapon"
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
	case strings.Contains(slotName, "Abyssal"):
		return "jewel.abyss"
	case strings.Contains(slotName, "Jewel"):
		return "jewel"
	case strings.Contains(slotName, "Flask"):
		return "flask"
	}
	return ""
}

// TradeCategoryLabel is the popup's Category label.
//
//pob:src/Classes/TradeHelpers.lua:392-396
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

// BuySimilarOptions is the popup's controls: the realm's API id, the league, the Listed index (1-based), the base type
// box, the item level boxes and, per defence and mod row, whether it is checked and its boxes; a nil box is empty.
type BuySimilarOptions struct {
	Realm    string
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

// tradeURLEncode is urlEncode: every byte but [A-Za-z0-9_.~-] as %XX.
//
//pob:src/Modules/Common.lua:1024-1029
func tradeURLEncode(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

var tradeFoulbornPrefixRe = regexp.MustCompile(`^Foulborn\s+`)
var tradeTrailingSpaceRe = regexp.MustCompile(`\s+$`)
var tradeTrailingIDRe = regexp.MustCompile(`^(.*?)\s+\d+$`)

// BuySimilarURL is the trade search URL for the popup's selections over its mod entries.
//
//pob:src/Classes/CompareBuySimilar.lua:47-206
func BuySimilarURL(item *Item, slotName string, modEntries []*BuySimilarModEntry, o BuySimilarOptions) string {
	isUnique := item.Rarity == RarityUnique || item.Rarity == RarityRelic
	realm := o.Realm
	if realm == "" {
		realm = "pc"
	}
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

	query := map[string]any{
		"status": map[string]any{"option": listedAPIValue},
		"stats":  []any{map[string]any{"type": "and", "filters": []any{}}},
	}
	queryTable := map[string]any{"query": query, "sort": map[string]any{"price": "asc"}}
	queryFilters := map[string]any{}

	if isUnique {
		title := item.Title
		if title == "" {
			title = item.Name
		}
		tradeName := tradeFoulbornPrefixRe.ReplaceAllString(title, "")
		// only strip a trailing numeric identifier to avoid e.g. including
		// timeless jewel ids or other numbers appended to the item name.
		tradeName = tradeTrailingSpaceRe.ReplaceAllString(tradeName, "")
		if m := tradeTrailingIDRe.FindStringSubmatch(tradeName); m != nil {
			tradeName = m[1]
		}
		query["name"] = tradeName
		query["type"] = item.BaseName
		if item.Foulborn {
			queryFilters["misc_filters"] = map[string]any{"filters": map[string]any{"foulborn_item": map[string]any{"option": "true"}}}
		}
	} else {
		if categoryStr := TradeCategory(slotName, item); categoryStr != "" {
			queryFilters["type_filters"] = map[string]any{"filters": map[string]any{"category": map[string]any{"option": categoryStr}}}
		}
		if o.BaseType {
			query["type"] = item.BaseName
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
			queryFilters["armour_filters"] = map[string]any{"filters": armourFilters}
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
				// timeless jewels use a min max range despite matching as an option
				var value map[string]any
				switch {
				case entry.OptionValue == nil:
					value = map[string]any{}
				case strings.Contains(tradeID, "timeless"):
					value = map[string]any{"min": entry.OptionValue, "max": entry.OptionValue}
				default:
					value = map[string]any{"option": entry.OptionValue}
				}
				if len(value) == 0 {
					// an empty Lua table encodes as an array
					filter["value"] = []any{}
				} else {
					filter["value"] = value
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
		stats := query["stats"].([]any)
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
			query["stats"] = append(stats, countFilter)
		}
	}

	// Only include filters if we have any
	if len(queryFilters) > 0 {
		query["filters"] = queryFilters
	}

	var queryJSON bytes.Buffer
	enc := json.NewEncoder(&queryJSON)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(queryTable)
	url := hostName + "trade/search"
	if realm != "pc" {
		url += "/" + realm
	}
	url += "/" + tradeURLEncode(league)
	url += "?q=" + tradeURLEncode(strings.TrimSuffix(queryJSON.String(), "\n"))
	return url
}

// BuySimilarModEntry is one mod row of the popup: the formatted lines of the mods it sums, their source type, the trade
// ids it searches (more than one when the line is ambiguous), whether the trade site has an option for it and the
// option's value, whether the site wants its value negated (nil when no stat description said), and its value.
type BuySimilarModEntry struct {
	Value          *float64
	OptionValue    any
	Invert         *bool
	FormattedLines []string
	TradeIDs       []string
	Type           string
	IsOption       bool
}

// existingStats is every trade id the site has.
//
//pob:src/Classes/CompareBuySimilar.lua:12-22
var existingStats = sync.OnceValue(func() map[string]bool {
	existing := map[string]bool{}
	for _, cat := range tradeStats() {
		for _, entry := range cat.Entries {
			existing[entry.ID] = true
		}
	}
	return existing
})

// BuySimilarAddModEntries is the popup's mod rows for item's enchant, implicit, explicit and scourge lines.
//
//pob:src/Classes/CompareBuySimilar.lua:211-283
//pob:block:src/Classes/CompareBuySimilar.lua:301-309
func BuySimilarAddModEntries(item *Item) []*BuySimilarModEntry {
	var modEntries []*BuySimilarModEntry
	// this adds a single aggregated entry for matching stats (e.g. transformed flat dmg mods) which
	// avoids issues with confusing results. mods with different types are not summed as e.g.
	// implicit and explicit mods are separate in the search. options are also avoided as they don't
	// represent values that can be added combined
	insertOrAddToExisting := func(entry *BuySimilarModEntry) {
		for _, existingFilter := range modEntries {
			// check if all result trade ids are equal
			sameHashes := len(entry.TradeIDs) > 0 && slices.Equal(entry.TradeIDs, existingFilter.TradeIDs)
			if sameHashes && existingFilter.Type == entry.Type && !entry.IsOption {
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
		list []ModLine
		typ  string
	}{
		{item.EnchantModLines, "enchant"},
		{item.ImplicitModLines, "implicit"},
		{item.ExplicitModLines, "explicit"},
		{item.ScourgeModLines, "scourge"},
	}
	for _, source := range modTypeSources {
		for _, modLine := range source.list {
			if !item.checkModLineVariant(modLine) {
				continue
			}
			// remove unsupported data. the formatting of unsupported
			// mods is confusing here
			copied := modLine
			copied.Extra = ""
			formatted, ok := formattedModLine(copied, false)
			if !ok {
				continue
			}
			// Use range-resolved text for matching
			resolvedLine := copied.Line
			if copied.Range != nil {
				scalar := copied.ValueScalar
				if scalar == 0 {
					scalar = 1
				}
				resolvedLine = data.ApplyRange(copied.Line, *copied.Range, &scalar, nil)
			}
			// check option first, because even if we match a line via the descriptors, the values for option-based stats are different
			tradeID, value := findTradeIDOption(resolvedLine, source.typ)
			entry := &BuySimilarModEntry{
				// this array will always start with one line, but if multiple mods are
				// aggregated together it will contain the original mod lines for each
				FormattedLines: []string{formatted},
				Type:           source.typ,
				IsOption:       tradeID != "",
				Invert:         new(false),
				OptionValue:    value,
			}
			if tradeID != "" {
				entry.TradeIDs = []string{tradeID}
			} else {
				resultHashes, value, invert := findTradeHash(resolvedLine)
				// convert hashes to string ids
				var resultIDs []string
				existing := existingStats()
				for _, hash := range resultHashes {
					id := fmt.Sprintf("%s.stat_%d", source.typ, hash)
					if existing[id] {
						resultIDs = append(resultIDs, id)
					}
				}
				entry.TradeIDs = resultIDs
				entry.OptionValue = nil
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

// BuySimilarModRow is a mod row as openPopup lays it out: its source type, its label lines with their colour codes,
// whether its box can be checked, whether it has Min and Max boxes and the Min box's text.
type BuySimilarModRow struct {
	Type      string
	Min       string
	Lines     []string
	CanSearch bool
	HasBoxes  bool
}

var (
	buySimilarColorRe    = regexp.MustCompile(`\^x[0-9A-Fa-f]{6}`)
	buySimilarShortColRe = regexp.MustCompile(`\^[0-9A-Fa-f]`)
)

// BuySimilarModRows is the popup's mod rows as openPopup lays them out.
//
//pob:block:src/Classes/CompareBuySimilar.lua:458-511
func BuySimilarModRows(modEntries []*BuySimilarModEntry) []BuySimilarModRow {
	rows := make([]BuySimilarModRow, 0, len(modEntries))
	for _, entry := range modEntries {
		canSearch := len(entry.TradeIDs) > 0
		displayTexts := make([]string, len(entry.FormattedLines))
		for index, displayText := range entry.FormattedLines {
			// shorten time-lost jewel affix labels to fit better
			displayText = strings.ReplaceAll(displayText, " Passive Skills in Radius also grant", ":")
			colorCodeLength := buySimilarColorRe.FindString(displayText)
			if colorCodeLength == "" {
				colorCodeLength = buySimilarShortColRe.ReplaceAllString(displayText, "")
			}
			if !canSearch {
				// strip color codes and replace with gray
				displayText = "^8" + buySimilarShortColRe.ReplaceAllString(buySimilarColorRe.ReplaceAllString(displayText, ""), "")
			}
			if len(displayText) > len(colorCodeLength)+62 {
				displayText = displayText[:len(colorCodeLength)+54] + "..."
			}
			displayTexts[index] = displayText
		}
		row := BuySimilarModRow{Type: entry.Type, Lines: displayTexts, CanSearch: canSearch}
		// when the trade site has a dropdown for the value, we opt to disable
		// the inputs as they are numeric
		if !entry.IsOption && entry.Value != nil {
			row.HasBoxes = true
			if *entry.Value != 0 {
				row.Min = fmt.Sprintf("%.14g", *entry.Value)
			}
		}
		rows = append(rows, row)
	}
	return rows
}
