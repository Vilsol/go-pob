package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/pob"
)

// TestGetActiveSkillCountMatchesTheGemsOwnEffects covers Calcs.lua:150-174.
func TestGetActiveSkillCountMatchesTheGemsOwnEffects(t *testing.T) {
	requireGameData(t, "3.29")

	skill := func(gemID, skillID string, count int, g1, g2 bool) *ActiveSkill {
		gem := pob.Gem{
			GemID:         gemID,
			SkillID:       skillID,
			Count:         &count,
			EnableGlobal1: g1,
			EnableGlobal2: g2,
			Enabled:       true,
		}
		gemData, _ := resolveGemInstance(gem)
		return &ActiveSkill{
			SocketGroup: &pob.Skill{},
			ActiveEffect: &GemEffect{
				SrcInstance: &gem,
				GemData:     gemData,
			},
		}
	}

	effectOf := func(id string) *poe.GrantedEffect {
		e := poe.GrantedEffectByID(id)
		if e == nil {
			t.Fatalf("granted effect %q is absent from the loaded data -- this test's population "+
				"has emptied and it is no longer measuring anything", id)
		}
		return e
	}

	cases := []struct {
		name      string
		gemID     string
		skillID   string
		effectID  string
		count     int
		g1, g2    bool
		wantCount float64
		wantOK    bool
		why       string
	}{
		{
			name: "vaal half is gated by enableGlobal1", gemID: "Metadata/Items/Gems/SkillGemVaalFireball",
			skillID: "VaalFireball", effectID: "VaalFireball", count: 3, g1: false, g2: true,
			wantCount: 3, wantOK: false,
			why: "grantedEffectList[1] is the Vaal skill and is gated on enableGlobal1 (Calcs.lua:160-161)",
		},
		{
			name: "base half is gated by enableGlobal2", gemID: "Metadata/Items/Gems/SkillGemVaalFireball",
			skillID: "VaalFireball", effectID: "Fireball", count: 3, g1: true, g2: false,
			wantCount: 3, wantOK: false,
			why: "grantedEffectList[2] is the ordinary skill and is gated on enableGlobal2 (Calcs.lua:162-163)",
		},
		{
			name: "a non-vaal gem's SECONDARY effect still carries the count", gemID: "Metadata/Items/Gems/SkillGemAbyssalCry",
			skillID: "InfernalCry", effectID: "Combust", count: 2, g1: true, g2: true,
			wantCount: 2, wantOK: true,
			why: "Calcs.lua:166's second disjunct, `ge == gemData.secondaryGrantedEffect`, has no enableGlobal gate",
		},
		{
			name: "a non-vaal gem's PRIMARY effect carries the count", gemID: "Metadata/Items/Gems/SkillGemAbyssalCry",
			skillID: "InfernalCry", effectID: "InfernalCry", count: 2, g1: true, g2: true,
			wantCount: 2, wantOK: true,
			why: "Calcs.lua:166's first disjunct",
		},
		{
			name: "the Fireball control", gemID: "Metadata/Items/Gems/SkillGemFireball",
			skillID: "Fireball", effectID: "Fireball", count: 4, g1: true, g2: true,
			wantCount: 4, wantOK: true,
			why: "the one gem the undecoded-FK path resolved correctly by accident",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			as := skill(c.gemID, c.skillID, c.count, c.g1, c.g2)
			as.ActiveEffect.GrantedEffect = &GrantedEffect{Raw: effectOf(c.effectID)}
			if as.ActiveEffect.GemData == nil {
				t.Fatalf("%s resolved no SkillGem -- the fixture, not the code, is broken", c.gemID)
			}

			gotCount, gotOK := getActiveSkillCount(as)
			if gotCount != c.wantCount || gotOK != c.wantOK {
				t.Errorf("getActiveSkillCount = (%v, %v), want (%v, %v) -- %s",
					gotCount, gotOK, c.wantCount, c.wantOK, c.why)
			}
		})
	}
}
