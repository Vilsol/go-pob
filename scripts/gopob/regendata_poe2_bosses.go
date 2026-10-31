package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var rdGeneratorsPoE2Bosses = []rdGenerator{
	{
		name:     "poe2-bosses",
		source:   rdGameDataFromPoB,
		out:      "data/poe2/bosses_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		luaArgs:  []string{"bosses"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"bosses"},
		needsPob: true,
		note:     "data.bosses (src/Data/Bosses.lua) and data.bossStats as Data.lua computes it. Game data with no PoE2 game source: the pin's Export/Scripts/bossData.lua is disabled (`goto exit` before any export), its Export/Enemies/Bosses.txt names no boss, and go-pob-data's 0.5 MonsterVarieties holds none of the file's monsters, so the pin's checked-in file is the only source",
	},
	{
		name:     "poe2-boss-skills",
		source:   rdGameDataFromPoB,
		out:      "data/poe2/boss_skills_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		luaArgs:  []string{"boss-skills"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"boss-skills"},
		needsPob: true,
		note:     "data.bossSkills (src/Data/BossSkills.lua), the presetBossSkills presets; tooltip and bossSkillsList dropped. Game data with no PoE2 game source: bossData.lua is disabled at the pin and go-pob-data's 0.5 MonsterVarieties and GrantedEffects hold none of its bosses or skills, so the pin's checked-in file is the only source",
	},
}

type p2eBossJSON struct {
	Name        string  `json:"name"`
	ArmourMult  float64 `json:"armourMult"`
	EvasionMult float64 `json:"evasionMult"`
	IsUber      bool    `json:"isUber"`
}

var p2eBossStatNames = []string{"PinnacleArmourMean", "PinnacleEvasionMean", "UberArmourMean", "UberEvasionMean"}

