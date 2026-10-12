package calculator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestSelfCastConditionMatchesPoB pins CalcPerform.lua:2603 — `modDB.conditions["SelfCast"..buff.name:gsub(" ","")] = not (Triggered or Aura)`.
func TestSelfCastConditionMatchesPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const build = "0001-necromancer-unknown"
	raw, err := os.ReadFile(filepath.Join(dumpsDirForLoadedData(t), build+".json"))
	if err != nil {
		t.Fatalf("REFUSING: cannot read PoB's dump for %s: %v", build, err)
	}
	var dump struct {
		Conditions map[string]bool `json:"conditions"`
	}
	if err := json.Unmarshal(raw, &dump); err != nil {
		t.Fatalf("REFUSING: %s did not decode: %v", build, err)
	}

	want := map[string]bool{}
	for k, v := range dump.Conditions {
		if len(k) > 8 && k[:8] == "SelfCast" {
			want[k] = v
		}
	}
	if len(want) == 0 {
		t.Fatalf("REFUSING: PoB sets no SelfCast condition on %s — the population emptied rather "+
			"than this test passing", build)
	}
	trues, falses := 0, 0
	for _, v := range want {
		if v {
			trues++
		} else {
			falses++
		}
	}
	if trues == 0 || falses == 0 {
		t.Fatalf("REFUSING: %s no longer carries BOTH values (%d true, %d false) — a fixture "+
			"with one value cannot tell the negation from a constant", build, trues, falses)
	}

	env := corpusEnvForLoadedData(t, build)
	for name, w := range want {
		if got := env.Player.ModDB.Conditions[name]; got != w {
			t.Errorf("%s = %v, want %v (PoB's own dump)", name, got, w)
		}
	}
}
