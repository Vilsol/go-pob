package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

const genEnchantment = 10

//pob:block:src/Export/Scripts/enchant.lua:41-48
var (
	enchantLab = map[int]string{
		32: "NORMAL",
		53: "CRUEL",
		66: "MERCILESS",
		75: "ENDGAME",
		83: "DEDICATION",
	}
	enchantSourceOrder = []string{"NORMAL", "CRUEL", "MERCILESS", "ENDGAME", "DEDICATION", "ENKINDLING", "INSTILLING", "HARVEST", "HEIST"}
)

//pob:block:src/Export/Scripts/enchant.lua:149-280
var enchantSkillMap = []struct{ pattern, name string }{
	{"Summone?d?RagingSpirit", "Summon Raging Spirit"},
	{"SpiritOffering", "Spirit Offering"},
	{"Discharge", "Discharge"},
	{"AncestorTotem[^S][^l]", "Ancestral Protector"},
	{"AncestorTotemSlamMelee", "Ancestral Warchief"},
	{"AnimateGuardian", "Animate Guardian"},
	{"AnimateWeapon", "Animate Weapon"},
	{"BlinkArrow", "Blink Arrow"},
	{"ConversionTrap", "Conversion Trap"},
	{"MirrorArrow", "Mirror Arrow"},
	{"Spectre", "Raise Spectre"},
	{"Zombie", "Raise Zombie"},
	{"ChaosGolem", "Summon Chaos Golem"},
	{"FlameGolem", "Summon Flame Golem"},
	{"IceGolem", "Summon Ice Golem"},
	{"LightningGolem", "Summon Lightning Golem"},
	{"StoneGolem", "Summon Stone Golem"},
	{"Skeleton", "Summon Skeletons"},
	{"Bladefall", "Bladefall"},
	{"BlastRain", "Blast Rain"},
	{"ChargedAttack", "Blade Flurry"},
	{"Desecrate", "Desecrate"},
	{"DetonateDead", "Detonate Dead"},
	{"DevouringTotem", "Devouring Totem"},
	{"DominatingBlow", "Dominating Blow"},
	{"FireBeam", "Scorching Ray"},
	{"Firestorm", "Firestorm"},
	{"FreezeMine", "Freeze Mine"},
	{"EnchantmentFrenzy", "Frenzy"},
	{"GroundSlam", "Ground Slam"},
	{"HeavyStrike", "Heavy Strike"},
	{"IceSpear", "Ice Spear"},
	{"ImmortalCall", "Immortal Call"},
	{"Incinerate", "Incinerate"},
	{"KineticBlast", "Kinetic Blast"},
	{"LightningArrow", "Lightning Arrow"},
	{"ChargedDash", "Charged Dash"},
	{"PhaseRun", "Phase Run"},
	{"Puncture", "Puncture"},
	{"RejuvinationTotem", "Rejuvenation Totem"},
	{"ShockNova", "Shock Nova"},
	{"SpectralThrow", "Spectral Throw"},
	{"TectonicSlam", "Tectonic Slam"},
	{"VolatileDead", "Volatile Dead"},
	{"BoneLance", "Unearth"},
	{"CorpseEruption", "Cremation"},
	{"PowerSiphon", "Power Siphon"},
	{"Smite", "Smite"},
	{"ConsecratedPath", "Consecrated Path"},
	{"ScourgeArrow", "Scourge Arrow"},
	{"HolyRelic", "Summon Holy Relic"},
	{"HeraldOfAgony", "Herald of Agony"},
	{"HeraldOfPurity", "Herald of Purity"},
	{"Bane", "Bane"},
	{"DivineIre", "Divine Ire"},
	{"PurifyingFlame", "Purifying Flame"},
	{"Soulrend", "Soulrend"},
	{"StormBurst", "Storm Burst"},
	{"CarrionGolem", "Summon Carrion Golem"},
	{"Steelskin", "Steelskin"},
	{"[^d]Dash", "Dash"},
	{"Bladestorm", "Bladestorm"},
	{"Perforate", "Perforate"},
	{"Frostblink", "Frostblink"},
	{"ChainHook", "Chain Hook"},
	{"Berserk", "Berserk"},
	{"WitheringStep", "Withering Step"},
	{"SnappingAdder", "Venom Gyre"},
	{"PlagueBearer", "Plague Bearer"},
	{"SummonSkitterbots", "Summon Skitterbots"},
	{"ArtilleryBallista", "Artillery Ballista"},
	{"ArcaneCloak", "Arcane Cloak"},
	{"KineticBolt", "Kinetic Bolt"},
	{"BladeBlast", "Blade Blast"},
	{"RuneBlast", "Stormbind"},
	{"Spellslinger", "Spellslinger"},
	{"AncestralCry", "Ancestral Cry"},
	{"EnduringCry", "Enduring Cry"},
	{"SeismicCry", "Seismic Cry"},
	{"Sunder", "Sunder"},
	{"Earthshatter", "Earthshatter"},
	{"ArcanistBrand", "Arcanist Brand"},
	{"BlazingSalvo", "Blazing Salvo"},
	{"Anger", "Anger"},
	{"Clarity", "Clarity"},
	{"Determination", "Determination"},
	{"Discipline", "Discipline"},
	{"Grace", "Grace"},
	{"Haste", "Haste"},
	{"Hatred", "Hatred"},
	{"Malevolence", "Malevolence"},
	{"Precision", "Precision"},
	{"Pride", "Pride"},
	{"Vitality", "Vitality"},
	{"Wrath", "Wrath"},
	{"Zealotry", "Zealotry"},
	{"PurityOfElements", "Purity of Elements"},
	{"PurityOfFire", "Purity of Fire"},
	{"PurityOfIce", "Purity of Ice"},
	{"PurityOfLightning", "Purity of Lightning"},
	{"MortarBarrageMine", "Pyroclast Mine"},
	{"ColdProjectileMine", "Icicle Mine"},
	{"LightningExplosionMine", "Stormblast Mine"},
	{"FleshAndStone", "Flesh and Stone"},
	{"DreadBanner", "Dread Banner"},
	{"WarBanner", "War Banner"},
	{"FrostShield", "Frost Shield"},
	{"VoidSphere", "Void Sphere"},
	{"CracklingLance", "Crackling Lance"},
	{"SigilOfPower", "Sigil of Power"},
	{"Hexblast", "Hexblast"},
	{"FlameWall", "Flame Wall"},
	{"WaterSphere", "Hydrosphere"},
	{"CorruptingFever", "Corrupting Fever"},
	{"Bloodreap", "Reap"},
	{"BladeTrap", "Blade Trap"},
	{"EyeOfWinter", "Eye of Winter"},
	{"StormRain", "Storm Rain"},
	{"RageVortex", "Rage Vortex"},
	{"ShieldCrush", "Shield Crush"},
	{"SummonedReaper", "Summon Reaper"},
	{"Boneshatter", "Boneshatter"},
	{"SpectralHelix", "Spectral Helix"},
	{"DefianceBanner", "Defiance Banner"},
	{"EnergyBlade", "Energy Blade"},
	{"TornadoShot", "Tornado Shot"},
	{"Tornado", "Tornado"},
	{"VolcanicFissure", "Volcanic Fissure"},
	{"Table Charge", "Shield Charge"},
	{"Flame Dash", "Flame Dash"},
}

