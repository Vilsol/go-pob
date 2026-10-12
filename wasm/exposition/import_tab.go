package exposition

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

func ImportTabExports(r bind.Registry) {
	r.Func(ImportItemsAndSkills, bind.AsPromise())
}

type characterData struct {
	Level     int              `json:"level"`
	Equipment []importItemData `json:"equipment"`
	Guardian  []importItemData `json:"guardian"`
	Passives  json.RawMessage  `json:"passives"`
}

type importProperty struct {
	Name   string  `json:"name"`
	Values [][]any `json:"values"`
}

func (p importProperty) value() string {
	if len(p.Values) == 0 || len(p.Values[0]) == 0 {
		return ""
	}
	s, _ := p.Values[0][0].(string)
	return s
}

type importSocket struct {
	Group   int    `json:"group"`
	SColour string `json:"sColour"`
}

type importModFlags struct {
	Fractured bool `json:"fractured"`
	Crafted   bool `json:"crafted"`
	Mutated   bool `json:"mutated"`
	Vestigial bool `json:"vestigial"`
}

type importMod struct {
	Description string
	Flags       importModFlags
}

func (m *importMod) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &m.Description); err == nil {
		return nil
	}
	var obj struct {
		Description string         `json:"description"`
		Flags       importModFlags `json:"flags"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return fmt.Errorf("item mod: %w", err)
	}
	m.Description, m.Flags = obj.Description, obj.Flags
	return nil
}

type importItemData struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	TypeLine      string           `json:"typeLine"`
	InventoryID   string           `json:"inventoryId"`
	X             int              `json:"x"`
	FrameType     int              `json:"frameType"`
	Ilvl          int              `json:"ilvl"`
	Influences    map[string]bool  `json:"influences"`
	Searing       bool             `json:"searing"`
	Tangled       bool             `json:"tangled"`
	Properties    []importProperty `json:"properties"`
	Requirements  []importProperty `json:"requirements"`
	Split         bool             `json:"split"`
	Mirrored      bool             `json:"mirrored"`
	Corrupted     bool             `json:"corrupted"`
	Fractured     bool             `json:"fractured"`
	Sockets       []importSocket   `json:"sockets"`
	SocketedItems []importItemData `json:"socketedItems"`
	EnchantMods   []string         `json:"enchantMods"`
	ScourgeMods   []string         `json:"scourgeMods"`
	ImplicitMods  []importMod      `json:"implicitMods"`
	FracturedMods []string         `json:"fracturedMods"`
	ExplicitMods  []importMod      `json:"explicitMods"`
	CrucibleMods  []string         `json:"crucibleMods"`
	CraftedMods   []string         `json:"craftedMods"`
	MutatedMods   []string         `json:"mutatedMods"`
	FlavourText   []string         `json:"flavourText"`
	FoilVariation *int             `json:"foilVariation"`
	IsRelic       bool             `json:"isRelic"`

	AbyssJewel     bool   `json:"abyssJewel"`
	Support        bool   `json:"support"`
	Socket         int    `json:"socket"`
	BuiltInSupport string `json:"builtInSupport"`
	Hybrid         *struct {
		BaseTypeName string `json:"baseTypeName"`
		IsVaalGem    bool   `json:"isVaalGem"`
	} `json:"hybrid"`
}

type importGem struct {
	name      string
	gameID    string
	skillID   string
	variantID string
}

type importGemTables struct {
	forBaseName map[string]importGem
	forVaalGem  map[string]importGem
}

//pob:block:src/Modules/Data.lua:1062-1076,src/Modules/Data.lua:1086-1119
var importGemLookup = sync.OnceValue(func() importGemTables {
	t := importGemTables{forBaseName: map[string]importGem{}, forVaalGem: map[string]importGem{}}
	add := func(row raw.PoBGem, gem importGem) {
		suffix := ""
		if raw.PoBSkills[row.GrantedEffectID].Support && row.GrantedEffectID != "SupportBarrage" {
			suffix = " Support"
		}
		t.forBaseName[strings.ToLower(gem.name+suffix)] = gem
		t.forBaseName[strings.ToLower(row.Name+suffix)] = gem
	}
	for _, id := range slices.Sorted(maps.Keys(raw.Gems)) {
		row := raw.Gems[id]
		gem := importGem{name: pob.SanitiseText(row.Name), gameID: id, skillID: row.GrantedEffectID}
		add(row, gem)
		if !row.Tags["vaal"] || row.SecondaryGrantedEffectID == "" {
			continue
		}
		t.forVaalGem[row.SecondaryGrantedEffectID] = gem
		for _, alt := range []string{"AltX", "AltY"} {
			skill, ok := raw.PoBSkills[row.SecondaryGrantedEffectID+alt]
			if !ok {
				continue
			}
			altGem := importGem{name: "Vaal " + skill.Name, gameID: id, skillID: row.GrantedEffectID, variantID: row.GrantedEffectID + alt}
			t.forVaalGem[row.SecondaryGrantedEffectID+alt] = altGem
			add(raw.PoBGem{Name: altGem.name, GrantedEffectID: row.GrantedEffectID}, altGem)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(raw.GemVariants)) {
		row := raw.GemVariants[id]
		add(row, importGem{name: pob.SanitiseText(row.Name), gameID: raw.GemGameIDForGrantedEffectID[id], skillID: row.GrantedEffectID})
	}
	return t
})

//pob:src/Classes/ImportTab.lua:1492-1494
var (
	importRarityMap = map[int]string{0: "NORMAL", 1: "MAGIC", 2: "RARE", 3: "UNIQUE", 9: "RELIC", 10: "RELIC"}
	importSlotMap   = map[string]string{"Weapon": "Weapon 1", "Offhand": "Weapon 2", "Weapon2": "Weapon 1 Swap", "Offhand2": "Weapon 2 Swap",
		"Helm": "Helmet", "BodyArmour": "Body Armour", "Gloves": "Gloves", "Boots": "Boots", "Amulet": "Amulet", "Ring": "Ring 1",
		"Ring2": "Ring 2", "Ring3": "Ring 3", "Belt": "Belt", "BrequelGrafts": "Graft 1", "BrequelGrafts2": "Graft 2"}
)

//pob:block:src/Modules/ItemTools.lua:16-25
var importInfluenceInfo = []struct{ key, display string }{
	{"shaper", "Shaper"}, {"elder", "Elder"}, {"adjudicator", "Warlord"}, {"basilisk", "Hunter"}, {"crusader", "Crusader"},
	{"eyrie", "Redeemer"}, {"cleansing", "Searing Exarch"}, {"tangle", "Eater of Worlds"},
}

//pob:block:src/Classes/Item.lua:14
var importCatalystList = []string{"Abrasive", "Accelerating", "Dextral", "Fertile", "Imbued", "Intrinsic", "Noxious", "Prismatic",
	"Sinistral", "Tempering", "Turbulent", "Unstable"}

//pob:block:src/Classes/ImportTab.lua:1594-1607
var importCatalystMap = map[string]int{"Attack": 1, "Speed": 2, "Suffix": 3, "Life and Mana": 4, "Caster": 5, "Attribute": 6,
	"Physical and Chaos Damage": 7, "Resistance": 8, "Prefix": 9, "Defense": 10, "Elemental Damage": 11, "Critical": 12}

//pob:block:src/Classes/ImportTab.lua:1777-1790
var importFoilVariants = []string{"Amethyst", "Verdant", "Ruby", "Cobalt", "Sunset", "Aureate", "Celestial Quartz", "Celestial Ruby",
	"Celestial Emerald", "Celestial Aureate", "Celestial Pearl", "Celestial Amethyst"}

var (
	importDigitsRe       = regexp.MustCompile(`\d+`)
	importCatalystNameRe = regexp.MustCompile(`Quality \((.*) Modifiers\)`)
	importUniqueIDRe     = regexp.MustCompile(`(?m)^Unique ID: (.+?)\r?$`)
)

type importModLine struct {
	line                                                 string
	crafted, scourge, crucible, mutated, fractured, vest bool
}

type importedItem struct {
	rarity, title, baseName, namePrefix, nameSuffix string
	name                                            string
	base                                            *calculator.ItemBase
	uniqueID                                        string
	influence                                       map[string]bool
	itemLevel                                       *int
	quality                                         *int
	catalyst, catalystQuality                       *int
	jewelRadiusLabel                                string
	limit                                           *int
	intangibility, memoryStrands                    *int
	armourData                                      map[string]float64
	split, mirrored, corrupted, fractured           bool
	sockets                                         []importSocket
	abyssalSocketCount                              int
	levelReq, classRestriction                      string
	hasLevelReq                                     bool
	enchant, scourge, implicit, explicit, crucible  []importModLine
	foilType                                        string
}

type importTab struct {
	build *pob.PathOfBuilding
}

func importDigits(s string) *int {
	m := importDigitsRe.FindString(s)
	if m == "" {
		return nil
	}
	n, _ := strconv.Atoi(m)
	return &n
}

func importNumber(s string) *int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	return &n
}

func importLines(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == '\n' })
}

//pob:block:src/Classes/ImportTab.lua:1544-1553
func importItemBase(name string) (string, int) {
	best, at := "", -1
	for _, baseName := range slices.Sorted(maps.Keys(calculator.ItemBases())) {
		if s := strings.Index(name, baseName); s >= 0 && len(baseName) > len(best) {
			best, at = baseName, s
		}
	}
	return best, at
}

//pob:src/Classes/ImportTab.lua:1496-1820
func (t *importTab) importItem(itemData importItemData, slotName string, ignoreWeaponSwap bool, itemSetID int) {
	if slotName == "" {
		switch {
		case itemData.InventoryID == "PassiveJewels":
			return
		case itemData.InventoryID == "Flask":
			slotName = "Flask " + strconv.Itoa(itemData.X+1)
		case !(ignoreWeaponSwap && (itemData.InventoryID == "Weapon2" || itemData.InventoryID == "Offhand2")):
			slotName = importSlotMap[itemData.InventoryID]
		}
	}
	if slotName == "" {
		return
	}

	item := &importedItem{}
	bases := calculator.ItemBases()
	rarity, hasRarity := importRarityMap[itemData.FrameType]
	item.rarity = rarity
	if len(itemData.Name) > 0 {
		item.title = pob.SanitiseText(itemData.Name)
		item.baseName = strings.Replace(pob.SanitiseText(itemData.TypeLine), "Synthesised ", "", -1)
		item.baseName = strings.TrimPrefix(item.baseName, "Vestigial ")
		item.name = item.title + ", " + item.baseName
		if item.baseName == "Two-Toned Boots" {
			item.baseName = "Two-Toned Boots (Armour/Energy Shield)"
		}
		item.base = bases[item.baseName]
	} else {
		item.name = pob.SanitiseText(itemData.TypeLine)
		if strings.Contains(item.name, "Energy Blade") {
			oneHanded := false
			for _, p := range itemData.Properties {
				if info := data.WeaponTypes[data.ItemClassName(p.Name)]; info != nil && info.OneHand {
					oneHanded = true
					break
				}
			}
			item.name = "Energy Blade Two Handed"
			if oneHanded {
				item.name = "Energy Blade One Handed"
			}
			item.rarity, hasRarity = "NORMAL", true
			itemData.ImplicitMods = nil
			itemData.ExplicitMods = nil
		}
		if baseName, s := importItemBase(item.name); s >= 0 {
			item.baseName = baseName
			item.namePrefix = item.name[:s]
			item.nameSuffix = item.name[s+len(baseName):]
		} else if s := strings.Index(item.name, "Two-Toned Boots"); s >= 0 {
			item.baseName = "Two-Toned Boots (Armour/Energy Shield)"
			item.namePrefix = item.name[:s]
			item.nameSuffix = item.name[s+len("Two-Toned Boots"):]
		}
		item.base = bases[item.baseName]
	}
	if item.base == nil || !hasRarity {
		return
	}

	item.uniqueID = itemData.ID
	item.influence = map[string]bool{}
	for _, info := range importInfluenceInfo {
		item.influence[info.key] = itemData.Influences[strings.ToLower(info.display)]
	}
	if itemData.Searing {
		item.influence["cleansing"] = true
	}
	if itemData.Tangled {
		item.influence["tangle"] = true
	}
	if itemData.Ilvl > 0 {
		ilvl := itemData.Ilvl
		item.itemLevel = &ilvl
	}
	if item.base.Weapon != nil || item.base.Armour != nil || item.base.Flask != nil || item.base.Tincture != nil {
		item.quality = new(int)
	}
	for _, property := range itemData.Properties {
		switch {
		case property.Name == "Quality":
			item.quality = importDigits(property.value())
		case strings.Contains(property.Name, "Quality ("):
			var catalyst *int
			if m := importCatalystNameRe.FindStringSubmatch(property.Name); m != nil {
				if n, ok := importCatalystMap[m[1]]; ok {
					catalyst = &n
				}
			}
			item.catalyst = catalyst
			item.catalystQuality = importDigits(property.value())
		case property.Name == "Radius":
			item.jewelRadiusLabel = property.value()
		case property.Name == "Limited to":
			item.limit = importNumber(property.value())
		case property.Name == "Evasion Rating":
			if item.baseName == "Two-Toned Boots (Armour/Energy Shield)" {
				item.baseName = "Two-Toned Boots (Armour/Evasion)"
				item.base = bases[item.baseName]
			}
		case property.Name == "Energy Shield":
			if item.baseName == "Two-Toned Boots (Armour/Evasion)" {
				item.baseName = "Two-Toned Boots (Evasion/Energy Shield)"
				item.base = bases[item.baseName]
			}
		case strings.Contains(property.Name, "Intangibility"):
			item.intangibility = importDigits(property.value())
		case property.Name == "Memory Strands":
			item.memoryStrands = importNumber(property.value())
		}
		switch property.Name {
		case "Energy Shield", "Ward", "Armour", "Evasion Rating":
			if item.armourData == nil {
				item.armourData = map[string]float64{}
			}
			key := strings.ReplaceAll(strings.ReplaceAll(property.Name, " Rating", ""), " ", "")
			for _, value := range property.Values {
				if len(value) > 0 {
					s, _ := value[0].(string)
					n, _ := strconv.ParseFloat(s, 64)
					item.armourData[key] += n
				}
			}
		}
	}
	item.split = itemData.Split
	item.mirrored = itemData.Mirrored
	item.corrupted = itemData.Corrupted
	item.fractured = itemData.Fractured
	if len(itemData.Sockets) > 0 {
		item.sockets = itemData.Sockets
		for _, socket := range itemData.Sockets {
			if socket.SColour == "A" {
				item.abyssalSocketCount = 1
			}
		}
		if item.abyssalSocketCount > 0 && strings.Contains(item.name, "Energy Blade") {
			itemData.ExplicitMods = append(itemData.ExplicitMods, importMod{Description: "Has " + strconv.Itoa(item.abyssalSocketCount) + " Abyssal Sockets"})
		}
	}
	if len(itemData.SocketedItems) > 0 {
		t.importSocketedItems(item, itemData.SocketedItems, slotName)
	}
	if itemData.Requirements != nil && len(itemData.SocketedItems) == 0 {
		for _, req := range itemData.Requirements {
			switch req.Name {
			case "Level":
				item.levelReq, item.hasLevelReq = req.value(), true
			case "Class:":
				item.classRestriction = req.value()
			}
		}
	}
	for _, line := range itemData.EnchantMods {
		for _, l := range importLines(line) {
			item.enchant = append(item.enchant, importModLine{line: l, crafted: true})
		}
	}
	for _, line := range itemData.ScourgeMods {
		for _, l := range importLines(line) {
			item.scourge = append(item.scourge, importModLine{line: l, scourge: true})
		}
	}
	for _, m := range itemData.ImplicitMods {
		for _, l := range importLines(m.Description) {
			item.implicit = append(item.implicit, importModLine{line: l, vest: m.Flags.Vestigial})
		}
	}
	for _, line := range itemData.FracturedMods {
		for _, l := range importLines(line) {
			item.explicit = append(item.explicit, importModLine{line: l, fractured: true})
		}
	}
	for _, m := range itemData.ExplicitMods {
		for _, l := range importLines(m.Description) {
			item.explicit = append(item.explicit, importModLine{line: l, fractured: m.Flags.Fractured, crafted: m.Flags.Crafted,
				mutated: m.Flags.Mutated, vest: m.Flags.Vestigial})
		}
	}
	for _, line := range itemData.CrucibleMods {
		for _, l := range importLines(line) {
			item.crucible = append(item.crucible, importModLine{line: l, crucible: true})
		}
	}
	for _, line := range itemData.CraftedMods {
		for _, l := range importLines(line) {
			item.explicit = append(item.explicit, importModLine{line: l, crafted: true})
		}
	}
	for _, line := range itemData.MutatedMods {
		for _, l := range importLines(line) {
			item.explicit = append(item.explicit, importModLine{line: l, mutated: true})
		}
	}
	//pob:block:src/Classes/ImportTab.lua:1752-1775
	for _, line := range itemData.FlavourText {
		for _, l := range importLines(line) {
			start, end := strings.Index(l, "{"), strings.Index(l, "}")
			if start >= 0 && end >= 0 && end > start {
				l = l[start+1 : end]
			}
			if calculator.ParseMod(l, false).ModList != nil {
				item.explicit = append(item.explicit, importModLine{line: l})
			}
		}
	}
	if itemData.FoilVariation != nil || itemData.IsRelic {
		item.foilType = "Rainbow"
		if itemData.FoilVariation != nil && *itemData.FoilVariation >= 1 && *itemData.FoilVariation <= len(importFoilVariants) {
			item.foilType = importFoilVariants[*itemData.FoilVariation-1]
		}
	}

	//pob:block:src/Classes/ImportTab.lua:1794-1819
	text := item.buildRaw()
	id := 0
	for _, existing := range t.build.Items.Items {
		if m := importUniqueIDRe.FindStringSubmatch(existing.Text); m != nil && m[1] == itemData.ID {
			id = existing.ID
			break
		}
	}
	if id != 0 {
		t.build.SetItemText(id, text)
	} else {
		id = t.build.AddItem(text)
	}
	if itemSetID != 0 && itemSetID != t.build.Items.ActiveItemSet {
		t.build.SetSetSlotItem(itemSetID, slotName, id)
	} else {
		t.build.SetSlotItem(slotName, id)
	}
}

//pob:block:src/Classes/Item.lua:1733-2008
func (item *importedItem) buildRaw() string {
	var out []string
	add := func(s ...string) { out = append(out, s...) }
	add("Rarity: " + item.rarity)
	if item.title != "" {
		add(item.title, item.baseName)
	} else {
		add(item.namePrefix + item.baseName + item.nameSuffix)
	}
	for _, typ := range []string{"Armour", "Evasion", "EnergyShield", "Ward"} {
		if v := item.armourData[typ]; v > 0 {
			add(strings.ReplaceAll(typ, "EnergyShield", "Energy Shield") + ": " + strconv.FormatFloat(v, 'f', -1, 64))
		}
	}
	if item.intangibility != nil {
		add("Intangibility: " + strconv.Itoa(*item.intangibility) + "%")
	}
	if item.uniqueID != "" {
		add("Unique ID: " + item.uniqueID)
	}
	for _, info := range importInfluenceInfo {
		if item.influence[info.key] {
			add(info.display + " Item")
		}
	}
	if item.catalyst != nil && *item.catalyst > 0 {
		add("Catalyst: " + importCatalystList[*item.catalyst-1])
	}
	if item.catalystQuality != nil {
		add("CatalystQuality: " + strconv.Itoa(*item.catalystQuality))
	}
	if item.itemLevel != nil {
		add("Item Level: " + strconv.Itoa(*item.itemLevel))
	}
	if item.memoryStrands != nil {
		add("Memory Strands: " + strconv.Itoa(*item.memoryStrands))
	}
	if item.quality != nil {
		add("Quality: " + strconv.Itoa(*item.quality))
	}
	if len(item.sockets) > 0 {
		line := "Sockets: "
		for i, socket := range item.sockets {
			line += socket.SColour
			if i+1 < len(item.sockets) {
				if socket.Group == item.sockets[i+1].Group {
					line += "-"
				} else {
					line += " "
				}
			}
		}
		add(line)
	}
	if item.hasLevelReq {
		add("LevelReq: " + item.levelReq)
	}
	if item.jewelRadiusLabel != "" {
		add("Radius: " + item.jewelRadiusLabel)
	}
	if item.limit != nil {
		add("Limited to: " + strconv.Itoa(*item.limit))
	}
	if item.classRestriction != "" {
		add("Requires Class " + item.classRestriction)
	}
	add("Implicits: " + strconv.Itoa(len(item.enchant)+len(item.implicit)+len(item.scourge)))
	for _, list := range [][]importModLine{item.enchant, item.scourge, item.implicit, item.explicit, item.crucible} {
		for _, ml := range list {
			add(ml.raw())
		}
	}
	if item.split {
		add("Split")
	}
	if item.mirrored {
		add("Mirrored")
	}
	if item.fractured {
		add("Fractured Item")
	}
	if item.corrupted {
		add("Corrupted")
	}
	if item.foilType != "" {
		add("Foil Unique (" + item.foilType + ")")
	}
	return strings.Join(out, "\n")
}

//pob:block:src/Classes/Item.lua:1803-1870
func (ml importModLine) raw() string {
	line := ml.line
	for _, f := range []struct {
		on  bool
		tag string
	}{{ml.crafted, "{crafted}"}, {ml.scourge, "{scourge}"}, {ml.crucible, "{crucible}"}, {ml.mutated, "{mutated}"},
		{ml.fractured, "{fractured}"}, {ml.vest, "{vestigial}"}} {
		if f.on {
			line = f.tag + line
		}
	}
	return line
}

//pob:src/Classes/ImportTab.lua:1822-1897
func (t *importTab) importSocketedItems(item *importedItem, socketedItems []importItemData, slotName string) {
	gems := importGemLookup()
	itemSocketGroupList := map[int]*pob.Skill{}
	abyssalSocketID := 1
	for _, socketedItem := range socketedItems {
		if socketedItem.AbyssJewel {
			t.importItem(socketedItem, slotName+" Abyssal Socket "+strconv.Itoa(abyssalSocketID), false, 0)
			abyssalSocketID++
			continue
		}
		gem, ok := gems.forBaseName[strings.ToLower(pob.SanitiseText(socketedItem.TypeLine))]
		if socketedItem.Hybrid != nil {
			gem, ok = gems.forBaseName[strings.ToLower(pob.SanitiseText(socketedItem.Hybrid.BaseTypeName))]
			if ok && socketedItem.Hybrid.IsVaalGem {
				gem, ok = gems.forVaalGem[gem.skillID]
			}
		}
		if !ok {
			continue
		}
		gemInstance := pob.Gem{Level: 20, Enabled: true, EnableGlobal1: true, GemID: gem.gameID, NameSpec: gem.name, SkillID: gem.skillID}
		if gem.variantID != "" {
			variantID := gem.variantID
			gemInstance.VariantID = &variantID
		}
		for _, property := range socketedItem.Properties {
			switch property.Name {
			case "Level":
				if n := importDigits(property.value()); n != nil {
					gemInstance.Level = *n
				}
			case "Quality":
				if n := importDigits(property.value()); n != nil {
					gemInstance.Quality = *n
				}
			}
		}
		if socketedItem.Socket >= len(item.sockets) {
			continue
		}
		groupID := item.sockets[socketedItem.Socket].Group
		if itemSocketGroupList[groupID] == nil {
			itemSocketGroupList[groupID] = &pob.Skill{Enabled: true, Slot: slotName}
		}
		socketGroup := itemSocketGroupList[groupID]
		socketGroup.Gems = append(socketGroup.Gems, gemInstance)
		if socketedItem.BuiltInSupport != "" {
			imbued := strings.Replace(socketedItem.BuiltInSupport, "Supported by Level 1 ", "", -1)
			socketGroup.ImbuedSupport = &imbued
		}
	}

	set := &t.build.Skills.SkillSets[t.build.ActiveSkillSetIndex()]
	for _, groupID := range slices.Sorted(maps.Keys(itemSocketGroupList)) {
		itemSocketGroup := itemSocketGroupList[groupID]
		var repGroup *pob.Skill
		for i := range set.Skills {
			socketGroup := &set.Skills[i]
			if len(socketGroup.Gems) != len(itemSocketGroup.Gems) || (socketGroup.Slot != "" && socketGroup.Slot != slotName) {
				continue
			}
			match := true
			for gemIndex, gem := range socketGroup.Gems {
				if !strings.EqualFold(gem.NameSpec, itemSocketGroup.Gems[gemIndex].NameSpec) {
					match = false
					break
				}
			}
			if match {
				repGroup = socketGroup
				break
			}
		}
		if repGroup != nil {
			for gemIndex := range repGroup.Gems {
				repGroup.Gems[gemIndex].Level = itemSocketGroup.Gems[gemIndex].Level
				repGroup.Gems[gemIndex].Quality = itemSocketGroup.Gems[gemIndex].Quality
			}
		} else {
			set.Skills = append(set.Skills, *itemSocketGroup)
		}
	}
}

const importGuardItemSet = "Animate Guardian"

//pob:src/Classes/ImportTab.lua:1341-1353
func (t *importTab) getOrCreateGuardianItemSet() int {
	for _, set := range t.build.Items.ItemSets {
		if set.Title != nil && *set.Title == importGuardItemSet {
			id, _ := strconv.Atoi(set.ID)
			return id
		}
	}
	return t.build.NewItemSet(importGuardItemSet)
}

//pob:src/Classes/ImportTab.lua:1356-1371
func (t *importTab) assignGuardianItemSet(itemSetID int) {
	setTitle := func(id int) (string, bool) {
		for _, set := range t.build.Items.ItemSets {
			if set.ID == strconv.Itoa(id) {
				return pointerString(set.Title), true
			}
		}
		return "", false
	}
	set := &t.build.Skills.SkillSets[t.build.ActiveSkillSetIndex()]
	for g := range set.Skills {
		for k := range set.Skills[g].Gems {
			gem := &set.Skills[g].Gems[k]
			if skill, ok := grantedEffectOf(*gem); !ok || skill.Name != importGuardItemSet {
				continue
			}
			if title, ok := setTitle(gem.SkillMinionItemSet); gem.SkillMinionItemSet == 0 || (ok && title == importGuardItemSet) {
				gem.SkillMinionItemSet = itemSetID
			}
			current := 0
			if gem.SkillMinionItemSetCalcs != nil {
				current = *gem.SkillMinionItemSetCalcs
			}
			if title, ok := setTitle(current); current == 0 || (ok && title == importGuardItemSet) {
				id := itemSetID
				gem.SkillMinionItemSetCalcs = &id
			}
		}
	}
}

func pointerString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

//pob:src/Classes/ImportTab.lua:1256-1268
func socketGroupReimportKey(socketGroup pob.Skill) string {
	const sep = "\x1f"
	names := make([]string, 0, len(socketGroup.Gems))
	for _, gem := range socketGroup.Gems {
		names = append(names, strings.ToLower(gem.NameSpec))
	}
	return strings.Join([]string{socketGroup.Slot, pointerString(socketGroup.SourceAttr), strconv.Itoa(len(socketGroup.Gems)),
		strings.Join(names, sep)}, sep)
}

type socketGroupReimportState struct {
	group       pob.Skill
	isMainGroup bool
}

//pob:block:src/Classes/ImportTab.lua:1304-1337
func applySocketGroupReimportState(socketGroup *pob.Skill, state socketGroupReimportState) {
	socketGroup.Enabled = state.group.Enabled
	socketGroup.IncludeInFullDPS = state.group.IncludeInFullDPS
	socketGroup.GroupCount = state.group.GroupCount
	socketGroup.Label = state.group.Label
	socketGroup.MainActiveSkill = state.group.MainActiveSkill
	socketGroup.MainActiveSkillCalcs = state.group.MainActiveSkillCalcs
	for gemIndex, from := range state.group.Gems {
		if gemIndex >= len(socketGroup.Gems) {
			continue
		}
		gem := &socketGroup.Gems[gemIndex]
		gem.Enabled = from.Enabled
		gem.Count = from.Count
		gem.SkillPart = from.SkillPart
		gem.SkillPartCalcs = from.SkillPartCalcs
		gem.SkillStageCount = from.SkillStageCount
		gem.SkillStageCountCalcs = from.SkillStageCountCalcs
		gem.SkillMineCount = from.SkillMineCount
		gem.SkillMineCountCalcs = from.SkillMineCountCalcs
		gem.SkillMinion = from.SkillMinion
		gem.SkillMinionCalcs = from.SkillMinionCalcs
		gem.SkillMinionItemSet = from.SkillMinionItemSet
		gem.SkillMinionItemSetCalcs = from.SkillMinionItemSetCalcs
		gem.SkillMinionSkill = from.SkillMinionSkill
		gem.SkillMinionSkillCalcs = from.SkillMinionSkillCalcs
		gem.EnableGlobal1 = from.EnableGlobal1
		gem.EnableGlobal2 = from.EnableGlobal2
	}
}

var importGemForSkill = sync.OnceValue(func() map[string]bool {
	out := map[string]bool{}
	for _, table := range []map[string]raw.PoBGem{raw.Gems, raw.GemVariants} {
		for _, row := range table {
			out[row.GrantedEffectID] = true
		}
	}
	return out
})

//pob:block:src/Classes/SkillsTab.lua:1141-1160
func grantedEffectOf(gem pob.Gem) (raw.PoBSkill, bool) {
	if gem.GemID != "" || gem.SkillID == "" || importGemForSkill()[gem.SkillID] {
		return raw.PoBSkill{}, false
	}
	skill, ok := raw.PoBSkills[gem.SkillID]
	return skill, ok
}

func activeSkillName(gem pob.Gem) string {
	skill, ok := grantedEffectOf(gem)
	if !ok || skill.Support {
		return ""
	}
	return skill.Name
}

//pob:src/Classes/ImportTab.lua:1380-1490
func ImportItemsAndSkills(build *pob.PathOfBuilding, characterJSON string, clearItems bool, clearSkills bool, ignoreWeaponSwap bool) (string, error) {
	if build == nil {
		return "", fmt.Errorf("no build")
	}
	var wrapper struct {
		Character *characterData `json:"character"`
	}
	if err := json.Unmarshal([]byte(characterJSON), &wrapper); err != nil {
		return "", fmt.Errorf("Invalid JSON format (decode error)")
	}
	charData := wrapper.Character
	if charData == nil {
		return "", fmt.Errorf("Invalid JSON format (character missing)")
	}
	if charData.Equipment == nil || charData.Passives == nil {
		return "", fmt.Errorf("Invalid JSON format (equipment or passives missing)")
	}
	if len(build.Skills.SkillSets) == 0 {
		build.Skills.SkillSets = []pob.SkillSet{{ID: 1}}
		build.Skills.ActiveSkillSet = 1
	}
	t := &importTab{build: build}

	if clearItems {
		for _, set := range build.Items.ItemSets {
			if id, _ := strconv.Atoi(set.ID); id != build.Items.ActiveItemSet {
				continue
			}
			for _, slot := range set.Slots {
				if slot.ItemID != 0 {
					build.DeleteItem(slot.ItemID)
				}
			}
		}
	}

	set := func() *pob.SkillSet { return &build.Skills.SkillSets[build.ActiveSkillSetIndex()] }
	mainSkillEmpty := len(set().Skills) == 0
	var skillOrder []string
	var preserved map[string][]socketGroupReimportState
	if clearSkills {
		skillOrder = []string{}
		preserved = map[string][]socketGroupReimportState{}
		for _, socketGroup := range set().Skills {
			for _, gem := range socketGroup.Gems {
				if name := activeSkillName(gem); name != "" {
					skillOrder = append(skillOrder, name)
				}
			}
		}
		for index, socketGroup := range set().Skills {
			key := socketGroupReimportKey(socketGroup)
			socketGroup.Gems = slices.Clone(socketGroup.Gems)
			preserved[key] = append(preserved[key], socketGroupReimportState{group: socketGroup, isMainGroup: index+1 == build.Build.MainSocketGroup})
		}
		set().Skills = []pob.Skill{}
	}
	for _, itemData := range charData.Equipment {
		t.importItem(itemData, "", ignoreWeaponSwap, 0)
	}
	if len(charData.Guardian) > 0 {
		guardianSet := t.getOrCreateGuardianItemSet()
		for _, itemData := range charData.Guardian {
			t.importItem(itemData, "", ignoreWeaponSwap, guardianSet)
		}
		t.assignGuardianItemSet(guardianSet)
	}
	//pob:block:src/Classes/ImportTab.lua:1424-1460
	if skillOrder != nil {
		order := func(group pob.Skill) int {
			best := -1
			for _, gem := range group.Gems {
				if name := activeSkillName(gem); name != "" {
					if i := slices.Index(skillOrder, name); i >= 0 && (best < 0 || i < best) {
						best = i
					}
				}
			}
			return best
		}
		groups := set().Skills
		orders := make([]int, len(groups))
		for i, group := range groups {
			orders[i] = order(group)
		}
		index := make([]int, len(groups))
		for i := range index {
			index[i] = i
		}
		sort.SliceStable(index, func(x, y int) bool {
			a, b := orders[index[x]], orders[index[y]]
			switch {
			case a >= 0 && b >= 0:
				return a < b
			case a < 0 && b < 0:
				return false
			default:
				return a >= 0
			}
		})
		sorted := make([]pob.Skill, len(groups))
		for i, from := range index {
			sorted[i] = groups[from]
		}
		set().Skills = sorted
	}
	if preserved != nil {
		restoredMainSocketGroup := 0
		for index := range set().Skills {
			socketGroup := &set().Skills[index]
			key := socketGroupReimportKey(*socketGroup)
			if stateList := preserved[key]; len(stateList) > 0 {
				state := stateList[0]
				preserved[key] = stateList[1:]
				applySocketGroupReimportState(socketGroup, state)
				if state.isMainGroup {
					restoredMainSocketGroup = index + 1
				}
			}
		}
		if restoredMainSocketGroup != 0 {
			build.SetMainSocketGroup(restoredMainSocketGroup)
		}
	}
	if mainSkillEmpty {
		build.SetMainSocketGroup(build.GuessMainSocketGroup())
	}
	build.SetLevel(charData.Level)
	return "Items and skills successfully imported.", nil
}
