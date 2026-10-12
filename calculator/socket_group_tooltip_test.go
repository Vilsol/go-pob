package calculator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

var socketGroupTooltipKnownDiff = map[string]bool{
	"0060-assassin-icicle-mine.xml/0": true,
	"0094-reliquarian-cyclone.xml/1":  true,
}

// TestSocketGroupTooltipAgainstPoB: Produced by oracle/socket_group_tooltip.lua from the pinned PoB.
func TestSocketGroupTooltipAgainstPoB(t *testing.T) {
	f, err := os.Open("../testdata/oracle/socket_group_tooltip_expected.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type row struct {
		File  string   `json:"file"`
		Group int      `json:"group"`
		Lines []string `json:"lines"`
	}
	byFile := map[string][]row{}
	var files []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		var r row
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if _, ok := byFile[r.File]; !ok {
			files = append(files, r.File)
		}
		byFile[r.File] = append(byFile[r.File], r)
	}
	if len(files) < 10 {
		t.Fatalf("only %d builds", len(files))
	}
	wrong, total := 0, 0
	for _, file := range files {
		env := sidebarEnv(t, file)
		for _, r := range byFile[file] {
			got := []string{}
			if text := strings.TrimSuffix(tooltipText(env.SocketGroupTooltip(r.Group)), "\n"); text != "" {
				got = strings.Split(text, "\n")
			}
			want := r.Lines
			if want == nil {
				want = []string{}
			}
			total++
			if socketGroupTooltipKnownDiff[fmt.Sprintf("%s/%d", file, r.Group)] {
				if sameStrings(got, want) {
					t.Errorf("%s group %d now matches PoB: drop it from socketGroupTooltipKnownDiff", file, r.Group)
				}
				continue
			}
			if !sameStrings(got, want) {
				wrong++
				if os.Getenv("SGT_ALL") != "" || wrong <= 5 {
					t.Errorf("%s group %d:\n got %q\nPoB %q", file, r.Group, got, want)
				}
			}
		}
	}
	if wrong > 0 {
		t.Errorf("%d of %d tooltips differ", wrong, total)
	}
}
