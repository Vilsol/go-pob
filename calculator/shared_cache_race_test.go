package calculator

import (
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestSharedCacheConcurrentCalculations runs calculations of several builds at once through ONE cache.
func TestSharedCacheConcurrentCalculations(t *testing.T) {
	files := []string{"0280-elementalist-fireball.xml", "0001-necromancer-unknown.xml", "0011-saboteur-cyclone.xml"}
	want := make([]string, len(files))
	for i, f := range files {
		want[i] = envSnapshot(BuildOutputWithCache(&Calculator{PoB: parseCorpusBuild(t, f)}, OutputModeMain, NewEnvironmentCache()))
	}

	for _, shared := range []struct {
		name  string
		cache *EnvironmentCache
	}{{"fresh", NewEnvironmentCache()}, {"variant", &EnvironmentCache{items: make(map[string]*Item)}}} {
		const rounds = 4
		got := make([]string, len(files)*rounds)
		parsed := make([]*pob.PathOfBuilding, len(got))
		for i := range parsed {
			parsed[i] = parseCorpusBuild(t, files[i%len(files)])
		}
		var wg sync.WaitGroup
		for i := range got {
			wg.Add(1)
			go func() {
				defer wg.Done()
				got[i] = envSnapshot(BuildOutputWithCache(&Calculator{PoB: parsed[i]}, OutputModeMain, shared.cache))
			}()
		}
		wg.Wait()
		for i, g := range got {
			if g != want[i%len(files)] {
				t.Errorf("%s cache: %s (task %d) differs from its serial calculation", shared.name, files[i%len(files)], i)
			}
		}
	}
}