func p2eBosses(body []byte) (string, error) {
	var dump struct {
		Bosses    []p2eBossJSON      `json:"bosses"`
		BossStats map[string]float64 `json:"bossStats"`
	}
	if err := p2eDecode(body, &dump); err != nil {
		return "", err
	}
	if len(dump.Bosses) == 0 {
		return "", fmt.Errorf("holds no bosses")
	}
	if len(dump.BossStats) != len(p2eBossStatNames) {
		return "", fmt.Errorf("bossStats holds %d entries, the emitter knows %d", len(dump.BossStats), len(p2eBossStatNames))
	}
	var b strings.Builder
	p2eHeader(&b, "bosses", "src/Modules/Data.lua:851-876")
	b.WriteString("// Boss is one row of data.bosses.\ntype Boss struct {\n\tArmourMult  float64\n\tEvasionMult float64\n\tIsUber      bool\n}\n\n")
	fmt.Fprintf(&b, "// Bosses is data.bosses, %d bosses keyed by name.\nvar Bosses = map[string]Boss{\n", len(dump.Bosses))
	for _, r := range dump.Bosses {
		if r.Name == "" {
			return "", fmt.Errorf("a boss has no name")
		}
		f := &p2eFields{}
		f.num("ArmourMult", r.ArmourMult)
		f.num("EvasionMult", r.EvasionMult)
		f.flag("IsUber", r.IsUber)
		b.WriteString("\t" + strconv.Quote(r.Name) + ": " + f.String() + ",\n")
	}
	b.WriteString("}\n\n// BossStats is data.bossStats, the armour and evasion means over data.bosses.\nvar BossStats = map[string]float64{\n")
	for _, k := range p2eBossStatNames {
		v, ok := dump.BossStats[k]
		if !ok {
			return "", fmt.Errorf("bossStats has no %s", k)
		}
		b.WriteString("\t" + strconv.Quote(k) + ": " + p2eFloat(v) + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type p2eBossSkillJSON struct {
	Name                   string                     `json:"name"`
	DamageType             string                     `json:"DamageType"`
	DamageMultipliers      map[string][]float64       `json:"DamageMultipliers"`
	UberDamageMultiplier   *float64                   `json:"UberDamageMultiplier"`
	DamagePenetrations     map[string]json.RawMessage `json:"DamagePenetrations"`
	UberDamagePenetrations map[string]json.RawMessage `json:"UberDamagePenetrations"`
	Speed                  *float64                   `json:"speed"`
	UberSpeed              *float64                   `json:"UberSpeed"`
	CritChance             *float64                   `json:"critChance"`
	EarlierUber            bool                       `json:"earlierUber"`
	Tooltip                string                     `json:"tooltip"`
	AdditionalStats        *struct {
		Base map[string]json.RawMessage `json:"base"`
		Uber map[string]json.RawMessage `json:"uber"`
	} `json:"additionalStats"`
}

// p2eBossValue is an optional number: absent and a present 0 differ (critChance = 0 overwrites the
// placeholder, an absent one leaves it).
func p2eBossValue(v *float64) string {
	if v == nil {
		return ""
	}
	return "BossSkillValue{Value: " + p2eFloat(*v) + ", Set: true}"
}

// p2eBossPens keeps a pen of "" as unset: SetPlaceholder("") reads back as tonumber("") = nil.
func p2eBossPens(row, field string, pens map[string]json.RawMessage) (string, error) {
	if pens == nil {
		return "", nil
	}
	parts := make([]string, 0, len(pens))
	for _, k := range insSortedKeys(rdRawKeys(pens)) {
		var s string
		if json.Unmarshal(pens[k], &s) == nil {
			if s != "" {
				return "", fmt.Errorf("%s.%s.%s is the string %q, not a number or \"\"", row, field, k, s)
			}
			parts = append(parts, strconv.Quote(k)+": {}")
			continue
		}
		var n float64
		if err := json.Unmarshal(pens[k], &n); err != nil {
			return "", fmt.Errorf("%s.%s.%s is neither a number nor \"\"", row, field, k)
		}
		parts = append(parts, strconv.Quote(k)+": {Value: "+p2eFloat(n)+", Set: true}")
	}
	return "map[string]BossSkillValue{" + strings.Join(parts, ", ") + "}", nil
}

// p2eBossStats keeps additionalStats' two value kinds apart: the literal "flag" makes a FLAG mod.
func p2eBossStats(row, field string, stats map[string]json.RawMessage) (string, error) {
	if stats == nil {
		return "nil", nil
	}
	parts := make([]string, 0, len(stats))
	for _, k := range insSortedKeys(rdRawKeys(stats)) {
		var s string
		if json.Unmarshal(stats[k], &s) == nil {
			if s != "flag" {
				return "", fmt.Errorf("%s.additionalStats.%s.%s is the string %q, not a number or \"flag\"", row, field, k, s)
			}
			parts = append(parts, strconv.Quote(k)+": {IsFlag: true}")
			continue
		}
		var n float64
		if err := json.Unmarshal(stats[k], &n); err != nil {
			return "", fmt.Errorf("%s.additionalStats.%s.%s is neither a number nor \"flag\"", row, field, k)
		}
		parts = append(parts, strconv.Quote(k)+": {Value: "+p2eFloat(n)+"}")
	}
	return "map[string]BossSkillStat{" + strings.Join(parts, ", ") + "}", nil
}

func rdRawKeys(m map[string]json.RawMessage) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

const p2eBossSkillTypes = `// BossSkillValue is an optional number of a preset: Set is false where PoB has nil.
type BossSkillValue struct {
	Value float64
	Set   bool
}

// BossSkillStat is one additionalStats entry: IsFlag where PoB has the string "flag".
type BossSkillStat struct {
	Value  float64
	IsFlag bool
}

// BossSkillAdditionalStats is additionalStats; a nil map is a difficulty PoB has no entry for.
type BossSkillAdditionalStats struct {
	Base map[string]BossSkillStat
	Uber map[string]BossSkillStat
}

// BossSkill is one row of data.bossSkills. DamageMultipliers keeps PoB's unnamed pair.
type BossSkill struct {
	DamageType             string
	DamageMultipliers      map[string][2]float64
	UberDamageMultiplier   BossSkillValue
	DamagePenetrations     map[string]BossSkillValue
	UberDamagePenetrations map[string]BossSkillValue
	Speed                  BossSkillValue
	UberSpeed              BossSkillValue
	CritChance             BossSkillValue
	EarlierUber            bool
	AdditionalStats        *BossSkillAdditionalStats
}

`

func p2eBossSkills(body []byte) (string, error) {
	var rows []p2eBossSkillJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("holds no boss skills")
	}
	var b strings.Builder
	p2eHeader(&b, "boss-skills", "src/Modules/Data.lua:878-880")
	b.WriteString(p2eBossSkillTypes)
	fmt.Fprintf(&b, "// BossSkills is data.bossSkills, %d presets keyed by the presetBossSkills value.\nvar BossSkills = map[string]BossSkill{\n", len(rows))
	for _, r := range rows {
		if r.Name == "" {
			return "", fmt.Errorf("a boss skill has no name")
		}
		f := &p2eFields{}
		f.str("DamageType", r.DamageType)
		if len(r.DamageMultipliers) == 0 {
			return "", fmt.Errorf("%s has no DamageMultipliers", r.Name)
		}
		mults := make([]string, 0, len(r.DamageMultipliers))
		types := make([]string, 0, len(r.DamageMultipliers))
		for k := range r.DamageMultipliers {
			types = append(types, k)
		}
		sort.Strings(types)
		for _, k := range types {
			m := r.DamageMultipliers[k]
			if len(m) != 2 {
				return "", fmt.Errorf("%s.DamageMultipliers.%s holds %d numbers, not 2", r.Name, k, len(m))
			}
			mults = append(mults, strconv.Quote(k)+": {"+p2eFloat(m[0])+", "+p2eFloat(m[1])+"}")
		}
		f.raw("DamageMultipliers", "map[string][2]float64{"+strings.Join(mults, ", ")+"}")
		f.raw("UberDamageMultiplier", p2eBossValue(r.UberDamageMultiplier))
		pens, err := p2eBossPens(r.Name, "DamagePenetrations", r.DamagePenetrations)
		if err != nil {
			return "", err
		}
		f.raw("DamagePenetrations", pens)
		uberPens, err := p2eBossPens(r.Name, "UberDamagePenetrations", r.UberDamagePenetrations)
		if err != nil {
			return "", err
		}
		f.raw("UberDamagePenetrations", uberPens)
		f.raw("Speed", p2eBossValue(r.Speed))
		f.raw("UberSpeed", p2eBossValue(r.UberSpeed))
		f.raw("CritChance", p2eBossValue(r.CritChance))
		f.flag("EarlierUber", r.EarlierUber)
		if a := r.AdditionalStats; a != nil {
			base, err := p2eBossStats(r.Name, "base", a.Base)
			if err != nil {
				return "", err
			}
			uber, err := p2eBossStats(r.Name, "uber", a.Uber)
			if err != nil {
				return "", err
			}
			af := &p2eFields{}
			af.raw("Base", base)
			af.raw("Uber", uber)
			f.raw("AdditionalStats", "&BossSkillAdditionalStats"+af.String())
		}
		b.WriteString("\t" + strconv.Quote(r.Name) + ": " + f.String() + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}
