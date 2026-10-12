package calculator

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

type partyBuff struct {
	modList          *moddb.ModList
	effectMult       float64
	isMark           bool
	warcryPowerBonus map[mod.Mod]float64
}

type partyMod struct {
	mod.Mod
	warcryPowerBonus    float64
	warcryPowerBonusSet bool
}

type partyBuffGroup struct {
	buffs map[string]*partyBuff
	vaal  map[string]*partyBuff
}

type partyBuffList struct {
	byType   map[string]*partyBuffGroup
	limit    float64
	limitSet bool
}

type partyActor struct {
	Aura         *partyBuffList
	Curse        *partyBuffList
	Warcry       *partyBuffList
	Link         *partyBuffList
	modDB        *moddb.ModDB
	output       map[string]float64
	outputTables map[string]map[string]float64
}

type partyTab struct {
	actor        *partyActor
	enemyModList *moddb.ModList
	skipped      []string
}

func newPartyBuffList() *partyBuffList {
	return &partyBuffList{byType: make(map[string]*partyBuffGroup)}
}

//pob:block:src/Classes/PartyTab.lua:18-20
func newPartyTab() *partyTab {
	return &partyTab{
		actor: &partyActor{
			Aura:         newPartyBuffList(),
			Curse:        newPartyBuffList(),
			Warcry:       newPartyBuffList(),
			Link:         newPartyBuffList(),
			modDB:        moddb.NewModDB(),
			output:       make(map[string]float64),
			outputTables: make(map[string]map[string]float64),
		},
		enemyModList: moddb.NewModList(),
	}
}

func luaXMLContent(s string) string {
	return strings.Trim(s, " \t\n\v\f\r")
}

//pob:src/Classes/PartyTab.lua:511-537
func loadPartyTab(party pob.Party) *partyTab {
	t := newPartyTab()
	for _, node := range party.Applied() {
		buf := luaXMLContent(node.Text)
		switch node.Name {
		case "PartyMemberStats":
			t.parseBuffs(nil, buf, "PartyMemberStats")
		case "Aura":
			t.parseBuffs(t.actor.Aura, buf, "Aura")
		case "Curse":
			t.parseBuffs(t.actor.Curse, buf, "Curse")
		case "Warcry Skills":
			t.parseBuffs(t.actor.Warcry, buf, "Warcry")
		case "Link Skills":
			t.parseBuffs(t.actor.Link, buf, "Link")
		case "EnemyConditions":
			t.parseBuffs(nil, buf, "EnemyConditions")
		case "EnemyMods":
			t.parseBuffs(nil, buf, "EnemyMods")
		}
	}
	return t
}

func (t *partyTab) skip(section string, err error) {
	t.skipped = append(t.skipped, section+": "+err.Error())
}

func luaLines(buf string) []string {
	return strings.Split(buf, "\n")
}

var (
	partyStatsNestedRe = regexp.MustCompile(`^([A-Za-z0-9 ]*?[A-Za-z0-9]+)\.([A-Za-z0-9 ]*?[A-Za-z0-9]+)=(.+)$`)
	partyStatsTaggedRe = regexp.MustCompile(`([A-Za-z0-9 ]*?[A-Za-z0-9]+)\|(.+)=(.+)`)
	partyStatsPlainRe  = regexp.MustCompile(`([A-Za-z0-9 ]*?[A-Za-z0-9]+)=(.+)`)
	partyOtherEffectRe = regexp.MustCompile(`([A-Za-z0-9 ']*?[A-Za-z0-9]+)\|([A-Za-z0-9]+)\|(.+)`)
	partyItemSourceRe  = regexp.MustCompile(`Item:(\d+):(.+)`)
	partySpaceLowerRe  = regexp.MustCompile(` [a-z]`)
)

