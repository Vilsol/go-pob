package calculator

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

const processGemLevelExpected = "../testdata/oracle/process_gem_level_expected.json"

func TestProcessGemLevelAgainstPoB(t *testing.T) {
	raw, err := os.ReadFile(processGemLevelExpected)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]map[string]int
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) < 400 {
		t.Fatalf("only %d gems in %s", len(want), processGemLevelExpected)
	}
	checked, wrong := 0, 0
	for gemID, rows := range want {
		for key, w := range rows {
			mode, level, _ := strings.Cut(key, ":")
			lvl, _ := strconv.Atoi(level)
			build := &pob.PathOfBuilding{Skills: pob.Skills{DefaultGemLevel: &mode}, Build: pob.Build{Level: lvl}}
			checked++
			if got := ProcessGemLevel(build, gemID, false); got != w {
				wrong++
				if wrong <= 10 {
					t.Errorf("%s %s: got %d, PoB %d", gemID, key, got, w)
				}
			}
		}
	}
	if wrong > 0 {
		t.Errorf("%d of %d wrong", wrong, checked)
	}
}

// TestDefaultGemLevelMode: Load's rule for the stored value.
func TestDefaultGemLevelMode(t *testing.T) {
	str := func(s string) *string { return &s }
	cases := []struct {
		skills pob.Skills
		want   string
	}{
		{pob.Skills{}, "normalMaximum"},
		{pob.Skills{DefaultGemLevel: str("20")}, "normalMaximum"},
		{pob.Skills{DefaultGemLevel: str("awakenedMaximum")}, "awakenedMaximum"},
		{pob.Skills{DefaultGemLevel: str("levelOne"), MatchGemLevelToCharacterLevel: true}, "characterLevel"},
		{pob.Skills{DefaultGemLevel: str("bogus")}, "normalMaximum"},
	}
	for _, c := range cases {
		if got := DefaultGemLevelMode(c.skills); got != c.want {
			t.Errorf("%+v: got %s, want %s", c.skills, got, c.want)
		}
	}
	build := &pob.PathOfBuilding{Skills: pob.Skills{DefaultGemLevel: str("awakenedMaximum")}}
	if got := ProcessGemLevel(build, "Metadata/Items/Gems/SkillGemFireball", true); got != 1 {
		t.Errorf("imbued: got %d", got)
	}
}
