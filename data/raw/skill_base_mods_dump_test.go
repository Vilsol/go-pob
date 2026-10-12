package raw

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestDumpSkillBaseMods is the generator cross-check: it serialises the compiled SkillBaseMods table so it can be diffed against a fresh parse of PoB's Lua.
func TestDumpSkillBaseMods(t *testing.T) {
	path := os.Getenv("SBM_DUMP")
	if path == "" {
		t.Skip("SBM_DUMP not set")
	}

	out := map[string][]map[string]any{}
	for id, mods := range SkillBaseMods {
		rows := make([]map[string]any, 0, len(mods))
		for _, m := range mods {
			row := map[string]any{
				"name":   m.Name(),
				"type":   string(m.Type()),
				"source": string(m.GetSource()),
				"flags":  int(m.Flags()),
				"kw":     int(m.KeywordFlags()),
			}
			switch m.Value().Type() {
			case mod.ModValueMultiTypeFloat:
				row["value"] = m.Value().Float()
			case mod.ModValueMultiTypeFlag:
				row["value"] = m.Value().Flag()
			case mod.ModValueMultiTypeList:
				sd, ok := m.Value().List().(mod.SkillData)
				if !ok {
					t.Fatalf("%s: unexpected list value %#v", id, m.Value().List())
				}
				row["value"] = map[string]any{"key": sd.Key, "value": sd.Value}
			default:
				t.Fatalf("%s: unexpected value type", id)
			}
			tags := make([]map[string]any, 0, len(m.Tags()))
			for _, raw := range m.Tags() {
				tag := map[string]any{"type": string(raw.Type())}
				switch tt := raw.(type) {
				case *mod.SkillPartTag:
					tag["part"] = tt.Part
					tag["partList"] = tt.PartList
					tag["neg"] = tt.Negative
				case *mod.ConditionTag:
					tag["vars"] = tt.VarList
					tag["neg"] = tt.Negative
				case *mod.ActorConditionTag:
					tag["actor"] = *tt.Actor
					tag["vars"] = tt.VariableList
					tag["neg"] = tt.Negative
				case *mod.MultiplierTag:
					tag["vars"] = tt.VariableList
					tag["base"] = tt.TagBase
					tag["div"] = tt.Division
					tag["limit"] = tt.TagLimit
					tag["limitVar"] = tt.TagLimitVariable
					tag["actor"] = tt.TagActor
				case *mod.MultiplierThresholdTag:
					tag["var"] = tt.Variable
					tag["threshold"] = tt.TagThreshold
				case *mod.PerStatTag:
					tag["stats"] = tt.StatList
					tag["div"] = tt.Divide
					tag["base"] = tt.Base
				case *mod.SkillTypeTag:
					tag["skillTypes"] = tt.SkillTypeList
				case *mod.GlobalEffectTag:
					tag["effects"] = tt.GlobalEffectList
					tag["name"] = tt.NameTag
					tag["unscalable"] = tt.UnscalableTag
				default:
					t.Fatalf("%s: unexpected tag %#v", id, raw)
				}
				tags = append(tags, tag)
			}
			row["tags"] = tags
			rows = append(rows, row)
		}
		out[id] = rows
	}

	blob, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatal(err)
	}
}