var enchantSkillRes = func() []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(enchantSkillMap))
	for i, s := range enchantSkillMap {
		out[i] = regexp.MustCompile(s.pattern)
	}
	return out
}()

func enchantFamily(mod *poe.Mod) string {
	if len(mod.Families) == 0 {
		return ""
	}
	return poe.ModFamilies[mod.Families[0]].ID
}

func enchantSpawns(mod *poe.Mod) bool {
	return len(mod.SpawnWeightValues) > 0 && mod.SpawnWeightValues[0] > 0
}

//pob:block:src/Export/Scripts/enchant.lua:50-74
func doLabEnchantment(x *modsExport, group string) (map[string][]string, error) {
	byDiff := map[string][]string{}
	for _, mod := range poe.Mods {
		if mod.GenerationType != genEnchantment || enchantFamily(mod) != group || !enchantSpawns(mod) {
			continue
		}
		stats, _, err := x.describeMod(mod, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", mod.ID, err)
		}
		diff := enchantLab[mod.Level]
		byDiff[diff] = append(byDiff[diff], strings.Join(stats, "/"))
	}
	return enchantBySourceOrder(byDiff), nil
}

func enchantBySourceOrder(byDiff map[string][]string) map[string][]string {
	out := map[string][]string{}
	for _, diff := range enchantSourceOrder {
		if byDiff[diff] != nil {
			out[diff] = byDiff[diff]
		}
	}
	return out
}

//pob:block:src/Export/Scripts/enchant.lua:80-140
func doOtherEnchantment(x *modsExport, groupsList map[int]map[string]string) (map[string][]string, error) {
	byDiff := map[string][]string{}
	generations := make([]int, 0, len(groupsList))
	for g := range groupsList {
		generations = append(generations, g)
	}
	sort.Ints(generations)
	for _, generation := range generations {
		for _, mod := range poe.Mods {
			if mod.GenerationType != generation {
				continue
			}
			diff, ok := groupsList[generation][enchantFamily(mod)]
			if !ok {
				continue
			}
			stats, _, err := x.describeMod(mod, nil)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", mod.ID, err)
			}
			byDiff[diff] = append(byDiff[diff], strings.Join(stats, "/"))
		}
	}
	return enchantBySourceOrder(byDiff), nil
}

