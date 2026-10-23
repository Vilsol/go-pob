//go:build ignore

// Fills the Go bundle cache with every data.pob.gg asset the wasm can ask for, so the e2e data route can serve all
// of them: what raw.InitializeAll and raw.LoadDeferred load, every asset go-pob-data's field list names for the game
// version (the tables only some pages read, such as the item art tables, included), and the tree of every tree
// version. The lists are Go's own, so a new table is covered without touching this file.
// Usage: go run frontend/e2e/populate_cache.go (GO_POB_CACHE_DIR picks the cache directory).
package main

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"reflect"
	"slices"

	"github.com/Vilsol/go-pob-data/fields"
	"github.com/Vilsol/go-pob-data/loader"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

func refuse(a ...any) {
	fmt.Fprintln(os.Stderr, append([]any{"REFUSING:"}, a...)...)
	os.Exit(1)
}

func fetch(url string) (fetched bool) {
	if storage.Get().ExistsInCache(raw.CacheKey(url)) {
		return false
	}
	resp, err := http.Get(url) //nolint:gosec,noctx // a one-shot fetch of a URL Go's own lists name
	if err != nil {
		refuse(url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		refuse(url, resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		refuse(url, err)
	}
	if err := storage.Get().SetCache(raw.CacheKey(url), b); err != nil {
		refuse(url, err)
	}
	return true
}

func main() {
	if err := raw.InitializeAll(raw.LatestVersion, nil); err != nil {
		refuse(err)
	}
	if err := raw.LoadDeferred(); err != nil {
		refuse(err)
	}

	list, ok := fields.For(raw.LatestVersion)
	if !ok {
		refuse("go-pob-data has no field list for", raw.LatestVersion)
	}
	ctx := raw.DataContext(context.Background())
	assets, added := 0, 0
	for _, asset := range slices.Sorted(maps.Keys(list.Assets)) {
		t, ok := fields.RowType(asset)
		if !ok {
			refuse("go-pob-data has no row type for", asset)
		}
		assets++
		if fetch(loader.VariantURL(ctx, raw.LatestVersion, asset, reflect.New(t).Interface())) {
			added++
		}
	}

	versions := make([]string, 0, len(data.TreeVersions))
	for v, t := range data.TreeVersions {
		if len(t.RawTree()) == 0 {
			refuse("tree", v, "is empty")
		}
		versions = append(versions, string(v))
	}
	slices.Sort(versions)
	fmt.Printf("bundle cache: game data %s, %d listed assets (%d not loaded at start), %d tree versions %v\n", raw.LatestVersion, assets, added, len(versions), versions)
}
