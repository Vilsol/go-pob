package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestCompareSkillBreakdownMatchesPoB: The Compare tab's skill breakdown panel.
func TestCompareSkillBreakdownMatchesPoB(t *testing.T) {
	t.Parallel()
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/ui-misc/compare_skill_breakdown.jsonl.gz")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	tableRe := regexp.MustCompile(`^table: 0x[0-9a-f]+\t`)
	keys := func(rows []string) []string {
		out := make([]string, len(rows))
		for i, r := range rows {
			v, rest, _ := strings.Cut(tableRe.ReplaceAllString(r, "?\t"), "\t")
			name, _, _ := strings.Cut(rest, "\t")
			out[i] = v + "\t" + name
		}
		return out
	}
	sorted := func(rows []string) []string {
		out := slices.Clone(rows)
		for i, r := range out {
			out[i] = tableRe.ReplaceAllString(r, "?\t")
		}
		slices.Sort(out)
		return out
	}
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 1<<22), 1<<22)
	n, same, seen, nonEmpty := 0, 0, 0, 0
	for sc.Scan() {
		var want struct {
			Build                    string
			SkillBuffs, SkillDebuffs []string
		}
		if err := json.Unmarshal(sc.Bytes(), &want); err != nil {
			t.Fatal(err)
		}
		n++
		c := &Calculator{PoB: loadCorpusBuild(t, want.Build)}
		cache := NewEnvironmentCache()
		BuildOutputWithCache(c, OutputModeMain, cache)
		env := BuildOutputWithCache(c, OutputModeCalcs, cache)
		for name, w := range map[string][]string{"SkillBuffs": want.SkillBuffs, "SkillDebuffs": want.SkillDebuffs} {
			seen++
			if len(w) > 0 {
				nonEmpty++
			}
			var got []string
			for _, r := range env.CompareSkillBreakdown(name) {
				got = append(got, r.Value+"\t"+r.Name+"\t"+r.Source+"\t"+r.SourceName)
			}
			if slices.Equal(keys(got), keys(w)) && slices.Equal(sorted(got), sorted(w)) {
				same++
			} else if testing.Verbose() {
				t.Logf("%s %s:\n ours %q\n PoB  %q", want.Build, name, got, w)
			}
		}
	}
	if n < 300 || nonEmpty < 100 {
		t.Fatalf("REFUSING: %d builds, %d non-empty panels in the fixture", n, nonEmpty)
	}
	t.Logf("%d of %d panels match the pin (%d non-empty)", same, seen, nonEmpty)
	if same < compareSkillBreakdownFloor {
		t.Errorf("only %d panels match; the floor is %d", same, compareSkillBreakdownFloor)
	}
}

const compareSkillBreakdownFloor = 756
