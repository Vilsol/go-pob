package raw

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/fields"
	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"
)

func mustRowType(t *testing.T, asset string) reflect.Type {
	t.Helper()
	rt, ok := fields.RowType(asset)
	if !ok {
		t.Fatalf("go-pob-data has no row type for %s", asset)
	}
	return rt
}

// TestDeferredTablesLoadOnFirstUse: after InitializeAll the deferred tables are still empty.
func TestDeferredTablesLoadOnFirstUse(t *testing.T) {
	if os.Getenv("DEFERRED_CHILD") == "1" {
		if err := InitializeAll(LatestVersion, nil); err != nil {
			t.Fatal(err)
		}
		if len(poe.Mods) != 0 || len(poe.MonsterVarieties) != 0 {
			t.Fatalf("deferred tables loaded eagerly: %d mods, %d monster varieties", len(poe.Mods), len(poe.MonsterVarieties))
		}
		for _, tb := range deferredTables {
			asset := "raw/" + tb.Name
			variant, ok := fields.MinVariant(LatestVersion, asset)
			if !ok || os.Getenv(FullAssetsEnv) != "" {
				variant = loader.TupleVariant(reflect.New(mustRowType(t, asset)).Interface())
			}
			if _, ok := lazyLoader().prefetched[loader.AssetURL(LatestVersion, asset, variant)]; !ok {
				t.Fatalf("%s is not held in memory as %s; its first use would go to the network or IndexedDB", tb.Name, variant)
			}
		}
		fmt.Printf("LOADED mods=%d monsterVarieties=%d\n", len(Mods()), len(MonsterVarieties()))
		return
	}
	if runtime.GOOS == "js" {
		t.Skip("needs a child process")
	}

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestDeferredTablesLoadOnFirstUse$", "-test.count=1", "-test.v") //nolint:gosec // re-runs this test binary
	cmd.Env = append(os.Environ(), "DEFERRED_CHILD=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child failed: %v\n%s", err, out)
	}
	var mods, varieties int
	for line := range strings.Lines(string(out)) {
		if _, err := fmt.Sscanf(strings.TrimSpace(line), "LOADED mods=%d monsterVarieties=%d", &mods, &varieties); err == nil {
			break
		}
	}
	if mods < 1000 || varieties < 1000 {
		t.Fatalf("first use loaded %d mods, %d monster varieties\n%s", mods, varieties, out)
	}
}
