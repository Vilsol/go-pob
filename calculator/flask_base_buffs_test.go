package calculator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	pinnedFlaskBaseRe = regexp.MustCompile(`(?s)itemBases\["([^"]+)"\] = \{(.*?)\n\}`)
	pinnedFlaskBuffRe = regexp.MustCompile(`(?s)buff = \{(.*?)\}`)
	pinnedQuotedRe    = regexp.MustCompile(`"([^"]*)"`)
)

// TestFlaskBaseBuffsMatchThePin gates `flaskBuffs` (calculator/item_bases.go) against src/Data/Bases/flask.lua in the PINNED PoB.
func TestFlaskBaseBuffsMatchThePin(t *testing.T) {
	pobRoot := os.Getenv("POB_DIR")
	if pobRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("NOT-RUN: cannot resolve $HOME to find the pinned PoB: %v", err)
		}
		pobRoot = filepath.Join(home, "Projects", "Vilsol", "PathOfBuilding")
	}
	path := filepath.Join(pobRoot, "src", "Data", "Bases", "flask.lua")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("NOT-RUN: no pinned PoB flask table at %s (set POB_DIR). This has NOT told you "+
			"flaskBuffs is current — it has told you it could not look.", path)
	}

	want := map[string][]string{}
	for _, m := range pinnedFlaskBaseRe.FindAllStringSubmatch(string(raw), -1) {
		name, body := m[1], m[2]
		buff := pinnedFlaskBuffRe.FindStringSubmatch(body)
		if buff == nil {
			continue
		}
		var lines []string
		for _, q := range pinnedQuotedRe.FindAllStringSubmatch(buff[1], -1) {
			lines = append(lines, q[1])
		}
		if len(lines) > 0 {
			want[name] = lines
		}
	}

	if len(want) == 0 {
		t.Fatalf("REFUSING: parsed 0 flask buffs out of %s. The dump shape changed and this test "+
			"is reading nothing, which would pass for any value of flaskBuffs", path)
	}
	if len(want) < 10 {
		t.Fatalf("REFUSING: parsed only %d flask buffs out of %s; the pin declares far more and "+
			"a partial read is worse than none", len(want), path)
	}

	for name, wantLines := range want {
		got, ok := flaskBuffs[name]
		if !ok {
			t.Errorf("%s: absent from flaskBuffs; pin declares %q", name, wantLines)
			continue
		}
		if strings.Join(got, "\x00") != strings.Join(wantLines, "\x00") {
			t.Errorf("%s:\n  ours %q\n  pin  %q", name, got, wantLines)
		}
	}
	for name := range flaskBuffs {
		if _, ok := want[name]; !ok {
			t.Errorf("%s: in flaskBuffs but the pin declares no buff for it", name)
		}
	}
}
