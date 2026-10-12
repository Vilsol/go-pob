package calculator

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

type ModSectionData struct {
	ModName   []string
	Multi     bool
	ModType   string
	Cfg       string
	ModSource string
	Enemy     bool
	Actor     string
}

type ModSectionRow struct {
	Value      string
	Name       string
	Flags      string
	Tags       string
	Source     string
	SourceName string
	Totals     []string
	ItemName   string
	NodeID     int64
}

var (
	modNameCaseRe  = regexp.MustCompile(`([a-z0-9]:?)([A-Z])`)
	modNameDigitRe = regexp.MustCompile(`([a-z])([0-9])`)
	treeSourceRe   = regexp.MustCompile(`^Tree:(\d+)`)
	tattooSourceRe = regexp.MustCompile(`^Tree:([[:alnum:]]+)`)
	itemSourceRe   = regexp.MustCompile(`^Item:(\d+):.+`)
)

func luaNumber(v float64) string {
	return fmt.Sprintf("%.14g", v)
}

//pob:src/Classes/CalcBreakdownControl.lua:500-502
func formatModName(modName string) string {
	return modNameDigitRe.ReplaceAllString(modNameCaseRe.ReplaceAllString(modName, "$1 $2"), "$1 $2")
}

//pob:src/Classes/CalcBreakdownControl.lua:504-506
func formatVarNameOrList(varList []string) string {
	return formatModName(strings.Join(varList, " / "))
}

//pob:src/Classes/CalcBreakdownControl.lua:508-510
func formatModBase(m mod.Mod, base float64) string {
	if m.Type() == mod.TypeBase {
		return luaFormatNumber(math.Abs(base), "g", true)
	}
	return luaNumber(math.Abs(base)) + "%"
}

//pob:src/Classes/CalcBreakdownControl.lua:512-540
func breakdownModValue(value float64, modType mod.Type) string {
	switch modType {
	case mod.TypeBase:
		return luaFormatNumber(value, "g", true) + " base"
	case mod.TypeIncrease:
		if value >= 0 {
			return luaNumber(value) + "% increased"
		}
		return luaNumber(-value) + "% reduced"
	case mod.TypeMore:
		if value >= 0 {
			return luaNumber(value) + "% more"
		}
		return luaNumber(-value) + "% less"
	case mod.TypeOverride:
		return "Override: " + luaNumber(value)
	case mod.TypeFlag:
		return "True"
	case mod.TypeList:
		return "?"
	}
	return luaNumber(value)
}

//pob:block:src/Data/Global.lua:95-132
var breakdownModFlagNames = map[string]mod.MFlag{
	"Attack": mod.MFlagAttack, "Spell": mod.MFlagSpell, "Hit": mod.MFlagHit, "Dot": mod.MFlagDot, "Cast": mod.MFlagCast,
	"Melee": mod.MFlagMelee, "Area": mod.MFlagArea, "Projectile": mod.MFlagProjectile, "SourceMask": mod.MFlagSourceMask,
	"Ailment": mod.MFlagAilment, "MeleeHit": mod.MFlagMeleeHit, "Weapon": mod.MFlagWeapon, "Axe": mod.MFlagAxe,
	"Bow": mod.MFlagBow, "Claw": mod.MFlagClaw, "Dagger": mod.MFlagDagger, "Mace": mod.MFlagMace, "Staff": mod.MFlagStaff,
	"Sword": mod.MFlagSword, "Wand": mod.MFlagWand, "Unarmed": mod.MFlagUnarmed, "Fishing": mod.MFlagFishing,
	"WeaponMelee": mod.MFlagWeaponMelee, "WeaponRanged": mod.MFlagWeaponRanged, "Weapon1H": mod.MFlagWeapon1H,
	"Weapon2H": mod.MFlagWeapon2H, "WeaponMask": mod.MFlag(0x2FFF0000),
}

