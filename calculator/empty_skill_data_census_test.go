package calculator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
)

func typesToSkillTypesOf(ge *poe.GrantedEffect) map[data.SkillType]bool {
	if as := ge.GetActiveSkill(); as != nil {
		return TypesToSkillTypes(as.GetActiveSkillTypes())
	}
	return map[data.SkillType]bool{}
}

func skillLadderID(s *ActiveSkill) string {
	if s == nil || s.ActiveEffect == nil {
		return "(no ActiveEffect)"
	}
	if s.ActiveEffect.GrantedEffect == nil {
		return "(no GrantedEffect)"
	}
	return s.ActiveEffect.GrantedEffect.ID()
}

const (
	emptySkillNoEffect   = "NO-EFFECT"
	emptySkillEmptyStats = "EMPTY-STATS"
	emptySkillShortStats = "SHORT-STATS"
	emptySkillEmptyTypes = "EMPTY-TYPES"
	emptySkillShortTypes = "MISSING-TYPE"
	emptySkillOK         = "OK"
)

type emptySkillRow struct {
	pobID, goID string
	name, loc   string
	hidden      bool
	support     bool
	shell       bool
	classes     []string
	ourStats    int
	pobStats    int
	ourTypes    int
	pobTypes    int
	missingType []string
	builds      int
}

// TestEmptySkillDataCensus enumerates, over PoB's WHOLE hand-written skill table, every granted effect go-pob builds EMPTY where PoB declares content.
func TestEmptySkillDataCensus(t *testing.T) {
	requireOperatorRun(t, "report-only census; `task empty-skill-data`")
	table, err := loadPoBSkillTable()
	if err != nil {
		t.Fatalf("cannot load PoB's skill table: %v", err)
	}

	goIDFor := map[string]string{}
	for goID, pobID := range skillFlagIDAliases {
		goIDFor[pobID] = goID
	}

	sized := os.Getenv("CORPUS") == "true"
	var buildCount map[string]int
	if sized {
		buildCount = countBuildsConstructingSkills(t)
	}

	rows := make([]emptySkillRow, 0, len(table.Skills))
	for pobID, entry := range table.Skills {
		goID := pobID
		if alias, ok := goIDFor[pobID]; ok {
			goID = alias
		}

		row := emptySkillRow{
			pobID: pobID, goID: goID,
			name: entry.Name, loc: entry.Loc,
			hidden: entry.Hidden, support: entry.Support,
			pobStats: len(entry.Stats), pobTypes: len(entry.SkillTypes),
			builds: -1,
		}
		if sized {
			row.builds = buildCount[goID]
		}

		ge := poe.GrantedEffectByID(goID)
		if ge == nil {
			ge = pobOnlySkillEffect(pobID)
			if ge == nil {
				row.classes = append(row.classes, emptySkillNoEffect)
				rows = append(rows, row)
				continue
			}
			row.shell = true
		}

		calc := raw2.GetCalculatedGrantedEffect(ge)
		row.ourStats = len(calc.GetCalculatedStats())
		switch {
		case row.pobStats > 0 && row.ourStats == 0:
			row.classes = append(row.classes, emptySkillEmptyStats)
		case row.pobStats > row.ourStats:
			row.classes = append(row.classes, emptySkillShortStats)
		}

		ourTypes := typesToSkillTypesOf(ge)
		for _, ty := range raw2.SkillHiddenLevels[ge.ID].SkillTypes {
			ourTypes[data.SkillType(ty)] = true
		}
		for ty := range pobOnlySkillTypes(pobOnlySkillEffect(ge.ID)) {
			ourTypes[ty] = true
		}
		row.ourTypes = len(ourTypes)
		if !entry.Support {
			for _, want := range entry.SkillTypes {
				if !ourTypes[data.SkillType(want)] {
					row.missingType = append(row.missingType, want)
				}
			}
			sort.Strings(row.missingType)
			switch {
			case row.pobTypes > 0 && row.ourTypes == 0:
				row.classes = append(row.classes, emptySkillEmptyTypes)
			case len(row.missingType) > 0:
				row.classes = append(row.classes, emptySkillShortTypes)
			}
		}

		if len(row.classes) == 0 {
			row.classes = append(row.classes, emptySkillOK)
		}
		rows = append(rows, row)
	}

	reportEmptySkillCensus(table, rows, sized)
}

