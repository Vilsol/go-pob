package raw

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/storage"
)

var (
	modsTable             = table{"Mods", poe.InitializeMods}
	monsterVarietiesTable = table{"MonsterVarieties", poe.InitializeMonsterVarieties}
	tincturesTable        = table{"Tinctures", poe.InitializeTinctures}
	deferredTables        = []table{modsTable, monsterVarietiesTable, tincturesTable}
)

var deferredOnce = map[string]*sync.Once{}

func init() {
	for _, t := range deferredTables {
		deferredOnce[t.Name] = &sync.Once{}
	}
}

var lazyCache atomic.Pointer[AssetLoaderWrapper]

func lazyLoader() AssetLoaderWrapper {
	if c := lazyCache.Load(); c != nil {
		return *c
	}
	return AssetLoaderWrapper{Storage: storage.Get()}
}

func lazyPrefetched(ctx context.Context, all map[string][]byte, version string) map[string][]byte {
	out := map[string][]byte{}
	for _, name := range slices.Concat(deferredNames(), timelessTables) {
		url := rawURL(ctx, version, name)
		if b, ok := all[url]; ok {
			out[url] = b
		}
	}
	return out
}

func deferredNames() []string {
	out := make([]string, 0, len(deferredTables))
	for _, t := range deferredTables {
		out = append(out, t.Name)
	}
	return out
}

func ensureDeferred(t table, present func() bool) {
	var err error
	deferredOnce[t.Name].Do(func() {
		if !present() {
			err = t.Load(DataContext(context.Background()), LatestVersion, lazyLoader())
		}
	})
	if !present() {
		panic(fmt.Sprintf("raw: deferred table %s is not loaded (%v)", t.Name, err))
	}
}

func Mods() []*poe.Mod {
	ensureDeferred(modsTable, func() bool { return len(poe.Mods) > 0 })
	return poe.Mods
}

func MonsterVarieties() []*poe.MonsterVariety {
	ensureDeferred(monsterVarietiesTable, func() bool { return len(poe.MonsterVarieties) > 0 })
	return poe.MonsterVarieties
}

func LoadDeferred() error {
	Mods()
	MonsterVarieties()
	Tinctures()
	return EnsureTimelessSources()
}

func Tinctures() []*poe.Tincture {
	ensureDeferred(tincturesTable, func() bool { return len(poe.Tinctures) > 0 })
	return poe.Tinctures
}
