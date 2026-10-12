package calculator

import (
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob/pob"
)

// TestGetActiveSkillCountSeparatesAbsentFromZero pins the READ half of `<Gem count>`.
func TestGetActiveSkillCountSeparatesAbsentFromZero(t *testing.T) {
	requireGameData(t, "3.29")

	const gemID = "Metadata/Items/Gems/SkillGemSpark"
	baseItem := poe.BaseItemTypeByIDMap[gemID]
	if baseItem == nil || baseItem.SkillGem() == nil {
		t.Fatalf("REFUSING: %s has no SkillGem in the loaded data — this fixture cannot reach "+
			"the count branch at all", gemID)
	}
	gemData := baseItem.SkillGem()
	if gemData.IsVaalGem {
		t.Fatalf("REFUSING: %s reports IsVaalGem, which routes to the enableGlobal arms instead "+
			"of the plain count arm this test is about", gemID)
	}
	granted := poe.GrantedEffectByID("Spark")
	if granted == nil {
		t.Fatal("REFUSING: no granted effect for Spark in the loaded data")
	}

	for _, tc := range []struct {
		name  string
		count *int
		want  float64
		why   string
	}{
		{name: "absent counts once", count: nil, want: 1,
			why: "`tonumber(nil) or 1` — this is 198960 of the corpus's 250539 gems"},
		{name: "explicit zero counts NONE", count: intPtr(0), want: 0,
			why: "Lua's 0 is truthy, so `or 1` never fires; 120 gems over 18 corpus builds"},
		{name: "an explicit count is carried", count: intPtr(3), want: 3,
			why: "the ordinary case the attribute exists for"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gem := &pob.Gem{
				NameSpec: "Spark", Enabled: true, Level: 20,
				SkillID: "Spark", GemID: gemID, Count: tc.count,
			}
			activeSkill := &ActiveSkill{
				SocketGroup: &pob.Skill{Enabled: true, Gems: []pob.Gem{*gem}},
				ActiveEffect: &GemEffect{
					SrcInstance:   gem,
					GemData:       gemData,
					GrantedEffect: &GrantedEffect{Raw: granted},
				},
			}

			got, enabled := getActiveSkillCount(activeSkill)
			if got != tc.want {
				t.Errorf("getActiveSkillCount(count=%s) = %v, want %v (%s)",
					intAttr(tc.count), got, tc.want, tc.why)
			}
			if !enabled {
				t.Errorf("getActiveSkillCount(count=%s) reported the skill DISABLED — the "+
					"fixture fell through to a Vaal/enableGlobal arm and proves nothing about count",
					intAttr(tc.count))
			}
		})
	}
}

func intAttr(i *int) string {
	if i == nil {
		return "absent"
	}
	return strconv.Itoa(*i)
}
