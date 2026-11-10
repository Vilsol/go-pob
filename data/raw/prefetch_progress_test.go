package raw

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/storage"
)

// TestPrefetchCountsEveryURL holds the progress prefetch reports to one call per url, counting up to the url count,
// failed fetches included.
func TestPrefetchCountsEveryURL(t *testing.T) {
	urls := make([]string, 6)
	for i := range urls {
		urls[i] = fmt.Sprintf("http://127.0.0.1:1/prefetch-progress-%d", i)
	}
	var mu sync.Mutex
	var done []int
	prefetch(context.Background(), assetcache.AssetLoaderWrapper{Storage: storage.Get()}, urls, func(d, total int) {
		mu.Lock()
		defer mu.Unlock()
		if total != len(urls) {
			t.Errorf("total %d, want %d", total, len(urls))
		}
		done = append(done, d)
	})
	if want := []int{1, 2, 3, 4, 5, 6}; !slices.Equal(done, want) {
		t.Fatalf("reported %v, want %v", done, want)
	}
}
