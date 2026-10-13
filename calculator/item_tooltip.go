package calculator

import (
	"fmt"
	"maps"
	"math"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

const (
	colorMagic        = "^x8888FF"
	colorUnique       = "^xAF6025"
	colorCrafted      = "^xB8DAF1"
	colorCustom       = "^x5CF0BB"
	colorUnsupported  = "^xF05050"
	colorDisabled     = "^x7F7F7F"
	colorFire         = "^xB97123"
	colorCold         = "^x3F6DB3"
	colorLightning    = "^xADAA47"
	colorChaos        = "^xD02090"
	colorPositive     = "^x33FF77"
	colorNegative     = "^xDD0022"
	colorSource       = "^x88FFFF"
	colorCrucible     = "^xFFA500"
	colorCurrency     = "^xAA9E82"
	colorProphecy     = "^xB54BFF"
	colorScion        = "^xFFF0F0"
	colorStrength     = "^xE05030"
	colorDexterity    = "^x70FF70"
	colorIntelligence = "^x7070FF"
	colorShaper       = "^x55BBFF"
	colorElder        = "^xAA77CC"
	colorFractured    = "^xA29160"
	colorMutated      = "^xCD2285"
	colorAdjudicator  = "^xE9F831"
	colorBasilisk     = "^x00CB3A"
	colorCrusader     = "^x2946FC"
	colorEyrie        = "^xAAB7B8"
	colorCleansing    = "^xF24141"
	colorTangle       = "^x038C8C"
	colorScourge      = "^xFF6E25"
	colorVestigial    = "^xCBA5F1"
	colorIntangible   = "^x9BF4BD"
	colorTip          = "^x80A080"
	colorMemory       = "^xBFE2FA"
	colorGrey         = "^x7F7F7F"
	colorWhite        = "^7"
)

const notSupportedTooltipText = " ^8(Not supported in PoB yet)"

var modLineKindColor = map[ModLineKind]string{
	ModLineMagic: colorMagic, ModLineFractured: colorFractured, ModLineCrafted: colorCrafted, ModLineMutated: colorMutated,
	ModLineScourge: colorScourge, ModLineCustom: colorCustom, ModLineVestigial: colorVestigial, ModLineCrucible: colorCrucible,
}

//pob:block:src/Classes/ItemsTab.lua:39-52
var catalystQualityFormat = []string{
	"^x7F7F7FQuality (Attack Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Speed Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Suffix Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Life and Mana Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Caster Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Attribute Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Physical and Chaos Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Resistance Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Prefix Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Defence Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Elemental Modifiers): " + colorMagic + "+%d%% (augmented)",
	"^x7F7F7FQuality (Critical Modifiers): " + colorMagic + "+%d%% (augmented)",
}

//pob:block:src/Modules/ItemTools.lua:15-25
var influenceInfoAll = []struct {
	on      func(ItemInfluence) bool
	display string
	color   string
}{
	{func(i ItemInfluence) bool { return i.Shaper }, "Shaper", colorShaper},
	{func(i ItemInfluence) bool { return i.Elder }, "Elder", colorElder},
	{func(i ItemInfluence) bool { return i.Adjudicator }, "Warlord", colorAdjudicator},
	{func(i ItemInfluence) bool { return i.Basilisk }, "Hunter", colorBasilisk},
	{func(i ItemInfluence) bool { return i.Crusader }, "Crusader", colorCrusader},
	{func(i ItemInfluence) bool { return i.Eyrie }, "Redeemer", colorEyrie},
	{func(i ItemInfluence) bool { return i.Cleansing }, "Searing Exarch", colorCleansing},
	{func(i ItemInfluence) bool { return i.Tangle }, "Eater of Worlds", colorTangle},
}

type TooltipLine struct {
	Text      string
	Separator bool
}

type ItemTooltip struct {
	Influence []string
	Title     []string
	Lines     []TooltipLine
	Skills    []TooltipSkill
	Children  [][]TooltipLine
}

type TooltipSkill struct {
	SkillID string
	Level   int
}

type ItemTooltipStats struct {
	Str                       float64
	Dex                       float64
	Int                       float64
	Omni                      float64
	OmniscienceRequirements   bool
	OmniAttributeRequirements float64
}

type ItemTooltipContext struct {
	CharacterLevel  int
	ClassName       string
	ShowFlavourText bool
	Stats           ItemTooltipStats
	RadiusData      map[string]float64
	DBMode          bool
}

func (env *Environment) ItemTooltipStats() ItemTooltipStats {
	if env == nil || env.Player == nil || env.ModDB == nil {
		return ItemTooltipStats{}
	}
	return ItemTooltipStats{
		Str: env.Player.Output["Str"], Dex: env.Player.Output["Dex"], Int: env.Player.Output["Int"], Omni: env.Player.Output["Omni"],
		OmniscienceRequirements:   env.ModDB.Flag(nil, "OmniscienceRequirements"),
		OmniAttributeRequirements: env.ModDB.Sum("INC", nil, "OmniAttributeRequirements"),
	}
}

//pob:src/Classes/Tooltip.lua:87-110
func (t *ItemTooltip) addLine(text string) {
	for _, line := range strings.Split(text, "\n") {
		t.Lines = append(t.Lines, TooltipLine{Text: line})
	}
}

//pob:src/Classes/Tooltip.lua:116-123
func (t *ItemTooltip) addSeparator() {
	if n := len(t.Lines); n > 0 && t.Lines[n-1].Separator {
		return
	}
	t.Lines = append(t.Lines, TooltipLine{Separator: true})
}

//pob:src/Modules/Main.lua:1537-1545
func statColor(stat float64, base *float64, limit *float64) string {
	if limit != nil && stat > *limit {
		return colorNegative
	} else if base != nil && stat != *base {
		return colorMagic
	}
	return colorWhite
}

func f64(v float64) *float64 { return &v }

func luaFixed(v float64, prec int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return strconv.FormatFloat(v, 'f', prec, 64)
	}
	exact := new(big.Float).SetFloat64(math.Abs(v)).Text('f', 1100)
	dot := strings.IndexByte(exact, '.')
	digits := []byte(exact[:dot] + exact[dot+1:dot+1+prec])
	rest := strings.TrimRight(exact[dot+1+prec:], "0")
	if rest != "" && rest[0] >= '5' {
		i := len(digits) - 1
		for ; i >= 0; i-- {
			if digits[i] == '9' {
				digits[i] = '0'
				continue
			}
			digits[i]++
			break
		}
		if i < 0 {
			digits = append([]byte{'1'}, digits...)
		}
	}
	intLen := len(digits) - prec
	out := string(digits[:intLen])
	if prec > 0 {
		out += "." + string(digits[intLen:])
	}
	if v < 0 && strings.Trim(out, "0.") != "" {
		out = "-" + out
	}
	return out
}

