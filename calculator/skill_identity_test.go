package calculator

import (
	"os"
	"testing"

	raw2 "github.com/Vilsol/go-pob/data/raw"
)

// TestMainSkillIdentity pins the single largest defect in the port: EVERY skill in EVERY build was resolving to Fireball.
func TestMainSkillIdentity(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build   string
		skillID string
	}{
		{"0045-hierophant-spark-of-the-nova.xml", "SparkAltX"},
		{"0011-saboteur-cyclone.xml", "RollingMagma"},
		{"0052-juggernaut-cleave.xml", "Cleave"},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			ms := env.Player.MainSkill
			if ms == nil || ms.ActiveEffect == nil || ms.ActiveEffect.GrantedEffect == nil {
				t.Fatal("no main skill resolved at all")
			}
			if got := ms.ActiveEffect.GrantedEffect.ID(); got != tc.skillID {
				t.Errorf("main skill granted effect = %q, want %q (the build's own mainSocketGroup "+
					"gem). %q means the gem resolved through the broken GrantedEffectsKey column.",
					got, tc.skillID, got)
			}
		})
	}
}

// TestSupportsAreNotActiveSkills is the structural half of the same defect, and needs no oracle.
func TestSupportsAreNotActiveSkills(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0045-hierophant-spark-of-the-nova.xml")
	if len(env.Player.ActiveSkillList) == 0 {
		t.Fatal("REFUSING: no active skills; this build has six gems in its main group")
	}
	fromSecondary := 0
	for _, as := range env.Player.ActiveSkillList {
		if as.ActiveEffect == nil || as.ActiveEffect.GrantedEffect == nil || as.ActiveEffect.GrantedEffect.Raw == nil {
			continue
		}
		if as.ActiveEffect.GrantedEffect.Raw.IsSupport {
			t.Errorf("support %q is in ActiveSkillList", as.ActiveEffect.GrantedEffect.ID())
		}
		if src := as.ActiveEffect.SrcInstance; src != nil && src.SkillID != "" {
			got := as.ActiveEffect.GrantedEffect.ID()
			secondary := raw2.Gems[src.GemID].SecondaryGrantedEffectID
			switch got {
			case src.SkillID:
			case secondary:
				fromSecondary++
			default:
				t.Errorf("gem %q (skillId %q, secondary %q) resolved to granted effect %q",
					src.NameSpec, src.SkillID, secondary, got)
			}
		}
	}

	if fromSecondary == 0 {
		t.Fatalf("REFUSING: no active skill came from a gem's SECOND granted effect; this build " +
			"sockets Vaal Molten Shell, so the secondary arm is no longer being exercised")
	}
}
