package raw

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestDumpSkillStatMaps is the generator cross-check.
func TestDumpSkillStatMaps(t *testing.T) {
	path := os.Getenv("SSM_DUMP")
	if path == "" {
		t.Skip("SSM_DUMP not set")
	}

	out := map[string]map[string]any{}
	for id, entries := range SkillStatMaps {
		perSkill := map[string]any{}
		for stat, entry := range entries {
			rows := make([]map[string]any, 0, len(entry.Mods))
			for _, m := range entry.Mods {
				rows = append(rows, dumpMod(t, id+"/"+stat, m))
			}
			perSkill[stat] = map[string]any{
				"mods":  rows,
				"value": entry.Value,
				"mult":  entry.Mult,
				"div":   entry.Div,
				"base":  entry.Base,
			}
		}
		out[id] = perSkill
	}

	blob, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatal(err)
	}
}

func dumpMod(t *testing.T, where string, m mod.Mod) map[string]any {
	row := map[string]any{
		"name":  m.Name(),
		"type":  string(m.Type()),
		"flags": int(m.Flags()),
		"kw":    int(m.KeywordFlags()),
	}
	switch m.Value().Type() {
	case mod.ModValueMultiTypeFloat:
		row["value"] = m.Value().Float()
	case mod.ModValueMultiTypeFlag:
		row["value"] = m.Value().Flag()
	case mod.ModValueMultiTypeList:
		switch v := m.Value().List().(type) {
		case mod.SkillData:
			row["value"] = map[string]any{"key": v.Key, "value": v.Value}
		case mod.ExtraMinionSkill:
			row["value"] = map[string]any{"skillId": v.SkillID}
		case mod.SupportedGemProperty:
			row["value"] = map[string]any{"keyword": v.Keyword, "propKey": v.Key, "value": v.Value}
		case mod.MinionModifier:
			row["value"] = map[string]any{"mod": dumpMod(t, where, v.Mod)}
		case mod.PactDamage:
			row["value"] = map[string]any{"mod": dumpMod(t, where, v.Mod)}
		default:
			t.Fatalf("%s: unexpected list value %#v", where, m.Value().List())
		}
	default:
		t.Fatalf("%s: unexpected value type", where)
	}
	row["tags"] = dumpTags(t, where, m.Tags())
	return row
}

func dumpTags(t *testing.T, where string, raws []mod.Tag) []map[string]any {
	tags := make([]map[string]any, 0, len(raws))
	for _, raw := range raws {
		tag := map[string]any{"type": string(raw.Type())}
		switch tt := raw.(type) {
		case *mod.GlobalEffectTag:
			tag["effects"] = tt.GlobalEffectList
			tag["name"] = tt.NameTag
			tag["unscalable"] = tt.UnscalableTag
		case *mod.MultiplierTag:
			tag["vars"] = tt.VariableList
			tag["div"] = tt.Division
			tag["base"] = tt.TagBase
			tag["limit"] = tt.TagLimit
			tag["limitVar"] = tt.TagLimitVariable
			tag["limitTotal"] = tt.TagLimitTotal
			tag["actor"] = tt.TagActor
		case *mod.MultiplierThresholdTag:
			tag["var"] = tt.Variable
			tag["threshold"] = tt.TagThreshold
			tag["upper"] = tt.TagUpper
			tag["actor"] = tt.TagActor
		case *mod.PerStatTag:
			tag["stats"] = tt.StatList
			tag["div"] = tt.Divide
			tag["base"] = tt.Base
			tag["limit"] = tt.TagLimit
			tag["limitVar"] = tt.TagLimitVariable
			tag["limitTotal"] = tt.TagLimitTotal
			tag["actor"] = tt.TagActor
		case *mod.PercentStatTag:
			tag["stat"] = tt.Stat
			tag["percent"] = tt.Percent
			tag["percentVar"] = tt.PercentVar
		case *mod.ConditionTag:
			tag["vars"] = tt.VarList
			tag["neg"] = tt.Negative
		case *mod.ActorConditionTag:
			tag["actor"] = *tt.Actor
			tag["vars"] = tt.VariableList
			tag["neg"] = tt.Negative
		case *mod.SkillTypeTag:
			tag["skillTypes"] = tt.SkillTypeList
			tag["neg"] = tt.Negative
		case *mod.SkillNameTag:
			tag["names"] = tt.SkillNameList
			tag["neg"] = tt.Negative
		case *mod.SkillPartTag:
			tag["part"] = tt.Part
			tag["partList"] = tt.PartList
			tag["neg"] = tt.Negative
		case *mod.StatThresholdTag:
			tag["stat"] = tt.Stat
			tag["threshold"] = tt.Threshold
			tag["upper"] = tt.TagUpper
		case *mod.DistanceRampTag:
			tag["ramp"] = tt.Ramp
		case *mod.MeleeProximityTag:
			tag["ramp"] = tt.Ramp
		case *mod.ModFlagOrTag:
			tag["modFlags"] = int(tt.Flag)
		case *mod.SkillIDTag:
			tag["skillId"] = tt.ID()
		case *mod.LimitTag:
			tag["limit"] = tt.TagLimit
			tag["limitVar"] = tt.LimitVar
		default:
			t.Fatalf("%s: unexpected tag %#v", where, raw)
		}
		tags = append(tags, tag)
	}
	return tags
}
