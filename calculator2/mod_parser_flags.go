package calculator2

import "github.com/Vilsol/go-pob/mod"

// PoE2 moves the weapon-class bits above 32 and adds weapon and Thorns bits, so mod.MFlag* (PoE1) cannot be used here.
//
//pob2:block:src/Data/Global.lua:231-268
const (
	ModFlagAttack       mod.MFlag = 0x0000000000000001
	ModFlagSpell        mod.MFlag = 0x0000000000000002
	ModFlagHit          mod.MFlag = 0x0000000000000004
	ModFlagDot          mod.MFlag = 0x0000000000000008
	ModFlagCast         mod.MFlag = 0x0000000000000010
	ModFlagThorns       mod.MFlag = 0x0000000000000020
	ModFlagMelee        mod.MFlag = 0x0000000000000100
	ModFlagArea         mod.MFlag = 0x0000000000000200
	ModFlagProjectile   mod.MFlag = 0x0000000000000400
	ModFlagSourceMask   mod.MFlag = 0x0000000000000600
	ModFlagAilment      mod.MFlag = 0x0000000000000800
	ModFlagMeleeHit     mod.MFlag = 0x0000000000001000
	ModFlagWeapon       mod.MFlag = 0x0000000000002000
	ModFlagAxe          mod.MFlag = 0x0000000000010000
	ModFlagBow          mod.MFlag = 0x0000000000020000
	ModFlagClaw         mod.MFlag = 0x0000000000040000
	ModFlagDagger       mod.MFlag = 0x0000000000080000
	ModFlagMace         mod.MFlag = 0x0000000000100000
	ModFlagStaff        mod.MFlag = 0x0000000000200000
	ModFlagSword        mod.MFlag = 0x0000000000400000
	ModFlagWand         mod.MFlag = 0x0000000000800000
	ModFlagUnarmed      mod.MFlag = 0x0000000001000000
	ModFlagFishing      mod.MFlag = 0x0000000002000000
	ModFlagCrossbow     mod.MFlag = 0x0000000004000000
	ModFlagFlail        mod.MFlag = 0x0000000008000000
	ModFlagSpear        mod.MFlag = 0x0000000010000000
	ModFlagWarstaff     mod.MFlag = 0x0000000020000000
	ModFlagTalisman     mod.MFlag = 0x0000000040000000
	ModFlagWeaponMelee  mod.MFlag = 0x0000000100000000
	ModFlagWeaponRanged mod.MFlag = 0x0000000200000000
	ModFlagWeapon1H     mod.MFlag = 0x0000000400000000
	ModFlagWeapon2H     mod.MFlag = 0x0000000800000000
	ModFlagWeaponMask   mod.MFlag = 0x0000000F5FFF0000
)

//pob2:block:src/Data/Global.lua:270-306
const (
	KeywordFlagAura         mod.KeywordFlag = 0x00000001
	KeywordFlagCurse        mod.KeywordFlag = 0x00000002
	KeywordFlagWarcry       mod.KeywordFlag = 0x00000004
	KeywordFlagMovement     mod.KeywordFlag = 0x00000008
	KeywordFlagPhysical     mod.KeywordFlag = 0x00000010
	KeywordFlagFire         mod.KeywordFlag = 0x00000020
	KeywordFlagCold         mod.KeywordFlag = 0x00000040
	KeywordFlagLightning    mod.KeywordFlag = 0x00000080
	KeywordFlagChaos        mod.KeywordFlag = 0x00000100
	KeywordFlagVaal         mod.KeywordFlag = 0x00000200
	KeywordFlagBow          mod.KeywordFlag = 0x00000400
	KeywordFlagArrow        mod.KeywordFlag = 0x00000800
	KeywordFlagTrap         mod.KeywordFlag = 0x00001000
	KeywordFlagMine         mod.KeywordFlag = 0x00002000
	KeywordFlagTotem        mod.KeywordFlag = 0x00004000
	KeywordFlagMinion       mod.KeywordFlag = 0x00008000
	KeywordFlagAttack       mod.KeywordFlag = 0x00010000
	KeywordFlagSpell        mod.KeywordFlag = 0x00020000
	KeywordFlagHit          mod.KeywordFlag = 0x00040000
	KeywordFlagAilment      mod.KeywordFlag = 0x00080000
	KeywordFlagBrand        mod.KeywordFlag = 0x00100000
	KeywordFlagPoison       mod.KeywordFlag = 0x00200000
	KeywordFlagBleed        mod.KeywordFlag = 0x00400000
	KeywordFlagIgnite       mod.KeywordFlag = 0x00800000
	KeywordFlagPhysicalDot  mod.KeywordFlag = 0x01000000
	KeywordFlagLightningDot mod.KeywordFlag = 0x02000000
	KeywordFlagColdDot      mod.KeywordFlag = 0x04000000
	KeywordFlagFireDot      mod.KeywordFlag = 0x08000000
	KeywordFlagChaosDot     mod.KeywordFlag = 0x10000000
	KeywordFlagMatchAll     mod.KeywordFlag = 0x40000000
)
