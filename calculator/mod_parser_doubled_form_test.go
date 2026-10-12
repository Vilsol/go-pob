package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func doubledShape(t *testing.T, line, name string) {
	t.Helper()

	entry := ParseMod(line, false)
	if entry.Extra != "" {
		t.Errorf("%q left %q unparsed", line, entry.Extra)
		return
	}
	if len(entry.ModList) != 2 {
		t.Errorf("%q gave %d mods, want 2 (ModParser.lua:6791-6806 emits a MORE and an OVERRIDE)", line, len(entry.ModList))
		return
	}

	more := entry.ModList[0]
	if more.Name() != name || more.Type() != mod.TypeMore || more.Value().Float() != 100 {
		t.Errorf("%q mod 1 = %s %s %v, want %s MORE 100", line, more.Name(), more.Type(), more.Value(), name)
	}
	var found *mod.MultiplierTag
	for _, tag := range more.Tags() {
		if m, ok := tag.(*mod.MultiplierTag); ok {
			found = m
		}
	}
	switch {
	case found == nil:
		t.Errorf("%q mod 1 carries no Multiplier tag; the global limit is then unreachable", line)
	case len(found.VariableList) != 1 || found.VariableList[0] != name+"Doubled":
		t.Errorf("%q Multiplier var %q, want %q", line, found.VariableList, name+"Doubled")
	case found.TagGlobalLimit == nil || *found.TagGlobalLimit != 100:
		t.Errorf("%q Multiplier globalLimit %v, want 100", line, found.TagGlobalLimit)
	case found.TagGlobalLimitKey == nil || *found.TagGlobalLimitKey != name+"DoubledLimit":
		t.Errorf("%q Multiplier globalLimitKey %v, want %q", line, found.TagGlobalLimitKey, name+"DoubledLimit")
	}

	over := entry.ModList[1]
	if over.Name() != "Multiplier:"+name+"Doubled" || over.Type() != mod.TypeOverride || over.Value().Float() != 1 {
		t.Errorf("%q mod 2 = %s %s %v, want Multiplier:%sDoubled OVERRIDE 1", line, over.Name(), over.Type(), over.Value(), name)
	}
	if len(over.Tags()) != 0 {
		t.Errorf("%q mod 2 carries %d tags; PoB gives the OVERRIDE none (modTagList[2] is nil)", line, len(over.Tags()))
	}
}

// TestDoubledFormExpandsAnyModName covers ModParser.lua:155-157 + :6791-6806, the DOUBLED form.
func TestDoubledFormExpandsAnyModName(t *testing.T) {
	requireGameData(t, "3.29")

	doubledShape(t, "Maximum number of Summoned Golems is Doubled", "ActiveGolemLimit")

	for _, c := range [][2]string{
		{"Maximum number of Summoned Skeletons is Doubled", "ActiveSkeletonLimit"},
		{"Maximum number of Raised Zombies is Doubled", "ActiveZombieLimit"},
		{"Maximum number of Raised Spiders is Doubled", "ActiveSpiderLimit"},
		{"Maximum number of Animated Weapons is Doubled", "ActiveAnimatedWeaponLimit"},
		{"Maximum number of Sentinels of Purity is Doubled", "ActiveSentinelOfPurityLimit"},
		{"Maximum number of Summoned Raging Spirits is Doubled", "ActiveRagingSpiritLimit"},
		{"Maximum number of Summoned Spectral Wolves is Doubled", "ActiveWolfLimit"},
		{"Maximum number of Living Lightning is Doubled", "ActiveLivingLightningLimit"},
		{"Maximum number of Holy Armaments is Doubled", "ActiveHolyStrikeMinionLimit"},
	} {
		doubledShape(t, c[0], c[1])
	}
}
