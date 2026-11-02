package calculator2

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/mod"
)

// ItemModSource is modLib.parseMod: the mods a line parses to and the unparsed remainder ("" for
// Lua's nil). A nil slice is Lua's nil modList; an empty non-nil slice is an empty table.
type ItemModSource func(line string) ([]mod.Mod, string)

//pob2:src/Modules/Main.lua:105-106
const (
	defaultItemAffixQuality = 0.5
	defaultItemQuality      = 20
)

var dmgTypeList = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

//pob2:src/Classes/Item.lua:14-30
var (
	catalystList           = []string{"Flesh", "Neural", "Carapace", "Uul-Netol's", "Xoph's", "Tul's", "Esh's", "Chayula's", "Reaver", "Sibilant", "Skittering", "Adaptive", "Necrotic"}
	catalystDescriptorList = []string{"Life", "Mana", "Defence", "Physical", "Fire", "Cold", "Lightning", "Chaos", "Attack", "Caster", "Speed", "Attribute", "Minion"}
	catalystTags           = [][]string{
		{"life"},
		{"mana"},
		{"defences", "armour", "evasion", "energyshield"},
		{"physical"},
		{"fire"},
		{"cold"},
		{"lightning"},
		{"chaos"},
		{"attack"},
		{"caster"},
		{"speed"},
		{"attribute"},
		{"minion"},
	}
)

//pob2:src/Data/Global.lua:7-75
var colorCodeNames = map[string]bool{
	"NORMAL": true, "MAGIC": true, "RARE": true, "UNIQUE": true, "RELIC": true, "GEM": true, "GEMINFO": true,
	"PROPHECY": true, "CURRENCY": true, "CRAFTED": true, "ENCHANTED": true, "CUSTOM": true, "SOURCE": true,
	"UNSUPPORTED": true, "DISABLED": true, "WARNING": true, "TIP": true, "FIRE": true, "COLD": true,
	"LIGHTNING": true, "CHAOS": true, "POSITIVE": true, "NEGATIVE": true, "HIGHLIGHT": true, "OFFENCE": true,
	"DEFENCE": true, "SCION": true, "MARAUDER": true, "WARRIOR": true, "RANGER": true, "HUNTRESS": true,
	"WITCH": true, "SORCERESS": true, "DUELIST": true, "MERCENARY": true, "TEMPLAR": true, "DRUID": true,
	"SHADOW": true, "MONK": true, "MAINHAND": true, "MAINHANDBG": true, "OFFHAND": true, "OFFHANDBG": true,
	"SHAPER": true, "ELDER": true, "FRACTURED": true, "MUTATED": true, "ADJUDICATOR": true, "BASILISK": true,
	"CRUSADER": true, "EYRIE": true, "CLEANSING": true, "TANGLE": true, "CHILLBG": true, "FREEZEBG": true,
	"SHOCKBG": true, "SCORCHBG": true, "BRITTLEBG": true, "SAPBG": true, "SCOURGE": true, "CRUCIBLE": true,
	"GEMDESCRIPTION": true, "SPLITPERSONALITY": true, "VESTIGIAL": true, "INTANGIBILITY": true, "MEMORY": true,
	"SPIRIT": true,
}

//pob2:src/Modules/Data.lua:669-682
var jewelRadiusLabels = map[int]string{1: "Small", 2: "Medium", 3: "Large", 4: "Very Large", 6: "Variable",
	7: "Variable", 8: "Variable", 9: "Variable", 10: "Variable", 11: "Variable", 12: "Variable", 13: "Variable"}

//pob2:src/Classes/Item.lua:101-114
var lineFlags = map[string]bool{
	"crafted": true, "custom": true, "disabled": true, "enchant": true, "fractured": true, "implicit": true,
	"desecrated": true, "mutated": true, "rune": true, "unscalable": true, "prefix": true, "suffix": true,
}

// ModLine is one modifier line on an item.
type ModLine struct {
	Line               string
	Extra              string
	ModList            []mod.Mod
	Range              *float64
	ValueScalar        *float64
	DisplayValueScalar *float64
	CorruptedRange     *float64
	ModTags            []string
	Order              *float64

	VariantList      map[int]bool
	VersionList      map[int]bool
	VariantGroupList map[int]bool
	BaseVariantList  map[int]bool

	Crafted, Custom, Disabled, Enchant, Fractured, Implicit, Desecrated, Mutated, Rune, Unscalable, Prefix, Suffix bool

	Bonded                           bool
	BondedModList                    []mod.Mod
	AugmentType                      string
	RuneCount                        int
	SocketedRuneEffectAlreadyApplied bool
	SocketedAugmentTypeOverride      string
	SocketedSoulCoreType             string
}

func (ml *ModLine) setFlag(k string) {
	switch k {
	case "crafted":
		ml.Crafted = true
	case "custom":
		ml.Custom = true
	case "disabled":
		ml.Disabled = true
	case "enchant":
		ml.Enchant = true
	case "fractured":
		ml.Fractured = true
	case "implicit":
		ml.Implicit = true
	case "desecrated":
		ml.Desecrated = true
	case "mutated":
		ml.Mutated = true
	case "rune":
		ml.Rune = true
	case "unscalable":
		ml.Unscalable = true
	case "prefix":
		ml.Prefix = true
	case "suffix":
		ml.Suffix = true
	}
}

// ItemSocket is one rune socket.
type ItemSocket struct {
	Group int
}

// ItemRequirements is item.requirements; a *Mod field is nil until BuildModList sets it.
type ItemRequirements struct {
	Level        *float64
	NaturalLevel *float64
	RuneLevel    float64
	Str          float64
	Dex          float64
	Int          float64
	StrMod       *float64
	DexMod       *float64
	IntMod       *float64
	StrBase      *float64
	DexBase      *float64
	IntBase      *float64
}

// ItemGrantedSkill is one entry of item.grantedSkills.
type ItemGrantedSkill struct {
	SkillID       string
	Level         int
	NoSupports    bool
	NoReservation bool
	Source        string
	Triggered     bool
	TriggerChance float64
}

type modMagnitudeMod struct {
	tags       []string
	anyTags    []string
	quality    float64
	multiplier *float64
	modType    string
	sourceLine *ModLine
}

type selectionTags struct {
	line             string
	variantList      map[int]bool
	versionList      map[int]bool
	variantGroupList map[int]bool
	baseList         map[int]bool
}

type baseLine struct {
	line             string
	variantList      map[int]bool
	baseVariantList  map[int]bool
	versionList      map[int]bool
	variantGroupList map[int]bool
}

// Item is PoB-PoE2's Item class: an equippable item parsed from its text.
type Item struct {
	ID         int
	Raw        string
	Name       string
	NamePrefix string
	NameSuffix string
	Title      string
	BaseName   string
	Base       *poe2.ItemBase
	Rarity     string
	Type       string

	CharmLimit  *float64
	SpiritValue *float64
	RunicItem   bool
	Quality     *float64
	ItemLevel   *float64
	UniqueID    string

	CanBeAnointed, CanHaveTwoEnchants, CanHaveThreeEnchants, CanHaveFourEnchants bool
	ImplicitsCannotBeChanged                                                     bool

	Sanctified, Corruptible, Mirrored, Corrupted, DoubleCorrupted, Desecrated, Mutated, Fractured bool
	Crafted, Implicit, Unreleased, AllowDuplicateVariants, HiddenSpecs, AdvancedCopy              bool

	ClassRestriction string
	League           string
	Source           string
	Note             string
	UpgradePaths     []string
	Catalyst         int
	CatalystQuality  *float64
	TalismanTier     *float64
	Limit            *float64
	JewelRadiusLabel string
	JewelRadiusIndex int

	CheckSection bool
	// Prefixes and Suffixes are a crafted item's affixes.
	Prefixes, Suffixes []ItemAffix
	Sockets            []ItemSocket
	Runes              []string
	ItemSocketCount    int
	JewelSocketCount   int

	ClassRequirementModLines []*ModLine
	BuffModLines             []*ModLine
	EnchantModLines          []*ModLine
	RuneModLines             []*ModLine
	ImplicitModLines         []*ModLine
	ExplicitModLines         []*ModLine

	SocketedAugmentTypeOverride string
	SocketedSoulCoreTypes       map[string]bool

	Variant                                                                       *int
	VariantAlt, VariantAlt2, VariantAlt3, VariantAlt4, VariantAlt5                *int
	HasAltVariant, HasAltVariant2, HasAltVariant3, HasAltVariant4, HasAltVariant5 bool
	VariantList                                                                   []string
	BaseList                                                                      []string
	VersionList                                                                   []string
	SelectedVersion                                                               *int
	SelectedBase                                                                  *int
	VariantGroups                                                                 map[int]map[int]map[int]bool
	VariantGroupSelections                                                        map[int]int

	Requirements ItemRequirements
	baseLines    map[string]baseLine

	modMagnitudeMods []modMagnitudeMod

	CraftedQuality *float64

	WeaponData map[int]map[string]float64
	// WeaponDataStr holds the weaponData entries whose Lua value is a string.
	WeaponDataStr map[int]map[string]string
	ArmourData    map[string]float64
	FlaskData     map[string]float64
	CharmData     map[string]float64
	JewelData     map[string]any

	BaseModList  []mod.Mod
	ModList      []mod.Mod
	SlotModList  []([]mod.Mod)
	BuffModList  []mod.Mod
	RangeLines   []*ModLine
	HasModTags   bool
	GrantedSkill []ItemGrantedSkill

	SocketedIdolsUseBondedModifiers   bool
	SocketedSoulCoreEffectModifier    float64
	SocketedRuneEffectModifier        float64
	SocketedAugmentItemEffectModifier float64
	SocketedJewelEffectModifier       float64
	CanSocketJewelBase                map[string]bool

	modSourceName    string
	activeBondedSet  bool
	activeBonded     string
	modSource        ItemModSource
	rawLines         []string
	affixesAvailable bool
}

