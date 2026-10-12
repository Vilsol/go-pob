package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var skillTypeEnumEntry = regexp.MustCompile(`([A-Za-z][A-Za-z0-9]*)\s*=\s*(\d+)`)

// TestPobSkillTypeNumberMatchesPin gates tag_canon_test.go's hand-written pobSkillTypeNumber against the pinned PoB's own SkillType table.
//
//pob:src/Data/Global.lua:217-352
func TestPobSkillTypeNumberMatchesPin(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	raw, err := os.ReadFile(filepath.Join(pobRoot, "src", "Data", "Global.lua"))
	if err != nil {
		t.Skipf("NOT-RUN: no pinned PoB Global.lua at %s (set POB_DIR). This has NOT told you the "+
			"enum is current — it has told you it could not look: %v", pobRoot, err)
	}

	_, after, found := strings.Cut(string(raw), "SkillType = {")
	if !found {
		t.Fatal("REFUSING: no `SkillType = {` in the pinned Global.lua — the table moved or was renamed")
	}
	body, _, found := strings.Cut(after, "\n}")
	if !found {
		t.Fatal("REFUSING: unterminated SkillType table in the pinned Global.lua")
	}

	want := map[string]int{}
	for _, line := range strings.Split(body, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		m := skillTypeEnumEntry.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatalf("REFUSING: %q is not a number in %q", m[2], line)
		}
		want[m[1]] = n
	}
	if len(want) < 100 {
		t.Fatalf("REFUSING: parsed only %d SkillType entries from the pin — the parser is broken, "+
			"not the list", len(want))
	}

	for name, n := range want {
		got, ok := pobSkillTypeNumber[name]
		if !ok {
			t.Errorf("pobSkillTypeNumber is MISSING %q (pin = %d) — every name after it in the "+
				"list is shifted", name, n)
			continue
		}
		if got != n {
			t.Errorf("pobSkillTypeNumber[%q] = %d, pin = %d", name, got, n)
		}
	}
	for name, n := range pobSkillTypeNumber {
		if _, ok := want[name]; !ok {
			t.Errorf("pobSkillTypeNumber has %q = %d, which the pin's SkillType table does not "+
				"contain", name, n)
		}
	}
	if len(pobSkillTypeNumber) != len(want) {
		t.Errorf("pobSkillTypeNumber has %d entries, pin has %d", len(pobSkillTypeNumber), len(want))
	}
}
