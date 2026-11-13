package raw

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob-data/fields"
	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob-data/utils"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/storage"
)

// DefaultVersion is the game data testdata's corpus and oracle belong to; a test binary loads it unless POE_VERSION says otherwise.
const DefaultVersion = "3.29"

// AppVersion is the game data every other binary loads unless POE_VERSION says otherwise.
const AppVersion = "3.30"

var LatestVersion = gameDataVersion(os.Getenv, testing.Testing())

func gameDataVersion(getenv func(string) string, test bool) string {
	if v := getenv("POE_VERSION"); v != "" {
		return v
	}
	if test {
		return DefaultVersion
	}
	return AppVersion
}

// UpdateFunc reports a loading stage; with a total, done of total within it.
type UpdateFunc func(stage string, done, total int)

func CacheKey(url string) string {
	return assetcache.CacheKey(url)
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
	"GemEffects", "GemTags", "ModType", "MonsterTypes", "PassiveJewelSlots",
	"PassiveOverrideLimits", "PassiveSkillOverrideTypes", "PassiveSkillOverrides", "PassiveSkillTattooTargetSets",
	"PassiveSkillTattoos", "PassiveSkillTreeConnectionArt", "PassiveSkillTreeNodeFrameArt",
	"PassiveTreeExpansionJewelSizes", "UniqueStashTypes",
	"VillageBalancePerLevelShared", "VirtualStatContextFlags", "WeaponPassiveSkillTypes", "WeaponPassiveSkills",
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
	cache := assetcache.AssetLoaderWrapper{Storage: storage.Get()}
	update := func(stage string, done, total int) {
		if updateFunc != nil {
			updateFunc(stage, done, total)
		}
	}
	urls := prefetchURLs(ctx, version)
	update("Downloading", 0, len(urls))
	cache.Prefetched = prefetch(ctx, cache, urls, func(done, total int) { update("Downloading", done, total) })

	for i, t := range tables {
		update(t.Name, i, len(tables))
		if err := t.Load(ctx, version, cache); err != nil {
			return fmt.Errorf("failed to initialize: %s: %w", t.Name, err)
		}
	}
	update("Indexing", 0, 0)
	for _, hook := range utils.RegisteredHooks {
		hook()
	}

	lazyCache.Store(&assetcache.AssetLoaderWrapper{Storage: cache.Storage, Prefetched: lazyPrefetched(ctx, cache.Prefetched, version)})
	return nil
}

func GameDataVersion() string {
	return LatestVersion
}