func intp(v int) *int           { return &v }
func floatp(v float64) *float64 { return &v }
func hasTag(b *poe2.ItemBase, tag string) bool {
	for _, t := range b.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ParseItem is new("Item"):Item(raw).
//
//pob2:src/Classes/Item.lua:93-98
func ParseItem(raw string, modSource ItemModSource) *Item {
	item := &Item{modSource: modSource, ID: -1}
	item.ParseRaw(sanitiseText(raw), "", false)
	return item
}

// ItemModRange is one <ModRange> child of a saved <Item>.
type ItemModRange struct {
	ID    int
	Range float64
}

// ItemVariants holds a saved <Item>'s variant attributes; nil means absent.
type ItemVariants struct {
	Variant                                                        *int
	VariantAlt, VariantAlt2, VariantAlt3, VariantAlt4, VariantAlt5 *int
}

// LoadItem loads one saved <Item> as ItemsTab:Load does: an empty item, its variant attributes, its
// text through ParseRaw (unsanitised), its ModRange children, then BuildModList.
//
//pob2:block:src/Classes/ItemsTab.lua:1227-1282
func LoadItem(id int, raw string, variants ItemVariants, modRanges []ItemModRange, modSource ItemModSource) *Item {
	item := &Item{modSource: modSource, ID: id}
	item.ParseRaw("", "", false)
	item.Variant = variants.Variant
	if variants.VariantAlt != nil {
		item.HasAltVariant, item.VariantAlt = true, variants.VariantAlt
	}
	if variants.VariantAlt2 != nil {
		item.HasAltVariant2, item.VariantAlt2 = true, variants.VariantAlt2
	}
	if variants.VariantAlt3 != nil {
		item.HasAltVariant3, item.VariantAlt3 = true, variants.VariantAlt3
	}
	if variants.VariantAlt4 != nil {
		item.HasAltVariant4, item.VariantAlt4 = true, variants.VariantAlt4
	}
	if variants.VariantAlt5 != nil {
		item.HasAltVariant5, item.VariantAlt5 = true, variants.VariantAlt5
	}
	item.ParseRaw(raw, "", false)
	for _, mr := range modRanges {
		id := mr.ID
		for _, list := range [][]*ModLine{item.BuffModLines, item.EnchantModLines, item.ImplicitModLines, item.ExplicitModLines} {
			if id <= len(list) {
				if id >= 1 {
					list[id-1].Range = floatp(mr.Range)
				}
				break
			}
			id -= len(list)
		}
	}
	fallBackJewelSocketCount := map[string]int{"Tabula Rasa": 6, "Sekhema's Resolve": 1}
	if item.Base != nil {
		if n, ok := fallBackJewelSocketCount[item.Title]; ok && item.JewelSocketCount == 0 {
			item.JewelSocketCount = n
		}
		item.BuildModList()
	}
	return item
}

// itemLines is raw:gmatch("%s*([^\n]*%S)").
func itemLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.Trim(line, " \t\n\v\f\r")
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

//pob2:src/Classes/Item.lua:32-63
func getCatalystScalar(catalystID int, ml *ModLine, quality *float64) float64 {
	if ml.Unscalable {
		return 1
	}
	if catalystID < 1 || catalystID > len(catalystTags) || len(ml.ModTags) == 0 {
		return 1
	}
	q := 20.0
	if quality != nil {
		q = *quality
	}
	tagLookup := map[string]bool{}
	for _, t := range ml.ModTags {
		tagLookup[t] = true
	}
	if ml.Prefix {
		tagLookup["prefix"] = true
	}
	if ml.Suffix {
		tagLookup["suffix"] = true
	}
	for _, catalystTag := range catalystTags[catalystID-1] {
		if tagLookup[catalystTag] {
			return (100 + q) / 100
		}
	}
	return 1
}

var grantsSkillRangeRe = regexp.MustCompile(`\(\d+-\d+\)`)

//pob2:src/Classes/Item.lua:116-126
func baseHasImplicitLine(base *poe2.ItemBase, line string) bool {
	if base == nil || base.Implicit == "" {
		return false
	}
	for _, implicitLine := range strings.Split(base.Implicit, "\n") {
		if implicitLine == "" || !strings.HasPrefix(implicitLine, "Grants Skill:") {
			continue
		}
		if implicitLine == line {
			return true
		}
		pat := grantsSkillRangeRe.ReplaceAllString(implicitLine, "%d+")
		if re, ok := luaPattern("^" + pat + "$"); ok && re.MatchString(line) {
			return true
		}
	}
	return false
}

var (
	specToNumberRe   = regexp.MustCompile(`^([+-]?[\d.]+)`)
	itemSpecRe       = regexp.MustCompile(`^([A-Za-z ()]+:?): (.+)$`)
	itemSpecReqRe    = regexp.MustCompile(`^(Requires [A-Za-z]+) (.+)$`)
	idSpecRe         = regexp.MustCompile(`\d+`)
	variantNameRe    = regexp.MustCompile(`\{([\w_]+)\}(.+)`)
	variantGroupRe   = regexp.MustCompile(`^(\d+)\s*=\s*(\d+)$`)
	variantTagRe     = regexp.MustCompile(`\{variant:([^}]*)\}`)
	versionTagRe     = regexp.MustCompile(`\{version:([^}]*)\}`)
	groupTagRe       = regexp.MustCompile(`\{group:([^}]*)\}`)
	baseTagRe        = regexp.MustCompile(`\{base:([^}]*)\}`)
	rarityLineRe     = regexp.MustCompile(`^Rarity: ([A-Za-z]+)`)
	reminderRe       = regexp.MustCompile(`^\([A-Za-z]+`)
	flaskRecoversRe  = regexp.MustCompile(`^Recovers .+ over .+ Seconds?$`)
	flaskConsumesRe  = regexp.MustCompile(`^Consumes \d+.*? of \d+.*? Charges on use$`)
	flaskCurrentRe   = regexp.MustCompile(`^Currently has \d+ Charges$`)
	advCopyFullRe    = regexp.MustCompile(`^\{ (.*?) - (.*?)  - (\d*).*\}$`)
	advCopyTagsRe    = regexp.MustCompile(`^\{ (.*?) - (.*?) \}$`)
	advCopyNameRe    = regexp.MustCompile(`^\{ (.*?) \}$`)
	advCopyModRe     = regexp.MustCompile(`^.*Modifier "(.*)"`)
	advCopyEnhRe     = regexp.MustCompile(`(.*)Enhancement.*`)
	advCopyFlagsRe   = regexp.MustCompile(`(.*)Modifier.*`)
	alphaWordRe      = regexp.MustCompile(`[A-Za-z]+`)
	whitespaceRe     = regexp.MustCompile(`\s+`)
	levelReqRe       = regexp.MustCompile(`^Requires:? Level (\d+)`)
	catalystSpecRe   = regexp.MustCompile(`Quality \([A-Za-z]+ Modifiers\)`)
	catalystNameRe   = regexp.MustCompile(`Quality \(([A-Za-z\s]+) Modifiers\)`)
	pctNumberRe      = regexp.MustCompile(`(\d+)%`)
	radiusLabelRe    = regexp.MustCompile(`^[A-Za-z ]+`)
	radiusWordRe     = regexp.MustCompile(`^[A-Za-z]+`)
	fracturedSpecRe  = regexp.MustCompile(`^\{fractured\}`)
	affixRangeRe     = regexp.MustCompile(`\{range:([^}]+)\}(.+)`)
	braceTagRe       = regexp.MustCompile(`\{([A-Za-z]*):?([^}]*)\}`)
	lowerFlagRe      = regexp.MustCompile(` \(([a-z]+)\)`)
	parenSuffixRe    = regexp.MustCompile(` \(.+\)`)
	superiorRe       = regexp.MustCompile(`^Superior `)
	bondedLineRe     = regexp.MustCompile(`^Bonded:\s+`)
	unscalableRe     = regexp.MustCompile(` - Unscalable Value$`)
	unscalableEmRe   = regexp.MustCompile(` — Unscalable Value$`)
	fixedValueRe     = regexp.MustCompile(`(-?\d+\.?\d*)\((-?\d+\.?\d*)\)`)
	advRollRe        = regexp.MustCompile(`(-?\d+\.?\d*)\((-?\d+\.?\d*--?\d+\.?\d*)\)`)
	advRollSplitRe   = regexp.MustCompile(`(-?\d+\.?\d*)-(-?\d+\.?\d*)`)
	advRollStripRe   = regexp.MustCompile(`(-?\d+\.?\d*)\(-?\d+\.?\d*--?\d+\.?\d*\)`)
	pendingRangeRe   = regexp.MustCompile(`-?\d+\.?\d*\(`)
	numberRe         = regexp.MustCompile(`-?\d+\.?\d*`)
	braceBlockRe     = regexp.MustCompile(`\{[^{}]*\}`)
	optLowerFlagRe   = regexp.MustCompile(` ?\([a-z]+\)`)
	prefixAllowedRe  = regexp.MustCompile(` prefix modifiers? allowed`)
	prefixPlusRe     = regexp.MustCompile(`\+(\d+) prefix modifiers? allowed`)
	prefixMinusRe    = regexp.MustCompile(`-(\d+) prefix modifiers? allowed`)
	suffixAllowedRe  = regexp.MustCompile(` suffix modifiers? allowed`)
	suffixPlusRe     = regexp.MustCompile(`\+(\d+) suffix modifiers? allowed`)
	suffixMinusRe    = regexp.MustCompile(`-(\d+) suffix modifiers? allowed`)
	augmentOverRe    = regexp.MustCompile(`^this item gains bonuses from socketed items as though it was a? ?(.+)$`)
	soulCoreTypeRe   = regexp.MustCompile(`^this item gains bonuses from socketed soul cores as though it was also a? ?(.+)$`)
	requiresClassRe  = regexp.MustCompile(`Requires Class (.+)`)
	variantStripRe   = regexp.MustCompile(`\{variant:([\d,]+)\}`)
	runeDigitsRe     = regexp.MustCompile(`(\d\.?\d*)`)
	runeBondedPrefix = "Bonded: "
	bondedStripRe    = regexp.MustCompile(`^Bonded:\s*`)
	normaliseNumRe   = regexp.MustCompile(`\d+\.?\d*`)
	normaliseRangeRe = regexp.MustCompile(`\(-?#-#\)`)
	runeforgedRe     = regexp.MustCompile(`Runeforged|Runemastered`)

	//pob2:block:src/Classes/Item.lua:1256-1260
	modMagnitudePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(\d+)% ([ir][ne][cd][ru][ec][ae][sd]e?d?) ?([A-Za-z\s]*) modifier magnitudes`),
		regexp.MustCompile(`(\d+)% ([ir][ne][cd][ru][ec][ae][sd]e?d?) effect of ([sp][ur][fe]fix)es`),
		regexp.MustCompile(`([A-Za-z\s]*) modifier magnitudes are doubled`),
	}
)

//pob2:src/Classes/Item.lua:324-327
func specToNumber(s string) *float64 {
	m := specToNumberRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	v, ok := luaToNumber(m[1])
	if !ok {
		return nil
	}
	return &v
}

func specToInt(s string) *int {
	v := specToNumber(s)
	if v == nil {
		return nil
	}
	return intp(int(*v))
}

//pob2:src/Classes/Item.lua:329-337
func parseItemSpec(line string) (string, string) {
	if m := itemSpecRe.FindStringSubmatch(line); m != nil {
		if m[1] == "Class:" {
			return "Requires Class", m[2]
		}
		return m[1], m[2]
	}
	if m := itemSpecReqRe.FindStringSubmatch(line); m != nil {
		return m[1], m[2]
	}
	return "", ""
}

//pob2:src/Classes/Item.lua:339-348
func parseIDSpec(spec string, positiveOnly bool) map[int]bool {
	ids := map[int]bool{}
	for _, s := range idSpecRe.FindAllString(spec, -1) {
		id, _ := strconv.Atoi(s)
		if !positiveOnly || id > 0 {
			ids[id] = true
		}
	}
	return ids
}

//pob2:src/Classes/Item.lua:350-358
var variantSelectionSpecNames = map[string]bool{
	"Version": true, "Variant": true, "Base Variant": true, "Selected Version": true,
	"Selected Variant Group": true, "Selected Variant": true, "Selected Base Variant": true,
}

//pob2:src/Classes/Item.lua:360-362
func (i *Item) HasVariantGroups() bool {
	return len(i.VariantGroups) > 0
}

//pob2:src/Classes/Item.lua:364-366
func (i *Item) HasIndependentVariants() bool {
	return i.VersionList != nil && i.VariantList != nil && !i.HasVariantGroups()
}

//pob2:src/Classes/Item.lua:368-370
func (i *Item) UsesVersionedOrGroupedVariants() bool {
	return i.VersionList != nil || i.HasVariantGroups()
}

//pob2:src/Classes/Item.lua:372-376
func (i *Item) isVariantGroupOptionEligible(groupID, variantID int) bool {
	versions := i.VariantGroups[groupID][variantID]
	if versions == nil {
		return false
	}
	return versions[0] || (i.SelectedVersion != nil && versions[*i.SelectedVersion])
}

func sortedIntKeys[V any](m map[int]V) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

//pob2:src/Classes/Item.lua:378-400
func (i *Item) getVariantGroupOptions(groupID int, excludeSelected bool) []int {
	var options []int
	if i.VariantGroups[groupID] == nil {
		return options
	}
	used := map[int]bool{}
	if excludeSelected {
		for _, otherGroupID := range sortedIntKeys(i.VariantGroups) {
			if otherGroupID != groupID {
				if variantID, ok := i.VariantGroupSelections[otherGroupID]; ok && i.isVariantGroupOptionEligible(otherGroupID, variantID) {
					used[variantID] = true
				}
			}
		}
	}
	for variantID := 1; variantID <= len(i.VariantList); variantID++ {
		if i.isVariantGroupOptionEligible(groupID, variantID) && !used[variantID] {
			options = append(options, variantID)
		}
	}
	return options
}

//pob2:src/Classes/Item.lua:402-443
func (i *Item) normaliseVariantSelections() {
	if len(i.VersionList) > 0 {
		sel := len(i.VersionList)
		if i.SelectedVersion != nil {
			sel = *i.SelectedVersion
		}
		i.SelectedVersion = intp(max(1, min(len(i.VersionList), sel)))
	} else {
		i.SelectedVersion = nil
	}
	if i.HasIndependentVariants() {
		v := len(i.VariantList)
		if i.Variant != nil {
			v = *i.Variant
		}
		i.Variant = intp(max(1, min(len(i.VariantList), v)))
	}
	if i.VariantGroupSelections == nil {
		i.VariantGroupSelections = map[int]int{}
	}
	for groupID := range i.VariantGroupSelections {
		if i.VariantGroups[groupID] == nil {
			delete(i.VariantGroupSelections, groupID)
		}
	}
	used := map[int]bool{}
	var needsSelection []int
	for _, groupID := range sortedIntKeys(i.VariantGroups) {
		if len(i.getVariantGroupOptions(groupID, false)) > 0 {
			selected, ok := i.VariantGroupSelections[groupID]
			if ok && i.isVariantGroupOptionEligible(groupID, selected) && !used[selected] {
				used[selected] = true
			} else {
				needsSelection = append(needsSelection, groupID)
			}
		}
	}
	for _, groupID := range needsSelection {
		selected := 0
		for _, variantID := range i.getVariantGroupOptions(groupID, false) {
			if !used[variantID] {
				selected = variantID
				break
			}
		}
		if selected != 0 {
			i.VariantGroupSelections[groupID] = selected
			used[selected] = true
		} else {
			delete(i.VariantGroupSelections, groupID)
		}
	}
}

// ParseRaw parses the item text; rarity "" is Lua's nil.
//
//pob2:src/Classes/Item.lua:472-1826
//nolint:gocyclo,maintidx // mirrors ParseRaw
func (i *Item) ParseRaw(raw, rarity string, highQuality bool) {
	i.Raw = raw
	i.Name = "?"
	i.NamePrefix = ""
	i.NameSuffix = ""
	i.Base = nil
	i.Rarity = rarity
	if i.Rarity == "" {
		i.Rarity = "UNIQUE"
	}
	i.CharmLimit = nil
	i.SpiritValue = nil
	i.RunicItem = false
	i.Quality = nil
	i.CanBeAnointed = false
	i.rawLines = nil
	for _, line := range itemLines(raw) {
		i.rawLines = append(i.rawLines, escapeGGGString(line))
	}
	mode := "WIKI"
	if rarity != "" {
		mode = "GAME"
	}
	l := 0
	at := func(n int) (string, bool) {
		if n < len(i.rawLines) {
			return i.rawLines[n], true
		}
		return "", false
	}
	itemClass := ""
	if line, ok := at(l); ok {
		if strings.HasPrefix(line, "Item Class:") {
			itemClass = line
			l++
		}
		if line, ok := at(l); ok {
			if m := rarityLineRe.FindStringSubmatch(line); m != nil {
				mode = "GAME"
				if colorCodeNames[strings.ToUpper(m[1])] {
					i.Rarity = strings.ToUpper(m[1])
				}
				if i.Rarity == "UNIQUE" {
					for _, rl := range i.rawLines {
						if strings.Contains(rl, "Foil Unique") {
							i.Rarity = "RELIC"
							break
						}
					}
				}
				l++
			}
		}
	}
	if line, ok := at(l); ok {
		if line == "--------" {
			l++
		}
		unidentified := false
		cur, _ := at(l)
		if i.Rarity == "UNIQUE" {
			next, _ := at(l + 1)
			unidentifiedBase, uok := poe2.BaseByName(cur)
			_, iok := poe2.BaseByName(next)
			if uok && !iok {
				unidentified = true
				i.Name = "Unidentified item"
				i.BaseName = cur
				i.Base = unidentifiedBase
			} else {
				i.Name = cur
			}
		} else {
			i.Name = cur
		}
		for _, rl := range i.rawLines {
			if rl == "Unidentified" {
				unidentified = true
				break
			}
		}
		if !(i.Rarity == "NORMAL" || i.Rarity == "MAGIC" || unidentified) {
			l++
		}
	}
	i.CheckSection = false
	i.Sockets = nil
	i.Runes = nil
	i.ItemSocketCount = 0
	i.JewelSocketCount = 0
	i.ClassRequirementModLines = nil
	i.BuffModLines = nil
	i.EnchantModLines = nil
	i.RuneModLines = nil
	i.SocketedAugmentTypeOverride = ""
	i.SocketedSoulCoreTypes = map[string]bool{}
	i.ImplicitModLines = nil
	i.ExplicitModLines = nil
	i.AdvancedCopy = false
	i.modMagnitudeMods = nil
	implicitLines := 0
	skippedRuneLines := 0
	i.VariantList = nil
	i.BaseList = nil
	i.VersionList = nil
	i.VariantGroups = map[int]map[int]map[int]bool{}
	if i.VariantGroupSelections == nil {
		i.VariantGroupSelections = map[int]int{}
	}

	//pob2:block:src/Classes/Item.lua:582-656
	selectionTagsByLine := map[int]*selectionTags{}
	for lineIndex, rawLine := range i.rawLines {
		specName, specVal := parseItemSpec(rawLine)
		if variantSelectionSpecNames[specName] {
			switch specName {
			case "Version":
				i.VersionList = append(i.VersionList, specVal)
			case "Base Variant":
				if i.SelectedBase == nil {
					i.SelectedBase = intp(1)
				}
				i.BaseList = append(i.BaseList, specVal)
			case "Variant":
				name := specVal
				if m := variantNameRe.FindStringSubmatch(specVal); m != nil {
					name = m[2]
				}
				i.VariantList = append(i.VariantList, name)
			case "Selected Version":
				i.SelectedVersion = specToInt(specVal)
			case "Selected Variant Group":
				if m := variantGroupRe.FindStringSubmatch(specVal); m != nil {
					g, _ := strconv.Atoi(m[1])
					v, _ := strconv.Atoi(m[2])
					i.VariantGroupSelections[g] = v
				}
			case "Selected Variant":
				i.Variant = specToInt(specVal)
			case "Selected Base Variant":
				i.SelectedBase = specToInt(specVal)
			}
		}
		variantSpec := variantTagRe.FindStringSubmatch(rawLine)
		versionSpec := versionTagRe.FindStringSubmatch(rawLine)
		groupSpec := groupTagRe.FindStringSubmatch(rawLine)
		baseSpec := baseTagRe.FindStringSubmatch(rawLine)
		if variantSpec != nil || versionSpec != nil || groupSpec != nil || baseSpec != nil {
			st := &selectionTags{line: rawLine}
			if variantSpec != nil {
				st.variantList = parseIDSpec(variantSpec[1], false)
			}
			if versionSpec != nil {
				st.versionList = parseIDSpec(versionSpec[1], false)
			}
			if groupSpec != nil {
				st.variantGroupList = parseIDSpec(groupSpec[1], true)
			}
			if baseSpec != nil {
				st.baseList = parseIDSpec(baseSpec[1], false)
			}
			selectionTagsByLine[lineIndex] = st
		}
	}
	for _, idx := range sortedIntKeys(selectionTagsByLine) {
		st := selectionTagsByLine[idx]
		if st.variantGroupList != nil && len(st.variantList) == 0 {
			continue
		} else if st.variantGroupList != nil {
			for groupID := range st.variantGroupList {
				group := i.VariantGroups[groupID]
				if group == nil {
					group = map[int]map[int]bool{}
					i.VariantGroups[groupID] = group
				}
				for variantID := range st.variantList {
					if variantID >= 1 && variantID <= len(i.VariantList) {
						versions := group[variantID]
						if versions == nil {
							versions = map[int]bool{}
							group[variantID] = versions
						}
						if st.versionList != nil {
							for versionID := range st.versionList {
								if versionID >= 1 && versionID <= len(i.VersionList) {
									versions[versionID] = true
								}
							}
						} else {
							versions[0] = true
						}
					}
				}
			}
		}
	}
	if i.UsesVersionedOrGroupedVariants() {
		i.normaliseVariantSelections()
	}
	i.Prefixes, i.Suffixes = nil, nil
	i.Requirements = ItemRequirements{}
	i.baseLines = map[string]baseLine{}
	var importedLevelReq *float64
	var charmBuffLines map[string]bool
	deferJewelRadiusIndexAssignment := false
	gameModeStage := "FINDIMPLICIT"
	foundExplicit, foundImplicit := false, false
	linePrefix, linePostfix := "", ""
	for ; l < len(i.rawLines); l++ {
		line := i.rawLines[l]
		switch {
		case charmBuffLines != nil && charmBuffLines[line]:
			delete(charmBuffLines, line)
		case line == "--------":
			linePrefix, linePostfix = "", ""
			i.CheckSection = true
		case line == "Sanctified":
			i.Sanctified = true
			i.Corruptible = false
		case line == "Mirrored":
			i.Mirrored = true
		case line == "Corrupted":
			i.Corrupted = true
		case line == "Twice Corrupted":
			i.Corrupted = true
			i.DoubleCorrupted = true
		case line == "Desecrated Prefix" || line == "Desecrated Suffix":
			i.Desecrated = true
		case line == "Requirements:":
		case reminderRe.MatchString(line):
			for l < len(i.rawLines) && !strings.HasSuffix(i.rawLines[l], ")") {
				l++
			}
		case i.Base != nil && i.Base.Flask != nil && (flaskRecoversRe.MatchString(line) || flaskConsumesRe.MatchString(line) || flaskCurrentRe.MatchString(line)):
		case strings.HasPrefix(line, "{ "):
			//pob2:block:src/Classes/Item.lua:708-768
			i.AdvancedCopy = true
			linePrefix, linePostfix = "", ""
			i.Crafted = true
			var fullModName, modTags string
			if m := advCopyFullRe.FindStringSubmatch(line); m != nil {
				fullModName, modTags = m[1], m[2]
			} else if m := advCopyTagsRe.FindStringSubmatch(line); m != nil {
				fullModName, modTags = m[1], m[2]
			} else if m := advCopyNameRe.FindStringSubmatch(line); m != nil {
				fullModName = m[1]
			}
			modName := ""
			if m := advCopyModRe.FindStringSubmatch(fullModName); m != nil {
				modName = m[1]
			}
			// self.affixes (data.itemMods) is not carried by data/poe2, so no affix is matched.
			if modName != "" && i.affixesAvailable {
				linePrefix = "{custom}"
			} else if advCopyEnhRe.MatchString(fullModName) {
				linePostfix = " (enchant)"
			}
			if m := advCopyFlagsRe.FindStringSubmatch(strings.ReplaceAll(fullModName, "Vaal Unique", "Mutated")); m != nil {
				for _, flag := range alphaWordRe.FindAllString(m[1], -1) {
					if fl := strings.ToLower(flag); lineFlags[fl] {
						linePrefix += "{" + fl + "}"
					}
				}
			}
			if modTags != "" {
				linePrefix += "{tags:" + whitespaceRe.ReplaceAllString(strings.ToLower(modTags), "") + "}"
			}
		default:
			i.parseLine(&l, linePrefix+line+linePostfix, mode, itemClass, &implicitLines, &skippedRuneLines,
				&gameModeStage, &foundExplicit, &foundImplicit, &importedLevelReq, &charmBuffLines,
				&deferJewelRadiusIndexAssignment, selectionTagsByLine)
		}
	}
	if runeforgedRe.MatchString(i.BaseName) {
		i.RunicItem = true
	}
	if i.BaseName != "" && i.Title != "" {
		i.Name = i.Title + ", " + parenSuffixRe.ReplaceAllString(i.BaseName, "")
	}
	if i.Base != nil {
		i.parseRunes(mode)
	}
	if i.AdvancedCopy && len(i.ExplicitModLines) > 1 {
		sortCraftedModLines(i.ExplicitModLines)
	}
	if i.AdvancedCopy || i.Crafted {
		i.applyModMagnitude()
	}

	//pob2:block:src/Classes/Item.lua:1722-1750
	for _, runeName := range i.Runes {
		for _, slotData := range poe2.RuneModsByName(runeName) {
			i.Requirements.RuneLevel = math.Max(i.Requirements.RuneLevel, slotData.LevelReq)
		}
	}
	if i.Base != nil {
		// GetUniqueDBItem needs main.uniqueDB, which data/poe2 does not carry: every item takes the
		// no-database branch.
		if i.Requirements.Level == nil {
			if importedLevelReq != nil && len(i.Sockets) == 0 {
				i.Requirements.Level = importedLevelReq
			} else if i.Base.Req.Level != 0 {
				i.Requirements.Level = floatp(i.Base.Req.Level)
			}
		}
		i.Requirements.NaturalLevel = i.Requirements.Level
		if i.Requirements.NaturalLevel == nil {
			i.Requirements.NaturalLevel = floatp(0)
		}
		if i.Requirements.Level == nil {
			i.Requirements.Level = i.Requirements.NaturalLevel
		}
		i.Requirements.Level = floatp(math.Max(math.Max(*i.Requirements.Level, *i.Requirements.NaturalLevel), i.Requirements.RuneLevel))
	}
	//pob2:block:src/Classes/Item.lua:1793-1810
	if !i.UsesVersionedOrGroupedVariants() && i.VariantList != nil {
		n := len(i.VariantList)
		clamp := func(p *int) *int {
			if p == nil {
				return intp(n)
			}
			return intp(min(n, *p))
		}
		i.Variant = clamp(i.Variant)
		if i.HasAltVariant {
			i.VariantAlt = clamp(i.VariantAlt)
		}
		if i.HasAltVariant2 {
			i.VariantAlt2 = clamp(i.VariantAlt2)
		}
		if i.HasAltVariant3 {
			i.VariantAlt3 = clamp(i.VariantAlt3)
		}
		if i.HasAltVariant4 {
			i.VariantAlt4 = clamp(i.VariantAlt4)
		}
		if i.HasAltVariant5 {
			i.VariantAlt5 = clamp(i.VariantAlt5)
		}
	}
	if i.Quality == nil {
		i.NormaliseQuality()
		if highQuality {
			i.NormaliseQuality()
		}
	}
	i.BuildModList()
	if deferJewelRadiusIndexAssignment {
		if v, ok := i.JewelData["radiusIndex"].(float64); ok {
			i.JewelRadiusIndex = int(v)
		} else {
			i.JewelRadiusIndex = 0
		}
	}
	if v, ok := i.JewelData["timeLostJewelRadiusOverride"].(float64); ok {
		i.JewelRadiusIndex = int(v)
	}
}

// parseLine is ParseRaw's else-branch for one line; a return is Lua's `goto continue`.
//
//pob2:src/Classes/Item.lua:769-1375
//nolint:gocyclo,maintidx // mirrors ParseRaw's line branch
func (i *Item) parseLine(lp *int, line, mode, itemClass string, implicitLines, skippedRuneLines *int,
	gameModeStage *string, foundExplicit, foundImplicit *bool, importedLevelReq **float64,
	charmBuffLines *map[string]bool, deferJewelRadius *bool, selectionTagsByLine map[int]*selectionTags,
) {
	l := *lp
	lineIsBaseImplicit := mode == "GAME" && !i.Crafted && baseHasImplicitLine(i.Base, line)
	if i.CheckSection {
		switch {
		case *gameModeStage == "IMPLICIT":
			if *foundImplicit && !lineIsBaseImplicit {
				*gameModeStage = "EXPLICIT"
				*foundExplicit = true
			} else {
				*gameModeStage = "FINDEXPLICIT"
			}
		case *gameModeStage == "EXPLICIT":
			*gameModeStage = "DONE"
		case *gameModeStage == "FINDIMPLICIT" && i.ItemLevel != nil && !strings.Contains(line, " (implicit)") &&
			!strings.Contains(line, " (enchant)") && !strings.Contains(line, "Talisman Tier"):
			*gameModeStage = "EXPLICIT"
			*foundExplicit = true
		}
		i.CheckSection = false
	}
	if m := levelReqRe.FindStringSubmatch(line); m != nil {
		v, _ := luaToNumber(m[1])
		i.Requirements.Level = &v
		return
	}
	specName, specVal := parseItemSpec(line)
	if specName != "" {
		//pob2:block:src/Classes/Item.lua:796-972
		switch {
		case specName == "Unique ID":
			i.UniqueID = specVal
			return
		case specName == "Item Level":
			i.ItemLevel = specToNumber(specVal)
		case specName == "Requires Class":
			i.ClassRestriction = specVal
		case specName == "Charm Slots":
			i.CharmLimit = specToNumber(specVal)
		case specName == "Spirit":
			i.SpiritValue = specToNumber(specVal)
		case catalystSpecRe.MatchString(specName):
			q := ""
			if m := pctNumberRe.FindStringSubmatch(specVal); m != nil {
				q = m[1]
			}
			i.CatalystQuality = specToNumber(q)
			if m := catalystNameRe.FindStringSubmatch(specName); m != nil {
				for idx, d := range catalystDescriptorList {
					if m[1] == d {
						i.Catalyst = idx + 1
					}
				}
			}
		case specName == "Quality":
			i.Quality = specToNumber(specVal)
		case specName == "Sockets":
			group := 0
			for _, c := range specVal {
				if c == 'S' {
					i.Sockets = append(i.Sockets, ItemSocket{Group: group})
					group++
				} else if c == 'J' {
					i.JewelSocketCount++
				}
			}
			i.ItemSocketCount = len(i.Sockets)
		case specName == "Rune":
			i.Runes = append(i.Runes, specVal)
		case specName == "Radius" && i.Type == "Jewel":
			i.JewelRadiusLabel = radiusLabelRe.FindString(specVal)
			if radiusWordRe.FindString(specVal) == "Variable" {
				*deferJewelRadius = true
			} else {
				for _, idx := range sortedIntKeys(jewelRadiusLabels) {
					if radiusLabelRe.FindString(specVal) == jewelRadiusLabels[idx] {
						i.JewelRadiusIndex = idx
						break
					}
				}
			}
		case specName == "Limited to" && i.Type == "Jewel":
			i.Limit = specToNumber(specVal)
		case variantSelectionSpecNames[specName]:
		case specName == "Talisman Tier":
			i.TalismanTier = specToNumber(specVal)
		case specName == "Armour" || specName == "Evasion Rating" || specName == "Evasion" || specName == "Energy Shield" || specName == "Ward" || specName == "Runic Ward":
			switch specName {
			case "Evasion Rating":
				specName = "Evasion"
				if i.BaseName == "Two-Toned Boots (Armour/Energy Shield)" {
					i.BaseName = "Two-Toned Boots (Armour/Evasion)"
					i.Base, _ = poe2.BaseByName(i.BaseName)
				}
			case "Energy Shield":
				specName = "EnergyShield"
				if i.BaseName == "Two-Toned Boots (Armour/Evasion)" {
					i.BaseName = "Two-Toned Boots (Evasion/Energy Shield)"
					i.Base, _ = poe2.BaseByName(i.BaseName)
				}
			case "Runic Ward":
				specName = "Ward"
			}
			if i.ArmourData == nil {
				i.ArmourData = map[string]float64{}
			}
			if v := specToNumber(specVal); v != nil {
				i.ArmourData[specName] = *v
			} else {
				delete(i.ArmourData, specName)
			}
		case specName == "Level":
			*importedLevelReq = specToNumber(specVal)
		case specName == "Requires Level" || specName == "LevelReq":
			i.Requirements.Level = specToNumber(specVal)
		case specName == "Has Alt Variant":
			i.HasAltVariant = true
		case specName == "Has Alt Variant Two":
			i.HasAltVariant2 = true
		case specName == "Has Alt Variant Three":
			i.HasAltVariant3 = true
		case specName == "Has Alt Variant Four":
			i.HasAltVariant4 = true
		case specName == "Has Alt Variant Five":
			i.HasAltVariant5 = true
		case specName == "Selected Alt Variant":
			i.VariantAlt = specToInt(specVal)
		case specName == "Selected Alt Variant Two":
			i.VariantAlt2 = specToInt(specVal)
		case specName == "Selected Alt Variant Three":
			i.VariantAlt3 = specToInt(specVal)
		case specName == "Selected Alt Variant Four":
			i.VariantAlt4 = specToInt(specVal)
		case specName == "Selected Alt Variant Five":
			i.VariantAlt5 = specToInt(specVal)
		case specName == "Selected Base Variant":
			i.SelectedBase = specToInt(specVal)
		case specName == "Allow Duplicate Variants":
			i.AllowDuplicateVariants = specVal == "true"
		case specName == "Has Variants" || specName == "Selected Variants":
			l++
			*lp = l
		case specName == "League":
			i.League = specVal
		case specName == "Crafted":
			i.Crafted = true
		case specName == "Implicit":
			i.Implicit = true
		case specName == "Prefix" || specName == "Suffix":
			//pob2:block:src/Classes/Item.lua:910-931
			affixes := &i.Prefixes
			if specName == "Suffix" {
				affixes = &i.Suffixes
			}
			fractured := strings.HasPrefix(specVal, "{fractured}")
			specVal = strings.TrimPrefix(specVal, "{fractured}")
			affix := ItemAffix{ModID: specVal, Fractured: fractured}
			if m := affixRangeRe.FindStringSubmatch(specVal); m != nil {
				affix.ModID = m[2]
				if strings.Contains(m[1], ",") {
					affix.Ranges = []float64{}
					for _, v := range strings.Split(m[1], ",") {
						if v == "" {
							continue
						}
						if n, ok := luaToNumber(v); ok {
							affix.Ranges = append(affix.Ranges, n)
						}
					}
				} else if n, ok := luaToNumber(m[1]); ok {
					affix.Range = &n
				}
			}
			if affix.Range == nil && affix.Ranges == nil && affix.ModID != "None" {
				affix.Range = floatp(defaultItemAffixQuality)
			}
			*affixes = append(*affixes, affix)
		case specName == "Implicits":
			if v := specToNumber(specVal); v != nil {
				*implicitLines = int(*v)
			} else {
				*implicitLines = 0
			}
			*gameModeStage = "EXPLICIT"
		case specName == "Unreleased":
			i.Unreleased = specVal == "true"
		case specName == "Upgrade":
			i.UpgradePaths = append(i.UpgradePaths, specVal)
		case specName == "Source":
			i.Source = specVal
		case specName == "Cluster Jewel Skill" || specName == "Cluster Jewel Node Count":
		case specName == "Catalyst":
			for idx, c := range catalystList {
				if specVal == c {
					i.Catalyst = idx + 1
				}
			}
		case specName == "CatalystQuality":
			i.CatalystQuality = specToNumber(specVal)
		case specName == "Note":
			i.Note = specVal
		case specName == "Critical Hit Range" || specName == "Attacks per Second" || specName == "Weapon Range" ||
			specName == "Critical Hit Chance" || specName == "Physical Damage" || specName == "Elemental Damage" ||
			specName == "Chaos Damage" || specName == "Fire Damage" || specName == "Cold Damage" || specName == "Lightning Damage" ||
			specName == "Reload Time" || specName == "Chance to Block" || specName == "Block chance" ||
			specName == "Armour" || specName == "Energy Shield" || specName == "Evasion" || specName == "Requires":
			i.HiddenSpecs = true
		case !lineIsBaseImplicit && !(luaFind(i.Name, specName) && luaFind(i.Name, specVal)):
			*foundExplicit = true
			*gameModeStage = "EXPLICIT"
		}
	}
	if line == "Prefixes:" {
		*foundExplicit = true
		*gameModeStage = "EXPLICIT"
	}
	if !(specName == "" || *foundExplicit || *foundImplicit || lineIsBaseImplicit) {
		return
	}
	ml := &ModLine{ModTags: []string{}}
	st := selectionTagsByLine[l]
	line = gsubFunc(braceTagRe, line, -1, func(g []string) string {
		k, val := g[1], g[2]
		switch {
		case k == "variant":
			if st != nil && st.variantList != nil {
				ml.VariantList = st.variantList
			} else {
				ml.VariantList = parseIDSpec(val, false)
			}
		case k == "version":
			if st != nil && st.versionList != nil {
				ml.VersionList = st.versionList
			} else {
				ml.VersionList = parseIDSpec(val, false)
			}
		case k == "base":
			if st != nil && st.baseList != nil {
				ml.BaseVariantList = st.baseList
			} else {
				ml.BaseVariantList = parseIDSpec(val, false)
			}
		case k == "group":
			if st != nil && st.variantGroupList != nil {
				ml.VariantGroupList = st.variantGroupList
			} else {
				ml.VariantGroupList = parseIDSpec(val, true)
			}
		case k == "tags":
			ml.ModTags = append(ml.ModTags, tagWordRe.FindAllString(val, -1)...)
		case k == "range":
			i.AdvancedCopy = true
			if v, ok := luaToNumber(val); ok {
				ml.Range = &v
			} else {
				ml.Range = nil
			}
		case k == "corruptedRange":
			if v, ok := luaToNumber(val); ok {
				ml.CorruptedRange = &v
			} else {
				ml.CorruptedRange = nil
			}
		case lineFlags[k]:
			ml.setFlag(k)
		}
		return ""
	})
	line = gsubFunc(lowerFlagRe, line, -1, func(g []string) string {
		if lineFlags[g[1]] {
			ml.setFlag(g[1])
		}
		return ""
	})
	if ml.Rune {
		ml.Enchant = true
	}
	if ml.Enchant {
		ml.Implicit = true
	}
	if lineIsBaseImplicit {
		ml.Implicit = true
	}
	if ml.Desecrated {
		i.Desecrated = true
	}
	if ml.Mutated {
		i.Mutated = true
	}
	if ml.Fractured {
		i.Fractured = true
	}

	//pob2:block:src/Classes/Item.lua:1031-1129
	baseName := ""
	if i.Base == nil && (i.Rarity == "NORMAL" || i.Rarity == "MAGIC") {
		if strings.Contains(i.Name, "Energy Blade") && itemClass != "" {
			if strings.Contains(itemClass, "One Hand") {
				i.Name = "Energy Blade One Handed"
			} else {
				i.Name = "Energy Blade Two Handed"
			}
		}
		if _, ok := poe2.BaseByName(i.Name); ok {
			baseName = i.Name
		} else {
			bestLen, bestS, bestE, best := -1, 0, 0, ""
			for idx := range poe2.ItemBases {
				name := poe2.ItemBases[idx].Name
				if s := strings.Index(i.Name, name); s >= 0 && name != "" {
					e := s + len(name) - 1
					if e-s > bestLen {
						best, bestLen, bestS, bestE = name, e-s, s, e
					}
				}
			}
			if best != "" {
				i.NamePrefix = i.Name[:bestS]
				i.NameSuffix = i.Name[bestE+1:]
				baseName = best
			}
		}
		if baseName == "" {
			if s := strings.Index(i.Name, "Two-Toned Boots"); s >= 0 {
				baseName = "Two-Toned Boots"
				i.NamePrefix = i.Name[:s]
				i.NameSuffix = i.Name[s+len("Two-Toned Boots"):]
			}
		}
		i.Name = parenSuffixRe.ReplaceAllString(i.Name, "")
	}
	if baseName == "" {
		baseName = superiorRe.ReplaceAllString(line, "")
	}
	if baseName == "Two-Toned Boots" {
		baseName = "Two-Toned Boots (Armour/Energy Shield)"
	}
	if base, ok := poe2.BaseByName(baseName); ok {
		i.baseLines[baseName] = baseLine{
			line: baseName, variantList: ml.VariantList, baseVariantList: ml.BaseVariantList,
			versionList: ml.VersionList, variantGroupList: ml.VariantGroupList,
		}
		uses := i.UsesVersionedOrGroupedVariants()
		var baseMatches bool
		if uses {
			baseMatches = i.CheckModLineVariant(ml)
		} else {
			baseMatches = (i.Variant == nil || ml.VariantList == nil || ml.VariantList[*i.Variant]) &&
				(i.SelectedBase == nil || ml.BaseVariantList == nil || ml.BaseVariantList[*i.SelectedBase])
		}
		if baseMatches {
			i.BaseName = baseName
			if !(i.Rarity == "NORMAL" || i.Rarity == "MAGIC") {
				i.Title = i.Name
			}
			i.Type = base.Type
			i.Base = base
			i.CharmLimit = nil
			if base.HasCharmLimit {
				i.CharmLimit = floatp(base.CharmLimit)
			}
			i.SpiritValue = nil
			if base.Spirit != 0 {
				i.SpiritValue = floatp(base.Spirit)
			}
			i.Corruptible = base.Type != "Flask" && base.Type != "Charm" && base.Type != "Transcendent Limb"
			i.Requirements.Str = base.Req.Str
			i.Requirements.Dex = base.Req.Dex
			i.Requirements.Int = base.Req.Int
			if base.Charm != nil && base.Charm.Buff != nil && *charmBuffLines == nil {
				*charmBuffLines = map[string]bool{}
				for _, bl := range base.Charm.Buff {
					(*charmBuffLines)[bl] = true
					modList, extra := i.modSource(bl)
					if modList == nil {
						modList = []mod.Mod{}
					}
					i.BuffModLines = append(i.BuffModLines, &ModLine{Line: bl, Extra: extra, ModList: modList})
				}
			}
		}
		return
	}
	if ml.Implicit {
		*foundImplicit = true
		*gameModeStage = "IMPLICIT"
	}
	if ml.Rune && !ml.Disabled && bondedLineRe.MatchString(line) {
		*skippedRuneLines++
		return
	}
	catalystScalar := 1.0
	if unscalableRe.MatchString(line) || unscalableEmRe.MatchString(line) {
		line = unscalableRe.ReplaceAllString(line, "")
		line = unscalableEmRe.ReplaceAllString(line, "")
		ml.Unscalable = true
	} else {
		catalystScalar = getCatalystScalar(i.Catalyst, ml, i.CatalystQuality)
	}
	line = fixedValueRe.ReplaceAllString(line, "$1")

	//pob2:block:src/Classes/Item.lua:1199-1237
	bestPrecisionDelta, bestPrecisionRange := -1.0, -1.0
	var firstRollRange *float64
	hasIndependentRolls := false
	line = stripEnumRanges(line)
	advancedCopyLine := line
	for _, g := range advRollRe.FindAllStringSubmatch(advancedCopyLine, -1) {
		value, rng := g[1], g[2]
		mm := advRollSplitRe.FindStringSubmatch(rng)
		lo, _ := luaToNumber(mm[1])
		hi, _ := luaToNumber(mm[2])
		v, _ := luaToNumber(value)
		delta := hi - lo
		rollRange := 0.5
		if delta > 0 {
			rollRange = round((v-lo)/delta, 6)
		}
		if firstRollRange != nil && *firstRollRange != rollRange {
			hasIndependentRolls = true
		}
		if firstRollRange == nil {
			firstRollRange = floatp(rollRange)
		}
		if delta > bestPrecisionDelta {
			bestPrecisionRange = rollRange
			bestPrecisionDelta = delta
		}
		re, ok := luaPattern(luaPatternQuoteNumber(value) + "%(" + strings.ReplaceAll(rng, "-", "%-") + "%)")
		if !ok {
			continue
		}
		if bestPrecisionRange > 1 || bestPrecisionRange < 0 {
			line = re.ReplaceAllLiteralString(line, value)
		} else {
			sign := ""
			if v < 0 {
				sign = "+"
			}
			line = re.ReplaceAllLiteralString(line, sign+"("+mm[1]+"-"+mm[2]+")")
		}
	}
	if hasIndependentRolls {
		line = advRollStripRe.ReplaceAllString(advancedCopyLine, "$1")
	} else if bestPrecisionRange <= 1 && bestPrecisionRange >= 0 {
		ml.Range = floatp(bestPrecisionRange)
	}

	rangedLine := applyRange(line, 1, nil, &catalystScalar, ml.CorruptedRange, i.modSource)
	modList, extra := i.modSource(rangedLine)
	if (modList == nil || extra != "") && l+1 < len(i.rawLines) {
		nextLine := stripBalanced(i.rawLines[l+1], '{', '}')
		nextLine = optLowerFlagRe.ReplaceAllString(nextLine, "")
		combLine := line + " " + nextLine
		rangedLine = applyRange(combLine, 1, nil, &catalystScalar, ml.CorruptedRange, i.modSource)
		modList, extra = i.modSource(rangedLine)
		if modList != nil && extra == "" {
			line = line + "\n" + nextLine
			l++
			*lp = l
		} else {
			modList, extra = i.modSource(rangedLine)
		}
	}

	lineLower := ""
	if !ml.Disabled {
		lineLower = strings.ToLower(line)
	}
	switch {
	case lineLower == "implicit modifiers cannot be changed":
		i.ImplicitsCannotBeChanged = true
	case prefixAllowedRe.MatchString(lineLower), suffixAllowedRe.MatchString(lineLower):
	case lineLower == "this item can be anointed by cassia" || lineLower == "raven-touched":
		i.CanBeAnointed = true
	case lineLower == "can have 1 additional instilled modifier" || lineLower == "can have an additional instilled modifier":
		i.CanHaveTwoEnchants = true
	case lineLower == "can have 2 additional instilled modifiers":
		i.CanHaveTwoEnchants = true
		i.CanHaveThreeEnchants = true
	case lineLower == "can have 3 additional instilled modifiers":
		i.CanHaveTwoEnchants = true
		i.CanHaveThreeEnchants = true
		i.CanHaveFourEnchants = true
	}
	if m := augmentOverRe.FindStringSubmatch(lineLower); m != nil {
		ml.SocketedAugmentTypeOverride = m[1]
	}
	if m := soulCoreTypeRe.FindStringSubmatch(lineLower); m != nil {
		ml.SocketedSoulCoreType = m[1]
	}

	//pob2:block:src/Classes/Item.lua:1283-1322
	modMagnitudeTagMap := map[string]string{"defence": "defences", "enchantment": "enchant"}
	for _, pattern := range modMagnitudePatterns {
		if ml.Disabled || !pattern.MatchString(strings.ToLower(rangedLine)) {
			continue
		}
		r := defaultItemAffixQuality
		if ml.Range != nil {
			r = *ml.Range
		}
		ranged := applyRange(line, r, nil, &catalystScalar, ml.CorruptedRange, i.modSource)
		m := pattern.FindStringSubmatch(strings.ToLower(ranged))
		if m == nil {
			continue
		}
		var amount, incDec, modTagsString string
		var multiplier *float64
		if len(m) == 2 {
			modTagsString, amount, incDec, multiplier = m[1], "100", "increased", floatp(2)
		} else {
			amount, incDec, modTagsString = m[1], m[2], m[3]
		}
		if incDec == "increased" || incDec == "reduced" {
			q, _ := luaToNumber(amount)
			if incDec == "reduced" {
				q = -q
			}
			if modTagsString == "explicit physical and chaos damage" {
				i.modMagnitudeMods = append(i.modMagnitudeMods, modMagnitudeMod{tags: []string{"damage"}, anyTags: []string{"physical", "chaos"}, quality: q, modType: "explicit", sourceLine: ml})
			} else {
				var tags []string
				modType := ""
				for _, word := range strings.Fields(modTagsString + " ") {
					word = strings.ToLower(word)
					if mapped, ok := modMagnitudeTagMap[word]; ok {
						word = mapped
					}
					if word == "implicit" || word == "explicit" || word == "enchant" {
						modType = word
					} else {
						tags = append(tags, word)
					}
				}
				i.modMagnitudeMods = append(i.modMagnitudeMods, modMagnitudeMod{tags: tags, quality: q, multiplier: multiplier, modType: modType, sourceLine: ml})
			}
			break
		}
	}

	//pob2:block:src/Classes/Item.lua:1323-1374
	var target *[]*ModLine
	switch {
	case ml.Rune:
		target = &i.RuneModLines
	case ml.Enchant:
		target = &i.EnchantModLines
	case strings.Contains(line, "Requires Class"):
		target = &i.ClassRequirementModLines
	case ml.Implicit || len(i.RuneModLines)+*skippedRuneLines+len(i.EnchantModLines)+len(i.ImplicitModLines) < *implicitLines:
		target = &i.ImplicitModLines
	default:
		target = &i.ExplicitModLines
	}
	ml.Line = line
	if i.CheckModLineVariant(ml) {
		if ml.SocketedAugmentTypeOverride != "" {
			i.SocketedAugmentTypeOverride = ml.SocketedAugmentTypeOverride
		} else if ml.SocketedSoulCoreType != "" {
			i.SocketedSoulCoreTypes[ml.SocketedSoulCoreType] = true
		}
	}
	if modList != nil {
		ml.ModList = modList
		ml.Extra = extra
		ml.ValueScalar = floatp(catalystScalar)
		if ml.Range == nil {
			ml.Range = floatp(defaultItemAffixQuality)
		}
		*target = append(*target, ml)
		if mode == "GAME" {
			switch *gameModeStage {
			case "FINDIMPLICIT":
				*gameModeStage = "IMPLICIT"
			case "FINDEXPLICIT":
				*foundExplicit = true
				*gameModeStage = "EXPLICIT"
			case "EXPLICIT":
				*foundExplicit = true
			}
		} else {
			*foundExplicit = true
		}
	} else if mode == "GAME" {
		_, isBase := poe2.BaseByName(line)
		baseTypeLine := i.Base != nil && (line == i.Base.Type || (i.Base.SubType != "" && line == i.Base.SubType+" "+i.Base.Type))
		if *gameModeStage == "IMPLICIT" || *gameModeStage == "EXPLICIT" || (*gameModeStage == "FINDIMPLICIT" && !isBase && i.Name != line && !strings.Contains(line, "Two-Toned") && !baseTypeLine) {
			ml.ModList = []mod.Mod{}
			ml.Extra = line
			*target = append(*target, ml)
		} else if *gameModeStage == "FINDEXPLICIT" {
			*gameModeStage = "DONE"
		}
	} else if *foundExplicit || *gameModeStage == "EXPLICIT" {
		ml.ModList = []mod.Mod{}
		ml.Extra = line
		*target = append(*target, ml)
	}
}

var tagWordRe = regexp.MustCompile(`[A-Za-z_]+`)

// stripEnumRanges is gsub("(%s*)(%b())", ...): a balanced paren group holding a dash and no digit
// is dropped with the spaces before it.
//
//pob2:block:src/Classes/Item.lua:1206-1211
func stripEnumRanges(line string) string {
	var out strings.Builder
	for i := 0; i < len(line); {
		j := i
		for j < len(line) && isLuaSpace(line[j]) {
			j++
		}
		if j < len(line) && line[j] == '(' {
			depth, end := 0, -1
			for k := j; k < len(line); k++ {
				if line[k] == '(' {
					depth++
				} else if line[k] == ')' {
					depth--
					if depth == 0 {
						end = k
						break
					}
				}
			}
			if end >= 0 {
				rng := line[j : end+1]
				if !(strings.Contains(rng, "-") && !strings.ContainsAny(rng, "0123456789")) {
					out.WriteString(line[i : end+1])
				}
				i = end + 1
				continue
			}
		}
		out.WriteByte(line[i])
		i++
	}
	return out.String()
}

func isLuaSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

// luaFind is s:match(pattern) for a pattern built from item text.
func luaFind(s, pattern string) bool {
	re, ok := luaPattern(pattern)
	return ok && re.MatchString(s)
}

// luaPatternQuoteNumber keeps a number's "." as Lua's any-character class, as the Lua's pattern
// built from it does.
func luaPatternQuoteNumber(value string) string {
	return strings.ReplaceAll(value, "-", "%-")
}

//pob2:src/Classes/Item.lua:65-69
func normaliseModLine(line string) string {
	line = normaliseNumRe.ReplaceAllString(line, "#")
	line = normaliseRangeRe.ReplaceAllString(line, "#")
	return strings.ReplaceAll(strings.ToLower(line), "\n", " ")
}

//pob2:src/Classes/Item.lua:73-88
func sortCraftedModLines(modLines []*ModLine) {
	group := func(ml *ModLine) int {
		switch {
		case ml.Crafted || ml.Custom:
			return 3
		case ml.Fractured:
			return 1
		}
		return 2
	}
	order := func(ml *ModLine) float64 {
		if ml.Order == nil {
			return math.Inf(1)
		}
		return *ml.Order
	}
	sort.SliceStable(modLines, func(a, b int) bool {
		ag, bg := group(modLines[a]), group(modLines[b])
		if ag != bg {
			return ag < bg
		}
		if ag < 3 && order(modLines[a]) != order(modLines[b]) {
			return order(modLines[a]) < order(modLines[b])
		}
		return false
	})
}

//pob2:block:src/Classes/Item.lua:1649-1720
func (i *Item) applyModMagnitude() {
	for _, mm := range i.modMagnitudeMods {
		if i.UsesVersionedOrGroupedVariants() && !i.CheckModLineVariant(mm.sourceLine) {
			continue
		}
		var lists [][]*ModLine
		switch mm.modType {
		case "implicit":
			lists = [][]*ModLine{i.ImplicitModLines}
		case "explicit":
			lists = [][]*ModLine{i.ExplicitModLines}
		case "enchant":
			lists = [][]*ModLine{i.EnchantModLines}
		case "":
			lists = [][]*ModLine{i.ImplicitModLines, i.ExplicitModLines, i.EnchantModLines}
		}
		for _, mods := range lists {
			for _, ml := range mods {
				if i.GetModLineVariantCount(ml) == 0 || ml.Unscalable {
					continue
				}
				grantsSkill := false
				for _, pm := range ml.ModList {
					if pm.Name() == "ExtraSkill" {
						grantsSkill = true
						break
					}
				}
				if ml.Extra != "" && !grantsSkill {
					grantsSkill = strings.HasPrefix(strings.ToLower(ml.Line), "grants skill:")
				}
				tagLookup := map[string]bool{}
				for _, t := range ml.ModTags {
					tagLookup[t] = true
				}
				if ml.Desecrated {
					tagLookup["desecrated"] = true
				}
				if ml.Prefix {
					tagLookup["prefix"] = true
				}
				if ml.Suffix {
					tagLookup["suffix"] = true
				}
				match := true
				for _, t := range mm.tags {
					if !tagLookup[t] {
						match = false
					}
				}
				if mm.anyTags != nil && !(tagLookup[mm.anyTags[0]] || tagLookup[mm.anyTags[1]]) {
					match = false
				}
				if match && !grantsSkill {
					vs := 1.0
					if ml.ValueScalar != nil {
						vs = *ml.ValueScalar
					}
					if mm.multiplier != nil {
						ml.ValueScalar = floatp(vs * *mm.multiplier)
					} else {
						ml.ValueScalar = floatp(vs + mm.quality/100)
					}
				}
				if ml.ValueScalar != nil && *ml.ValueScalar != 1 {
					r := 1.0
					if ml.Range != nil {
						r = *ml.Range
					}
					one := 1.0
					rangedLine := applyRange(ml.Line, r, nil, ml.ValueScalar, &one, i.modSource)
					if modList, extra := i.modSource(rangedLine); modList != nil {
						ml.DisplayValueScalar = floatp(1)
						ml.ModList = modList
						ml.Extra = extra
					}
				}
			}
		}
	}
}

//pob2:src/Classes/Item.lua:1828-1836
func (i *Item) NormaliseQuality() {
	if i.Base != nil && i.Base.Quality != 0 {
		if i.Quality == nil {
			i.Quality = floatp(0)
		} else if i.UniqueID == "" && !i.Corrupted && !i.Mirrored && i.Base.Type != "Charm" && *i.Quality < i.Base.Quality {
			i.Quality = floatp(defaultItemQuality)
		}
	}
}

//pob2:src/Classes/Item.lua:2323-2358
func (i *Item) CheckModLineVariant(ml *ModLine) bool {
	if i.UsesVersionedOrGroupedVariants() {
		if ml.VersionList != nil && (i.SelectedVersion == nil || !ml.VersionList[*i.SelectedVersion]) {
			return false
		}
		if ml.BaseVariantList != nil && (i.SelectedBase == nil || !ml.BaseVariantList[*i.SelectedBase]) {
			return false
		}
		if ml.VariantGroupList != nil {
			if ml.VariantList == nil {
				return false
			}
			for groupID := range ml.VariantGroupList {
				if sel, ok := i.VariantGroupSelections[groupID]; ok && ml.VariantList[sel] {
					return true
				}
			}
			return false
		}
		if i.HasIndependentVariants() && ml.VariantList != nil {
			return i.Variant != nil && ml.VariantList[*i.Variant]
		}
		return ml.VariantList == nil
	}
	if i.BaseList != nil && ml.BaseVariantList != nil && !(i.SelectedBase != nil && ml.BaseVariantList[*i.SelectedBase]) {
		return false
	}
	in := func(p *int) bool { return p != nil && ml.VariantList[*p] }
	return ml.VariantList == nil || in(i.Variant) ||
		(i.HasAltVariant && in(i.VariantAlt)) ||
		(i.HasAltVariant2 && in(i.VariantAlt2)) ||
		(i.HasAltVariant3 && in(i.VariantAlt3)) ||
		(i.HasAltVariant4 && in(i.VariantAlt4)) ||
		(i.HasAltVariant5 && in(i.VariantAlt5))
}

//pob2:src/Classes/Item.lua:2360-2376
func (i *Item) GetModLineVariantCount(ml *ModLine) int {
	if i.UsesVersionedOrGroupedVariants() || !i.AllowDuplicateVariants || ml.VariantList == nil {
		if i.CheckModLineVariant(ml) {
			return 1
		}
		return 0
	}
	count := 0
	if i.Variant != nil && ml.VariantList[*i.Variant] {
		count++
	}
	for _, alt := range []struct {
		has bool
		v   *int
	}{{i.HasAltVariant, i.VariantAlt}, {i.HasAltVariant2, i.VariantAlt2}, {i.HasAltVariant3, i.VariantAlt3}, {i.HasAltVariant4, i.VariantAlt4}, {i.HasAltVariant5, i.VariantAlt5}} {
		if alt.has && alt.v != nil && ml.VariantList[*alt.v] {
			count++
		}
	}
	return count
}

//pob2:src/Classes/Item.lua:2378-2392
func (i *Item) GetSocketedAugmentTypes() (string, string) {
	subType := strings.ToLower(i.Base.SubType)
	itemType := strings.ToLower(i.Base.Type)
	baseType := ""
	switch {
	case i.Base.Weapon != nil:
		baseType = "weapon"
	case i.Base.Armour != nil:
		baseType = "armour"
	case hasTag(i.Base, "wand") || hasTag(i.Base, "staff") || hasTag(i.Base, "sceptre"):
		baseType = "caster"
	}
	specificType := itemType
	switch {
	case subType == "warstaff":
		specificType = "quarterstaff"
	case itemType == "shield" && subType == "evasion":
		specificType = "buckler"
	}
	if i.SocketedAugmentTypeOverride != "" {
		return "armour", i.SocketedAugmentTypeOverride
	}
	return baseType, specificType
}

//pob2:src/Classes/Item.lua:2395-2411
func (i *Item) GetPrimarySlot() string {
	switch {
	case i.Base.Weapon != nil || i.Base.Type == "Wand" || i.Base.Type == "Sceptre" || i.Base.Type == "Staff":
		return "Weapon 1"
	case i.Type == "Quiver" || i.Type == "Shield":
		return "Weapon 2"
	case i.Type == "Ring":
		return "Ring 1"
	case i.Type == "Flask":
		return "Flask 1"
	case i.Base.SubType == "Transcendent Leg":
		return "Leg 1"
	case i.Base.SubType == "Transcendent Arm":
		return "Arm 1"
	}
	return i.Type
}
