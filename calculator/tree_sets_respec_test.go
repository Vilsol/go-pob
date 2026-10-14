package calculator

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	respecGoldRe  = regexp.MustCompile(`^\^xFFD700([\d,]+) Gold`)
	respecNodesRe = regexp.MustCompile(`^\^7\t(\d+) (Passive|Ascendancy) node`)
)

var respecKnownDiff = map[string]bool{"0044-assassin-charged-dash.xml/3": true}

// TestTreeSpecRespecCostMatchesPoB: Produced by oracle/tree_spec_respec.lua from the pinned PoB.
func TestTreeSpecRespecCostMatchesPoB(t *testing.T) {
	t.Parallel()
	f, err := os.Open("../testdata/oracle/tree_spec_respec_expected.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	total, withGold, wrong := 0, 0, 0
	for scanner.Scan() {
		var row struct {
			File  string   `json:"file"`
			Spec  int      `json:"spec"`
			Lines []string `json:"lines"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		want := TreeSpecRespec{}
		for _, l := range row.Lines {
			if m := respecGoldRe.FindStringSubmatch(l); m != nil {
				want.Gold, _ = strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
			}
			if m := respecNodesRe.FindStringSubmatch(l); m != nil {
				n, _ := strconv.Atoi(m[1])
				if m[2] == "Passive" {
					want.Passives = n
				} else {
					want.Ascendancy = n
				}
			}
		}
		got := TreeSpecRespecCost(parseCorpusBuild(t, row.File), row.Spec)
		total++
		if respecKnownDiff[fmt.Sprintf("%s/%d", row.File, row.Spec)] {
			if got == want {
				t.Errorf("%s tree %d now matches PoB: drop it from respecKnownDiff", row.File, row.Spec)
			}
			continue
		}
		if want.Gold > 0 {
			withGold++
		}
		if got != want {
			wrong++
			t.Errorf("%s tree %d: %+v, PoB %+v", row.File, row.Spec, got, want)
		}
	}
	if total < 100 || withGold < 50 {
		t.Fatalf("%d trees, %d with gold: no control", total, withGold)
	}
	if wrong > 0 {
		t.Errorf("%d of %d trees differ", wrong, total)
	}
}