func countBuildsConstructingSkills(t *testing.T) map[string]int {
	t.Helper()

	entries, err := os.ReadDir(corpusDirForLoadedData(t))
	if err != nil {
		t.Fatalf("cannot size the population: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("the loaded version's corpus holds no builds, so nothing can be sized. Refusing rather " +
			"than reporting every skill as reaching 0 builds.")
	}
	sort.Strings(names)

	named := os.Getenv("SKILL")

	var (
		mu               sync.Mutex
		count            = map[string]int{}
		namedSkillBuilds []string
	)
	runConcurrently(len(names), evalWorkers(), func(i int, cache *EnvironmentCache) {
		raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), names[i]))
		if err != nil {
			return
		}
		build, err := builds.ParseBuild(raw)
		if err != nil {
			return
		}
		var env *Environment
		func() {
			defer func() { _ = recover() }()
			env = BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, cache)
		}()
		if env == nil {
			return
		}

		seen := map[string]bool{}
		collect := func(list []*ActiveSkill) {
			for _, s := range list {
				if id := skillLadderID(s); !strings.HasPrefix(id, "(") {
					seen[id] = true
				}
			}
		}
		if env.Player != nil {
			collect(env.Player.ActiveSkillList)
		}
		if env.Minion != nil && env.Minion.Actor != nil {
			collect(env.Minion.ActiveSkillList)
		}
		collect(env.AuxSkillList)

		mu.Lock()
		for id := range seen {
			count[id]++
		}
		if named != "" && seen[named] {
			namedSkillBuilds = append(namedSkillBuilds, strings.TrimSuffix(names[i], ".xml"))
		}
		mu.Unlock()
	})
	sort.Strings(namedSkillBuilds)

	if err := sizingIsCredible(len(count), len(names)); err != nil {
		t.Fatal(err)
	}
	t.Logf("sizing: %d distinct granted-effect ids constructed across %d corpus builds", len(count), len(names))

	if named != "" {
		if len(namedSkillBuilds) == 0 {
			t.Logf("SKILL=%q: NO corpus build constructs it. That is an answer, not an error — "+
				"but check the spelling against the go-pob-data id (the census's goID column), "+
				"not PoB's, since the two differ for the aliased ids.", named)
		} else {
			t.Logf("builds constructing %q (%d): %s", named, len(namedSkillBuilds), strings.Join(namedSkillBuilds, " "))
		}
	}

	return count
}

const minDistinctConstructedSkills = 200

func sizingIsCredible(distinct, builds int) error {
	if distinct >= minDistinctConstructedSkills {
		return nil
	}
	return fmt.Errorf("the population walk found only %d distinct constructed skill ids over %d "+
		"builds, which cannot be right — every BUILDS column would read 0 and the census would "+
		"report the whole class as costing nothing. Refusing rather than reporting that",
		distinct, builds)
}

// TestSizingGuardRefusesAnEmptyWalk proves the guard above discriminates.
func TestSizingGuardRefusesAnEmptyWalk(t *testing.T) {
	if err := sizingIsCredible(0, 2027); err == nil {
		t.Error("a walk that constructed NOTHING was accepted as a measurement")
	}
	if err := sizingIsCredible(minDistinctConstructedSkills-1, 2027); err == nil {
		t.Error("a walk just under the floor was accepted")
	}
	if err := sizingIsCredible(449, 2027); err != nil {
		t.Errorf("the measured healthy value was refused: %v", err)
	}
}

