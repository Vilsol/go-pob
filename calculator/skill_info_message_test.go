package calculator

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

// TestSkillInfoMessagesMatchPoB: Produced by oracle/skill_info_message.lua from the pinned PoB over the first 400 corpus builds.
func TestSkillInfoMessagesMatchPoB(t *testing.T) {
	t.Parallel()
	f, err := os.Open("../testdata/oracle/skill_info_message_expected.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	total, nonEmpty, wrong := 0, 0, 0
	for scanner.Scan() {
		var w struct {
			File        string `json:"file"`
			Info        string `json:"info"`
			Info2       string `json:"info2"`
			MinionInfo  string `json:"minionInfo"`
			MinionInfo2 string `json:"minionInfo2"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &w); err != nil {
			t.Fatal(err)
		}
		c := sidebarEnv(t, w.File).MainSkillControls()
		got := [4]string{}
		if c != nil {
			got = [4]string{c.InfoMessage, c.InfoMessage2, c.MinionInfoMessage, c.MinionInfoMessage2}
		}
		want := [4]string{w.Info, w.Info2, w.MinionInfo, w.MinionInfo2}
		total++
		if want != [4]string{} {
			nonEmpty++
		}
		if skillInfoKnownDiff[w.File] {
			if got == want {
				t.Errorf("%s now matches PoB: drop it from skillInfoKnownDiff", w.File)
			}
			continue
		}
		if got != want {
			wrong++
			if os.Getenv("SIM_ALL") != "" || wrong <= 10 {
				t.Errorf("%s:\n got %q\nPoB %q", w.File, got, want)
			}
		}
	}
	if total < 300 || nonEmpty < 50 {
		t.Fatalf("%d builds, %d with a message: no control", total, nonEmpty)
	}
	if wrong > 0 {
		t.Errorf("%d of %d builds differ", wrong, total)
	}
}

var skillInfoKnownDiff = map[string]bool{"0184-elementalist-exsanguinate-of-transmission.xml": true, "0393-elementalist-reap.xml": true}