func modFlagText(m mod.Mod) string {
	flags, keywordFlags := int64(m.Flags()), int64(m.KeywordFlags())
	if flags == 0 && keywordFlags == 0 {
		return ""
	}
	var names []string
	add := func(bits int64, src map[string]int64) {
		for name, val := range src {
			if bits&val == val {
				names = append(names, name)
			}
		}
	}
	modFlags := map[string]int64{}
	for name, v := range breakdownModFlagNames {
		modFlags[name] = int64(v)
	}
	keywords := map[string]int64{}
	for name, v := range mod.KeywordFlagsByName {
		keywords[name] = int64(v)
	}
	if flags == keywordFlags {
		add(keywordFlags, keywords)
	} else {
		add(flags, modFlags)
		add(keywordFlags, keywords)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func modRawValue(m mod.Mod) (float64, bool) {
	if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
		return v.Float(), true
	}
	return 0, false
}

//pob:block:src/Classes/CalcBreakdownControl.lua:448
func conditionTagText(neg bool, vars []string) string {
	if len(vars) == 0 {
		return ""
	}
	return ": " + utilsTernary(neg, "Not ", "") + formatVarNameOrList(vars)
}

//nolint:gocyclo // Mirrors the Lua elseif chain.
func modTagText(m mod.Mod) string {
	tags := m.Tags()
	if len(tags) == 0 {
		return ""
	}
	baseVal := ""
	if v, ok := modRawValue(m); ok {
		baseVal = formatModBase(m, v) + " "
	}
	var out []string
	for _, tag := range tags {
		var desc string
		switch t := tag.(type) {
		case *mod.ConditionTag:
			desc = "Condition" + conditionTagText(t.Negative, t.VarList)
		case *mod.ActorConditionTag:
			actor := ""
			if t.Actor != nil && *t.Actor != "" {
				actor = strings.ToUpper((*t.Actor)[:1]) + (*t.Actor)[1:] + " "
			}
			desc = actor + "Condition" + conditionTagText(t.Negative, t.VariableList)
		case *mod.MultiplierTag:
			base := baseVal
			if t.TagBase != 0 {
				v, _ := modRawValue(m)
				base = formatModBase(m, t.TagBase) + " + " + luaNumber(math.Abs(v)) + " "
			}
			div := ""
			if t.DivisionSet {
				div = luaNumber(t.Division) + " "
			}
			desc = base + "per " + div + formatVarNameOrList(t.VariableList)
			baseVal = ""
		case *mod.PerStatTag:
			base := baseVal
			if t.Base != 0 {
				v, _ := modRawValue(m)
				base = formatModBase(m, t.Base) + " + " + luaNumber(math.Abs(v)) + " "
			}
			div := t.Divide
			if div == 0 {
				div = 1
			}
			actor := ""
			if t.TagActor != "" {
				actor = t.TagActor + " "
			}
			desc = base + "per " + luaNumber(div) + " " + actor + formatVarNameOrList(t.StatList)
			baseVal = ""
		case *mod.PercentStatTag:
			v, _ := modRawValue(m)
			percent := t.Percent
			if percent == 0 {
				percent = 1
			}
			finalPercent := (v * (percent / 100)) * 100
			actor := ""
			if t.TagActor != "" {
				actor = t.TagActor + " "
			}
			stat := t.Stat
			if t.PercentVar != "" {
				stat = t.PercentVar
			}
			desc = formatModBase(m, finalPercent) + "% of " + actor + formatModName(stat)
			baseVal = ""
		case *mod.MultiplierThresholdTag:
			threshold := ""
			switch {
			case t.TagThreshold != nil:
				threshold = luaNumber(*t.TagThreshold)
			case t.ThresholdVariable != nil:
				threshold = formatModName(*t.ThresholdVariable)
			}
			name := formatModName(t.Variable)
			if len(t.VariableList) > 0 {
				name = formatVarNameOrList(t.VariableList)
			}
			desc = "If " + name + utilsTernary(t.TagUpper, " <= ", " >= ") + threshold
		case *mod.StatThresholdTag:
			pct := ""
			if t.ThresholdPercent != 0 {
				pct = luaNumber(t.ThresholdPercent) + "% "
			}
			threshold := luaNumber(t.Threshold)
			if t.TagThresholdStat != "" {
				threshold = formatModName(t.TagThresholdStat)
			}
			desc = "If " + formatModName(t.Stat) + utilsTernary(t.TagUpper, " <= ", " >= ") + pct + threshold
		case *mod.SkillNameTag:
			desc = "Skill: " + strings.Join(t.SkillNameList, " / ")
		case *mod.SkillIDTag:
			name := t.IDTag
			if text, ok := raw2.SkillTextFor(t.IDTag); ok {
				name = text.Name
			}
			desc = "Skill: " + name
		case *mod.SkillTypeTag:
			name := "?"
			if len(t.SkillTypeList) > 0 {
				name = formatModName(t.SkillTypeList[0])
			}
			desc = "Skill type: " + utilsTernary(t.Negative, "Not ", "") + name
		case *mod.SlotNumberTag:
			desc = "When in slot #" + strconv.Itoa(t.N)
		case *mod.GlobalEffectTag:
			if len(t.GlobalEffectList) > 0 {
				desc = formatModName(t.GlobalEffectList[0])
			}
		case *mod.LimitTag:
			if t.LimitVar != "" {
				desc = "Limited to " + formatModName(t.LimitVar)
			} else {
				desc = "Limited to " + formatModBase(m, t.TagLimit)
			}
		case *mod.MonsterTagTag:
			desc = "Monster Tag: " + strings.Join(t.MonsterTagList, "/")
		default:
			desc = formatModName(string(tag.Type()))
		}
		if desc != "" {
			out = append(out, desc)
		}
	}
	return strings.Join(out, ", ")
}

func utilsTernary(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

func (a *ActiveSkill) breakdownCfg(name string) *moddb.ListCfg {
	switch name {
	case "skill":
		return a.SkillCfg
	case "weapon1":
		return a.Weapon1Cfg
	case "weapon2":
		return a.Weapon2Cfg
	case "bleed":
		return a.BleedCfg
	case "OHbleed":
		return a.OHBleedCfg
	case "poison":
		return a.PoisonCfg
	case "OHpoison":
		return a.OHPoisonCfg
	case "ignite":
		return a.IgniteCfg
	case "OHignite":
		return a.OHIgniteCfg
	case "decay":
		return a.DecayCfg
	case "dot":
		return a.DotCfg
	}
	if damageType, ok := strings.CutPrefix(name, "dot"); ok {
		return a.DotTypeCfg[data.DamageType(damageType)]
	}
	return nil
}

//pob:src/Classes/PassiveTree.lua:754-758
var tattooIDMap = sync.OnceValue(func() map[string]string {
	idMap := make(map[string]string, len(data.TattooNodes))
	for _, node := range data.TattooNodes {
		idMap[node.ID] = node.DN
	}
	return idMap
})

func (env *Environment) modSourceName(source string) (string, string, string, int64) {
	sourceType, _, _ := strings.Cut(source, ":")
	switch sourceType {
	case "Item":
		if m := itemSourceRe.FindStringSubmatch(source); m != nil {
			id, _ := strconv.Atoi(m[1])
			if text, ok := itemTextByID(env)[id]; ok {
				item := ParseItem(text)
				return sourceType, rarityColor[item.Rarity] + item.Name, item.Name, 0
			}
		}
		if parts := strings.SplitN(source, ":", 3); len(parts) == 3 && parts[1] == "-1" {
			if item := env.itemByName(parts[2]); item != nil {
				return sourceType, rarityColor[item.Rarity] + item.Name, item.Name, 0
			}
		}
	case "Tree":
		if m := treeSourceRe.FindStringSubmatch(source); m != nil {
			id, _ := strconv.ParseInt(m[1], 10, 64)
			if replacement := nodeRewrites(env)[m[1]].effectiveReplacement(); replacement != nil {
				return sourceType, replacement.DN, "", id
			}
			node, ok := env.clusterNodes[m[1]]
			if !ok {
				node = resolveTreeNode(data.TreeVersions[data.LatestTreeVersion].Tree(), m[1])
			}
			if node.Name != nil {
				return sourceType, *node.Name, "", id
			}
		} else if m := tattooSourceRe.FindStringSubmatch(source); m != nil {
			//pob:block:src/Classes/CalcBreakdownControl.lua:414-415
			return sourceType, tattooIDMap()[m[1]], "", 0
		}
	case "Skill":
		if text, ok := raw2.SkillTextFor(strings.TrimPrefix(source, "Skill:")); ok {
			return sourceType, text.Name, "", 0
		}
	case "Pantheon", "Spectre", "Custom":
		return sourceType, strings.TrimPrefix(source, sourceType+":"), "", 0
	}
	return sourceType, "", "", 0
}

//pob:src/Classes/CalcBreakdownControl.lua:271-498
//nolint:gocyclo // Mirrors the Lua.
func (env *Environment) CalcsModSection(sd ModSectionData, showMinion bool) []ModSectionRow {
	if env == nil || env.Player == nil {
		return nil
	}
	actor := env.Player
	if showMinion && env.Minion != nil && env.Minion.Actor != nil {
		actor = env.Minion.Actor
	}
	if actor.MainSkill == nil {
		return nil
	}
	cfg := &moddb.ListCfg{}
	if sd.Cfg != "" {
		if c := actor.MainSkill.breakdownCfg(sd.Cfg); c != nil {
			copied := *c
			cfg = &copied
		}
	}
	cfg.Source = nil
	if sd.ModSource != "" {
		src := mod.Source(sd.ModSource)
		cfg.Source = &src
	}
	cfg.Actor = sd.Actor

	type store interface {
		Tabulate(modType mod.Type, cfg *moddb.ListCfg, modNames ...string) []moddb.ModResult
		Combine(modType mod.Type, cfg *moddb.ListCfg, modNames ...string) *mod.ModValueMulti
	}
	var modStore store
	switch {
	case sd.Enemy && actor.Enemy != nil:
		modStore = actor.Enemy.ModDB
	case sd.Cfg != "" && actor.MainSkill.SkillModList != nil:
		modStore = actor.MainSkill.SkillModList
	default:
		modStore = actor.ModDB
	}
	rowList := modStore.Tabulate(mod.Type(sd.ModType), cfg, sd.ModName...)
	if len(rowList) == 0 {
		return nil
	}

	byValue := func(a, b moddb.ModResult) bool {
		if a.Mod.Name() != b.Mod.Name() {
			return a.Mod.Name() > b.Mod.Name()
		}
		return a.Value > b.Value
	}
	if sd.ModType == "" {
		sort.SliceStable(rowList, func(i, j int) bool {
			a, b := rowList[i], rowList[j]
			if a.Mod.Type() == b.Mod.Type() {
				return byValue(a, b)
			}
			return a.Mod.Type() < b.Mod.Type()
		})
	} else {
		sort.SliceStable(rowList, func(i, j int) bool { return byValue(rowList[i], rowList[j]) })
	}

	sourceTotals := map[string][]string{}
	if sd.ModSource == "" {
		var typeList []mod.Type
		for _, row := range rowList {
			if !slices.Contains(typeList, row.Mod.Type()) {
				typeList = append(typeList, row.Mod.Type())
			}
			sourceType, _, _ := strings.Cut(string(row.Mod.GetSource()), ":")
			sourceTotals[sourceType] = nil
		}
		for sourceType := range sourceTotals {
			if sourceType == "" {
				continue
			}
			src := mod.Source(sourceType)
			cfg.Source = &src
			var lines []string
			total := func(modType mod.Type, names ...string) (string, bool) {
				v := modStore.Combine(modType, cfg, names...)
				if v == nil {
					return "", false
				}
				switch v.Type() {
				case mod.ModValueMultiTypeFlag:
					return breakdownModValue(0, modType), v.Flag()
				case mod.ModValueMultiTypeList:
					return breakdownModValue(0, modType), true
				case mod.ModValueMultiTypeFloat:
				default:
					return "", false
				}
				t := v.Float()
				if modType == mod.TypeMore {
					t = math.Floor((t-1)*100 + 0.5)
				}
				return breakdownModValue(t, modType), t != 0
			}
			for _, modType := range typeList {
				if sd.Multi {
					for _, modName := range sd.ModName {
						if t, ok := total(modType, modName); ok {
							lines = append(lines, t+" "+modNameSpaced(modName))
						}
					}
				} else if t, ok := total(modType, sd.ModName...); ok {
					lines = append(lines, t)
				}
			}
			sourceTotals[sourceType] = lines
		}
	}

	rows := make([]ModSectionRow, 0, len(rowList))
	for _, r := range rowList {
		row := ModSectionRow{}
		if sd.ModType == "" {
			row.Value = breakdownModValue(r.Value, r.Mod.Type())
		} else {
			row.Value = luaNumber(luaRound(r.Value, 2))
		}
		if sd.Multi {
			row.Name = formatModName(r.Mod.Name())
		}
		row.Source, row.SourceName, row.ItemName, row.NodeID = env.modSourceName(string(r.Mod.GetSource()))
		if sd.ModSource == "" {
			row.Totals = append([]string{"Total from " + row.Source + ":"}, sourceTotals[row.Source]...)
		}
		row.Flags = modFlagText(r.Mod)
		row.Tags = modTagText(r.Mod)
		rows = append(rows, row)
	}
	return rows
}

//pob:block:src/Classes/CalcBreakdownControl.lua:281-282,src/Classes/CalcBreakdownControl.lua:319-322,src/Classes/CalcBreakdownControl.lua:372-398
func (env *Environment) CalcsBreakdownModList(name string, showMinion bool) []ModSectionRow {
	if env == nil || env.Player == nil {
		return nil
	}
	actor := env.Player
	if showMinion && env.Minion != nil && env.Minion.Actor != nil {
		actor = env.Minion.Actor
	}
	rowList := slices.Clone(actor.breakdownModLists[name])
	if len(rowList) == 0 {
		return nil
	}
	sort.SliceStable(rowList, func(i, j int) bool {
		a, b := rowList[i], rowList[j]
		if a.Mod.Name() != b.Mod.Name() {
			return a.Mod.Name() > b.Mod.Name()
		}
		return a.Value > b.Value
	})
	rows := make([]ModSectionRow, 0, len(rowList))
	for _, r := range rowList {
		row := ModSectionRow{Value: breakdownModValue(r.Value, r.Mod.Type()), Name: formatModName(r.Mod.Name())}
		row.Source, row.SourceName, row.ItemName, row.NodeID = env.modSourceName(string(r.Mod.GetSource()))
		row.Flags = modFlagText(r.Mod)
		row.Tags = modTagText(r.Mod)
		rows = append(rows, row)
	}
	return rows
}

func modNameSpaced(modName string) string {
	return modNameSpacedRe.ReplaceAllString(modName, "$1 $2")
}

var modNameSpacedRe = regexp.MustCompile(`([a-z])([A-Z])`)

func ItemByName(build *pob.PathOfBuilding, name string) *Item {
	ids := make([]int, 0, len(build.Items.Items))
	texts := map[int]string{}
	for _, it := range build.Items.Items {
		ids = append(ids, it.ID)
		texts[it.ID] = it.Text
	}
	sort.Ints(ids)
	for _, id := range ids {
		if item := ParseItem(texts[id]); item != nil && item.Name == name {
			return item
		}
	}
	return nil
}

func (env *Environment) itemByName(name string) *Item {
	if env.itemsByName == nil {
		env.itemsByName = map[string]*Item{}
		ids := make([]int, 0, len(itemTextByID(env)))
		for id := range itemTextByID(env) {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			if item := ParseItem(itemTextByID(env)[id]); item != nil {
				if _, ok := env.itemsByName[item.Name]; !ok {
					env.itemsByName[item.Name] = item
				}
			}
		}
	}
	return env.itemsByName[name]
}
