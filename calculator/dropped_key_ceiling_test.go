package calculator

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
	"github.com/Vilsol/go-pob/utils"
)

const droppedKeyDistinctCeiling = 37

const droppedKeyCorpusPrefix = 0

// TestDroppedKeyClassesAreRatcheted restores the consumer those four comments promise.
func TestDroppedKeyClassesAreRatcheted(t *testing.T) {
	t.Parallel()

	if os.Getenv("CORPUS") == "" {
		t.Skip("set CORPUS=true — this walks the corpus")
	}
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	_, names := corpusNamesForLoadedData(t)
	n := droppedKeyCorpusPrefix
	if n == 0 {
		n = len(names)
	}
	if v := os.Getenv("DROPPED_KEY_BUILDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}
	if n > len(names) {
		n = len(names)
	}

	utils.ResetDroppedKeys()
	defer utils.ResetDroppedKeys()

	for _, name := range names[:n] {
		func() {
			defer func() { _ = recover() }()
			build := loadCorpusBuild(t, name)
			_ = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, corpusCacheForTest(t))
		}()
	}

	total, byKey := utils.DroppedKeys()
	if len(byKey) <= droppedKeyDistinctCeiling {
		t.Logf("dropped-key classes: %d distinct over %d builds (%d occurrences), ceiling %d",
			len(byKey), n, total, droppedKeyDistinctCeiling)
		return
	}

	var b strings.Builder
	for i, e := range byKey {
		if i >= 20 {
			b.WriteString("      ...\n")
			break
		}
		b.WriteString("      " + e.Op + " " + e.Key + " -> " + e.Type + "\n")
	}
	t.Errorf("dropped-key classes: %d distinct over %d builds (%d occurrences), ceiling %d\n"+
		"    A dropped key is an input that never arrives, so no value-based gate can see it.\n"+
		"    Each line is a string key resolving onto NO struct field of its target type:\n%s"+
		"    Add the field, or close the emitter. Do NOT raise the ceiling to make this pass.",
		len(byKey), n, total, droppedKeyDistinctCeiling, b.String())
}