//pob:block:src/Export/Scripts/enchant.lua:285-335
func enchantSkillOf(mod *poe.Mod, bySpecific, bySecondary map[int][]*poe.ActiveSkill) string {
	skill := ""
	pick := func(skills []*poe.ActiveSkill) {
		for _, activeSkill := range skills {
			if !strings.Contains(activeSkill.ID, "vaal") && activeSkill.DisplayedName != "" {
				skill = activeSkill.DisplayedName
				return
			}
		}
	}
	for _, key := range []*int{mod.StatsKey1, mod.StatsKey2, mod.StatsKey3, mod.StatsKey4, mod.StatsKey5, mod.StatsKey6} {
		if key == nil {
			continue
		}
		pick(bySpecific[*key])
		pick(bySecondary[*key])
	}
	for i, re := range enchantSkillRes {
		if loc := re.FindStringIndex(mod.ID); loc != nil {
			if len(skill) < loc[1]-1-loc[0] {
				skill = enchantSkillMap[i].name
			}
		}
	}
	for _, s := range enchantSkillMap {
		if s.pattern == skill {
			return s.name
		}
	}
	return skill
}

//pob:block:src/Export/Scripts/enchant.lua:282-370
func doHelmetEnchantment(x *modsExport) (map[string]map[string][]string, error) {
	bySpecific, bySecondary := map[int][]*poe.ActiveSkill{}, map[int][]*poe.ActiveSkill{}
	for _, a := range poe.ActiveSkills {
		for _, s := range a.InputStatKeys {
			bySpecific[s] = append(bySpecific[s], a)
		}
		for _, s := range a.SecondarySkillSpecificStats {
			bySecondary[s] = append(bySecondary[s], a)
		}
	}
	bySkill := map[string]map[string][]string{}
	for _, mod := range poe.Mods {
		if mod.GenerationType != genEnchantment || enchantFamily(mod) != "SkillEnchantment" || !enchantSpawns(mod) {
			continue
		}
		skill := enchantSkillOf(mod, bySpecific, bySecondary)
		stats, _, err := x.describeMod(mod, nil)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", mod.ID, err)
		}
		if len(stats) == 0 {
			continue
		}
		if bySkill[skill] == nil {
			bySkill[skill] = map[string][]string{}
		}
		diff := enchantLab[mod.Level]
		bySkill[skill][diff] = append(bySkill[skill][diff], strings.Join(stats, "/"))
	}
	for skill, byDiff := range bySkill {
		bySkill[skill] = enchantBySourceOrder(byDiff)
	}
	return bySkill, nil
}

type craftEnchantments struct {
	Skills  map[string]map[string][]string `json:"skills,omitempty"`
	Sources map[string][]string            `json:"sources,omitempty"`
}

//pob:block:src/Modules/Data.lua:675-711
func dumpEnchantments(weaponTypes []string) (map[string]craftEnchantments, error) {
	ctx, cache := gameDataContext(), assetCache()
	if err := poe.InitializeActiveSkills(ctx, raw.LatestVersion, cache); err != nil {
		return nil, fmt.Errorf("ActiveSkills: %w", err)
	}
	describer, err := loadStatFile("stat_descriptions")
	if err != nil {
		return nil, err
	}
	x := &modsExport{describer: describer}
	out := map[string]craftEnchantments{}
	helmet, err := doHelmetEnchantment(x)
	if err != nil {
		return nil, err
	}
	out["Helmet"] = craftEnchantments{Skills: helmet}
	for itemType, group := range map[string]string{"Boots": "ConditionalBuffEnchantment", "Gloves": "TriggerEnchantment", "Belt": "BuffEnchantment"} {
		bySource, err := doLabEnchantment(x, group)
		if err != nil {
			return nil, err
		}
		out[itemType] = craftEnchantments{Sources: bySource}
	}
	for itemType, groups := range map[string]map[int]map[string]string{
		"UtilityFlask": {21: {"FlaskEnchantment": "ENKINDLING"}, 22: {"FlaskEnchantment": "INSTILLING"}},
		"Body Armour":  {3: {"AlternateArmourQuality": "HARVEST", "EnchantmentHeistArmour": "HEIST"}},
		"Weapon":       {3: {"AlternateWeaponQuality": "HARVEST", "EnchantmentHeistWeapon": "HEIST"}},
	} {
		bySource, err := doOtherEnchantment(x, groups)
		if err != nil {
			return nil, err
		}
		out[itemType] = craftEnchantments{Sources: bySource}
	}
	out["Flask"] = out["UtilityFlask"]
	for _, baseType := range weaponTypes {
		out[baseType] = out["Weapon"]
	}
	delete(out, "Weapon")
	return out, nil
}
