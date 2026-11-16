package mcp

// The non-unique item estimate: the base's poe.ninja price at the item's own level (unlistedBaseChaos when it has none); a magic item adds
// magicItemChaos; a rare adds each explicit affix by its tier on the base, each bench-crafted mod's bench cost and the
// extras below. Tune the estimate here and nowhere else.
const (
	tier1Chaos     = 20.0
	tier2Chaos     = 5.0
	tierLowerChaos = 1.0
	// an explicit mod the base's affix table cannot place (influence, essence, veiled, delve and other special mods)
	// costs as a T1 affix
	unplacedAffixChaos = tier1Chaos
)

// a weapon's local physical, elemental or chaos damage affix (increased physical, added damage, the hybrids; "dmg" in
// detail) is where a weapon's value lies, so it is priced above the flat tier table
const (
	weaponDamageTier1Chaos = 100.0
	weaponDamageTier2Chaos = 20.0
)

func weaponDamageTierChaos(tier int) float64 {
	switch tier {
	case 1:
		return weaponDamageTier1Chaos
	case 2:
		return weaponDamageTier2Chaos
	default:
		return tierLowerChaos
	}
}

func affixTierChaos(tier int) float64 {
	switch tier {
	case 1:
		return tier1Chaos
	case 2:
		return tier2Chaos
	default:
		return tierLowerChaos
	}
}

const (
	// poe.ninja's base list holds crafting bases at item level 82 to 86 only
	ninjaBaseMinLevel = 82
	// a base poe.ninja does not price at the item's own level (no Item Level line, below ninjaBaseMinLevel, or a base
	// it does not list) is a drop-grade base, worth nothing beside its mods
	unlistedBaseChaos = 0.0
	// a magic item is priced as its base plus this, never by its affixes' tiers; a normal item is its base
	magicItemChaos = 1.0
	// an anoint (an "Allocates" enchant): the oils are not priced, so every anoint costs this
	anointChaos = 10.0
	// each level an explicit "+N to Level of all ... Skill Gems" line grants, on top of its affix tier; an item with two
	// or more such lines doubles this bucket, since the combination is far rarer than either line
	gemLevelChaos = 50.0
)

// eldritchTierChaos is a Searing Exarch or Eater of Worlds implicit by its tier, 1 (Lesser) to 6 (Perfect): the tier is
// the implicit mod's number, at the best tier whose range holds the roll.
var eldritchTierChaos = [7]float64{0, 0, 1, 3, 10, 30, 80}
