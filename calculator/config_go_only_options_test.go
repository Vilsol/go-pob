package calculator

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestGoPobOnlyConfigOptionsStayInert guards the one direction that would turn goPobOnlyConfigOptions from dead weight into a defect nothing else here could see.
func TestGoPobOnlyConfigOptionsStayInert(t *testing.T) {
	if len(goPobOnlyConfigOptions) == 0 {
		t.Fatal("goPobOnlyConfigOptions is empty: every assertion below would pass vacuously")
	}
	if !sort.StringsAreSorted(goPobOnlyConfigOptions) {
		t.Error("goPobOnlyConfigOptions is not sorted, which makes a diff of it unreadable")
	}

	inOrder := map[string]bool{}
	for _, n := range configOrder {
		inOrder[n] = true
	}
	seeded := map[string]string{}
	for _, n := range configDefaults {
		seeded[n] = "configDefaults"
	}
	for n := range configIndexDefaults {
		seeded[n] = "configIndexDefaults"
	}
	for _, n := range configPlaceholderVars {
		seeded[n] = "configPlaceholderVars"
	}

	for _, name := range goPobOnlyConfigOptions {
		if !inOrder[name] {
			t.Errorf("%s is listed as a go-pob-only config option but is not in configOrder, so it "+
				"is not an option at all — remove it from the list or from `configurations`", name)
		}
		if _, ok := configurations[name]; !ok {
			t.Errorf("%s is listed as a go-pob-only config option but has no handler", name)
		}
		if where, ok := seeded[name]; ok {
			t.Errorf("%s is in %s, so it now applies to EVERY build even though the pinned PoB "+
				"declares no such option and no export can set it — that is over-application on "+
				"100%% of the corpus, which no value gate can isolate", name, where)
		}
	}

	if len(seeded) == 0 {
		t.Fatal("no config is seeded by any default list; the over-application check is vacuous")
	}
}

// TestGoPobOnlyConfigOptionsAreAbsentFromTheCorpus is the other half, and it is the one that would notice a corpus promotion making these live.
func TestGoPobOnlyConfigOptionsAreAbsentFromTheCorpus(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	files, err := filepath.Glob(filepath.Join(corpusDirForLoadedData(t), "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: no corpus XMLs (%v)", err)
	}

	needles := make([]string, 0, len(goPobOnlyConfigOptions))
	for _, n := range goPobOnlyConfigOptions {
		needles = append(needles, `name="`+n+`"`)
	}

	const control = `name="conditionEnemyCursed"`
	controlHits := 0
	hits := map[string]int{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		body := configSectionOf(string(raw))
		if body == "" {
			continue
		}
		if strings.Contains(body, control) {
			controlHits++
		}
		for i, needle := range needles {
			if strings.Contains(body, needle) {
				hits[goPobOnlyConfigOptions[i]]++
			}
		}
	}
	if controlHits == 0 {
		t.Fatalf("REFUSING: the control %s appears in 0 of %d builds, so this scan found nothing "+
			"because it is broken, not because the options are absent", control, len(files))
	}
	knownCarriers := map[string]int{"playerCannotLeech": 1}
	for name, n := range hits {
		if want, ok := knownCarriers[name]; ok {
			if n != want {
				t.Errorf("%s is now set by %d corpus builds, not the declared %d: re-review "+
					"whether a go-pob-only option should be reachable at all", name, n, want)
			}
			continue
		}
		t.Errorf("%s is set by %d corpus builds, so it is no longer inert: grade its handler "+
			"against upstream or delete it", name, n)
	}
	for name, want := range knownCarriers {
		if hits[name] == 0 {
			t.Errorf("%s is declared as set by %d corpus builds and is now set by none — the scan "+
				"or the corpus changed, and a stale declaration hides the next real carrier",
				name, want)
		}
	}
}
