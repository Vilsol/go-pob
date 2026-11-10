package raw

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"sync"

	"github.com/Vilsol/go-pob-data/fields"
	"github.com/Vilsol/go-pob-data/loader"

	"github.com/Vilsol/go-pob/data/assetcache"
)

const treeBase = "https://data.pob.gg/data/%s/tree/data.json.br"

func TreeURL(version string) string {
	return fmt.Sprintf(treeBase, version)
}

func rawURL(ctx context.Context, version string, name string) string {
	t, ok := fields.RowType("raw/" + name)
	if !ok {
		panic("raw: go-pob-data has no row type for raw/" + name)
	}
	return loader.VariantURL(ctx, version, "raw/"+name, reflect.New(t).Interface())
}

func prefetchURLs(ctx context.Context, version string) []string {
	urls := make([]string, 0, len(tables)+len(deferredTables)+len(timelessTables)+1)
	for _, t := range slices.Concat(tables, deferredTables) {
		urls = append(urls, rawURL(ctx, version, t.Name))
	}
	for _, name := range timelessTables {
		urls = append(urls, rawURL(ctx, version, name))
	}
	return append(urls, TreeURL(version))
}

// prefetch fetches every url at once; fetched reports each finished url, counted, in finishing order.
func prefetch(ctx context.Context, cache assetcache.AssetLoaderWrapper, urls []string, fetched func(done, total int)) map[string][]byte {
	out := make(map[string][]byte, len(urls))
	var mu sync.Mutex
	var wg sync.WaitGroup
	done := 0
	for _, url := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b := prefetchOne(ctx, cache, url)
			mu.Lock()
			defer mu.Unlock()
			if len(b) > 0 {
				out[url] = b
			}
			done++
			if fetched != nil {
				fetched(done, len(urls))
			}
		}()
	}
	wg.Wait()
	return out
}

func prefetchOne(ctx context.Context, cache assetcache.AssetLoaderWrapper, url string) []byte {
	if cache.Exists(url) {
		if b, err := cache.Get(url); err == nil && len(b) > 0 {
			return b
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if err := cache.Set(url, b); err != nil {
		slog.Warn("failed to store prefetched asset", slog.String("url", url), slog.Any("error", err))
	}
	return b
}
