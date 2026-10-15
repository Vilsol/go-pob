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
	// src/Data/Skills/act_dex.lua:2885
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
	// src/Data/Skills/act_dex.lua:3927
	"ChargedDash": {
		{Name: "Channelling, No Stages"},
		{Name: "Channelling, Max Stages"},
		{Name: "Release", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:4052
	"ChargedDashAltX": {
		{Name: "Channelling, No Stages"},
		{Name: "Channelling, Max Stages"},
		{Name: "Release", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:4249
	"Conflagration": {
		{Name: "Projectile"},
		{Name: "Explosion"},
	},
	// src/Data/Skills/act_dex.lua:4360
	"Cremation": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true, "projectile": false}},
	},
	// src/Data/Skills/act_dex.lua:4478
	"CremationAltX": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true, "projectile": false}},
	},
	// src/Data/Skills/act_int.lua:3881
	"DarkPact": {
		{Name: "Cast on Player"},
		{Name: "Cast on Skeleton"},
	},
	// src/Data/Skills/act_int.lua:3998
	"DarkPactAltX": {
		{Name: "Average"},
		{Name: "Max Ruin"},
	},
	// src/Data/Skills/other.lua:1269
	"DeathWish": {
		{Name: "Channelling", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Minion Explosion", Flags: map[string]bool{"spell": false, "cast": true, "stages": true}},
	},
	// src/Data/Skills/act_dex.lua:5171
	"DetonateDead": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_dex.lua:5339
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
	// src/Data/Skills/act_int.lua:4607
	"DivineIre": {
		{Name: "Channelling", Flags: map[string]bool{"area": false}},
		{Name: "Release", Flags: map[string]bool{"area": true, "stages": true, "channelRelease": true}},
	},
	// src/Data/Skills/act_int.lua:4717
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
	// src/Data/Skills/act_dex.lua:6057
	"ElementalHit": {
		{Name: "Fire Attack", Flags: map[string]bool{"area": false}},
		{Name: "Fire AoE", Flags: map[string]bool{"area": true}},
		{Name: "Cold Attack", Flags: map[string]bool{"area": false}},
		{Name: "Cold AoE", Flags: map[string]bool{"area": true}},
		{Name: "Lightning Attack", Flags: map[string]bool{"area": false}},
		{Name: "Lightning AoE", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:6229
	"ElementalHitAltX": {
		{Name: "Attack", Flags: map[string]bool{"area": false}},
		{Name: "AoE Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/other.lua:1792
	"ElementalWarding": {
		{Name: "Convert to fire"},
		{Name: "Convert to cold"},
		{Name: "Convert to lightning"},
	},
	// src/Data/Skills/other.lua:6193
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
	// src/Data/Skills/act_dex.lua:6684
	"ExplosiveArrow": {
		{Name: "Explosion (# of fuses)", Flags: map[string]bool{"area": true, "stages": true}},
		{Name: "Explosion (Maximum Sustainable Fuses)", Flags: map[string]bool{"area": true}},
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
	},
	// src/Data/Skills/act_dex.lua:6896
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
	// src/Data/Skills/act_dex.lua:7022
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
	// src/Data/Skills/act_dex.lua:7145
	"ExplosiveTrap": {
		{Name: "One explosion hitting"},
		{Name: "Average explosions hitting"},
		{Name: "All explosions hitting"},
	},
	// src/Data/Skills/act_int.lua:5798
	"Fireball": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:5970
	"Firestorm": {
		{Name: "First Impact"},
		{Name: "Subsequent Impacts"},
	},
	// src/Data/Skills/act_int.lua:6489
	"FlameWall": {
		{Name: "Primary Debuff", Flags: map[string]bool{"area": true}},
		{Name: "Secondary Debuff", Flags: map[string]bool{"area": false}},
	},
	// src/Data/Skills/act_dex.lua:7677
	"FlamethrowerTrap": {
		{Name: "One trap (good placement)"},
		{Name: "One trap (bad placement)"},
		{Name: "Average # traps (good placement)"},
		{Name: "Average # traps (bad placement)"},
	},
	// src/Data/Skills/act_int.lua:7382
	"ForbiddenRite": {
		{Name: "1 Projectile"},
		{Name: "All Primary Projectiles"},
	},
	// src/Data/Skills/act_int.lua:7541
	"ForbiddenRiteAltX": {
		{Name: "1 Projectile"},
		{Name: "All Primary Projectiles"},
	},
	// src/Data/Skills/act_dex.lua:8391
	"FrostBlades": {
		{Name: "Melee hit", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Icy blades", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_dex.lua:8494
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
	// src/Data/Skills/act_dex.lua:8599
	"GalvanicArrow": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:8763
	"GalvanicArrowAltY": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:8546
	"GalvanicField": {
		{Name: "Single Target"},
		{Name: "Multi Target"},
	},
	// src/Data/Skills/act_int.lua:8666
	"GalvanicFieldAltX": {
		{Name: "Single Target"},
		{Name: "Multi Target"},
	},
	// src/Data/Skills/act_int.lua:8775
	"GlacialCascade": {
		{Name: "Initial Bursts"},
		{Name: "Final Burst"},
	},
	// src/Data/Skills/act_int.lua:9094
	"Hexblast": {
		{Name: "Target", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:9197
	"HexblastAltX": {
		{Name: "Target", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:9300
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
	// src/Data/Skills/act_int.lua:8949
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
	// src/Data/Skills/act_dex.lua:9545
	"IceShot": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:9629
	"IceShotAltX": {
		{Name: "Arrow", Flags: map[string]bool{"area": false}},
		{Name: "Cone", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:9810
	"IceSpear": {
		{Name: "First Form, 1 Projectile"},
		{Name: "Second Form, 1 Projectile"},
		{Name: "First Form, All Projectiles"},
		{Name: "Second Form, All Projectiles"},
	},
	// src/Data/Skills/act_int.lua:9920
	"IceSpearAltX": {
		{Name: "First Form, 1 Projectile"},
		{Name: "Second Form, 1 Projectile"},
		{Name: "First Form, All Projectiles"},
		{Name: "Second Form, All Projectiles"},
	},
	// src/Data/Skills/act_int.lua:10289
	"Incinerate": {
		{Name: "Channelling", Flags: map[string]bool{"stages": true}},
		{Name: "Release", Flags: map[string]bool{"stages": true, "channelRelease": true}},
	},
	// src/Data/Skills/act_int.lua:10427
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
	// src/Data/Skills/act_int.lua:10676
	"KineticBlast": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosions", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:10762
	"KineticBlastAltX": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosions", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:11001
	"KineticFusillade": {
		{Name: "All Projectiles"},
		{Name: "1 Projectile"},
	},
	// src/Data/Skills/act_int.lua:11219
	"KineticFusilladeAltX": {
		{Name: "All Projectiles"},
		{Name: "1 Projectile"},
	},
	// src/Data/Skills/act_dex.lua:9963
	"Lacerate": {
		{Name: "One slash"},
		{Name: "Both slashes"},
	},
	// src/Data/Skills/act_dex.lua:10056
	"LacerateAltX": {
		{Name: "One slash"},
		{Name: "Both slashes"},
	},
	// src/Data/Skills/act_dex.lua:10144
	"LacerateAltY": {
		{Name: "One slash"},
		{Name: "Both slashes"},
	},
	// src/Data/Skills/act_dex.lua:10242
	"LancingSteel": {
		{Name: "Single Projectile Hit"},
		{Name: "All Projectiles Hit"},
	},
	// src/Data/Skills/act_dex.lua:10345
	"LancingSteelAltX": {
		{Name: "Single Projectile Hit"},
		{Name: "All Projectiles Hit"},
	},
	// src/Data/Skills/act_int.lua:11789
	"LightningSpireTrap": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_int.lua:12029
	"LightningSpireTrapAltX": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_int.lua:12261
	"LightningSpireTrapAltY": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_dex.lua:10697
	"LightningStrike": {
		{Name: "Melee hit", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Projectiles", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_dex.lua:10798
	"LightningStrikeAltX": {
		{Name: "Melee hit", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Projectiles", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_int.lua:12539
	"LightningTendrils": {
		{Name: "Average DPS"},
		{Name: "Normal pulse"},
		{Name: "Stronger pulse"},
	},
	// src/Data/Skills/act_int.lua:12663
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
	// src/Data/Skills/act_int.lua:13713
	"OrbOfStormsAltX": {
		{Name: "Self Cast"},
		{Name: "Channelling"},
	},
	// src/Data/Skills/act_int.lua:13904
	"PenanceBrandAltX": {
		{Name: "Energy Pulse", Flags: map[string]bool{"stages": true}},
		{Name: "Max Pulse per Brand"},
	},
	// src/Data/Skills/act_dex.lua:17415
	"PoisonousConcoction": {
		{Name: "No Flask"},
		{Name: "Life"},
	},
	// src/Data/Skills/act_dex.lua:17516
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
	// src/Data/Skills/act_int.lua:14359
	"PurifyingFlame": {
		{Name: "Initial hit"},
		{Name: "Shockwave"},
	},
	// src/Data/Skills/act_int.lua:14463
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
	// src/Data/Skills/act_int.lua:15722
	"ScorchingRay": {
		{Name: "Manual Stages", Flags: map[string]bool{"stages": true}},
		{Name: "Maximum Stages"},
	},
	// src/Data/Skills/act_int.lua:15824
	"ScorchingRayAltX": {
		{Name: "Manual Stages", Flags: map[string]bool{"stages": true}},
		{Name: "Maximum Stages"},
	},
	// src/Data/Skills/act_dex.lua:12993
	"ScourgeArrow": {
		{Name: "Release", Flags: map[string]bool{"stages": true}},
		{Name: "Thorn Arrows", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:13092
	"ScourgeArrowAltX": {
		{Name: "Release"},
		{Name: "Thorn Arrows"},
	},
	// src/Data/Skills/act_str.lua:9644
	"SearingBondAltX": {
		{Name: "Beam"},
		{Name: "Explosion"},
	},
	// src/Data/Skills/act_dex.lua:13497
	"SeismicTrap": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
		{Name: "Average active traps, one wave"},
		{Name: "Average active traps, average waves"},
		{Name: "Average active traps, all waves"},
	},
	// src/Data/Skills/act_dex.lua:13672
	"SeismicTrapAltX": {
		{Name: "One wave hitting"},
		{Name: "Average waves hitting configured size enemy"},
		{Name: "All waves hitting"},
	},
	// src/Data/Skills/act_dex.lua:13189
	"ShatteringSteel": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Cone AoE"},
	},
	// src/Data/Skills/act_dex.lua:13278
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
	// src/Data/Skills/act_int.lua:15922
	"ShockNova": {
		{Name: "Ring + Nova"},
		{Name: "Ring"},
		{Name: "Nova"},
	},
	// src/Data/Skills/act_int.lua:16015
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
	// src/Data/Skills/act_int.lua:16381
	"SomaticShell": {
		{Name: "Projectile", Flags: map[string]bool{"projectile": true}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_int.lua:16713
	"Spark": {
		{Name: "1 Hit"},
		{Name: "Maximum Hits"},
	},
	// src/Data/Skills/act_int.lua:16803
	"SparkAltX": {
		{Name: "1 Hit"},
		{Name: "Maximum Hits"},
	},
	// src/Data/Skills/act_int.lua:16894
	"SparkAltY": {
		{Name: "1 Hit"},
		{Name: "Maximum Hits"},
	},
	// src/Data/Skills/act_dex.lua:14305
	"SpectralShieldThrow": {
		{Name: "Shield"},
		{Name: "Shards"},
	},
	// src/Data/Skills/act_dex.lua:14405
	"SpectralShieldThrowAltX": {
		{Name: "Shield"},
		{Name: "Shards"},
	},
	// src/Data/Skills/act_dex.lua:14492
	"SpectralShieldThrowAltY": {
		{Name: "Shield"},
		{Name: "Shards"},
	},
	// src/Data/Skills/act_str.lua:10731
	"StaticStrike": {
		{Name: "Melee hit", Flags: map[string]bool{"chaining": false}},
		{Name: "Beams", Flags: map[string]bool{"chaining": true}},
	},
	// src/Data/Skills/act_int.lua:18025
	"StormBurst": {
		{Name: "1 Orb Tick"},
		{Name: "Max Channelled Orbs"},
		{Name: "Max Duration Explode"},
	},
	// src/Data/Skills/act_int.lua:18143
	"StormBurstAltX": {
		{Name: "1 Orb Tick"},
		{Name: "Max Duration Explode"},
	},
	// src/Data/Skills/act_dex.lua:11739
	"StormRain": {
		{Name: "Arrow"},
		{Name: "Beam"},
	},
	// src/Data/Skills/act_dex.lua:11857
	"StormRainAltX": {
		{Name: "Arrow"},
		{Name: "Beam"},
	},
	// src/Data/Skills/act_dex.lua:11965
	"StormRainAltY": {
		{Name: "Arrow"},
		{Name: "Beam"},
	},
	// src/Data/Skills/act_int.lua:17659
	"Stormbind": {
		{Name: "Unimproved"},
		{Name: "1 Improvement"},
		{Name: "2 Improvements"},
		{Name: "3 Improvements"},
	},
	// src/Data/Skills/act_int.lua:17841
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
	// src/Data/Skills/act_dex.lua:17842
	"Thunderstorm": {
		{Name: "Projectile"},
		{Name: "Storm Explosion", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:17949
	"ThunderstormMiniTornados": {
		{Name: "Projectile", Flags: map[string]bool{"projectile": true}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:18131
	"TornadoAltY": {
		{Name: "Fire", Flags: map[string]bool{"stages": true}},
		{Name: "Lightning", Flags: map[string]bool{"stages": true}},
		{Name: "Cold", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_dex.lua:16043
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
	// src/Data/Skills/act_dex.lua:5436
	"VaalDetonateDead": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_str.lua:3575
	"VaalEarthquake": {
		{Name: "Initial impact"},
		{Name: "Aftershock"},
	},
	// src/Data/Skills/act_int.lua:5882
	"VaalFireball": {
		{Name: "Projectile", Flags: map[string]bool{"area": false}},
		{Name: "Explosion", Flags: map[string]bool{"area": true}},
	},
	// src/Data/Skills/act_dex.lua:10582
	"VaalLightningArrow": {
		{Name: "1 Projectile"},
		{Name: "All Projectiles"},
	},
	// src/Data/Skills/act_dex.lua:10901
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
	// src/Data/Skills/act_dex.lua:16293
	"VenomGyre": {
		{Name: "Outgoing Projectile"},
		{Name: "Returning Projectile"},
	},
	// src/Data/Skills/act_dex.lua:16657
	"VolatileDead": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_dex.lua:16760
	"VolatileDeadAltX": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_dex.lua:16864
	"VolatileDeadAltY": {
		{Name: "Spell", Flags: map[string]bool{"spell": true, "cast": false}},
		{Name: "Corpse Explosion", Flags: map[string]bool{"spell": false, "cast": true}},
	},
	// src/Data/Skills/act_str.lua:12651
	"VolcanicFissure": {
		{Name: "Ground Fissure", Flags: map[string]bool{"melee": true, "projectile": false}},
		{Name: "Magma Balls", Flags: map[string]bool{"melee": false, "projectile": true}},
	},
	// src/Data/Skills/act_dex.lua:17053
	"WildStrike": {
		{Name: "Fire hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Fire explosion", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": false, "area": true}},
		{Name: "Lightning hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Lightning bolt", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": true, "area": false}},
		{Name: "Cold hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Icy wave", Flags: map[string]bool{"melee": false, "projectile": true, "chaining": false, "area": false}},
	},
	// src/Data/Skills/act_dex.lua:17194
	"WildStrikeAltX": {
		{Name: "Fire hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Fire explosion", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": false, "area": true}},
		{Name: "Lightning hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Lightning bolt", Flags: map[string]bool{"melee": false, "projectile": false, "chaining": true, "area": false}},
		{Name: "Cold hit", Flags: map[string]bool{"melee": true, "projectile": false, "chaining": false, "area": false}},
		{Name: "Icy wave", Flags: map[string]bool{"melee": false, "projectile": true, "chaining": false, "area": false}},
	},
	// src/Data/Skills/act_int.lua:20973
	"WinterOrb": {
		{Name: "Channelling", Flags: map[string]bool{"stages": true}},
		{Name: "Idle", Flags: map[string]bool{"stages": true}},
	},
	// src/Data/Skills/act_int.lua:21132
	"WintertideBrand": {
		{Name: "Average Damage"},
		{Name: "Manual Stages", Flags: map[string]bool{"stages": true}},
	},
}