//pob:src/Classes/ItemsTab.lua:4149-4218
func itemTooltipInfluence(item *Item) []string {
	var header []string
	add := func(name string) {
		if len(header) < 2 {
			header = append(header, name)
		}
	}
	inf := item.Influence
	if inf.Cleansing && inf.Tangle {
		add("Exarch")
		add("Eater")
	} else {
		if item.Fractured {
			add("Fractured")
		}
		if item.Title != "" && strings.Contains(item.Title, "Replica") {
			add("Experimented")
		}
		if item.Foulborn {
			add("Foulborn")
		}
		if inf.Cleansing {
			add("Exarch")
		}
		if inf.Tangle {
			add("Eater")
		}
		if inf.Shaper {
			add("Shaper")
		}
		if inf.Elder {
			add("Elder")
		}
		if inf.Crusader {
			add("Crusader")
		}
		if inf.Eyrie {
			add("Redeemer")
		}
		if inf.Basilisk {
			add("Hunter")
		}
		if inf.Adjudicator {
			add("Warlord")
		}
		if item.Synthesised && len(header) == 0 {
			add("Synthesis")
		}
		if item.MemoryStrands != nil && len(header) == 0 {
			add("Memory")
		}
	}
	if len(header) == 1 {
		header = append(header, header[0])
	}
	return header
}

//pob:src/Classes/ItemsTab.lua:4339-4367
func socketDescriptionLine(item *Item) string {
	var b strings.Builder
	group := 0
	for i, s := range item.Sockets {
		if i > 0 {
			if s.Group == group {
				b.WriteString("^7=")
			} else {
				b.WriteString("  ")
			}
			group = s.Group
		}
		var code string
		switch s.Color {
		case "R":
			code = colorStrength
		case "G":
			code = colorDexterity
		case "B":
			code = colorIntelligence
		case "W":
			code = colorScion
		case "A":
			code = "^xB0B0B0"
		}
		b.WriteString(code + s.Color)
	}
	return b.String()
}

//pob:src/Modules/Build.lua:1941-1975
func addRequirementsToTooltip(t *ItemTooltip, ctx ItemTooltipContext, level, str, dex, intel, strBase, dexBase, intBase float64) {
	var req []string
	if level > 0 {
		req = append(req, fmt.Sprintf("^x7F7F7FLevel %s%d", statColor(level, nil, f64(float64(ctx.CharacterLevel))), int(level)))
	}
	s := ctx.Stats
	if s.OmniscienceRequirements {
		omniSatisfy := s.OmniAttributeRequirements
		highestAttribute := 0.0
		for _, stat := range []float64{str, dex, intel} {
			if stat > highestAttribute {
				highestAttribute = stat
			}
		}
		omni := math.Floor(highestAttribute * (100 / omniSatisfy))
		if omni > 0 || omni > s.Omni {
			req = append(req, fmt.Sprintf("%s%d ^x7F7F7FOmni", statColor(omni, f64(0), f64(s.Omni)), int(omni)))
		}
	} else {
		if str > 14 || str > s.Str {
			req = append(req, fmt.Sprintf("%s%d ^x7F7F7FStr", statColor(str, f64(strBase), f64(s.Str)), int(str)))
		}
		if dex > 14 || dex > s.Dex {
			req = append(req, fmt.Sprintf("%s%d ^x7F7F7FDex", statColor(dex, f64(dexBase), f64(s.Dex)), int(dex)))
		}
		if intel > 14 || intel > s.Int {
			req = append(req, fmt.Sprintf("%s%d ^x7F7F7FInt", statColor(intel, f64(intBase), f64(s.Int)), int(intel)))
		}
	}
	if len(req) > 0 {
		t.addLine("^x7F7F7FRequires " + strings.Join(req, "^x7F7F7F, "))
		t.addSeparator()
	}
}

//pob:src/Modules/ItemTools.lua:357-378
func formattedModLine(ml ModLine, dbMode bool) (string, bool) {
	line, kind, ok := formatModLine(ml, dbMode)
	if !ok {
		return "", false
	}
	switch kind {
	case ModLineDisabled:
		return colorDisabled + line, true
	case ModLineUnsupported:
		return colorUnsupported + line + notSupportedTooltipText, true
	}
	return modLineKindColor[kind] + line, true
}

var (
	sourceUniqueRe   = regexp.MustCompile(`unique\{([^}]+)\}`)
	sourceNormalRe   = regexp.MustCompile(`normal\{([^}]+)\}`)
	sourceCurrencyRe = regexp.MustCompile(`currency\{([^}]+)\}`)
	sourceProphecyRe = regexp.MustCompile(`prophecy\{([^}]+)\}`)
)

//pob:src/Classes/ItemsTab.lua:4220-4225
func FormatItemSource(text string) string {
	text = sourceUniqueRe.ReplaceAllString(text, colorUnique+"${1}"+colorSource)
	text = sourceNormalRe.ReplaceAllString(text, colorNormal+"${1}"+colorSource)
	text = sourceCurrencyRe.ReplaceAllString(text, colorCurrency+"${1}"+colorSource)
	return sourceProphecyRe.ReplaceAllString(text, colorProphecy+"${1}"+colorSource)
}

