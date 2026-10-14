package calculator

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestCalcsSkillBuffsMatchPoB: The Calcs tab's SkillBuffs / SkillDebuffs breakdowns.
func TestCalcsSkillBuffsMatchPoB(t *testing.T) {
	t.Parallel()
	requireGameData(t, "3.29")
	f, err := os.Open("../testdata/oracle/calcs_skill_buffs_expected.jsonl.gz")
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	rows := func(list []string) []string {
		out := make([]string, 0, len(list))
		for _, r := range list {
			r = luaTableValueRe.ReplaceAllString(r, "|LIST|table|")
			out = append(out, luaFlagValueRe.ReplaceAllString(r, "|FLAG|flag|"))
		}
		slices.Sort(out)
		return out
	}
	ours := func(lists map[string][]modResultRow, name string) []string {
		out := make([]string, 0, len(lists[name]))
		for _, r := range lists[name] {
			out = append(out, r.String())
		}
		slices.Sort(out)
		return out
	}
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 1<<22), 1<<22)
	n, same, seen := 0, 0, 0
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
		got := map[string][]modResultRow{}
		for k, v := range env.Player.breakdownModLists {
			for _, r := range v {
				got[k] = append(got[k], modResultRow{r.Mod, r.Value})
			}
		}
		for name, w := range map[string][]string{"SkillBuffs": rows(want.SkillBuffs), "SkillDebuffs": rows(want.SkillDebuffs)} {
			seen++
			if g := ours(got, name); slices.Equal(g, w) {
				same++
			} else if testing.Verbose() {
				t.Logf("%s %s:\n ours %q\n PoB  %q", want.Build, name, g, w)
			}
		}
	}
	if n < 300 {
		t.Fatalf("REFUSING: only %d builds in the fixture", n)
	}
	t.Logf("%d of %d SkillBuffs / SkillDebuffs lists match the pin", same, seen)
	if same < calcsSkillBuffsFloor {
		t.Errorf("only %d lists match; the floor is %d", same, calcsSkillBuffsFloor)
	}
}

const calcsSkillBuffsFloor = 753

var (
	luaTableValueRe = regexp.MustCompile(`\|LIST\|table: 0x[0-9a-f]+\|`)
	luaFlagValueRe  = regexp.MustCompile(`\|FLAG\|[^|]*\|`)
)

type modResultRow struct {
	m     mod.Mod
	value float64
}

func (r modResultRow) String() string {
	value := fmt.Sprintf("%.4f", r.value)
	switch r.m.Type() {
	case mod.TypeFlag:
		value = "flag"
	case mod.TypeList:
		value = "table"
	}
	return fmt.Sprintf("%s|%s|%s|%s", r.m.Name(), r.m.Type(), value, r.m.GetSource())
}
