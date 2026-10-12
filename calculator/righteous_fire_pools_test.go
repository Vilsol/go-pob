package calculator

import "testing"

// TestRighteousFirePoolsMatchThePin pins act_int.lua:15428-15433, Righteous Fire's preDamageFunc.
//
//	if activeSkill.skillFlags.totem and output.TotemLife > 1 then
//	elseif output.LifeUnreserved > 1 then
//	end
func TestRighteousFirePoolsMatchThePin(t *testing.T) {
	fn := skillFuncsFor("RighteousFire")["preDamageFunc"]
	if fn == nil {
		t.Fatal("REFUSING: RighteousFire has no preDamageFunc registered; every case below would " +
			"trivially see an untouched FireDot and read as a pass")
	}

	const (
		rfLife = 0.7
		rfES   = 0.7
	)

	run := func(t *testing.T, totem bool, priorFireDot float64, output map[string]float64) float64 {
		t.Helper()
		skill := &ActiveSkill{
			SkillFlags: map[SkillFlag]bool{SkillFlagTotem: totem},
			SkillData: &SkillData{
				FireDot:          priorFireDot,
				RFLifeMultiplier: rfLife,
				RFESMultiplier:   rfES,
			},
		}
		fn(skill, output, nil)
		return skill.SkillData.FireDot
	}

	t.Run("fully reserved life writes nothing", func(t *testing.T) {
		got := run(t, false, 0, map[string]float64{
			"Life": 4000, "EnergyShield": 2000, "LifeUnreserved": 0,
		})
		if got != 0 {
			t.Fatalf("LifeUnreserved 0: FireDot = %v, want 0 — the pin's `elseif output.LifeUnreserved > 1` "+
				"leaves it untouched, so this is a burn on a build upstream reports dark", got)
		}
	})

	t.Run("unreserved life burns both pools", func(t *testing.T) {
		got := run(t, false, 0, map[string]float64{
			"Life": 4000, "EnergyShield": 2000, "LifeUnreserved": 3500,
		})
		if want := 4000*rfLife + 2000*rfES; got != want {
			t.Fatalf("FireDot = %v, want %v", got, want)
		}
	})

	t.Run("totem arm includes TotemEnergyShield", func(t *testing.T) {
		got := run(t, true, 0, map[string]float64{
			"TotemLife": 5000, "TotemEnergyShield": 800,
			"Life": 4000, "EnergyShield": 2000, "LifeUnreserved": 3500,
		})
		if want := 5000*rfLife + 800*rfES; got != want {
			t.Fatalf("FireDot = %v, want %v — :15430 adds output.TotemEnergyShield * RFESMultiplier "+
				"and this tree dropped the whole term", got, want)
		}
	})

	t.Run("totem with no TotemLife falls through to the player pools", func(t *testing.T) {
		got := run(t, true, 0, map[string]float64{
			"TotemLife": 0, "TotemEnergyShield": 0,
			"Life": 4000, "EnergyShield": 2000, "LifeUnreserved": 3500,
		})
		if want := 4000*rfLife + 2000*rfES; got != want {
			t.Fatalf("FireDot = %v, want %v — `elseif` falls through when TotemLife <= 1; an `if/else` "+
				"on the flag alone strands the skill on a zero", got, want)
		}
	})

	t.Run("assigns rather than accumulates", func(t *testing.T) {
		got := run(t, false, 1234, map[string]float64{
			"Life": 4000, "EnergyShield": 2000, "LifeUnreserved": 3500,
		})
		if want := 4000*rfLife + 2000*rfES; got != want {
			t.Fatalf("FireDot = %v, want %v — :15432 is `=`, not `+=`; a prior value must be "+
				"OVERWRITTEN or a second pass compounds the burn", got, want)
		}
	})

	t.Run("control: a met threshold still writes over a prior value", func(t *testing.T) {
		got := run(t, false, 1234, map[string]float64{
			"Life": 100, "EnergyShield": 0, "LifeUnreserved": 2,
		})
		if got != 100*rfLife {
			t.Fatalf("FireDot = %v, want %v — the write is gated, not removed", got, 100*rfLife)
		}
	})
}
