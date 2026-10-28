package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
)

// The row structs name go-pob-data's columns; where PoB-PoE2's spec.lua names the same bytes otherwise,
// the PoB name is in the comment.

type poe2BaseItemType struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	InheritsFrom string `json:"InheritsFrom"` // BaseType
	DropLevel    int    `json:"DropLevel"`
	ImplicitMods []int  `json:"Implicit_Mods"`
	Tags         []int  `json:"Tags"`
}

type poe2Tag struct {
	ID string `json:"Id"`
}

type poe2ModType struct {
	Name string `json:"Name"` // Id
}

type poe2ItemMod struct {
	ID            string `json:"Id"`
	ModType       *int   `json:"ModType"` // Type
	Level         int    `json:"Level"`
	Stat1         *int   `json:"Stat1"`
	Stat2         *int   `json:"Stat2"`
	Stat3         *int   `json:"Stat3"`
	Stat4         *int   `json:"Stat4"`
	Stat5         *int   `json:"Stat5"`
	Stat6         *int   `json:"Stat6"`
	Stat1ValueMin int    `json:"Stat1ValueMin"`
	Stat1ValueMax int    `json:"Stat1ValueMax"`
	Stat2ValueMin int    `json:"Stat2ValueMin"`
	Stat2ValueMax int    `json:"Stat2ValueMax"`
	Stat3ValueMin int    `json:"Stat3ValueMin"`
	Stat3ValueMax int    `json:"Stat3ValueMax"`
	Stat4ValueMin int    `json:"Stat4ValueMin"`
	Stat4ValueMax int    `json:"Stat4ValueMax"`
	Stat5ValueMin int    `json:"Stat5ValueMin"`
	Stat5ValueMax int    `json:"Stat5ValueMax"`
	Stat6ValueMin int    `json:"Stat6ValueMin"`
	Stat6ValueMax int    `json:"Stat6ValueMax"`
	ImplicitTags  []int  `json:"ImplicitTags"`
	// IsUnscalable: spec.lua's byte 21 from the row's end, which go-pob-data's schema calls IsEssenceOnlyModifier
	IsUnscalable bool `json:"IsEssenceOnlyModifier"`
}

type poe2ItemModStat struct {
	stat     *int
	min, max int
}

func (m *poe2ItemMod) stats() [6]poe2ItemModStat {
	return [6]poe2ItemModStat{
		{m.Stat1, m.Stat1ValueMin, m.Stat1ValueMax}, {m.Stat2, m.Stat2ValueMin, m.Stat2ValueMax},
		{m.Stat3, m.Stat3ValueMin, m.Stat3ValueMax}, {m.Stat4, m.Stat4ValueMin, m.Stat4ValueMax},
		{m.Stat5, m.Stat5ValueMin, m.Stat5ValueMax}, {m.Stat6, m.Stat6ValueMin, m.Stat6ValueMax},
	}
}

type poe2ItemSpirit struct {
	BaseItemType int `json:"BaseItemType"`
	Value        int `json:"SpiritGranted"`
}

type poe2ItemInherentSkill struct {
	BaseItemType  int   `json:"BaseItemType"`
	Skill         []int `json:"SkillsGranted"`
	NoReservation bool  `json:"IsWeapon"`
}

type poe2SkillGem struct {
	BaseItemType int `json:"BaseItemType"`
	// IsSupport: spec.lua reads a Bool (byte == 1) at the first byte of the i32 go-pob-data's schema calls GemType
	GemType             int   `json:"GemType"`
	GemLevelProgression *int  `json:"ItemExperienceType"`
	GemEffects          []int `json:"GemEffects"`
}

type poe2GemEffect struct {
	GrantedEffect *int `json:"GrantedEffect"`
}

type poe2GrantedEffect struct {
	ActiveSkill *int `json:"ActiveSkill"`
}

type poe2ActiveSkill struct {
	DisplayName string `json:"DisplayedName"`
}

type poe2ItemExperiencePerLevel struct {
	ItemExperienceType *int `json:"ItemExperienceType"`
}

type poe2WeaponType struct {
	BaseItemType int `json:"BaseItemType"`
	CritChance   int `json:"CritChance"`
	Speed        int `json:"Speed"`
	DamageMin    int `json:"DamageMin"`
	DamageMax    int `json:"DamageMax"`
	Range        int `json:"RangeMax"`
	ReloadTime   int `json:"ReloadTime"`
}