//pob:src/Classes/ItemsTab.lua:4368-4747
//nolint:gocyclo // Mirrors the Lua branch chain.
func AddItemTooltip(item *Item, ctx ItemTooltipContext) ItemTooltip {
	t := ItemTooltip{}
	if item.Base == nil || item.BaseName == "" {
		name := item.Title
		if name == "" {
			name = item.Name
		}
		if name == "" {
			name = "Unknown Item"
		}
		t.Title = []string{name}
		t.addSeparator()
		t.addLine(colorNegative + "Item base is not supported by the current version.")
		return t
	}
	t.Influence = itemTooltipInfluence(item)
	baseName := parenSuffixRe.ReplaceAllString(item.BaseName, "")
	if item.Title != "" {
		t.Title = []string{item.Title, baseName}
	} else {
		t.Title = []string{item.NamePrefix + baseName + item.NameSuffix}
	}
	for _, info := range influenceInfoAll {
		if info.on(item.Influence) && !ctx.ShowFlavourText {
			t.addLine(info.color + info.display + " Item")
		}
	}
	if item.Fractured && !ctx.ShowFlavourText {
		t.addLine(colorFractured + "Fractured Item")
	}
	if item.Synthesised && !ctx.ShowFlavourText {
		t.addLine(colorCrafted + "Synthesised Item")
	}
	t.addSeparator()

	//pob:block:src/Classes/ItemsTab.lua:4406-4444
	if ctx.DBMode {
		if item.UsesVariantGroups {
			if len(item.VersionList) > 0 && item.SelectedVersion != nil {
				t.addLine("^xFFFF30Version: " + item.VersionList[*item.SelectedVersion-1])
			}
			if selected := item.selectedGroupVariants(); len(selected) > 0 {
				t.addLine("^xFFFF30Variants: " + strings.Join(selected, ", "))
			}
		} else if len(item.VariantList) > 0 {
			if len(item.VariantList) == 1 {
				t.addLine("^xFFFF30Variant: " + item.VariantList[0])
			} else if item.Variant != nil && *item.Variant >= 1 && *item.Variant <= len(item.VariantList) {
				t.addLine(fmt.Sprintf("^xFFFF30Variant: %s (%d variants)", item.VariantList[*item.Variant-1], len(item.VariantList)))
			}
		}
		if item.League != "" {
			t.addLine("^xFF5555Exclusive to: " + item.League)
		}
		if item.Unreleased {
			t.addLine(colorNegative + "Not yet available")
		}
		if item.Source != "" {
			t.addLine(colorSource + "Source: " + FormatItemSource(item.Source))
		}
		for _, path := range item.UpgradePaths {
			t.addLine(colorSource + FormatItemSource(path))
		}
		t.addSeparator()
	}

	base := item.Base
	quality := 0
	if item.Quality != nil {
		quality = *item.Quality
	}
	switch {
	case base.Weapon != nil:
		wd := item.WeaponData[1]
		label := base.Type
		if info := data.WeaponTypes[data.ItemClassName(base.Type)]; info != nil && info.Label != "" {
			label = info.Label
		} else if base.SubType != "" {
			label = base.SubType
		}
		t.addLine(fmt.Sprintf("^x7F7F7F%s", label))
		if quality > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FQuality: "+colorMagic+"+%d%%", quality))
		}
		if wd == nil {
			break
		}
		totalDamageTypes := 0
		if _, ok := wd["PhysicalDPS"]; ok {
			t.addLine(fmt.Sprintf("^x7F7F7FPhysical Damage: "+colorMagic+"%d-%d (%s DPS)", int(wd["PhysicalMin"]), int(wd["PhysicalMax"]), luaFixed(wd["PhysicalDPS"], 1)))
			totalDamageTypes++
		}
		if _, ok := wd["ElementalDPS"]; ok {
			elemLine := ""
			for _, v := range []struct{ name, color string }{{"Fire", colorFire}, {"Cold", colorCold}, {"Lightning", colorLightning}} {
				if _, ok := wd[v.name+"DPS"]; ok {
					if elemLine != "" {
						elemLine += "^x7F7F7F, "
					} else {
						elemLine = "^x7F7F7FElemental Damage: "
					}
					elemLine += fmt.Sprintf("%s%d-%d", v.color, int(wd[v.name+"Min"]), int(wd[v.name+"Max"]))
				}
			}
			t.addLine(elemLine)
			t.addLine(fmt.Sprintf("^x7F7F7FElemental DPS: "+colorMagic+"%s", luaFixed(wd["ElementalDPS"], 1)))
			totalDamageTypes++
		}
		if _, ok := wd["ChaosDPS"]; ok {
			t.addLine(fmt.Sprintf("^x7F7F7FChaos Damage: "+colorChaos+"%d-%d "+colorMagic+"(%s DPS)", int(wd["ChaosMin"]), int(wd["ChaosMax"]), luaFixed(wd["ChaosDPS"], 1)))
			totalDamageTypes++
		}
		if totalDamageTypes > 1 {
			t.addLine(fmt.Sprintf("^x7F7F7FTotal DPS: "+colorMagic+"%s", luaFixed(wd["TotalDPS"], 1)))
		}
		t.addLine(fmt.Sprintf("^x7F7F7FCritical Strike Chance: %s%s%%", statColor(wd["CritChance"], f64(base.Weapon.CritChanceBase), nil), luaFixed(wd["CritChance"], 2)))
		t.addLine(fmt.Sprintf("^x7F7F7FAttacks per Second: %s%s", statColor(wd["AttackRate"], f64(base.Weapon.AttackRateBase), nil), luaFixed(wd["AttackRate"], 2)))
		if wd["range"] < 120 {
			t.addLine(fmt.Sprintf("^x7F7F7FWeapon Range: %s%s ^x7F7F7Fmetres", statColor(wd["range"], f64(float64(base.Weapon.Range)), nil), luaFixed(wd["range"]/10, 1)))
		}
	case base.Armour != nil:
		ad := item.ArmourData
		if quality > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FQuality: "+colorMagic+"+%d%%", quality))
		}
		if base.Armour.BlockChance != 0 && ad["BlockChance"] > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FChance to Block: %s%d%%", statColor(ad["BlockChance"], f64(float64(base.Armour.BlockChance)), nil), int(ad["BlockChance"])))
		}
		if ad["Armour"] > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FArmour: %s%d", statColor(ad["Armour"], nil, nil), int(ad["Armour"])))
		}
		if ad["Evasion"] > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FEvasion Rating: %s%d", statColor(ad["Evasion"], nil, nil), int(ad["Evasion"])))
		}
		if ad["EnergyShield"] > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FEnergy Shield: %s%d", statColor(ad["EnergyShield"], nil, nil), int(ad["EnergyShield"])))
		}
		if ad["Ward"] > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FWard: %s%d", statColor(ad["Ward"], nil, nil), int(ad["Ward"])))
		}
	case base.Flask != nil:
		fd := item.FlaskData
		if quality > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FQuality: "+colorMagic+"+%d%%", quality))
		}
		_, hasLife := fd["lifeTotal"]
		_, hasMana := fd["manaTotal"]
		if hasLife {
			if fd["lifeGradual"] != 0 {
				t.addLine(fmt.Sprintf("^x7F7F7FRecovers %s%d ^x7F7F7FLife over %s%s0 ^x7F7F7FSeconds",
					statColor(fd["lifeTotal"], f64(float64(base.Flask.Life)), nil), int(fd["lifeGradual"]),
					statColor(fd["duration"], f64(base.Flask.Duration), nil), luaFixed(fd["duration"], 1)))
			}
			if fd["lifeInstant"] != 0 {
				t.addLine(fmt.Sprintf("^x7F7F7FRecovers %s%d ^x7F7F7FLife instantly", statColor(fd["lifeTotal"], f64(float64(base.Flask.Life)), nil), int(fd["lifeInstant"])))
			}
		}
		if hasMana {
			if fd["manaGradual"] != 0 {
				t.addLine(fmt.Sprintf("^x7F7F7FRecovers %s%d ^x7F7F7FMana over %s%s0 ^x7F7F7FSeconds",
					statColor(fd["manaTotal"], f64(float64(base.Flask.Mana)), nil), int(fd["manaGradual"]),
					statColor(fd["duration"], f64(base.Flask.Duration), nil), luaFixed(fd["duration"], 1)))
			}
			if fd["manaInstant"] != 0 {
				t.addLine(fmt.Sprintf("^x7F7F7FRecovers %s%d ^x7F7F7FMana instantly", statColor(fd["manaTotal"], f64(float64(base.Flask.Mana)), nil), int(fd["manaInstant"])))
			}
		}
		if !hasLife && !hasMana {
			t.addLine(fmt.Sprintf("^x7F7F7FLasts %s%s ^x7F7F7FSeconds", statColor(fd["duration"], f64(base.Flask.Duration), nil), luaFixed(fd["duration"], 2)))
		}
		t.addLine(fmt.Sprintf("^x7F7F7FConsumes %s%d ^x7F7F7Fof %s%d ^x7F7F7FCharges on use",
			statColor(fd["chargesUsed"], f64(float64(base.Flask.ChargesUsed)), nil), int(fd["chargesUsed"]),
			statColor(fd["chargesMax"], f64(float64(base.Flask.ChargesMax)), nil), int(fd["chargesMax"])))
		for _, line := range base.Flask.Buff {
			if ParseMod(line, false).Extra != "" {
				t.addLine(colorUnsupported + line + notSupportedTooltipText)
			} else {
				t.addLine(colorMagic + line)
			}
		}
	case base.Tincture != nil:
		td := item.TinctureData
		tb := tinctureBaseOf(base.Name)
		if quality > 0 {
			t.addLine(fmt.Sprintf("^x7F7F7FQuality: "+colorMagic+"+%d%%", quality))
		}
		t.addLine(fmt.Sprintf("^x7F7F7FInflicts Mana Burn every %s%s ^x7F7F7FSeconds", statColor(td["manaBurn"], f64(tb.ManaBurn), nil), luaFixed(td["manaBurn"], 2)))
		t.addLine(fmt.Sprintf("^x7F7F7F%s%s ^x7F7F7FSecond Cooldown When Deactivated", statColor(td["cooldown"], f64(tb.Cooldown), nil), luaFixed(td["cooldown"], 2)))
	case item.Type == "Jewel":
		if item.Limit != nil {
			t.addLine(fmt.Sprintf("^x7F7F7FLimited to: ^7%d", *item.Limit))
		}
		if item.ClassRestriction != "" {
			code := colorNegative
			if ctx.ClassName == item.ClassRestriction {
				code = colorPositive
			}
			t.addLine("^x7F7F7FRequires Class " + code + item.ClassRestriction)
		}
		if item.JewelRadiusLabel != "" {
			t.addLine("^x7F7F7FRadius: ^7" + item.JewelRadiusLabel)
		}
		//pob:block:src/Classes/ItemsTab.lua:4580-4592
		if ctx.RadiusData != nil {
			line := ""
			for i, stat := range []string{"Str", "Dex", "Int"} {
				if v := ctx.RadiusData[stat]; v != 0 {
					if line != "" {
						line += ", "
					}
					line += fmt.Sprintf("%s%d %s^7", []string{colorStrength, colorDexterity, colorIntelligence}[i], int(v), stat)
				}
			}
			if line != "" {
				t.addLine("^x7F7F7FAttributes in Radius: " + line)
			}
		}
	}
	if item.Catalyst != nil && *item.Catalyst > 0 && *item.Catalyst <= len(catalystQualityFormat) && item.CatalystQuality != nil && *item.CatalystQuality > 0 {
		t.addLine(fmt.Sprintf(catalystQualityFormat[*item.Catalyst-1], *item.CatalystQuality))
		t.addSeparator()
	}
	if len(item.Sockets) > 0 {
		t.addLine("^x7F7F7FSockets: " + socketDescriptionLine(item))
	}
	if item.TalismanTier != nil {
		t.addLine(fmt.Sprintf("^x7F7F7FTalisman Tier ^xFFFFFF%d", *item.TalismanTier))
		t.addSeparator()
	}
	if item.MemoryStrands != nil {
		if ctx.ShowFlavourText {
			t.addLine("")
		}
		t.addLine(colorMemory + fmt.Sprintf("Memory Strands: ^7%d", *item.MemoryStrands))
		if ctx.ShowFlavourText {
			t.addLine("")
		}
	}
	if item.Intangibility != nil {
		if ctx.ShowFlavourText {
			t.addLine("")
		}
		t.addLine(colorIntangible + fmt.Sprintf("Intangibility: ^7%d%%", *item.Intangibility))
		if ctx.ShowFlavourText {
			t.addLine("")
		}
	}
	t.addSeparator()

	addRequirementsToTooltip(&t, ctx, item.Requirements["level"], item.Requirements["strMod"], item.Requirements["dexMod"], item.Requirements["intMod"],
		item.Requirements["str"], item.Requirements["dex"], item.Requirements["int"])

	for _, modList := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines, item.CrucibleModLines} {
		if len(modList) == 0 {
			continue
		}
		for _, ml := range modList {
			if !item.ItemModLineVisible(ml) {
				continue
			}
			if line, ok := formattedModLine(ml, ctx.DBMode); ok {
				t.addLine(line)
			}
		}
		t.addSeparator()
	}

	if cj := readClusterJewelData(item); cj != nil {
		t.addSeparator()
		nodeMap := clusterNodeMap(data.LatestTreeVersion)
		addNode := func(name string) {
			node, ok := nodeMap[name]
			if !ok {
				return
			}
			dn := ""
			if node.Name != nil {
				dn = *node.Name
			}
			t.addLine(colorMagic + dn)
			for _, stat := range node.Stats {
				t.addLine("^x7F7F7F" + stat)
			}
		}
		if len(cj.notables) > 0 {
			for _, name := range cj.notables {
				addNode(name)
			}
		} else if cj.keystone != "" {
			addNode(cj.keystone)
		}
		t.addSeparator()
	}

	if item.Corrupted || item.Split || item.Mirrored {
		if len(item.ExplicitModLines) == 0 {
			t.addSeparator()
		}
		if item.Split {
			t.addLine(colorNegative + "Split")
		}
		if item.Mirrored {
			t.addLine(colorNegative + "Mirrored")
		}
		if item.Corrupted {
			t.addLine(colorNegative + "Corrupted")
		}
		t.addSeparator()
	}

	baseFlavour, hasBaseFlavour := data.BaseFlavourText(item.Base.Name)
	if (item.Rarity == "UNIQUE" || item.Rarity == "RELIC" || hasBaseFlavour) && ctx.ShowFlavourText {
		var flavour []string
		if hasBaseFlavour {
			flavour = baseFlavour
		} else if rows := data.UniqueFlavourText(foulbornPrefixRe.ReplaceAllString(item.Title, "")); len(rows) > 0 {
			if strings.Contains(item.Title, "Grand Spectrum") {
				if id := grandSpectrumFlavourID(t.Lines); id != "" {
					for _, r := range rows {
						if r.ID == id {
							flavour = r.Text
						}
					}
				}
			} else {
				flavour = rows[0].Text
			}
		}
		if len(flavour) > 0 {
			for _, line := range flavour {
				t.addLine(colorUnique + line)
			}
			t.addSeparator()
		}
	}
	//pob:block:src/Classes/ItemsTab.lua:4756-4769
	for _, g := range item.GrantedSkills {
		t.Skills = append(t.Skills, TooltipSkill{SkillID: g.SkillID, Level: g.Level})
	}
	for _, m := range item.modListForSlot(1) {
		if m.Name() == "ExtraSupport" {
			if v, ok := m.Value().List().(mod.ExtraSupport); ok {
				t.Skills = append(t.Skills, TooltipSkill{SkillID: v.SkillID, Level: v.Level})
			}
		}
	}
	if len(t.Skills) > 0 {
		plural := ""
		if len(t.Skills) > 1 {
			plural = "s"
		}
		t.addSeparator()
		t.addLine(colorTip + "Tip: Hold Shift to display a tooltip for the granted skill" + plural + ".")
	}
	return t
}

