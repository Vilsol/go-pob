package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

// TestSocketGroupSourceIsATruthyTest pins the READ half of the `<Skill source>` split.
func TestSocketGroupSourceIsATruthyTest(t *testing.T) {
	for _, tc := range []struct {
		name       string
		sourceAttr *string
		wantSource bool
		why        string
	}{
		{
			name: "absent is unsourced", sourceAttr: nil, wantSource: false,
			why: "an ordinary socket group -- PoB's `not socketGroup.source` is true",
		},
		{
			name: "explicit empty is SOURCED", sourceAttr: strPtr(""), wantSource: true,
			why: "`\"\"` is TRUTHY in Lua, so CalcSetup.lua:1212 excludes this group",
		},
		{
			name: "a real source is sourced", sourceAttr: strPtr("Item:6:Soulwrest, Ezomyte Staff"),
			wantSource: true,
			why:        "the item-granted case the attribute exists for",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			group := &pob.Skill{Enabled: true, Slot: "Helmet", SourceAttr: tc.sourceAttr}
			if got := socketGroupIsSourced(group); got != tc.wantSource {
				t.Errorf("socketGroupIsSourced(SourceAttr=%v) = %v, want %v (%s)",
					quoteAttr(tc.sourceAttr), got, tc.wantSource, tc.why)
			}
		})
	}
}

// TestUpdateSocketCountsExcludesAnEmptySourcedGroup is the behavioural half.
func TestUpdateSocketCountsExcludesAnEmptySourcedGroup(t *testing.T) {
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		name       string
		sourceAttr *string
		wantEmpty  float64
		wantGems   float64
	}{
		{name: "absent counts its gems", sourceAttr: nil, wantEmpty: 1, wantGems: 2},
		{name: "explicit empty is excluded", sourceAttr: strPtr(""), wantEmpty: 3, wantGems: 0},
		{name: "a real source is excluded", sourceAttr: strPtr("Item:6:Soulwrest"), wantEmpty: 3, wantGems: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gem := pob.Gem{
				NameSpec: "Spark", Enabled: true, Level: 20, Quality: 0,
				SkillID: "Spark", GemID: "Metadata/Items/Gems/SkillGemSpark",
			}
			group := pob.Skill{
				Enabled:    true,
				Slot:       "Helmet",
				SourceAttr: tc.sourceAttr,
				Gems:       []pob.Gem{gem, gem},
			}
			if tc.sourceAttr != nil {
				group.Source = *tc.sourceAttr
			}
			env := &Environment{
				ItemModDB: moddb.NewModDB(),
				Build: &pob.PathOfBuilding{
					Skills: pob.Skills{
						ActiveSkillSet: 1,
						SkillSets:      []pob.SkillSet{{ID: 1, Skills: []pob.Skill{group}}},
					},
				},
			}
			item := &Item{Sockets: []ItemSocket{{Color: "B"}, {Color: "B"}, {Color: "B"}}}

			updateSocketCounts(env, item, "Helmet")

			m := env.ItemModDB.Multipliers
			if got := m["EmptySocketInHelmet"]; got != tc.wantEmpty {
				t.Errorf("EmptySocketInHelmet = %v, want %v — the group was %s",
					got, tc.wantEmpty, branchWord(tc.wantEmpty == 3))
			}
			if got := m["SocketedGemsInHelmet"]; got != tc.wantGems {
				t.Errorf("SocketedGemsInHelmet = %v, want %v", got, tc.wantGems)
			}
		})
	}
}

// TestCursePriorityTreatsEmptySourceAsUnsourced is the NEGATIVE control for the fix above.
func TestCursePriorityTreatsEmptySourceAsUnsourced(t *testing.T) {
	withSource := func(attr *string) float64 {
		group := &pob.Skill{Enabled: true, Slot: "Helmet", SourceAttr: attr}
		if attr != nil {
			group.Source = *attr
		}
		return determineCursePriority("Despair", &ActiveSkill{SocketGroup: group})
	}

	absent := withSource(nil)
	empty := withSource(strPtr(""))
	realSource := withSource(strPtr("Item:32:Lost Unity, Formless Ring"))

	if absent != empty {
		t.Errorf("determineCursePriority: absent=%v empty=%v — CalcPerform.lua:569 coalesces "+
			"source to \"\" before :585's `source ~= \"\"`, so these two MUST agree; this site "+
			"is not part of the truthy-presence fix", absent, empty)
	}
	if realSource == empty {
		t.Errorf("determineCursePriority: a real source scored %v, the same as an empty one — "+
			"CurseFromEquipment at :586 never fired, so this control is vacuous", realSource)
	}
}

func strPtr(s string) *string { return &s }

func quoteAttr(s *string) string {
	if s == nil {
		return "nil"
	}
	return `"` + *s + `"`
}

func branchWord(excluded bool) string {
	if excluded {
		return "excluded (PoB counts it as item-granted)"
	}
	return "counted (an ordinary socket group)"
}
