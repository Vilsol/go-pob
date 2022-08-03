package mod

type Mod interface {
	Name() string
	Type() Type
	Source(source Source) Mod
	Flag(flag MFlag) Mod
	KeywordFlag(keywordFlags KeywordFlag) Mod
	Tag(tag Tag) Mod
	Flags() MFlag
	KeywordFlags() KeywordFlag
	GetSource() Source
	Tags() []Tag
	Value() interface{}
	Clone() Mod
}

type MFlag int

const (
	// Damage modes

	MFlagAttack = MFlag(0x00000001)
	MFlagSpell  = MFlag(0x00000002)
	MFlagHit    = MFlag(0x00000004)
	MFlagDot    = MFlag(0x00000008)
	MFlagCast   = MFlag(0x00000010)

	// Damage sources

	MFlagMelee      = MFlag(0x00000100)
	MFlagArea       = MFlag(0x00000200)
	MFlagProjectile = MFlag(0x00000400)
	MFlagSourceMask = MFlag(0x00000600)
	MFlagAilment    = MFlag(0x00000800)
	MFlagMeleeHit   = MFlag(0x00001000)
	MFlagWeapon     = MFlag(0x00002000)

	// Weapon types

	MFlagAxe     = MFlag(0x00010000)
	MFlagBow     = MFlag(0x00020000)
	MFlagClaw    = MFlag(0x00040000)
	MFlagDagger  = MFlag(0x00080000)
	MFlagMace    = MFlag(0x00100000)
	MFlagStaff   = MFlag(0x00200000)
	MFlagSword   = MFlag(0x00400000)
	MFlagWand    = MFlag(0x00800000)
	MFlagUnarmed = MFlag(0x01000000)
	MFlagFishing = MFlag(0x02000000)

	// Weapon classes

	MFlagWeaponMelee  = MFlag(0x02000000)
	MFlagWeaponRanged = MFlag(0x04000000)
	MFlagWeapon1H     = MFlag(0x08000000)
	MFlagWeapon2H     = MFlag(0x10000000)
	MFlagWeaponMask   = MFlag(0x1FFF0000)
)
