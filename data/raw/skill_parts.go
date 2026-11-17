package raw

// SkillPart is one entry of PoB's hand-written per-skill `parts` table
// (src/Data/Skills/*.lua). PoB keys a part's flag overrides by skill-flag name in the same
// table as `name`, and CalcActiveSkill.lua:206-212 walks it setting the flag for `true` and
// CLEARING it for `false`; Flags keeps the two apart because a Go map read cannot.
type SkillPart struct {
	Name  string
	Flags map[string]bool
}

// SkillParts mirrors `grantedEffect.parts`, keyed by granted-effect id. It exists for the
// same reason as SkillBaseFlags: the table is hand-written PoB data that go-pob-data does
// not ship. PoB reads it at src/Modules/CalcActiveSkill.lua:196 to pick the part the build
// selected (`<Gem skillPart="N">`), apply that part's flag overrides, and set the
// `multiPart` flag. Flag names are PoB's raw strings — data/raw cannot import calculator,
// whose SkillFlag is a string type holding exactly these values.
//
// Generated from PoB v2.18.1 by oracle/skill_parts.py.

var SkillParts = map[string][]SkillPart{
	// src/Data/Skills/act_int.lua:1078
	"BallLightning": {
		{Name: "One Bolt"},
		{Name: "All Bolts in Range"},
	},
	// src/Data/Skills/act_int.lua:1276
	"BallLightningAltX": {
		{Name: "One Bolt"},
		{Name: "Half Bolts Hitting"},
		{Name: "All Bolts Hitting"},
	},
	// src/Data/Skills/act_dex.lua:1018
	"Barrage": {
		{Name: "1 Projectile"},
		{Name: "All Projectiles"},
	},
	// src/Data/Skills/act_dex.lua:1106
	"BarrageAltX": {
		{Name: "1 Projectile"},
		{Name: "All Projectiles"},
	},
	// src/Data/Skills/act_dex.lua:1364
	"BladeBlast": {
		{Name: "Blade Hits Per Cast", Flags: map[string]bool{"stages": true}},
		{Name: "Blade Hits Per Sec", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:1456
	"BladeBlastAltX": {
		{Name: "Blade Hits Per Cast", Flags: map[string]bool{"stages": true}},
		{Name: "Blade Hits Per Sec", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:1896
	"BladeFlurry": {
		{Name: "Channelling", Flags: map[string]bool{"stages": true}},
		{Name: "Channel & Release", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:2013
	"BladeFlurryAltX": {
		{Name: "Channelling", Flags: map[string]bool{"stages": true}},
		{Name: "Channel & Release", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:2212
	"BladeVortex": {
		{Name: "0 Blades"},
		{Name: "5 Blades"},
		{Name: "10 Blades"},
	},
	// src/Data/Skills/act_str.lua:1021
	"Bladestorm": {
		{Name: "Attack"},
		{Name: "Bladestorm"},
	},
	// src/Data/Skills/act_str.lua:1129
	"BladestormAltX": {
		{Name: "Attack"},
		{Name: "Bladestorm"},
	},
	// src/Data/Skills/act_dex.lua:2887
	"BlastRain": {
		{Name: "1 Projectile"},
		{Name: "All Projectiles"},
	},
	// src/Data/Skills/act_int.lua:1815
	"BlazingSalvo": {
		{Name: "1 Projectile"},
		{Name: "All Projectiles"},
	},
	// src/Data/Skills/act_int.lua:1907
	"Blight": {
		{Name: "Manual Stacks", Flags: map[string]bool{"stages": true}},
		{Name: "Maximum Sustainable Stacks"},
	},
	// src/Data/Skills/act_int.lua:2003
	"BlightAltX": {
		{Name: "Manual Stacks", Flags: map[string]bool{"stages": true}},
		{Name: "Maximum Sustainable Stacks"},
	},
	// src/Data/Skills/act_int.lua:2099
	"BlightAltY": {
		{Name: "Manual Stacks", Flags: map[string]bool{"stages": true}},
		{Name: "Maximum Sustainable Stacks"},
	},
	// src/Data/Skills/act_int.lua:2286
	"Bodyswap": {
		{Name: "Self Explosion", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_int.lua:2401
	"BodyswapAltX": {
		{Name: "Self Explosion", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Minion Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_str.lua:1315
	"Boneshatter": {
		{Name: "Attack", Flags: map[string]bool{"area": false}},
		{Name: "Pulse", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_str.lua:1425
	"BoneshatterAltX": {
		{Name: "Attack", Flags: map[string]bool{"area": false}},
		{Name: "Pulse", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_str.lua:1539
	"BoneshatterAltY": {
		{Name: "Attack", Flags: map[string]bool{"area": false}},
		{Name: "Pulse", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:3929
	"ChargedDash": {
		{Name: "Channelling, No Stages"},
		{Name: "Channelling, Max Stages"},
		{Name: "Release", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:4054
	"ChargedDashAltX": {
		{Name: "Channelling, No Stages"},
		{Name: "Channelling, Max Stages"},
		{Name: "Release", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:4251
	"Conflagration": {
		{Name: "Projectile"},
		{Name: "Explosion"},
	},
	// src/Data/Skills/act_dex.lua:4362
	"Cremation": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true, "projectile": false}},
	},
	// src/Data/Skills/act_dex.lua:4480
	"CremationAltX": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true, "projectile": false}},
	},
	// src/Data/Skills/act_int.lua:3882
	"DarkPact": {
		{Name: "Cast on Player"},
		{Name: "Cast on Skeleton"},
	},
	// src/Data/Skills/act_int.lua:3999
	"DarkPactAltX": {
		{Name: "Average"},
		{Name: "Max Ruin"},
	},
	// src/Data/Skills/other.lua:1270
	"DeathWish": {
		{Name: "Channelling", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Minion Explosion", Flags: map[string]bool{"spell": false, "cast": true, "stages": true}},
	},
	// src/Data/Skills/act_dex.lua:5173
	"DetonateDead": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_dex.lua:5341
	"DetonateDeadAltY": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_str.lua:2811
	"DivineBlast": {
		{Name: "Beam Implosion"},
		{Name: "Explosion"},
	},
	// src/Data/Skills/act_str.lua:2910
	"DivineBlastAltX": {
		{Name: "Beam Implosion"},
		{Name: "Explosion"},
	},
	// src/Data/Skills/act_int.lua:4608
	"DivineIre": {
		{Name: "Channelling", Flags: map[string]bool{"area": false}},
		{Name: "Release", Flags: map[string]bool{"area": true, "stages": true, "channelRelease": true}},
	},
	// src/Data/Skills/act_int.lua:4718
	"DivineIreAltX": {
		{Name: "Channelling", Flags: map[string]bool{"area": false}},
		{Name: "Release", Flags: map[string]bool{"area": true, "stages": true, "channelRelease": true}},
	},
	// src/Data/Skills/act_str.lua:3367
	"Earthquake": {
		{Name: "Initial impact"},
		{Name: "Aftershock"},
	},
	// src/Data/Skills/act_str.lua:3469
	"EarthquakeAltX": {
		{Name: "Initial impact"},
		{Name: "Aftershock"},
	},
	// src/Data/Skills/act_str.lua:3680
	"Earthshatter": {
		{Name: "Initial impact"},
		{Name: "Spike"},
	},
	// src/Data/Skills/act_str.lua:3791
	"EarthshatterAltX": {
		{Name: "Initial impact"},
		{Name: "Spike"},
	},
	// src/Data/Skills/act_str.lua:3899
	"EarthshatterAltY": {
		{Name: "Initial impact"},
		{Name: "Spike"},
	},
	// src/Data/Skills/act_dex.lua:6059
	"ElementalHit": {
		{Name: "Fire Attack", Flags: map[string]bool{"area": false}},
		{Name: "Fire AoE", Flags: map[string]bool{"area": true}},
		{Name: "Cold Attack", Flags: map[string]bool{"area": false}},
		{Name: "Cold AoE", Flags: map[string]bool{"area": true}},
		{Name: "Lightning Attack", Flags: map[string]bool{"area": false}},
		{Name: "Lightning AoE", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:6231
	"ElementalHitAltX": {
		{Name: "Attack", Flags: map[string]bool{"area": false}},
		{Name: "AoE Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/other.lua:1793
	"ElementalWarding": {
		{Name: "Convert to fire"},
		{Name: "Convert to cold"},
		{Name: "Convert to lightning"},
	},
	// src/Data/Skills/other.lua:6356
	"EnemyExplode": {
		{Name: "This Source Only"},
		{Name: "This Source Only, Ignoring Chance"},
		{Name: "Average Of All Sources"},
	},
	// src/Data/Skills/act_str.lua:4084
	"Eviscerate": {
		{Name: "1 Wave"},
		{Name: "Both Waves"},
	},
	// src/Data/Skills/act_dex.lua:6686
	"ExplosiveArrow": {
		{Name: "Explosion (# of fuses)", Flags: map[string]bool{"area": true, "stages": true}},
		{Name: "Explosion (Maximum Sustainable Fuses)", Flags: map[string]bool{"area": true}},
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
	},
	// src/Data/Skills/act_dex.lua:6898
	"ExplosiveConcoction": {
		{Name: "No Flasks"},
		{Name: "Sapphire"},
		{Name: "Topaz"},
		{Name: "Ruby"},
		{Name: "Sapphire + Topaz"},
		{Name: "Sapphire + Ruby"},
		{Name: "Topaz + Ruby"},
		{Name: "All Flasks"},
	},
	// src/Data/Skills/act_dex.lua:7024
	"ExplosiveConcoctionAltX": {
		{Name: "No Flasks"},
		{Name: "Sapphire"},
		{Name: "Topaz"},
		{Name: "Ruby"},
		{Name: "Sapphire + Topaz"},
		{Name: "Sapphire + Ruby"},
		{Name: "Topaz + Ruby"},
		{Name: "All Flasks"},
	},
	// src/Data/Skills/act_dex.lua:7147
	"ExplosiveTrap": {
		{Name: "One explosion hitting"},
		{Name: "Average explosions hitting"},
		{Name: "All explosions hitting"},
	},
	// src/Data/Skills/act_int.lua:5799
	"Fireball": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:6050
	"Firestorm": {
		{Name: "First Impact"},
		{Name: "Subsequent Impacts"},
	},
	// src/Data/Skills/act_int.lua:6569
	"FlameWall": {
		{Name: "Primary Debuff", Flags: map[string]bool{"area": true}},
		{Name: "Secondary Debuff", Flags: map[string]bool{"area": false}},
	},
	// src/Data/Skills/act_dex.lua:7679
	"FlamethrowerTrap": {
		{Name: "One trap (good placement)"},
		{Name: "One trap (bad placement)"},
		{Name: "Average # traps (good placement)"},
		{Name: "Average # traps (bad placement)"},
	},
	// src/Data/Skills/act_int.lua:7462
	"ForbiddenRite": {
		{Name: "1 Projectile"},
		{Name: "All Primary Projectiles"},
	},
	// src/Data/Skills/act_int.lua:7621
	"ForbiddenRiteAltX": {
		{Name: "1 Projectile"},
		{Name: "All Primary Projectiles"},
	},
	// src/Data/Skills/act_dex.lua:8393
	"FrostBlades": {
		{Name: "Melee hit", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Icy blades", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_dex.lua:8496
	"FrostBladesAltX": {
		{Name: "Melee Hit", Flags: map[string]bool{"attack": true, "melee": true, "projectile": true, "area": false}},
		{Name: "Ground DoT", Flags: map[string]bool{"attack": false, "hit": false, "melee": false, "projectile": false, "area": true}},
	},
	// src/Data/Skills/act_str.lua:4671
	"FrozenSweep": {
		{Name: "1 charge"},
		{Name: "Wait for max charges"},
	},
	// src/Data/Skills/act_str.lua:4789
	"FrozenSweepAltX": {
		{Name: "1 charge"},
		{Name: "Wait for max charges"},
	},
	// src/Data/Skills/act_dex.lua:8601
	"GalvanicArrow": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:8765
	"GalvanicArrowAltY": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:8626
	"GalvanicField": {
		{Name: "Single Target"},
		{Name: "Multi Target"},
	},
	// src/Data/Skills/act_int.lua:8746
	"GalvanicFieldAltX": {
		{Name: "Single Target"},
		{Name: "Multi Target"},
	},
	// src/Data/Skills/act_int.lua:8855
	"GlacialCascade": {
		{Name: "Initial Bursts"},
		{Name: "Final Burst"},
	},
	// src/Data/Skills/act_int.lua:9174
	"Hexblast": {
		{Name: "Target", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:9277
	"HexblastAltX": {
		{Name: "Target", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:9380
	"HexblastAltY": {
		{Name: "Target", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_str.lua:6103
	"HolyHammers": {
		{Name: "Slam"},
		{Name: "First Hammer"},
		{Name: "Other Hammers"},
	},
	// src/Data/Skills/act_int.lua:9029
	"Hydrosphere": {
		{Name: "Autopulse (Frozen)"},
		{Name: "Autopulse (Shocked)"},
		{Name: "Autopulse (Frozen & Shocked)"},
		{Name: "Cast (No Ailment)"},
		{Name: "Cast (Frozen)"},
		{Name: "Cast (Shocked)"},
		{Name: "Cast (Frozen & Shocked)"},
	},
	// src/Data/Skills/act_str.lua:6581
	"IceCrash": {
		{Name: "First Hit"},
		{Name: "Second Hit"},
		{Name: "Third Hit"},
	},
	// src/Data/Skills/act_dex.lua:9547
	"IceShot": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:9631
	"IceShotAltX": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:9890
	"IceSpear": {
		{Name: "First Form, 1 Projectile"},
		{Name: "Second Form, 1 Projectile"},
		{Name: "First Form, All Projectiles"},
		{Name: "Second Form, All Projectiles"},
	},
	// src/Data/Skills/act_int.lua:10000
	"IceSpearAltX": {
		{Name: "First Form, 1 Projectile"},
		{Name: "Second Form, 1 Projectile"},
		{Name: "First Form, All Projectiles"},
		{Name: "Second Form, All Projectiles"},
	},
	// src/Data/Skills/act_int.lua:10369
	"Incinerate": {
		{Name: "Channelling", Flags: map[string]bool{"stages": true}},
		{Name: "Release", Flags: map[string]bool{"stages": true, "channelRelease": true}},
	},
	// src/Data/Skills/act_int.lua:10507
	"IncinerateAltX": {
		{Name: "Channelling", Flags: map[string]bool{"stages": true}},
		{Name: "Release", Flags: map[string]bool{"stages": true, "channelRelease": true}},
	},
	// src/Data/Skills/act_str.lua:7124
	"InfernalBlow": {
		{Name: "Melee Hit", Flags: map[string]bool{"area": false}},
		{Name: "Debuff Explosion - 1 Stack", Flags: map[string]bool{"area": true}},
		{Name: "Debuff Explosion - 6 Stacks", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_str.lua:7250
	"InfernalBlowAltX": {
		{Name: "Melee Hit", Flags: map[string]bool{"area": false}},
		{Name: "Debuff Explosion - 1 Stack", Flags: map[string]bool{"area": true}},
		{Name: "Debuff Explosion - 6 Stacks", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:10756
	"KineticBlast": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosions", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:10842
	"KineticBlastAltX": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosions", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:11081
	"KineticFusillade": {
		{Name: "All Projectiles"},
		{Name: "1 Projectile"},
	},
	// src/Data/Skills/act_int.lua:11299
	"KineticFusilladeAltX": {
		{Name: "All Projectiles"},
		{Name: "1 Projectile"},
	},
	// src/Data/Skills/act_dex.lua:9965
	"Lacerate": {
		{Name: "One slash"},
		{Name: "Both slashes"},
	},
	// src/Data/Skills/act_dex.lua:10058
	"LacerateAltX": {
		{Name: "One slash"},
		{Name: "Both slashes"},
	},
	// src/Data/Skills/act_dex.lua:10146
	"LacerateAltY": {
		{Name: "One slash"},
		{Name: "Both slashes"},
	},
	// src/Data/Skills/act_dex.lua:10244
	"LancingSteel": {
		{Name: "Single Projectile Hit"},
		{Name: "All Projectiles Hit"},
	},
	// src/Data/Skills/act_dex.lua:10347
	"LancingSteelAltX": {
		{Name: "Single Projectile Hit"},
		{Name: "All Projectiles Hit"},
	},
	// src/Data/Skills/act_int.lua:11869
	"LightningSpireTrap": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_int.lua:12109
	"LightningSpireTrapAltX": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_int.lua:12341
	"LightningSpireTrapAltY": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_dex.lua:10699
	"LightningStrike": {
		{Name: "Melee hit", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Projectiles", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_dex.lua:10800
	"LightningStrikeAltX": {
		{Name: "Melee hit", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Projectiles", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_int.lua:12619
	"LightningTendrils": {
		{Name: "Average DPS"},
		{Name: "Normal pulse"},
		{Name: "Stronger pulse"},
	},
	// src/Data/Skills/act_int.lua:12743
	"LightningTendrilsAltX": {
		{Name: "Average DPS"},
		{Name: "Normal pulse"},
		{Name: "Stronger pulse"},
	},
	// src/Data/Skills/act_str.lua:7812
	"MoltenStrike": {
		{Name: "Melee Hit", Flags: map[string]bool{"melee": true, "projectile": false, "area": false}},
		{Name: "Magma Balls", Flags: map[string]bool{"melee": false, "projectile": true, "area": true}},
		{Name: "Average Balls Hitting", Flags: map[string]bool{"melee": false, "projectile": true, "area": true}},
	},
	// src/Data/Skills/act_str.lua:7973
	"MoltenStrikeAltX": {
		{Name: "Melee Hit", Flags: map[string]bool{"melee": true, "projectile": false, "area": false}},
		{Name: "Magma Balls", Flags: map[string]bool{"melee": false, "projectile": true, "area": true}},
		{Name: "Average Balls Hitting", Flags: map[string]bool{"melee": false, "projectile": true, "area": true}},
		{Name: "Magma Balls (5th attack)", Flags: map[string]bool{"melee": false, "projectile": true, "area": true}},
		{Name: "Average Balls (5th attack)", Flags: map[string]bool{"melee": false, "projectile": true, "area": true}},
		{Name: "Total Weighted Ball Average", Flags: map[string]bool{"melee": false, "projectile": true, "area": true}},
	},
	// src/Data/Skills/act_int.lua:13796
	"OrbOfStormsAltX": {
		{Name: "Self Cast"},
		{Name: "Channelling"},
	},
	// src/Data/Skills/act_int.lua:13987
	"PenanceBrandAltX": {
		{Name: "Energy Pulse", Flags: map[string]bool{"stages": true}},
		{Name: "Max Pulse per Brand"},
	},
	// src/Data/Skills/act_dex.lua:17418
	"PoisonousConcoction": {
		{Name: "No Flask"},
		{Name: "Life"},
	},
	// src/Data/Skills/act_dex.lua:17519
	"PoisonousConcoctionAltX": {
		{Name: "No Flask"},
		{Name: "Life"},
	},
	// src/Data/Skills/sup_int.lua:5414
	"PrismaticBurst": {
		{Name: "Fire"},
		{Name: "Cold"},
		{Name: "Lightning"},
	},
	// src/Data/Skills/act_int.lua:14442
	"PurifyingFlame": {
		{Name: "Initial hit"},
		{Name: "Shockwave"},
	},
	// src/Data/Skills/act_int.lua:14546
	"PurifyingFlameAltX": {
		{Name: "Initial hit"},
		{Name: "Shockwave"},
	},
	// src/Data/Skills/act_str.lua:9092
	"RageVortex": {
		{Name: "Melee"},
		{Name: "Rage Storm"},
	},
	// src/Data/Skills/act_str.lua:9207
	"RageVortexAltX": {
		{Name: "Melee"},
		{Name: "Rage Storm"},
	},
	// src/Data/Skills/act_int.lua:15805
	"ScorchingRay": {
		{Name: "Manual Stages", Flags: map[string]bool{"stages": true}},
		{Name: "Maximum Stages"},
	},
	// src/Data/Skills/act_int.lua:15907
	"ScorchingRayAltX": {
		{Name: "Manual Stages", Flags: map[string]bool{"stages": true}},
		{Name: "Maximum Stages"},
	},
	// src/Data/Skills/act_dex.lua:12995
	"ScourgeArrow": {
		{Name: "Release", Flags: map[string]bool{"stages": true}},
		{Name: "Thorn Arrows", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:13094
	"ScourgeArrowAltX": {
		{Name: "Release"},
		{Name: "Thorn Arrows"},
	},
	// src/Data/Skills/act_str.lua:9644
	"SearingBondAltX": {
		{Name: "Beam"},
		{Name: "Explosion"},
	},
	// src/Data/Skills/act_dex.lua:13499
	"SeismicTrap": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_dex.lua:13674
	"SeismicTrapAltX": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
	},
	// src/Data/Skills/act_dex.lua:13191
	"ShatteringSteel": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Cone AoE"},
	},
	// src/Data/Skills/act_dex.lua:13280
	"ShatteringSteelAltX": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Cone AoE"},
	},
	// src/Data/Skills/act_str.lua:9929
	"ShieldCrush": {
		{Name: "1 Wave"},
		{Name: "2 Overlapping Waves"},
	},
	// src/Data/Skills/act_str.lua:10023
	"ShieldCrushAltX": {
		{Name: "1 Wave"},
		{Name: "2 Overlapping Waves"},
	},
	// src/Data/Skills/act_int.lua:16005
	"ShockNova": {
		{Name: "Ring + Nova"},
		{Name: "Ring"},
		{Name: "Nova"},
	},
	// src/Data/Skills/act_int.lua:16098
	"ShockNovaAltX": {
		{Name: "Ring + Nova"},
		{Name: "Nova"},
	},
	// src/Data/Skills/act_str.lua:10377
	"Smite": {
		{Name: "Melee Hit", Flags: map[string]bool{"area": false}},
		{Name: "Area Hit", Flags: map[string]bool{"area": true, "melee": false}},
	},
	// src/Data/Skills/act_str.lua:10502
	"SmiteAltX": {
		{Name: "Melee Hit", Flags: map[string]bool{"area": false}},
		{Name: "Area Hit", Flags: map[string]bool{"area": true, "melee": false}},
	},
	// src/Data/Skills/act_int.lua:16464
	"SomaticShell": {
		{Name: "Projectile", Flags: map[string]bool{"projectile": true}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:16796
	"Spark": {
		{Name: "1 Hit"},
		{Name: "Maximum Hits"},
	},
	// src/Data/Skills/act_int.lua:16886
	"SparkAltX": {
		{Name: "1 Hit"},
		{Name: "Maximum Hits"},
	},
	// src/Data/Skills/act_int.lua:16977
	"SparkAltY": {
		{Name: "1 Hit"},
		{Name: "Maximum Hits"},
	},
	// src/Data/Skills/act_dex.lua:14308
	"SpectralShieldThrow": {
		{Name: "Shield"},
		{Name: "Shards"},
	},
	// src/Data/Skills/act_dex.lua:14408
	"SpectralShieldThrowAltX": {
		{Name: "Shield"},
		{Name: "Shards"},
	},
	// src/Data/Skills/act_dex.lua:14495
	"SpectralShieldThrowAltY": {
		{Name: "Shield"},
		{Name: "Shards"},
	},
	// src/Data/Skills/act_str.lua:10731
	"StaticStrike": {
		{Name: "Melee hit", Flags: map[string]bool{"chaining": false}},
		{Name: "Beams", Flags: map[string]bool{"chaining": true}},
	},
	// src/Data/Skills/act_int.lua:18108
	"StormBurst": {
		{Name: "1 Orb Tick"},
		{Name: "Max Channelled Orbs"},
		{Name: "Max Duration Explode"},
	},
	// src/Data/Skills/act_int.lua:18226
	"StormBurstAltX": {
		{Name: "1 Orb Tick"},
		{Name: "Max Duration Explode"},
	},
	// src/Data/Skills/act_dex.lua:11741
	"StormRain": {
		{Name: "Arrow"},
		{Name: "Beam"},
	},
	// src/Data/Skills/act_dex.lua:11859
	"StormRainAltX": {
		{Name: "Arrow"},
		{Name: "Beam"},
	},
	// src/Data/Skills/act_dex.lua:11967
	"StormRainAltY": {
		{Name: "Arrow"},
		{Name: "Beam"},
	},
	// src/Data/Skills/act_int.lua:17742
	"Stormbind": {
		{Name: "Unimproved"},
		{Name: "1 Improvement"},
		{Name: "2 Improvements"},
		{Name: "3 Improvements"},
	},
	// src/Data/Skills/act_int.lua:17924
	"StormbindAltX": {
		{Name: "Unimproved"},
		{Name: "1 Improvement"},
		{Name: "2 Improvements"},
		{Name: "3 Improvements"},
	},
	// src/Data/Skills/act_str.lua:11892
	"Sunder": {
		{Name: "First Wave Area"},
		{Name: "Final Wave Area"},
		{Name: "Shockwaves"},
	},
	// src/Data/Skills/act_str.lua:12003
	"SunderAltX": {
		{Name: "Wave"},
		{Name: "Shockwaves"},
	},
	// src/Data/Skills/act_str.lua:12105
	"SunderAltY": {
		{Name: "First Wave Area"},
		{Name: "Final Wave Area"},
		{Name: "Shockwaves"},
	},
	// src/Data/Skills/act_str.lua:6377
	"Sweep": {
		{Name: "Melee Hit"},
		{Name: "Holy Hammer"},
	},
	// src/Data/Skills/act_str.lua:6471
	"SweepAltX": {
		{Name: "Melee Hit"},
		{Name: "Holy Hammer"},
	},
	// src/Data/Skills/act_dex.lua:17845
	"Thunderstorm": {
		{Name: "Projectile"},
		{Name: "Storm Explosion", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:17952
	"ThunderstormMiniTornados": {
		{Name: "Projectile", Flags: map[string]bool{"projectile": true}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:18134
	"TornadoAltY": {
		{Name: "Fire", Flags: map[string]bool{"stages": true}},
		{Name: "Lightning", Flags: map[string]bool{"stages": true}},
		{Name: "Cold", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:16046
	"TornadoShot": {
		{Name: "Single Projectile"},
		{Name: "Combined Average Projectiles"},
	},
	// src/Data/Skills/sup_str.lua:4982
	"TriggeredSupportVoidShockwave": {
		{Name: "Maim Shockwave"},
		{Name: "Shockwave Repeat"},
	},
	// src/Data/Skills/sup_dex.lua:4917
	"TriggeredSupportWindburst": {
		{Name: "Projectile", Flags: map[string]bool{"projectile": true}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:5438
	"VaalDetonateDead": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_str.lua:3575
	"VaalEarthquake": {
		{Name: "Initial impact"},
		{Name: "Aftershock"},
	},
	// src/Data/Skills/act_int.lua:5962
	"VaalFireball": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:10584
	"VaalLightningArrow": {
		{Name: "1 Projectile"},
		{Name: "All Projectiles"},
	},
	// src/Data/Skills/act_dex.lua:10903
	"VaalLightningStrike": {
		{Name: "Melee hit", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Projectiles", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_str.lua:8190
	"VaalMoltenStrike": {
		{Name: "Melee Hit", Flags: map[string]bool{"melee": true, "projectile": false, "area": false, "chaining": false}},
		{Name: "Magma Balls", Flags: map[string]bool{"melee": false, "projectile": true, "area": true, "chaining": true}},
	},
	// src/Data/Skills/act_str.lua:10602
	"VaalSmite": {
		{Name: "Melee Hit", Flags: map[string]bool{"area": false}},
		{Name: "Area Hit", Flags: map[string]bool{"area": true, "melee": false}},
	},
	// src/Data/Skills/act_str.lua:12828
	"VaalVolcanicFissure": {
		{Name: "Ground Fissure", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Magma Balls", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_dex.lua:16296
	"VenomGyre": {
		{Name: "Outgoing Projectile"},
		{Name: "Returning Projectile"},
	},
	// src/Data/Skills/act_dex.lua:16660
	"VolatileDead": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_dex.lua:16763
	"VolatileDeadAltX": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_dex.lua:16867
	"VolatileDeadAltY": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_str.lua:12651
	"VolcanicFissure": {
		{Name: "Ground Fissure", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Magma Balls", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_dex.lua:17056
	"WildStrike": {
		{Name: "Fire hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Fire explosion", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": false, "area": true}},
		{Name: "Lightning hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Lightning bolt", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": true, "area": false}},
		{Name: "Cold hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Icy wave", Flags: map[string]bool{"melee": false, "projectile": true, "chaining": false, "area": false}},
	},
	// src/Data/Skills/act_dex.lua:17197
	"WildStrikeAltX": {
		{Name: "Fire hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Fire explosion", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": false, "area": true}},
		{Name: "Lightning hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Lightning bolt", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": true, "area": false}},
		{Name: "Cold hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Icy wave", Flags: map[string]bool{"melee": false, "projectile": true, "chaining": false, "area": false}},
	},
	// src/Data/Skills/act_int.lua:21056
	"WinterOrb": {
		{Name: "Channelling", Flags: map[string]bool{"stages": true}},
		{Name: "Idle", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_int.lua:21300
	"WintertideBrand": {
		{Name: "Average Damage"},
		{Name: "Manual Stages", Flags: map[string]bool{"stages": true}},
	},
}
