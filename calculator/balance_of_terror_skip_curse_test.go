package calculator

import (
	"os"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/moddb"
)

// TestBalanceOfTerrorSkipsTheCurse pins CalcPerform.lua:2604-2609.
func TestBalanceOfTerrorSkipsTheCurse(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const carrier = "0787-juggernaut-molten-strike-of-the-zenith-2"
	env := corpusEnvForLoadedData(t, carrier)

	found := false
	for _, in := range env.Build.Config.Inputs {
		if in.Name == "balanceOfTerrorSelfCastDespair" {
			found = true
		}
	}
	if !found {
		t.Fatalf("REFUSING: %s no longer carries <Input name=\"balanceOfTerrorSelfCastDespair\"> — "+
			"the population emptied rather than this test passing", carrier)
	}
	if !balanceOfTerrorSelfOnly(env, "Despair") {
		t.Errorf("balanceOfTerrorSelfOnly(%s, Despair) = false, want true — the config input is not read", carrier)
	}
	if balanceOfTerrorSelfOnly(env, "NoSuchCurse") {
		t.Error("balanceOfTerrorSelfOnly answers true for a curse with no input at all")
	}

	control := corpusEnvForLoadedData(t, "0001-necromancer-unknown")
	if balanceOfTerrorSelfOnly(control, "Despair") {
		t.Error("the control build answers true for Despair — the read is not build-scoped")
	}
	if countSourcedMods(control.EnemyModDB, "Despair") == 0 {
		t.Fatal("REFUSING: the control applies no Despair-sourced mod to the enemy either — this " +
			"test cannot tell a skipped curse from a curse that never applies")
	}
	if got := countSourcedMods(env.EnemyModDB, "Despair"); got != 0 {
		t.Errorf("%s applies %d Despair-sourced enemy mods, want 0 (PoB's own dump has 0)", carrier, got)
	}
}

func countSourcedMods(db *moddb.ModDB, source string) int {
	n := 0
	for _, list := range db.Mods {
		for _, m := range list {
			if strings.Contains(string(m.GetSource()), source) {
				n++
			}
		}
	}
	return n
}