//pob:block:src/Modules/Data.lua:1062-1065
var gemForSkill = sync.OnceValue(func() map[string]string {
	out := map[string]string{}
	for _, id := range slices.Sorted(maps.Keys(raw2.Gems)) {
		if eff := raw2.Gems[id].GrantedEffectID; eff != "" {
			if _, ok := out[eff]; !ok {
				out[eff] = id
			}
		}
	}
	return out
})

//pob:block:src/Classes/ItemsTab.lua:4769-4793
func GrantedSkillTooltips(t ItemTooltip, ctx ItemTooltipContext) [][]TooltipLine {
	out := make([][]TooltipLine, len(t.Skills))
	for k, s := range t.Skills {
		level := s.Level
		if level == 0 {
			level = 1
		}
		child := ItemTooltip{}
		if gemID, ok := gemForSkill()[s.SkillID]; ok {
			AddGemTooltip(&child, ctx, pob.Gem{GemID: gemID, SkillID: s.SkillID, Level: level, Enabled: true}, nil, false)
		} else {
			AddSkillTooltip(&child, ctx, s.SkillID, level)
		}
		out[k] = child.Lines
	}
	return out
}

var foulbornPrefixRe = regexp.MustCompile(`^Foulborn\s+`)

//pob:block:src/Classes/ItemsTab.lua:4696-4728
func grandSpectrumFlavourID(lines []TooltipLine) string {
	for _, l := range lines {
		switch text := l.Text; {
		case strings.Contains(text, "Power"):
			return "UniqueJewel170"
		case strings.Contains(text, "Endurance"):
			return "UniqueJewel168"
		case strings.Contains(text, "Frenzy"):
			return "UniqueJewel169"
		case strings.Contains(text, "Elemental Damage"):
			return "UniqueJewel168"
		case strings.Contains(text, "Elemental Ailments"):
			return "UniqueJewel166"
		case strings.Contains(text, "Elemental Resistances") || strings.Contains(text, "Armour"):
			return "UniqueJewel76"
		case strings.Contains(text, "Maximum Life"):
			return "UniqueJewel165"
		case strings.Contains(text, "Critical Strike Multiplier"):
			return "UniqueJewel167"
		case strings.Contains(text, "Critical Strike Chance") || strings.Contains(text, "Mana"):
			return "UniqueJewel75"
		}
	}
	return ""
}

