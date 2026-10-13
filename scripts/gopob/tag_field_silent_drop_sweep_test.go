package main

import (
	"fmt"
	"sort"
	"testing"
)

var modStoreTagFields = map[string][]string{
	"Multiplier": {"var", "varList", "div", "divVar", "base", "limit", "limitVar", "limitStat",
		"limitTotal", "limitNegTotal", "limitActor", "actor", "noFloor", "invert"},
	"MultiplierThreshold": {"var", "varList", "threshold", "thresholdVar", "thresholdActor",
		"upper", "equals", "actor"},
	"PerStat": {"stat", "statList", "div", "divVar", "base", "limit", "limitVar", "limitTotal", "actor"},
	"PercentStat": {"stat", "statList", "percent", "percentVar", "floor", "base", "limit",
		"limitVar", "limitTotal", "actor"},
	"StatThreshold": {"stat", "statList", "threshold", "thresholdStat", "thresholdPercent",
		"thresholdPercentVar", "upper"},
	"Condition":      {"var", "varList", "neg"},
	"ActorCondition": {"actor", "var", "varList", "neg"},
	"SkillName":      {"skillName", "skillNameList", "neg", "includeTransfigured", "summonSkill"},
	"SkillId":        {"skillId"},
	"SkillPart":      {"skillPart", "skillPartList", "neg"},
	"SkillType":      {"skillType", "skillTypeList", "neg"},
	"DistanceRamp":   {"ramp"},
	"MeleeProximity": {"ramp"},
	"ModFlagOr":      {"modFlags"},
	"Limit":          {"limit", "limitVar"},
}

var tfBase = map[string]map[string]any{
	"Multiplier":          {"type": "Multiplier", "var": "X"},
	"MultiplierThreshold": {"type": "MultiplierThreshold", "var": "X", "threshold": mustNumber("2")},
	"PerStat":             {"type": "PerStat", "stat": "Str"},
	"PercentStat":         {"type": "PercentStat", "stat": "Life", "percent": mustNumber("1")},
	"StatThreshold":       {"type": "StatThreshold", "stat": "Str", "threshold": mustNumber("2")},
	"Condition":           {"type": "Condition", "var": "X"},
	"ActorCondition":      {"type": "ActorCondition", "actor": "enemy", "var": "X"},
	"SkillName":           {"type": "SkillName", "skillName": "Cleave"},
	"SkillId":             {"type": "SkillId", "skillId": "Cleave"},
	"SkillPart":           {"type": "SkillPart", "skillPart": mustNumber("1")},
	"SkillType":           {"type": "SkillType", "skillType": mustNumber("1")},
	"DistanceRamp":        {"type": "DistanceRamp", "ramp": []any{[]any{mustNumber("1"), mustNumber("2")}}},
	"MeleeProximity":      {"type": "MeleeProximity", "ramp": []any{mustNumber("1")}},
	"ModFlagOr":           {"type": "ModFlagOr", "modFlags": mustNumber("0")},
	"Limit":               {"type": "Limit", "limit": mustNumber("3")},
}

var tfAltBase = map[string]map[string]any{
	"Multiplier/varList":                {"type": "Multiplier"},
	"MultiplierThreshold/varList":       {"type": "MultiplierThreshold", "threshold": mustNumber("2")},
	"MultiplierThreshold/thresholdVar":  {"type": "MultiplierThreshold", "var": "X"},
	"PerStat/statList":                  {"type": "PerStat"},
	"PercentStat/statList":              {"type": "PercentStat", "percent": mustNumber("1")},
	"PercentStat/percentVar":            {"type": "PercentStat", "stat": "Life"},
	"StatThreshold/statList":            {"type": "StatThreshold", "threshold": mustNumber("2")},
	"StatThreshold/thresholdStat":       {"type": "StatThreshold", "stat": "Str"},
	"StatThreshold/thresholdPercentVar": {"type": "StatThreshold", "stat": "Str", "threshold": mustNumber("2")},
	"Condition/varList":                 {"type": "Condition"},
	"ActorCondition/varList":            {"type": "ActorCondition", "actor": "enemy"},
	"SkillName/skillNameList":           {"type": "SkillName"},
	"SkillPart/skillPartList":           {"type": "SkillPart"},
	"SkillType/skillTypeList":           {"type": "SkillType"},
	"Limit/limitVar":                    {"type": "Limit"},
}

