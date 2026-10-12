package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var censusInputRe = regexp.MustCompile(`<Input\s+[^>]*name="([^"]+)"`)

func configSectionOf(raw string) string {
	start := strings.Index(raw, "<Config")
	if start < 0 {
		return ""
	}
	open := strings.Index(raw[start:], ">")
	if open < 0 {
		return ""
	}
	start += open + 1
	end := strings.Index(raw[start:], "</Config>")
	if end < 0 {
		return ""
	}
	return raw[start : start+end]
}

func TestConfigCoverageCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census")
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	corpus := corpusDirForLoadedData(t)
	files, err := filepath.Glob(filepath.Join(corpus, "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("REFUSING: no corpus XMLs under %s (%v)", corpus, err)
	}

	used := map[string]int{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		body := configSectionOf(string(raw))
		if body == "" {
			continue
		}
		seen := map[string]bool{}
		for _, m := range censusInputRe.FindAllStringSubmatch(body, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				used[m[1]]++
			}
		}
	}
	if len(used) == 0 {
		t.Fatalf("REFUSING: read %d corpus files and found no <Input name=\"...\"> inside a "+
			"<Config> section — the element or the section this census reads has moved, and an "+
			"empty result would read as full coverage", len(files))
	}

	covered := map[string]bool{}
	inOrder := map[string]bool{}
	for _, name := range configOrder {
		inOrder[name] = true
	}
	for name := range configurations {
		if inOrder[name] {
			covered[name] = true
		}
	}
	for _, name := range configDefaults {
		covered[name] = true
	}
	for name := range configIndexDefaults {
		covered[name] = true
	}
	for _, name := range configPlaceholderVars {
		covered[name] = true
	}

	var halfWired []string
	for name := range configurations {
		if !inOrder[name] {
			halfWired = append(halfWired, name)
		}
	}
	sort.Strings(halfWired)

	type row struct {
		name   string
		builds int
	}
	var missing, orderedOnly []row
	coveredSet, missingSet, orderedOnlySet := 0, 0, 0
	for name, n := range used {
		if covered[name] {
			coveredSet++
			continue
		}
		if inOrder[name] {
			orderedOnlySet++
			orderedOnly = append(orderedOnly, row{name, n})
			continue
		}
		missingSet++
		missing = append(missing, row{name, n})
	}
	sort.Slice(orderedOnly, func(i, j int) bool {
		if orderedOnly[i].builds != orderedOnly[j].builds {
			return orderedOnly[i].builds > orderedOnly[j].builds
		}
		return orderedOnly[i].name < orderedOnly[j].name
	})
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].builds != missing[j].builds {
			return missing[i].builds > missing[j].builds
		}
		return missing[i].name < missing[j].name
	})

	total := 0
	for _, r := range missing {
		total += r.builds
	}
	orderedTotal := 0
	for _, r := range orderedOnly {
		orderedTotal += r.builds
	}
	fmt.Printf("=== CONFIG COVERAGE CENSUS === %d corpus builds; %d distinct config inputs used, "+
		"%d known here, %d in configOrder ONLY (read elsewhere, NOT work), %d MODELLED NOWHERE\n",
		len(files), len(used), coveredSet, orderedOnlySet, missingSet)
	fmt.Printf("build-settings summed over the modelled-nowhere configs: %d (an UPPER BOUND — a "+
		"config set to its default costs nothing); over the configOrder-only ones: %d\n",
		total, orderedTotal)
	if len(halfWired) > 0 {
		fmt.Printf("\n⚠️  INERT (%d) — a handler in `configurations` that configOrder never dispatches:\n", len(halfWired))
		for _, h := range halfWired {
			fmt.Printf("     %s\n", h)
		}
	}
	if len(orderedOnly) > 0 {
		fmt.Printf("\nIN configOrder, NO `configurations` HANDLER (%d) — read elsewhere or inert; "+
			"this census cannot tell which, so it does not claim either:\n", len(orderedOnly))
		for i, r := range orderedOnly {
			if i >= 15 {
				fmt.Printf("     ... %d further\n", len(orderedOnly)-15)
				break
			}
			fmt.Printf("  %8d  %s\n", r.builds, r.name)
		}
	}
	fmt.Printf("\nMODELLED NOWHERE — the worklist\n")
	fmt.Printf("\n%8s  %s\n", "BUILDS", "CONFIG")
	for i, r := range missing {
		if i >= 60 {
			fmt.Printf("... %d further configs not shown\n", len(missing)-60)
			break
		}
		fmt.Printf("%8d  %s\n", r.builds, r.name)
	}
}

// TestConfigSectionOfExcludesTheCalcsTab pins the reach of the census above.
func TestConfigSectionOfExcludesTheCalcsTab(t *testing.T) {
	const build = `<PathOfBuilding>
	<Calcs>
		<Input name="misc_buffMode" string="EFFECTIVE"/>
		<Input name="skill_number" number="2"/>
	</Calcs>
	<Config activeConfigSet="1">
		<Input name="conditionEnemyCursed" boolean="true"/>
	</Config>
</PathOfBuilding>`

	body := configSectionOf(build)
	if body == "" {
		t.Fatal("configSectionOf found no <Config> section: `<Config` carries attributes")
	}
	for _, calcsOnly := range []string{"misc_buffMode", "skill_number"} {
		if strings.Contains(body, calcsOnly) {
			t.Errorf("the <Calcs> input %s reached the config census", calcsOnly)
		}
	}
	if !strings.Contains(body, "conditionEnemyCursed") {
		t.Error("the <Config> section's own inputs did not survive the cut")
	}
	if got := configSectionOf(`<PathOfBuilding><Calcs><Input name="skill_number"/></Calcs></PathOfBuilding>`); got != "" {
		t.Errorf("a build with no <Config> yielded %q, not the empty string", got)
	}
}
