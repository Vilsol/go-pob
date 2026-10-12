package raw

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/Vilsol/go-pob-data/fields"
	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/utils"

	"github.com/Vilsol/go-pob/storage"
)

const DefaultVersion = "3.29"

var LatestVersion = gameDataVersion(os.Getenv)

func gameDataVersion(getenv func(string) string) string {
	if v := getenv("POE_VERSION"); v != "" {
		return v
	}
	return DefaultVersion
}

type UpdateFunc func(data string)

type AssetLoaderWrapper struct {
	storage.Storage
	prefetched map[string][]byte
}

func (a AssetLoaderWrapper) Get(key string) ([]byte, error) {
	if b, ok := a.prefetched[key]; ok {
		return b, nil
	}
	return a.Storage.GetCache(CacheKey(key))
}

func (a AssetLoaderWrapper) Set(key string, value []byte) error {
	return a.Storage.SetCache(CacheKey(key), value)
}

func (a AssetLoaderWrapper) Exists(key string) bool {
	if _, ok := a.prefetched[key]; ok {
		return true
	}
	return a.Storage.ExistsInCache(CacheKey(key))
}

type table struct {
	Name string
	Load poe.InitFunction
}

var tables = []table{
	{"ActiveSkillType", poe.InitializeActiveSkillTypes},
	{"ActiveSkills", poe.InitializeActiveSkills},
	{"ArmourTypes", poe.InitializeArmourTypes},
	{"BaseItemTypes", poe.InitializeBaseItemTypes},
	{"ComponentAttributeRequirements", poe.InitializeComponentAttributeRequirements},
	{"ComponentCharges", poe.InitializeComponentCharges},
	{"CostTypes", poe.InitializeCostTypes},
	{"Flasks", poe.InitializeFlasks},
	{"GrantedEffectQualityStats", poe.InitializeGrantedEffectQualityStats},
	{"GrantedEffectStatSets", poe.InitializeGrantedEffectStatSets},
	{"GrantedEffectStatSetsPerLevel", poe.InitializeGrantedEffectStatSetsPerLevels},
	{"GrantedEffects", poe.InitializeGrantedEffects},
	{"GrantedEffectsPerLevel", poe.InitializeGrantedEffectsPerLevels},
	{"ItemClasses", poe.InitializeItemClasses},
	{"ShieldTypes", poe.InitializeShieldTypes},
	{"SkillGems", poe.InitializeSkillGems},
	{"SkillTotemVariations", poe.InitializeSkillTotemVariations},
	{"Stats", poe.InitializeStats},
	{"Tags", poe.InitializeTags},
	{"WeaponTypes", poe.InitializeWeaponTypes},
}

var unreadTables = []string{
	"AlternatePassiveAdditions", "AlternatePassiveSkills", "CraftingBenchOptions", "DefaultMonsterStats",
	"Essences", "ItemExperiencePerLevel", "MonsterMapBossDifficulty", "MonsterMapDifficulty",
	"PantheonPanelLayout", "PassiveTreeExpansionJewels", "PassiveTreeExpansionSkills",
	"PassiveTreeExpansionSpecialSkills", "SkillTotems",
	"AlternateTreeArt", "BuffTemplates", "ClientStrings", "CurrencyExchange", "FlavourText", "GameConstants",
	"GemEffects", "GemTags", "ItemVisualIdentity", "ModType", "MonsterTypes", "PassiveJewelSlots",
	"PassiveOverrideLimits", "PassiveSkillOverrideTypes", "PassiveSkillOverrides", "PassiveSkillTattooTargetSets",
	"PassiveSkillTattoos", "PassiveSkillTreeConnectionArt", "PassiveSkillTreeNodeFrameArt",
	"PassiveTreeExpansionJewelSizes", "UniqueStashLayout", "UniqueStashTypes",
	"VillageBalancePerLevelShared", "VirtualStatContextFlags", "WeaponPassiveSkillTypes", "WeaponPassiveSkills",
	"Words",
	"ModFamily",
	"CraftingItemClassCategories",
	"MonsterResistances",
	"EffectivenessCostConstants",
}

var (
	initializeOnce sync.Once
	errInitialize  error
)

func InitializeAll(version string, updateFunc UpdateFunc) error {
	initializeOnce.Do(func() { errInitialize = initializeAll(version, updateFunc) })
	return errInitialize
}

const FullAssetsEnv = "GO_POB_FULL_ASSETS"

func DataContext(ctx context.Context) context.Context {
	if os.Getenv(FullAssetsEnv) != "" {
		return ctx
	}
	return loader.WithVariants(ctx, fields.MinVariant)
}

func initializeAll(version string, updateFunc UpdateFunc) error {
	ctx := DataContext(context.Background())
	cache := AssetLoaderWrapper{Storage: storage.Get()}
	if updateFunc != nil {
		updateFunc("Downloading")
	}
	cache.prefetched = prefetch(ctx, cache, prefetchURLs(ctx, version))

	for _, t := range tables {
		if updateFunc != nil {
			updateFunc(t.Name)
		}
		if err := t.Load(ctx, version, cache); err != nil {
			return fmt.Errorf("failed to initialize: %s: %w", t.Name, err)
		}
	}
	if updateFunc != nil {
		updateFunc("Indexing")
	}
	for _, hook := range utils.RegisteredHooks {
		hook()
	}

	lazyCache.Store(&AssetLoaderWrapper{Storage: cache.Storage, prefetched: lazyPrefetched(ctx, cache.prefetched, version)})
	return nil
}

func GameDataVersion() string {
	return LatestVersion
}