//pob:src/Classes/PartyTab.lua:713-891
func (t *partyTab) parseBuffs(list *partyBuffList, buf string, buffType string) {
	switch buffType {
	case "EnemyConditions":
		for _, line := range luaLines(buf) {
			if line != "" {
				t.enemyModList.AddMod(mod.NewFlag(strings.ReplaceAll(line, "Condition:", "Condition:Party:"), true).Source("Party"))
			}
		}
	case "EnemyMods":
		for _, line := range luaLines(buf) {
			if !strings.Contains(line, "|") {
				continue
			}
			pm, err := parseFormattedSourceMod(line)
			if err != nil {
				t.skip(buffType, err)
				continue
			}
			if pm != nil {
				pm.Source("Party" + pm.GetSource())
				t.enemyModList.AddMod(pm.Mod)
			}
		}
	case "PartyMemberStats":
		output := t.actor.output
		for _, line := range luaLines(buf) {
			switch {
			case strings.Contains(line, "="):
				if g := partyStatsNestedRe.FindStringSubmatch(line); g != nil {
					if t.actor.outputTables[g[1]] == nil {
						t.actor.outputTables[g[1]] = make(map[string]float64)
					}
					v, _ := luaToNumber(g[3])
					t.actor.outputTables[g[1]][g[2]] = v
				} else if strings.Contains(line, "|") {
					g := partyStatsTaggedRe.FindStringSubmatch(line)
					if g == nil {
						t.skip(buffType, fmt.Errorf("PartyMemberStats line %q does not match k|tags=v", line))
						continue
					}
					v, _ := luaToNumber(g[3])
					for _, tag := range strings.Split(g[2], "|") {
						switch tag {
						case "percent":
							v /= 100
						case "max":
							prev, ok := output[g[1]]
							if !ok {
								prev = 1
							}
							v = max(v, prev)
						}
					}
					output[g[1]] = v
				} else if g := partyStatsPlainRe.FindStringSubmatch(line); g != nil {
					v, _ := luaToNumber(g[2])
					output[g[1]] = v
				}
			case line != "":
				t.actor.modDB.AddMod(mod.NewFlag(line, true).Source("Party"))
			}
		}
	default:
		t.parseNamedBuffs(list, buf, buffType)
	}
}

//pob:block:src/Classes/PartyTab.lua:788-891
func (t *partyTab) parseNamedBuffs(list *partyBuffList, buf string, buffType string) {
	mode := "Name"
	if buffType == "Curse" {
		mode = "CurseLimit"
	}
	var currentName string
	var currentEffect float64
	var isMark bool
	currentModType := "Unknown"
	switch buffType {
	case "Link":
		currentModType = "Link"
	case "Warcry":
		currentModType = "Warcry"
	}
	for _, line := range luaLines(buf) {
		switch {
		case line != "---" && strings.Contains(line, "---"):
		case mode == "CurseLimit" && line != "":
			list.limit, list.limitSet = luaToNumber(line)
			mode = "Name"
		case mode == "Name" && line != "":
			currentName = strings.ReplaceAll(line, "_Debuff", "")
			currentEffect = 0
			if line == "extraAura" || line == "otherEffects" {
				currentModType = line
				mode = "Stats"
			} else {
				mode = "Effect"
				currentModType = "Unknown"
			}
		case mode == "Effect":
			currentEffect, _ = luaToNumber(line)
			if buffType == "Curse" {
				mode = "isMark"
			} else {
				mode = "Stats"
			}
		case mode == "isMark":
			isMark = line == "true"
			mode = "Stats"
		case line == "---":
			mode = "Name"
		default:
			if !strings.Contains(line, "|") || currentName == "SKIP" || strings.Contains(line, "MinionModifier|LIST") {
				continue
			}
			if currentModType == "otherEffects" {
				g := partyOtherEffectRe.FindStringSubmatch(line)
				if g == nil {
					t.skip(buffType, fmt.Errorf("otherEffects line %q does not match name|effect|mod", line))
					continue
				}
				currentName, line = g[1], g[3]
				currentEffect, _ = luaToNumber(g[2])
			}
			pm, err := parseFormattedSourceMod(line)
			if err != nil {
				t.skip(buffType, err)
				continue
			}
			if pm == nil {
				continue
			}
			m := pm.Mod
			for _, tag := range m.Tags() {
				if ge, ok := tag.(*mod.GlobalEffectTag); ok && currentModType != "Link" && currentModType != "Warcry" && currentModType != "otherEffects" {
					currentModType = ge.GlobalEffectList[0]
				}
			}
			group := list.byType[currentModType]
			if group == nil {
				group = &partyBuffGroup{buffs: make(map[string]*partyBuff)}
				list.byType[currentModType] = group
			}
			listElement := group.buffs
			if strings.HasPrefix(currentName, "Vaal") {
				if group.vaal == nil {
					group.vaal = make(map[string]*partyBuff)
				}
				listElement = group.vaal
			}
			if existing, ok := listElement[currentName]; !ok {
				listElement[currentName] = &partyBuff{modList: moddb.NewModList(), effectMult: currentEffect, isMark: isMark}
			} else if existing.effectMult != currentEffect {
				if existing.effectMult < currentEffect {
					listElement[currentName] = &partyBuff{modList: moddb.NewModList(), effectMult: currentEffect}
				} else {
					currentName = "SKIP"
				}
			}
			if currentName == "SKIP" {
				continue
			}
			source := string(m.GetSource())
			if strings.Contains(source, "Item") {
				g := partyItemSourceRe.FindStringSubmatch(source)
				if g == nil {
					t.skip(buffType, fmt.Errorf("item source %q does not match Item:<n>:<name>", source))
					continue
				}
				source = "Party - " + g[2]
			}
			if strings.Contains(source, "Skill") {
				if idx := strings.Index(source, "Skill:"); idx >= 0 {
					skillID := source[idx+len("Skill:"):]
					if _, ok := SkillBaseFlagsFor(skillID); !ok {
						minimisedName := strings.ReplaceAll(partySpaceLowerRe.ReplaceAllStringFunc(currentName, strings.ToUpper), " ", "")
						if _, ok := SkillBaseFlagsFor(minimisedName); ok {
							source = "Skill:" + minimisedName
						} else {
							source = skillID
						}
					}
				}
			}
			m.Source(mod.Source(source))
			if buffType == "Link" {
				m.SetName(strings.ReplaceAll(m.Name(), "Parent", "PartyMember"))
				for _, tag := range m.Tags() {
					if mt, ok := tag.(*mod.MultiplierTag); ok && mt.TagActor == "parent" {
						mt.TagActor = "partyMembers"
					}
				}
			}
			listElement[currentName].modList.AddMod(m)
			if pm.warcryPowerBonusSet {
				if listElement[currentName].warcryPowerBonus == nil {
					listElement[currentName].warcryPowerBonus = make(map[mod.Mod]float64)
				}
				listElement[currentName].warcryPowerBonus[m] = pm.warcryPowerBonus
			}
		}
	}
}

