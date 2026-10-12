package calculator

import (
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// TestWardNotBreakIsAConditionFlag pins the ward-break flag END TO END.
func TestWardNotBreakIsAConditionFlag(t *testing.T) {
	requireGameData(t, "3.29")

	e := ParseMod("Ward does not Break during Effect", false)
	if e == nil || strings.TrimSpace(modExtra(e)) != "" {
		t.Fatalf("REFUSING: the line does not parse cleanly, so nothing below is attributable")
	}

	db := moddb.NewModDB()
	db.AddMod(mod.NewFlag("Condition:UsingFlask", true).Source("Config"))
	for _, m := range e.ModList {
		db.AddMod(m)
	}

	if !db.Flag(nil, "Condition:WardNotBreak") {
		names := make([]string, 0, len(e.ModList))
		for _, m := range e.ModList {
			names = append(names, m.Name())
		}
		t.Errorf("parser produced %v — no Condition:WardNotBreak, so a Condition(\"WardNotBreak\") "+
			"tag (ModParser.lua:5392) can never see it", names)
	}

	output := map[string]float64{
		"Ward": 100, "LifeRecoverable": 1000, "Life": 1000,
		"EnergyShieldRecoveryCap": 0, "ManaUnreserved": 0, "DamageReductionMax": 90,
	}
	hit := func(m *moddb.ModDB) *PoolTable {
		return ReducePoolsByDamage(&PoolTable{},
			map[data.DamageType]float64{data.DamageTypePhysical: 50},
			&Actor{ModDB: m, Output: output})
	}

	if bare := hit(moddb.NewModDB()); bare.Ward == nil || *bare.Ward != 0 {
		t.Fatalf("REFUSING: with no ward flag the ward should break to 0, got %v — the fixture "+
			"does not reach the break path and nothing below is attributable", bare.Ward)
	}

	if got := hit(db); got.Ward == nil || *got.Ward != 100 {
		t.Errorf("with the parsed ward-break mod applied the ward must not break: Ward = %v, "+
			"want 100 (producer and consumer disagree about the flag name)", *got.Ward)
	}
}