type poe2ArmourType struct {
	BaseItemType    int `json:"BaseItemType"`
	Armour          int `json:"Armour"`
	Evasion         int `json:"Evasion"`
	EnergyShield    int `json:"EnergyShield"`
	Ward            int `json:"Ward"`
	MovementPenalty int `json:"IncreasedMovementSpeed"`
}

type poe2ShieldType struct {
	BaseItemType int `json:"BaseItemType"`
	Block        int `json:"Block"`
}

type poe2Flask struct {
	BaseItemType int   `json:"BaseItemType"`
	LifePerUse   int   `json:"LifePerUse"`
	ManaPerUse   int   `json:"ManaPerUse"`
	RecoveryTime int   `json:"RecoveryTime"`
	UtilityBuffs []int `json:"UtilityBuff"`
}

type poe2UtilityFlaskBuff struct {
	BuffDefinition int   `json:"BuffDefinition"`
	StatValues     []int `json:"StatValues"`
}

type poe2BuffDefinition struct {
	GrantedFlags []int `json:"GrantedFlags"`
	GrantedStats []int `json:"GrantedStats"`
}

type poe2ComponentCharges struct {
	BaseItemType string `json:"BaseItemTypesKey"`
	Max          int    `json:"MaxCharges"`
	PerUse       int    `json:"PerCharge"`
}

type poe2AttributeRequirement struct {
	BaseType int `json:"BaseItemType"`
	ReqStr   int `json:"ReqStr"`
	ReqInt   int `json:"ReqInt"`
	ReqDex   int `json:"ReqDex"`
}

type poe2BasesExport struct {
	describer       *poe2Describer
	baseItemTypes   []poe2BaseItemType
	tags            []poe2Tag
	mods            []poe2ItemMod
	modTypes        []poe2ModType
	stats           []poe2Stat
	itemSpirit      []poe2ItemSpirit
	inherentSkills  []poe2ItemInherentSkill
	skillGems       []poe2SkillGem
	gemEffects      []poe2GemEffect
	grantedEffects  []poe2GrantedEffect
	activeSkills    []poe2ActiveSkill
	itemExperience  []poe2ItemExperiencePerLevel
	weaponTypes     []poe2WeaponType
	armourTypes     []poe2ArmourType
	shieldTypes     []poe2ShieldType
	flasks          []poe2Flask
	utilityBuffs    []poe2UtilityFlaskBuff
	buffDefinitions []poe2BuffDefinition
	componentCharge []poe2ComponentCharges
	attributeReqs   []poe2AttributeRequirement
	itFiles         map[string]*string
}

type poe2BaseState struct {
	typ, subType         string
	forceShow, forceHide bool
	socketLimit          *float64
}

//pob2:block:src/Export/Scripts/bases.lua:490-519
var poe2BaseItemTypeNames = []string{"axe", "bow", "claw", "crossbow", "dagger", "fishing", "flail", "mace", "sceptre", "spear",
	"staff", "sword", "wand", "helmet", "body", "focus", "gloves", "boots", "shield", "quiver", "traptool", "amulet", "ring",
	"belt", "jewel", "flask", "talisman", "incursionlimb"}

var (
	itMaxQualityRe = regexp.MustCompile(`max_quality = (.+)`)
)

func poe2TableInto[T any](dst *[]T, name string, err *error) {
	if *err == nil {
		*dst, *err = poe2Table[T](name)
	}
}