//pob:block:src/Classes/ItemsTab.lua:4803-5001
//nolint:gocyclo // Mirrors the Lua branch chain.
func (m *MiscCalculator) EffectiveFlaskStats(item *Item) []string {
	if item == nil || item.Base == nil || item.Base.Flask == nil || m.base == nil || m.base.ModDB == nil {
		return nil
	}
	var stats []string
	add := func(format string, args ...any) { stats = append(stats, fmt.Sprintf(format, args...)) }
	fd := item.FlaskData
	fb := item.Base.Flask
	modDB := m.base.ModDB
	output := m.base.Player.Output
	durInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskDuration")
	effectInc := modDB.Sum(mod.TypeIncrease, playerActorCfg, "FlaskEffect")
	lifeDur, manaDur := 0.0, 0.0

	if item.Rarity == "MAGIC" && fb.Life == 0 && fb.Mana == 0 {
		effectInc += modDB.Sum(mod.TypeIncrease, playerActorCfg, "MagicUtilityFlaskEffect")
	}

	if fb.Life != 0 || fb.Mana != 0 {
		rateInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskRecoveryRate")
		if fb.Life != 0 {
			lifeInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskLifeRecovery")
			lifeMore := modDB.More(nil, "FlaskLifeRecovery")
			lifeRateInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskLifeRecoveryRate")
			instantPerc := fd["instantPerc"] + modDB.Sum(mod.TypeBase, nil, "LifeFlaskInstantRecovery")
			lifeMoreOnLowLife := modDB.More(nil, "FlaskLifeRecoveryLowLife")
			lowLifeMulti := 1.0
			if lifeMoreOnLowLife > 1 {
				lowLifeMulti = ((lifeMoreOnLowLife - 1) / (1 + effectInc/100)) + 1
			}
			inst := float64(fd["lifeBase"] * instantPerc / 100 * (1 + lifeInc/100) * lifeMore * (1 + effectInc/100) * lowLifeMulti)
			base := fd["lifeBase"] * (1 - instantPerc/100) * (1 + lifeInc/100) * lifeMore * (1 + effectInc/100) * (1 + durInc/100) * lowLifeMulti
			grad := float64(base * output["LifeRecoveryRateMod"])
			esGrad := float64(base * output["EnergyShieldRecoveryRateMod"])
			lifeDur = fd["duration"] * (1 + durInc/100) / (1 + rateInc/100) / (1 + lifeRateInc/100)

			if fd["lifeAdditional"] > 0 && !configInputBool(m.build, "conditionFullLife") {
				totalAdditionalAmount := float64((fd["lifeAdditional"] / 100) * fd["lifeTotal"] * output["LifeRecoveryRateMod"])
				additionalGrad := float64((lifeDur / 10) * totalAdditionalAmount)
				leftoverDur := 10 - lifeDur
				leftoverAmount := totalAdditionalAmount - additionalGrad
				switch {
				case inst > 0 && grad > 0:
					add("^8Life recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %ss^8, and an additional ^7%d ^8over subsequent ^7%ss^8)",
						int(inst+grad+totalAdditionalAmount), int(inst), int(grad+additionalGrad), luaFixed(lifeDur, 2), int(leftoverAmount), luaFixed(leftoverDur, 2))
				case inst > 0:
					lifeDur = 0
					add("^8Life recovered: ^7%d ^8(^7%d^8 instantly, and an additional ^7%d ^8over ^7%ss^8)",
						int(inst+totalAdditionalAmount), int(inst), int(totalAdditionalAmount), luaFixed(10, 2))
				default:
					add("^8Life recovered: ^7%d ^8(^7%d ^8over ^7%ss^8, and an additional ^7%d ^8over subsequent ^7%ss^8)",
						int(grad+totalAdditionalAmount), int(grad+additionalGrad), luaFixed(lifeDur, 2), int(leftoverAmount), luaFixed(leftoverDur, 2))
				}
			} else {
				if inst > 0 && grad > 0 {
					add("^8Life recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %ss^8)", int(inst+grad), int(inst), int(grad), luaFixed(lifeDur, 2))
				} else if inst+grad != fd["lifeTotal"] || (inst == 0 && lifeDur != fd["duration"]) {
					if inst > 0 {
						lifeDur = 0
						add("^8Life recovered: ^7%d ^8instantly", int(inst))
					} else if grad > 0 {
						add("^8Life recovered: ^7%d ^8over ^7%ss", int(grad), luaFixed(lifeDur, 2))
					}
				}
			}
			if modDB.Flag(nil, "LifeFlaskAppliesToEnergyShield") {
				switch {
				case inst > 0 && esGrad > 0:
					add("^8Energy Shield recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %ss^8)", int(inst+esGrad), int(inst), int(esGrad), luaFixed(lifeDur, 2))
				case inst > 0 && esGrad == 0:
					add("^8Energy Shield recovered: ^7%d ^8instantly", int(inst))
				case inst == 0 && esGrad > 0:
					add("^8Energy Shield recovered: ^7%d ^8over ^7%ss", int(esGrad), luaFixed(lifeDur, 2))
				}
			}
		}
		if fb.Mana != 0 {
			manaInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskManaRecovery")
			manaRateInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskManaRecoveryRate")
			instantPerc := fd["instantPerc"] + modDB.Sum(mod.TypeBase, nil, "ManaFlaskInstantRecovery")
			inst := float64(fd["manaBase"] * instantPerc / 100 * (1 + manaInc/100) * (1 + effectInc/100))
			base := fd["manaBase"] * (1 - instantPerc/100) * (1 + manaInc/100) * (1 + effectInc/100) * (1 + durInc/100)
			grad := float64(base * output["ManaRecoveryRateMod"])
			lifeGrad := base * output["LifeRecoveryRateMod"]
			manaDur = fd["duration"] * (1 + durInc/100) / (1 + rateInc/100) / (1 + manaRateInc/100)
			if inst > 0 && grad > 0 {
				add("^8Mana recovered: ^7%d ^8(^7%d^8 instantly, plus ^7%d ^8over^7 %ss^8)", int(inst+grad), int(inst), int(grad), luaFixed(manaDur, 2))
			} else if inst+grad != fd["manaTotal"] || (inst == 0 && manaDur != fd["duration"]) {
				if inst > 0 {
					manaDur = 0
					add("^8Mana recovered: ^7%d ^8instantly", int(inst))
				} else if grad > 0 {
					add("^8Mana recovered: ^7%d ^8over ^7%ss", int(grad), luaFixed(manaDur, 2))
				}
			}
			if modDB.Flag(nil, "ManaFlaskAppliesToLife") && lifeGrad > 0 {
				add("^8Life recovered: ^7%d ^8over ^7%ss", int(lifeGrad), luaFixed(manaDur, 2))
			}
		}
	} else if durInc != 0 {
		add("^8Flask effect duration: ^7%s0s", luaFixed(fd["duration"]*(1+durInc/100), 1))
	}
	if effectMod := 1 + (fd["effectInc"]+effectInc)/100; effectMod != 1 {
		add("^8Flask effect modifier: ^7%+d%%", int(float64(effectMod*100)-100))
	}
	usedInc := modDB.Sum(mod.TypeIncrease, nil, "FlaskChargesUsed")
	if usedInc != 0 {
		used := math.Floor(fd["chargesUsed"] * (1 + usedInc/100))
		add("^8Charges used: ^7%d ^8of ^7%d ^8(^7%d ^8uses)", int(used), int(fd["chargesMax"]), int(math.Floor(fd["chargesMax"]/used)))
	}
	gainMod := fd["gainMod"] * (1 + modDB.Sum(mod.TypeIncrease, nil, "FlaskChargesGained")/100)
	if gainMod != 1 {
		add("^8Charge gain modifier: ^7%+d%%", int(float64(gainMod*100)-100))
	}

	chargesGenerated := modDB.Sum(mod.TypeBase, nil, "FlaskChargesGenerated")
	if fb.Life != 0 {
		chargesGenerated += modDB.Sum(mod.TypeBase, nil, "LifeFlaskChargesGenerated")
	}
	if fb.Mana != 0 {
		chargesGenerated += modDB.Sum(mod.TypeBase, nil, "ManaFlaskChargesGenerated")
	}
	if fb.Mana == 0 && fb.Life == 0 {
		chargesGenerated += modDB.Sum(mod.TypeBase, nil, "UtilityFlaskChargesGenerated")
	}
	chargesGeneratedOnWardBreak := 0.0
	if item.BaseName == "Iron Flask" {
		chargesGeneratedOnWardBreak += modDB.Sum(mod.TypeBase, nil, "IronFlaskChargesGeneratedOnWardBreak")
	}
	chargesGeneratedPerFlask := float64(modDB.Sum(mod.TypeBase, nil, "FlaskChargesGeneratedPerEmptyFlask") * float64(emptyFlaskSlots(m.build)))
	chargesGenerated = float64(chargesGenerated * gainMod)
	chargesGeneratedOnWardBreak *= gainMod
	chargesGeneratedPerFlask = float64(chargesGeneratedPerFlask * gainMod)
	if total := chargesGenerated + chargesGeneratedPerFlask; total > 0 {
		add("^8Charges generated: ^7%s^8 per second", luaFixed(total, 2))
	}
	if chargesGeneratedOnWardBreak > 0 {
		add("^8Charges generated on Ward Break: ^7%s", luaFixed(chargesGeneratedOnWardBreak, 2))
	}
	chanceToNotConsumeCharges := math.Min(modDB.Sum(mod.TypeBase, nil, "FlaskChanceNotConsumeCharges"), 100)
	if chanceToNotConsumeCharges != 0 {
		add("^8Chance to not consume charges: ^7%d%%", int(chanceToNotConsumeCharges))
	}

	hasUptime := fb.Life == 0 && fb.Mana == 0
	flaskDuration := fd["duration"] * (1 + durInc/100)
	if fb.Life != 0 && (fd["lifeEffectNotRemoved"] != 0 || modDB.Flag(nil, "LifeFlaskEffectNotRemoved")) {
		hasUptime = true
		flaskDuration = lifeDur
	} else if fb.Mana != 0 && (fd["manaEffectNotRemoved"] != 0 || modDB.Flag(nil, "ManaFlaskEffectNotRemoved")) {
		hasUptime = true
		flaskDuration = manaDur
	}
	if hasUptime {
		flaskChargesUsed := fd["chargesUsed"] * (1 + usedInc/100)
		if flaskChargesUsed > 0 && flaskDuration > 0 {
			per3Duration := flaskDuration - luaMod(flaskDuration, 3)
			per5Duration := flaskDuration - luaMod(flaskDuration, 5)
			minimumChargesGenerated := float64(per3Duration*chargesGenerated) + float64(per5Duration*chargesGeneratedPerFlask)
			percentageMin := math.Min(minimumChargesGenerated/flaskChargesUsed*100, 100)
			if percentageMin < 100 && chanceToNotConsumeCharges < 100 {
				averageChargesGenerated := (chargesGenerated + chargesGeneratedPerFlask) * flaskDuration
				averageChargesUsed := flaskChargesUsed * (100 - chanceToNotConsumeCharges) / 100
				percentageAvg := math.Min(averageChargesGenerated/averageChargesUsed*100, 100)
				add("^8Flask uptime: ^7%d%%^8 average, ^7%d%%^8 minimum", int(percentageAvg), int(percentageMin))
			} else {
				add("^8Flask uptime: ^7100%%^8")
			}
		}
	}
	if len(stats) == 0 {
		return nil
	}
	return append([]string{"^7Effective flask stats:"}, stats...)
}

