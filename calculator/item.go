package calculator

import (
	"maps"
	"math"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

var defaultItemAffixQuality = func() *float64 { v := 0.5; return &v }()

const luaSpace = " \t\n\v\f\r"

type ItemRarity string

const (
	RarityNormal = ItemRarity("NORMAL")
	RarityMagic  = ItemRarity("MAGIC")
	RarityRare   = ItemRarity("RARE")
	RarityUnique = ItemRarity("UNIQUE")
	RarityRelic  = ItemRarity("RELIC")
)

var validRarities = map[string]bool{
	"NORMAL": true, "MAGIC": true, "RARE": true, "UNIQUE": true, "RELIC": true,
}

type ItemSocket struct {
	Color string
	Group int
}

type ModLine struct {
	Line        string
	Extra       string
	Range       *float64
	ModList     []mod.Mod
	ModTags     []string
	VariantList map[int]bool
	Crafted     bool
	Custom      bool
	Fractured   bool
	Mutated     bool
	Vestigial   bool
	Scourge     bool
	Implicit    bool
	ValueScalar float64

	CorruptedRange *float64

	Prefix   bool
	Suffix   bool
	Unveiled bool

	Disabled bool

	Order *float64

	ModID            string
	NewModID         string
	ModGroup         string
	Crucible         bool
	VersionList      map[int]bool
	VariantGroupList map[int]bool
}

//pob:block:src/Classes/Item.lua:1244-1283
type ModMagnitudeMod struct {
	Tags       []string
	AnyTags    []string
	Quality    float64
	Multiplier *float64
	ModType    string
}

type ItemInfluence struct {
	Shaper      bool
	Elder       bool
	Adjudicator bool
	Basilisk    bool
	Crusader    bool
	Eyrie       bool
	Cleansing   bool
	Tangle      bool
}

var influenceDisplay = []struct {
	display string
	set     func(*ItemInfluence)
}{
	{"Shaper", func(i *ItemInfluence) { i.Shaper = true }},
	{"Elder", func(i *ItemInfluence) { i.Elder = true }},
	{"Warlord", func(i *ItemInfluence) { i.Adjudicator = true }},
	{"Hunter", func(i *ItemInfluence) { i.Basilisk = true }},
	{"Crusader", func(i *ItemInfluence) { i.Crusader = true }},
	{"Redeemer", func(i *ItemInfluence) { i.Eyrie = true }},
	{"Searing Exarch", func(i *ItemInfluence) { i.Cleansing = true }},
	{"Eater of Worlds", func(i *ItemInfluence) { i.Tangle = true }},
}

//pob:src/Classes/Item.lua:14-29
var catalystList = []string{
	"Abrasive", "Accelerating", "Dextral", "Fertile", "Imbued", "Intrinsic",
	"Noxious", "Prismatic", "Sinistral", "Tempering", "Turbulent", "Unstable",
}

var catalystDescriptorList = []string{
	"Attack", "Speed", "Suffix", "Life and Mana", "Caster", "Attribute",
	"Physical and Chaos", "Resistance", "Prefix", "Defence", "Elemental", "Critical",
}

var catalystQualitySpecRe = regexp.MustCompile(`Quality \(([a-zA-Z\s]+) Modifiers\)`)

var rangedModLineRe = regexp.MustCompile(`\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)

var catalystTags = [][]string{
	{"attack"},
	{"speed"},
	{"suffix"},
	{"life", "mana", "resource"},
	{"caster"},
	{"jewellery_attribute", "attribute"},
	{"physical_damage", "chaos_damage"},
	{"jewellery_resistance", "resistance"},
	{"prefix"},
	{"jewellery_defense", "defences", "armour", "evasion", "energyshield"},
	{"jewellery_elemental", "elemental_damage"},
	{"critical"},
}

type Item struct {
	Raw      string
	Name     string
	Title    string
	BaseName string
	Rarity   ItemRarity
	Type     string
	Base     *ItemBase

	Quality          *int
	ItemLevel        *int
	UniqueID         string
	League           string
	Note             string
	Source           string
	ClassRestriction string
	Unreleased       bool
	UpgradePaths     []string

	Corrupted      bool
	Mirrored       bool
	RareLikeUnique bool
	FoilType       string
	Split          bool
	Fractured      bool
	Foulborn       bool
	Synthesised    bool
	Crafted        bool
	Scourge        bool
	CheckSection   bool

	Influence ItemInfluence

	Sockets      []ItemSocket
	Requirements map[string]float64

	ImplicitModLines         []ModLine
	ExplicitModLines         []ModLine
	CrucibleModLines         []ModLine
	EnchantModLines          []ModLine
	ScourgeModLines          []ModLine
	ClassRequirementModLines []ModLine

	AdvancedCopy bool

	ModMagnitudeMods []ModMagnitudeMod

	VariantList []string
	Variant     *int

	VersionList            []string
	SelectedVersion        *int
	VariantGroups          map[int]map[int]map[int]bool
	VariantGroupSelections map[int]int
	UsesVariantGroups      bool
	AllowDuplicateVariants bool

	Catalyst        *int
	CatalystQuality *int

	HasAltVariant  bool
	HasAltVariant2 bool
	HasAltVariant3 bool
	HasAltVariant4 bool
	HasAltVariant5 bool
	VariantAlt     *int
	VariantAlt2    *int
	VariantAlt3    *int
	VariantAlt4    *int
	VariantAlt5    *int

	NamePrefix string
	NameSuffix string

	TalismanTier         *int
	MemoryStrands        *int
	Intangibility        *int
	Limit                *int
	JewelRadiusLabel     string
	CanBeAnointed        bool
	CanHaveTwoEnchants   bool
	CanHaveThreeEnchants bool
	CanHaveFourEnchants  bool

	Mods         []mod.Mod
	BaseModList  []mod.Mod
	slotModList2 []mod.Mod
	slotModList3 []mod.Mod

	BuffModList []mod.Mod

	GrantedSkills []ItemGrantedSkill

	AbyssalSocketCount          float64
	SocketedJewelEffectModifier float64

	SocketColourAlwaysMatches bool

	WeaponData map[int]map[string]float64
	ArmourData map[string]float64
	FlaskData  map[string]float64

	TinctureData map[string]float64

	rawLines []string
}

type ItemGrantedSkill struct {
	SkillID       string
	Level         int
	NoSupports    bool
	Source        string
	Triggered     bool
	triggerChance float64
}

var dmgTypeList = []string{"Physical", "Lightning", "Cold", "Fire", "Chaos"}

var (
	rarityRe         = regexp.MustCompile(`^Rarity: ([a-zA-Z]+)`)
	specHexRe        = regexp.MustCompile(`^([a-zA-Z ()]+): ([0-9a-fA-F]+)$`)
	specNumRe        = regexp.MustCompile(`^([a-zA-Z ()]+): \+?([0-9+\-.,]+)`)
	specAnyRe        = regexp.MustCompile(`^([a-zA-Z ()]+): (.+)$`)
	reqClassRe       = regexp.MustCompile(`^(Requires Class) (.+)$`)
	classColonRe     = regexp.MustCompile(`^Class:: (.+)$`)
	requiresRe       = regexp.MustCompile(`^(Requires) (.+)$`)
	modGroupRe       = regexp.MustCompile(`\{modGroup:([^}]+)\}`)
	corruptedRangeRe = regexp.MustCompile(`\{corruptedRange:([^}]+)\}`)

	variantSpecRe       = regexp.MustCompile(`\{variant:([\d,]+)\}`)
	variantDigitRe      = regexp.MustCompile(`\d+`)
	variantNameRe       = regexp.MustCompile(`\{([\w_]+)\}(.+)`)
	rangeSpecRe         = regexp.MustCompile(`\{range:([\d.]+)\}`)
	reminderTextRe      = regexp.MustCompile(`^\([A-Za-z]|^\(\d+%? of `)
	flaskLastsRe        = regexp.MustCompile(`^Lasts .+ Seconds$`)
	flaskConsumesRe     = regexp.MustCompile(`^Consumes \d+ of \d+ Charges on use$`)
	flaskChargesRe      = regexp.MustCompile(`^Currently has \d+ Charges$`)
	advancedCopyRangeRe = regexp.MustCompile(`\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)
	tagsRe              = regexp.MustCompile(`\{tags:([^}]*)\}`)
	levelRe             = regexp.MustCompile(`Level (\d+)`)
	parenSuffixRe       = regexp.MustCompile(` \(.+\)`)
	foilTypeRe          = regexp.MustCompile(`\(([^)]*)\)`)
	augmentedRe         = regexp.MustCompile(` \(augmented\)`)
	socketColorRe       = regexp.MustCompile(`[RGBWA]`)
	alphaPrefixRe       = regexp.MustCompile(`^[a-zA-Z]+`)

	pairRangeDetectRe = regexp.MustCompile(`\(\d+-\d+ to \d+-\d+\)`)
	toRangeDetectRe   = regexp.MustCompile(`\(-?[\d.]+ to -?[\d.]+\)`)
	dashRangeDetectRe = regexp.MustCompile(`\(-?[\d.]+-[\d.]+\)`)
)

func stripBalancedBraces(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			out.WriteByte(s[i])
			continue
		}

		depth, end := 0, -1
		for j := i; j < len(s); j++ {
			switch s[j] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = j
				}
			}
			if end >= 0 {
				break
			}
		}

		if end < 0 {
			out.WriteByte(s[i])
			continue
		}
		i = end
	}

	return out.String()
}

//pob:block:src/Classes/Item.lua:1028
func hasRange(line string) bool {
	return pairRangeDetectRe.MatchString(line) ||
		toRangeDetectRe.MatchString(line) ||
		dashRangeDetectRe.MatchString(line)
}

func luaToNumber(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func intPtr(v int) *int { return &v }

func modExtra(e *ModCacheEntry) string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Extra) == "" {
		return ""
	}
	return e.Extra
}

//pob:src/Classes/Item.lua:28-49
func getCatalystScalar(catalystID *int, tags []string, quality *int, prefix, suffix bool) float64 {
	if catalystID == nil || *catalystID < 1 || *catalystID > len(catalystTags) || len(tags) == 0 {
		return 1
	}

	q := 20
	if quality != nil {
		q = *quality
	}

	lookup := make(map[string]bool, len(tags)+2)
	for _, t := range tags {
		lookup[t] = true
	}

	if prefix {
		lookup["prefix"] = true
	}
	if suffix {
		lookup["suffix"] = true
	}

	for _, ct := range catalystTags[*catalystID-1] {
		if lookup[ct] {
			return float64(100+q) / 100
		}
	}

	return 1
}

var xmlEntityDecoder = strings.NewReplacer(
	"&lt;", "<",
	"&gt;", ">",
	"&apos;", "'",
	"&quot;", `"`,
	"&amp;", "&",
)

//pob:src/Classes/Item.lua:53-57
func ParseItem(raw string) *Item {
	i := &Item{}
	ranges := parseModRangeElements(raw)
	i.ParseRaw(data.SanitiseItemText(xmlEntityDecoder.Replace(raw)))
	i.applyModRanges(ranges)
	//pob:src/Classes/ItemsTab.lua:1258-1259
	if i.Base != nil {
		i.BuildModList()
	}
	return i
}