var tfProbe = map[string]any{
	"actor": "parent", "base": mustNumber("7"), "div": mustNumber("3"),
	"divVar": "V", "equals": true, "floor": true,
	"includeTransfigured": true, "invert": true, "limit": mustNumber("9"),
	"limitActor": "parent", "limitNegTotal": true, "limitStat": "Str",
	"limitTotal": true, "limitVar": "V", "modFlags": mustNumber("0"),
	"neg": true, "noFloor": true, "percent": mustNumber("5"), "percentVar": "V",
	"ramp":          []any{[]any{mustNumber("1"), mustNumber("2")}},
	"skillId":       "Cleave",
	"skillName":     "Cleave",
	"skillNameList": []any{"Cleave"},
	"skillPart":     mustNumber("1"),
	"skillPartList": []any{mustNumber("1")},
	"skillType":     mustNumber("1"),
	"skillTypeList": []any{mustNumber("1")},
	"stat":          "Str", "statList": []any{"Str", "Dex"},
	"summonSkill": true, "threshold": mustNumber("4"),
	"thresholdActor": "parent", "thresholdPercent": mustNumber("50"),
	"thresholdPercentVar": "V", "thresholdStat": "Dex", "thresholdVar": "V",
	"upper": true, "var": "X", "varList": []any{"X", "Y"},
}

var tfSkillTypes = map[string]any{"1": "Attack"}

var tfArms = []struct {
	name string
	emit func(map[string]any) (string, error)
}{
	{"skill-base-mods/sbmEmitTag", func(t map[string]any) (string, error) {
		return sbmEmitTag(t, tfSkillTypes, nil)
	}},
	{"skill-stat-map/ssmRun.emitTag", func(t map[string]any) (string, error) {
		return (&ssmRun{skillTypes: tfSkillTypes, vocab: &mcVocab{}}).emitTag(t)
	}},
}

func tfWithout(base map[string]any, field string) map[string]any {
	out := make(map[string]any, len(base))
	for k, v := range base {
		if k != field {
			out[k] = v
		}
	}
	return out
}

func tfCopy(base map[string]any, field string, value any) map[string]any {
	out := make(map[string]any, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out[field] = value
	return out
}

// TestNoTagFieldIsSilentlyDropped is the whole sweep.
func TestNoTagFieldIsSilentlyDropped(t *testing.T) {
	types := make([]string, 0, len(modStoreTagFields))
	for tt := range modStoreTagFields {
		types = append(types, tt)
	}
	sort.Strings(types)

	for _, arm := range tfArms {
		for _, tt := range types {
			base, ok := tfBase[tt]
			if !ok {
				t.Fatalf("%s: tag type %s has no probe base", arm.name, tt)
			}
			for _, field := range modStoreTagFields[tt] {
				key := tt + "/" + field
				b := base
				if alt, ok := tfAltBase[key]; ok {
					b = alt
				}
				if _, present := b[field]; present {
					b = tfWithout(b, field)
				}
				probe, ok := tfProbe[field]
				if !ok {
					t.Fatalf("%s: no probe value for %s", arm.name, field)
				}

				t.Run(arm.name+"/"+key, func(t *testing.T) {
					baseOut, baseErr := arm.emit(b)
					gotOut, gotErr := arm.emit(tfCopy(b, field, probe))

					if gotErr != nil {
						return
					}
					if baseErr == nil && gotOut == baseOut {
						t.Fatalf("SILENT DROP: %s carrying %s=%s emits byte-identical Go source\n"+
							"  %s\n"+
							"pob:src/Classes/ModStore.lua reads this field; the arm must emit it or "+
							"refuse with mcUnrep",
							tt, field, fmt.Sprint(probe), gotOut)
					}
				})
			}
		}
	}
}
