package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

// TestStatMapGroupsAreEmitted is the producer half of PoB's GROUP form.
func TestStatMapGroupsAreEmitted(t *testing.T) {
	rejuv := raw.SkillStatMaps["RejuvenationTotem"]["base_life_regeneration_rate_per_minute"]
	if rejuv == nil {
		t.Fatal("RejuvenationTotem has no base_life_regeneration_rate_per_minute entry")
	}
	if len(rejuv.Groups) != 2 {
		t.Fatalf("RejuvenationTotem entry has %d groups, want 2", len(rejuv.Groups))
	}
	for i, want := range []struct {
		name string
		div  float64
	}{{"LifeRegen", 60}, {"ManaRegen", 400}} {
		g := rejuv.Groups[i]
		if len(g.Mods) != 1 || g.Mods[0].Name() != want.name {
			t.Errorf("group %d holds %d mods, want one named %q", i, len(g.Mods), want.name)
			continue
		}
		if g.Div == nil || *g.Div != want.div {
			t.Errorf("group %d Div = %v, want %v", i, g.Div, want.div)
		}
	}

	if rejuv.Div != nil || rejuv.Mult != nil || rejuv.Value != nil || rejuv.Base != nil {
		t.Errorf("the entry itself carries a scalar (div=%v mult=%v value=%v base=%v); PoB puts "+
			"them on the GROUPS, and 60 and 400 cannot both be one entry Div",
			rejuv.Div, rejuv.Mult, rejuv.Value, rejuv.Base)
	}

	rav := raw.SkillStatMaps["Ravenous"]["ravenous_buff_magnitude"]
	if rav == nil {
		t.Fatal("Ravenous has no ravenous_buff_magnitude entry")
	}
	if len(rav.Groups) != 2 {
		t.Fatalf("Ravenous entry has %d groups, want 2", len(rav.Groups))
	}
	if rav.Groups[0].Mult == nil || *rav.Groups[0].Mult != -1 {
		t.Errorf("Ravenous group 0 Mult = %v, want -1", rav.Groups[0].Mult)
	}
	if rav.Groups[1].Mult != nil {
		t.Errorf("Ravenous group 1 Mult = %v, want unset — the source writes no scalar on it",
			rav.Groups[1].Mult)
	}

	entries, groups, mixed := 0, 0, 0
	for _, statMap := range raw.SkillStatMaps {
		for _, e := range statMap {
			if len(e.Groups) == 0 {
				continue
			}
			entries++
			groups += len(e.Groups)
			if len(e.Mods) != 0 {
				mixed++
			}
			for _, g := range e.Groups {
				if len(g.Groups) != 0 {
					t.Errorf("a group carries nested groups; PoB's inner loop at :70 reads a "+
						"group's elements as mods and would drop them (%d)", len(g.Groups))
				}
				if len(g.Mods) == 0 {
					t.Error("an emitted group has no mods; its scalars apply to nothing")
				}
			}
		}
	}
	if entries != 2 || groups != 4 {
		t.Errorf("%d entries carry %d groups; the pin has 2 entries and 4 groups "+
			"(act_str.lua:9328, other.lua:3375)", entries, groups)
	}
	if mixed != 0 {
		t.Errorf("%d entries mix bare mods with groups; neither pin entry does, and Mods/Groups "+
			"being two lists assumes it", mixed)
	}
}

// TestStatMapGroupScalarIsPerGroup is the CONSUMER half: the arithmetic each group's mods get.
func TestStatMapGroupScalarIsPerGroup(t *testing.T) {
	entry := &raw.StatMap{
		Mods: []mod.Mod{},
		Groups: []*raw.StatMap{
			{Mods: []mod.Mod{mod.NewFloat("LifeRegen", mod.TypeBase, 0)}, Div: utils.Ptr(float64(60))},
			{Mods: []mod.Mod{mod.NewFloat("ManaRegen", mod.TypeBase, 0)}, Div: utils.Ptr(float64(400))},
			{Mods: []mod.Mod{mod.NewFloat("Damage", mod.TypeMore, 0)}, Mult: utils.Ptr(float64(-1))},
			{Mods: []mod.Mod{mod.NewFloat("DamageTaken", mod.TypeMore, 0)}},
		},
	}

	list := moddb.NewModList()
	mergeStatMapEntry(list, entry, 1200)

	got := map[string]float64{}
	for _, m := range list.Mods() {
		got[m.Name()] = m.Value().Float()
	}
	for _, c := range []struct {
		name string
		want float64
	}{
		{"LifeRegen", 20},
		{"ManaRegen", 3},
		{"Damage", -1200},
		{"DamageTaken", 1200},
	} {
		if got[c.name] != c.want {
			t.Errorf("%s = %v, want %v — each group's own value/mult/div/base applies to the mods "+
				"inside it (CalcActiveSkill.lua:71), not the entry's", c.name, got[c.name], c.want)
		}
	}
}
