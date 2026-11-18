package calculator

import "testing"

// TestKalandraInRingThreeAddsOnlyExtraSkill: CalcSetup.lua:1222 takes the Kalandra's Touch branch in any slot. In
// Ring 3 nothing mirrors (the other-ring lookup is items[false]), and only ExtraSkill mods apply, so a "+50 to maximum
// Life" line on it does nothing. The want is the pinned PoB's Life on the edited build.
//
//pob:block:src/Modules/CalcSetup.lua:1222-1260
func TestKalandraInRingThreeAddsOnlyExtraSkill(t *testing.T) {
	env := corpusEnvWithEdit(t, "0016-gladiator-dual-strike-of-ambidexterity",
		"Woespike\nSteel Ring\nUnique ID: b5a901f007e0e82a8cb4269dcf8d005c862e1882e7e5f4d54879fc0cecd86dec\nCatalyst: Intrinsic\nCatalystQuality: 20\nItem Level: 87\nLevelReq: 80\nImplicits: 1\nAdds 4 to 12 Physical Damage to Attacks\n+33 to Strength and Dexterity\nAdds 10 to 24 Physical Damage to Attacks\n15% chance to Impale Enemies on Hit with Attacks\n30% chance on Melee Hit for the Strongest Impale on target to last for 1 additional Hit\n42% less Impale Duration\n",
		"Kalandra&apos;s Touch\nRing\nImplicits: 0\n+50 to maximum Life\nMirrored\n")
	if got, want := env.Player.Output["Life"], 5459.0; got != want {
		t.Errorf("Life = %v, PoB %v", got, want)
	}
}
