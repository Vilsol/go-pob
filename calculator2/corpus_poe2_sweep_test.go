package calculator2

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/pob2"
)

// The PoE2 corpus tests that calculate a corpus the same way share one calculation per build: the
// first of them to run prepares every selected one, calculates each build once and hands it to each
// test's visitor, so the later tests only report. A calculated Environment is too large to keep for a
// whole corpus, so visitors keep only what their report reads.

// poe2Calc is one corpus build, read, parsed and calculated.
type poe2Calc struct {
	build *pob2.PathOfBuilding2
	env   *Environment
	// err is the first of the read, parse and calculation errors; a panic in BuildOutput is
	// "panic: <value>" with the value in panicked.
	err      error
	panicked any
}

// poe2Part is one corpus a test calculates, with or without items.
type poe2Part struct {
	corpus  string
	noItems bool
	// names are build file names under corpus; visit gets names[i] on worker w < poe2Workers().
	names []string
	visit func(w, i int, c *poe2Calc)
	done  bool
}

type poe2Job struct {
	parts  []*poe2Part
	report func(t *testing.T)
}

// a prep returns the job, or a poe2SkipError, or an error the test fails on
type poe2Prep func() (*poe2Job, error)

type poe2SkipError string

func (s poe2SkipError) Error() string { return string(s) }

type poe2Consumer struct {
	test string
	prep poe2Prep
}

// poe2Consumers in the order their visitors see a build. Breakdowns goes last: it calculates the CALCS
// environment from the main one.
var poe2Consumers = []poe2Consumer{
	{"TestCorpusPoE2VsOracle", prepPoE2VsOracle},
	{"TestCorpusPoE2OffenceVsOracle", prepPoE2Offence},
	{"TestCorpusPoE2SkillsOnVsOracle", prepPoE2SkillsOn},
	{"TestCorpusPoE2OffenceSkillDataVsDumps", prepPoE2OffenceSkillData},
	{"TestCorpusPoE2RadiusJewelsVsOracle", prepPoE2RadiusJewels},
	{"TestCorpusPoE2ItemsVsOracle", prepPoE2Items},
	{"TestCorpusPoE2MainSkillVsDumps", prepPoE2MainSkill},
	{"TestCorpusPoE2SkillModListVsDumps", prepPoE2SkillModList},
	{"TestCorpusPoE2BreakdownsVsDumps", prepPoE2Breakdowns},
}

var poe2Jobs = map[string]*poe2Job{}

// poe2Selected reports whether -test.run and -test.skip select test, matching their first level only.
func poe2Selected(test string) bool {
	matches := func(name string) bool {
		f := flag.Lookup(name)
		if f == nil || f.Value.String() == "" {
			return false
		}
		ok, err := regexp.MatchString(strings.Split(f.Value.String(), "/")[0], test)
		return err == nil && ok
	}
	run := flag.Lookup("test.run")
	return (run == nil || run.Value.String() == "" || matches("test.run")) && !matches("test.skip")
}

func poe2RecNames(recs []poe2OracleRecord) []string {
	names := make([]string, len(recs))
	for i, r := range recs {
		names[i] = r.Build
	}
	return names
}

// runPoE2Job runs test's job, calculating each of its corpora once for every selected test that reads it.
func runPoE2Job(t *testing.T, test string) {
	t.Helper()
	job := poe2Jobs[test]
	if job == nil {
		var prep poe2Prep
		for _, c := range poe2Consumers {
			if c.test == test {
				prep = c.prep
			}
		}
		j, err := prep()
		var skip poe2SkipError
		if errors.As(err, &skip) {
			t.Skip(string(skip))
		}
		if err != nil {
			t.Fatal(err)
		}
		job = j
		poe2Jobs[test] = job
	}
	defer delete(poe2Jobs, test)
	for _, p := range job.parts {
		if !p.done {
			sweepPoE2(p)
		}
	}
	job.report(t)
}

// sweepPoE2 calculates first's corpus once for first and every other selected, not yet run test that reads it the same way.
func sweepPoE2(first *poe2Part) {
	var parts []*poe2Part
	for _, c := range poe2Consumers {
		job := poe2Jobs[c.test]
		if job == nil {
			if !poe2Selected(c.test) {
				continue
			}
			j, err := c.prep()
			if err != nil {
				continue
			}
			job = j
			poe2Jobs[c.test] = job
		}
		for _, p := range job.parts {
			if !p.done && p.corpus == first.corpus && p.noItems == first.noItems {
				parts = append(parts, p)
			}
		}
	}
	type want struct {
		part *poe2Part
		i    int
	}
	wants := map[string][]want{}
	var names []string
	for _, p := range parts {
		for i, n := range p.names {
			if wants[n] == nil {
				names = append(names, n)
			}
			wants[n] = append(wants[n], want{p, i})
		}
	}
	tree, _ := poe2TestTree()
	work := make(chan string)
	var wg sync.WaitGroup
	for w := range poe2Workers() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range work {
				c := calculatePoE2(tree, filepath.Join("..", "testdata", first.corpus, n), first.noItems)
				for _, x := range wants[n] {
					x.part.visit(w, x.i, c)
				}
			}
		}()
	}
	for _, n := range names {
		work <- n
	}
	close(work)
	wg.Wait()
	for _, p := range parts {
		p.done = true
	}
}

func calculatePoE2(tree *tree2.PassiveTree, path string, noItems bool) *poe2Calc {
	c := &poe2Calc{}
	raw, err := os.ReadFile(path)
	if err == nil {
		c.build, err = pob2.ParseBuild(raw)
	}
	if err == nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					c.panicked = r
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			opts := Options{Tree: tree}
			if !noItems {
				opts.Items = NewItemsTab(c.build, tree, nil)
			}
			c.env, err = BuildOutput(c.build, opts)
		}()
	}
	c.err = err
	return c
}

var poe2Oracles sync.Map

// sharedPoE2Oracle loads an oracle blob once for every test that reads it; they only read the records.
func sharedPoE2Oracle(path string) ([]poe2OracleRecord, error) {
	load, _ := poe2Oracles.LoadOrStore(path, sync.OnceValues(func() ([]poe2OracleRecord, error) { return loadPoE2Oracle(path) }))
	return load.(func() ([]poe2OracleRecord, error))()
}
