package mod

import (
	"math/bits"
	"testing"
)

// TestMFlagNamesDoNotShareABit: The MFlag namespace is go-pob's own.
func TestMFlagNamesDoNotShareABit(t *testing.T) {
	masks := map[string]bool{"MFlagSourceMask": true, "MFlagWeaponMask": true}

	single := map[string]MFlag{
		"MFlagAttack": MFlagAttack, "MFlagSpell": MFlagSpell, "MFlagHit": MFlagHit,
		"MFlagDot": MFlagDot, "MFlagCast": MFlagCast,
		"MFlagMelee": MFlagMelee, "MFlagArea": MFlagArea, "MFlagProjectile": MFlagProjectile,
		"MFlagAilment": MFlagAilment, "MFlagMeleeHit": MFlagMeleeHit, "MFlagWeapon": MFlagWeapon,
		"MFlagAxe": MFlagAxe, "MFlagBow": MFlagBow, "MFlagClaw": MFlagClaw,
		"MFlagDagger": MFlagDagger, "MFlagMace": MFlagMace, "MFlagStaff": MFlagStaff,
		"MFlagSword": MFlagSword, "MFlagWand": MFlagWand, "MFlagUnarmed": MFlagUnarmed,
		"MFlagFishing": MFlagFishing, "MFlagWeaponMelee": MFlagWeaponMelee,
		"MFlagWeaponRanged": MFlagWeaponRanged, "MFlagWeapon1H": MFlagWeapon1H,
		"MFlagWeapon2H": MFlagWeapon2H,
	}

	owner := map[MFlag]string{}
	for name, v := range single {
		if masks[name] {
			continue
		}
		if v == 0 {
			t.Errorf("%s is 0 — a zero flag matches nothing and is masked by every test", name)
			continue
		}
		if n := bits.OnesCount64(mflagBits(v)); n != 1 {
			t.Errorf("%s = %#08x holds %d bits; every non-mask flag must hold exactly one", name, mflagBits(v), n)
			continue
		}
		if prev, ok := owner[v]; ok {
			t.Errorf("BIT COLLISION: %s and %s are both %#08x — `flags & %s` is true for a %s",
				prev, name, mflagBits(v), prev, name)
			continue
		}
		owner[v] = name
	}

	for v, name := range owner {
		if v&MFlagWeaponMask == 0 && v >= MFlagAxe {
			t.Errorf("%s = %#08x is a weapon bit outside MFlagWeaponMask %#08x",
				name, mflagBits(v), mflagBits(MFlagWeaponMask))
		}
	}
	if t.Failed() {
		t.Logf("distinct bits claimed: %d of %d names", len(owner), len(single))
	}
}

func mflagBits(f MFlag) uint64 {
	if f < 0 {
		return 0
	}
	return uint64(f)
}
