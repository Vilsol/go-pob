package calculator

import (
	"sort"
	"strings"
	"testing"
)

// TestSelfAurasSkipAHostileMinion: with Penance Mark as the main skill the minion is the hostile Shackled Phantasm,
// and the pinned PoB gives it none of the player's auras (its dumped minion conditions carry no AffectedBy key).
//
//pob:block:src/Modules/CalcPerform.lua:2513
func TestSelfAurasSkipAHostileMinion(t *testing.T) {
	env := corpusEnvWithEdit(t, "0134-chieftain-detonate-dead", `mainSocketGroup="2"`, `mainSocketGroup="12"`)
	if env.Minion == nil || env.Minion.MinionData == nil || !env.Minion.MinionData.Hostile {
		t.Fatal("premise broken: the main skill's minion is not hostile")
	}
	var got []string
	for k, v := range env.Minion.ModDB.Conditions {
		if v && strings.HasPrefix(k, "AffectedBy") {
			got = append(got, k)
		}
	}
	sort.Strings(got)
	if len(got) != 0 {
		t.Errorf("hostile minion conditions %v, PoB sets none", got)
	}
}
