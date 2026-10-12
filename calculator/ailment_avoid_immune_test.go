package calculator

import (
	"math"
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
)

// TestAilmentAvoidChanceHonoursImmunityAndFloors pins CalcDefence.lua:1580-1586, which this port had collapsed into a single `min(Sum(...), 100)`.
//
//	for _, ailment in ipairs(data.nonElementalAilmentTypeList) do
//	    output[ailment.."AvoidChance"] = modDB:Flag(nil, ailment.."Immune") and 100
//	        or m_floor(m_min(modDB:Sum("BASE", nil, "Avoid"..ailment, "AvoidAilments"), 100))
//	end
//	for _, ailment in ipairs(data.elementalAilmentTypeList) do
//	    ... modDB:Flag(nil, ailment.."Immune", "ElementalAilmentImmune") and 100 or m_floor(...)
func TestAilmentAvoidChanceHonoursImmunityAndFloors(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	elemental := map[data.Ailment]bool{}
	for _, a := range data.ElementalAilment("").Values() {
		elemental[a] = true
	}

	var immuneSeen, immuneWrong, fracSeen, fracWrong, builds int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		db := env.Player.ModDB
		builds++

		for _, ailment := range data.Ailment("").Values() {
			key := string(ailment + "AvoidChance")
			want, has := rec.Output[key]
			if !has {
				continue
			}
			got := env.Player.Output[key]

			flags := []string{string(ailment + "Immune")}
			sumNames := []string{string("Avoid" + ailment), "AvoidAilments"}
			if elemental[ailment] {
				flags = append(flags, "ElementalAilmentImmune")
				sumNames = append(sumNames, "AvoidElementalAilments")
			}

			switch {
			case db.Flag(nil, flags...):
				immuneSeen++
				if !oracleClose(got, want, oracleDumpTolerance) {
					immuneWrong++
				}
			default:
				raw := math.Min(db.Sum(mod.TypeBase, nil, sumNames...), 100)
				if raw != math.Floor(raw) {
					fracSeen++
					if !oracleClose(got, want, oracleDumpTolerance) {
						fracWrong++
					}
				}
			}
		}
		if immuneSeen >= 200 {
			break
		}
	}

	if builds == 0 {
		t.Fatal("REFUSING: no graded build was built; this test measured nothing")
	}
	if immuneSeen == 0 {
		t.Fatalf("REFUSING: over %d builds not one carries an <Ailment>Immune or "+
			"ElementalAilmentImmune flag, so the short-circuit branch decides nothing and this "+
			"test no longer measures its mechanism", builds)
	}
	if fracSeen != 0 {
		t.Logf("m_floor is now LIVE: %d (build, ailment) pairs have a fractional avoid sum "+
			"over %d builds; it was a measured zero when this test was written", fracSeen, builds)
	}
	if immuneWrong > 0 {
		t.Errorf("immunity short-circuit: %d of %d (build, ailment) pairs with an Immune flag "+
			"disagree with PoB", immuneWrong, immuneSeen)
	}
	if fracWrong > 0 {
		t.Errorf("m_floor: %d of %d (build, ailment) pairs with a fractional avoid sum disagree "+
			"with PoB", fracWrong, fracSeen)
	}
}
