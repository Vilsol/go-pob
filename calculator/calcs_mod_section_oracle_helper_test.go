package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/builds"
)

func oracleModSectionTables(t *testing.T, file string) (*Environment, []modSectionCase) {
	t.Helper()
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/calcs_mod_section_expected.jsonl.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../testdata/corpus-3.29/" + file)
	if err != nil {
		t.Fatal(err)
	}
	build, err := builds.ParseBuild(raw)
	if err != nil {
		t.Fatal(err)
	}
	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeCalcs, NewEnvironmentCache())
	var cases []modSectionCase
	scanner := bufio.NewScanner(zr)
	scanner.Buffer(make([]byte, 1<<22), 1<<22)
	for scanner.Scan() {
		var c modSectionCase
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		if c.File == file {
			cases = append(cases, c)
		}
	}
	if len(cases) == 0 {
		t.Fatal("no PoB table for " + file)
	}
	return env, cases
}

func (c modSectionCase) data() ModSectionData {
	return ModSectionData{ModName: c.ModName, Multi: c.Multi, ModType: c.ModType, Cfg: c.Cfg, ModSource: c.ModSource, Enemy: c.Enemy, Actor: c.Actor}
}

func assertTablesMatch(t *testing.T, file string, keep func(c modSectionCase) bool) {
	t.Helper()
	env, cases := oracleModSectionTables(t, file)
	checked := 0
	for _, c := range cases {
		if !keep(c) {
			continue
		}
		var want, got []string
		for _, r := range c.Rows {
			want = append(want, modSectionRowText(r.Value, r.Name, r.Flags, r.Tags, r.Source, r.SourceName, r.Totals))
		}
		for _, r := range env.CalcsModSection(c.data(), false) {
			got = append(got, modSectionRowText(r.Value, r.Name, r.Flags, r.Tags, r.Source, r.SourceName, r.Totals))
		}
		slices.Sort(want)
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%v %s cfg=%s:\n got %q\nPoB %q", c.ModName, c.ModType, c.Cfg, got, want)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no table to check")
	}
}