func poe2LoadTables(x *poe2BasesExport) error {
	var err error
	poe2TableInto(&x.baseItemTypes, "BaseItemTypes", &err)
	poe2TableInto(&x.tags, "Tags", &err)
	poe2TableInto(&x.mods, "Mods", &err)
	poe2TableInto(&x.modTypes, "ModType", &err)
	poe2TableInto(&x.stats, "Stats", &err)
	poe2TableInto(&x.itemSpirit, "ItemSpirit", &err)
	poe2TableInto(&x.inherentSkills, "ItemInherentSkills", &err)
	poe2TableInto(&x.skillGems, "SkillGems", &err)
	poe2TableInto(&x.gemEffects, "GemEffects", &err)
	poe2TableInto(&x.grantedEffects, "GrantedEffects", &err)
	poe2TableInto(&x.activeSkills, "ActiveSkills", &err)
	poe2TableInto(&x.itemExperience, "ItemExperiencePerLevel", &err)
	poe2TableInto(&x.weaponTypes, "WeaponTypes", &err)
	poe2TableInto(&x.armourTypes, "ArmourTypes", &err)
	poe2TableInto(&x.shieldTypes, "ShieldTypes", &err)
	poe2TableInto(&x.flasks, "Flasks", &err)
	poe2TableInto(&x.utilityBuffs, "UtilityFlaskBuffs", &err)
	poe2TableInto(&x.buffDefinitions, "BuffDefinitions", &err)
	poe2TableInto(&x.componentCharge, "ComponentCharges", &err)
	poe2TableInto(&x.attributeReqs, "AttributeRequirements", &err)
	return err
}

// poe2RowBy is DatFile:GetRow over a key column: the first row whose key is the given row index.
func poe2RowBy[T any](rows []T, key func(*T) int, idx int) *T {
	for i := range rows {
		if key(&rows[i]) == idx {
			return &rows[i]
		}
	}
	return nil
}

func (x *poe2BasesExport) itFile(name string) *string {
	if f, ok := x.itFiles[name]; ok {
		return f
	}
	var out *string
	if b, err := loader.LoadFile(gameDataContext(), poe2Version, name+".it", assetCache()); err == nil {
		s := string(b)
		out = &s
	}
	x.itFiles[name] = out
	return out
}

func itLines(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return r == '\r' || r == '\n' })
}

//pob2:block:src/Export/Scripts/bases.lua:41-65
func (x *poe2BasesExport) baseItemTags(baseItemType string) []string {
	if baseItemType == "nothing" { // base case
		return []string{}
	}
	file := x.itFile(baseItemType)
	if file == nil {
		return nil
	}
	tags := []string{}
	for _, line := range itLines(*file) {
		if m := itExtendsRe.FindStringSubmatch(line); m != nil {
			tags = append(tags, x.baseItemTags(m[1])...)
		} else if strings.Contains(line, "remove_tag") {
			// table.remove(tags, nil) removes the last element
			i := len(tags) - 1
			if m := itRemoveTagRe.FindStringSubmatch(line); m != nil {
				for j, t := range tags {
					if t == m[1] {
						i = j
						break
					}
				}
			}
			if i >= 0 {
				tags = append(tags[:i], tags[i+1:]...)
			}
		} else if strings.Contains(line, "tag") {
			if m := itTagRe.FindStringSubmatch(line); m != nil {
				tags = append(tags, m[1])
			}
		}
	}
	return tags
}

// maximumQuality's string is the .it file's own text; "" stands for Lua's 0 and nil for nil.
//
//pob2:block:src/Export/Scripts/bases.lua:67-84
func (x *poe2BasesExport) maximumQuality(baseItemType string) *string {
	zero := ""
	if baseItemType == "nothing" { // base case
		return &zero
	}
	file := x.itFile(baseItemType)
	if file == nil {
		return nil
	}
	var superClassQuality *string
	for _, line := range itLines(*file) {
		if m := itExtendsRe.FindStringSubmatch(line); m != nil {
			superClassQuality = x.maximumQuality(m[1])
		} else if strings.Contains(line, "max_quality") {
			if m := itMaxQualityRe.FindStringSubmatch(line); m != nil {
				return &m[1]
			}
			return nil
		}
	}
	return superClassQuality
}

//pob2:src/Export/statdesc.lua:420-433
func (x *poe2BasesExport) describeMod(mod *poe2ItemMod) ([]string, []float64, string, error) {
	stats := map[string]*edRange{}
	for _, s := range mod.stats() {
		if s.stat != nil {
			stats[x.stats[*s.stat].ID] = &edRange{min: float64(s.min), max: float64(s.max)}
		}
	}
	out, orders, err := x.describer.describeStats(stats)
	if err != nil {
		return nil, nil, "", fmt.Errorf("%s: %w", mod.ID, err)
	}
	return out, orders, x.describeModTags(mod.ImplicitTags), nil
}