func luaMod(a, b float64) float64 {
	return a - float64(math.Floor(a/b)*b)
}

func configInputBool(build *pob.PathOfBuilding, name string) bool {
	if build == nil {
		return false
	}
	for _, input := range build.Config.Inputs {
		if input.Name == name && input.Boolean != nil {
			return *input.Boolean
		}
	}
	return false
}

//pob:block:src/Classes/ItemsTab.lua:4943-4948
func emptyFlaskSlots(build *pob.PathOfBuilding) int {
	if build == nil {
		return 0
	}
	filled := map[string]bool{}
	for _, set := range build.Items.ItemSets {
		if set.ID != strconv.Itoa(build.Items.ActiveItemSet) {
			continue
		}
		for _, s := range set.Slots {
			if strings.HasPrefix(s.Name, "Flask") && s.ItemID != 0 {
				filled[s.Name] = true
			}
		}
	}
	empty := 0
	for i := 1; i <= 5; i++ {
		if !filled["Flask "+strconv.Itoa(i)] {
			empty++
		}
	}
	return empty
}

//pob:block:src/Classes/ItemsTab.lua:5011-5038
func (m *MiscCalculator) EffectiveTinctureStats(item *Item) []string {
	if item == nil || item.Base == nil || item.Base.Tincture == nil || m.base == nil || m.base.ModDB == nil {
		return nil
	}
	var stats []string
	td := item.TinctureData
	modDB := m.base.ModDB
	effectInc := modDB.Sum(mod.TypeIncrease, playerActorCfg, "TinctureEffect")
	if item.Rarity == "MAGIC" {
		effectInc += modDB.Sum(mod.TypeIncrease, playerActorCfg, "MagicTinctureEffect")
	}
	quality := 0.0
	if item.Quality != nil {
		quality = float64(*item.Quality)
	}
	effectMod := (1 + (td["effectInc"]+effectInc)/100) * (1 + quality/100)
	if effectMod != 1 {
		stats = append(stats, fmt.Sprintf("^8Tincture effect modifier: ^7%+d%%", int(float64(effectMod*100)-100)))
	}
	burn := 1 / (td["manaBurn"] / (1 + modDB.Sum(mod.TypeIncrease, playerActorCfg, "TinctureManaBurnRate")/100) / (1 + modDB.Sum(mod.TypeMore, playerActorCfg, "TinctureManaBurnRate")/100))
	stats = append(stats, "^8Mana Burn Inflicted Every Second: ^7"+luaFixed(burn, 2))
	if notBurn := math.Min(modDB.Sum(mod.TypeBase, nil, "TincturesNotInflictManaBurn"), 100); notBurn != 0 {
		stats = append(stats, fmt.Sprintf("^8Chance to not inflict Mana Burn: ^7%d%%", int(notBurn)))
	}
	cooldown := tinctureBaseOf(item.Base.Name).Cooldown / (1 + (modDB.Sum(mod.TypeIncrease, playerActorCfg, "TinctureCooldownRecovery")+td["cooldownInc"])/100)
	stats = append(stats, "^8Tincture Cooldown when deactivated: ^7"+luaFixed(cooldown, 2)+"^8 seconds")
	return append([]string{"^7Effective tincture stats:"}, stats...)
}

