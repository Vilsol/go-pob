package raw

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/storage"
)

type recordingCache struct {
	AssetLoaderWrapper
	mu   *sync.Mutex
	seen map[string]bool
}

func (r recordingCache) Exists(key string) bool {
	r.mu.Lock()
	r.seen[key] = true
	r.mu.Unlock()
	return r.AssetLoaderWrapper.Exists(key)
}

// TestPrefetchCoversInitializeAll holds prefetchURLs equal to the assets the loaders request.
func TestPrefetchCoversInitializeAll(t *testing.T) {
	if os.Getenv("PREFETCH_RECORD") == "1" {
		rec := recordingCache{AssetLoaderWrapper: AssetLoaderWrapper{Storage: storage.Get()}, mu: &sync.Mutex{}, seen: map[string]bool{}}
		for _, tb := range slices.Concat(tables, deferredTables) {
			if err := tb.Load(DataContext(context.Background()), LatestVersion, rec); err != nil {
				t.Fatal(err)
			}
		}
		if err := initTimelessSources(DataContext(context.Background()), LatestVersion, rec); err != nil {
			t.Fatal(err)
		}
		for url := range rec.seen {
			fmt.Println("ASSET " + url)
		}
		return
	}
	if runtime.GOOS == "js" {
		t.Skip("needs a child process")
	}

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestPrefetchCoversInitializeAll$", "-test.count=1") //nolint:gosec // re-runs this test binary
	cmd.Env = append(os.Environ(), "PREFETCH_RECORD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("recording child failed: %v\n%s", err, out)
	}

	var requested []string
	for line := range strings.Lines(string(out)) {
		if url, ok := strings.CutPrefix(strings.TrimSpace(line), "ASSET "); ok {
			requested = append(requested, url)
		}
	}
	if len(requested) < len(tables) {
		t.Fatalf("the child recorded %d assets; a recording that sees nothing makes any list pass\n%s", len(requested), out)
	}

	requested = append(requested, TreeURL(LatestVersion))
	want := prefetchURLs(DataContext(context.Background()), LatestVersion)
	slices.Sort(requested)
	slices.Sort(want)
	for _, url := range requested {
		if !slices.Contains(want, url) {
			t.Errorf("loaded but not prefetched (fetched one at a time on a cold load): %s", url)
		}
	}
	for _, url := range want {
		if !slices.Contains(requested, url) {
			t.Errorf("prefetched but never loaded (a wasted request on every cold load): %s", url)
		}
	}
}