type modRange struct {
	ID    int
	Range float64
}

var (
	modRangeElemRE = regexp.MustCompile(`<ModRange\b[^>]*>`)
	modRangeIDRE   = regexp.MustCompile(`\bid="(\d+)"`)
	modRangeValRE  = regexp.MustCompile(`\brange="([^"]*)"`)
)

//pob:block:src/Classes/ItemsTab.lua:1242-1244
func parseModRangeElements(raw string) []modRange {
	if !strings.Contains(raw, "<ModRange") {
		return nil
	}
	var out []modRange
	for _, elem := range modRangeElemRE.FindAllString(raw, -1) {
		id := 0
		if m := modRangeIDRE.FindStringSubmatch(elem); m != nil {
			v, err := strconv.Atoi(m[1])
			if err != nil {
				continue
			}
			id = v
		}
		rv := 1.0
		if m := modRangeValRE.FindStringSubmatch(elem); m != nil {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				rv = v
			}
		}
		out = append(out, modRange{ID: id, Range: rv})
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:1239-1256
func (i *Item) applyModRanges(ranges []modRange) {
	if len(ranges) == 0 {
		return
	}
	lists := [...][]ModLine{
		make([]ModLine, i.buffModLineCount()),
		i.EnchantModLines,
		i.ScourgeModLines,
		i.ImplicitModLines,
		i.ExplicitModLines,
		i.CrucibleModLines,
	}
	for _, mr := range ranges {
		id := mr.ID
		for _, list := range lists {
			if id <= len(list) {
				if id >= 1 {
					v := mr.Range
					list[id-1].Range = &v
				}
				break
			}
			id -= len(list)
		}
	}
}

//pob:block:src/Classes/Item.lua:1034-1049
func (i *Item) buffModLineCount() int {
	if i.Base == nil || i.Base.Flask == nil {
		return 0
	}
	return len(i.Base.Flask.Buff)
}

func (i *Item) ParseRaw(raw string) {
	i.parseRaw(raw, "", false)
}

//pob:src/Classes/Item.lua:67-634
//nolint:gocyclo,maintidx // Deliberately mirrors the structure of ItemClass:ParseRaw.
func (i *Item) parseRaw(raw string, rarity ItemRarity, highQuality bool) {
	i.Raw = raw
	i.Name = "?"
	i.Rarity = RarityUnique
	if rarity != "" {
		i.Rarity = rarity
	}
	i.Quality = nil

	i.rawLines = nil
	for _, line := range strings.Split(strings.ReplaceAll(i.Raw+"\r\n", "\r\n", "\n"), "\n") {
		line = strings.Trim(line, luaSpace)
		if line != "" {
			i.rawLines = append(i.rawLines, line)
		}
	}

	mode := "WIKI"
	if rarity != "" {
		mode = "GAME"
	}
	l := 0
	if l < len(i.rawLines) {
		if strings.HasPrefix(i.rawLines[l], "Item Class:") {
			l++
		}
		if l < len(i.rawLines) {
			if m := rarityRe.FindStringSubmatch(i.rawLines[l]); m != nil {
				mode = "GAME"
				if r := strings.ToUpper(m[1]); validRarities[r] {
					i.Rarity = ItemRarity(r)
				}
				//pob:src/Classes/Item.lua:434-444
				if i.Rarity == RarityUnique || i.Rarity == RarityRelic {
					for _, line := range i.rawLines {
						if strings.Contains(line, "Foil Unique") {
							i.Rarity = RarityRelic
							i.FoilType = foilTypeOf(line)
							break
						}
					}
				}
				l++
			}
		}
	}

	if l < len(i.rawLines) {
		i.Name = i.rawLines[l]
		if !(i.Rarity == RarityNormal || i.Rarity == RarityMagic) {
			l++
		}
	}

	i.CheckSection = false
	i.Sockets = nil
	i.ClassRequirementModLines = nil
	i.EnchantModLines = nil
	i.ScourgeModLines = nil
	i.ImplicitModLines = nil
	i.ExplicitModLines = nil
	i.CrucibleModLines = nil
	i.ModMagnitudeMods = nil
	i.AdvancedCopy = false
	i.Requirements = map[string]float64{}

	implicitLines := 0
	var importedLevelReq *int
	gameModeStage := "FINDIMPLICIT"
	foundExplicit, foundImplicit := false, false

	processInfluenceLine := func(line string) bool {
		for _, inf := range influenceDisplay {
			if line == inf.display+" Item" {
				inf.set(&i.Influence)
				return true
			}
		}
		return false
	}

	for l < len(i.rawLines) {
		line := i.rawLines[l]

		switch {
		case line == "--------":
			i.CheckSection = true
		case line == "Corrupted":
			i.Corrupted = true
		case line == "Fractured Item":
			i.Fractured = true
		case line == "Synthesised Item":
			i.Synthesised = true
		//pob:src/Classes/Item.lua:533-535
		case strings.Contains(line, "Foil Unique"):
			i.FoilType = foilTypeOf(line)
		case line == "Mirrored":
			i.Mirrored = true
		//pob:src/Classes/Item.lua:523
		case line == "Split":
			i.Split = true
		case processInfluenceLine(line):
		case line == "Requirements:":
			// nothing to do

		//pob:src/Classes/Item.lua:540-544
		case strings.HasPrefix(line, "(") && reminderTextRe.MatchString(line):
			for l < len(i.rawLines) && !strings.HasSuffix(i.rawLines[l], ")") {
				l++
			}

		//pob:src/Classes/Item.lua:545-549
		case i.Base != nil && i.Base.Flask != nil &&
			(flaskLastsRe.MatchString(line) ||
				flaskConsumesRe.MatchString(line) ||
				flaskChargesRe.MatchString(line)):
			// In-game flask state and base properties aren't modifier lines.

		default:
			if i.CheckSection {
				switch {
				case gameModeStage == "IMPLICIT":
					if foundImplicit {
						gameModeStage = "EXPLICIT"
						foundExplicit = true
					} else {
						gameModeStage = "FINDEXPLICIT"
					}
				case gameModeStage == "EXPLICIT":
					gameModeStage = "DONE"
				case gameModeStage == "FINDIMPLICIT" && i.ItemLevel != nil &&
					!strings.Contains(line, " (implicit)") &&
					!strings.Contains(line, " (enchant)") &&
					!strings.Contains(line, "Talisman Tier"):
					gameModeStage = "EXPLICIT"
					foundExplicit = true
				}
				i.CheckSection = false
			}

			specName, specVal := matchSpec(line)

			if specName != "" {
				skip := i.applySpec(specName, specVal, &implicitLines, &gameModeStage, &foundExplicit, &importedLevelReq)
				if skip {
					l++
				}
			}

			if line == "Prefixes:" {
				foundExplicit = true
				gameModeStage = "EXPLICIT"
			}

			if specName == "" || foundExplicit || foundImplicit {
				l = i.parseModLine(line, l, mode, implicitLines, &gameModeStage, &foundExplicit, &foundImplicit)
			}
		}

		l++
	}

	i.assignUniqueModStatOrder()

	if i.Base != nil {
		if _, ok := i.Requirements["level"]; !ok {
			if importedLevelReq != nil && len(i.Sockets) == 0 {
				i.Requirements["level"] = float64(*importedLevelReq)
			} else if i.Base.Req.Level > 0 {
				i.Requirements["level"] = float64(i.Base.Req.Level)
			}
		}
	}

	//pob:block:src/Classes/Item.lua:1369-1376
	if luaLower(i.Title) == "cane of kulemak" && (i.Rarity == RarityUnique || i.Rarity == RarityRelic) && i.AdvancedCopy {
		for k := range i.ExplicitModLines {
			if !strings.Contains(i.ExplicitModLines[k].Line, "magnitude") {
				i.ExplicitModLines[k].Unveiled = true
			}
		}
	}

	i.applyModMagnitude()

	//pob:src/Classes/Item.lua:1433-1435
	if i.AdvancedCopy && len(i.ExplicitModLines) > 1 {
		sortCraftedModLines(i.ExplicitModLines)
	}

	if i.Base != nil && i.Base.SocketLimit > 0 && len(i.Sockets) == 0 {
		color := i.defaultSocketColor()
		for n := 0; n < i.Base.SocketLimit; n++ {
			i.Sockets = append(i.Sockets, ItemSocket{Color: color, Group: 0})
		}
	}

	//pob:block:src/Classes/Item.lua:1493-1511
	if i.UsesVariantGroups {
		i.normaliseVariantSelections()
	} else if len(i.VariantList) > 0 {
		latest := func(v *int) *int {
			if v == nil {
				return intPtr(len(i.VariantList))
			}
			return intPtr(min(len(i.VariantList), *v))
		}
		i.Variant = latest(i.Variant)
		for _, alt := range []struct {
			has bool
			v   **int
		}{{i.HasAltVariant, &i.VariantAlt}, {i.HasAltVariant2, &i.VariantAlt2}, {i.HasAltVariant3, &i.VariantAlt3}, {i.HasAltVariant4, &i.VariantAlt4}, {i.HasAltVariant5, &i.VariantAlt5}} {
			if alt.has {
				*alt.v = latest(*alt.v)
			}
		}
	}
	i.markMutatedLines()

	//pob:block:src/Classes/Item.lua:1559-1563
	for _, ml := range i.ExplicitModLines {
		if ml.Mutated {
			i.Foulborn = true
		}
	}

	//pob:block:src/Classes/Item.lua:1564-1570
	if i.Title != "" {
		hasFoulbornPrefix := titleFoulbornRe.MatchString(i.Title)
		if i.Foulborn && !hasFoulbornPrefix {
			i.Title = "Foulborn " + i.Title
		} else if !i.Foulborn && hasFoulbornPrefix {
			i.Title = foulbornAnyRe.ReplaceAllString(i.Title, "")
		}
	}

	if i.BaseName != "" && i.Title != "" {
		i.Name = i.Title + ", " + parenSuffixRe.ReplaceAllString(i.BaseName, "")
	}

	if i.Quality == nil {
		i.NormaliseQuality()
		if highQuality {
			i.NormaliseQuality()
		}
	}

	i.BuildModList()
}

var (
	advancedCopyBaseValueRe = regexp.MustCompile(`(-?[0-9]+\.?[0-9]*)\((-?[0-9]+\.?[0-9]*)\)`)
	advancedCopyRollRe      = regexp.MustCompile(`(-?[0-9]+\.?[0-9]*)\((-?[0-9]+\.?[0-9]*--?[0-9]+\.?[0-9]*)\)`)
	advancedCopyMinMaxRe    = regexp.MustCompile(`(-?[0-9]+\.?[0-9]*)-(-?[0-9]+\.?[0-9]*)`)
)

func luaRound(val float64, dec int) float64 {
	f := math.Pow(10, float64(dec))
	return math.Floor(float64(val*f)+0.5) / f
}

func luaValuePattern(value string) string {
	return strings.ReplaceAll(regexp.QuoteMeta(value), `\.`, `.`)
}

func stripEnumRanges(line string) string {
	var b strings.Builder
	for pos := 0; pos < len(line); {
		j := pos
		for j < len(line) && isLuaSpace(line[j]) {
			j++
		}
		end := -1
		if j < len(line) && line[j] == '(' {
			depth := 0
			for k := j; k < len(line); k++ {
				if line[k] == '(' {
					depth++
				} else if line[k] == ')' {
					depth--
					if depth == 0 {
						end = k + 1
						break
					}
				}
			}
		}
		if end < 0 {
			b.WriteByte(line[pos])
			pos++
			continue
		}
		group := line[j:end]
		if !(strings.Contains(group, "-") && !strings.ContainsAny(group, "0123456789")) {
			b.WriteString(line[pos:end])
		}
		pos = end
	}
	return b.String()
}

func isLuaSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

//pob:block:src/Classes/Item.lua:1121-1160
func advancedCopyRollRanges(line string) (string, *float64) {
	bestPrecisionDelta := -1.0
	bestPrecisionRange := -1.0
	var firstRollRange *float64
	hasIndependentRolls := false

	if !strings.Contains(line, "(") {
		return line, nil
	}
	line = stripEnumRanges(line)
	advancedCopyLine := line

	for _, m := range advancedCopyRollRe.FindAllStringSubmatch(line, -1) {
		value, rng := m[1], m[2]
		mm := advancedCopyMinMaxRe.FindStringSubmatch(rng)
		if mm == nil {
			continue
		}
		minS, maxS := mm[1], mm[2]
		minV, _ := strconv.ParseFloat(minS, 64)
		maxV, _ := strconv.ParseFloat(maxS, 64)
		if minV > maxV {
			minS, maxS = maxS, minS
			minV, maxV = maxV, minV
		}
		valueV, _ := strconv.ParseFloat(value, 64)
		delta := maxV - minV
		rollRange := 0.5
		if delta > 0 {
			rollRange = luaRound((valueV-minV)/delta, 6)
		}
		if firstRollRange != nil && *firstRollRange != rollRange {
			hasIndependentRolls = true
		}
		if firstRollRange == nil {
			v := rollRange
			firstRollRange = &v
		}
		if delta > bestPrecisionDelta {
			bestPrecisionRange = rollRange
			bestPrecisionDelta = delta
		}
		re := regexp.MustCompile(luaValuePattern(value) + `\(` + regexp.QuoteMeta(rng) + `\)`)
		if bestPrecisionRange > 1 || bestPrecisionRange < 0 {
			line = re.ReplaceAllLiteralString(line, value)
		} else {
			sign := ""
			if valueV < 0 {
				sign = "+"
			}
			line = re.ReplaceAllLiteralString(line, sign+"("+minS+"-"+maxS+")")
		}
	}
	if hasIndependentRolls {
		return advancedCopyRollRe.ReplaceAllString(advancedCopyLine, "$1"), nil
	}
	if bestPrecisionRange <= 1 && bestPrecisionRange >= 0 {
		v := bestPrecisionRange
		return line, &v
	}
	return line, nil
}

//pob:src/Classes/Item.lua:64-68
var (
	titleFoulbornRe = regexp.MustCompile(`^[Ff]oulborn `)
	foulbornAnyRe   = regexp.MustCompile(`[Ff]oulborn `)
)

//pob:block:src/Classes/Item.lua:998-1005,src/Classes/Item.lua:1513-1558
func (i *Item) markMutatedLines() {
	if i.Rarity != RarityUnique || i.Title == "" || strings.Contains(luaLower(i.Title), "might of the meek") {
		return
	}
	mutatedLines := data.FoulbornMap(titleFoulbornRe.ReplaceAllString(i.Title, ""))
	checkMod := func(modID, newModID string, mutated bool) {
		originalMod, ok := data.FoulbornModLines(modID, mutated)
		if !ok {
			return
		}
		findMatchingLines := func(lines []string) []int {
			var matching []int
			matched := map[int]bool{}
			for _, line := range lines {
				statLine := normaliseModLine(line)
				for k, ml := range i.ExplicitModLines {
					if !matched[k] && normaliseModLine(ml.Line) == statLine && i.checkModLineVariant(ml) {
						matched[k] = true
						matching = append(matching, k)
						break
					}
				}
			}
			if len(lines) != len(matching) {
				return nil
			}
			return matching
		}
		matchingLines := findMatchingLines([]string{strings.Join(originalMod, " ")})
		if matchingLines == nil {
			matchingLines = findMatchingLines(originalMod)
		}
		for _, k := range matchingLines {
			i.ExplicitModLines[k].ModID = modID
			i.ExplicitModLines[k].NewModID = newModID
			if mutated {
				i.ExplicitModLines[k].Mutated = true
			}
		}
	}
	for _, origModID := range slices.Sorted(maps.Keys(mutatedLines)) {
		foulModID := mutatedLines[origModID]
		checkMod(origModID, foulModID, false)
		checkMod(foulModID, origModID, true)
	}
}

//pob:src/Classes/Item.lua:1607-1643
func (i *Item) mutateMod(modID, newModID string, mutated bool) bool {
	newMod, ok := data.FoulbornModOf(newModID, mutated)
	if !ok {
		return false
	}
	insertIdx := -1
	var variantList map[int]bool
	for k := 0; k < len(i.ExplicitModLines); {
		ml := i.ExplicitModLines[k]
		if ml.ModID == modID && i.checkModLineVariant(ml) {
			if insertIdx < 0 {
				insertIdx, variantList = k, ml.VariantList
			}
			i.ExplicitModLines = slices.Delete(i.ExplicitModLines, k, k+1)
			continue
		}
		k++
	}
	if insertIdx < 0 {
		return false
	}
	i.ExplicitModLines = slices.Insert(i.ExplicitModLines, insertIdx, ModLine{
		Line:        strings.Join(newMod.Lines, "\n"),
		ModTags:     newMod.ModTags,
		VariantList: variantList,
		Mutated:     mutated,
		ModGroup:    newModID,
	})
	return true
}

func normaliseModLine(line string) string {
	line = normaliseNumberRe.ReplaceAllString(line, "#")
	line = normaliseRangeRe.ReplaceAllString(line, "#")
	return strings.ReplaceAll(luaLower(line), "\n", " ")
}

var (
	normaliseNumberRe = regexp.MustCompile(`[0-9]+\.?[0-9]*`)
	normaliseRangeRe  = regexp.MustCompile(`\(-?#-#\)`)
)

func luaLower(s string) string {
	b := []byte(s)
	for k, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[k] = c + ('a' - 'A')
		}
	}
	return string(b)
}