//pob2:src/Export/statdesc.lua:140-155
func (x *poe2BasesExport) describeModTags(modTags []int) string {
	var modTagsText string
	for _, t := range modTags {
		if modTagsText != "" {
			modTagsText += ", "
		}
		modTagsText += `"` + x.tags[t].ID + `"`
	}
	return modTagsText
}

func luaFloor(v float64) string { return luaNumStr(math.Floor(v)) }

//pob2:src/Modules/Common.lua:722-728
func luaRoundTo(val float64, dec int) float64 {
	p := math.Pow(10, float64(dec))
	return math.Floor(val*p+0.5) / p
}

//pob2:block:src/Export/Scripts/bases.lua:31-399
func (x *poe2BasesExport) base(st *poe2BaseState, args string, out *strings.Builder) error {
	w := func(parts ...string) {
		for _, p := range parts {
			_, _ = out.WriteString(p)
		}
	}
	baseTypeID, displayName := args, ""
	if m := baseDirectiveRe.FindStringSubmatch(args); m != nil {
		baseTypeID, displayName = m[1], m[2]
	}
	baseIdx := -1
	for i := range x.baseItemTypes {
		if x.baseItemTypes[i].ID == baseTypeID {
			baseIdx = i
			break
		}
	}
	if baseIdx < 0 {
		fmt.Fprintf(os.Stderr, "Invalid Id %s\n", baseTypeID)
		return nil
	}
	baseItemType := &x.baseItemTypes[baseIdx]
	baseItemTags := x.baseItemTags(baseItemType.InheritsFrom)
	maximumQuality := x.maximumQuality(baseItemType.InheritsFrom)
	if displayName == "" {
		displayName = baseItemType.Name
	}
	displayName = strings.ReplaceAll(displayName, "ö", "o")
	displayName = strings.Trim(displayName, " \t\n\v\f\r") // trim spaces GGG might leave in by accident
	if displayName == "Energy Blade" {
		if st.typ == "One Hand Sword" {
			displayName = "Energy Blade One Handed"
		} else {
			displayName = "Energy Blade Two Handed"
		}
	}
	if strings.Contains(displayName, "DNT") {
		return nil
	}
	hidden := (st.forceHide && !strings.Contains(baseTypeID, "Talisman") && !st.forceShow) || strings.Contains(baseTypeID, "Unique") || strings.Contains(displayName, "Runemastered")
	w(`itemBases["`, displayName, "\"] = {\n")
	w("\ttype = \"", st.typ, "\",\n")
	if st.subType != "" {
		w("\tsubType = \"", st.subType, "\",\n")
	}
	if maximumQuality == nil {
		return fmt.Errorf("%s: no .it file gives max_quality for %s", baseTypeID, baseItemType.InheritsFrom)
	}
	if *maximumQuality != "" { //If it crashes on this line, you are missing a .It file for a base ConPrintf(baseItemType.BaseType)
		w("\tquality = ", *maximumQuality, ",\n")
	}
	if st.typ == "Belt" {
		w("\tcharmLimit = 0,\n")
	}
	if itemSpirit := poe2RowBy(x.itemSpirit, func(r *poe2ItemSpirit) int { return r.BaseItemType }, baseIdx); itemSpirit != nil {
		w("\tspirit = ", luaNumStr(float64(itemSpirit.Value)), ",\n")
	}
	if hidden {
		w("\thidden = true,\n")
	}
	if st.socketLimit != nil {
		w("\tsocketLimit = ", luaNumStr(*st.socketLimit), ",\n")
	}
	w("\ttags = { ")
	combinedTags := map[string]bool{}
	for _, tag := range baseItemTags {
		combinedTags[tag] = true
	}
	for _, tag := range baseItemType.Tags {
		combinedTags[x.tags[tag].ID] = true
	}
	// Genesis Tree tags are granted at runtime and omitted from BaseItemTypes,
	// so the genesis_tree_* mods (belt/ring caster and minion affixes) match nothing.
	// Re-add them; each mod's own belt/ring exclusion routes it to the correct slot.
	if combinedTags["belt"] || combinedTags["ring"] {
		combinedTags["genesis_tree_minion"] = true
		combinedTags["genesis_tree_caster"] = true
	}
	combinedTagList := make([]string, 0, len(combinedTags))
	for tag := range combinedTags {
		combinedTagList = append(combinedTagList, tag)
	}
	sort.Strings(combinedTagList)
	for _, tag := range combinedTagList {
		w(tag, " = true, ")
	}
	w("},\n")

	var implicitLines, implicitModTypes, variantList []string
	type described struct {
		mod    *poe2ItemMod
		lines  []string
		orders []float64
		tags   string
	}
	var implicitMods []described
	hasCharmSlots := false
	addMod := func(mod *poe2ItemMod) error {
		lines, orders, tags, err := x.describeMod(mod)
		if err != nil {
			return err
		}
		implicitMods = append(implicitMods, described{mod, lines, orders, tags})
		return nil
	}
	for _, m := range baseItemType.ImplicitMods {
		mod := &x.mods[m]
		if err := addMod(mod); err != nil {
			return err
		}
		if mod.ModType != nil && x.modTypes[*mod.ModType].Name == "CharmSlots" {
			hasCharmSlots = true
		}
	}
	if st.typ == "Belt" && !hasCharmSlots {
		for i := range x.mods {
			if x.mods[i].ID == "BeltImplicitCharmSlots3" {
				if err := addMod(&x.mods[i]); err != nil {
					return err
				}
				break
			}
		}
	}
	firstOrder := func(d described) float64 {
		if len(d.orders) == 0 {
			return 0
		}
		return d.orders[0]
	}
	sort.SliceStable(implicitMods, func(i, j int) bool { return firstOrder(implicitMods[i]) < firstOrder(implicitMods[j]) })
	for _, d := range implicitMods {
		for _, line := range d.lines {
			prefix := ""
			if d.mod.IsUnscalable {
				prefix = "{unscalable}"
			}
			implicitLines = append(implicitLines, prefix+line)
			implicitModTypes = append(implicitModTypes, d.tags)
		}
		if d.mod.ID == "SpearImplicitDisplaySpearThrow1" {
			implicitLines = append(implicitLines, "Grants Skill: Spear Throw")
		}
	}
	inherentSkillType := poe2RowBy(x.inherentSkills, func(r *poe2ItemInherentSkill) int { return r.BaseItemType }, baseIdx)
	if inherentSkillType != nil {
		if inherentSkillType.NoReservation {
			w("\tgrantedSkillsHaveNoReservation = true,\n")
		}
		hasVariants := len(inherentSkillType.Skill) > 1
		for index, skill := range inherentSkillType.Skill {
			skillGem := poe2RowBy(x.skillGems, func(r *poe2SkillGem) int { return r.BaseItemType }, x.skillGems[skill].BaseItemType)
			if len(skillGem.GemEffects) == 0 || x.gemEffects[skillGem.GemEffects[0]].GrantedEffect == nil {
				return fmt.Errorf("%s: inherent skill %d has no granted effect", baseTypeID, skill)
			}
			grantedEffect := x.grantedEffects[*x.gemEffects[skillGem.GemEffects[0]].GrantedEffect]
			if grantedEffect.ActiveSkill == nil {
				return fmt.Errorf("%s: inherent skill %d has no active skill", baseTypeID, skill)
			}
			skillName := x.activeSkills[*grantedEffect.ActiveSkill].DisplayName
			naturalMaxLevel := 1
			if skillGem.GemType&0xff != 1 {
				naturalMaxLevel = 0
				for _, row := range x.itemExperience {
					if (row.ItemExperienceType == nil) == (skillGem.GemLevelProgression == nil) &&
						(row.ItemExperienceType == nil || *row.ItemExperienceType == *skillGem.GemLevelProgression) {
						naturalMaxLevel++
					}
				}
			}
			if naturalMaxLevel <= 0 {
				naturalMaxLevel = 1
			}
			implicitLine := "Grants Skill: "
			if naturalMaxLevel != 1 {
				implicitLine += "Level (1-" + luaNumStr(float64(naturalMaxLevel)) + ") "
			}
			implicitLine += skillName
			if hasVariants {
				variantList = append(variantList, skillName)
				implicitLine = "{variant:" + luaNumStr(float64(index+1)) + "}" + implicitLine
			}
			implicitLines = append(implicitLines, implicitLine)
		}
	}
	if len(variantList) > 0 {
		w("\tvariantList = { ")
		for _, variant := range variantList {
			w(`"`, strings.ReplaceAll(variant, `"`, `\"`), `", `)
		}
		w("},\n")
	}
	if len(implicitLines) > 0 {
		w("\timplicit = \"", strings.Join(implicitLines, `\n`), "\",\n")
	}
	w("\timplicitModTypes = { ")
	for _, t := range implicitModTypes {
		w("{ ", t, " }, ")
	}
	w("},\n")

	weaponType := poe2RowBy(x.weaponTypes, func(r *poe2WeaponType) int { return r.BaseItemType }, baseIdx)
	if weaponType != nil {
		w("\tweapon = { ")
		modConversionMap := map[string]string{
			"local_weapon_implicit_hidden_%_base_damage_is_fire":      "Fire",
			"local_weapon_implicit_hidden_%_base_damage_is_cold":      "Cold",
			"local_weapon_implicit_hidden_%_base_damage_is_lightning": "Lightning",
			"local_weapon_implicit_hidden_%_base_damage_is_chaos":     "Chaos",
		}
		modAddedMinMap := map[string]string{
			"local_weapon_implicit_hidden_added_minimum_fire_damage":      "Fire",
			"local_weapon_implicit_hidden_added_minimum_cold_damage":      "Cold",
			"local_weapon_implicit_hidden_added_minimum_lightning_damage": "Lightning",
			"local_weapon_implicit_hidden_added_minimum_chaos_damage":     "Chaos",
		}
		modAddedMaxMap := map[string]string{
			"local_weapon_implicit_hidden_added_maximum_fire_damage":      "Fire",
			"local_weapon_implicit_hidden_added_maximum_cold_damage":      "Cold",
			"local_weapon_implicit_hidden_added_maximum_lightning_damage": "Lightning",
			"local_weapon_implicit_hidden_added_maximum_chaos_damage":     "Chaos",
		}
		conversion := map[string]float64{"Physical": 100}
		addedMin := map[string]float64{}
		addedMax := map[string]float64{}
		total := 0.0
		for _, m := range baseItemType.ImplicitMods {
			for _, s := range x.mods[m].stats() {
				if s.stat == nil {
					continue
				}
				id := x.stats[*s.stat].ID
				if dmgType, ok := modConversionMap[id]; ok {
					conversion[dmgType] += float64(s.min)
					total += float64(s.min)
				}
				if t, ok := modAddedMinMap[id]; ok {
					addedMin[t] += float64(s.min)
				}
				if t, ok := modAddedMaxMap[id]; ok {
					addedMax[t] += float64(s.min)
				}
			}
		}
		factor := 1.0
		if total > 100 {
			factor = 100 / total
		}
		for _, typ := range []string{"Physical", "Fire", "Cold", "Lightning", "Chaos"} {
			if typ == "Physical" {
				conversion[typ] = 1 - math.Min(total/100, 1)
			} else {
				conversion[typ] = conversion[typ] * factor / 100
			}
			if conversion[typ] != 0 {
				w(typ, "Min = ", luaFloor(float64(weaponType.DamageMin)*conversion[typ]), ", ", typ, "Max = ", luaFloor(float64(weaponType.DamageMax)*conversion[typ]), ", ")
			}
			if addedMin[typ] != 0 {
				w(typ, "Min = ", luaNumStr(addedMin[typ]), ", ")
			}
			if addedMax[typ] != 0 {
				w(typ, "Max = ", luaNumStr(addedMax[typ]), ", ")
			}
		}
		w("CritChanceBase = ", luaNumStr(float64(weaponType.CritChance)/100), ", ")
		w("AttackRateBase = ", luaNumStr(luaRoundTo(1000/float64(weaponType.Speed), 2)), ", ")
		w("Range = ", luaNumStr(float64(weaponType.Range)), ", ")
		if weaponType.ReloadTime > 0 {
			w("ReloadTimeBase = ", luaNumStr(luaRoundTo(float64(weaponType.ReloadTime)/1000, 2)), ", ")
		}
		w("},\n")
	}
	armourType := poe2RowBy(x.armourTypes, func(r *poe2ArmourType) int { return r.BaseItemType }, baseIdx)
	if armourType != nil {
		w("\tarmour = { ")
		if shield := poe2RowBy(x.shieldTypes, func(r *poe2ShieldType) int { return r.BaseItemType }, baseIdx); shield != nil {
			w("BlockChance = ", luaNumStr(float64(shield.Block)), ", ")
		}
		if armourType.Armour > 0 {
			w("Armour = ", luaNumStr(float64(armourType.Armour)), ", ")
		}
		if armourType.Evasion > 0 {
			w("Evasion = ", luaNumStr(float64(armourType.Evasion)), ", ")
		}
		if armourType.EnergyShield > 0 {
			w("EnergyShield = ", luaNumStr(float64(armourType.EnergyShield)), ", ")
		}
		if armourType.Ward > 0 {
			w("Ward = ", luaNumStr(float64(armourType.Ward)), ", ")
		}
		if armourType.MovementPenalty != 0 {
			w("MovementPenalty = ", luaNumStr(-float64(armourType.MovementPenalty)/10000), ", ")
		}
		w("},\n")
	}
	if st.typ == "Flask" || st.typ == "Charm" {
		if flask := poe2RowBy(x.flasks, func(r *poe2Flask) int { return r.BaseItemType }, baseIdx); flask != nil {
			var compCharges *poe2ComponentCharges
			for i := range x.componentCharge {
				if x.componentCharge[i].BaseItemType == baseItemType.ID {
					compCharges = &x.componentCharge[i]
					break
				}
			}
			if compCharges == nil {
				return fmt.Errorf("%s has no ComponentCharges row", baseTypeID)
			}
			if st.typ == "Charm" {
				w("\tcharm = { ")
			} else {
				w("\tflask = { ")
			}
			if flask.LifePerUse > 0 {
				w("life = ", luaNumStr(float64(flask.LifePerUse)), ", ")
			}
			if flask.ManaPerUse > 0 {
				w("mana = ", luaNumStr(float64(flask.ManaPerUse)), ", ")
			}
			w("duration = ", luaNumStr(float64(flask.RecoveryTime)/10), ", ")
			w("chargesUsed = ", luaNumStr(float64(compCharges.PerUse)), ", ")
			w("chargesMax = ", luaNumStr(float64(compCharges.Max)), ", ")
			if len(flask.UtilityBuffs) > 0 {
				stats := map[string]*edRange{}
				for _, b := range flask.UtilityBuffs {
					buff := x.utilityBuffs[b]
					def := x.buffDefinitions[buff.BuffDefinition]
					for i, stat := range def.GrantedStats {
						if i >= len(buff.StatValues) {
							return fmt.Errorf("%s: utility flask buff %d has no value for stat %s", baseTypeID, b, x.stats[stat].ID)
						}
						stats[x.stats[stat].ID] = &edRange{min: float64(buff.StatValues[i]), max: float64(buff.StatValues[i])}
					}
					for _, stat := range def.GrantedFlags {
						stats[x.stats[stat].ID] = &edRange{min: 1, max: 1}
					}
				}
				lines, _, err := x.describer.describeStats(stats)
				if err != nil {
					return err
				}
				w("buff = { \"", strings.Join(lines, "\", \""), "\" }, ")
			}
			w("},\n")
		}
	}
	w("\treq = { ")
	reqLevel := 1
	compAtt := poe2RowBy(x.attributeReqs, func(r *poe2AttributeRequirement) int { return r.BaseType }, baseIdx)
	if weaponType != nil || armourType != nil || inherentSkillType != nil || compAtt != nil {
		if baseItemType.DropLevel > 4 {
			reqLevel = baseItemType.DropLevel
		}
	}
	if st.typ == "Flask" || st.typ == "Charm" {
		if baseItemType.DropLevel > 2 {
			reqLevel = baseItemType.DropLevel
		}
	}
	for _, m := range baseItemType.ImplicitMods {
		reqLevel = max(reqLevel, int(math.Floor(float64(x.mods[m].Level)*0.8)))
	}
	if reqLevel > 1 {
		w("level = ", luaNumStr(float64(reqLevel)), ", ")
	}
	if compAtt != nil {
		if compAtt.ReqStr > 0 {
			w("str = ", luaNumStr(float64(compAtt.ReqStr)), ", ")
		}
		if compAtt.ReqDex > 0 {
			w("dex = ", luaNumStr(float64(compAtt.ReqDex)), ", ")
		}
		if compAtt.ReqInt > 0 {
			w("int = ", luaNumStr(float64(compAtt.ReqInt)), ", ")
		}
	}
	w("},\n}\n")
	return nil
}