var partyModFlagByName = map[string]mod.MFlag{
	"Attack": mod.MFlagAttack, "Spell": mod.MFlagSpell, "Hit": mod.MFlagHit, "Dot": mod.MFlagDot,
	"Cast": mod.MFlagCast, "Melee": mod.MFlagMelee, "Area": mod.MFlagArea,
	"Projectile": mod.MFlagProjectile, "SourceMask": mod.MFlagSourceMask, "Ailment": mod.MFlagAilment,
	"MeleeHit": mod.MFlagMeleeHit, "Weapon": mod.MFlagWeapon, "Axe": mod.MFlagAxe, "Bow": mod.MFlagBow,
	"Claw": mod.MFlagClaw, "Dagger": mod.MFlagDagger, "Mace": mod.MFlagMace, "Staff": mod.MFlagStaff,
	"Sword": mod.MFlagSword, "Wand": mod.MFlagWand, "Unarmed": mod.MFlagUnarmed,
	"Fishing": mod.MFlagFishing, "WeaponMelee": mod.MFlagWeaponMelee,
	"WeaponRanged": mod.MFlagWeaponRanged, "Weapon1H": mod.MFlagWeapon1H, "Weapon2H": mod.MFlagWeapon2H,
	"WeaponMask": mod.MFlag(0x2FFF0000),
}

//pob:src/Modules/ModTools.lua:78-97
func parseFormattedSourceMod(line string) (*partyMod, error) {
	modStrings := strings.Split(line, "|")
	if len(modStrings) < 4 {
		return nil, nil //nolint:nilnil // ModTools.lua:83 returns nil for a short line; callers skip it
	}
	field := func(i int) string {
		if i < len(modStrings) {
			return modStrings[i]
		}
		return ""
	}
	name, modType := modStrings[2], mod.Type(modStrings[3])
	var m mod.Mod
	if modStrings[0] == "true" {
		if modType != mod.TypeFlag {
			return nil, fmt.Errorf("mod %q: boolean value on type %s", line, modType)
		}
		m = mod.NewFlag(name, true)
	} else {
		v, _ := luaToNumber(modStrings[0])
		switch modType {
		case mod.TypeFlag:
			m = mod.NewFlag(name, true)
		case mod.TypeList:
			return nil, fmt.Errorf("mod %q: numeric value on type %s", line, modType)
		default:
			m = mod.NewFloat(name, modType, v)
		}
	}
	m.Source(mod.Source(modStrings[1]))
	m.Flag(partyModFlagByName[field(4)])
	m.KeywordFlag(mod.KeywordFlagsByName[field(5)])
	pm := &partyMod{Mod: m}
	if len(modStrings) > 6 {
		tags, bonus, err := parseTags(modStrings[6])
		if err != nil {
			return nil, fmt.Errorf("mod %q: %w", line, err)
		}
		m.Tag(tags...)
		if bonus != nil {
			pm.warcryPowerBonus, pm.warcryPowerBonusSet = *bonus, true
		}
	}
	return pm, nil
}

