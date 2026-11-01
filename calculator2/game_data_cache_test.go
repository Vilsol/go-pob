package calculator2

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/loader"
	game "github.com/Vilsol/go-pob-data/poe2"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/storage"
)

// freshPruned reads the originals from the asset cache and keeps the pruned copies in memory, so every table is
// pruned by the code under test.
type freshPruned struct {
	loader.AssetCache
	pruned map[string][]byte
}

func isPrunedKey(key string) bool { return strings.Contains(key, "#columns-") }

func (f freshPruned) Get(key string) ([]byte, error) {
	if isPrunedKey(key) {
		return f.pruned[key], nil
	}
	b, err := f.AssetCache.Get(key)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return b, nil
}

func (f freshPruned) Set(key string, value []byte) error {
	if isPrunedKey(key) {
		f.pruned[key] = value
		return nil
	}
	if err := f.AssetCache.Set(key, value); err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return nil
}

func (f freshPruned) Exists(key string) bool {
	if isPrunedKey(key) {
		_, ok := f.pruned[key]
		return ok
	}
	return f.AssetCache.Exists(key)
}

func TestPrunedTablesDecodeAsTheOriginals(t *testing.T) {
	ctx := context.Background()
	original := assetcache.AssetLoaderWrapper{Storage: storage.Get()}
	fresh := freshPruned{AssetCache: original, pruned: map[string][]byte{}}
	pruned := assetcache.Columns{AssetCache: fresh, Rows: prunedTables}

	wantSkills, err := game.LoadSkillTables(ctx, skillDataVersion, original)
	if err != nil {
		t.Fatal(err)
	}
	gotSkills, err := game.LoadSkillTables(ctx, skillDataVersion, pruned)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotSkills, wantSkills) {
		t.Error("the skill tables decode differently from their pruned copies")
	}

	wantMinions, err := game.LoadMinionTables(ctx, skillDataVersion, original)
	if err != nil {
		t.Fatal(err)
	}
	gotMinions, err := game.LoadMinionTables(ctx, skillDataVersion, pruned)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotMinions, wantMinions) {
		t.Error("the minion tables decode differently from their pruned copies")
	}

	wantFlavour, err := loader.LoadJSONTable[game.FlavourText](ctx, skillDataVersion, game.FlavourTextTable, original)
	if err != nil {
		t.Fatal(err)
	}
	gotFlavour, err := loader.LoadJSONTable[game.FlavourText](ctx, skillDataVersion, game.FlavourTextTable, pruned)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotFlavour, wantFlavour) {
		t.Error("the flavour text decodes differently from its pruned copy")
	}

	if len(fresh.pruned) != len(prunedTables) {
		t.Errorf("%d of the %d pruned tables were read pruned", len(fresh.pruned), len(prunedTables))
	}
}
