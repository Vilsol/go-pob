package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

// TestSelfCurseEffectClampsAtZero pins CalcPerform.lua:3116's `m_max((1 + inc / 100) * more, 0)`.
func TestSelfCurseEffectClampsAtZero(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "1308-hierophant-flame-dash.xml")
	for _, r := range env.ModDB.Tabulate(mod.TypeBase, nil, "FireResist") {
		if r.Mod.GetSource() == "Skill:Flammability" && r.Value != 0 {
			t.Errorf("self-cursed Flammability FireResist = %v, want 0", r.Value)
		}
	}
	assertBuildStats(t, "1308-hierophant-flame-dash.xml", []string{"FireResistTotal"}, nil)
}
