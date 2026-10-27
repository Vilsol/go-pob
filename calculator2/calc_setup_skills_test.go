package calculator2

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/poe2skills"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

func readTSV(t *testing.T, path string, cols int) [][]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows [][]string
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != cols {
			t.Fatalf("%s: %q has %d columns, want %d", path, line, len(f), cols)
		}
		rows = append(rows, f)
	}
	if len(rows) == 0 {
		t.Fatalf("%s: no rows", path)
	}
	return rows
}

// TestGetGemStatRequirement grades the gem attribute requirement formula against PoB-PoE2's own.
func TestGetGemStatRequirement(t *testing.T) {
	for _, f := range readTSV(t, "testdata/gem_stat_requirement.tsv", 4) {
		level, _ := strconv.ParseFloat(f[0], 64)
		multi, _ := strconv.ParseFloat(f[1], 64)
		want, _ := strconv.ParseFloat(f[3], 64)
		if got := getGemStatRequirement(level, multi, f[2] == "1"); got != want {
			t.Errorf("getGemStatRequirement(%v, %v, %v) = %v, PoB %v", level, multi, f[2], got, want)
		}
	}
}

// TestDoesTypeExpressionMatch grades the skill type postfix evaluation against PoB-PoE2's own.
func TestDoesTypeExpressionMatch(t *testing.T) {
	set := func(s string) map[string]bool {
		if s == "" {
			return nil
		}
		m := map[string]bool{}
		for _, w := range strings.Fields(s) {
			m[w] = true
		}
		return m
	}
	for _, f := range readTSV(t, "testdata/type_expression_match.tsv", 4) {
		got := doesTypeExpressionMatch(strings.Fields(f[0]), set(f[1]), set(f[2]))
		if strconv.FormatBool(got) != f[3] {
			t.Errorf("doesTypeExpressionMatch(%q, %q, %q) = %v, PoB %s", f[0], f[1], f[2], got, f[3])
		}
	}
}

// CalcSetup.lua 564-568: a gemRequirements entry matches only a gem whose requirement exceeds it.
func TestApplyGemModsGemRequirements(t *testing.T) {
	prop := mod.GemProperty{Keyword: utils.Ptr("all"), Key: "level", Value: 2, KeyOfScaledMod: "value", GemRequirements: map[string]float64{"reqInt": 1}}
	for _, c := range []struct {
		reqInt float64
		want   int
	}{{100, 3}, {0, 1}, {1, 1}} {
		effect := &GemEffect{Level: 1, GemData: &poe2skills.Gem{Data: &poe2.Gem{Name: "Probe", ReqInt: c.reqInt}}}
		applyGemMods(effect, []any{prop})
		if effect.Level != c.want {
			t.Errorf("reqInt %v: level %d, want %d", c.reqInt, effect.Level, c.want)
		}
	}
}