//pob:block:src/Classes/Item.lua:1336-1353
func (i *Item) assignUniqueModStatOrder() {
	if !i.AdvancedCopy || (i.Rarity != RarityUnique && i.Rarity != RarityRelic) {
		return
	}
	for k := range i.ExplicitModLines {
		ml := &i.ExplicitModLines[k]
		exactLine := strings.ReplaceAll(luaLower(ml.Line), "\n", " ")
		if e, ok := raw.ItemModStatOrder[exactLine]; ok && e.Exact > 0 {
			v := e.Exact
			ml.Order = &v
			continue
		}
		if e, ok := raw.ItemModStatOrder[normaliseModLine(ml.Line)]; ok && e.Normalised > 0 {
			v := e.Normalised
			ml.Order = &v
			continue
		}
		ml.Order = nil
	}
}

//pob:src/Classes/Item.lua:72-87
func sortCraftedModLines(modLines []ModLine) {
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
		ga, gb := group(&modLines[a]), group(&modLines[b])
		if ga != gb {
			return ga < gb
		}
		if ga < 3 {
			return order(&modLines[a]) < order(&modLines[b])
		}
		return false
	})
}

//pob:src/Classes/Item.lua:971-979
func (i *Item) checkModLineVariant(ml ModLine) bool {
	if i.UsesVariantGroups {
		return i.checkModLineVariantGroups(ml)
	}
	if ml.VariantList == nil {
		return true
	}
	if i.Variant != nil && ml.VariantList[*i.Variant] {
		return true
	}
	if i.HasAltVariant && i.VariantAlt != nil && ml.VariantList[*i.VariantAlt] {
		return true
	}
	if i.HasAltVariant2 && i.VariantAlt2 != nil && ml.VariantList[*i.VariantAlt2] {
		return true
	}
	if i.HasAltVariant3 && i.VariantAlt3 != nil && ml.VariantList[*i.VariantAlt3] {
		return true
	}
	if i.HasAltVariant4 && i.VariantAlt4 != nil && ml.VariantList[*i.VariantAlt4] {
		return true
	}
	if i.HasAltVariant5 && i.VariantAlt5 != nil && ml.VariantList[*i.VariantAlt5] {
		return true
	}
	return false
}

