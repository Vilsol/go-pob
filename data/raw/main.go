package raw

import (
	"runtime"
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

func (i initBlock) Load(version string) error {
	log.Trace().Str("func", i.Name).Msg("running initialization")
	start := time.Now()
	if err := i.Func(version); err != nil {
		return errors.Wrap(err, "failed to initialize: "+i.Name)
	}
	log.Trace().Str("func", i.Name).Dur("took", time.Since(start)).Msg("completed initialization")
	return nil
}

var initFunctions = []initBlock{
	{
		Func: InitializeActiveSkillTypes,
		Name: "ActiveSkillTypes",
	},
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
		Func: InitializeGrantedEffectStatSets,
		Name: "GrantedEffectStatSets",
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
		Func: InitializeItemClasses,
		Name: "ItemClasses",
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
}

type UpdateFunc func(data string)

var alreadyInitialized = false

func InitializeAll(version string, updateFunc UpdateFunc) error {
	if alreadyInitialized {
		return nil
	}
	alreadyInitialized = true

	if runtime.GOMAXPROCS(0) == 1 {
		for _, function := range initFunctions {
			if updateFunc != nil {
				updateFunc(function.Name)
			}
			if err := function.Load(version); err != nil {
				return err
			}
		}
	} else {
		g := new(errgroup.Group)
		for _, function := range initFunctions {
			fn := function
			g.Go(func() error {
				return fn.Load(version)
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}
	}

	return nil
}
