package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/mod"
)

// TestFarrulPresenceRowsBuildTheirExtraAuras: ModParser.lua:5087-5101 is a family of THREE and go-pob carried NONE of them.
func TestFarrulPresenceRowsBuildTheirExtraAuras(t *testing.T) {
	for _, tc := range []struct {
		line string
		want []struct {
			name  string
			typ   mod.Type
			value float64
		}
	}{
		{
			line: "Raised Beast Spectres have Farrul's Farric Presence",
			want: []struct {
				name  string
				typ   mod.Type
				value float64
			}{
				{"Accuracy", mod.TypeIncrease, 80},
				{"CritChance", mod.TypeIncrease, 120},
				{"ReduceCritExtraDamage", mod.TypeBase, 100},
			},
		},
		{
			line: "Raised Beast Spectres have Farrul's Fertile Presence",
			want: []struct {
				name  string
				typ   mod.Type
				value float64
			}{
				{"Damage", mod.TypeIncrease, 100},
				{"LifeRegenPercent", mod.TypeBase, 3},
				{"StunImmune", mod.TypeFlag, 0},
			},
		},
		{
			line: "Raised Beast Spectres have Farrul's Wild Presence",
			want: []struct {
				name  string
				typ   mod.Type
				value float64
			}{
				{"Speed", mod.TypeIncrease, 20},
				{"MovementSpeed", mod.TypeIncrease, 20},
				{"MinimumActionSpeed", mod.TypeMAX, 100},
			},
		},
	} {
		mods, extra := parseMod(tc.line, 1)
		testza.AssertEqual(t, "", extra, "%q must parse completely — an unparsed remainder discards the whole line", tc.line)
		testza.AssertEqual(t, len(tc.want), len(mods), "%q must produce one ExtraAura per granted mod", tc.line)
		if len(mods) != len(tc.want) {
			continue
		}

		for i, want := range tc.want {
			m := mods[i]
			testza.AssertEqual(t, "ExtraAura", m.Name(), "%q mod %d", tc.line, i)
			testza.AssertEqual(t, mod.TypeList, m.Type(), "%q mod %d", tc.line, i)

			aura, isAura := m.Value().List().(mod.ExtraAura)
			testza.AssertTrue(t, isAura, "%q mod %d is not an ExtraAura, got %T", tc.line, i, m.Value().List())
			if !isAura {
				continue
			}

			testza.AssertFalse(t, aura.OnlyAllies, "%q mod %d must reach the player too", tc.line, i)

			testza.AssertEqual(t, want.name, aura.Mod.Name(), "%q mod %d", tc.line, i)
			testza.AssertEqual(t, want.typ, aura.Mod.Type(), "%q mod %d", tc.line, i)
			if want.typ != mod.TypeFlag {
				testza.AssertEqual(t, want.value, aura.Mod.Value().Float(), "%q mod %d", tc.line, i)
			} else {
				testza.AssertTrue(t, aura.Mod.Value().Flag(), "%q mod %d is a FLAG", tc.line, i)
			}

			testza.AssertEqual(t, 1, len(m.Tags()), "%q mod %d must carry exactly its condition", tc.line, i)
			for _, tag := range m.Tags() {
				cond, isCond := tag.(*mod.ConditionTag)
				testza.AssertTrue(t, isCond, "%q mod %d: tag %T is not a condition", tc.line, i, tag)
				if isCond {
					testza.AssertEqual(t, []string{"HaveBeastSpectre"}, cond.VarList, "%q mod %d", tc.line, i)
				}
			}
		}
	}
}

// TestProfaneGroundNeighbourIsNotSwallowed: The NEIGHBOUR.
func TestProfaneGroundNeighbourIsNotSwallowed(t *testing.T) {
	mods, extra := parseMod("Profane Ground you create also affects you and your allies, granting Chaotic Might", 1)
	testza.AssertEqual(t, "", extra)
	testza.AssertEqual(t, 1, len(mods))
	if len(mods) != 1 {
		return
	}
	aura, isAura := mods[0].Value().List().(mod.ExtraAura)
	testza.AssertTrue(t, isAura)
	if isAura {
		testza.AssertEqual(t, "Condition:ChaoticMight", aura.Mod.Name())
	}
}
