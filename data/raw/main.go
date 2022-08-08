package raw

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

const LatestVersion = "3.18"

type InitFunction func(version string) error

type initBlock struct {
	Func InitFunction
	Name string
}

var initFunctions = []initBlock{
	{
		Func: InitializeActiveSkills,
		Name: "ActiveSkills",
	},
	{
		Func: InitializeAlternatePassiveAdditions,
		Name: "AlternatePassiveAdditions",
	},
	{
		Func: InitializeAlternatePassiveSkills,
		Name: "AlternatePassiveSkills",
	},
	{
		Func: InitializeArmourTypes,
		Name: "ArmourTypes",
	},
	{
		Func: InitializeBaseItemTypes,
		Name: "BaseItemTypes",
	},
	{
		Func: InitializeComponentAttributeRequirements,
		Name: "ComponentAttributeRequirements",
	},
	{
		Func: InitializeComponentCharges,
		Name: "ComponentCharges",
	},
	{
		Func: InitializeCostTypes,
		Name: "CostTypes",
	},
	{
		Func: InitializeCraftingBenchOptions,
		Name: "CraftingBenchOptions",
	},
	{
		Func: InitializeDefaultMonsterStats,
		Name: "DefaultMonsterStats",
	},
	{
		Func: InitializeEssences,
		Name: "Essences",
	},
	{
		Func: InitializeFlasks,
		Name: "Flasks",
	},
	{
		Func: InitializeGrantedEffectQualityStats,
		Name: "GrantedEffectQualityStats",
	},
	{
		Func: InitializeGrantedEffectStatSetsPerLevels,
		Name: "GrantedEffectStatSetsPerLevels",
	},
	{
		Func: InitializeGrantedEffects,
		Name: "GrantedEffects",
	},
	{
		Func: InitializeGrantedEffectsPerLevels,
		Name: "GrantedEffectsPerLevels",
	},
	{
		Func: InitializeItemExperiencePerLevels,
		Name: "ItemExperiencePerLevels",
	},
	{
		Func: InitializeMods,
		Name: "Mods",
	},
	{
		Func: InitializeMonsterMapBossDifficulties,
		Name: "MonsterMapBossDifficulties",
	},
	{
		Func: InitializeMonsterMapDifficulties,
		Name: "MonsterMapDifficulties",
	},
	{
		Func: InitializeMonsterVarieties,
		Name: "MonsterVarieties",
	},
	{
		Func: InitializePantheonPanelLayouts,
		Name: "PantheonPanelLayouts",
	},
	{
		Func: InitializePassiveTreeExpansionJewels,
		Name: "PassiveTreeExpansionJewels",
	},
	{
		Func: InitializePassiveTreeExpansionSkills,
		Name: "PassiveTreeExpansionSkills",
	},
	{
		Func: InitializePassiveTreeExpansionSpecialSkills,
		Name: "PassiveTreeExpansionSpecialSkills",
	},
	{
		Func: InitializeShieldTypes,
		Name: "ShieldTypes",
	},
	{
		Func: InitializeSkillGems,
		Name: "SkillGems",
	},
	{
		Func: InitializeSkillTotemVariations,
		Name: "SkillTotemVariations",
	},
	{
		Func: InitializeSkillTotems,
		Name: "SkillTotems",
	},
	{
		Func: InitializeStats,
		Name: "Stats",
	},
	{
		Func: InitializeTags,
		Name: "Tags",
	},
	{
		Func: InitializeWeaponTypes,
		Name: "WeaponTypes",
	},
	{
		Func: InitializeActiveSkillTypes,
		Name: "ActiveSkillTypes",
	},
	{
		Func: InitializeItemClasses,
		Name: "ItemClasses",
	},
	{
		Func: InitializeGrantedEffectStatSets,
		Name: "GrantedEffectStatSets",
	},
}

var alreadyInitialized = false

func InitializeAll(version string) error {
	if alreadyInitialized {
		return nil
	}
	alreadyInitialized = true

	g := new(errgroup.Group)
	for _, function := range initFunctions {
		fn := function
		g.Go(func() error {
			log.Trace().Str("func", fn.Name).Msg("running initialization")
			start := time.Now()
			if err := fn.Func(version); err != nil {
				return errors.Wrap(err, "failed to initialize: "+fn.Name)
			}
			log.Trace().Str("func", fn.Name).Dur("took", time.Since(start)).Msg("completed initialization")
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