//pob2:block:src/Export/Scripts/bases.lua:401-417
func (x *poe2BasesExport) baseMatch(st *poe2BaseState, argstr string, out *strings.Builder) error {
	args := strings.Fields(argstr)
	if len(args) == 0 {
		return nil
	}
	// Default to look at the Id column for matching
	key, value := "Id", args[0]
	// If column name is specified, use that
	if len(args) > 1 {
		key, value = args[0], args[1]
	}
	re, err := luaPatternRe(value)
	if err != nil {
		return err
	}
	for _, t := range x.baseItemTypes {
		col := t.ID
		switch key {
		case "Id":
		case "BaseType":
			col = t.InheritsFrom
		default:
			return fmt.Errorf("baseMatch column %s is not modelled", key)
		}
		if re.MatchString(col) {
			if err := x.base(st, t.ID, out); err != nil {
				return err
			}
		}
	}
	return nil
}

//pob2:block:src/Export/Main.lua:112-135
func (x *poe2BasesExport) processTemplateFile(pobDir, name string) (string, error) {
	text, err := os.ReadFile(filepath.Join(pobDir, "src", "Export", "Bases", name+".txt")) //nolint:gosec // the pinned PoB-PoE2 checkout
	if err != nil {
		return "", fmt.Errorf("template %s: %w", name, err)
	}
	var out strings.Builder
	out.WriteString("-- This file is automatically generated, do not edit!\n")
	st := &poe2BaseState{}
	lines := strings.Split(string(text), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		m := templateLineRe.FindStringSubmatch(line)
		if m == nil {
			switch {
			case strings.HasPrefix(line, "--") || strings.HasPrefix(line, "local") || line == "":
				out.WriteString(line + "\n")
			case strings.HasPrefix(line, "legacy"):
				out.WriteString("\t" + line + ",\n")
			default:
				out.WriteString("\t\t\t" + line + "\n")
			}
			continue
		}
		args := m[2]
		switch m[1] {
		case "type":
			st.typ = args
		case "subType":
			st.subType = args
		case "forceShow":
			st.forceShow = args == "true"
		case "forceHide":
			st.forceHide = args == "true"
		case "socketLimit":
			if v, ok := luaToNumber(strings.TrimSpace(args)); ok {
				st.socketLimit = &v
			} else {
				st.socketLimit = nil
			}
		case "base":
			err = x.base(st, args, &out)
		case "baseMatch":
			err = x.baseMatch(st, args, &out)
		case "baseGroup", "setBestBase", "setBase":
			// Rares.txt's directives; bases.lua:526 runs them over Bases/Rares.txt only
		default:
			fmt.Fprintf(os.Stderr, "Unknown directive '%s'\n", m[1])
		}
		if err != nil {
			return "", err
		}
	}
	return out.String(), nil
}

func newPoE2BasesExport() (*poe2BasesExport, error) {
	x := &poe2BasesExport{itFiles: map[string]*string{}}
	if err := poe2LoadTables(x); err != nil {
		return nil, err
	}
	var err error
	if x.describer, err = poe2LoadStatFile("stat_descriptions"); err != nil {
		return nil, err
	}
	return x, nil
}

// poe2ItemBaseFiles is bases.lua's Data/Bases/<type>.lua for every template, keyed by that path.
//
//pob2:block:src/Export/Scripts/bases.lua:520-522
func poe2ItemBaseFiles(pobDir string) (map[string]string, error) {
	x, err := newPoE2BasesExport()
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	for _, name := range poe2BaseItemTypeNames {
		text, err := x.processTemplateFile(pobDir, name)
		if err != nil {
			return nil, err
		}
		files["Data/Bases/"+name+".lua"] = text
	}
	return files, nil
}

func dumpPoE2ItemBases() (any, error) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		return nil, fmt.Errorf("POB_DIR is not set; the base templates are the pinned PoB-PoE2's Export/Bases")
	}
	return poe2ItemBaseFiles(pobDir)
}