//pob:src/Classes/Item.lua:1219-1375
func (i *Item) BuildModList() {
	i.Mods = nil
	i.WeaponData = nil
	i.FlaskData = nil
	i.TinctureData = nil
	i.BuffModList = nil
	i.GrantedSkills = nil
	i.AbyssalSocketCount = 0
	i.SocketedJewelEffectModifier = 1
	if i.Base == nil {
		return
	}

	baseList := make([]mod.Mod, 0)
	source := "Item:-1:" + i.Name

	if i.Base.Flask != nil {
		for _, line := range i.Base.Flask.Buff {
			entry := ParseMod(line, false)
			if strings.TrimSpace(entry.Extra) != "" {
				continue
			}
			for _, m := range entry.ModList {
				i.BuffModList = append(i.BuffModList, m.Clone().Source(mod.Source(source)))
			}
		}
	}

	processModLine := func(ml ModLine) {
		if ml.Disabled {
			return
		}
		if !i.checkModLineVariant(ml) || ml.Extra != "" {
			return
		}

		mods := ml.ModList
		if ml.Range != nil {
			stripped := strings.ReplaceAll(ml.Line, "\n", " ")
			//pob:block:src/Classes/Item.lua:2471
			if rangedModLineRe.MatchString(ml.Line) {
				scalar := ml.ValueScalar
				if scalar == 0 {
					scalar = 1
				}
				rolled := data.ApplyRange(stripped, *ml.Range, &scalar, ml.CorruptedRange)
				entry := ParseMod(rolled, false)
				if entry.ModList != nil && strings.TrimSpace(entry.Extra) == "" {
					mods = entry.ModList
				}
			}
		}

		for _, m := range mods {
			baseList = append(baseList, mod.SetSource(m.Clone(), mod.Source(source)))
		}
	}

	for _, ml := range i.EnchantModLines {
		processModLine(ml)
	}
	for _, ml := range i.ScourgeModLines {
		processModLine(ml)
	}
	for _, ml := range i.ClassRequirementModLines {
		processModLine(ml)
	}
	for _, ml := range i.ImplicitModLines {
		processModLine(ml)
	}
	for _, ml := range i.ExplicitModLines {
		processModLine(ml)
	}
	for _, ml := range i.CrucibleModLines {
		processModLine(ml)
	}

	//pob:block:src/Classes/Item.lua:2588-2635
	socketCount := calcLocal(&baseList, "SocketCount", mod.TypeBase, 0)
	i.AbyssalSocketCount = calcLocal(&baseList, "AbyssalSocketCount", mod.TypeBase, 0)
	baseSocketLimit := 0.0
	if i.Base != nil {
		baseSocketLimit = float64(i.Base.SocketLimit)
	}
	selectableSocketCount := math.Max(baseSocketLimit, float64(len(i.Sockets))) - i.AbyssalSocketCount
	switch {
	case calcLocalFlag(&baseList, "NoSockets", 0):
		i.Sockets = nil
		i.AbyssalSocketCount = 0
	case socketCount > 0:
		want := int(socketCount)
		newSockets := make([]ItemSocket, 0, want)
		group := 0
		for n := range want {
			if n < len(i.Sockets) {
				group = i.Sockets[n].Group
				newSockets = append(newSockets, i.Sockets[n])
				continue
			}
			newSockets = append(newSockets, ItemSocket{Color: i.defaultSocketColor(), Group: group})
		}
		i.Sockets = newSockets
	case i.AbyssalSocketCount > 0:
		newSockets := make([]ItemSocket, 0, len(i.Sockets))
		group := 0
		for _, socket := range i.Sockets {
			if socket.Color == "A" {
				continue
			}
			if float64(len(newSockets)) >= selectableSocketCount {
				break
			}
			newSockets = append(newSockets, socket)
			group = socket.Group
		}
		for range int(i.AbyssalSocketCount) {
			group++
			newSockets = append(newSockets, ItemSocket{Color: "A", Group: group})
		}
		i.Sockets = newSockets
	}

	//pob:src/Classes/Item.lua:2636-2638
	i.SocketColourAlwaysMatches = len(i.Sockets) > 0 && calcLocalFlag(&baseList, "SocketAlwaysMatches", 0)

	//pob:src/Classes/Item.lua:2639
	i.SocketedJewelEffectModifier = 1 + calcLocal(&baseList, "SocketedJewelEffect", mod.TypeIncrease, 0)/100

	//pob:block:src/Classes/Item.lua:2561-2565
	if i.Name == "Tabula Rasa, Simple Robe" || i.Name == "Skin of the Loyal, Simple Robe" ||
		i.Name == "Skin of the Lords, Simple Robe" || i.Name == "The Apostate, Cabalist Regalia" {
		baseList = append(baseList, mod.NewList("ArmourData", mod.ArmourData{Key: "EnergyShield", Value: 0}))
		i.Requirements["int"] = 0
	}

	if i.Base.Weapon != nil || i.Type == "Ring" {
		i.WeaponData = map[int]map[string]float64{}
		i.Mods = i.buildModListForSlotNum(baseList, 1)
		i.slotModList2 = i.buildModListForSlotNum(baseList, 2)
		//pob:block:src/Classes/Item.lua:2646-2648
		if i.Type == "Ring" {
			i.slotModList3 = i.buildModListForSlotNum(baseList, 3)
		}
	} else {
		i.Mods = i.buildModListForSlotNum(baseList, 0)
	}

	if calcLocalFlag(&baseList, "NoAttributeRequirements", 0) {
		i.Requirements["strMod"] = 0
		i.Requirements["dexMod"] = 0
		i.Requirements["intMod"] = 0
	} else {
		i.Requirements["strMod"] = math.Floor((i.Requirements["str"] + calcLocal(&baseList, "StrRequirement", mod.TypeBase, 0)) * (1 + calcLocal(&baseList, "StrRequirement", mod.TypeIncrease, 0)/100))
		i.Requirements["dexMod"] = math.Floor((i.Requirements["dex"] + calcLocal(&baseList, "DexRequirement", mod.TypeBase, 0)) * (1 + calcLocal(&baseList, "DexRequirement", mod.TypeIncrease, 0)/100))
		i.Requirements["intMod"] = math.Floor((i.Requirements["int"] + calcLocal(&baseList, "IntRequirement", mod.TypeBase, 0)) * (1 + calcLocal(&baseList, "IntRequirement", mod.TypeIncrease, 0)/100))
	}

	//pob:block:src/Classes/Item.lua:1359-1370
	for _, value := range utils.CastSlice[mod.ExtraSkill](asModList(baseList).List(nil, "ExtraSkill")) {
		if value.SkillID == "" {
			continue
		}
		i.GrantedSkills = append(i.GrantedSkills, ItemGrantedSkill{
			SkillID:    value.SkillID,
			Level:      value.Level,
			NoSupports: value.NoSupports,
			Source:     source,
			Triggered:  value.Triggered,

			triggerChance: value.TriggerChance,
		})
	}
	i.BaseModList = baseList
}

func (i *Item) modListForSlot(slotNum int) []mod.Mod {
	switch {
	case slotNum == 2 && i.slotModList2 != nil:
		return i.slotModList2
	case slotNum == 3 && i.slotModList3 != nil:
		return i.slotModList3
	}
	return i.Mods
}

//pob:src/Classes/Item.lua:2148-2175
func calcLocal(modList *[]mod.Mod, name string, modType mod.Type, flags mod.MFlag) float64 {
	var result float64
	if modType == mod.TypeMore {
		result = 1
	}

	kept := (*modList)[:0]
	for _, m := range *modList {
		if isLocalMod(m, name, modType, flags) {
			switch modType {
			case mod.TypeMore:
				result *= (100 + m.Value().Float()) / 100
			default:
				result += m.Value().Float()
			}
		} else {
			kept = append(kept, m)
		}
	}
	*modList = kept
	return result
}

//pob:src/Classes/Item.lua:2148-2175
func calcLocalFlag(modList *[]mod.Mod, name string, flags mod.MFlag) bool {
	result := false
	kept := (*modList)[:0]
	for _, m := range *modList {
		if isLocalMod(m, name, mod.TypeFlag, flags) {
			result = result || m.Value().Flag()
		} else {
			kept = append(kept, m)
		}
	}
	*modList = kept
	return result
}

//pob:block:src/Classes/Item.lua:2160
func isLocalMod(m mod.Mod, name string, modType mod.Type, flags mod.MFlag) bool {
	if m.Name() != name || m.Type() != modType || m.Flags() != flags || m.KeywordFlags() != 0 {
		return false
	}
	tags := m.Tags()
	return len(tags) == 0 || tags[0].Type() == mod.TypeInSlot
}

//pob:block:src/Classes/Item.lua:2278-2280
func listValues(modList []mod.Mod, name string) map[string]float64 {
	out := map[string]float64{}
	for _, m := range modList {
		if m.Name() != name {
			continue
		}
		if kv, ok := m.Value().ValueList.(mod.WeaponData); ok {
			out[kv.Key] = kv.Value
		}
		if kv, ok := m.Value().ValueList.(mod.ArmourData); ok {
			out[kv.Key] = kv.Value
		}
	}
	return out
}

const (
	slotNamePlaceholder     = "{SlotName}"
	handPlaceholder         = "{Hand}"
	otherSlotNumPlaceholder = "{OtherSlotNum}"
)

//pob:src/Classes/Item.lua:2127-2143
func (i *Item) getPrimarySlot() string {
	switch {
	case i.Base != nil && i.Base.Weapon != nil:
		return "Weapon 1"
	case i.Type == "Quiver", i.Type == "Shield":
		return "Weapon 2"
	case i.Type == "Ring":
		return "Ring 1"
	case i.Type == "Flask":
		return "Flask 1"
	case i.Type == "Tincture":
		return "Flask 1"
	case i.Type == "Graft":
		return "Graft 1"
	}
	return i.Type
}

//pob:block:src/Classes/Item.lua:2196-2198
func slotPlaceholderReplacer(slotName string, slotNum int) *strings.Replacer {
	hand, otherSlotNum := "OffHand", "1"
	if slotNum == 1 {
		hand, otherSlotNum = "MainHand", "2"
	}
	return strings.NewReplacer(
		slotNamePlaceholder, slotName,
		handPlaceholder, hand,
		otherSlotNumPlaceholder, otherSlotNum,
	)
}

//pob:block:src/Classes/Item.lua:2194-2200
func substituteSlotPlaceholders(m mod.Mod, r *strings.Replacer) mod.Mod {
	out := m
	for idx, tag := range m.Tags() {
		rewritten, changed := rewriteTagStrings(tag, r)
		if !changed {
			continue
		}
		if out == m {
			out = m.Clone()
		}
		out.Tags()[idx] = rewritten
	}
	return out
}

//pob:block:src/Classes/Item.lua:2195-2199
func rewriteTagStrings(tag mod.Tag, r *strings.Replacer) (mod.Tag, bool) {
	v := reflect.ValueOf(tag)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return tag, false
	}

	src := v.Elem()
	var dst reflect.Value
	field := func(idx int) reflect.Value {
		if !dst.IsValid() {
			dst = reflect.New(src.Type())
			dst.Elem().Set(src)
		}
		return dst.Elem().Field(idx)
	}

	for idx := 0; idx < src.NumField(); idx++ {
		if src.Type().Field(idx).PkgPath != "" {
			continue
		}

		f := src.Field(idx)
		switch f.Kind() {
		case reflect.String:
			if s := f.String(); r.Replace(s) != s {
				field(idx).SetString(r.Replace(s))
			}
		case reflect.Pointer:
			if f.IsNil() || f.Elem().Kind() != reflect.String {
				continue
			}
			if s := f.Elem().String(); r.Replace(s) != s {
				p := reflect.New(f.Type().Elem())
				p.Elem().SetString(r.Replace(s))
				field(idx).Set(p)
			}
		case reflect.Slice:
			if f.Type().Elem().Kind() != reflect.String {
				continue
			}
			var replaced reflect.Value
			for j := 0; j < f.Len(); j++ {
				s := f.Index(j).String()
				if r.Replace(s) == s {
					continue
				}
				if !replaced.IsValid() {
					replaced = reflect.MakeSlice(f.Type(), f.Len(), f.Len())
					reflect.Copy(replaced, f)
				}
				replaced.Index(j).SetString(r.Replace(s))
			}
			if replaced.IsValid() {
				field(idx).Set(replaced)
			}
		}
	}

	if !dst.IsValid() {
		return tag, false
	}
	return dst.Interface().(mod.Tag), true
}

var socketColourNames = map[string]string{"R": "Red", "G": "Green", "B": "Blue", "W": "White"}