func (m *MiscCalculator) SocketJewelTooltip(nodeID int, ctx ItemTooltipContext) (ItemTooltip, bool) {
	text, ok := buildItemText(m.build, m.build.SocketedJewel(nodeID))
	if !ok {
		return ItemTooltip{}, false
	}
	if m.base != nil {
		ctx.RadiusData = m.base.JewelRadiusData[strconv.Itoa(nodeID)]
	}
	return AddItemTooltip(parseItemGlobal(text), ctx), true
}

//pob:src/Classes/ItemsTab.lua:2398-2406
func checkLineForAllocates(line string, nodes map[string]data.Node) string {
	if nodes != nil && strings.Contains(line, "Allocates") {
		if id := allocatesIDRe.FindString(line); id != "" {
			if n, ok := nodes[id]; ok && n.Name != nil {
				return "Allocates " + *n.Name
			}
		}
	}
	return line
}

var allocatesIDRe = regexp.MustCompile(`\d+`)

//pob:src/Classes/ItemsTab.lua:2408-2422
func (m *MiscCalculator) CompareAddedMod(text string, lines []string, modType string) (*Environment, *Environment) {
	item := ParseItem(text)
	if item.Base == nil {
		return nil, nil
	}
	slotName := item.getPrimarySlot()
	nodes := data.TreeVersions[data.LatestTreeVersion].Tree().Nodes
	added := strings.TrimRight(text, "\r\n")
	for _, line := range lines {
		prefix := ""
		if modType != "" {
			prefix = "{" + modType + "}"
		}
		added += "\n" + prefix + checkLineForAllocates(line, nodes)
	}
	return m.Calc(&CalcOverride{RepSlotName: slotName, RepItem: item}), m.Calc(&CalcOverride{RepSlotName: slotName, RepItem: ParseItem(added)})
}

func (m *MiscCalculator) Build() *pob.PathOfBuilding {
	return m.build
}
