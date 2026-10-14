package calculator

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds/parsecache"
)

// TestConfigPlaceholdersAgainstPoB is `task config-placeholders`.
func TestConfigPlaceholdersAgainstPoB(t *testing.T) {
	requireOperatorRun(t, "reads oracle/config_placeholders.lua output; `task config-placeholders`")
	path := requireTaskInput(t, "config-placeholders", "CONFIG_PLACEHOLDERS")[0]
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var pobByBuild map[string]map[string]any
	if err := json.Unmarshal(raw, &pobByBuild); err != nil {
		t.Fatal(err)
	}
	if len(pobByBuild) == 0 {
		t.Fatal("REFUSING: no builds in the PoB placeholder dump")
	}
	corpusDir := corpusDirForLoadedData(t)
	perKey := map[string][2]int{}
	examples := map[string]string{}
	graded := 0
	for _, name := range slices.Sorted(maps.Keys(pobByBuild)) {
		want := pobByBuild[name]
		if _, failed := want["error"]; failed {
			continue
		}
		xml, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			t.Fatal(err)
		}
		build, err := parsecache.ParseCached(xml)
		if err != nil {
			t.Fatal(err)
		}
		got := ConfigPlaceholders(build)
		graded++
		for _, k := range slices.Sorted(maps.Keys(want)) {
			w, ok := want[k].(float64)
			if !ok {
				continue
			}
			c := perKey[k]
			c[0]++
			if g, present := got[k]; !present || !oracleClose(g, w, oracleDumpTolerance) {
				c[1]++
				if _, seen := examples[k]; !seen {
					examples[k] = fmt.Sprintf("%s: ours %v (present %t), PoB %v", name, g, present, w)
				}
			}
			perKey[k] = c
		}
	}
	var b strings.Builder
	wrong := 0
	for _, k := range slices.Sorted(maps.Keys(perKey)) {
		c := perKey[k]
		if c[1] > 0 {
			wrong++
			fmt.Fprintf(&b, "  %-36s %4d / %4d differ  e.g. %s\n", k, c[1], c[0], examples[k])
		}
	}
	fmt.Printf("config-placeholders: %d builds, %d placeholder keys, %d differ somewhere\n%s", graded, len(perKey), wrong, b.String())
}