//pob:src/Classes/Item.lua:2178-2468
//nolint:gocyclo,maintidx // Mirrors the Lua derivation blocks.
func (i *Item) buildModListForSlotNum(baseList []mod.Mod, slotNum int) []mod.Mod {
	//pob:block:src/Classes/Item.lua:2179-2181
	slotName := i.getPrimarySlot()
	if slotNum != 1 {
		slotName = strings.ReplaceAll(slotName, "1", utils.Ternary(slotNum == 0, "nil", strconv.Itoa(slotNum)))
	}

	replacer := slotPlaceholderReplacer(slotName, slotNum)
	modList := make([]mod.Mod, 0, len(baseList))
	for _, m := range baseList {
		if !modAppliesToSlotNum(m, slotNum) {
			continue
		}
		modList = append(modList, substituteSlotPlaceholders(m, replacer))
	}

	for _, socket := range i.Sockets {
		if colour, ok := socketColourNames[socket.Color]; ok {
			modList = append(modList, mod.NewFloat("Multiplier:"+colour+"SocketIn"+slotName, mod.TypeBase, 1).
				Source(mod.Source("Item Sockets")))
		}
	}

	if len(i.Sockets) > 0 {
		groupCounts := map[int]int{}
		for _, socket := range i.Sockets {
			groupCounts[socket.Group]++
		}
		unlinked := 0
		for _, count := range groupCounts {
			if count == 1 {
				unlinked++
			}
		}
		modList = append(modList, mod.NewFloat("Multiplier:UnlinkedSocketIn"+slotName, mod.TypeBase, float64(unlinked)).
			Source(mod.Source("Unlinked Item Sockets")))
	}

	calcLocal(&modList, "Quality", mod.TypeBase, 0)

	//pob:block:src/Classes/Item.lua:2237-2239
	quality := 0.0
	if i.Quality != nil {
		quality = float64(*i.Quality)
		modList = append(modList, mod.NewFloat("Multiplier:QualityOn"+slotName, mod.TypeBase, quality).
			Source(mod.Source("Quality")))
	}

	switch {
	case i.Base.Weapon != nil:
		wb := i.Base.Weapon
		weaponData := map[string]float64{}
		i.WeaponData[slotNum] = weaponData

		weaponData["AttackSpeedInc"] = calcLocal(&modList, "Speed", mod.TypeIncrease, mod.MFlagAttack) +
			math.Floor(quality/8*calcLocal(&modList, "AlternateQualityLocalAttackSpeedPer8Quality", mod.TypeIncrease, 0))
		weaponData["AttackRate"] = utils.RoundTo(wb.AttackRateBase*(1+weaponData["AttackSpeedInc"]/100), 2)
		weaponData["rangeBonus"] = calcLocal(&modList, "WeaponRange", mod.TypeBase, 0) +
			float64(10*calcLocal(&modList, "WeaponRangeMetre", mod.TypeBase, 0)) +
			math.Floor(quality/10*calcLocal(&modList, "AlternateQualityLocalWeaponRangePer10Quality", mod.TypeBase, 0))
		weaponData["range"] = float64(wb.Range) + weaponData["rangeBonus"]

		//pob:block:src/Classes/Item.lua:2250
		localIncEle := calcLocal(&modList, "LocalElementalDamage", mod.TypeIncrease, 0)
		for _, dmgType := range dmgTypeList {
			minVal := weaponBase(wb, dmgType+"Min") + calcLocal(&modList, dmgType+"Min", mod.TypeBase, 0)
			maxVal := weaponBase(wb, dmgType+"Max") + calcLocal(&modList, dmgType+"Max", mod.TypeBase, 0)
			switch {
			case dmgType == "Physical":
				physInc := calcLocal(&modList, "PhysicalDamage", mod.TypeIncrease, 0)
				qualityScalar := quality
				if calcLocal(&modList, "AlternateQualityWeapon", mod.TypeBase, 0) > 0 {
					qualityScalar = 0
				}
				minVal = utils.RoundTo(minVal*(1+physInc/100)*(1+qualityScalar/100), 0)
				maxVal = utils.RoundTo(maxVal*(1+physInc/100)*(1+qualityScalar/100), 0)
			case dmgType != "Chaos":
				//pob:block:src/Classes/Item.lua:2262-2265
				localInc := calcLocal(&modList, "Local"+dmgType+"Damage", mod.TypeIncrease, 0) + localIncEle
				minVal = utils.RoundTo(minVal*(1+localInc/100), 0)
				maxVal = utils.RoundTo(maxVal*(1+localInc/100), 0)
			}
			if minVal > 0 && maxVal > 0 {
				weaponData[dmgType+"Min"] = minVal
				weaponData[dmgType+"Max"] = maxVal
				dps := float64((minVal + maxVal) / 2 * weaponData["AttackRate"])
				weaponData[dmgType+"DPS"] = dps
				if dmgType != "Physical" && dmgType != "Chaos" {
					weaponData["ElementalDPS"] += dps
				}
			}
		}
		//pob:block:src/Classes/Item.lua:2277
		weaponData["CritChance"] = utils.RoundTo((wb.CritChanceBase+calcLocal(&modList, "CritChance", mod.TypeBase, 0))*
			(1+(calcLocal(&modList, "CritChance", mod.TypeIncrease, 0)+
				math.Floor(quality/4*calcLocal(&modList, "AlternateQualityLocalCritChancePer4Quality", mod.TypeIncrease, 0)))/100), 2)
		//pob:block:src/Classes/Item.lua:2278-2280
		for _, m := range modList {
			if m.Name() != "WeaponData" {
				continue
			}
			if kv, ok := m.Value().ValueList.(mod.WeaponData); ok {
				if kv.Unset {
					delete(weaponData, kv.Key)
				} else {
					weaponData[kv.Key] = kv.Value
				}
			}
		}
		convertWeaponModsToLocal(modList, slotNum)
		weaponData["TotalDPS"] = 0
		for _, dmgType := range dmgTypeList {
			weaponData["TotalDPS"] += weaponData[dmgType+"DPS"]
		}

	case i.Base.Armour != nil:
		ab := i.Base.Armour
		armourData := i.ArmourData
		if armourData == nil {
			armourData = map[string]float64{}
			i.ArmourData = armourData
		}

		armourBase := calcLocal(&modList, "Armour", mod.TypeBase, 0) + float64(ab.ArmourMin)
		armourVariance := float64(ab.ArmourMax - ab.ArmourMin)
		armourEvasionBase := calcLocal(&modList, "ArmourAndEvasion", mod.TypeBase, 0)
		evasionBase := calcLocal(&modList, "Evasion", mod.TypeBase, 0) + float64(ab.EvasionMin)
		evasionVariance := float64(ab.EvasionMax - ab.EvasionMin)
		evasionEnergyShieldBase := calcLocal(&modList, "EvasionAndEnergyShield", mod.TypeBase, 0)
		energyShieldBase := calcLocal(&modList, "EnergyShield", mod.TypeBase, 0) + float64(ab.EnergyShieldMin)
		energyShieldVariance := float64(ab.EnergyShieldMax - ab.EnergyShieldMin)
		armourEnergyShieldBase := calcLocal(&modList, "ArmourAndEnergyShield", mod.TypeBase, 0)
		wardBase := calcLocal(&modList, "Ward", mod.TypeBase, 0) + float64(ab.WardMin)
		wardVariance := float64(ab.WardMax - ab.WardMin)
		armourInc := calcLocal(&modList, "Armour", mod.TypeIncrease, 0)
		armourEvasionInc := calcLocal(&modList, "ArmourAndEvasion", mod.TypeIncrease, 0)
		evasionInc := calcLocal(&modList, "Evasion", mod.TypeIncrease, 0)
		evasionEnergyShieldInc := calcLocal(&modList, "EvasionAndEnergyShield", mod.TypeIncrease, 0)
		energyShieldInc := calcLocal(&modList, "EnergyShield", mod.TypeIncrease, 0)
		wardInc := calcLocal(&modList, "Ward", mod.TypeIncrease, 0)
		armourEnergyShieldInc := calcLocal(&modList, "ArmourAndEnergyShield", mod.TypeIncrease, 0)
		defencesInc := calcLocal(&modList, "Defences", mod.TypeIncrease, 0)
		qualityScalar := quality
		if calcLocal(&modList, "AlternateQualityArmour", mod.TypeBase, 0) > 0 {
			qualityScalar = 0
		}

		if v, ok := armourData["Armour"]; ok && v > 0 {
			if _, set := armourData["ArmourBasePercentile"]; !set && armourVariance != 0 {
				p := (v/((1+(armourInc+armourEvasionInc+armourEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100)) - armourBase) / armourVariance
				armourData["ArmourBasePercentile"] = utils.RoundTo(math.Max(math.Min(p, 1), 0), 4)
			}
		}
		if v, ok := armourData["Evasion"]; ok && v > 0 {
			if _, set := armourData["EvasionBasePercentile"]; !set && evasionVariance != 0 {
				p := (v/((1+(evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100)) - evasionBase) / evasionVariance
				armourData["EvasionBasePercentile"] = utils.RoundTo(math.Max(math.Min(p, 1), 0), 4)
			}
		}
		if v, ok := armourData["EnergyShield"]; ok && v > 0 {
			if _, set := armourData["EnergyShieldBasePercentile"]; !set && energyShieldVariance != 0 {
				p := (v/((1+(energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100)) - energyShieldBase) / energyShieldVariance
				armourData["EnergyShieldBasePercentile"] = utils.RoundTo(math.Max(math.Min(p, 1), 0), 4)
			}
		}
		if v, ok := armourData["Ward"]; ok && v > 0 {
			if _, set := armourData["WardBasePercentile"]; !set && wardVariance != 0 {
				p := (v/((1+(wardInc+defencesInc)/100)*(1+qualityScalar/100)) - wardBase) / wardVariance
				armourData["WardBasePercentile"] = utils.RoundTo(math.Max(math.Min(p, 1), 0), 4)
			}
		}

		armourData["Armour"] = utils.RoundTo((armourBase+armourEvasionBase+armourEnergyShieldBase+float64(armourVariance*percentileOr(armourData, "ArmourBasePercentile")))*(1+(armourInc+armourEvasionInc+armourEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100), 0)
		armourData["Evasion"] = utils.RoundTo((evasionBase+armourEvasionBase+evasionEnergyShieldBase+float64(evasionVariance*percentileOr(armourData, "EvasionBasePercentile")))*(1+(evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100), 0)
		armourData["EnergyShield"] = utils.RoundTo((energyShieldBase+evasionEnergyShieldBase+armourEnergyShieldBase+float64(energyShieldVariance*percentileOr(armourData, "EnergyShieldBasePercentile")))*(1+(energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)/100)*(1+qualityScalar/100), 0)
		armourData["Ward"] = utils.RoundTo((wardBase+float64(wardVariance*percentileOr(armourData, "WardBasePercentile")))*(1+(wardInc+defencesInc)/100)*(1+qualityScalar/100), 0)

		//pob:block:src/Classes/Item.lua:2345-2356
		for _, k := range []string{"Armour", "Evasion", "EnergyShield", "Ward"} {
			if _, set := armourData[k+"BasePercentile"]; !set && armourData[k] > 0 {
				armourData[k+"BasePercentile"] = 1
			}
		}

		if ab.BlockChance != 0 {
			blockChance := float64(ab.BlockChance) + calcLocal(&modList, "BlockChance", mod.TypeBase, 0)
			blockChanceInc := calcLocal(&modList, "BlockChance", mod.TypeIncrease, 0)
			armourData["BlockChance"] = math.Floor(blockChance * (1 + blockChanceInc/100))
		}
		if ab.MovementPenalty != 0 {
			modList = append(modList, mod.NewFloat("MovementSpeed", mod.TypeIncrease, -float64(ab.MovementPenalty)).
				Source(mod.Source("Item:-1:"+i.Name)).Tag(mod.Condition("IgnoreMovementPenalties").Neg(true)))
		}
		for k, v := range listValues(modList, "ArmourData") {
			armourData[k] = v
		}

	case i.Base.Flask != nil:
		fb := i.Base.Flask
		flaskData := map[string]float64{}
		i.FlaskData = flaskData

		durationInc := calcLocal(&modList, "Duration", mod.TypeIncrease, 0)
		durationMore := calcLocal(&modList, "Duration", mod.TypeMore, 0)
		if fb.Life > 0 || fb.Mana > 0 {
			// Recovery flask.
			flaskData["instantPerc"] = calcLocal(&modList, "FlaskInstantRecovery", mod.TypeBase, 0)
			//pob:block:src/Classes/Item.lua:2374
			flaskData["instantLowLifePerc"] = calcLocal(&modList, "FlaskLowLifeInstantRecovery", mod.TypeBase, 0)
			recoveryMod := 1 + calcLocal(&modList, "FlaskRecovery", mod.TypeIncrease, 0)/100
			rateMod := 1 + calcLocal(&modList, "FlaskRecoveryRate", mod.TypeIncrease, 0)/100
			flaskData["duration"] = utils.RoundTo(fb.Duration*(1+durationInc/100)/rateMod*durationMore, 1)
			if fb.Life > 0 {
				flaskData["lifeBase"] = float64(fb.Life) * (1 + quality/100) * recoveryMod
				flaskData["lifeInstant"] = flaskData["lifeBase"] * flaskData["instantPerc"] / 100
				flaskData["lifeGradual"] = flaskData["lifeBase"] * (1 - flaskData["instantPerc"]/100)
				flaskData["lifeTotal"] = flaskData["lifeInstant"] + flaskData["lifeGradual"]
				flaskData["lifeAdditional"] = calcLocal(&modList, "FlaskAdditionalLifeRecovery", mod.TypeBase, 0)
				//pob:block:src/Classes/Item.lua:2384
				flaskData["lifeEffectNotRemoved"] = utils.Ternary(calcLocalFlag(&baseList, "LifeFlaskEffectNotRemoved", 0), 1.0, 0.0)
			}
			if fb.Mana > 0 {
				flaskData["manaBase"] = float64(fb.Mana) * (1 + quality/100) * recoveryMod
				flaskData["manaInstant"] = flaskData["manaBase"] * flaskData["instantPerc"] / 100
				flaskData["manaGradual"] = flaskData["manaBase"] * (1 - flaskData["instantPerc"]/100)
				flaskData["manaTotal"] = flaskData["manaInstant"] + flaskData["manaGradual"]
				//pob:block:src/Classes/Item.lua:2391
				flaskData["manaEffectNotRemoved"] = utils.Ternary(calcLocalFlag(&baseList, "ManaFlaskEffectNotRemoved", 0), 1.0, 0.0)
			}
		} else {
			// Utility flask.
			//pob:block:src/Classes/Item.lua:2395
			flaskData["duration"] = utils.RoundTo(fb.Duration*(1+durationInc/100)*(1+quality/100)*durationMore, 1)
		}
		flaskData["chargesMax"] = float64(fb.ChargesMax) + calcLocal(&modList, "FlaskCharges", mod.TypeBase, 0)
		flaskData["chargesUsed"] = math.Floor(float64(fb.ChargesUsed) * (1 + calcLocal(&modList, "FlaskChargesUsed", mod.TypeIncrease, 0)/100))
		flaskData["gainMod"] = 1 + calcLocal(&modList, "FlaskChargeRecovery", mod.TypeIncrease, 0)/100
		//pob:block:src/Classes/Item.lua:2400
		flaskData["effectInc"] = calcLocal(&modList, "FlaskEffect", mod.TypeIncrease, 0) +
			calcLocal(&modList, "LocalEffect", mod.TypeIncrease, 0)

	case i.Base.Tincture != nil:
		//pob:block:src/Classes/Item.lua:2404-2409
		tinctureData := map[string]float64{}
		i.TinctureData = tinctureData
		tb := tinctureBaseOf(i.Base.Name)
		tinctureData["manaBurn"] = (tb.ManaBurn + 0.01) / (1 + calcLocal(&modList, "TinctureManaBurnRate", mod.TypeIncrease, 0)/100) /
			(1 + calcLocal(&modList, "TinctureManaBurnRate", mod.TypeMore, 0)/100)
		tinctureData["cooldownInc"] = calcLocal(&modList, "TinctureCooldownRecovery", mod.TypeIncrease, 0) +
			calcLocal(&modList, "CooldownRecovery", mod.TypeIncrease, 0)
		tinctureData["cooldown"] = tb.Cooldown / (1 + tinctureData["cooldownInc"]/100)
		tinctureData["effectInc"] = calcLocal(&modList, "TinctureEffect", mod.TypeIncrease, 0) +
			calcLocal(&modList, "LocalEffect", mod.TypeIncrease, 0)

	case i.Type == "Jewel":
		//pob:block:src/Classes/Item.lua:2413-2418
		if strings.Contains(i.Name, "Grand Spectrum") {
			spectrumMod := mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1).
				Source(mod.Source(i.Name))
			modList = append(modList,
				spectrumMod,
				MOD("MinionModifier", "LIST", mod.MinionModifier{Mod: spectrumMod}).
					Source(mod.Source(i.Name)))
		}
	}

	return modList
}

//pob:block:src/Classes/Item.lua:1116-1127
func convertWeaponModsToLocal(modList []mod.Mod, slotNum int) {
	hand := "OffHandAttack"
	if slotNum == 1 {
		hand = "MainHandAttack"
	}

	for idx, m := range modList {
		name, flags, kw, tags := m.Name(), m.Flags(), m.KeywordFlags(), m.Tags()
		switch {
		case ((name == "Accuracy" && flags == 0) ||
			(name == "ImpaleChance" && flags != mod.MFlagSpell) ||
			((name == "LifeOnHit" || name == "ManaOnHit") && flags == mod.MFlagAttack) ||
			((name == "PhysicalDamageLifeLeech" || name == "PhysicalDamageManaLeech") && flags == mod.MFlagAttack)) &&
			(kw == 0 || kw == mod.KeywordFlagAttack) && len(tags) == 0:
			modList[idx] = m.Clone().Tag(mod.Condition(hand))
		case (name == "PoisonChance" || name == "BleedChance") && flags != mod.MFlagSpell &&
			(len(tags) == 0 || (len(tags) == 1 && isCriticalStrikeCondition(tags[0]))):
			modList[idx] = m.Clone().Tag(mod.Condition(hand))
		}
	}
}

//pob:block:src/Classes/Item.lua:1124
func isCriticalStrikeCondition(tag mod.Tag) bool {
	c, ok := tag.(*mod.ConditionTag)
	return ok && len(c.VarList) == 1 && c.VarList[0] == "CriticalStrike"
}

//pob:block:src/Classes/Item.lua:2252-2253
func weaponBase(wb *ItemBaseWeapon, key string) float64 {
	switch key {
	case "PhysicalMin":
		return float64(wb.PhysicalMin)
	case "PhysicalMax":
		return float64(wb.PhysicalMax)
	default:
		return 0
	}
}

//pob:block:src/Classes/Item.lua:1175-1178
func percentileOr(armourData map[string]float64, key string) float64 {
	if v, ok := armourData[key]; ok {
		return v
	}
	return 1
}

//pob:block:src/Classes/Item.lua:1032-1033
func (i *Item) defaultSocketColor() string {
	str, dex, intel := i.Requirements["str"], i.Requirements["dex"], i.Requirements["int"]
	maxReq := str
	if dex > maxReq {
		maxReq = dex
	}
	if intel > maxReq {
		maxReq = intel
	}

	switch {
	case maxReq == dex:
		return "G"
	case maxReq == intel:
		return "B"
	default:
		return "R"
	}
}

//pob:src/Classes/Item.lua:1644-1652
func (i *Item) NormaliseQuality() {
	if i.Base == nil || !(i.Base.Armour != nil || i.Base.Weapon != nil || i.Base.Flask != nil ||
		i.Base.Tincture != nil) {
		return
	}

	if i.Quality == nil {
		i.Quality = intPtr(0)
		return
	}

	if i.UniqueID == "" && !i.Corrupted && !i.Split && !i.Mirrored && *i.Quality < 20 {
		i.Quality = intPtr(20)
	}
}

//pob:block:src/Classes/Item.lua:193-212
func matchSpec(line string) (string, string) {
	if strings.Contains(line, ": ") {
		if m := specHexRe.FindStringSubmatch(line); m != nil {
			return m[1], m[2]
		}

		if v, ok := strings.CutPrefix(line, "Selected Variant Group: "); ok {
			return "Selected Variant Group", v
		}
		if m := specNumRe.FindStringSubmatch(line); m != nil {
			if _, ok := luaToNumber(m[2]); ok {
				return m[1], m[2]
			}
		}

		if m := specAnyRe.FindStringSubmatch(line); m != nil {
			return m[1], m[2]
		}
	}

	if m := reqClassRe.FindStringSubmatch(line); m != nil {
		return m[1], m[2]
	}

	if m := classColonRe.FindStringSubmatch(line); m != nil {
		return "Requires Class", m[1]
	}

	if m := requiresRe.FindStringSubmatch(line); m != nil {
		return m[1], m[2]
	}

	return "", ""
}

//pob:block:src/Classes/Item.lua:194-356
//nolint:gocyclo // Mirrors the Lua branch chain.
func (i *Item) applySpec(specName, specVal string, implicitLines *int, gameModeStage *string, foundExplicit *bool, importedLevelReq **int) bool {
	num := func() *int {
		if v, ok := specToNumber(specVal); ok {
			return intPtr(int(v))
		}
		return nil
	}

	switch specName {
	case "Unique ID":
		i.UniqueID = specVal
	case "Item Level":
		i.ItemLevel = num()
	case "Requires Class":
		i.ClassRestriction = specVal
	case "Quality":
		i.Quality = num()
	case "Sockets":
		group := 0
		for _, c := range strings.Split(specVal, "") {
			switch {
			case socketColorRe.MatchString(c):
				i.Sockets = append(i.Sockets, ItemSocket{Color: c, Group: group})
			case c == " ":
				group++
			}
		}
	case "Radius":
		if i.Type == "Jewel" {
			i.JewelRadiusLabel = alphaPrefixRe.FindString(specVal)
		}
	case "Limited to":
		if i.Type == "Jewel" {
			i.Limit = num()
		}
	case "Variant":
		if m := variantNameRe.FindStringSubmatch(specVal); m != nil {
			i.VariantList = append(i.VariantList, m[2])
		} else {
			i.VariantList = append(i.VariantList, specVal)
		}
	case "Talisman Tier":
		i.TalismanTier = num()
	case "Requires Level", "LevelReq":
		if v := num(); v != nil {
			i.Requirements["level"] = float64(*v)
		}
	case "Requires":
		if m := levelRe.FindStringSubmatch(specVal); m != nil {
			if v, ok := luaToNumber(m[1]); ok {
				i.Requirements["level"] = v
			}
		}
	case "Level":
		*importedLevelReq = num()
	case "Selected Variant":
		i.Variant = num()
	//pob:block:src/Classes/Item.lua:682-687,src/Classes/Item.lua:739-747,src/Classes/Item.lua:760-761
	case "Version":
		i.VersionList = append(i.VersionList, specVal)
		i.UsesVariantGroups = true
	case "Selected Version":
		i.SelectedVersion = num()
		i.UsesVariantGroups = true
	case "Selected Variant Group":
		if m := variantGroupSpecRe.FindStringSubmatch(specVal); m != nil {
			g, _ := strconv.Atoi(m[1])
			v, _ := strconv.Atoi(m[2])
			if i.VariantGroupSelections == nil {
				i.VariantGroupSelections = map[int]int{}
			}
			i.VariantGroupSelections[g] = v
			i.UsesVariantGroups = true
		}
	case "Allow Duplicate Variants":
		i.AllowDuplicateVariants = specVal == "true"
	case "Has Variants", "Selected Variants":
		return true
	case "League":
		i.League = specVal
	case "Crafted":
		i.Crafted = true
	case "Scourge":
		i.Scourge = true
	case "Catalyst":
		for idx, name := range catalystList {
			if specVal == name {
				i.Catalyst = intPtr(idx + 1)
			}
		}
	case "CatalystQuality":
		i.CatalystQuality = num()
	case "Note":
		i.Note = specVal
	case "Source":
		i.Source = specVal
	case "Str", "Strength", "Dex", "Dexterity", "Int", "Intelligence":
		if v, ok := luaToNumber(specVal); ok {
			i.Requirements[strings.ToLower(specName[:3])] = v
		}
	case "Implicits":
		if v, ok := luaToNumber(specVal); ok {
			*implicitLines = int(v)
		} else {
			*implicitLines = 0
		}
		*gameModeStage = "EXPLICIT"
	case "Armour", "Evasion Rating", "Evasion", "Energy Shield", "Ward":
		if i.ArmourData == nil {
			i.ArmourData = map[string]float64{}
		}
		key := strings.ReplaceAll(strings.ReplaceAll(specName, "Rating", ""), " ", "")
		if v, ok := luaToNumber(augmentedRe.ReplaceAllString(specVal, "")); ok {
			i.ArmourData[key] = v
		}
	case "Critical Strike Range", "Attacks per Second", "Weapon Range",
		"Critical Strike Chance", "Physical Damage", "Elemental Damage",
		"Chaos Damage", "Chance to Block", "Block chance":
	case "Has Alt Variant":
		i.HasAltVariant = true
	case "Has Alt Variant Two":
		i.HasAltVariant2 = true
	case "Has Alt Variant Three":
		i.HasAltVariant3 = true
	case "Has Alt Variant Four":
		i.HasAltVariant4 = true
	case "Has Alt Variant Five":
		i.HasAltVariant5 = true
	case "Selected Alt Variant":
		i.VariantAlt = num()
	case "Selected Alt Variant Two":
		i.VariantAlt2 = num()
	case "Selected Alt Variant Three":
		i.VariantAlt3 = num()
	case "Selected Alt Variant Four":
		i.VariantAlt4 = num()
	case "Selected Alt Variant Five":
		i.VariantAlt5 = num()
	case "Memory Strands":
		i.MemoryStrands = num()
	case "Intangibility":
		i.Intangibility = num()
	//pob:block:src/Classes/Item.lua:801-805
	case "Unreleased":
		i.Unreleased = specVal == "true"
	case "Upgrade":
		i.UpgradePaths = append(i.UpgradePaths, specVal)
	case "Prefix", "Suffix", "Implicit",
		"Cluster Jewel Skill", "Cluster Jewel Node Count",
		"Crucible":
	default:
		if m := catalystQualitySpecRe.FindStringSubmatch(specName); m != nil {
			if v, ok := specToNumber(specVal); ok {
				i.CatalystQuality = intPtr(int(v))
			}
			for idx, descriptor := range catalystDescriptorList {
				if m[1] == descriptor {
					i.Catalyst = intPtr(idx + 1)
				}
			}
			return false
		}

		if strings.Contains(specName, "BasePercentile") {
			if i.ArmourData == nil {
				i.ArmourData = map[string]float64{}
			}
			if v, ok := luaToNumber(specVal); ok {
				i.ArmourData[specName] = v
			} else {
				i.ArmourData[specName] = 0
			}
			return false
		}

		if !(strings.Contains(i.Name, specName) && strings.Contains(i.Name, specVal)) {
			*foundExplicit = true
			*gameModeStage = "EXPLICIT"
		}
	}

	return false
}

//pob:block:src/Classes/Item.lua:361-552
//nolint:gocyclo,maintidx // Mirrors the Lua block.
func (i *Item) parseModLine(line string, l int, mode string, implicitLines int, gameModeStage *string, foundExplicit, foundImplicit *bool) int {
	var variantList map[int]bool
	if m := variantSpecRe.FindStringSubmatch(line); m != nil {
		variantList = map[int]bool{}
		for _, d := range variantDigitRe.FindAllString(m[1], -1) {
			if v, err := strconv.Atoi(d); err == nil {
				variantList[v] = true
			}
		}
	}

	if variantSpecRe.ReplaceAllString(line, "") == "Two-Toned Boots" {
		line = "Two-Toned Boots (Armour/Energy Shield)"
	}

	if i.resolveBase(line, variantList) {
		return l
	}

	fractured := strings.Contains(line, "{fractured}") || strings.Contains(line, " (fractured)")
	rangeSpec := rangeSpecRe.FindStringSubmatch(line)
	if rangeSpec != nil {
		i.AdvancedCopy = true
	}
	enchant := strings.Contains(line, " (enchant)")
	classReq := strings.Contains(line, "Requires Class")
	scourge := strings.Contains(line, "{scourge}") || strings.Contains(line, " (scourge)")
	crafted := strings.Contains(line, "{crafted}") || strings.Contains(line, " (crafted)") || enchant
	custom := strings.Contains(line, "{custom}")
	mutated := strings.Contains(line, "{mutated}")
	crucible := strings.Contains(line, "{crucible}") || strings.Contains(line, " (crucible)")
	vestigial := strings.Contains(line, "{vestigial}")
	disabled := strings.Contains(line, "{disabled}") || strings.Contains(line, " (disabled)")
	prefix := strings.Contains(line, "{prefix}")
	suffix := strings.Contains(line, "{suffix}")

	modGroup := ""
	if m := modGroupRe.FindStringSubmatch(line); m != nil {
		modGroup = m[1]
	}
	//pob:block:src/Classes/Item.lua:856-870
	var versionList, variantGroupList map[int]bool
	if m := versionSpecRe.FindStringSubmatch(line); m != nil {
		versionList = idSet(m[1])
		i.UsesVariantGroups = true
	}
	if m := groupSpecRe.FindStringSubmatch(line); m != nil {
		variantGroupList = map[int]bool{}
		for id := range idSet(m[1]) {
			if id > 0 {
				variantGroupList[id] = true
			}
		}
		i.UsesVariantGroups = true
	}

	var corruptedRange *float64
	if m := corruptedRangeRe.FindStringSubmatch(line); m != nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(m[1]), 64); err == nil {
			corruptedRange = &v
		}
	}

	var modTags []string
	if m := tagsRe.FindStringSubmatch(line); m != nil {
		for _, t := range strings.Split(m[1], ",") {
			if t = strings.TrimSpace(t); t != "" {
				modTags = append(modTags, t)
			}
		}
	}

	implicit := strings.Contains(line, " (implicit)") || enchant
	if implicit {
		*foundImplicit = true
		*gameModeStage = "IMPLICIT"
	}

	line = stripBalancedBraces(line)
	for _, s := range []string{" (fractured)", " (crafted)", " (implicit)", " (enchant)", " (scourge)"} {
		line = strings.ReplaceAll(line, s, "")
	}

	if advancedCopyRangeRe.MatchString(line) {
		i.AdvancedCopy = true
	}

	catalystScalar := getCatalystScalar(i.Catalyst, modTags, i.CatalystQuality, prefix, suffix)

	var lineRange *float64
	if rangeSpec != nil {
		if v, ok := luaToNumber(rangeSpec[1]); ok {
			lineRange = &v
		}
	}

	//pob:block:src/Classes/Item.lua:1065-1067
	if strings.Contains(line, "(") {
		line = advancedCopyBaseValueRe.ReplaceAllString(line, "$1")
	}

	var bestPrecision *float64
	line, bestPrecision = advancedCopyRollRanges(line)
	if bestPrecision != nil {
		lineRange = bestPrecision
	}

	rangedLine := data.ApplyRange(line, 1, &catalystScalar, corruptedRange)
	entry := ParseMod(rangedLine, false)

	nextModGroup := ""
	if l+1 < len(i.rawLines) {
		if m := modGroupRe.FindStringSubmatch(i.rawLines[l+1]); m != nil {
			nextModGroup = m[1]
		}
	}

	if (entry.ModList == nil || modExtra(entry) != "") && l+1 < len(i.rawLines) && modGroup == nextModGroup {
		nextLine := stripBalancedBraces(i.rawLines[l+1])
		for _, s := range []string{" (fractured)", " (crafted)", " (implicit)", " (enchant)", " (scourge)",
			"(fractured)", "(crafted)", "(implicit)", "(enchant)", "(scourge)"} {
			nextLine = strings.ReplaceAll(nextLine, s, "")
		}

		combLine := line + " " + nextLine
		rangedLine = data.ApplyRange(combLine, 1, &catalystScalar, corruptedRange)

		combEntry := ParseMod(rangedLine, true)
		if combEntry.ModList != nil && modExtra(combEntry) == "" {
			line = line + "\n" + nextLine
			l++
			entry = combEntry
		} else {
			entry = ParseMod(rangedLine, false)
		}
	}

	lineLower := strings.ToLower(line)
	if disabled {
		lineLower = ""
	}

	switch {
	case strings.Contains(lineLower, "can be anointed"):
		i.CanBeAnointed = true
	case lineLower == "can have a second enchantment modifier" || lineLower == "can have 1 additional enchantment modifiers":
		i.CanHaveTwoEnchants = true
	case lineLower == "can have 2 additional enchantment modifiers":
		i.CanHaveTwoEnchants = true
		i.CanHaveThreeEnchants = true
	case lineLower == "can have 3 additional enchantment modifiers":
		i.CanHaveTwoEnchants = true
		i.CanHaveThreeEnchants = true
		i.CanHaveFourEnchants = true
	}

	i.collectModMagnitude(line, rangedLine, disabled, lineRange, catalystScalar, corruptedRange)

	var target2 *[]ModLine
	switch {
	case enchant || (crafted && len(i.EnchantModLines)+len(i.ImplicitModLines) < implicitLines):
		target2 = &i.EnchantModLines
	case scourge:
		target2 = &i.ScourgeModLines
	case classReq:
		target2 = &i.ClassRequirementModLines
	case implicit || (!crafted && len(i.EnchantModLines)+len(i.ScourgeModLines)+len(i.ImplicitModLines) < implicitLines):
		target2 = &i.ImplicitModLines
	//pob:block:src/Classes/Item.lua:1293-1294
	case crucible:
		target2 = &i.CrucibleModLines
	default:
		target2 = &i.ExplicitModLines
	}

	newLine := ModLine{
		Line:           line,
		VariantList:    variantList,
		Crafted:        crafted,
		Custom:         custom,
		Fractured:      fractured,
		Mutated:        mutated,
		Vestigial:      vestigial,
		Scourge:        scourge,
		Implicit:       implicit,
		Prefix:         prefix,
		Suffix:         suffix,
		Disabled:       disabled,
		CorruptedRange: corruptedRange,
		ModGroup:       modGroup,
		Crucible:       crucible,

		VersionList:      versionList,
		VariantGroupList: variantGroupList,
	}
	i.registerVariantGroups(newLine)

	switch {
	case entry.ModList != nil:
		newLine.Extra = modExtra(entry)
		newLine.ModList = entry.ModList
		newLine.ModTags = modTags
		newLine.ValueScalar = catalystScalar
		newLine.Range = defaultItemAffixQuality
		if lineRange != nil {
			newLine.Range = lineRange
		}
		*target2 = append(*target2, newLine)

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
	//pob:src/Classes/Item.lua:1317-1325
	case mode == "GAME":
		if *gameModeStage == "IMPLICIT" || *gameModeStage == "EXPLICIT" ||
			(*gameModeStage == "FINDIMPLICIT" && ItemBaseByName(line) == nil && i.Name != line &&
				!strings.Contains(line, "Two-Toned") && !i.lineIsBaseType(line)) {
			newLine.Extra = line
			newLine.Range = defaultItemAffixQuality
			*target2 = append(*target2, newLine)
		} else if *gameModeStage == "FINDEXPLICIT" {
			*gameModeStage = "DONE"
		}
	case *foundExplicit:
		newLine.Extra = line
		*target2 = append(*target2, newLine)
	}

	return l
}

//pob:block:src/Classes/Item.lua:1318
func (i *Item) lineIsBaseType(line string) bool {
	if i.Base == nil {
		return false
	}
	if line == i.Base.Type {
		return true
	}
	return i.Base.SubType != "" && line == i.Base.SubType+" "+i.Base.Type
}

//pob:block:src/Classes/Item.lua:931-1053
func (i *Item) resolveBase(line string, variantList map[int]bool) bool {
	baseName := ""

	if i.Base == nil && (i.Rarity == RarityNormal || i.Rarity == RarityMagic) {
		if ItemBaseByName(i.Name) != nil {
			baseName = i.Name
		} else {
			if itemBaseName, s := findItemBaseInName(i.Name); s >= 0 {
				i.NamePrefix = i.Name[:s]
				i.NameSuffix = i.Name[s+len(itemBaseName):]
				baseName = itemBaseName
			} else if hasEmptyItemBaseName() {
				i.NamePrefix = ""
				i.NameSuffix = i.Name
			}
		}

		if baseName == "" {
			if s := strings.Index(i.Name, "Two-Toned Boots"); s >= 0 {
				baseName = "Two-Toned Boots (Armour/Energy Shield)"
				i.NamePrefix = i.Name[:s]
				i.NameSuffix = i.Name[s+len("Two-Toned Boots"):]
			}
		}

		i.Name = parenSuffixRe.ReplaceAllString(i.Name, "")
	}

	if baseName == "" {
		baseName = stripBaseNamePrefixes(variantSpecRe.ReplaceAllString(line, ""))
	}

	base := ItemBaseByName(baseName)
	if baseName == "" || base == nil {
		return false
	}

	if i.Variant == nil || variantList == nil || variantList[*i.Variant] {
		i.BaseName = baseName
		if !(i.Rarity == RarityNormal || i.Rarity == RarityMagic) {
			i.Title = i.Name
		}
		i.Type = base.Type
		i.Base = base
		//pob:block:src/Classes/Item.lua:1011-1016
		if i.Title != "" {
			i.RareLikeUnique = rareLikeUniques[luaLower(i.Title)]
		}
		i.Requirements["str"] = float64(base.Req.Str)
		i.Requirements["dex"] = float64(base.Req.Dex)
		i.Requirements["int"] = float64(base.Req.Int)
	}

	return true
}

//pob:block:src/Classes/Item.lua:439-440
func foilTypeOf(line string) string {
	if m := foilTypeRe.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	return "Rainbow"
}

//pob:block:src/Classes/Item.lua:973
func stripBaseNamePrefixes(line string) string {
	for _, p := range []string{"Superior ", "Synthesised ", "Vestigial "} {
		line = strings.TrimPrefix(line, p)
	}
	return line
}

//pob:block:src/Classes/Item.lua:2187-2193
func modAppliesToSlotNum(m mod.Mod, slotNum int) bool {
	for _, tag := range m.Tags() {
		switch t := tag.(type) {
		case *mod.SlotNumberTag:
			if t.N != slotNum {
				return false
			}
		case *mod.InSlotTag:
			if t.N != slotNum {
				return false
			}
		}
	}
	return true
}

//pob:src/Classes/Item.lua:330-333
var reSpecLeadingNumber = regexp.MustCompile(`^[+-]?[\d.]+`)

func specToNumber(s string) (float64, bool) {
	n := reSpecLeadingNumber.FindString(s)
	if n == "" {
		return 0, false
	}
	return luaToNumber(n)
}

var (
	modMagnitudeTagsRe    = regexp.MustCompile(`(\d+)% ([ir][ne][cd][ru][ec][ae][sd]e?d?) ?([a-zA-Z\s]*) modifier magnitudes`)
	modMagnitudeAffixRe   = regexp.MustCompile(`(\d+)% ([ir][ne][cd][ru][ec][ae][sd]e?d?) effect of ([sp][ur][fe]fix)es`)
	modMagnitudeDoubledRe = regexp.MustCompile(`([a-zA-Z\s]*) modifier magnitudes are doubled`)
)

var modMagnitudeTagMap = map[string]string{
	"defence":     "defences",
	"enchantment": "enchant",
}

func (i *Item) collectModMagnitude(line, rangedLine string, disabled bool, lineRange *float64, catalystScalar float64, corruptedRange *float64) {
	if disabled {
		return
	}

	lowered := strings.ToLower(rangedLine)
	if !strings.Contains(lowered, " modifier magnitudes") && !strings.Contains(lowered, " effect of ") {
		return
	}
	for _, pattern := range []*regexp.Regexp{modMagnitudeTagsRe, modMagnitudeAffixRe, modMagnitudeDoubledRe} {
		if !pattern.MatchString(lowered) {
			continue
		}

		//pob:block:src/Classes/Item.lua:1251
		rangeVal := 1.0
		switch {
		case lineRange != nil:
			rangeVal = *lineRange
		case defaultItemAffixQuality != nil:
			rangeVal = *defaultItemAffixQuality
		}
		reRanged := strings.ToLower(data.ApplyRange(line, rangeVal, &catalystScalar, corruptedRange))
		m := pattern.FindStringSubmatch(reRanged)
		if m == nil {
			continue
		}

		var quality float64
		var multiplier *float64
		var tagsString string

		if pattern == modMagnitudeDoubledRe {
			tagsString = m[1]
			quality = 100
			multiplier = &[]float64{2}[0]
		} else {
			amount, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			switch m[2] {
			case "increased":
				quality = amount
			case "reduced":
				quality = -amount
			default:
				continue
			}
			tagsString = m[3]
		}

		if tagsString == "explicit physical and chaos damage" {
			i.ModMagnitudeMods = append(i.ModMagnitudeMods, ModMagnitudeMod{
				Tags: []string{"damage"}, AnyTags: []string{"physical", "chaos"},
				Quality: quality, ModType: "explicit",
			})
			return
		}

		entry := ModMagnitudeMod{Quality: quality, Multiplier: multiplier}
		for _, word := range strings.Fields(tagsString) {
			word = strings.ToLower(word)
			if mapped, ok := modMagnitudeTagMap[word]; ok {
				word = mapped
			}
			switch word {
			case "implicit", "explicit", "enchant":
				entry.ModType = word
			default:
				entry.Tags = append(entry.Tags, word)
			}
		}
		i.ModMagnitudeMods = append(i.ModMagnitudeMods, entry)
		return
	}
}

func (i *Item) applyModMagnitude() {
	if !(i.AdvancedCopy || i.Crafted) || len(i.ModMagnitudeMods) == 0 {
		return
	}

	for _, mag := range i.ModMagnitudeMods {
		var lists []*[]ModLine
		switch mag.ModType {
		case "implicit":
			lists = []*[]ModLine{&i.ImplicitModLines}
		case "explicit":
			lists = []*[]ModLine{&i.ExplicitModLines}
		case "enchant":
			lists = []*[]ModLine{&i.EnchantModLines}
		default:
			lists = []*[]ModLine{&i.ImplicitModLines, &i.ExplicitModLines, &i.EnchantModLines}
		}

		for _, list := range lists {
			for idx := range *list {
				ml := &(*list)[idx]
				if ml.VariantList != nil && !i.checkModLineVariant(*ml) {
					continue
				}

				tagLookup := make(map[string]bool, len(ml.ModTags)+2)
				for _, t := range ml.ModTags {
					tagLookup[t] = true
				}
				if ml.Prefix {
					tagLookup["prefix"] = true
				}
				if ml.Suffix {
					tagLookup["suffix"] = true
				}
				if ml.Unveiled {
					tagLookup["unveiled"] = true
				}

				match := true
				for _, t := range mag.Tags {
					if !tagLookup[t] {
						match = false
					}
				}
				if len(mag.AnyTags) == 2 && !(tagLookup[mag.AnyTags[0]] || tagLookup[mag.AnyTags[1]]) {
					match = false
				}

				if match {
					if ml.ValueScalar == 0 {
						ml.ValueScalar = 1
					}
					if mag.Multiplier != nil {
						ml.ValueScalar *= *mag.Multiplier
					} else {
						ml.ValueScalar += mag.Quality / 100
					}
				}

				if ml.ValueScalar != 0 && ml.ValueScalar != 1 {
					rangeVal := 1.0
					if ml.Range != nil {
						rangeVal = *ml.Range
					}
					one := 1.0
					rolled := data.ApplyRange(ml.Line, rangeVal, &ml.ValueScalar, &one)
					entry := ParseMod(rolled, false)
					ml.ModList = entry.ModList
					ml.Extra = modExtra(entry)
				}
			}
		}
	}
}

//pob:src/Modules/Main.lua:598-600
//pob:src/Modules/Main.lua:1099-1102
func SetDefaultItemAffixQuality(q float64) {
	q = math.Min(utils.RoundTo(q, 2), 1)
	defaultItemAffixQuality = &q

	globalItemMu.Lock()
	globalItemCache = make(map[string]*Item)
	globalItemMu.Unlock()

	envCache.mu.Lock()
	if envCache.items != nil {
		envCache.items = make(map[string]*Item)
	}
	envCache.mu.Unlock()

	resetItemDB()
}