//pob:src/Modules/ModTools.lua:50-76
func parseTags(line string) ([]mod.Tag, *float64, error) {
	if line == "" || line == "-" {
		return nil, nil, nil
	}
	var warcryPowerBonus *float64
	var tags []mod.Tag
	for _, tagGroup := range strings.Split(line, ",") {
		if tagGroup == "" {
			continue
		}
		fields := make(map[string]string)
		var order []string
		for _, tag := range strings.Split(tagGroup, "/") {
			if tag == "" {
				continue
			}
			k, v, ok := strings.Cut(tag, "=")
			if !ok || k == "" || v == "" {
				return nil, nil, fmt.Errorf("tag invalid: %q", tag)
			}
			fields[k] = v
			order = append(order, k)
		}
		tag, bonus, err := buildPartyTag(fields, order)
		if err != nil {
			return nil, nil, fmt.Errorf("tag %q: %w", tagGroup, err)
		}
		if len(tags) == 0 {
			warcryPowerBonus = bonus
		}
		tags = append(tags, tag)
	}
	return tags, warcryPowerBonus, nil
}

func buildPartyTag(fields map[string]string, order []string) (mod.Tag, *float64, error) {
	num := func(k string) (float64, error) {
		v, ok := luaToNumber(fields[k])
		if !ok {
			return 0, fmt.Errorf("field %s=%q is not a number", k, fields[k])
		}
		return v, nil
	}
	var (
		tag              mod.Tag
		handled          func(k string) (bool, error)
		warcryPowerBonus *float64
	)
	switch fields["type"] {
	case "GlobalEffect":
		ge := mod.GlobalEffect(fields["effectType"])
		tag = ge
		handled = func(k string) (bool, error) {
			switch k {
			case "effectType":
			case "effectName":
				ge.Name(fields[k])
			case "effectCond":
				ge.EffectCond(fields[k])
			case "effectEnemyCond":
				ge.EffectEnemyCond(fields[k])
			case "modCond":
				ge.ModCond(fields[k])
			case "unscalable":
				ge.Unscalable(fields[k] == "true")
			case "div":
				v, err := num(k)
				if err != nil {
					return false, err
				}
				ge.Div(v)
			case "limit":
				v, err := num(k)
				if err != nil {
					return false, err
				}
				ge.Limit(v)
			case "warcryPowerBonus":
				v, err := num(k)
				if err != nil {
					return false, err
				}
				warcryPowerBonus = &v
			default:
				return false, nil
			}
			return true, nil
		}
	case "Condition":
		c := mod.Condition(strings.Split(fields["var"], "|")...)
		tag = c
		handled = func(k string) (bool, error) {
			switch k {
			case "var":
			case "neg":
				c.Neg(fields[k] == "true")
			default:
				return false, nil
			}
			return true, nil
		}
	case "Multiplier":
		mt := mod.Multiplier(fields["var"])
		tag = mt
		handled = func(k string) (bool, error) {
			switch k {
			case "var":
			case "actor":
				mt.Actor(fields[k])
			case "limitTotal":
				mt.LimitTotal(fields[k] == "true")
			case "div", "limit", "base":
				v, err := num(k)
				if err != nil {
					return false, err
				}
				switch k {
				case "div":
					mt.Div(v)
				case "limit":
					mt.Limit(v)
				case "base":
					mt.Base(v)
				}
			default:
				return false, nil
			}
			return true, nil
		}
	default:
		return nil, nil, fmt.Errorf("unsupported tag type %q", fields["type"])
	}
	for _, k := range order {
		if k == "type" {
			continue
		}
		ok, err := handled(k)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, fmt.Errorf("unsupported field %q on %s tag", k, fields["type"])
		}
	}
	return tag, warcryPowerBonus, nil
}

//pob:block:src/Modules/CalcPerform.lua:1275
func (env *Environment) partyMembers() *partyActor {
	if env.partyTab == nil {
		env.partyTab = newPartyTab()
	}
	return env.partyTab.actor
}

func (l *partyBuffList) strongerThan(modType string, name string, mult float64) bool {
	g := l.byType[modType]
	if g == nil {
		return false
	}
	ally := g.buffs[name]
	return ally != nil && ally.effectMult/100 > mult
}

func sortedPartyBuffNames(m map[string]*partyBuff) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

const pairsPickPartyCurses = "partyCurses"

