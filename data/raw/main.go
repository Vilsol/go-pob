package raw

import (
	"reflect"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const LatestVersion = "3.18-3"

type InitFunction func(version string) error

var initFunctions = []InitFunction{
	InitializeActiveSkills,
	InitializeAlternatePassiveAdditions,
	InitializeAlternatePassiveSkills,
	InitializeArmourTypes,
	InitializeBaseItemTypes,
	InitializeComponentAttributeRequirements,
	InitializeComponentCharges,
	InitializeCostTypes,
	InitializeCraftingBenchOptions,
	InitializeDefaultMonsterStats,
	InitializeEssences,
	InitializeFlasks,
	InitializeGrantedEffectQualityStats,
	InitializeGrantedEffectStatSetsPerLevels,
	InitializeGrantedEffects,
	InitializeGrantedEffectsPerLevels,
	InitializeItemExperiencePerLevels,
	InitializeMods,
	InitializeMonsterMapBossDifficulties,
	InitializeMonsterMapDifficulties,
	InitializeMonsterVarieties,
	InitializePantheonPanelLayouts,
	InitializePassiveTreeExpansionJewels,
	InitializePassiveTreeExpansionSkills,
	InitializePassiveTreeExpansionSpecialSkills,
	InitializeShieldTypes,
	InitializeSkillGems,
	InitializeSkillTotemVariations,
	InitializeSkillTotems,
	InitializeStats,
	InitializeTags,
	InitializeWeaponTypes,
	InitializeActiveSkillTypes,
	InitializeItemClasses,
	InitializeGrantedEffectStatSets,
}

var alreadyInitialized = false

func InitializeAll(version string) error {
	if alreadyInitialized {
		return nil
	}
	alreadyInitialized = true

	for _, function := range initFunctions {
		funcName := runtime.FuncForPC(reflect.ValueOf(function).Pointer()).Name()
		log.Trace().Str("func", funcName).Msg("running initialization")
		start := time.Now()
		if err := function(version); err != nil {
			return errors.Wrap(err, "failed to initialize")
		}
		log.Trace().Str("func", funcName).Dur("took", time.Since(start)).Msg("completed initialization")
	}

	return nil
}