func reportEmptySkillCensus(table *pobSkillTable, rows []emptySkillRow, sized bool) {
	byClass := map[string][]emptySkillRow{}
	for _, r := range rows {
		for _, c := range r.classes {
			byClass[c] = append(byClass[c], r)
		}
	}

	fmt.Printf("\n=== EMPTY SKILL DATA CENSUS ===\n")
	fmt.Printf("PoB v2.18.1 declares %d skills (%d hidden, %d support). Compared against what\n"+
		"raw.GetCalculatedGrantedEffect produces from go-pob-data.\n",
		table.SkillCount, table.HiddenCount, table.SupportCount)
	if sized {
		fmt.Printf("POPULATION MEASURED over the 2027-build corpus, by walking the BUILT ENVIRONMENT\n" +
			"(player + minion + aux skill lists). A `builds` of 0 means no corpus build\n" +
			"constructs it, so the row costs nothing today.\n")
	} else {
		fmt.Printf("⚠️ POPULATION UNMEASURED — every `builds` column reads `?`. Re-run with\n" +
			"CORPUS=true to size it. Do NOT read `?` as 0: this census enumerates the whole of\n" +
			"PoB's table, most of which no build can reach, so the counts are the entire worklist.\n")
	}

	fmt.Printf("\n%-14s %6s   what it means\n", "CLASS", "SKILLS")
	for _, c := range []string{emptySkillNoEffect, emptySkillEmptyStats, emptySkillShortStats, emptySkillEmptyTypes, emptySkillShortTypes, emptySkillOK} {
		fmt.Printf("%-14s %6d   %s\n", c, len(byClass[c]), emptySkillClassMeaning(c))
	}

	for _, c := range []string{emptySkillNoEffect, emptySkillEmptyStats, emptySkillShortStats, emptySkillEmptyTypes, emptySkillShortTypes} {
		list := byClass[c]
		if len(list) == 0 {
			continue
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].builds != list[j].builds {
				return list[i].builds > list[j].builds
			}
			return list[i].pobID < list[j].pobID
		})
		reached := 0
		for _, r := range list {
			if r.builds > 0 {
				reached++
			}
		}
		fmt.Printf("\n--- %s (%d skills, %s)\n", c, len(list), reachedPhrase(reached, sized))
		fmt.Printf("%-40s %-30s %7s %8s %8s  %s\n", "PoB ID", "NAME", "BUILDS", "STATS", "TYPES", "src/Data/Skills")
		for _, r := range list {
			flags := ""
			if r.hidden {
				flags += " hidden"
			}
			if r.support {
				flags += " support"
			}
			if r.shell {
				flags += " PoB-ONLY-SHELL"
			}
			fmt.Printf("%-40s %-30s %7s %8s %8s  %s%s\n",
				r.pobID, truncateSkillName(r.name, 30), buildsColumn(r.builds),
				fmt.Sprintf("%d/%d", r.ourStats, r.pobStats),
				fmt.Sprintf("%d/%d", r.ourTypes, r.pobTypes),
				r.loc, flags)
			if len(r.missingType) > 0 {
				fmt.Printf("%-40s   missing: %s\n", "", strings.Join(r.missingType, " "))
			}
		}
	}

	fmt.Printf("\nHOW TO READ A ROW. STATS and TYPES are ours/PoB's. Take the top of\n" +
		"EMPTY-STATS and EMPTY-TYPES with a nonzero BUILDS column and walk one with\n" +
		"`task diag-skill BUILD=<b> SKILL=<id>` — that shows the same gap in its build's\n" +
		"context, with the rung it breaks at.\n")
}

func emptySkillClassMeaning(c string) string {
	switch c {
	case emptySkillNoEffect:
		return "go-pob-data has NO granted effect — the skill cannot be built at all"
	case emptySkillEmptyStats:
		return "builds and holds NOTHING: PoB declares stats, our stat list is empty"
	case emptySkillShortStats:
		return "builds with FEWER stats than PoB declares"
	case emptySkillEmptyTypes:
		return "no ActiveSkillTypes: every SkillType-tagged mod aimed at it misses"
	case emptySkillShortTypes:
		return "PoB declares a skillType we do not have"
	default:
		return "our data agrees with PoB's declaration"
	}
}

func reachedPhrase(reached int, sized bool) string {
	if !sized {
		return "population UNMEASURED — re-run with CORPUS=true"
	}
	if reached == 0 {
		return "NONE reached by any corpus build"
	}
	return fmt.Sprintf("%d reached by at least one corpus build", reached)
}

func buildsColumn(n int) string {
	if n < 0 {
		return "?"
	}
	return fmt.Sprint(n)
}

func truncateSkillName(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