func pinPairsOrder[T any](items []T, pick []string, name func(T) string) []T {
	if len(pick) == 0 {
		return items
	}
	out := make([]T, 0, len(items))
	used := make([]bool, len(items))
	for _, want := range pick {
		for i, it := range items {
			if !used[i] && name(it) == want {
				out = append(out, it)
				used[i] = true
				break
			}
		}
	}
	for i, it := range items {
		if !used[i] {
			out = append(out, it)
		}
	}
	return out
}

type PartySummaryEntry struct {
	Group string
	Name  string
	Value string
}

//pob:block:src/Classes/PartyTab.lua:892-975
func PartySummaries(party pob.Party) map[string][]PartySummaryEntry {
	out := map[string][]PartySummaryEntry{}
	t := newPartyTab()
	for _, node := range party.ImportedBuffs {
		buf := luaXMLContent(node.Text)
		switch node.Name {
		case "Aura":
			list := newPartyBuffList()
			t.parseBuffs(list, buf, "Aura")
			out[node.Name] = partyAuraSummary(list)
		case "Curse", "Warcry Skills", "Link Skills":
			buffType := strings.TrimSuffix(node.Name, " Skills")
			list := newPartyBuffList()
			t.parseBuffs(list, buf, buffType)
			out[node.Name] = partyEffectSummary(list.byType[buffType], false)
		case "EnemyMods":
			out[node.Name] = partyEnemyModsSummary(buf)
		}
	}
	return out
}

func partyEffectSummary(g *partyBuffGroup, vaal bool) []PartySummaryEntry {
	if g == nil {
		return nil
	}
	m := g.buffs
	if vaal {
		m = g.vaal
	}
	out := make([]PartySummaryEntry, 0, len(m))
	for name, b := range m {
		out = append(out, PartySummaryEntry{Name: name, Value: luaNumber(b.effectMult)})
	}
	sortPartySummary(out)
	return out
}

func sortPartySummary(entries []PartySummaryEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name+": "+entries[i].Value+"%\n" < entries[j].Name+": "+entries[j].Value+"%\n"
	})
}

//pob:block:src/Classes/PartyTab.lua:893-940
func partyAuraSummary(list *partyBuffList) []PartySummaryEntry {
	var out []PartySummaryEntry
	aura, debuff := list.byType["Aura"], list.byType["AuraDebuff"]
	if aura != nil {
		out = append(out, partyEffectSummary(aura, false)...)
		out = append(out, partyEffectSummary(aura, true)...)
	}
	if debuff != nil {
		for _, e := range partyEffectSummary(debuff, false) {
			if aura == nil || aura.buffs[e.Name] == nil {
				out = append(out, e)
			}
		}
		out = append(out, partyEffectSummary(debuff, true)...)
	}
	sortPartySummary(out)
	modLine := func(group string, m mod.Mod) PartySummaryEntry {
		name := m.Name()
		if m.Type() != mod.TypeFlag {
			name = string(m.Type()) + " " + name
		}
		value := "true"
		if m.Value().Type() == mod.ModValueMultiTypeFloat {
			value = luaNumber(m.Value().Float())
		} else if m.Value().Type() == mod.ModValueMultiTypeFlag && !m.Value().Flag() {
			value = "false"
		}
		return PartySummaryEntry{Group: group, Name: name, Value: value}
	}
	if g := list.byType["extraAura"]; g != nil && g.buffs["extraAura"] != nil {
		for _, m := range g.buffs["extraAura"].modList.Mods() {
			out = append(out, modLine("extraAuras", m))
		}
	}
	if g := list.byType["otherEffects"]; g != nil {
		for _, name := range sortedPartyBuffNames(g.buffs) {
			for _, m := range g.buffs[name].modList.Mods() {
				out = append(out, modLine("otherEffects", m))
			}
		}
	}
	return out
}

//pob:block:src/Classes/PartyTab.lua:720-755
func partyEnemyModsSummary(buf string) []PartySummaryEntry {
	var out []PartySummaryEntry
	index := map[string]int{}
	current := ""
	for _, line := range luaLines(buf) {
		if !strings.Contains(line, "|") {
			current = line
			if _, ok := index[current]; !ok && current != "" {
				index[current] = len(out)
				out = append(out, PartySummaryEntry{Name: current})
			}
			continue
		}
		pm, err := parseFormattedSourceMod(line)
		if err != nil || pm == nil {
			continue
		}
		i, ok := index[current]
		if !ok {
			continue
		}
		typ := string(pm.Type())
		if pm.Type() == mod.TypeFlag {
			typ = ""
		}
		out[i].Value += " True " + typ + ", "
	}
	return out
}
