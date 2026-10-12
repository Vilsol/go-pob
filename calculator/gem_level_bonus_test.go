package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
)

// TestSocketedGemLevelBonusIsApplied pins resolveGemInstance's contract and CalcSetup.lua:1049-1067.
//
//	if baseItemType := poe.BaseItemTypeByIDMap[gemInstance.GemID]; baseItemType != nil {
//		if sg := baseItemType.SkillGem(); sg != nil {
func TestSocketedGemLevelBonusIsApplied(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0001-necromancer-unknown.xml")

	var hoa *ActiveSkill
	for _, as := range env.Player.ActiveSkillList {
		if as.ActiveEffect != nil && as.ActiveEffect.GrantedEffect != nil &&
			as.ActiveEffect.GrantedEffect.Raw.ID == "HeraldOfAgony" {
			hoa = as
			break
		}
	}
	if hoa == nil {
		t.Fatal("premise broken: no HeraldOfAgony skill on this build. If every skill now reads " +
			"Fireball, the undecoded SkillGem.GrantedEffect column has been re-entered — see " +
			"the note above, this is the exact trap.")
	}
	ae := hoa.ActiveEffect

	if ae.GemData == nil {
		t.Fatal("GemData is nil for a socketed gem; resolveGemInstance found the SkillGem and " +
			"discarded it, and env.go guards the whole GemProperty loop on this being non-nil")
	}

	if ae.SrcInstance == nil {
		t.Fatal("premise broken: no SrcInstance, so the raw XML level is unavailable")
	}
	if ae.Level != 34 {
		t.Errorf("effective gem level = %d, want 34 (PoB's own value; raw XML level is %d)",
			ae.Level, ae.SrcInstance.Level)
	}

	for _, as := range env.Player.ActiveSkillList {
		if as.ActiveEffect == nil || as.ActiveEffect.GrantedEffect == nil {
			continue
		}
		if id := as.ActiveEffect.GrantedEffect.Raw.ID; id == "Fireball" {
			t.Fatalf("a skill resolved to %q — SkillGem.GrantedEffect is undecoded and answers "+
				"Fireball for all 847 gems, so gemData must never pick the granted effect", id)
		}
	}
}

// TestGemIsTypeMatchesPoBArms covers the arms of CalcTools.lua:155-167 that TestGemIsTypeFoldsCase does not reach.
func TestGemIsTypeMatchesPoBArms(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	gemOf := func(id string) *poe.SkillGem {
		base := poe.BaseItemTypeByIDMap[id]
		if base == nil {
			t.Fatalf("premise broken: no base item type %q", id)
		}
		gem := base.SkillGem()
		if gem == nil {
			t.Fatalf("premise broken: %q is not a skill gem", id)
		}
		return gem
	}

	fireball := gemOf("Metadata/Items/Gems/SkillGemFireball")
	multistrike := gemOf("Metadata/Items/Gems/SupportGemMultistrike")

	for _, tc := range []struct {
		name string
		gem  *poe.SkillGem
		kw   string
		want bool
	}{
		{"active-skill alias: grants_active_skill", fireball, "grants_active_skill", true},
		{"active-skill alias: skill", fireball, "skill", true},
		{"a support is not an active skill", multistrike, "active skill", false},
		{"a support is not an active skill (underscore)", multistrike, "active_skill", false},
		{"support tag still matches directly", multistrike, "support", true},
		{"non-vaal is true for a non-vaal gem", fireball, "non-vaal", true},
		{"non-exceptional is true for a normal gem", fireball, "non-exceptional", true},
		{"gem name matches", fireball, "fireball", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CalcGemIsType(tc.gem, tc.kw); got != tc.want {
				t.Errorf("CalcGemIsType(%q) = %v, want %v", tc.kw, got, tc.want)
			}
		})
	}
}
