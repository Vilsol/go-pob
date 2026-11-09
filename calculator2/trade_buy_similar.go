package calculator2

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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
}

type BuySimilarDefenceFilter struct {
	Min     *int
	Max     *int
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

// BuySimilarURL is the trade search URL for the popup's selections, without its mod rows. The realm is PoE2's, which
// the popup does not let change.
//
//pob2:block:src/Classes/CompareBuySimilar.lua:44-133
//pob2:block:src/Classes/CompareBuySimilar.lua:177-195
//pob2:block:src/Classes/CompareBuySimilar.lua:385-396
func BuySimilarURL(item *Item, slotName string, o BuySimilarOptions) string {
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
