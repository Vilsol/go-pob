package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

type canonTag struct {
	typ      string
	params   string
	unmapped bool
}

func (c canonTag) String() string {
	if c.params == "" {
		return c.typ
	}
	return c.typ + "/" + c.params
}

func canonParams(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v == "" || v == "false" ||
			(k == "div" && (canonNumber(v) == "1" || canonNumber(v) == "0")) ||
			(k == "base" && canonNumber(v) == "0") ||
			(k == "globalLimit" && canonNumber(v) == "0") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + canonNumber(m[k])
	}
	return strings.Join(parts, "/")
}

func canonNumber(v string) string {
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return fmt.Sprintf("%.14g", f)
	}
	return v
}

func canonList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return "{" + strings.Join(items, ",") + "}"
}

func pobTagSegment(name string, m tagDumpMod) (string, bool) {
	flags := m.Flags
	if flags == "" {
		flags = "-"
	}
	kw := m.Kw
	if kw == "" {
		kw = "-"
	}
	marker := " = " + name + "|" + m.Type + "|" + flags + "|" + kw + "|"
	i := strings.LastIndex(m.Fmt, marker)
	if i < 0 {
		return "", false
	}
	return m.Fmt[i+len(marker):], true
}

func splitTop(s string, sep byte) []string {
	out := make([]string, 0, 4)
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

func parsePobTags(seg string) []canonTag {
	if seg == "" || seg == "-" {
		return nil
	}
	var raw []string
	for _, r := range splitTop(seg, ',') {
		if len(raw) > 0 && !strings.HasPrefix(r, "type=") {
			raw[len(raw)-1] += "," + r
			continue
		}
		raw = append(raw, r)
	}
	out := make([]canonTag, 0, len(raw))
	for _, r := range raw {
		typ := ""
		params := map[string]string{}
		for _, kv := range splitTop(r, '/') {
			k, v, found := strings.Cut(kv, "=")
			if !found {
				continue
			}
			if k == "type" {
				typ = v
				continue
			}
			params[k] = v
		}
		out = append(out, canonPobTag(typ, params))
	}
	return out
}

func canonPobTag(typ string, params map[string]string) canonTag {
	fold := func(singular, list, into string) {
		if v, ok := params[list]; ok {
			delete(params, list)
			params[into] = strings.TrimSuffix(strings.TrimPrefix(v, "{"), "}")
		}
		if v, ok := params[singular]; ok {
			delete(params, singular)
			if _, taken := params[into]; !taken {
				params[into] = v
			}
		}
	}
	fold("var", "varList", "vars")
	fold("stat", "statList", "stats")
	fold("skillName", "skillNameList", "skillNames")
	fold("skillType", "skillTypeList", "skillTypes")
	fold("skillPart", "skillPartList", "skillParts")
	if typ == string(mod.TypeSlotName) {
		fold("slotName", "slotNameList", "slotNames")
	}

	if v, ok := params["skillTypes"]; ok {
		parts := strings.Split(v, ",")
		for i := range parts {
			parts[i] = canonNumber(parts[i])
		}
		params["skillTypes"] = strings.Join(parts, ",")
	}
	return canonTag{typ: typ, params: canonParams(params)}
}

func canonOurTag(t mod.Tag) canonTag {
	p := map[string]string{}
	set := func(k, v string) {
		if v != "" {
			p[k] = v
		}
	}
	setNum := func(k string, v float64) { p[k] = strconv.FormatFloat(v, 'g', -1, 64) }
	setBool := func(k string, v bool) {
		if v {
			p[k] = "true"
		}
	}
	setNumPtr := func(k string, v *float64) {
		if v != nil {
			setNum(k, *v)
		}
	}
	setStrPtr := func(k string, v *string) {
		if v != nil {
			set(k, *v)
		}
	}

	switch tag := t.(type) {
	case *mod.ConditionTag:
		set("vars", strings.Join(tag.VarList, ","))
		setBool("neg", tag.Negative)
	case *mod.ActorConditionTag:
		setStrPtr("actor", tag.Actor)
		set("vars", strings.Join(tag.VariableList, ","))
		setBool("neg", tag.Negative)
	case *mod.MultiplierTag:
		set("vars", strings.Join(tag.VariableList, ","))
		setNum("base", tag.TagBase)
		setNum("div", tag.Division)
		setNumPtr("limit", tag.TagLimit)
		setStrPtr("limitVar", tag.TagLimitVariable)
		setBool("limitTotal", tag.TagLimitTotal)
		set("actor", tag.TagActor)
		setNumPtr("globalLimit", tag.TagGlobalLimit)
		setStrPtr("globalLimitKey", tag.TagGlobalLimitKey)
		setStrPtr("divVar", tag.TagDivVariable)
		setBool("noFloor", tag.TagNoFloor)
		setStrPtr("limitStat", tag.TagLimitStat)
		setBool("limitNegTotal", tag.TagLimitNegTotal)
		set("limitActor", tag.TagLimitActor)
	case *mod.MultiplierThresholdTag:
		if len(tag.VariableList) > 0 {
			set("vars", strings.Join(tag.VariableList, ","))
		} else {
			set("vars", tag.Variable)
		}
		setNumPtr("threshold", tag.TagThreshold)
		setStrPtr("thresholdVar", tag.ThresholdVariable)
		setBool("upper", tag.TagUpper)
		set("actor", tag.TagActor)
	case *mod.PerStatTag:
		set("stats", strings.Join(tag.StatList, ","))
		setNum("div", tag.Divide)
		setNum("base", tag.Base)
		setNumPtr("limit", tag.TagLimit)
		setStrPtr("limitVar", tag.TagLimitVariable)
		setBool("limitTotal", tag.TagLimitTotal)
		set("actor", tag.TagActor)
		setNum("globalLimit", tag.TagGlobalLimit)
		set("globalLimitKey", tag.TagGlobalLimitKey)
	case *mod.PercentStatTag:
		set("stats", tag.Stat)
		if tag.PercentVar != "" {
			set("percentVar", tag.PercentVar)
		} else {
			setNum("percent", tag.Percent)
		}
		set("actor", tag.TagActor)
		setBool("floor", tag.TagFloor)
	case *mod.StatThresholdTag:
		set("stats", tag.Stat)
		if tag.TagThresholdStat != "" {
			set("thresholdStat", tag.TagThresholdStat)
		} else {
			setNum("threshold", tag.Threshold)
		}
		setBool("upper", tag.TagUpper)
		if tag.ThresholdPercent != 0 {
			setNum("thresholdPercent", tag.ThresholdPercent)
		}
	case *mod.SkillTypeTag:
		nums := make([]string, 0, len(tag.SkillTypeList))
		for _, st := range tag.SkillTypeList {
			n, ok := pobSkillTypeNumber[st]
			if !ok {
				return canonTag{typ: string(mod.TypeSkillType), unmapped: true}
			}
			nums = append(nums, strconv.Itoa(n))
		}
		set("skillTypes", strings.Join(nums, ","))
		setBool("neg", tag.Negative)
	case *mod.MonsterTagTag:
		set("monsterTag", strings.Join(tag.MonsterTagList, ","))
		setBool("neg", tag.Negative)
	case *mod.SkillNameTag:
		set("skillNames", strings.Join(tag.SkillNameList, ","))
		setBool("neg", tag.Negative)
		setBool("summonSkill", tag.TagSummonSkill)
		setBool("includeTransfigured", tag.IncludeTransfigured)
	case *mod.SkillIDTag:
		set("skillId", tag.ID())
	case *mod.SkillPartTag:
		if len(tag.PartList) > 0 {
			parts := make([]string, len(tag.PartList))
			for i, v := range tag.PartList {
				parts[i] = strconv.Itoa(v)
			}
			set("skillParts", strings.Join(parts, ","))
		} else {
			set("skillParts", strconv.Itoa(tag.Part))
		}
		setBool("neg", tag.Negative)
	case *mod.SocketedInTag:
		set("slotName", tag.SlotName)
		set("keyword", tag.TagKeyword)
		set("socketColor", tag.TagSocketColor)
		if len(tag.TagSockets) > 0 {
			nums := make([]string, len(tag.TagSockets))
			for i, n := range tag.TagSockets {
				nums[i] = strconv.Itoa(n)
			}
			set("sockets", canonList(nums))
		}
		if tag.TagSocketsAll {
			set("sockets", "all")
		}
		if tag.TagSocketsCount != 0 {
			setNum("sockets", tag.TagSocketsCount)
		}
		setBool("neg", tag.TagNeg)
	case *mod.SlotNameTag:
		set("slotNames", strings.Join(tag.SlotNameList, ","))
	case *mod.SlotNumberTag:
		setNum("num", float64(tag.N))
	case *mod.GlobalEffectTag:
		if len(tag.GlobalEffectList) > 0 {
			set("effectType", tag.GlobalEffectList[0])
		}
		set("effectName", tag.NameTag)
		set("effectCond", tag.EffectCondTag)
		set("modCond", tag.ModCondTag)
		setBool("unscalable", tag.UnscalableTag)
		set("effectEnemyCond", tag.EffectEnemyCondTag)
		set("effectStackVar", tag.EffectStackVarTag)
		if tag.EffectStackLimitTag != 0 {
			setNum("effectStackLimit", tag.EffectStackLimitTag)
		}
		set("effectStackLimitVar", tag.EffectStackLimitVarTag)
		setBool("applyNotPlayer", tag.ApplyNotPlayerTag)
		setBool("applyMinions", tag.ApplyMinionsTag)
		setBool("allowTotemBuff", tag.AllowTotemBuffTag)
		if tag.DivSet {
			setNum("div", tag.DivTag)
		}
		if tag.LimitSet {
			setNum("limit", tag.LimitTag)
		}
		setBool("neg", tag.Negative)
	case *mod.DisablesItemTag:
		set("slotName", tag.TagSlotName)
		set("excludeItemType", tag.TagExcludeItemType)
	case *mod.ItemConditionTag:
		set("itemSlot", tag.ItemSlot)
		set("searchCond", tag.SearchCond)
		set("nameCond", tag.NameCond)
		set("rarityCond", tag.RarityCond)
		if tag.CorruptedCond != nil {
			setBool("corruptedCond", *tag.CorruptedCond)
		}
		setBool("allSlots", tag.AllSlots)
		setBool("bothSlots", tag.BothSlots)
		setBool("excludeSelf", tag.ExcludeSelf)
		setBool("shaperCond", tag.ShaperCond)
		setBool("elderCond", tag.ElderCond)
		setBool("neg", tag.Neg)
	case *mod.GlobalTag:
		setBool("neg", tag.Negative)
	case *mod.IgnoreCondTag, mod.IgnoreCondTag:
		return canonTag{typ: string(mod.TypeIgnoreCond)}
	case *mod.ModFlagTag:
		setNum("modFlags", float64(tag.Flag))
	case *mod.ModFlagOrTag:
		setNum("modFlags", float64(tag.Flag))
	case *mod.KeywordFlagAndTag:
		setNum("keywordFlags", float64(tag.KeywordFlags))
	case *mod.DistanceRampTag:
		set("ramp", canonRampFloat(tag.Ramp))
	case *mod.MeleeProximityTag:
		steps := make([]string, len(tag.Ramp))
		for i, v := range tag.Ramp {
			steps[i] = strconv.Itoa(v)
		}
		set("ramp", canonList(steps))
	case *mod.LimitTag:
		if tag.LimitVar != "" {
			set("limitVar", tag.LimitVar)
		} else {
			setNum("limit", tag.TagLimit)
		}
	case *mod.InSlotTag:
		setNum("num", float64(tag.N))
	case mod.InSlotTag:
		setNum("num", float64(tag.N))
	case *mod.FlagTag:
		setBool("value", tag.Value)
	case mod.FlagTag:
		setBool("value", tag.Value)
	default:
		return canonTag{typ: string(t.Type()), unmapped: true}
	}
	return canonTag{typ: string(t.Type()), params: canonParams(p)}
}

func canonRampFloat(ramp [][]float64) string {
	if len(ramp) == 0 {
		return ""
	}
	steps := make([]string, 0, len(ramp))
	for _, pair := range ramp {
		kv := make([]string, 0, len(pair))
		for i, v := range pair {
			kv = append(kv, strconv.Itoa(i+1)+"="+strconv.FormatFloat(v, 'g', -1, 64))
		}
		steps = append(steps, strings.Join(kv, "/"))
	}
	return canonList(steps)
}

//pob:src/Data/Global.lua:218-353
var pobSkillTypeNumber = func() map[string]int {
	names := []string{
		"Attack", "Spell", "Projectile", "DualWieldOnly",
		"Buff", "Minion", "Damage", "Area",
		"Duration", "RequiresShield", "ProjectileSpeed", "HasReservation",
		"ReservationBecomesCost", "Trappable", "Totemable", "Mineable",
		"ElementalStatus", "MinionsCanExplode", "Chains", "Melee",
		"MeleeSingleTarget", "Multicastable", "TotemCastsAlone", "Multistrikeable",
		"CausesBurning", "SummonsTotem", "TotemCastsWhenNotDetached", "Physical",
		"Fire", "Cold", "Lightning", "Triggerable",
		"Trapped", "Movement", "DamageOverTime", "RemoteMined",
		"Triggered", "Vaal", "Aura", "CanTargetUnusableCorpse",
		"RangedAttack", "Chaos", "FixedSpeedProjectile", "ThresholdJewelArea",
		"ThresholdJewelProjectile", "ThresholdJewelDuration", "ThresholdJewelRangedAttack", "Channel",
		"DegenOnlySpellDamage", "InbuiltTrigger", "Golem", "Herald",
		"AuraAffectsEnemies", "NoRuthless", "ThresholdJewelSpellDamage", "Cascadable",
		"ProjectilesFromUser", "MirageArcherCanUse", "ProjectileSpiral", "SingleMainProjectile",
		"MinionsPersistWhenSkillRemoved", "ProjectileNumber", "Warcry", "Instant",
		"Brand", "DestroysCorpse", "NonHitChill", "ChillingArea",
		"AppliesCurse", "CanRapidFire", "AuraDuration", "AreaSpell",
		"OR", "AND", "NOT", "AppliesMaim",
		"CreatesMinion", "Guard", "Travel", "Blink",
		"CanHaveBlessing", "ProjectilesNotFromUser", "AttackInPlaceIsDefault", "Nova",
		"InstantNoRepeatWhenHeld", "InstantShiftAttackForLeftMouse", "AuraNotOnCaster", "Banner",
		"Rain", "Cooldown", "ThresholdJewelChaining", "Slam",
		"Stance", "NonRepeatable", "OtherThingUsesSkill", "Steel",
		"Hex", "Mark", "Aegis", "Orb",
		"KillNoDamageModifiers", "RandomElement", "LateConsumeCooldown", "Arcane",
		"FixedCastTime", "RequiresOffHandNotWeapon", "Link", "Blessing",
		"ZeroReservation", "DynamicCooldown", "Microtransaction", "OwnerCannotUse",
		"ProjectilesNumberModifiersNotApplied", "TotemsAreBallistae", "SkillGrantedBySupport", "PreventHexTransfer",
		"MinionsAreUndamagable", "InnateTrauma", "DualWieldRequiresDifferentTypes", "NoVolley",
		"Retaliation", "NeverExertable", "DisallowTriggerSupports", "ProjectileCannotReturn",
		"Offering", "SupportedByBane", "WandAttack", "GainsIntensity",
		"CreatesSentinelMinion", "SupportedByAutoExertion", "SupportedByCrabTotem", "SupportedBySpellTotem",
		"CreatesCorpse", "RequiresStaff", "Pact",
	}
	out := make(map[string]int, len(names))
	for i, n := range names {
		out[n] = i + 1
	}
	return out
}()

var canonUnserialised = map[string]string{
	"IgnoreCondTag":             "carries only TagType — there is genuinely nothing to render",
	"SkillIDTag.IDTag":          "canonTag renders the resolved skill id, not the tag's own field",
	"MultiplierTag.DivisionSet": "whether the Lua tag carries a div at all, read only by the breakdown's tag text; canonTag renders the value",
}

const canonUnserialisedCeiling = 4

// TestCanonTagRendersEveryTagField scans mod/*.go for every `*Tag` struct field and requires each to be mentioned in canonTag's case for that type.
func TestCanonTagRendersEveryTagField(t *testing.T) {
	canonSrc, err := os.ReadFile("tag_canon_test.go")
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("../mod/*.go")
	if err != nil {
		t.Fatal(err)
	}

	structRe := regexp.MustCompile(`(?s)type (\w+Tag) struct \{(.*?)\n\}`)
	fieldRe := regexp.MustCompile(`(?m)^\t(\w+)\s+\S`)

	var gaps []string
	seenTypes := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range structRe.FindAllStringSubmatch(string(src), -1) {
			name, body := m[1], m[2]
			seenTypes++
			caseRe := regexp.MustCompile(`(?s)case \*mod\.` + name + `:(.*?)(?:\n\tcase \*mod\.|\n\t\}|\z)`)
			cm := caseRe.FindStringSubmatch(string(canonSrc))
			blk := ""
			if cm != nil {
				blk = cm[1]
			}
			if cm == nil {
				if _, ok := canonUnserialised[name]; ok {
					continue
				}
			}
			for _, fm := range fieldRe.FindAllStringSubmatch(body, -1) {
				field := fm[1]
				if field == "TagType" {
					continue
				}
				key := name + "." + field
				if regexp.MustCompile(`\b` + regexp.QuoteMeta(field) + `\b`).MatchString(blk) {
					continue
				}
				if _, ok := canonUnserialised[key]; ok {
					continue
				}
				if _, ok := canonUnserialised[name]; ok {
					continue
				}
				gaps = append(gaps, key)
			}
		}
	}
	sort.Strings(gaps)

	if seenTypes == 0 {
		t.Fatal("scanned no *Tag structs out of ../mod — has the layout changed?")
	}
	if len(gaps) > 0 {
		t.Errorf("tag field(s) %v are not rendered by canonTag and are not declared in "+
			"canonUnserialised.\nTestTagCensus is the ONLY harness that compares tags, and it can "+
			"only see what canonTag renders — an ignored field reads as AGREEMENT on both sides, "+
			"which is worse than a reported gap. SkillName/summonSkill hid that way at the top of "+
			"the census table until 2026-08-03. Render it, or declare it with a reason.", gaps)
	}
	if n := len(canonUnserialised); n > canonUnserialisedCeiling {
		t.Errorf("canonUnserialised has %d entries, ceiling %d — this list may only SHRINK", n, canonUnserialisedCeiling)
	}
}

func TestParsePobTagsKeepsACommaInsideAValue(t *testing.T) {
	got := parsePobTags("type=SkillName/skillName=Summon Skeletons,type=ItemCondition/itemSlot=Weapon 1/nameCond=The Iron Mass, Gladius")
	if len(got) != 2 || got[0].typ != "SkillName" || got[1].typ != "ItemCondition" {
		t.Fatalf("parsed %+v, want SkillName then ItemCondition", got)
	}
}
